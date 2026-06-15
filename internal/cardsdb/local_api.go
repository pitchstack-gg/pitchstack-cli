package cardsdb

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type CardSummary struct {
	Identifier        string   `json:"identifier"`
	Name              string   `json:"name"`
	Cost              string   `json:"cost,omitempty"`
	Pitch             string   `json:"pitch,omitempty"`
	Power             string   `json:"power,omitempty"`
	Defense           string   `json:"defense,omitempty"`
	Health            string   `json:"health,omitempty"`
	Intelligence      string   `json:"intelligence,omitempty"`
	ColorIdentity     string   `json:"colorIdentity,omitempty"`
	Arcane            string   `json:"arcane,omitempty"`
	Types             []string `json:"types,omitempty"`
	Keywords          []string `json:"keywords,omitempty"`
	FunctionalText    string   `json:"functionalText,omitempty"`
	IsDoubleFacedCard bool     `json:"isDoubleFacedCard"`
	DefaultImageURL   string   `json:"defaultImageUrl,omitempty"`
	PitchSiblingIDs   []string `json:"pitchSiblingIds,omitempty"`
	ReferencedCards   []string `json:"referencedCards,omitempty"`
	CardsReferencedBy []string `json:"cardsReferencedBy,omitempty"`
	Legality          any      `json:"legality,omitempty"`
}

type SearchCardsResponse struct {
	Summaries []CardSummary `json:"summaries"`
	NextToken string        `json:"nextToken,omitempty"`
}

type GetCardResponse struct {
	Summary *CardSummary `json:"summary,omitempty"`
}

type BatchGetCardsResponse struct {
	Cards       map[string]CardSummary `json:"cards"`
	NotFoundIDs []string               `json:"notFoundIds,omitempty"`
}

type PrintingSummary struct {
	Identifier    string           `json:"identifier"`
	CardID        string           `json:"cardId"`
	SetPrintingID string           `json:"setPrintingId,omitempty"`
	PrintingName  string           `json:"printingName,omitempty"`
	Artists       []string         `json:"artists,omitempty"`
	FlavorText    string           `json:"flavorText,omitempty"`
	ImageURL      string           `json:"imageUrl,omitempty"`
	SetID         string           `json:"setId,omitempty"`
	SetName       string           `json:"setName,omitempty"`
	Rarity        string           `json:"rarity,omitempty"`
	Products      []ProductSummary `json:"products,omitempty"`
}

type ListPrintingsResponse struct {
	Summaries []PrintingSummary `json:"summaries"`
	NextToken string            `json:"nextToken,omitempty"`
}

type GetPrintingResponse struct {
	Summary *PrintingSummary `json:"summary,omitempty"`
}

type BatchGetPrintingsResponse struct {
	Printings   map[string]PrintingSummary `json:"printings"`
	NotFoundIDs []string                   `json:"notFoundIds,omitempty"`
}

type ProductSummary struct {
	Identifier         string `json:"identifier"`
	Type               string `json:"type,omitempty"`
	CardID             string `json:"cardId,omitempty"`
	PrintingID         string `json:"printingId,omitempty"`
	ProductGroupID     string `json:"productGroupId,omitempty"`
	Name               string `json:"name,omitempty"`
	PrintedDate        string `json:"printedDate,omitempty"`
	PrintedLanguage    string `json:"printedLanguage,omitempty"`
	ReleaseDate        string `json:"releaseDate,omitempty"`
	Quantity           int    `json:"quantity,omitempty"`
	TCGPlayerURL       string `json:"tcgPlayerUrl,omitempty"`
	TCGPlayerProductID string `json:"tcgPlayerProductId,omitempty"`
	SetCode            string `json:"setCode,omitempty"`
	SetName            string `json:"setName,omitempty"`
}

type ListProductsParams struct {
	Type           string
	SetCode        string
	ProductGroupID string
	CardID         string
	PrintingID     string
	PageSize       int
	NextToken      string
}

type ListProductsResponse struct {
	Summaries []ProductSummary `json:"summaries"`
	NextToken string           `json:"nextToken,omitempty"`
}

type GetProductResponse struct {
	Summary *ProductSummary `json:"summary,omitempty"`
}

type BatchGetProductsResponse struct {
	Products    map[string]ProductSummary `json:"products"`
	NotFoundIDs []string                  `json:"notFoundIds,omitempty"`
}

type SetSummary struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	ReleaseDate string `json:"releaseDate,omitempty"`
}

type ListSetsResponse struct {
	Summaries []SetSummary `json:"summaries"`
	NextToken string       `json:"nextToken,omitempty"`
}

type GetSetResponse struct {
	Summary *SetSummary `json:"summary,omitempty"`
}

type BatchGetSetsResponse struct {
	Sets          map[string]SetSummary `json:"sets"`
	NotFoundCodes []string              `json:"notFoundCodes,omitempty"`
}

type SnapshotResponse struct {
	DBPath    string    `json:"dbPath,omitempty"`
	Metadata  *Metadata `json:"metadata,omitempty"`
	Features  Features  `json:"features"`
	CheckedAt time.Time `json:"checkedAt"`
}

func (r *Repository) SearchCards(ctx context.Context, params SearchCardsParams) (*SearchCardsResponse, error) {
	if !r.features.HasCardCores {
		results, err := r.Search(ctx, SearchParams{Query: params.SearchTerm, Limit: pageSize(params.PageSize) + 1})
		if err != nil {
			return nil, err
		}
		return searchResultsResponse(results, params), nil
	}
	results, next, err := r.searchCardsRich(ctx, params)
	if err != nil {
		return nil, err
	}
	return &SearchCardsResponse{Summaries: results, NextToken: next}, nil
}

func (r *Repository) searchCardsRich(ctx context.Context, params SearchCardsParams) ([]CardSummary, string, error) {
	where := []string{"1=1"}
	args := []any{}
	projectionJoin := ""
	if r.features.HasCardSearchProjection {
		projectionJoin = "LEFT JOIN card_search_projection csp ON csp.card_id = cards.id"
	}
	r.addTextPredicate(&where, &args, strings.TrimSpace(params.SearchTerm))
	r.addColorPredicate(&where, &args, params.ColorIdentity)
	r.addNumericPredicates(&where, &args, params)
	r.addSelectorFilter(&where, &args, selectorFilter{
		Raw:               params.Class,
		ProjectionColumn:  "csp.classes_json",
		ProjectionEnabled: r.features.HasProjectionClassesJSON,
		Table:             "card_core_classes",
		Alias:             "ccc",
		ValueColumn:       "class",
		NormColumn:        "class_norm",
		TableEnabled:      r.features.HasCardCoreClasses,
		NormEnabled:       r.features.HasCardCoreClassNorm,
		MatchCoreIndex:    true,
	})
	r.addSelectorFilter(&where, &args, selectorFilter{
		Raw:               params.Type,
		ProjectionColumn:  "csp.types_json",
		ProjectionEnabled: r.features.HasProjectionTypesJSON,
		Table:             "card_core_types",
		Alias:             "cct",
		ValueColumn:       "type",
		NormColumn:        "type_norm",
		TableEnabled:      r.features.HasCardCoreTypes,
		NormEnabled:       r.features.HasCardCoreTypeNorm,
		MatchCoreIndex:    true,
	})
	r.addSelectorFilter(&where, &args, selectorFilter{
		Raw:               params.Subtype,
		ProjectionColumn:  "csp.subtypes_json",
		ProjectionEnabled: r.features.HasProjectionSubtypesJSON,
		Table:             "card_core_subtypes",
		Alias:             "ccs",
		ValueColumn:       "subtype",
		NormColumn:        "subtype_norm",
		TableEnabled:      r.features.HasCardCoreSubtypes,
		NormEnabled:       r.features.HasCardCoreSubtypeNorm,
		MatchCoreIndex:    true,
	})
	r.addSelectorFilter(&where, &args, selectorFilter{
		Raw:               params.Talent,
		ProjectionColumn:  "csp.talents_json",
		ProjectionEnabled: r.features.HasProjectionTalentsJSON,
		Table:             "card_core_talents",
		Alias:             "ccta",
		ValueColumn:       "talent",
		NormColumn:        "talent_norm",
		TableEnabled:      r.features.HasCardCoreTalents,
		NormEnabled:       r.features.HasCardCoreTalentNorm,
		MatchCoreIndex:    true,
	})
	r.addSelectorFilter(&where, &args, selectorFilter{
		Raw:               params.Keyword,
		ProjectionColumn:  "csp.keywords_json",
		ProjectionEnabled: r.features.HasProjectionKeywordsJSON,
		Table:             "card_keywords",
		Alias:             "ck",
		ValueColumn:       "keyword",
		NormColumn:        "keyword_norm",
		TableEnabled:      r.features.HasCardKeywords,
		NormEnabled:       r.features.HasCardKeywordNorm,
	})
	r.addLegalityFilters(&where, &args, params)
	r.addPrintingFilters(&where, &args, params)
	if params.IsDoubleFaced != nil {
		clause := `EXISTS (
  SELECT 1 FROM printings p2
  WHERE p2.card_id = cards.id
    AND (SELECT COUNT(1) FROM printing_faces pf2 WHERE pf2.printing_id = p2.id) > 1
)`
		if !*params.IsDoubleFaced {
			clause = "NOT " + clause
		}
		where = append(where, clause)
	}

	limit := pageSize(params.PageSize)
	offset := parseOffset(params.NextToken)
	args = append(args, limit+1, offset)
	query := fmt.Sprintf(`
WITH default_core AS (
  SELECT card_id, MIN(core_index) AS core_index
  FROM card_cores
  GROUP BY card_id
)
SELECT
  cards.id,
  %s AS name,
  COALESCE(%s, core.typebox, '') AS type_line,
  COALESCE(%s, core.textbox, '') AS text,
  COALESCE(%s, core.cost, '') AS cost,
  COALESCE(CAST(%s AS TEXT), core.pitch_text, '') AS pitch,
  COALESCE(CAST(%s AS TEXT), core.power, '') AS power,
  COALESCE(CAST(%s AS TEXT), core.defense, '') AS defense,
  COALESCE(CAST(%s AS TEXT), core.life, '') AS health,
  COALESCE(CAST(%s AS TEXT), core.intellect, '') AS intelligence,
  COALESCE(core.chi_text, '') AS arcane,
  COALESCE(%s, core.color, '') AS color_identity,
  %s AS image_url,
  COALESCE(%s, '[]') AS types_json,
  COALESCE(%s, '[]') AS keywords_json,
  %s AS pitch_siblings_json,
  %s AS references_json,
  %s AS referenced_by_json,
  %s AS legality
FROM cards
JOIN default_core dc ON dc.card_id = cards.id
LEFT JOIN card_cores core ON core.card_id = dc.card_id AND core.core_index = dc.core_index
%s
WHERE %s
ORDER BY name COLLATE NOCASE ASC, cards.id ASC
LIMIT ? OFFSET ?`,
		r.displayNameExpr(),
		projectionExpr(r.features.HasCardSearchProjection, "csp.typebox"),
		projectionExpr(r.features.HasCardSearchProjection, "csp.textbox"),
		projectionExpr(r.features.HasCardSearchProjection, "csp.cost_num"),
		projectionExpr(r.features.HasCardSearchProjection, "csp.pitch_value"),
		projectionExpr(r.features.HasCardSearchProjection, "csp.power_num"),
		projectionExpr(r.features.HasCardSearchProjection, "csp.defense_num"),
		projectionExpr(r.features.HasCardSearchProjection, "csp.life_num"),
		projectionExpr(r.features.HasCardSearchProjection, "csp.intellect_num"),
		r.projectionColorExpr(),
		r.defaultImageExpr("cards.id", "csp"),
		projectionExpr(r.features.HasProjectionTypesJSON, "csp.types_json"),
		projectionExpr(r.features.HasProjectionKeywordsJSON, "csp.keywords_json"),
		cardJSONExpr(r.features.HasPitchSiblingsJSON, "cards.pitch_siblings_json"),
		cardJSONExpr(r.features.HasReferencesJSON, "cards.references_json"),
		cardJSONExpr(r.features.HasReferencedByJSON, "cards.referenced_by_json"),
		cardLegalityExpr(r.features.HasCardLegalityJSON),
		projectionJoin,
		strings.Join(where, " AND "),
	)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	out := []CardSummary{}
	for rows.Next() {
		var s CardSummary
		var typeLine, typesJSON, keywordsJSON, pitchSiblingsJSON, referencesJSON, referencedByJSON, legality string
		if err := rows.Scan(&s.Identifier, &s.Name, &typeLine, &s.FunctionalText, &s.Cost, &s.Pitch, &s.Power, &s.Defense, &s.Health, &s.Intelligence, &s.Arcane, &s.ColorIdentity, &s.DefaultImageURL, &typesJSON, &keywordsJSON, &pitchSiblingsJSON, &referencesJSON, &referencedByJSON, &legality); err != nil {
			return nil, "", err
		}
		s.Types = parseStringArray(typesJSON)
		if len(s.Types) == 0 && typeLine != "" {
			s.Types = splitTypeLine(typeLine)
		}
		s.Keywords = parseStringArray(keywordsJSON)
		s.PitchSiblingIDs = parseStringArray(pitchSiblingsJSON)
		s.ReferencedCards = parseStringArray(referencesJSON)
		s.CardsReferencedBy = parseStringArray(referencedByJSON)
		s.Legality = parseJSONValue(legality)
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	next := ""
	if len(out) > limit {
		out = out[:limit]
		next = strconv.Itoa(offset + limit)
	}
	return out, next, nil
}

func (r *Repository) GetCardSummary(ctx context.Context, cardID string) (*GetCardResponse, error) {
	detail, err := r.GetCard(ctx, cardID)
	if err != nil {
		if err == sql.ErrNoRows {
			return &GetCardResponse{}, nil
		}
		return nil, err
	}
	summary := cardDetailToSummary(detail)
	related, _ := r.ListRelatedCards(ctx, detail.ID)
	if related != nil {
		for _, item := range related.References {
			summary.ReferencedCards = append(summary.ReferencedCards, item.ID)
		}
		for _, item := range related.ReferencedBy {
			summary.CardsReferencedBy = append(summary.CardsReferencedBy, item.ID)
		}
	}
	return &GetCardResponse{Summary: &summary}, nil
}

func (r *Repository) BatchGetCardSummaries(ctx context.Context, ids []string) (*BatchGetCardsResponse, error) {
	out := &BatchGetCardsResponse{Cards: map[string]CardSummary{}}
	for _, id := range uniqueStrings(ids, 200) {
		resp, err := r.GetCardSummary(ctx, id)
		if err != nil {
			return nil, err
		}
		if resp.Summary == nil {
			out.NotFoundIDs = append(out.NotFoundIDs, id)
			continue
		}
		out.Cards[id] = *resp.Summary
	}
	return out, nil
}

func (r *Repository) ListPrintingSummaries(ctx context.Context, cardID string, pageSizeValue int, nextToken string) (*ListPrintingsResponse, error) {
	printings, err := r.ListPrintings(ctx, cardID)
	if err != nil {
		return nil, err
	}
	offset := parseOffset(nextToken)
	limit := pageSize(pageSizeValue)
	end := offset + limit
	next := ""
	if end < len(printings) {
		next = strconv.Itoa(end)
	} else {
		end = len(printings)
	}
	if offset > len(printings) {
		offset = len(printings)
	}
	out := make([]PrintingSummary, 0, end-offset)
	for _, printing := range printings[offset:end] {
		out = append(out, printingToSummary(cardID, printing, nil))
	}
	return &ListPrintingsResponse{Summaries: out, NextToken: next}, nil
}

func (r *Repository) GetPrintingSummary(ctx context.Context, printingID string) (*GetPrintingResponse, error) {
	printingID = strings.TrimSpace(printingID)
	if printingID == "" || !r.features.HasPrintings {
		return &GetPrintingResponse{}, nil
	}
	cardID, printing, err := r.getPrinting(ctx, printingID)
	if err != nil {
		if err == sql.ErrNoRows {
			return &GetPrintingResponse{}, nil
		}
		return nil, err
	}
	products, _ := r.ListCardProducts(ctx, cardID)
	summary := printingToSummary(cardID, *printing, products)
	return &GetPrintingResponse{Summary: &summary}, nil
}

func (r *Repository) BatchGetPrintingSummaries(ctx context.Context, ids []string) (*BatchGetPrintingsResponse, error) {
	out := &BatchGetPrintingsResponse{Printings: map[string]PrintingSummary{}}
	for _, id := range uniqueStrings(ids, 200) {
		resp, err := r.GetPrintingSummary(ctx, id)
		if err != nil {
			return nil, err
		}
		if resp.Summary == nil {
			out.NotFoundIDs = append(out.NotFoundIDs, id)
			continue
		}
		out.Printings[id] = *resp.Summary
	}
	return out, nil
}

func (r *Repository) ListPrintingsForSetNumber(ctx context.Context, setNumber string) (*ListPrintingsResponse, error) {
	setNumber = strings.TrimSpace(setNumber)
	if setNumber == "" || !r.features.HasPrintings {
		return &ListPrintingsResponse{}, nil
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT DISTINCT COALESCE(card_id, ''), id
FROM printings
WHERE id = ? OR id LIKE (? || '-%') OR id LIKE ('%_' || ?) OR id LIKE ('%_' || ? || '-%')
ORDER BY id ASC`, setNumber, setNumber, setNumber, setNumber)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PrintingSummary{}
	for rows.Next() {
		var cardID, id string
		if err := rows.Scan(&cardID, &id); err != nil {
			return nil, err
		}
		resp, err := r.GetPrintingSummary(ctx, id)
		if err != nil {
			return nil, err
		}
		if resp.Summary != nil {
			if resp.Summary.CardID == "" {
				resp.Summary.CardID = cardID
			}
			out = append(out, *resp.Summary)
		}
	}
	return &ListPrintingsResponse{Summaries: out}, rows.Err()
}

func (r *Repository) GetProductSummary(ctx context.Context, productID string) (*GetProductResponse, error) {
	product, err := r.productByID(ctx, productID)
	if err != nil {
		if err == sql.ErrNoRows {
			return &GetProductResponse{}, nil
		}
		return nil, err
	}
	return &GetProductResponse{Summary: product}, nil
}

func (r *Repository) BatchGetProductSummaries(ctx context.Context, ids []string) (*BatchGetProductsResponse, error) {
	out := &BatchGetProductsResponse{Products: map[string]ProductSummary{}}
	for _, id := range uniqueStrings(ids, 200) {
		resp, err := r.GetProductSummary(ctx, id)
		if err != nil {
			return nil, err
		}
		if resp.Summary == nil {
			out.NotFoundIDs = append(out.NotFoundIDs, id)
			continue
		}
		out.Products[id] = *resp.Summary
	}
	return out, nil
}

func (r *Repository) ListProductSummaries(ctx context.Context, params ListProductsParams) (*ListProductsResponse, error) {
	if !r.features.HasProductProjection {
		return &ListProductsResponse{}, nil
	}
	where := []string{"1=1"}
	args := []any{}
	addEq := func(field, value string) {
		if strings.TrimSpace(value) == "" {
			return
		}
		where = append(where, "LOWER(COALESCE("+field+", '')) = ?")
		args = append(args, strings.ToLower(strings.TrimSpace(value)))
	}
	addEq("type", params.Type)
	addEq("set_code", params.SetCode)
	addEq("product_group_id", params.ProductGroupID)
	addEq("card_id", params.CardID)
	addEq("printing_id", params.PrintingID)
	limit := pageSize(params.PageSize)
	offset := parseOffset(params.NextToken)
	args = append(args, limit+1, offset)
	rows, err := r.db.QueryContext(ctx, `
SELECT product_id
FROM product_projection
WHERE `+strings.Join(where, " AND ")+`
ORDER BY COALESCE(release_date, product_group_release_date, printed_date, '') DESC, product_id ASC
LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ProductSummary{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		product, err := r.productByID(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, *product)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	next := ""
	if len(out) > limit {
		out = out[:limit]
		next = strconv.Itoa(offset + limit)
	}
	return &ListProductsResponse{Summaries: out, NextToken: next}, nil
}

func (r *Repository) GetSetSummary(ctx context.Context, code string) (*GetSetResponse, error) {
	if !r.features.HasSets {
		return &GetSetResponse{}, nil
	}
	code = strings.TrimSpace(code)
	var s SetSummary
	err := r.db.QueryRowContext(ctx, `SELECT COALESCE(code, ''), COALESCE(name, code, ''), COALESCE(release_date, '') FROM sets WHERE LOWER(code) = LOWER(?) LIMIT 1`, code).Scan(&s.Code, &s.Name, &s.ReleaseDate)
	if err != nil {
		if err == sql.ErrNoRows {
			return &GetSetResponse{}, nil
		}
		return nil, err
	}
	return &GetSetResponse{Summary: &s}, nil
}

func (r *Repository) ListSetSummaries(ctx context.Context, pageSizeValue int, nextToken string) (*ListSetsResponse, error) {
	if !r.features.HasSets {
		return &ListSetsResponse{}, nil
	}
	limit := pageSize(pageSizeValue)
	offset := parseOffset(nextToken)
	rows, err := r.db.QueryContext(ctx, `SELECT COALESCE(code, ''), COALESCE(name, code, ''), COALESCE(release_date, '') FROM sets ORDER BY release_date DESC, code ASC LIMIT ? OFFSET ?`, limit+1, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SetSummary{}
	for rows.Next() {
		var s SetSummary
		if err := rows.Scan(&s.Code, &s.Name, &s.ReleaseDate); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	next := ""
	if len(out) > limit {
		out = out[:limit]
		next = strconv.Itoa(offset + limit)
	}
	return &ListSetsResponse{Summaries: out, NextToken: next}, nil
}

func (r *Repository) BatchGetSetSummaries(ctx context.Context, codes []string) (*BatchGetSetsResponse, error) {
	out := &BatchGetSetsResponse{Sets: map[string]SetSummary{}}
	for _, code := range uniqueStrings(codes, 200) {
		resp, err := r.GetSetSummary(ctx, code)
		if err != nil {
			return nil, err
		}
		if resp.Summary == nil {
			out.NotFoundCodes = append(out.NotFoundCodes, code)
			continue
		}
		out.Sets[code] = *resp.Summary
	}
	return out, nil
}

func (r *Repository) Snapshot(dbPath string, meta *Metadata) *SnapshotResponse {
	return &SnapshotResponse{DBPath: dbPath, Metadata: meta, Features: r.features, CheckedAt: time.Now().UTC()}
}
