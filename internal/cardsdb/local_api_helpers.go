package cardsdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func pageSize(value int) int {
	if value <= 0 {
		return 20
	}
	if value > 200 {
		return 200
	}
	return value
}

func parseOffset(token string) int {
	n, err := strconv.Atoi(strings.TrimSpace(token))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func searchResultsResponse(results []SearchResult, params SearchCardsParams) *SearchCardsResponse {
	limit := pageSize(params.PageSize)
	offset := parseOffset(params.NextToken)
	if offset > len(results) {
		offset = len(results)
	}
	end := offset + limit
	next := ""
	if end < len(results) {
		next = strconv.Itoa(end)
	} else {
		end = len(results)
	}
	out := make([]CardSummary, 0, end-offset)
	for _, result := range results[offset:end] {
		out = append(out, CardSummary{
			Identifier:      result.ID,
			Name:            result.Name,
			Cost:            result.Cost,
			Pitch:           result.Pitch,
			Power:           result.Power,
			Defense:         result.Defense,
			Health:          result.Health,
			Types:           splitTypeLine(result.TypeLine),
			FunctionalText:  result.Text,
			DefaultImageURL: result.ImageURL,
		})
	}
	return &SearchCardsResponse{Summaries: out, NextToken: next}
}

func (r *Repository) addTextPredicate(where *[]string, args *[]any, search string) {
	if search == "" {
		return
	}
	if r.features.HasCardSearchProjection {
		*where = append(*where, "(cards.id LIKE ? COLLATE NOCASE OR COALESCE(csp.core_name, core.name, '') LIKE ? COLLATE NOCASE OR COALESCE(csp.textbox, core.textbox, '') LIKE ? COLLATE NOCASE)")
		like := "%" + search + "%"
		*args = append(*args, like, like, like)
		return
	}
	if fts := ftsMatch(search); fts != "" && (r.features.HasOpenNameFTS || r.features.HasOpenRulesFTS || r.features.HasOpenTextFTS) {
		clauses := []string{}
		if r.features.HasOpenNameFTS {
			clauses = append(clauses, "EXISTS (SELECT 1 FROM open_name_fts WHERE open_name_fts.card_id = cards.id AND open_name_fts MATCH ?)")
			*args = append(*args, fts)
		}
		if r.features.HasOpenRulesFTS {
			clauses = append(clauses, "EXISTS (SELECT 1 FROM open_rules_fts WHERE open_rules_fts.card_id = cards.id AND open_rules_fts MATCH ?)")
			*args = append(*args, fts)
		}
		if r.features.HasOpenTextFTS && len(clauses) == 0 {
			clauses = append(clauses, "EXISTS (SELECT 1 FROM open_text_fts WHERE open_text_fts.card_id = cards.id AND open_text_fts MATCH ?)")
			*args = append(*args, fts)
		}
		*where = append(*where, "("+strings.Join(clauses, " OR ")+")")
		return
	}
	*where = append(*where, "(cards.id LIKE ? COLLATE NOCASE OR core.name LIKE ? COLLATE NOCASE OR core.textbox LIKE ? COLLATE NOCASE)")
	like := "%" + search + "%"
	*args = append(*args, like, like, like)
}

func (r *Repository) addColorPredicate(where *[]string, args *[]any, colorIdentity string) {
	color := mapColorIdentity(colorIdentity)
	if color == "" {
		return
	}
	if r.features.HasCardSearchProjection && r.features.HasProjectionColorNorm {
		*where = append(*where, "LOWER(COALESCE(csp.color_norm, '')) = ?")
	} else if r.features.HasCardSearchProjection && r.features.HasProjectionColor {
		*where = append(*where, "LOWER(COALESCE(csp.color, core.color, '')) = ?")
	} else {
		*where = append(*where, "LOWER(COALESCE(core.color, '')) = ?")
	}
	*args = append(*args, strings.ToLower(color))
}

func (r *Repository) addNumericPredicates(where *[]string, args *[]any, params SearchCardsParams) {
	fields := []struct {
		value string
		expr  string
	}{
		{params.Cost, projectionOrNumeric(r.features.HasCardSearchProjection, "csp.cost_num", "core.cost")},
		{params.Defense, projectionOrNumeric(r.features.HasCardSearchProjection, "csp.defense_num", "core.defense")},
		{params.Pitch, projectionOrNumeric(r.features.HasCardSearchProjection, "csp.pitch_value", "core.pitch_value")},
		{params.Power, projectionOrNumeric(r.features.HasCardSearchProjection, "csp.power_num", "core.power")},
		{params.Health, projectionOrNumeric(r.features.HasCardSearchProjection, "csp.life_num", "core.life")},
		{params.Intelligence, projectionOrNumeric(r.features.HasCardSearchProjection, "csp.intellect_num", "core.intellect")},
	}
	for _, field := range fields {
		if clause, value, ok := numericPredicate(field.value, field.expr); ok {
			*where = append(*where, clause)
			*args = append(*args, value)
		}
	}
}

func projectionOrNumeric(useProjection bool, projection, fallback string) string {
	if useProjection {
		return projection
	}
	return "CASE WHEN (" + fallback + " IS NOT NULL AND " + fallback + " != '' AND " + fallback + " NOT GLOB '*[^0-9]*') THEN CAST(" + fallback + " AS INTEGER) END"
}

func numericPredicate(raw, expr string) (string, string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", false
	}
	op := "="
	for _, candidate := range []string{">=", "<=", ">", "<", "="} {
		if strings.HasPrefix(raw, candidate) {
			op = candidate
			raw = strings.TrimSpace(strings.TrimPrefix(raw, candidate))
			break
		}
	}
	if raw == "" {
		return "", "", false
	}
	return expr + " " + op + " ?", raw, true
}

func (r *Repository) addJSONFilter(where *[]string, args *[]any, column string, enabled bool, raw string) {
	values := splitSelectorValues(raw)
	if len(values) == 0 || !enabled || !r.features.HasCardSearchProjection {
		return
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(values)), ",")
	*where = append(*where, "EXISTS (SELECT 1 FROM json_each(COALESCE("+column+", '[]')) je WHERE LOWER(CAST(je.value AS TEXT)) IN ("+placeholders+"))")
	for _, value := range values {
		*args = append(*args, value)
	}
}

type selectorFilter struct {
	Raw               string
	ProjectionColumn  string
	ProjectionEnabled bool
	Table             string
	Alias             string
	ValueColumn       string
	NormColumn        string
	TableEnabled      bool
	NormEnabled       bool
	MatchCoreIndex    bool
}

func (r *Repository) addSelectorFilter(where *[]string, args *[]any, filter selectorFilter) {
	values := splitSelectorValues(filter.Raw)
	if len(values) == 0 {
		return
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(values)), ",")
	if r.features.HasCardSearchProjection && filter.ProjectionEnabled {
		*where = append(*where, "EXISTS (SELECT 1 FROM json_each(COALESCE("+filter.ProjectionColumn+", '[]')) je WHERE LOWER(CAST(je.value AS TEXT)) IN ("+placeholders+"))")
		for _, value := range values {
			*args = append(*args, value)
		}
		return
	}
	if !filter.TableEnabled {
		*where = append(*where, "0=1")
		return
	}
	valueExpr := filter.Alias + "." + filter.ValueColumn
	if filter.NormEnabled {
		valueExpr = "COALESCE(" + filter.Alias + "." + filter.NormColumn + ", " + valueExpr + ")"
	}
	clauses := []string{
		filter.Alias + ".card_id = cards.id",
		"LOWER(COALESCE(" + valueExpr + ", '')) IN (" + placeholders + ")",
	}
	if filter.MatchCoreIndex {
		clauses = append(clauses, filter.Alias+".core_index = dc.core_index")
	}
	*where = append(*where, "EXISTS (SELECT 1 FROM "+filter.Table+" "+filter.Alias+" WHERE "+strings.Join(clauses, " AND ")+")")
	for _, value := range values {
		*args = append(*args, value)
	}
}

func (r *Repository) addPrintingFilters(where *[]string, args *[]any, params SearchCardsParams) {
	if !r.features.HasPrintings {
		return
	}
	var clauses []string
	for _, filter := range []struct {
		field string
		value string
	}{
		{"p.set_code", params.SetCode},
		{"p.rarity", params.Rarity},
	} {
		values := splitSelectorValues(filter.value)
		if filter.field == "p.rarity" {
			values = normalizeRaritySelectorValues(filter.value)
		}
		if len(values) == 0 {
			continue
		}
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(values)), ",")
		if filter.field == "p.set_code" && r.features.HasSets {
			clauses = append(clauses, "(LOWER(COALESCE(p.set_code, '')) IN ("+placeholders+") OR EXISTS (SELECT 1 FROM sets sf WHERE sf.code = p.set_code AND LOWER(COALESCE(sf.name, '')) IN ("+placeholders+")))")
			for _, value := range values {
				*args = append(*args, value)
			}
		} else {
			clauses = append(clauses, "LOWER(COALESCE("+filter.field+", '')) IN ("+placeholders+")")
		}
		for _, value := range values {
			*args = append(*args, value)
		}
	}
	languages := expandLanguageCodes(splitSelectorValues(params.Language))
	if len(languages) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(languages)), ",")
		clauses = append(clauses, "LOWER(COALESCE(p.language, '')) IN ("+placeholders+")")
		for _, value := range languages {
			*args = append(*args, value)
		}
	}
	artistValues := splitSelectorValues(params.Artist)
	if len(artistValues) > 0 && r.features.HasPrintingFaces {
		artistClauses := make([]string, 0, len(artistValues))
		for _, artist := range artistValues {
			artistClauses = append(artistClauses, "LOWER(COALESCE(pf.artist, '')) LIKE ?")
			*args = append(*args, "%"+artist+"%")
		}
		clauses = append(clauses, "EXISTS (SELECT 1 FROM printing_faces pf WHERE pf.printing_id = p.id AND ("+strings.Join(artistClauses, " OR ")+"))")
	}
	if len(clauses) == 0 {
		return
	}
	*where = append(*where, "EXISTS (SELECT 1 FROM printings p WHERE p.card_id = cards.id AND "+strings.Join(clauses, " AND ")+")")
}

func normalizeRaritySelectorValues(value string) []string {
	rawValues := splitSelectorValues(value)
	out := []string{}
	seen := map[string]bool{}
	push := func(values ...string) {
		for _, entry := range values {
			entry = strings.TrimSpace(strings.ToLower(entry))
			if entry == "" || seen[entry] {
				continue
			}
			seen[entry] = true
			out = append(out, entry)
		}
	}
	for _, raw := range rawValues {
		switch strings.ReplaceAll(strings.ReplaceAll(raw, "-", "_"), " ", "_") {
		case "b", "basic":
			push("basic")
		case "t", "token":
			push("token")
		case "c", "common":
			push("common")
		case "r", "rare":
			push("rare")
		case "s", "sr", "super", "superrare", "super_rare":
			push("super_rare", "super-rare", "super rare", "superrare")
		case "m", "majestic":
			push("majestic")
		case "l", "legendary":
			push("legendary")
		case "f", "fabled":
			push("fabled")
		case "mv", "marvel":
			push("marvel")
		case "p", "promo":
			push("promo")
		default:
			push(raw)
		}
	}
	return out
}

func expandLanguageCodes(values []string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(values ...string) {
		for _, value := range values {
			if value != "" && !seen[value] {
				seen[value] = true
				out = append(out, value)
			}
		}
	}
	for _, value := range values {
		switch value {
		case "ja":
			add("ja", "jp")
		case "jp":
			add("jp", "ja")
		case "es":
			add("es", "sp")
		case "sp":
			add("sp", "es")
		case "zhs", "cs", "zh":
			add("zhs", "cs", "zh")
		case "zht", "ct":
			add("zht", "ct")
		default:
			add(value)
		}
	}
	return out
}

func (r *Repository) addLegalityFilters(where *[]string, args *[]any, params SearchCardsParams) {
	filters := []struct {
		enabled *bool
		format  string
		status  string
		jsonKey string
		proj    string
	}{
		{params.BlitzLegal, "Blitz", "legal", "blitz", "csp.is_blitz_legal"},
		{params.CCLegal, "Classic Constructed", "legal", "classicConstructed", "csp.is_classic_constructed_legal"},
		{params.CommonerLegal, "Commoner", "legal", "commoner", "csp.is_commoner_legal"},
		{params.ProjectBlueLegal, "Silver Age", "legal", "projectBlue", "csp.is_silver_age_legal"},
		{params.BlitzBanned, "Blitz", "banned", "blitz", ""},
		{params.CCBanned, "Classic Constructed", "banned", "classicConstructed", ""},
		{params.CommonerBanned, "Commoner", "banned", "commoner", ""},
		{params.UPFBanned, "Ultimate Pit Fight", "banned", "ultimatePitFight", ""},
		{params.LLBanned, "Living Legend", "banned", "livingLegend", ""},
		{params.LLRestricted, "Living Legend", "restricted", "livingLegend", ""},
		{params.ProjectBlueBanned, "Silver Age", "banned", "projectBlue", ""},
		{params.BlitzSuspended, "Blitz", "suspended", "blitz", ""},
		{params.CCSuspended, "Classic Constructed", "suspended", "classicConstructed", ""},
		{params.CommonerSuspended, "Commoner", "suspended", "commoner", ""},
		{params.ProjectBlueSuspended, "Silver Age", "suspended", "projectBlue", ""},
		{params.BlitzLivingLegend, "Blitz", "living legend", "blitz", ""},
		{params.CCLivingLegend, "Classic Constructed", "living legend", "classicConstructed", ""},
	}
	for _, filter := range filters {
		if filter.enabled == nil || !*filter.enabled {
			continue
		}
		if r.features.HasCardSearchProjection && filter.proj != "" && r.hasProjectionLegalityColumn(filter.proj) {
			*where = append(*where, filter.proj+" = 1")
			continue
		}
		if r.features.HasCardLegalities {
			*where = append(*where, "EXISTS (SELECT 1 FROM card_legalities cl WHERE cl.card_id = cards.id AND cl.format = ? AND LOWER(cl.legality) = ?)")
			*args = append(*args, filter.format, filter.status)
			continue
		}
		if r.features.HasCardLegalityJSON {
			clause, clauseArgs := legalityJSONPredicate(filter.jsonKey, filter.status)
			*where = append(*where, clause)
			*args = append(*args, clauseArgs...)
		}
	}
}

func (r *Repository) hasProjectionLegalityColumn(proj string) bool {
	switch proj {
	case "csp.is_blitz_legal":
		return r.features.HasProjectionBlitzLegal
	case "csp.is_classic_constructed_legal":
		return r.features.HasProjectionCCLegal
	case "csp.is_commoner_legal":
		return r.features.HasProjectionCommonerLegal
	case "csp.is_silver_age_legal":
		return r.features.HasProjectionProjectBlueLegal
	default:
		return false
	}
}

func legalityJSONPredicate(key string, status string) (string, []any) {
	boolField := map[string]string{
		"legal":         "isLegal",
		"banned":        "banned",
		"suspended":     "suspended",
		"restricted":    "restricted",
		"living legend": "isLivingLegend",
	}[status]
	keys := legalityJSONKeys(key)
	clauses := make([]string, 0, len(keys)*2)
	args := []any{}
	for _, jsonKey := range keys {
		if boolField != "" {
			clauses = append(clauses, fmt.Sprintf("json_extract(cards.card_legality_json, '$.\"%s\".%s') = 1", jsonKey, boolField))
		}
		clauses = append(clauses, fmt.Sprintf("LOWER(COALESCE(json_extract(cards.card_legality_json, '$.\"%s\".legality'), json_extract(cards.card_legality_json, '$.\"%s\".status'), json_extract(cards.card_legality_json, '$.\"%s\"'), '')) LIKE ?", jsonKey, jsonKey, jsonKey))
		args = append(args, "%"+strings.ReplaceAll(status, " ", "%")+"%")
	}
	return "(" + strings.Join(clauses, " OR ") + ")", args
}

func legalityJSONKeys(key string) []string {
	switch key {
	case "blitz":
		return []string{"blitz", "Blitz"}
	case "classicConstructed":
		return []string{"classicConstructed", "classic_constructed", "Classic Constructed"}
	case "commoner":
		return []string{"commoner", "Commoner"}
	case "ultimatePitFight":
		return []string{"ultimatePitFight", "ultimate_pit_fight", "Ultimate Pit Fight"}
	case "livingLegend":
		return []string{"livingLegend", "living_legend", "Living Legend"}
	case "projectBlue":
		return []string{"projectBlue", "project_blue", "Silver Age", "Project Blue"}
	default:
		return []string{key}
	}
}

func (r *Repository) projectionColorExpr() string {
	if r.features.HasCardSearchProjection && r.features.HasProjectionColorNorm {
		return "csp.color_norm"
	}
	if r.features.HasCardSearchProjection && r.features.HasProjectionColor {
		return "csp.color"
	}
	return "NULL"
}

func cardJSONExpr(ok bool, expr string) string {
	if ok {
		return "COALESCE(" + expr + ", '[]')"
	}
	return "'[]'"
}

func mapColorIdentity(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "color_identity_red", "red":
		return "Red"
	case "color_identity_yellow", "yellow":
		return "Yellow"
	case "color_identity_blue", "blue":
		return "Blue"
	case "color_identity_none", "none":
		return "None"
	default:
		return ""
	}
}

func splitTypeLine(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return strings.FieldsFunc(value, func(r rune) bool {
		return r == '-' || r == ',' || r == '/'
	})
}

func parseJSONValue(raw string) any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return nil
	}
	return value
}

func cardDetailToSummary(detail *CardDetail) CardSummary {
	s := CardSummary{
		Identifier:      detail.ID,
		Name:            detail.Name,
		Cost:            detail.Cost,
		Pitch:           detail.Pitch,
		Power:           detail.Power,
		Defense:         detail.Defense,
		Health:          detail.Health,
		Intelligence:    detail.Intelligence,
		Arcane:          detail.Arcane,
		Types:           detail.BaseTypes,
		Keywords:        detail.Keywords,
		FunctionalText:  detail.Text,
		DefaultImageURL: detail.ImageURL,
		Legality:        parseJSONValue(detail.Legality),
	}
	if len(s.Types) == 0 {
		s.Types = splitTypeLine(detail.TypeLine)
	}
	return s
}

func (r *Repository) getPrinting(ctx context.Context, printingID string) (string, *CardPrinting, error) {
	setNameExpr := "''"
	joinSets := ""
	if r.features.HasSets {
		setNameExpr = "COALESCE(s.name, '')"
		joinSets = "LEFT JOIN sets s ON s.code = p.set_code"
	}
	row := r.db.QueryRowContext(ctx, fmt.Sprintf(`
SELECT
  COALESCE(p.card_id, ''),
  p.id,
  COALESCE(pf.name, p.product_name, p.id, ''),
  COALESCE(p.set_code, ''),
  %s AS set_name,
  COALESCE(p.rarity, ''),
  COALESCE(p.language, pf.language, ''),
  COALESCE(pf.image_large, pf.image_normal, pf.image_small, ''),
  COALESCE(%s, '') AS art_url,
  COALESCE(pf.artist, ''),
  COALESCE(pf.flavor_text, ''),
  COALESCE(%s, '') AS printed_text
FROM printings p
LEFT JOIN printing_faces pf ON pf.printing_id = p.id
%s
WHERE p.id = ?
ORDER BY (pf.face_id LIKE '%%_BACK') ASC, pf.face_id ASC
LIMIT 1`, setNameExpr, r.printingFaceCropExpr(), r.printingRulesTextExpr(), joinSets), printingID)
	var item CardPrinting
	var cardID, artist string
	if err := row.Scan(&cardID, &item.ID, &item.Name, &item.SetCode, &item.SetName, &item.Rarity, &item.Language, &item.ImageURL, &item.ArtURL, &artist, &item.FlavorText, &item.PrintedText); err != nil {
		return "", nil, err
	}
	if artist != "" {
		item.Artists = []string{artist}
	}
	return cardID, &item, nil
}

func printingToSummary(cardID string, printing CardPrinting, products []PrintingProduct) PrintingSummary {
	summary := PrintingSummary{
		Identifier:    printing.ID,
		CardID:        cardID,
		SetPrintingID: printing.ID,
		PrintingName:  printing.Name,
		Artists:       printing.Artists,
		FlavorText:    printing.FlavorText,
		ImageURL:      printing.ImageURL,
		SetID:         printing.SetCode,
		SetName:       printing.SetName,
		Rarity:        printing.Rarity,
	}
	for _, product := range products {
		if product.PrintingID == printing.ID {
			summary.Products = append(summary.Products, productToSummary(product))
		}
	}
	return summary
}

func (r *Repository) productByID(ctx context.Context, productID string) (*ProductSummary, error) {
	productID = strings.TrimSpace(productID)
	if productID == "" || !r.features.HasProductProjection {
		return nil, sql.ErrNoRows
	}
	columns, err := r.columnNames(ctx, "product_projection")
	if err != nil {
		return nil, err
	}
	expr := func(name string) string {
		if columns[name] {
			return "COALESCE(" + name + ", '')"
		}
		return "''"
	}
	quantityExpr := "0"
	if columns["quantity"] {
		quantityExpr = "COALESCE(quantity, 0)"
	}
	row := r.db.QueryRowContext(ctx, fmt.Sprintf(`
SELECT
  COALESCE(product_id, ''),
  %s,
  %s,
  %s,
  %s,
  %s,
  %s,
  %s,
  %s,
  %s,
  %s,
  %s
FROM product_projection
WHERE product_id = ?
LIMIT 1`,
		expr("type"),
		expr("card_id"),
		expr("printing_id"),
		expr("product_group_id"),
		expr("name"),
		expr("printed_date"),
		expr("printed_language"),
		expr("release_date"),
		expr("tcgplayer_url"),
		expr("tcgplayer_product_id"),
		quantityExpr,
	), productID)
	var p ProductSummary
	if err := row.Scan(&p.Identifier, &p.Type, &p.CardID, &p.PrintingID, &p.ProductGroupID, &p.Name, &p.PrintedDate, &p.PrintedLanguage, &p.ReleaseDate, &p.TCGPlayerURL, &p.TCGPlayerProductID, &p.Quantity); err != nil {
		return nil, err
	}
	if columns["set_code"] {
		_ = r.db.QueryRowContext(ctx, `SELECT COALESCE(set_code, '') FROM product_projection WHERE product_id = ?`, productID).Scan(&p.SetCode)
	}
	if columns["resolved_set_name"] {
		_ = r.db.QueryRowContext(ctx, `SELECT COALESCE(resolved_set_name, '') FROM product_projection WHERE product_id = ?`, productID).Scan(&p.SetName)
	}
	return &p, nil
}

func productToSummary(product PrintingProduct) ProductSummary {
	return ProductSummary{
		Identifier:         product.ID,
		Type:               product.Type,
		PrintingID:         product.PrintingID,
		Name:               product.Name,
		ReleaseDate:        product.ReleaseDate,
		PrintedDate:        product.PrintedDate,
		PrintedLanguage:    product.Language,
		TCGPlayerURL:       product.TCGPlayerURL,
		TCGPlayerProductID: product.TCGPlayerProductID,
		Quantity:           product.Quantity,
		SetCode:            product.SetCode,
		SetName:            product.SetName,
	}
}
