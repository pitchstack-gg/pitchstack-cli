package cards

import (
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/pitchstack-gg/pitchstack-cli/internal/cardsdb"
	clientv1 "github.com/pitchstack-gg/pitchstack-go/client/v1"
)

func TestModelResizeSetsResponsiveDimensions(t *testing.T) {
	t.Parallel()
	model := New(Options{})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	next := updated.(Model)

	if next.results.Width() <= 0 || next.results.Height() <= 0 {
		t.Fatalf("results size = %dx%d", next.results.Width(), next.results.Height())
	}
	if next.art.Width() <= 0 || next.art.Height() <= 0 {
		t.Fatalf("art size = %dx%d", next.art.Width(), next.art.Height())
	}
	if next.detail.Width() <= 0 || next.detail.Height() <= 0 {
		t.Fatalf("detail size = %dx%d", next.detail.Width(), next.detail.Height())
	}
}

func TestModelResizeBalancesArtAndDetailSpace(t *testing.T) {
	t.Parallel()
	model := New(Options{})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 180, Height: 45})
	next := updated.(Model)

	if next.art.Height() < 18 {
		t.Fatalf("art height = %d, want at least 18 content rows", next.art.Height())
	}
	if next.detail.Height() < 12 {
		t.Fatalf("detail height = %d, want at least 12 content rows", next.detail.Height())
	}
}

func TestModelWideLayoutPlacesPrintingsBesideArt(t *testing.T) {
	t.Parallel()
	model := New(Options{})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 180, Height: 45})
	next := updated.(Model)

	if next.prints.Height() != next.art.Height() {
		t.Fatalf("printings height = %d, want art height %d", next.prints.Height(), next.art.Height())
	}
	if next.detail.Width() <= next.art.Width() {
		t.Fatalf("detail width = %d, want wider than art width %d", next.detail.Width(), next.art.Width())
	}
}

func TestPrintingsSelectionPagesViewport(t *testing.T) {
	t.Parallel()
	model := New(Options{})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 180, Height: 45})
	next := updated.(Model)
	next.selectedID = "card"
	next.prints.SetHeight(8)
	for i := 0; i < 8; i++ {
		next.printings = append(next.printings, cardsdb.CardPrinting{
			ID:       "printing",
			SetCode:  "SET",
			Name:     "Printing",
			Language: "en",
		})
	}
	next.printingLangs = printingLanguages(next.printings)
	next.activeLanguage = "en"

	_ = next.selectPrinting(0)
	if next.prints.YOffset() != 0 {
		t.Fatalf("initial printings offset = %d, want 0", next.prints.YOffset())
	}
	_ = next.selectPrinting(2)
	if next.prints.YOffset() != 8 {
		t.Fatalf("paged printings offset = %d, want 8", next.prints.YOffset())
	}
}

func TestPrintingsLanguageFilterGatesSelection(t *testing.T) {
	t.Parallel()
	model := New(Options{})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 180, Height: 45})
	next := updated.(Model)
	next.selectedID = "card"
	next.printings = []cardsdb.CardPrinting{
		{ID: "en-1", SetCode: "EN1", Name: "English One", Language: "en"},
		{ID: "fr-1", SetCode: "FR1", Name: "French One", Language: "fr"},
		{ID: "en-2", SetCode: "EN2", Name: "English Two", Language: "en"},
	}
	next.printingLangs = printingLanguages(next.printings)
	next.activeLanguage = defaultPrintingLanguage(next.printingLangs)

	_ = next.selectPrinting(1)
	selected, ok := next.selectedPrinting()
	if !ok || selected.ID != "en-2" {
		t.Fatalf("selected printing = %#v, ok %v; want en-2", selected, ok)
	}
	output := stripANSILite(next.renderPrintings())
	if !strings.Contains(output, "[EN]") || strings.Contains(output, "French One") {
		t.Fatalf("renderPrintings() did not gate by EN:\n%s", output)
	}
	if strings.Contains(output, "switch printing") || strings.Contains(output, "l language") {
		t.Fatalf("renderPrintings() should not include local helper text:\n%s", output)
	}

	_ = next.selectPrintingLanguage(1)
	selected, ok = next.selectedPrinting()
	if !ok || selected.ID != "fr-1" {
		t.Fatalf("selected printing after language switch = %#v, ok %v; want fr-1", selected, ok)
	}
	output = stripANSILite(next.renderPrintings())
	if !strings.Contains(output, "[FR]") || strings.Contains(output, "English One") {
		t.Fatalf("renderPrintings() did not gate by FR:\n%s", output)
	}
}

func TestRenderPrintingsUsesPrintingIDAndSetName(t *testing.T) {
	t.Parallel()
	model := New(Options{})
	model.printings = []cardsdb.CardPrinting{{
		ID:       "ANQ004-RF",
		SetCode:  "ANQ",
		SetName:  "Compendium of Rathe - Antiquity Pack",
		Rarity:   "gold",
		Name:     "Balance of Justice",
		Language: "en",
		Artists:  []string{"Sariya Asavmetha"},
	}}
	model.printingLangs = []string{"en"}
	model.activeLanguage = "en"

	output := stripANSILite(model.renderPrintings())
	if !strings.Contains(output, "ANQ004-RF - Gold") || !strings.Contains(output, "Compendium of Rathe - Antiquity Pack") {
		t.Fatalf("renderPrintings() missing printing id/rarity/set name:\n%s", output)
	}
	if strings.Contains(output, "Balance of Justice") || strings.Contains(output, "Sariya Asavmetha") {
		t.Fatalf("renderPrintings() should not include card name or artist rows:\n%s", output)
	}
}

func TestModelRenderCropsPanelsToTerminalHeight(t *testing.T) {
	t.Parallel()
	model := New(Options{})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 180, Height: 45})
	next := updated.(Model)
	next.art.SetContent(strings.Repeat("art\n", 80))
	next.detail.SetContent(strings.Repeat("detail\n", 80))

	lines := strings.Split(next.render(), "\n")
	if len(lines) > 45 {
		t.Fatalf("rendered height = %d lines, want at most 45", len(lines))
	}
}

func TestRenderPriceHistoryChartShowsAxisLabels(t *testing.T) {
	t.Parallel()
	jan1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	jan15 := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	feb1 := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	output := stripANSILite(renderPriceHistoryChart([]clientv1.PriceEntry{
		{Price: 10, RecordAt: &jan1},
		{Price: 15, RecordAt: &jan15},
		{Price: 20, RecordAt: &feb1},
	}, 72, 8))

	if !strings.Contains(output, "$20.00") || !strings.Contains(output, "$10.00") {
		t.Fatalf("chart missing y-axis labels:\n%s", output)
	}
	if !strings.Contains(output, "Jan 1") || !strings.Contains(output, "Feb 1") {
		t.Fatalf("chart missing x-axis labels:\n%s", output)
	}
}

func TestRenderPricesTabOmitsHistoryCountAndHandlesNotFound(t *testing.T) {
	t.Parallel()
	may13 := time.Date(2026, 5, 13, 0, 0, 0, 0, time.UTC)
	output := stripANSILite(renderPricesTab(priceRange3M, "product", []clientv1.PriceEntry{{
		ProductID: "product",
		Price:     12,
		RecordAt:  &may13,
	}}, []clientv1.PriceEntry{{ProductID: "product", Price: 11, RecordAt: &may13}}, false, "", "", 96, 20))

	if strings.Contains(output, "price history points") {
		t.Fatalf("prices tab should not show history point count:\n%s", output)
	}

	output = stripANSILite(renderPricesTab(priceRange3M, "product", nil, nil, false, "", noPriceRecordedMessage, 96, 20))
	if !strings.Contains(output, noPriceRecordedMessage) || strings.Contains(output, "Pricing API") {
		t.Fatalf("404 prices tab should show no-price placeholder without error:\n%s", output)
	}
}

func TestIsPriceNotFoundError(t *testing.T) {
	t.Parallel()
	err := &clientv1.APIError{Metadata: clientv1.ResponseMetadata{StatusCode: 404}}
	if !isPriceNotFoundError(err) {
		t.Fatalf("isPriceNotFoundError() = false, want true")
	}
}

func TestCardDelegateRendersTypeAndStatsOnSeparateLines(t *testing.T) {
	t.Parallel()
	item := resultItem{SearchResult: cardsdb.SearchResult{
		Name:     "Act of Glory (R)",
		TypeLine: "Guardian Instant - Aura",
		Cost:     "4",
		Pitch:    "1",
		Defense:  "3",
	}}
	model := list.New([]list.Item{item}, cardDelegate{}, 32, 4)
	var rendered strings.Builder

	cardDelegate{}.Render(&rendered, model, 0, item)
	output := rendered.String()
	lines := strings.Split(output, "\n")
	if len(lines) != 3 {
		t.Fatalf("rendered %d lines, want 3: %q", len(lines), output)
	}
	if !strings.Contains(lines[1], "Guardian Instant - Aura") {
		t.Fatalf("type line missing from second row: %q", output)
	}
	if !strings.Contains(lines[2], "Cost 4  Pitch 1  Defense 3") {
		t.Fatalf("stats missing from third row: %q", output)
	}
	if strings.Contains(output, "…") {
		t.Fatalf("row output should not contain truncation ellipsis: %q", output)
	}
}

func TestRenderDetailUsesFrontendStyleSections(t *testing.T) {
	t.Parallel()
	output := stripANSILite(renderDetail(&cardsdb.CardDetail{
		ID:         "aether-spindle-blue",
		Name:       "Aether Spindle (B)",
		TypeLine:   "Wizard Action",
		Text:       "Deal 2 arcane damage.{br}**Opt X**.",
		Cost:       "2",
		Pitch:      "3",
		Defense:    "3",
		Classes:    []string{"Wizard"},
		BaseTypes:  []string{"Action"},
		Keywords:   []string{"opt"},
		Printing:   "Aether Spindle",
		SetName:    "History Pack 1 - Black Label",
		Rarity:     "rare",
		Artists:    []string{"Anastasiya Grintsova"},
		FlavorText: "The power of the Dracai.",
		Legality:   `{"Blitz":{"legality":"legal"},"Classic Constructed":{"legality":"legal"},"Commoner":{"legality":"not legal"}}`,
	}, rulesTextTrue, 96))

	for _, want := range []string{
		"Pitch 3",
		"Cost 2",
		"Defense 3",
		"Rules Text",
		"Deal 2 arcane damage.",
		"Opt X.",
		"Legality",
		"Classic Constructed",
		"Not Legal",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("renderDetail() missing %q in:\n%s", want, output)
		}
	}
	if strings.Contains(output, "legality") {
		t.Fatalf("renderDetail() should not show raw legality JSON:\n%s", output)
	}
	if strings.Contains(output, "Types\n") || strings.Contains(output, "Illustrated by:") {
		t.Fatalf("renderDetail() should not show type section or artist credit:\n%s", output)
	}
	if strings.Contains(output, "Printing\n") {
		t.Fatalf("renderDetail() should not show printing section:\n%s", output)
	}
}

func TestRenderDetailCanUsePrintedRulesText(t *testing.T) {
	t.Parallel()
	detail := &cardsdb.CardDetail{
		Name:        "Localized Card",
		Text:        "True card-object text.",
		PrintedText: "Printed face text.",
	}

	trueOutput := stripANSILite(renderDetail(detail, rulesTextTrue, 80))
	if !strings.Contains(trueOutput, "Rules Text") || !strings.Contains(trueOutput, "True card-object text.") {
		t.Fatalf("true rules output missing card text:\n%s", trueOutput)
	}
	if strings.Contains(trueOutput, "Printed face text.") {
		t.Fatalf("true rules output should not show printed text:\n%s", trueOutput)
	}

	printedOutput := stripANSILite(renderDetail(detail, rulesTextPrinted, 80))
	if !strings.Contains(printedOutput, "Printed Text") || !strings.Contains(printedOutput, "Printed face text.") {
		t.Fatalf("printed rules output missing printed text:\n%s", printedOutput)
	}
}

func TestModelToggleRulesTextMode(t *testing.T) {
	t.Parallel()
	model := New(Options{})
	model.focus = focusResults
	model.baseDetail = &cardsdb.CardDetail{
		Name:        "Toggle Card",
		Text:        "True text.",
		PrintedText: "Printed text.",
	}
	model.detail.SetWidth(80)
	model.detail.SetHeight(10)
	model.detail.SetContent(model.renderDetail())

	updated, _ := model.Update(tea.KeyPressMsg(tea.Key{Text: "t", Code: 't'}))
	next := updated.(Model)
	if next.rulesMode != rulesTextPrinted {
		t.Fatalf("rulesMode = %s, want %s", next.rulesMode, rulesTextPrinted)
	}
	if output := stripANSILite(next.detail.View()); !strings.Contains(output, "Printed text.") {
		t.Fatalf("detail view missing printed text:\n%s", output)
	}
}

func TestModelDetailTabsRenderRelatedAndPrices(t *testing.T) {
	t.Parallel()
	model := New(Options{})
	model.focus = focusResults
	model.detail.SetWidth(96)
	model.detail.SetHeight(20)
	model.baseDetail = &cardsdb.CardDetail{Name: "Main Card", Text: "Rules."}
	model.related = &cardsdb.RelatedCards{
		References: []cardsdb.RelatedCard{{ID: "gold", Name: "Gold", TypeLine: "Generic Token"}},
	}
	model.products = []cardsdb.PrintingProduct{{
		ID:                 "product",
		Name:               "Main Card Rainbow Foil",
		SetCode:            "MST",
		SetName:            "Part the Mistveil",
		ReleaseDate:        "2024-05-31",
		TCGPlayerProductID: "123",
		Quantity:           2,
	}}
	model.printings = []cardsdb.CardPrinting{{ID: "printing", Name: "Main Card", SetCode: "MST", Rarity: "majestic", Language: "en"}}
	model.printingLangs = []string{"en"}
	model.activeLanguage = "en"

	updated, _ := model.Update(tea.KeyPressMsg(tea.Key{Text: "2", Code: '2'}))
	next := updated.(Model)
	if next.detailTab != detailTabRelated {
		t.Fatalf("detailTab = %s, want related", next.detailTab)
	}
	if output := stripANSILite(next.detail.View()); !strings.Contains(output, "Referenced Cards") || !strings.Contains(output, "Gold") {
		t.Fatalf("related tab missing content:\n%s", output)
	}

	updated, _ = next.Update(tea.KeyPressMsg(tea.Key{Text: "3", Code: '3'}))
	next = updated.(Model)
	if next.detailTab != detailTabPrices {
		t.Fatalf("detailTab = %s, want prices", next.detailTab)
	}
	if output := stripANSILite(next.detail.View()); !strings.Contains(output, "Prices") || !strings.Contains(output, "[5 3M]") {
		t.Fatalf("prices tab missing content:\n%s", output)
	}
}

func TestModelTypingSchedulesSearchWhenRepoReady(t *testing.T) {
	t.Parallel()
	model := New(Options{})
	model.repo = &cardsdb.Repository{}
	_ = model.input.Focus()

	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Text: "a", Code: 'a'}))
	next := updated.(Model)
	if next.input.Value() != "a" {
		t.Fatalf("input value = %q, want a", next.input.Value())
	}
	if !next.searching {
		t.Fatalf("searching = false, want true")
	}
	if cmd == nil {
		t.Fatalf("cmd = nil, want debounce command")
	}
}

func TestModelTabSwitchesBetweenSearchAndResults(t *testing.T) {
	t.Parallel()
	model := New(Options{})
	_ = model.input.Focus()

	updated, _ := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	next := updated.(Model)
	if next.focus != focusResults {
		t.Fatalf("focus = %s, want %s", next.focus, focusResults)
	}
	if next.input.Focused() {
		t.Fatalf("input focused = true, want false")
	}

	updated, _ = next.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	next = updated.(Model)
	if next.focus != focusSearch {
		t.Fatalf("focus = %s, want %s", next.focus, focusSearch)
	}
}

func TestModelSyntaxPopupOpensAndCloses(t *testing.T) {
	t.Parallel()
	model := New(Options{})

	updated, _ := model.Update(tea.KeyPressMsg(tea.Key{Text: "?", Code: '?'}))
	next := updated.(Model)
	if !next.showSyntax {
		t.Fatalf("showSyntax = false, want true")
	}

	updated, _ = next.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}))
	next = updated.(Model)
	if next.showSyntax {
		t.Fatalf("showSyntax = true, want false")
	}
}

func TestSyntaxPopupOverlaysWithoutAddingHeight(t *testing.T) {
	t.Parallel()
	model := New(Options{})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	next := updated.(Model)
	next.showSyntax = true

	rendered := next.render()
	lines := strings.Split(rendered, "\n")
	if len(lines) > 40 {
		t.Fatalf("rendered height = %d lines, want at most 40", len(lines))
	}
	if !strings.Contains(rendered, "Search Syntax") {
		t.Fatalf("rendered syntax popup missing header")
	}
}
