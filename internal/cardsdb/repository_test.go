package cardsdb

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestRepositorySearchAndGetCardWithSimpleSchema(t *testing.T) {
	t.Parallel()
	dbPath := createSimpleCardsDB(t)
	repo, err := OpenRepository(dbPath)
	if err != nil {
		t.Fatalf("OpenRepository() error = %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })

	results, err := repo.Search(context.Background(), SearchParams{Query: "alpha", Limit: 10})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Search() len = %d, want 1", len(results))
	}
	if results[0].ID != "card-alpha" || results[0].Name != "Alpha Strike" {
		t.Fatalf("Search()[0] = %#v", results[0])
	}

	detail, err := repo.GetCard(context.Background(), "card-alpha")
	if err != nil {
		t.Fatalf("GetCard() error = %v", err)
	}
	if detail.Name != "Alpha Strike" || detail.Power != "3" || detail.ImageURL != "https://example.test/alpha.png" {
		t.Fatalf("detail = %#v", detail)
	}
}

func TestRepositorySearchReturnsEmptyForNoMatch(t *testing.T) {
	t.Parallel()
	dbPath := createSimpleCardsDB(t)
	repo, err := OpenRepository(dbPath)
	if err != nil {
		t.Fatalf("OpenRepository() error = %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })

	results, err := repo.Search(context.Background(), SearchParams{Query: "does-not-exist", Limit: 10})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("Search() len = %d, want 0", len(results))
	}
}

func TestRepositorySearchWithCardLevelProjectionSchema(t *testing.T) {
	t.Parallel()
	dbPath := createProjectedCardsDB(t)
	repo, err := OpenRepository(dbPath)
	if err != nil {
		t.Fatalf("OpenRepository() error = %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })

	results, err := repo.Search(context.Background(), SearchParams{Query: "projected", Limit: 10})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Search() len = %d, want 1", len(results))
	}
	if results[0].Name != "Projected Alpha" || results[0].ImageURL != "https://example.test/projection-normal.png" || results[0].ArtURL != "https://example.test/projection-crop-medium.png" {
		t.Fatalf("Search()[0] = %#v", results[0])
	}

	detail, err := repo.GetCard(context.Background(), "card-projected")
	if err != nil {
		t.Fatalf("GetCard() error = %v", err)
	}
	if detail.Name != "Projected Alpha" || detail.ImageURL != "https://example.test/projection-normal.png" || detail.ArtURL != "https://example.test/projection-crop-medium.png" {
		t.Fatalf("detail = %#v", detail)
	}
	if detail.PrintedText != "Printed text" {
		t.Fatalf("detail.PrintedText = %q, want Printed text", detail.PrintedText)
	}
	if len(detail.Classes) != 1 || detail.Classes[0] != "Warrior" || len(detail.BaseTypes) != 1 || detail.BaseTypes[0] != "Action" {
		t.Fatalf("detail type groups = classes %#v baseTypes %#v", detail.Classes, detail.BaseTypes)
	}

	printings, err := repo.ListPrintings(context.Background(), "card-projected")
	if err != nil {
		t.Fatalf("ListPrintings() error = %v", err)
	}
	if len(printings) != 2 {
		t.Fatalf("ListPrintings() len = %d, want 2: %#v", len(printings), printings)
	}
	if printings[0].ID != "printing-projected" || printings[0].Artists[0] != "Artist Name" || printings[0].PrintedText != "Printed text" {
		t.Fatalf("ListPrintings()[0] = %#v", printings[0])
	}

	related, err := repo.ListRelatedCards(context.Background(), "card-projected")
	if err != nil {
		t.Fatalf("ListRelatedCards() error = %v", err)
	}
	if len(related.References) != 1 || related.References[0].Name != "Referenced Token" {
		t.Fatalf("related = %#v", related)
	}

	products, err := repo.ListCardProducts(context.Background(), "card-projected")
	if err != nil {
		t.Fatalf("ListCardProducts() error = %v", err)
	}
	if len(products) != 1 || products[0].Name != "Projected Product" || products[0].TCGPlayerProductID != "123" {
		t.Fatalf("products = %#v", products)
	}
}

func TestRepositorySearchCardsUsesFrontendQueryFilters(t *testing.T) {
	t.Parallel()
	dbPath := createProjectedCardsDB(t)
	repo, err := OpenRepository(dbPath)
	if err != nil {
		t.Fatalf("OpenRepository() error = %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })

	params := ParseCardSearchQuery(`class:warrior type:action subtype:attack talent:lightning keyword:"go again" pitch>=2 set:tst r:r artist:"Artist Name" legal:blitz projected`)
	params.PageSize = 1
	resp, err := repo.SearchCards(context.Background(), params)
	if err != nil {
		t.Fatalf("SearchCards() error = %v", err)
	}
	if len(resp.Summaries) != 1 {
		t.Fatalf("SearchCards() len = %d, want 1: %#v", len(resp.Summaries), resp)
	}
	if resp.Summaries[0].Identifier != "card-projected" || resp.Summaries[0].Name != "Projected Alpha" {
		t.Fatalf("SearchCards()[0] = %#v", resp.Summaries[0])
	}
}

func TestRepositorySearchCardsSetAliasWithSetsTable(t *testing.T) {
	t.Parallel()
	dbPath := createProjectedCardsDB(t)
	repo, err := OpenRepository(dbPath)
	if err != nil {
		t.Fatalf("OpenRepository() error = %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })

	resp, err := repo.SearchCards(context.Background(), ParseCardSearchQuery(`s:tst r:r`))
	if err != nil {
		t.Fatalf("SearchCards() error = %v", err)
	}
	if len(resp.Summaries) != 1 || resp.Summaries[0].Identifier != "card-projected" {
		t.Fatalf("SearchCards() = %#v, want card-projected", resp)
	}
}

func TestRepositorySearchCardsUsesCoreSelectorTablesWhenProjectionJSONIsMissing(t *testing.T) {
	t.Parallel()
	dbPath := createNormalizedSelectorCardsDB(t)
	repo, err := OpenRepository(dbPath)
	if err != nil {
		t.Fatalf("OpenRepository() error = %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })

	resp, err := repo.SearchCards(context.Background(), ParseCardSearchQuery(`c:Wizard s:WTR`))
	if err != nil {
		t.Fatalf("SearchCards() error = %v", err)
	}
	if len(resp.Summaries) != 1 {
		t.Fatalf("SearchCards() len = %d, want 1: %#v", len(resp.Summaries), resp)
	}
	if resp.Summaries[0].Identifier != "wizard-card" {
		t.Fatalf("SearchCards()[0] = %#v, want wizard-card", resp.Summaries[0])
	}
}

func TestRepositoryBatchAndProductSummaries(t *testing.T) {
	t.Parallel()
	dbPath := createProjectedCardsDB(t)
	repo, err := OpenRepository(dbPath)
	if err != nil {
		t.Fatalf("OpenRepository() error = %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })

	cards, err := repo.BatchGetCardSummaries(context.Background(), []string{"card-projected", "missing"})
	if err != nil {
		t.Fatalf("BatchGetCardSummaries() error = %v", err)
	}
	if _, ok := cards.Cards["card-projected"]; !ok || len(cards.NotFoundIDs) != 1 || cards.NotFoundIDs[0] != "missing" {
		t.Fatalf("cards batch = %#v", cards)
	}

	product, err := repo.GetProductSummary(context.Background(), "product-projected")
	if err != nil {
		t.Fatalf("GetProductSummary() error = %v", err)
	}
	if product.Summary == nil || product.Summary.Name != "Projected Product" || product.Summary.TCGPlayerProductID != "123" {
		t.Fatalf("product = %#v", product)
	}
}

func TestRepositoryDisplayNameAddsPitchColorForPitchSiblings(t *testing.T) {
	t.Parallel()
	dbPath := createPitchSiblingCardsDB(t)
	repo, err := OpenRepository(dbPath)
	if err != nil {
		t.Fatalf("OpenRepository() error = %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })

	results, err := repo.Search(context.Background(), SearchParams{Query: "Sink Below", Limit: 10})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Search() len = %d, want 1", len(results))
	}
	if results[0].Name != "Sink Below (R)" {
		t.Fatalf("Search()[0].Name = %q, want Sink Below (R)", results[0].Name)
	}
}

func createSimpleCardsDB(t *testing.T) string {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "cards.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`
CREATE TABLE cards (
  id TEXT PRIMARY KEY,
  name TEXT,
  types TEXT,
  functional_text TEXT,
  cost TEXT,
  pitch TEXT,
  power TEXT,
  defense TEXT,
  health TEXT,
  intelligence TEXT,
  arcane TEXT,
  default_image_url TEXT
);
INSERT INTO cards (
  id, name, types, functional_text, cost, pitch, power, defense, health, intelligence, arcane, default_image_url
) VALUES (
  'card-alpha', 'Alpha Strike', 'Attack Action', 'Deal damage.', '1', '2', '3', '2', '', '', '', 'https://example.test/alpha.png'
), (
  'card-beta', 'Beta Block', 'Defense Reaction', 'Prevent damage.', '0', '3', '', '3', '', '', '', ''
);`)
	if err != nil {
		t.Fatal(err)
	}
	return dbPath
}

func createProjectedCardsDB(t *testing.T) string {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "cards.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`
CREATE TABLE cards (
  id TEXT PRIMARY KEY,
  card_legality_json TEXT,
  references_json TEXT,
  referenced_by_json TEXT,
  pitch_siblings_json TEXT
);
CREATE TABLE card_cores (
  card_id TEXT NOT NULL,
  core_index INTEGER NOT NULL,
  name TEXT,
  pitch_text TEXT,
  pitch_value INTEGER,
  chi_text TEXT,
  color TEXT,
  cost TEXT,
  power TEXT,
  defense TEXT,
  intellect TEXT,
  life TEXT,
  textbox TEXT,
  typebox TEXT,
  traitbox TEXT,
  PRIMARY KEY(card_id, core_index)
);
CREATE TABLE card_search_projection (
  card_id TEXT PRIMARY KEY,
  core_name TEXT,
  pitch_value INTEGER,
  cost_num INTEGER,
  power_num INTEGER,
  defense_num INTEGER,
  intellect_num INTEGER,
  life_num INTEGER,
  textbox TEXT,
  typebox TEXT,
  image_small TEXT,
  image_normal TEXT,
  image_large TEXT,
  image_crop TEXT,
  image_crop_xlarge TEXT,
  image_crop_medium TEXT,
  image_crop_small TEXT,
  classes_json TEXT,
  talents_json TEXT,
  types_json TEXT,
  subtypes_json TEXT,
  keywords_json TEXT
);
CREATE TABLE printings (
  id TEXT PRIMARY KEY,
  card_id TEXT,
  set_code TEXT,
  rarity TEXT,
  language TEXT,
  product_name TEXT
);
CREATE TABLE sets (
  code TEXT PRIMARY KEY,
  name TEXT,
  release_date TEXT
);
CREATE TABLE card_references (
  card_id TEXT NOT NULL,
  ref_card_id TEXT NOT NULL
);
CREATE TABLE product_projection (
  product_id TEXT PRIMARY KEY,
  type TEXT,
  card_id TEXT,
  printing_id TEXT,
  set_code TEXT,
  name TEXT,
  release_date TEXT,
  printed_date TEXT,
  printed_language TEXT,
  tcgplayer_url TEXT,
  tcgplayer_product_id TEXT,
  resolved_set_name TEXT,
  product_group_name TEXT,
  product_group_release_date TEXT,
  quantity INTEGER
);
CREATE TABLE printing_faces (
  printing_id TEXT,
  face_id TEXT,
  name TEXT,
  image_small TEXT,
  image_normal TEXT,
  image_large TEXT,
  artist TEXT,
  flavor_text TEXT,
  rules_text TEXT,
  language TEXT
);
INSERT INTO cards(id, card_legality_json, references_json, referenced_by_json, pitch_siblings_json)
VALUES ('card-projected', '{"blitz":{"isLegal":true}}', '["referenced-token"]', '[]', '[]');
INSERT INTO cards(id, card_legality_json, references_json, referenced_by_json, pitch_siblings_json)
VALUES ('referenced-token', '{}', '[]', '["card-projected"]', '[]');
INSERT INTO card_cores(card_id, core_index, name, pitch_text, pitch_value, cost, power, defense, life, textbox, typebox)
VALUES ('card-projected', 0, 'Fallback Alpha', '2', 2, '1', '3', '2', '', 'Fallback text', 'Action');
INSERT INTO card_cores(card_id, core_index, name, pitch_text, pitch_value, cost, power, defense, life, textbox, typebox)
VALUES ('referenced-token', 0, 'Referenced Token', '', NULL, '', '', '', '', 'Token text', 'Generic Token');
INSERT INTO card_search_projection(card_id, core_name, pitch_value, cost_num, power_num, defense_num, textbox, typebox, image_small, image_normal, image_large, image_crop, image_crop_xlarge, image_crop_medium, image_crop_small, classes_json, talents_json, types_json, subtypes_json, keywords_json)
VALUES ('card-projected', 'Projected Alpha', 2, 1, 3, 2, 'Projected text', 'Attack Action', 'https://example.test/projection-small.png', 'https://example.test/projection-normal.png', 'https://example.test/projection-large.png', 'https://example.test/projection-crop.png', 'https://example.test/projection-crop-xlarge.png', 'https://example.test/projection-crop-medium.png', 'https://example.test/projection-crop-small.png', '["Warrior"]', '["Lightning"]', '["Action"]', '["Attack"]', '["go again"]');
INSERT INTO card_search_projection(card_id, core_name, textbox, typebox, classes_json, talents_json, types_json, subtypes_json, keywords_json)
VALUES ('referenced-token', 'Referenced Token', 'Token text', 'Generic Token', '[]', '[]', '["Token"]', '[]', '[]');
INSERT INTO card_references(card_id, ref_card_id) VALUES ('card-projected', 'referenced-token');
INSERT INTO printings(id, card_id, set_code, rarity, language, product_name)
VALUES ('printing-projected', 'card-projected', 'TST', 'RARE', 'en', 'Projected Product');
INSERT INTO printings(id, card_id, set_code, rarity, language, product_name)
VALUES ('printing-projected-2', 'card-projected', 'ZZZ', 'COMMON', 'en', 'Second Product');
INSERT INTO sets(code, name, release_date) VALUES ('TST', 'Test Set', '2024-01-01');
INSERT INTO sets(code, name, release_date) VALUES ('ZZZ', 'Second Set', '2024-02-01');
INSERT INTO printing_faces(printing_id, face_id, name, image_small, image_normal, image_large, artist, flavor_text, rules_text, language)
VALUES ('printing-projected', 'FRONT', 'Projected Alpha', 'https://example.test/printing-small.png', 'https://example.test/printing-normal.png', 'https://example.test/printing-large.png', 'Artist Name', 'Flavor', 'Printed text', 'en');
INSERT INTO printing_faces(printing_id, face_id, name, image_small, image_normal, image_large, artist, flavor_text, rules_text, language)
VALUES ('printing-projected-2', 'FRONT', 'Projected Alpha', 'https://example.test/printing2-small.png', 'https://example.test/printing2-normal.png', 'https://example.test/printing2-large.png', 'Second Artist', 'Second Flavor', 'Second printed text', 'en');
INSERT INTO product_projection(product_id, type, card_id, printing_id, set_code, name, release_date, printed_language, tcgplayer_product_id, resolved_set_name, quantity)
VALUES ('product-projected', 'single', 'card-projected', 'printing-projected', 'TST', 'Projected Product', '2024-01-01', 'en', '123', 'Test Set', 1);
`)
	if err != nil {
		t.Fatal(err)
	}
	return dbPath
}

func createNormalizedSelectorCardsDB(t *testing.T) string {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "cards.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`
CREATE TABLE cards (
  id TEXT PRIMARY KEY
);
CREATE TABLE card_cores (
  card_id TEXT NOT NULL,
  core_index INTEGER NOT NULL,
  name TEXT,
  pitch_text TEXT,
  pitch_value INTEGER,
  chi_text TEXT,
  color TEXT,
  cost TEXT,
  power TEXT,
  defense TEXT,
  intellect TEXT,
  life TEXT,
  textbox TEXT,
  typebox TEXT,
  traitbox TEXT,
  PRIMARY KEY(card_id, core_index)
);
CREATE TABLE card_search_projection (
  card_id TEXT PRIMARY KEY,
  core_name TEXT,
  pitch_value INTEGER,
  cost_num INTEGER,
  power_num INTEGER,
  defense_num INTEGER,
  intellect_num INTEGER,
  life_num INTEGER,
  textbox TEXT,
  typebox TEXT
);
CREATE TABLE card_core_classes (
  card_id TEXT NOT NULL,
  core_index INTEGER NOT NULL,
  class TEXT NOT NULL,
  class_norm TEXT NOT NULL,
  PRIMARY KEY(card_id, core_index, class)
);
CREATE TABLE printings (
  id TEXT PRIMARY KEY,
  card_id TEXT,
  set_code TEXT,
  rarity TEXT,
  language TEXT,
  product_name TEXT
);
CREATE TABLE sets (
  code TEXT PRIMARY KEY,
  name TEXT,
  release_date TEXT
);
INSERT INTO cards(id) VALUES ('wizard-card'), ('warrior-card');
INSERT INTO card_cores(card_id, core_index, name, pitch_text, pitch_value, cost, power, defense, life, textbox, typebox)
VALUES ('wizard-card', 0, 'Arcane Example', '3', 3, '1', '', '2', '', 'Wizard text', 'Wizard Action');
INSERT INTO card_cores(card_id, core_index, name, pitch_text, pitch_value, cost, power, defense, life, textbox, typebox)
VALUES ('warrior-card', 0, 'Blade Example', '2', 2, '1', '3', '', '', 'Warrior text', 'Warrior Action');
INSERT INTO card_search_projection(card_id, core_name, pitch_value, cost_num, power_num, defense_num, textbox, typebox)
VALUES ('wizard-card', 'Arcane Example', 3, 1, NULL, 2, 'Wizard text', 'Wizard Action');
INSERT INTO card_search_projection(card_id, core_name, pitch_value, cost_num, power_num, defense_num, textbox, typebox)
VALUES ('warrior-card', 'Blade Example', 2, 1, 3, NULL, 'Warrior text', 'Warrior Action');
INSERT INTO card_core_classes(card_id, core_index, class, class_norm)
VALUES ('wizard-card', 0, 'Wizard', 'wizard');
INSERT INTO card_core_classes(card_id, core_index, class, class_norm)
VALUES ('warrior-card', 0, 'Warrior', 'warrior');
INSERT INTO printings(id, card_id, set_code, rarity, language, product_name)
VALUES ('wizard-printing', 'wizard-card', 'WTR', 'RARE', 'en', 'Arcane Example');
INSERT INTO printings(id, card_id, set_code, rarity, language, product_name)
VALUES ('warrior-printing', 'warrior-card', 'WTR', 'RARE', 'en', 'Blade Example');
INSERT INTO sets(code, name, release_date) VALUES ('WTR', 'Welcome to Rathe', '2019-10-11');
`)
	if err != nil {
		t.Fatal(err)
	}
	return dbPath
}

func createPitchSiblingCardsDB(t *testing.T) string {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "cards.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`
CREATE TABLE cards (
  id TEXT PRIMARY KEY,
  pitch_siblings_json TEXT
);
CREATE TABLE card_cores (
  card_id TEXT NOT NULL,
  core_index INTEGER NOT NULL,
  name TEXT,
  pitch_text TEXT,
  pitch_value INTEGER,
  chi_text TEXT,
  color TEXT,
  cost TEXT,
  power TEXT,
  defense TEXT,
  intellect TEXT,
  life TEXT,
  textbox TEXT,
  typebox TEXT,
  traitbox TEXT,
  PRIMARY KEY(card_id, core_index)
);
CREATE TABLE card_search_projection (
  card_id TEXT PRIMARY KEY,
  core_name TEXT,
  pitch_value INTEGER,
  cost_num INTEGER,
  power_num INTEGER,
  defense_num INTEGER,
  intellect_num INTEGER,
  life_num INTEGER,
  textbox TEXT,
  typebox TEXT
);
INSERT INTO cards(id, pitch_siblings_json) VALUES ('sink-below-r', '["sink-below-y","sink-below-b"]');
INSERT INTO card_cores(card_id, core_index, name, pitch_text, pitch_value, cost, power, defense, life, textbox, typebox)
VALUES ('sink-below-r', 0, 'Sink Below', '1', 1, '0', '', '', '', 'Defense text', 'Defense Reaction');
INSERT INTO card_search_projection(card_id, core_name, pitch_value, cost_num, textbox, typebox)
VALUES ('sink-below-r', 'Sink Below', 1, 0, 'Defense text', 'Defense Reaction');
`)
	if err != nil {
		t.Fatal(err)
	}
	return dbPath
}
