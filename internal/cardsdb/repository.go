package cardsdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

type Repository struct {
	db       *sql.DB
	features Features
}

type Features struct {
	HasCards                         bool
	HasCardCores                     bool
	HasCardSearchProjection          bool
	HasOpenNameFTS                   bool
	HasOpenRulesFTS                  bool
	HasOpenTextFTS                   bool
	HasPrintings                     bool
	HasPrintingFaces                 bool
	HasSets                          bool
	HasCardReferences                bool
	HasCardPitchSiblings             bool
	HasCardCoreClasses               bool
	HasCardCoreTypes                 bool
	HasCardCoreSubtypes              bool
	HasCardCoreTalents               bool
	HasCardCoreClassNorm             bool
	HasCardCoreTypeNorm              bool
	HasCardCoreSubtypeNorm           bool
	HasCardCoreTalentNorm            bool
	HasCardKeywords                  bool
	HasCardKeywordNorm               bool
	HasCardLegalities                bool
	HasProductProjection             bool
	HasProducts                      bool
	HasProductGroups                 bool
	HasCardLegalityJSON              bool
	HasReferencesJSON                bool
	HasReferencedByJSON              bool
	HasPitchSiblingsJSON             bool
	HasProjectionImageSmall          bool
	HasProjectionImageNormal         bool
	HasProjectionImageLarge          bool
	HasProjectionImageCrop           bool
	HasProjectionImageCropSmall      bool
	HasProjectionImageCropMedium     bool
	HasProjectionImageCropXlarge     bool
	HasProjectionImageCropColor      bool
	HasProjectionClassesJSON         bool
	HasProjectionTalentsJSON         bool
	HasProjectionTypesJSON           bool
	HasProjectionSubtypesJSON        bool
	HasProjectionKeywordsJSON        bool
	HasProjectionColor               bool
	HasProjectionColorNorm           bool
	HasProjectionPreferredPrintingID bool
	HasProjectionBlitzLegal          bool
	HasProjectionCCLegal             bool
	HasProjectionCommonerLegal       bool
	HasProjectionProjectBlueLegal    bool
	HasPrintingImageCrop             bool
	HasPrintingImageCropSmall        bool
	HasPrintingImageCropMedium       bool
	HasPrintingImageCropXlarge       bool
	HasPrintingRulesText             bool
}

type SearchParams struct {
	Query string
	Limit int
}

type SearchResult struct {
	ID         string
	Name       string
	TypeLine   string
	Text       string
	Cost       string
	Pitch      string
	Power      string
	Defense    string
	Health     string
	ImageURL   string
	ArtURL     string
	SetName    string
	Rarity     string
	PrintingID string
}

func (r SearchResult) FilterValue() string {
	return strings.TrimSpace(r.Name + " " + r.ID + " " + r.TypeLine)
}

type CardDetail struct {
	ID           string
	Name         string
	TypeLine     string
	Text         string
	PrintedText  string
	Cost         string
	Pitch        string
	Power        string
	Defense      string
	Health       string
	Intelligence string
	Arcane       string
	ImageURL     string
	ArtURL       string
	PrimaryColor string
	PrintingID   string
	Printing     string
	SetCode      string
	SetName      string
	Rarity       string
	Artists      []string
	FlavorText   string
	Legality     string
	Classes      []string
	Talents      []string
	BaseTypes    []string
	Subtypes     []string
	Keywords     []string
}

type CardPrinting struct {
	ID          string
	Name        string
	SetCode     string
	SetName     string
	Rarity      string
	Language    string
	ImageURL    string
	ArtURL      string
	Artists     []string
	FlavorText  string
	PrintedText string
}

type RelatedCards struct {
	Siblings     []RelatedCard
	References   []RelatedCard
	ReferencedBy []RelatedCard
}

type RelatedCard struct {
	ID       string
	Name     string
	TypeLine string
	ArtURL   string
}

type PrintingProduct struct {
	ID                 string
	Type               string
	PrintingID         string
	SetCode            string
	SetName            string
	Name               string
	ReleaseDate        string
	PrintedDate        string
	Language           string
	TCGPlayerURL       string
	TCGPlayerProductID string
	Quantity           int
}

func OpenRepository(dbPath string) (*Repository, error) {
	u := url.URL{Scheme: "file", Path: dbPath}
	db, err := sql.Open("sqlite", u.String()+"?mode=ro")
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`PRAGMA query_only = ON; PRAGMA foreign_keys = ON; PRAGMA temp_store = MEMORY; PRAGMA cache_size = -20000;`); err != nil {
		_ = db.Close()
		return nil, err
	}
	repo := &Repository{db: db}
	features, err := repo.detectFeatures(context.Background())
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	if !features.HasCards {
		_ = db.Close()
		return nil, fmt.Errorf("%s does not look like a Pitchstack cards database", filepath.Base(dbPath))
	}
	repo.features = features
	return repo, nil
}

func (r *Repository) Close() error {
	if r == nil || r.db == nil {
		return nil
	}
	return r.db.Close()
}

func (r *Repository) Features() Features {
	if r == nil {
		return Features{}
	}
	return r.features
}

func (r *Repository) Search(ctx context.Context, params SearchParams) ([]SearchResult, error) {
	limit := params.Limit
	if limit <= 0 || limit > 100 {
		limit = 40
	}
	query := strings.TrimSpace(params.Query)
	if r.features.HasCardCores {
		return r.searchRich(ctx, query, limit)
	}
	return r.searchSimple(ctx, query, limit)
}

func (r *Repository) GetCard(ctx context.Context, cardID string) (*CardDetail, error) {
	cardID = strings.TrimSpace(cardID)
	if cardID == "" {
		return nil, fmt.Errorf("card id is required")
	}
	var detail *CardDetail
	var err error
	if r.features.HasCardCores {
		detail, err = r.getRichCard(ctx, cardID)
	} else {
		detail, err = r.getSimpleCard(ctx, cardID)
	}
	if err != nil {
		return nil, err
	}
	if detail == nil {
		return nil, sql.ErrNoRows
	}
	printing, err := r.bestPrinting(ctx, cardID)
	if err == nil && printing != nil {
		mergePrinting(detail, printing)
	}
	return detail, nil
}

func (r *Repository) ListPrintings(ctx context.Context, cardID string) ([]CardPrinting, error) {
	cardID = strings.TrimSpace(cardID)
	if cardID == "" {
		return nil, fmt.Errorf("card id is required")
	}
	if !r.features.HasPrintings || !r.features.HasPrintingFaces {
		return nil, nil
	}
	setNameExpr := "''"
	joinSets := ""
	if r.features.HasSets {
		setNameExpr = "COALESCE(s.name, '')"
		joinSets = "LEFT JOIN sets s ON s.code = p.set_code"
	}
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
SELECT
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
WHERE p.card_id = ?
  AND (pf.face_id IS NULL OR pf.face_id NOT LIKE '%%_BACK')
ORDER BY
  (COALESCE(p.language, pf.language, 'en') != 'en') ASC,
  COALESCE(p.set_code, '') ASC,
  p.id ASC,
  pf.face_id ASC`,
		setNameExpr, r.printingFaceCropExpr(), r.printingRulesTextExpr(), joinSets), cardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CardPrinting{}
	seen := map[string]bool{}
	for rows.Next() {
		var item CardPrinting
		var artist string
		if err := rows.Scan(&item.ID, &item.Name, &item.SetCode, &item.SetName, &item.Rarity, &item.Language, &item.ImageURL, &item.ArtURL, &artist, &item.FlavorText, &item.PrintedText); err != nil {
			return nil, err
		}
		if item.ID == "" || seen[item.ID] {
			continue
		}
		seen[item.ID] = true
		if artist != "" {
			item.Artists = []string{artist}
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *Repository) ListRelatedCards(ctx context.Context, cardID string) (*RelatedCards, error) {
	cardID = strings.TrimSpace(cardID)
	if cardID == "" {
		return nil, fmt.Errorf("card id is required")
	}
	related := &RelatedCards{}
	var err error
	siblingIDs := []string{}
	if r.features.HasPitchSiblingsJSON {
		siblingIDs, err = r.cardJSONIDs(ctx, cardID, "pitch_siblings_json")
		if err != nil {
			return nil, err
		}
	} else if r.features.HasCardPitchSiblings {
		siblingIDs, err = r.relationIDs(ctx, `
SELECT sibling_card_id
FROM card_pitch_siblings
WHERE card_id = ?`, cardID)
		if err != nil {
			return nil, err
		}
	}
	related.Siblings, err = r.relatedCards(ctx, siblingIDs)
	if err != nil {
		return nil, err
	}

	referenceIDs := []string{}
	if r.features.HasReferencesJSON {
		referenceIDs, err = r.cardJSONIDs(ctx, cardID, "references_json")
		if err != nil {
			return nil, err
		}
	} else if r.features.HasCardReferences {
		referenceIDs, err = r.relationIDs(ctx, `
SELECT ref_card_id
FROM card_references
WHERE card_id = ?`, cardID)
		if err != nil {
			return nil, err
		}
	}
	related.References, err = r.relatedCards(ctx, referenceIDs)
	if err != nil {
		return nil, err
	}

	referencedByIDs := []string{}
	if r.features.HasReferencedByJSON {
		referencedByIDs, err = r.cardJSONIDs(ctx, cardID, "referenced_by_json")
		if err != nil {
			return nil, err
		}
	} else if r.features.HasCardReferences {
		referencedByIDs, err = r.relationIDs(ctx, `
SELECT card_id
FROM card_references
WHERE ref_card_id = ?`, cardID)
		if err != nil {
			return nil, err
		}
	}
	related.ReferencedBy, err = r.relatedCards(ctx, referencedByIDs)
	if err != nil {
		return nil, err
	}
	return related, nil
}

func (r *Repository) ListCardProducts(ctx context.Context, cardID string) ([]PrintingProduct, error) {
	cardID = strings.TrimSpace(cardID)
	if cardID == "" {
		return nil, fmt.Errorf("card id is required")
	}
	if !r.features.HasProductProjection {
		return nil, nil
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT
  COALESCE(product_id, ''),
  COALESCE(type, ''),
  COALESCE(printing_id, ''),
  COALESCE(set_code, ''),
  COALESCE(resolved_set_name, product_group_name, ''),
  COALESCE(name, ''),
  COALESCE(release_date, product_group_release_date, printed_date, ''),
  COALESCE(printed_date, ''),
  COALESCE(printed_language, ''),
  COALESCE(tcgplayer_url, ''),
  COALESCE(tcgplayer_product_id, ''),
  COALESCE(quantity, 0)
FROM product_projection
WHERE card_id = ?
ORDER BY COALESCE(release_date, product_group_release_date, printed_date, '') DESC, product_id ASC
LIMIT 24`, cardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PrintingProduct{}
	for rows.Next() {
		var item PrintingProduct
		if err := rows.Scan(
			&item.ID, &item.Type, &item.PrintingID, &item.SetCode, &item.SetName, &item.Name,
			&item.ReleaseDate, &item.PrintedDate, &item.Language, &item.TCGPlayerURL,
			&item.TCGPlayerProductID, &item.Quantity,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *Repository) detectFeatures(ctx context.Context) (Features, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT lower(name) FROM sqlite_master WHERE type IN ('table', 'view')`)
	if err != nil {
		return Features{}, err
	}
	defer rows.Close()
	names := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return Features{}, err
		}
		names[name] = true
	}
	if err := rows.Err(); err != nil {
		return Features{}, err
	}
	cardColumns, err := r.columnNames(ctx, "cards")
	if err != nil {
		return Features{}, err
	}
	projectionColumns := map[string]bool{}
	if names["card_search_projection"] {
		projectionColumns, err = r.columnNames(ctx, "card_search_projection")
		if err != nil {
			return Features{}, err
		}
	}
	printingFaceColumns := map[string]bool{}
	if names["printing_faces"] {
		printingFaceColumns, err = r.columnNames(ctx, "printing_faces")
		if err != nil {
			return Features{}, err
		}
	}
	coreClassColumns := map[string]bool{}
	if names["card_core_classes"] {
		coreClassColumns, err = r.columnNames(ctx, "card_core_classes")
		if err != nil {
			return Features{}, err
		}
	}
	coreTypeColumns := map[string]bool{}
	if names["card_core_types"] {
		coreTypeColumns, err = r.columnNames(ctx, "card_core_types")
		if err != nil {
			return Features{}, err
		}
	}
	coreSubtypeColumns := map[string]bool{}
	if names["card_core_subtypes"] {
		coreSubtypeColumns, err = r.columnNames(ctx, "card_core_subtypes")
		if err != nil {
			return Features{}, err
		}
	}
	coreTalentColumns := map[string]bool{}
	if names["card_core_talents"] {
		coreTalentColumns, err = r.columnNames(ctx, "card_core_talents")
		if err != nil {
			return Features{}, err
		}
	}
	keywordColumns := map[string]bool{}
	if names["card_keywords"] {
		keywordColumns, err = r.columnNames(ctx, "card_keywords")
		if err != nil {
			return Features{}, err
		}
	}
	return Features{
		HasCards:                         names["cards"],
		HasCardCores:                     names["card_cores"],
		HasCardSearchProjection:          names["card_search_projection"],
		HasOpenNameFTS:                   names["open_name_fts"],
		HasOpenRulesFTS:                  names["open_rules_fts"],
		HasOpenTextFTS:                   names["open_text_fts"],
		HasPrintings:                     names["printings"],
		HasPrintingFaces:                 names["printing_faces"],
		HasSets:                          names["sets"],
		HasCardReferences:                names["card_references"],
		HasCardPitchSiblings:             names["card_pitch_siblings"],
		HasCardCoreClasses:               names["card_core_classes"] && coreClassColumns["class"],
		HasCardCoreTypes:                 names["card_core_types"] && coreTypeColumns["type"],
		HasCardCoreSubtypes:              names["card_core_subtypes"] && coreSubtypeColumns["subtype"],
		HasCardCoreTalents:               names["card_core_talents"] && coreTalentColumns["talent"],
		HasCardCoreClassNorm:             coreClassColumns["class_norm"],
		HasCardCoreTypeNorm:              coreTypeColumns["type_norm"],
		HasCardCoreSubtypeNorm:           coreSubtypeColumns["subtype_norm"],
		HasCardCoreTalentNorm:            coreTalentColumns["talent_norm"],
		HasCardKeywords:                  names["card_keywords"] && keywordColumns["keyword"],
		HasCardKeywordNorm:               keywordColumns["keyword_norm"],
		HasCardLegalities:                names["card_legalities"],
		HasProductProjection:             names["product_projection"],
		HasProducts:                      names["products"],
		HasProductGroups:                 names["product_groups"],
		HasCardLegalityJSON:              cardColumns["card_legality_json"],
		HasReferencesJSON:                cardColumns["references_json"],
		HasReferencedByJSON:              cardColumns["referenced_by_json"],
		HasPitchSiblingsJSON:             cardColumns["pitch_siblings_json"],
		HasProjectionImageSmall:          projectionColumns["image_small"],
		HasProjectionImageNormal:         projectionColumns["image_normal"],
		HasProjectionImageLarge:          projectionColumns["image_large"],
		HasProjectionImageCrop:           projectionColumns["image_crop"],
		HasProjectionImageCropSmall:      projectionColumns["image_crop_small"],
		HasProjectionImageCropMedium:     projectionColumns["image_crop_medium"],
		HasProjectionImageCropXlarge:     projectionColumns["image_crop_xlarge"],
		HasProjectionImageCropColor:      projectionColumns["image_crop_primary_color"],
		HasProjectionClassesJSON:         projectionColumns["classes_json"],
		HasProjectionTalentsJSON:         projectionColumns["talents_json"],
		HasProjectionTypesJSON:           projectionColumns["types_json"],
		HasProjectionSubtypesJSON:        projectionColumns["subtypes_json"],
		HasProjectionKeywordsJSON:        projectionColumns["keywords_json"],
		HasProjectionColor:               projectionColumns["color"],
		HasProjectionColorNorm:           projectionColumns["color_norm"],
		HasProjectionPreferredPrintingID: projectionColumns["preferred_printing_id"],
		HasProjectionBlitzLegal:          projectionColumns["is_blitz_legal"],
		HasProjectionCCLegal:             projectionColumns["is_classic_constructed_legal"],
		HasProjectionCommonerLegal:       projectionColumns["is_commoner_legal"],
		HasProjectionProjectBlueLegal:    projectionColumns["is_silver_age_legal"],
		HasPrintingImageCrop:             printingFaceColumns["image_crop"],
		HasPrintingImageCropSmall:        printingFaceColumns["image_crop_small"],
		HasPrintingImageCropMedium:       printingFaceColumns["image_crop_medium"],
		HasPrintingImageCropXlarge:       printingFaceColumns["image_crop_xlarge"],
		HasPrintingRulesText:             printingFaceColumns["rules_text"],
	}, nil
}

func (r *Repository) columnNames(ctx context.Context, table string) (map[string]bool, error) {
	rows, err := r.db.QueryContext(ctx, `PRAGMA table_info(`+table+`)`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			return nil, err
		}
		columns[strings.ToLower(strings.TrimSpace(name))] = true
	}
	return columns, rows.Err()
}

func (r *Repository) searchRich(ctx context.Context, search string, limit int) ([]SearchResult, error) {
	where := []string{"1=1"}
	args := []any{}
	projectionJoin := ""
	if r.features.HasCardSearchProjection {
		projectionJoin = "LEFT JOIN card_search_projection csp ON csp.card_id = cards.id"
	}
	if search != "" {
		if r.features.HasCardSearchProjection {
			where = append(where, "(cards.id LIKE ? COLLATE NOCASE OR COALESCE(csp.core_name, core.name, '') LIKE ? COLLATE NOCASE OR COALESCE(csp.textbox, core.textbox, '') LIKE ? COLLATE NOCASE)")
			like := "%" + search + "%"
			args = append(args, like, like, like)
		} else if fts := ftsMatch(search); fts != "" && (r.features.HasOpenNameFTS || r.features.HasOpenRulesFTS || r.features.HasOpenTextFTS) {
			clauses := []string{}
			if r.features.HasOpenNameFTS {
				clauses = append(clauses, "EXISTS (SELECT 1 FROM open_name_fts WHERE open_name_fts.card_id = cards.id AND open_name_fts MATCH ?)")
				args = append(args, fts)
			}
			if r.features.HasOpenRulesFTS {
				clauses = append(clauses, "EXISTS (SELECT 1 FROM open_rules_fts WHERE open_rules_fts.card_id = cards.id AND open_rules_fts MATCH ?)")
				args = append(args, fts)
			}
			if r.features.HasOpenTextFTS && len(clauses) == 0 {
				clauses = append(clauses, "EXISTS (SELECT 1 FROM open_text_fts WHERE open_text_fts.card_id = cards.id AND open_text_fts MATCH ?)")
				args = append(args, fts)
			}
			where = append(where, "("+strings.Join(clauses, " OR ")+")")
		} else {
			where = append(where, "(cards.id LIKE ? COLLATE NOCASE OR core.name LIKE ? COLLATE NOCASE OR core.textbox LIKE ? COLLATE NOCASE)")
			like := "%" + search + "%"
			args = append(args, like, like, like)
		}
	}
	args = append(args, limit)

	sqlText := fmt.Sprintf(`
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
  %s AS image_url,
  %s AS art_url
FROM cards
JOIN default_core dc ON dc.card_id = cards.id
LEFT JOIN card_cores core ON core.card_id = dc.card_id AND core.core_index = dc.core_index
%s
WHERE %s
ORDER BY name COLLATE NOCASE ASC
LIMIT ?`,
		r.displayNameExpr(),
		projectionExpr(r.features.HasCardSearchProjection, "csp.typebox"),
		projectionExpr(r.features.HasCardSearchProjection, "csp.textbox"),
		projectionExpr(r.features.HasCardSearchProjection, "csp.cost_num"),
		projectionExpr(r.features.HasCardSearchProjection, "csp.pitch_value"),
		projectionExpr(r.features.HasCardSearchProjection, "csp.power_num"),
		projectionExpr(r.features.HasCardSearchProjection, "csp.defense_num"),
		projectionExpr(r.features.HasCardSearchProjection, "csp.life_num"),
		r.defaultImageExpr("cards.id", "csp"),
		r.cropImageExpr("cards.id", "csp"),
		projectionJoin,
		strings.Join(where, " AND "),
	)
	rows, err := r.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSearchRows(rows)
}

func (r *Repository) searchSimple(ctx context.Context, search string, limit int) ([]SearchResult, error) {
	where := "1=1"
	args := []any{}
	if search != "" {
		where = "(id LIKE ? COLLATE NOCASE OR COALESCE(name, '') LIKE ? COLLATE NOCASE OR COALESCE(functional_text, '') LIKE ? COLLATE NOCASE)"
		like := "%" + search + "%"
		args = append(args, like, like, like)
	}
	args = append(args, limit)
	rows, err := r.db.QueryContext(ctx, `
SELECT
  id,
  COALESCE(name, id) AS name,
  COALESCE(types, '') AS type_line,
  COALESCE(functional_text, '') AS text,
  COALESCE(cost, '') AS cost,
  COALESCE(pitch, '') AS pitch,
  COALESCE(power, '') AS power,
  COALESCE(defense, '') AS defense,
  COALESCE(health, '') AS health,
  COALESCE(default_image_url, '') AS image_url,
  COALESCE(default_image_url, '') AS art_url
FROM cards
WHERE `+where+`
ORDER BY name COLLATE NOCASE ASC
LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSearchRows(rows)
}

func (r *Repository) getRichCard(ctx context.Context, cardID string) (*CardDetail, error) {
	projectionJoin := ""
	if r.features.HasCardSearchProjection {
		projectionJoin = "LEFT JOIN card_search_projection csp ON csp.card_id = cards.id"
	}
	sqlText := fmt.Sprintf(`
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
  %s AS legality,
  %s AS image_url,
  COALESCE(%s, '') AS art_url,
  COALESCE(%s, '') AS primary_color,
  COALESCE(%s, '[]') AS classes_json,
  COALESCE(%s, '[]') AS talents_json,
  COALESCE(%s, '[]') AS types_json,
  COALESCE(%s, '[]') AS subtypes_json,
  COALESCE(%s, '[]') AS keywords_json
FROM cards
JOIN default_core dc ON dc.card_id = cards.id
LEFT JOIN card_cores core ON core.card_id = dc.card_id AND core.core_index = dc.core_index
%s
WHERE cards.id = ?
LIMIT 1`,
		r.displayNameExpr(),
		projectionExpr(r.features.HasCardSearchProjection, "csp.typebox"),
		projectionExpr(r.features.HasCardSearchProjection, "csp.textbox"),
		projectionExpr(r.features.HasCardSearchProjection, "csp.cost_num"),
		projectionExpr(r.features.HasCardSearchProjection, "csp.pitch_value"),
		projectionExpr(r.features.HasCardSearchProjection, "csp.power_num"),
		projectionExpr(r.features.HasCardSearchProjection, "csp.defense_num"),
		projectionExpr(r.features.HasCardSearchProjection, "csp.life_num"),
		projectionExpr(r.features.HasCardSearchProjection, "csp.intellect_num"),
		cardLegalityExpr(r.features.HasCardLegalityJSON),
		r.defaultImageExpr("cards.id", "csp"),
		r.cropImageExpr("cards.id", "csp"),
		projectionExpr(r.features.HasProjectionImageCropColor, "csp.image_crop_primary_color"),
		projectionExpr(r.features.HasProjectionClassesJSON, "csp.classes_json"),
		projectionExpr(r.features.HasProjectionTalentsJSON, "csp.talents_json"),
		projectionExpr(r.features.HasProjectionTypesJSON, "csp.types_json"),
		projectionExpr(r.features.HasProjectionSubtypesJSON, "csp.subtypes_json"),
		projectionExpr(r.features.HasProjectionKeywordsJSON, "csp.keywords_json"),
		projectionJoin,
	)
	var d CardDetail
	var classesJSON, talentsJSON, typesJSON, subtypesJSON, keywordsJSON string
	if err := r.db.QueryRowContext(ctx, sqlText, cardID).Scan(
		&d.ID, &d.Name, &d.TypeLine, &d.Text, &d.Cost, &d.Pitch, &d.Power, &d.Defense, &d.Health,
		&d.Intelligence, &d.Arcane, &d.Legality, &d.ImageURL, &d.ArtURL, &d.PrimaryColor,
		&classesJSON, &talentsJSON, &typesJSON, &subtypesJSON, &keywordsJSON,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	d.Legality = compactJSON(d.Legality)
	d.Classes = parseStringArray(classesJSON)
	d.Talents = parseStringArray(talentsJSON)
	d.BaseTypes = parseStringArray(typesJSON)
	d.Subtypes = parseStringArray(subtypesJSON)
	d.Keywords = parseStringArray(keywordsJSON)
	return &d, nil
}

func (r *Repository) getSimpleCard(ctx context.Context, cardID string) (*CardDetail, error) {
	var d CardDetail
	err := r.db.QueryRowContext(ctx, `
SELECT
  id,
  COALESCE(name, id),
  COALESCE(types, ''),
  COALESCE(functional_text, ''),
  COALESCE(cost, ''),
  COALESCE(pitch, ''),
  COALESCE(power, ''),
  COALESCE(defense, ''),
  COALESCE(health, ''),
  COALESCE(intelligence, ''),
  COALESCE(arcane, ''),
  COALESCE(default_image_url, ''),
  COALESCE(default_image_url, '')
FROM cards
WHERE id = ?
	LIMIT 1`, cardID).Scan(
		&d.ID, &d.Name, &d.TypeLine, &d.Text, &d.Cost, &d.Pitch, &d.Power, &d.Defense, &d.Health,
		&d.Intelligence, &d.Arcane, &d.ImageURL, &d.ArtURL,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}

func (r *Repository) bestPrinting(ctx context.Context, cardID string) (*CardDetail, error) {
	if !r.features.HasPrintings {
		return nil, sql.ErrNoRows
	}
	if r.features.HasPrintingFaces {
		setNameExpr := "''"
		joinSets := ""
		if r.features.HasSets {
			setNameExpr = "COALESCE(s.name, '')"
			joinSets = "LEFT JOIN sets s ON s.code = p.set_code"
		}
		cropExpr := r.printingFaceCropExpr()
		rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
SELECT
  p.id,
  COALESCE(p.set_code, ''),
  %s AS set_name,
  COALESCE(p.rarity, ''),
  COALESCE(pf.name, p.product_name, ''),
  COALESCE(pf.image_large, pf.image_normal, pf.image_small, ''),
  COALESCE(%s, '') AS art_url,
  COALESCE(pf.artist, ''),
  COALESCE(pf.flavor_text, ''),
  COALESCE(%s, '') AS printed_text
FROM printings p
LEFT JOIN printing_faces pf ON pf.printing_id = p.id
%s
WHERE p.card_id = ?
ORDER BY
  (COALESCE(p.language, pf.language, 'en') != 'en') ASC,
  COALESCE(p.set_code, '') ASC,
  p.id ASC,
  (pf.face_id LIKE '%%_BACK') ASC,
  pf.face_id ASC
 LIMIT 1`, setNameExpr, cropExpr, r.printingRulesTextExpr(), joinSets), cardID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		if rows.Next() {
			var d CardDetail
			var artist string
			if err := rows.Scan(&d.PrintingID, &d.SetCode, &d.SetName, &d.Rarity, &d.Printing, &d.ImageURL, &d.ArtURL, &artist, &d.FlavorText, &d.PrintedText); err != nil {
				return nil, err
			}
			if artist != "" {
				d.Artists = []string{artist}
			}
			return &d, rows.Err()
		}
		return nil, rows.Err()
	}
	return nil, sql.ErrNoRows
}

func (r *Repository) defaultImageExpr(cardIDExpr, projectionAlias string) string {
	projectionExprs := []string{}
	if projectionAlias != "" && r.features.HasCardSearchProjection {
		if r.features.HasProjectionImageNormal {
			projectionExprs = append(projectionExprs, projectionAlias+".image_normal")
		}
		if r.features.HasProjectionImageLarge {
			projectionExprs = append(projectionExprs, projectionAlias+".image_large")
		}
		if r.features.HasProjectionImageSmall {
			projectionExprs = append(projectionExprs, projectionAlias+".image_small")
		}
	}
	if len(projectionExprs) > 0 {
		return "COALESCE(" + strings.Join(append(projectionExprs, r.printingImageExpr(cardIDExpr), "''"), ", ") + ")"
	}
	return "COALESCE(" + r.printingImageExpr(cardIDExpr) + ", '')"
}

func (r *Repository) displayNameExpr() string {
	base := "COALESCE(" + projectionExpr(r.features.HasCardSearchProjection, "csp.core_name") + ", core.name, cards.id)"
	if !r.features.HasPitchSiblingsJSON {
		return base
	}
	pitchLetter := pitchLetterExpr(projectionExpr(r.features.HasCardSearchProjection, "csp.pitch_value"), "core.pitch_value")
	return fmt.Sprintf(`CASE
    WHEN cards.pitch_siblings_json IS NOT NULL
      AND cards.pitch_siblings_json != ''
      AND cards.pitch_siblings_json != 'null'
      AND cards.pitch_siblings_json != '[]'
      AND %s != ''
    THEN %s || ' (' || %s || ')'
    ELSE %s
  END`, pitchLetter, base, pitchLetter, base)
}

func pitchLetterExpr(values ...string) string {
	usable := []string{}
	for _, value := range values {
		if strings.TrimSpace(value) != "" && strings.ToUpper(strings.TrimSpace(value)) != "NULL" {
			usable = append(usable, value)
		}
	}
	valueExpr := "NULL"
	if len(usable) == 1 {
		valueExpr = usable[0]
	} else if len(usable) > 1 {
		valueExpr = "COALESCE(" + strings.Join(usable, ", ") + ")"
	}
	return "CASE CAST(" + valueExpr + " AS INTEGER) WHEN 1 THEN 'R' WHEN 2 THEN 'Y' WHEN 3 THEN 'B' ELSE '' END"
}

func (r *Repository) cropImageExpr(cardIDExpr, projectionAlias string) string {
	projectionExprs := []string{}
	if projectionAlias != "" && r.features.HasCardSearchProjection {
		if r.features.HasProjectionImageCropMedium {
			projectionExprs = append(projectionExprs, projectionAlias+".image_crop_medium")
		}
		if r.features.HasProjectionImageCrop {
			projectionExprs = append(projectionExprs, projectionAlias+".image_crop")
		}
		if r.features.HasProjectionImageCropSmall {
			projectionExprs = append(projectionExprs, projectionAlias+".image_crop_small")
		}
		if r.features.HasProjectionImageCropXlarge {
			projectionExprs = append(projectionExprs, projectionAlias+".image_crop_xlarge")
		}
	}
	if len(projectionExprs) > 0 {
		return "COALESCE(" + strings.Join(append(projectionExprs, r.printingCropExpr(cardIDExpr), r.defaultImageExpr(cardIDExpr, projectionAlias), "''"), ", ") + ")"
	}
	return "COALESCE(" + r.printingCropExpr(cardIDExpr) + ", " + r.defaultImageExpr(cardIDExpr, projectionAlias) + ", '')"
}

func (r *Repository) printingCropExpr(cardIDExpr string) string {
	if !r.features.HasPrintings || !r.features.HasPrintingFaces {
		return "NULL"
	}
	cropExpr := r.printingFaceCropExpr()
	if cropExpr == "NULL" {
		return "NULL"
	}
	return fmt.Sprintf(`COALESCE((
    SELECT %s
    FROM printings p
    JOIN printing_faces pf ON pf.printing_id = p.id
    WHERE p.card_id = %s
      AND %s IS NOT NULL
    ORDER BY COALESCE(p.set_code, '') ASC, p.id ASC, (pf.face_id LIKE '%%_BACK') ASC, pf.face_id ASC
    LIMIT 1
  ), NULL)`, cropExpr, cardIDExpr, cropExpr)
}

func (r *Repository) printingFaceCropExpr() string {
	exprs := []string{}
	if r.features.HasPrintingImageCropMedium {
		exprs = append(exprs, "pf.image_crop_medium")
	}
	if r.features.HasPrintingImageCrop {
		exprs = append(exprs, "pf.image_crop")
	}
	if r.features.HasPrintingImageCropSmall {
		exprs = append(exprs, "pf.image_crop_small")
	}
	if r.features.HasPrintingImageCropXlarge {
		exprs = append(exprs, "pf.image_crop_xlarge")
	}
	if len(exprs) == 0 {
		return "NULL"
	}
	return "COALESCE(" + strings.Join(exprs, ", ") + ")"
}

func (r *Repository) printingRulesTextExpr() string {
	if r.features.HasPrintingRulesText {
		return "pf.rules_text"
	}
	return "NULL"
}

func (r *Repository) printingImageExpr(cardIDExpr string) string {
	if !r.features.HasPrintings || !r.features.HasPrintingFaces {
		return "NULL"
	}
	return fmt.Sprintf(`COALESCE((
    SELECT COALESCE(pf.image_normal, pf.image_large, pf.image_small)
    FROM printings p
    JOIN printing_faces pf ON pf.printing_id = p.id
    WHERE p.card_id = %s
      AND (pf.image_normal IS NOT NULL OR pf.image_large IS NOT NULL OR pf.image_small IS NOT NULL)
    ORDER BY COALESCE(p.set_code, '') ASC, p.id ASC, (pf.face_id LIKE '%%_BACK') ASC, pf.face_id ASC
    LIMIT 1
  ), NULL)`, cardIDExpr)
}

func (r *Repository) cardJSONIDs(ctx context.Context, cardID, column string) ([]string, error) {
	if column != "references_json" && column != "referenced_by_json" && column != "pitch_siblings_json" {
		return nil, fmt.Errorf("unsupported related-card column %q", column)
	}
	var raw string
	if err := r.db.QueryRowContext(ctx, `SELECT COALESCE(`+column+`, '[]') FROM cards WHERE id = ?`, cardID).Scan(&raw); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return parseStringArray(raw), nil
}

func (r *Repository) relationIDs(ctx context.Context, sqlText string, args ...any) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		id = strings.TrimSpace(id)
		if id != "" {
			out = append(out, id)
		}
	}
	return out, rows.Err()
}

func (r *Repository) relatedCards(ctx context.Context, ids []string) ([]RelatedCard, error) {
	ids = uniqueStrings(ids, 24)
	if !r.features.HasCardCores || len(ids) == 0 {
		return nil, nil
	}
	projectionJoin := ""
	if r.features.HasCardSearchProjection {
		projectionJoin = "LEFT JOIN card_search_projection csp ON csp.card_id = cards.id"
	}
	values := make([]string, 0, len(ids))
	args := make([]any, 0, len(ids)*2)
	for i, id := range ids {
		values = append(values, "(?, ?)")
		args = append(args, id, i)
	}
	query := fmt.Sprintf(`
WITH default_core AS (
  SELECT card_id, MIN(core_index) AS core_index
  FROM card_cores
  GROUP BY card_id
),
related(id, ord) AS (
  VALUES %s
)
SELECT
  cards.id,
  %s AS name,
  COALESCE(%s, core.typebox, '') AS type_line,
  %s AS art_url
FROM related
JOIN cards ON cards.id = related.id
JOIN default_core dc ON dc.card_id = cards.id
LEFT JOIN card_cores core ON core.card_id = dc.card_id AND core.core_index = dc.core_index
%s
GROUP BY cards.id
ORDER BY related.ord ASC, name COLLATE NOCASE ASC
LIMIT 12`,
		strings.Join(values, ", "),
		r.displayNameExpr(),
		projectionExpr(r.features.HasCardSearchProjection, "csp.typebox"),
		r.cropImageExpr("cards.id", "csp"),
		projectionJoin,
	)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []RelatedCard{}
	for rows.Next() {
		var item RelatedCard
		if err := rows.Scan(&item.ID, &item.Name, &item.TypeLine, &item.ArtURL); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func projectionExpr(ok bool, expr string) string {
	if ok {
		return expr
	}
	return "NULL"
}

func uniqueStrings(values []string, limit int) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func cardLegalityExpr(ok bool) string {
	if ok {
		return "COALESCE(cards.card_legality_json, '')"
	}
	return "''"
}

func scanSearchRows(rows *sql.Rows) ([]SearchResult, error) {
	var out []SearchResult
	for rows.Next() {
		var item SearchResult
		if err := rows.Scan(&item.ID, &item.Name, &item.TypeLine, &item.Text, &item.Cost, &item.Pitch, &item.Power, &item.Defense, &item.Health, &item.ImageURL, &item.ArtURL); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func mergePrinting(detail *CardDetail, printing *CardDetail) {
	if printing.PrintingID != "" {
		detail.PrintingID = printing.PrintingID
	}
	if printing.Printing != "" {
		detail.Printing = printing.Printing
	}
	if printing.SetCode != "" {
		detail.SetCode = printing.SetCode
	}
	if printing.SetName != "" {
		detail.SetName = printing.SetName
	}
	if printing.Rarity != "" {
		detail.Rarity = printing.Rarity
	}
	if detail.ImageURL == "" && printing.ImageURL != "" {
		detail.ImageURL = printing.ImageURL
	}
	if detail.ArtURL == "" && printing.ArtURL != "" {
		detail.ArtURL = printing.ArtURL
	}
	if len(printing.Artists) > 0 {
		detail.Artists = printing.Artists
	}
	if printing.FlavorText != "" {
		detail.FlavorText = printing.FlavorText
	}
	if printing.PrintedText != "" {
		detail.PrintedText = printing.PrintedText
	}
}

func ftsMatch(search string) string {
	fields := strings.Fields(search)
	if len(fields) == 0 {
		return ""
	}
	terms := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.Map(func(r rune) rune {
			if r == '"' || r == '\'' || r == ':' || r == '*' {
				return -1
			}
			return r
		}, field)
		field = strings.TrimSpace(field)
		if field != "" {
			terms = append(terms, field+"*")
		}
	}
	return strings.Join(terms, " ")
}

func compactJSON(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return s
	}
	data, err := json.Marshal(v)
	if err != nil {
		return s
	}
	return string(data)
}

func parseStringArray(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var values []any
	if err := json.Unmarshal([]byte(raw), &values); err == nil {
		out := make([]string, 0, len(values))
		for _, value := range values {
			text := strings.TrimSpace(fmt.Sprint(value))
			if text != "" {
				out = append(out, text)
			}
		}
		return out
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '|' || r == ';' || r == '/'
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
