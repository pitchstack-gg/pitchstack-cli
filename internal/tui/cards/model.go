package cards

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/pitchstack-gg/pitchstack-cli/internal/cardsdb"
	clientv1 "github.com/pitchstack-gg/pitchstack-go/client/v1"
)

type PricingClient interface {
	BatchGetProductPrices(context.Context, *clientv1.BatchGetProductPricesRequest, ...clientv1.RequestOpt) (*clientv1.BatchGetProductPricesResponse, error)
	GetProductPriceHistory(context.Context, *clientv1.GetProductPriceHistoryRequest, ...clientv1.RequestOpt) (*clientv1.GetProductPriceHistoryResponse, error)
}

type Options struct {
	Manager       *cardsdb.Manager
	ImageCacheDir string
	PricingClient PricingClient
	ForceRefresh  bool
	Offline       bool
}

type Model struct {
	options Options
	repo    *cardsdb.Repository

	input   textinput.Model
	results list.Model
	art     viewport.Model
	detail  viewport.Model
	prints  viewport.Model
	spinner spinner.Model
	help    help.Model

	width          int
	height         int
	searchSeq      int
	selectedID     string
	selectedURL    string
	selectedArtist string
	status         string
	err            string
	focus          focusArea
	showSyntax     bool
	loadingDB      bool
	searching      bool
	loadingCard    bool
	loadingArt     bool
	loadingPrices  bool
	baseDetail     *cardsdb.CardDetail
	related        *cardsdb.RelatedCards
	products       []cardsdb.PrintingProduct
	priceProductID string
	priceRange     priceRange
	priceEntries   []clientv1.PriceEntry
	priceHistory   []clientv1.PriceEntry
	priceErr       string
	printings      []cardsdb.CardPrinting
	printingLangs  []string
	activeLanguage string
	printingIndex  int
	rulesMode      rulesTextMode
	detailTab      detailTab
}

type focusArea string

const (
	focusSearch  focusArea = "search"
	focusResults focusArea = "results"
)

type rulesTextMode string

const (
	rulesTextTrue    rulesTextMode = "true"
	rulesTextPrinted rulesTextMode = "printed"
)

type detailTab string

const (
	detailTabDetails detailTab = "details"
	detailTabRelated detailTab = "related"
	detailTabPrices  detailTab = "prices"
)

type priceRange string

const (
	priceRange1M priceRange = "1m"
	priceRange3M priceRange = "3m"
	priceRange6M priceRange = "6m"
	priceRange1Y priceRange = "1y"
)

const printingRowHeight = 3
const pricingSourceTCGPlayer = "tcgplayer"
const noPriceRecordedMessage = "No price recorded"

type resultItem struct {
	cardsdb.SearchResult
}

func (i resultItem) Title() string {
	if strings.TrimSpace(i.Name) != "" {
		return i.Name
	}
	return i.ID
}

func (i resultItem) Description() string {
	parts := []string{}
	if i.TypeLine != "" {
		parts = append(parts, i.TypeLine)
	}
	stats := formatStats(i.Cost, i.Pitch, i.Power, i.Defense, i.Health)
	if stats != "" {
		parts = append(parts, stats)
	}
	if i.SetName != "" {
		parts = append(parts, i.SetName)
	}
	return strings.Join(parts, "  ")
}

type cardDelegate struct{}

func (d cardDelegate) Height() int {
	return 3
}

func (d cardDelegate) Spacing() int {
	return 1
}

func (d cardDelegate) Update(tea.Msg, *list.Model) tea.Cmd {
	return nil
}

func (d cardDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	card, ok := item.(resultItem)
	if !ok {
		return
	}
	name := nonEmpty(card.Name, card.ID)
	typeLine := strings.TrimSpace(card.TypeLine)
	stats := formatStats(card.Cost, card.Pitch, card.Power, card.Defense, card.Health)

	selected := index == m.Index() && m.FilterState() != list.Filtering
	prefix := "  "
	nameStyle := listTitleStyle
	metaStyle := listMetaStyle
	if selected {
		prefix = "│ "
		nameStyle = listSelectedTitleStyle
		metaStyle = listSelectedMetaStyle
	}
	metaPrefix := "  "
	if selected {
		metaPrefix = "│ "
	}

	lines := []string{
		nameStyle.Render(prefix + name),
		metaStyle.Render(metaPrefix + oneLine(typeLine)),
		metaStyle.Render(metaPrefix + oneLine(stats)),
	}
	_, _ = fmt.Fprint(w, strings.Join(lines, "\n"))
}

type keyMap struct {
	Enter   key.Binding
	Focus   key.Binding
	Search  key.Binding
	Syntax  key.Binding
	Prints  key.Binding
	Lang    key.Binding
	Rules   key.Binding
	Tabs    key.Binding
	Range   key.Binding
	Refresh key.Binding
	Quit    key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Focus, k.Search, k.Enter, k.Prints, k.Lang, k.Tabs, k.Range, k.Rules, k.Syntax, k.Refresh, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Focus, k.Search, k.Enter, k.Prints, k.Lang, k.Tabs, k.Range, k.Rules, k.Syntax, k.Refresh, k.Quit}}
}

var keys = keyMap{
	Enter:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	Focus:   key.NewBinding(key.WithKeys("tab", "esc"), key.WithHelp("tab/esc", "switch focus")),
	Search:  key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
	Syntax:  key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "syntax")),
	Prints:  key.NewBinding(key.WithKeys("[", "]"), key.WithHelp("[/]", "printings")),
	Lang:    key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "language")),
	Rules:   key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "rules text")),
	Tabs:    key.NewBinding(key.WithKeys("1", "2", "3"), key.WithHelp("1/2/3", "tabs")),
	Range:   key.NewBinding(key.WithKeys("4", "5", "6", "7"), key.WithHelp("4-7", "price range")),
	Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

type ensureDoneMsg struct {
	path    string
	updated bool
	err     error
}

type searchDebounceMsg struct {
	seq   int
	query string
}

type searchDoneMsg struct {
	seq     int
	query   string
	results []cardsdb.SearchResult
	err     error
}

type detailDoneMsg struct {
	id     string
	detail *cardsdb.CardDetail
	err    error
}

type printingsDoneMsg struct {
	id        string
	printings []cardsdb.CardPrinting
	err       error
}

type relatedDoneMsg struct {
	id      string
	related *cardsdb.RelatedCards
	err     error
}

type productsDoneMsg struct {
	id       string
	products []cardsdb.PrintingProduct
	err      error
}

type pricesDoneMsg struct {
	id        string
	productID string
	rangeKey  priceRange
	current   []clientv1.PriceEntry
	history   []clientv1.PriceEntry
	err       error
}

type artDoneMsg struct {
	id  string
	art string
	err error
}

func New(opts Options) Model {
	input := textinput.New()
	input.Placeholder = "Search cards by name, id, or rules text"
	input.Prompt = "Search*: "
	input.CharLimit = 120
	input.SetWidth(64)
	_ = input.Focus()
	results := list.New(nil, cardDelegate{}, 40, 12)
	results.Title = "Cards"
	results.SetShowTitle(false)
	results.SetShowFilter(false)
	results.SetShowHelp(false)
	results.SetShowStatusBar(false)
	results.SetShowPagination(false)
	results.DisableQuitKeybindings()

	art := viewport.New()
	art.SoftWrap = false
	art.SetContent("Card art will appear here.")
	detail := viewport.New()
	detail.SoftWrap = true
	detail.SetContent("Select a card to view details.")
	prints := viewport.New()
	prints.SoftWrap = true
	prints.SetContent("Select a card to view printings.")
	spin := spinner.New(spinner.WithSpinner(spinner.Line))

	return Model{
		options:    opts,
		input:      input,
		results:    results,
		art:        art,
		detail:     detail,
		prints:     prints,
		spinner:    spin,
		help:       help.New(),
		status:     "Preparing card browser...",
		focus:      focusSearch,
		loadingDB:  true,
		rulesMode:  rulesTextTrue,
		detailTab:  detailTabDetails,
		priceRange: priceRange3M,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.input.Focus(), m.spinner.Tick, m.ensureDBCmd(false))
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
		if m.selectedID != "" && m.selectedURL != "" {
			cmds = append(cmds, m.renderArtCmd(m.selectedID, m.selectedURL))
		}
	case tea.KeyPressMsg:
		keyText := msg.String()
		if m.showSyntax {
			switch keyText {
			case "ctrl+c":
				return m, tea.Quit
			case "?", "esc", "enter", "q":
				m.showSyntax = false
			}
			return m, tea.Batch(cmds...)
		}

		switch keyText {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.focus == focusResults {
				return m, tea.Quit
			}
		case "?":
			m.showSyntax = true
			return m, tea.Batch(cmds...)
		case "tab":
			cmds = append(cmds, m.toggleFocus())
			return m, tea.Batch(cmds...)
		case "esc":
			cmds = append(cmds, m.setFocus(focusResults))
			return m, tea.Batch(cmds...)
		case "/":
			if m.focus == focusResults {
				cmds = append(cmds, m.setFocus(focusSearch))
				return m, tea.Batch(cmds...)
			}
		case "r":
			if m.focus == focusResults {
				m.err = ""
				m.status = "Refreshing card database..."
				m.loadingDB = true
				cmds = append(cmds, m.ensureDBCmd(true), m.spinner.Tick)
				return m, tea.Batch(cmds...)
			}
		case "[", "]":
			if m.focus == focusResults && len(m.visiblePrintings()) > 0 {
				delta := 1
				if keyText == "[" {
					delta = -1
				}
				cmds = append(cmds, m.selectPrinting(m.printingIndex+delta))
				return m, tea.Batch(cmds...)
			}
		case "l":
			if m.focus == focusResults && len(m.printingLangs) > 1 {
				cmds = append(cmds, m.selectPrintingLanguage(1))
				return m, tea.Batch(cmds...)
			}
		case "t":
			if m.focus == focusResults && m.baseDetail != nil {
				m.toggleRulesTextMode()
				return m, tea.Batch(cmds...)
			}
		case "1", "2", "3":
			if m.focus == focusResults {
				m.setDetailTab(keyText)
				return m, tea.Batch(cmds...)
			}
		case "4", "5", "6", "7":
			if m.focus == focusResults && m.detailTab == detailTabPrices {
				if cmd := m.setPriceRange(keyText); cmd != nil {
					cmds = append(cmds, cmd)
				}
				return m, tea.Batch(cmds...)
			}
		case "enter":
			if m.focus == focusSearch {
				cmds = append(cmds, m.setFocus(focusResults))
				return m, tea.Batch(cmds...)
			}
			if item, ok := m.results.SelectedItem().(resultItem); ok {
				cmds = append(cmds, m.selectCard(item.ID, nonEmpty(item.ArtURL, item.ImageURL)))
				return m, tea.Batch(cmds...)
			}
		case "down":
			if m.focus == focusSearch {
				cmds = append(cmds, m.setFocus(focusResults))
				return m, tea.Batch(cmds...)
			}
		}
	case ensureDoneMsg:
		m.loadingDB = false
		if msg.err != nil {
			m.err = msg.err.Error()
			m.status = "Card database unavailable."
			break
		}
		repo, err := cardsdb.OpenRepository(msg.path)
		if err != nil {
			m.err = err.Error()
			m.status = "Could not open card database."
			break
		}
		if m.repo != nil {
			_ = m.repo.Close()
		}
		m.repo = repo
		if msg.updated {
			m.status = "Card database updated."
		} else {
			m.status = "Card database ready."
		}
		cmds = append(cmds, m.scheduleSearch())
	case searchDebounceMsg:
		if msg.seq == m.searchSeq {
			cmds = append(cmds, m.searchCmd(msg.seq, msg.query))
		}
	case searchDoneMsg:
		if msg.seq != m.searchSeq {
			break
		}
		m.searching = false
		if msg.err != nil {
			m.err = msg.err.Error()
			m.status = "Search failed."
			break
		}
		m.err = ""
		items := make([]list.Item, 0, len(msg.results))
		for _, result := range msg.results {
			items = append(items, resultItem{SearchResult: result})
		}
		cmds = append(cmds, m.results.SetItems(items))
		if len(items) == 0 {
			m.selectedID = ""
			m.selectedURL = ""
			m.selectedArtist = ""
			m.baseDetail = nil
			m.related = nil
			m.products = nil
			m.priceProductID = ""
			m.priceEntries = nil
			m.priceHistory = nil
			m.priceErr = ""
			m.loadingPrices = false
			m.printings = nil
			m.printingLangs = nil
			m.activeLanguage = ""
			m.printingIndex = 0
			m.art.SetContent("No matching card art.")
			m.detail.SetContent("No cards found.")
			m.prints.SetContent("No printings.")
			m.status = fmt.Sprintf("No results for %q.", msg.query)
			break
		}
		m.results.Select(0)
		first := items[0].(resultItem)
		m.status = fmt.Sprintf("%d result(s).", len(items))
		cmds = append(cmds, m.selectCard(first.ID, nonEmpty(first.ArtURL, first.ImageURL)))
	case detailDoneMsg:
		if msg.id != m.selectedID {
			break
		}
		m.loadingCard = false
		if msg.err != nil {
			m.err = msg.err.Error()
			m.detail.SetContent("Could not load card details.")
			break
		}
		m.err = ""
		m.baseDetail = msg.detail
		m.detail.SetContent(m.renderDetail())
		if msg.detail != nil {
			artURL := nonEmpty(msg.detail.ArtURL, msg.detail.ImageURL)
			if artURL != "" {
				m.selectedURL = artURL
				m.selectedArtist = strings.Join(msg.detail.Artists, ", ")
				cmds = append(cmds, m.renderArtCmd(msg.id, artURL))
			}
		}
	case printingsDoneMsg:
		if msg.id != m.selectedID {
			break
		}
		if msg.err != nil {
			m.err = msg.err.Error()
			m.prints.SetContent("Could not load printings.")
			break
		}
		m.printings = msg.printings
		m.printingLangs = printingLanguages(msg.printings)
		m.activeLanguage = defaultPrintingLanguage(m.printingLangs)
		m.printingIndex = clampIndex(m.printingIndex, len(m.visiblePrintings()))
		m.prints.SetContent(m.renderPrintings())
		if len(m.visiblePrintings()) > 0 {
			cmds = append(cmds, m.selectPrinting(m.printingIndex))
		}
	case relatedDoneMsg:
		if msg.id != m.selectedID {
			break
		}
		if msg.err != nil {
			m.err = msg.err.Error()
			break
		}
		m.related = msg.related
		if m.detailTab == detailTabRelated {
			m.detail.SetContent(m.renderDetail())
		}
	case productsDoneMsg:
		if msg.id != m.selectedID {
			break
		}
		if msg.err != nil {
			m.err = msg.err.Error()
			break
		}
		m.products = msg.products
		cmds = append(cmds, m.preparePriceLoad())
		if m.detailTab == detailTabPrices {
			m.detail.SetContent(m.renderDetail())
		}
	case pricesDoneMsg:
		if msg.id != m.selectedID || msg.productID != m.priceProductID || msg.rangeKey != m.priceRange {
			break
		}
		m.loadingPrices = false
		m.priceEntries = msg.current
		m.priceHistory = msg.history
		if isPriceNotFoundError(msg.err) {
			m.priceErr = noPriceRecordedMessage
		} else if msg.err != nil {
			m.priceErr = msg.err.Error()
		} else {
			m.priceErr = ""
		}
		if m.detailTab == detailTabPrices {
			m.detail.SetContent(m.renderDetail())
		}
	case artDoneMsg:
		if msg.id != m.selectedID {
			break
		}
		m.loadingArt = false
		if msg.err != nil {
			m.art.SetContent("Could not render card art:\n" + msg.err.Error())
			break
		}
		m.art.SetContent(msg.art)
	}

	if m.loadingDB || m.searching || m.loadingCard || m.loadingArt || m.loadingPrices {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
		if m.loadingPrices && m.detailTab == detailTabPrices {
			m.detail.SetContent(m.renderDetail())
		}
	}

	if m.focus == focusSearch {
		oldQuery := m.input.Value()
		var inputCmd tea.Cmd
		m.input, inputCmd = m.input.Update(msg)
		cmds = append(cmds, inputCmd)
		if m.repo != nil && m.input.Value() != oldQuery {
			cmds = append(cmds, m.scheduleSearch())
		}
	} else {
		oldSelected := m.selectedID
		var listCmd tea.Cmd
		m.results, listCmd = m.results.Update(msg)
		cmds = append(cmds, listCmd)
		if item, ok := m.results.SelectedItem().(resultItem); ok && item.ID != "" && item.ID != oldSelected {
			cmds = append(cmds, m.selectCard(item.ID, nonEmpty(item.ArtURL, item.ImageURL)))
		}
	}

	var artCmd tea.Cmd
	m.art, artCmd = m.art.Update(msg)
	cmds = append(cmds, artCmd)
	var detailCmd tea.Cmd
	m.detail, detailCmd = m.detail.Update(msg)
	cmds = append(cmds, detailCmd)
	var printsCmd tea.Cmd
	m.prints, printsCmd = m.prints.Update(msg)
	cmds = append(cmds, printsCmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() tea.View {
	content := m.render()
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func (m *Model) resize() {
	if m.width <= 0 || m.height <= 0 {
		return
	}
	headerH := 3
	if m.err != "" {
		headerH++
	}
	footerH := 1
	bodyH := max(4, m.height-headerH-footerH)
	verticalChrome := panelStyle.GetVerticalFrameSize()
	horizontalChrome := panelStyle.GetHorizontalFrameSize()
	maxArtOuterH := 24
	if m.width < 86 {
		resultOuterH := max(7, bodyH/3)
		remainingH := max(8, bodyH-resultOuterH)
		artOuterH := min(maxArtOuterH, max(10, (remainingH*58)/100))
		detailOuterH := max(6, remainingH-artOuterH)
		m.results.SetSize(m.width, max(3, resultOuterH-verticalChrome-1))
		m.art.SetWidth(max(8, m.width-horizontalChrome))
		m.art.SetHeight(max(4, artOuterH-verticalChrome))
		m.detail.SetWidth(max(20, m.width-horizontalChrome))
		m.detail.SetHeight(max(6, detailOuterH-verticalChrome))
		m.prints.SetWidth(max(20, m.width-horizontalChrome))
		m.prints.SetHeight(max(5, detailOuterH-verticalChrome))
	} else {
		leftW := min(42, max(30, m.width/3))
		rightW := max(20, m.width-leftW-2)
		artOuterH := min(maxArtOuterH, max(14, (bodyH*58)/100))
		detailOuterH := max(6, bodyH-artOuterH)
		printsOuterW := min(38, max(28, rightW/4))
		artOuterW := max(20, rightW-printsOuterW)
		m.results.SetSize(leftW, max(3, bodyH-verticalChrome-1))
		m.art.SetWidth(max(8, artOuterW-horizontalChrome))
		m.art.SetHeight(max(4, artOuterH-verticalChrome))
		m.detail.SetWidth(max(20, rightW-horizontalChrome))
		m.detail.SetHeight(max(6, detailOuterH-verticalChrome))
		m.prints.SetWidth(max(20, printsOuterW-horizontalChrome))
		m.prints.SetHeight(max(5, artOuterH-verticalChrome))
	}
	m.input.SetWidth(max(20, m.width-10))
	m.help.SetWidth(m.width)
}

func (m *Model) setFocus(next focusArea) tea.Cmd {
	m.focus = next
	if next == focusSearch {
		m.input.Prompt = "Search*: "
		return m.input.Focus()
	}
	m.input.Prompt = "Search: "
	m.input.Blur()
	return nil
}

func (m *Model) toggleFocus() tea.Cmd {
	if m.focus == focusSearch {
		return m.setFocus(focusResults)
	}
	return m.setFocus(focusSearch)
}

func (m Model) render() string {
	title := titleStyle.Render("Pitchstack")
	search := sectionLabelStyle.Render("Search") + "\n" + inputStyle.Width(max(20, m.width)).Render(m.input.View())
	header := lipgloss.JoinVertical(lipgloss.Left, title, search)
	if m.err != "" {
		header = lipgloss.JoinVertical(lipgloss.Left, header, errorStyle.Render(m.err))
	}

	body := ""
	resultPanel := panelStyle
	if m.focus == focusResults {
		resultPanel = activePanelStyle
	}
	resultContent := sectionLabelStyle.Render("Cards") + "\n" + m.results.View()
	resultOuterH := m.results.Height() + panelStyle.GetVerticalFrameSize() + 1
	if m.width < 86 {
		body = lipgloss.JoinVertical(lipgloss.Left,
			renderPanel(resultPanel, max(20, m.width-2), resultOuterH, resultContent),
			renderPanel(panelStyle, max(20, m.width-2), m.art.Height()+panelStyle.GetVerticalFrameSize(), m.art.View()),
			renderPanel(panelStyle, max(20, m.width-2), m.detail.Height()+panelStyle.GetVerticalFrameSize(), m.detail.View()),
		)
	} else {
		top := lipgloss.JoinHorizontal(lipgloss.Top,
			renderPanel(panelStyle, m.art.Width()+panelStyle.GetHorizontalFrameSize(), m.art.Height()+panelStyle.GetVerticalFrameSize(), m.art.View()),
			renderPanel(panelStyle, m.prints.Width()+panelStyle.GetHorizontalFrameSize(), m.prints.Height()+panelStyle.GetVerticalFrameSize(), m.prints.View()),
		)
		right := lipgloss.JoinVertical(lipgloss.Left,
			top,
			renderPanel(panelStyle, m.detail.Width()+panelStyle.GetHorizontalFrameSize(), m.detail.Height()+panelStyle.GetVerticalFrameSize(), m.detail.View()),
		)
		body = lipgloss.JoinHorizontal(lipgloss.Top,
			renderPanel(resultPanel, m.results.Width(), resultOuterH, resultContent),
			right,
		)
	}
	footer := m.help.ShortHelpView(keys.ShortHelp())
	if m.help.ShowAll {
		footer = m.help.FullHelpView(keys.FullHelp())
	}
	content := lipgloss.JoinVertical(lipgloss.Left, header, body, mutedStyle.Render(footer))
	if m.showSyntax {
		return overlayModal(m.width, m.height, content, syntaxModal(m.width))
	}
	return content
}

func renderPanel(style lipgloss.Style, width, height int, content string) string {
	return style.Width(width).Height(height).MaxWidth(width).MaxHeight(height).Render(content)
}

func (m Model) ensureDBCmd(force bool) tea.Cmd {
	return func() tea.Msg {
		if m.options.Manager == nil {
			return ensureDoneMsg{err: fmt.Errorf("cards database manager is not configured")}
		}
		result, err := m.options.Manager.Ensure(context.Background(), cardsdb.EnsureOptions{
			Force:   force || m.options.ForceRefresh,
			Offline: m.options.Offline,
		})
		if err != nil {
			return ensureDoneMsg{err: err}
		}
		return ensureDoneMsg{path: result.DBPath, updated: result.Updated}
	}
}

func (m *Model) scheduleSearch() tea.Cmd {
	m.searchSeq++
	seq := m.searchSeq
	query := m.input.Value()
	m.searching = true
	m.status = "Searching..."
	return tea.Tick(220*time.Millisecond, func(time.Time) tea.Msg {
		return searchDebounceMsg{seq: seq, query: query}
	})
}

func (m Model) searchCmd(seq int, query string) tea.Cmd {
	return func() tea.Msg {
		if m.repo == nil {
			return searchDoneMsg{seq: seq, query: query, err: fmt.Errorf("card database is not ready")}
		}
		results, err := m.repo.Search(context.Background(), cardsdb.SearchParams{Query: query, Limit: 50})
		return searchDoneMsg{seq: seq, query: query, results: results, err: err}
	}
}

func (m *Model) selectCard(id, imageURL string) tea.Cmd {
	m.selectedID = id
	m.selectedURL = imageURL
	m.selectedArtist = ""
	m.baseDetail = nil
	m.related = nil
	m.products = nil
	m.priceProductID = ""
	m.priceEntries = nil
	m.priceHistory = nil
	m.priceErr = ""
	m.loadingPrices = false
	m.printings = nil
	m.printingLangs = nil
	m.activeLanguage = ""
	m.printingIndex = 0
	m.detailTab = detailTabDetails
	m.loadingCard = true
	m.loadingArt = imageURL != ""
	m.status = "Loading card..."
	m.detail.SetContent("Loading details...")
	m.prints.SetContent("Loading printings...")
	if imageURL == "" {
		m.art.SetContent("No image available for this card.")
	} else {
		m.art.SetContent("Rendering card art...")
	}
	return tea.Batch(m.detailCmd(id), m.printingsCmd(id), m.relatedCmd(id), m.productsCmd(id), m.renderArtCmd(id, imageURL), m.spinner.Tick)
}

func (m Model) detailCmd(id string) tea.Cmd {
	return func() tea.Msg {
		if m.repo == nil {
			return detailDoneMsg{id: id, err: fmt.Errorf("card database is not ready")}
		}
		detail, err := m.repo.GetCard(context.Background(), id)
		return detailDoneMsg{id: id, detail: detail, err: err}
	}
}

func (m Model) printingsCmd(id string) tea.Cmd {
	return func() tea.Msg {
		if m.repo == nil {
			return printingsDoneMsg{id: id, err: fmt.Errorf("card database is not ready")}
		}
		printings, err := m.repo.ListPrintings(context.Background(), id)
		return printingsDoneMsg{id: id, printings: printings, err: err}
	}
}

func (m Model) relatedCmd(id string) tea.Cmd {
	return func() tea.Msg {
		if m.repo == nil {
			return relatedDoneMsg{id: id, err: fmt.Errorf("card database is not ready")}
		}
		related, err := m.repo.ListRelatedCards(context.Background(), id)
		return relatedDoneMsg{id: id, related: related, err: err}
	}
}

func (m Model) productsCmd(id string) tea.Cmd {
	return func() tea.Msg {
		if m.repo == nil {
			return productsDoneMsg{id: id, err: fmt.Errorf("card database is not ready")}
		}
		products, err := m.repo.ListCardProducts(context.Background(), id)
		return productsDoneMsg{id: id, products: products, err: err}
	}
}

func (m Model) pricesCmd(id, productID string, rangeKey priceRange, productIDs []string) tea.Cmd {
	return func() tea.Msg {
		client := m.options.PricingClient
		if client == nil {
			return pricesDoneMsg{id: id, productID: productID, rangeKey: rangeKey, err: fmt.Errorf("pricing API client is not configured")}
		}
		productIDs = uniqueValues(productIDs, 20)
		if productID == "" || len(productIDs) == 0 {
			return pricesDoneMsg{id: id, productID: productID, rangeKey: rangeKey}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()

		var current []clientv1.PriceEntry
		var fetchErr error
		batch, err := client.BatchGetProductPrices(ctx, &clientv1.BatchGetProductPricesRequest{
			ProductIDs: productIDs,
			Source:     pricingSourceTCGPlayer,
		})
		if err != nil {
			fetchErr = err
		} else if batch != nil {
			current = batch.Prices
		}

		cfg := priceRangeConfig(rangeKey)
		limit := cfg.limit
		end := time.Now()
		start := cfg.start(end)
		var history []clientv1.PriceEntry
		hist, err := client.GetProductPriceHistory(ctx, &clientv1.GetProductPriceHistoryRequest{
			ProductID: productID,
			StartDate: start.UTC().Format(time.RFC3339Nano),
			EndDate:   end.UTC().Format(time.RFC3339Nano),
			Limit:     &limit,
			Source:    pricingSourceTCGPlayer,
		})
		if err != nil {
			if fetchErr == nil {
				fetchErr = err
			}
		} else if hist != nil {
			history = hist.Entries
		}

		return pricesDoneMsg{id: id, productID: productID, rangeKey: rangeKey, current: current, history: history, err: fetchErr}
	}
}

func (m Model) renderArtCmd(id, imageURL string) tea.Cmd {
	return func() tea.Msg {
		if strings.TrimSpace(imageURL) == "" {
			return artDoneMsg{id: id, art: "No image available for this card."}
		}
		renderer := ArtRenderer{CacheDir: m.options.ImageCacheDir}
		artist := strings.TrimSpace(m.selectedArtist)
		artHeight := m.art.Height()
		if artHeight > 5 {
			artHeight--
		}
		art, err := renderer.Render(context.Background(), imageURL, max(8, m.art.Width()), max(4, artHeight))
		if err == nil {
			credit := ""
			if artist != "" {
				credit = mutedStyle.Render("Illustrated by: " + artist)
			}
			art = lipgloss.JoinVertical(lipgloss.Left, art, lipgloss.PlaceHorizontal(max(8, m.art.Width()), lipgloss.Right, credit))
		}
		return artDoneMsg{id: id, art: art, err: err}
	}
}

func (m *Model) selectPrinting(index int) tea.Cmd {
	visible := m.visiblePrintings()
	if len(visible) == 0 {
		m.prints.SetContent("No printings found.")
		return nil
	}
	m.printingIndex = wrapIndex(index, len(visible))
	m.prints.SetContent(m.renderPrintings())
	m.ensureSelectedPrintingVisible()
	selected := visible[m.printingIndex]
	m.selectedURL = nonEmpty(selected.ArtURL, selected.ImageURL, m.selectedURL)
	m.selectedArtist = strings.Join(selected.Artists, ", ")
	priceCmd := m.preparePriceLoad()
	if m.baseDetail != nil {
		m.detail.SetContent(m.renderDetail())
	}
	if m.selectedURL == "" {
		m.art.SetContent("No image available for this printing.")
		return priceCmd
	}
	m.loadingArt = true
	m.status = "Rendering printing art..."
	return tea.Batch(m.renderArtCmd(m.selectedID, m.selectedURL), priceCmd)
}

func (m *Model) preparePriceLoad() tea.Cmd {
	if m.options.PricingClient == nil || len(m.products) == 0 {
		m.loadingPrices = false
		return nil
	}
	productID := m.selectedPriceProductID()
	if productID == "" {
		m.loadingPrices = false
		return nil
	}
	if productID == m.priceProductID && (m.loadingPrices || len(m.priceEntries) > 0 || len(m.priceHistory) > 0 || m.priceErr != "") {
		return nil
	}
	m.priceProductID = productID
	m.priceEntries = nil
	m.priceHistory = nil
	m.priceErr = ""
	m.loadingPrices = true
	productIDs := make([]string, 0, len(m.products))
	for _, product := range m.products {
		if strings.TrimSpace(product.ID) != "" {
			productIDs = append(productIDs, product.ID)
		}
	}
	return tea.Batch(m.pricesCmd(m.selectedID, productID, m.priceRange, productIDs), m.spinner.Tick)
}

func (m Model) selectedPriceProductID() string {
	printing, ok := m.selectedPrinting()
	selectedPrintingID := ""
	if ok {
		selectedPrintingID = strings.TrimSpace(printing.ID)
	}
	for _, product := range m.products {
		if selectedPrintingID != "" && strings.TrimSpace(product.PrintingID) == selectedPrintingID && strings.TrimSpace(product.ID) != "" {
			return strings.TrimSpace(product.ID)
		}
	}
	for _, product := range m.products {
		if strings.TrimSpace(product.ID) != "" {
			return strings.TrimSpace(product.ID)
		}
	}
	return ""
}

func (m Model) selectedPriceProduct() (cardsdb.PrintingProduct, bool) {
	id := strings.TrimSpace(m.priceProductID)
	if id == "" {
		id = m.selectedPriceProductID()
	}
	for _, product := range m.products {
		if strings.TrimSpace(product.ID) == id {
			return product, true
		}
	}
	return cardsdb.PrintingProduct{}, false
}

func (m Model) detailWithSelectedPrinting() *cardsdb.CardDetail {
	if m.baseDetail == nil {
		return nil
	}
	detail := *m.baseDetail
	printing, ok := m.selectedPrinting()
	if !ok {
		return &detail
	}
	if printing.ID != "" {
		detail.PrintingID = printing.ID
	}
	if printing.Name != "" {
		detail.Printing = printing.Name
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
	if printing.ImageURL != "" {
		detail.ImageURL = printing.ImageURL
	}
	if printing.ArtURL != "" {
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
	return &detail
}

func (m Model) renderDetail() string {
	width := max(20, m.detail.Width())
	tabs := renderDetailTabs(m.detailTab, width)
	var body string
	switch m.detailTab {
	case detailTabRelated:
		body = renderRelatedTab(m.related, width)
	case detailTabPrices:
		body = renderPricesTab(m.priceRange, m.priceProductID, m.priceEntries, m.priceHistory, m.loadingPrices, m.spinner.View(), m.priceErr, width, m.detail.Height())
	default:
		body = renderDetail(m.detailWithSelectedPrinting(), m.rulesMode, width)
	}
	return strings.TrimRight(tabs+"\n\n"+body, "\n")
}

func (m *Model) toggleRulesTextMode() {
	if m.rulesMode == rulesTextPrinted {
		m.rulesMode = rulesTextTrue
		m.status = "Showing true rules text."
	} else {
		m.rulesMode = rulesTextPrinted
		m.status = "Showing printed rules text."
	}
	m.detail.SetContent(m.renderDetail())
}

func (m *Model) setDetailTab(keyText string) {
	switch keyText {
	case "2":
		m.detailTab = detailTabRelated
	case "3":
		m.detailTab = detailTabPrices
	default:
		m.detailTab = detailTabDetails
	}
	m.detail.SetYOffset(0)
	m.detail.SetContent(m.renderDetail())
}

func (m *Model) setPriceRange(keyText string) tea.Cmd {
	next := priceRangeFromKey(keyText)
	if next == "" || next == m.priceRange {
		return nil
	}
	m.priceRange = next
	m.priceEntries = nil
	m.priceHistory = nil
	m.priceErr = ""
	m.loadingPrices = false
	m.detail.SetYOffset(0)
	cmd := m.preparePriceLoad()
	m.detail.SetContent(m.renderDetail())
	return cmd
}

func (m Model) renderPrintings() string {
	if len(m.printings) == 0 {
		return sectionLabelStyle.Render("Printings") + "\n" + mutedStyle.Render("No printings found.")
	}
	visible := m.visiblePrintings()
	lines := []string{
		sectionLabelStyle.Render(fmt.Sprintf("Printings  %d", len(visible))),
		m.renderLanguageTabs(),
		"",
	}
	if len(visible) == 0 {
		lines = append(lines, mutedStyle.Render("No printings for "+strings.ToUpper(m.activeLanguage)+"."))
		return strings.Join(lines, "\n")
	}
	rowWidth := m.prints.Width() - 2
	if rowWidth <= 0 {
		rowWidth = 40
	}
	for i, printing := range visible {
		prefix := "  "
		nameStyle := listTitleStyle
		metaStyle := listMetaStyle
		if i == m.printingIndex {
			prefix = "│ "
			nameStyle = listSelectedTitleStyle
			metaStyle = listSelectedMetaStyle
		}
		label := nonEmpty(printing.ID, printing.SetCode, printing.SetName)
		if printing.Rarity != "" {
			label += " - " + titleStatus(printing.Rarity)
		}
		setName := nonEmpty(printing.SetName, printing.SetCode)
		lines = append(lines, nameStyle.Render(prefix+label))
		for _, line := range wrapIndentedText(setName, rowWidth-lipgloss.Width(prefix), "") {
			lines = append(lines, metaStyle.Render(prefix+line))
		}
		lines = append(lines, "")
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}

func (m *Model) ensureSelectedPrintingVisible() {
	if len(m.visiblePrintings()) == 0 || m.prints.Height() <= 0 {
		return
	}
	line := 3 + m.printingIndex*printingRowHeight
	pageSize := max(1, m.prints.Height())
	pageTop := (line / pageSize) * pageSize
	if line < m.prints.YOffset() || line >= m.prints.YOffset()+pageSize {
		m.prints.SetYOffset(pageTop)
	}
}

func (m *Model) selectPrintingLanguage(delta int) tea.Cmd {
	if len(m.printingLangs) == 0 {
		return nil
	}
	current := 0
	for i, lang := range m.printingLangs {
		if lang == m.activeLanguage {
			current = i
			break
		}
	}
	m.activeLanguage = m.printingLangs[wrapIndex(current+delta, len(m.printingLangs))]
	m.printingIndex = 0
	m.prints.SetYOffset(0)
	return m.selectPrinting(0)
}

func (m Model) selectedPrinting() (cardsdb.CardPrinting, bool) {
	visible := m.visiblePrintings()
	if len(visible) == 0 {
		return cardsdb.CardPrinting{}, false
	}
	return visible[wrapIndex(m.printingIndex, len(visible))], true
}

func (m Model) visiblePrintings() []cardsdb.CardPrinting {
	if strings.TrimSpace(m.activeLanguage) == "" {
		return m.printings
	}
	out := make([]cardsdb.CardPrinting, 0, len(m.printings))
	for _, printing := range m.printings {
		if normalizeLanguage(printing.Language) == m.activeLanguage {
			out = append(out, printing)
		}
	}
	return out
}

func (m Model) renderLanguageTabs() string {
	if len(m.printingLangs) == 0 {
		return mutedStyle.Render("Languages: none")
	}
	parts := make([]string, 0, len(m.printingLangs))
	for _, lang := range m.printingLangs {
		label := strings.ToUpper(lang)
		if lang == m.activeLanguage {
			parts = append(parts, listSelectedTitleStyle.Render("["+label+"]"))
		} else {
			parts = append(parts, mutedStyle.Render(label))
		}
	}
	return strings.Join(parts, "  ")
}

func renderDetailTabs(active detailTab, width int) string {
	tabs := []struct {
		key   string
		tab   detailTab
		label string
	}{
		{"1", detailTabDetails, "Details"},
		{"2", detailTabRelated, "Related"},
		{"3", detailTabPrices, "Prices"},
	}
	parts := make([]string, 0, len(tabs))
	for _, tab := range tabs {
		label := tab.key + " " + tab.label
		if tab.tab == active {
			parts = append(parts, detailTabActiveStyle.Render("["+label+"]"))
		} else {
			parts = append(parts, detailTabStyle.Render(label))
		}
	}
	return lipgloss.NewStyle().Width(width).Render(strings.Join(parts, "  "))
}

func renderDetail(d *cardsdb.CardDetail, mode rulesTextMode, width int) string {
	if d == nil {
		return "No card details."
	}
	width = max(20, width)
	headerRight := strings.TrimSpace(formatStats(d.Cost, d.Pitch, "", "", ""))
	lines := []string{splitLine(detailTitleStyle.Render(nonEmpty(d.Name, d.ID)), headerRight, width)}
	otherStats := renderInlineStats(d, false)
	if d.TypeLine != "" || otherStats != "" {
		lines = append(lines, splitLine(oneLine(d.TypeLine), mutedStyle.Render(otherStats), width))
	}

	cardStats := renderStatLines(d)
	if cardStats != "" && width < 78 {
		appendDetailSection(&lines, "Stats", cardStats)
	}

	rawRulesText := d.Text
	rulesTitle := "Rules Text"
	if mode == rulesTextPrinted {
		rulesTitle = "Printed Text"
		rawRulesText = nonEmpty(d.PrintedText, d.Text)
	}
	rulesText := renderRulesText(rawRulesText)
	if rulesText == "" {
		rulesText = mutedStyle.Render("No rules text.")
	}
	textBody := detailSectionStyle.Render(rulesTitle) + "\n" + wrapBlock(rulesText, contentWidth(width))
	if d.FlavorText != "" {
		textBody += "\n\n" + detailSectionStyle.Render("Flavor") + "\n" + mutedStyle.Italic(true).Render(wrapBlock(renderRulesText(d.FlavorText), contentWidth(width)))
	}

	if width >= 78 {
		leftW := max(30, width-38)
		rightW := min(34, width-leftW-2)
		left := lipgloss.NewStyle().Width(leftW).Render(textBody)
		right := lipgloss.NewStyle().Width(rightW).Render(renderLegalityColumn(d.Legality))
		lines = append(lines, "", lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right))
	} else {
		lines = append(lines, "", textBody)
		if legality := renderLegalityColumn(d.Legality); legality != "" {
			appendDetailSection(&lines, "Legality", legality)
		}
	}

	return strings.Join(lines, "\n")
}

func renderRelatedTab(related *cardsdb.RelatedCards, width int) string {
	if related == nil {
		return mutedStyle.Render("Loading related cards...")
	}
	lines := []string{}
	appendRelatedSection := func(title string, cards []cardsdb.RelatedCard) {
		if len(cards) == 0 {
			return
		}
		lines = append(lines, detailSectionStyle.Render(title))
		for _, card := range cards {
			name := nonEmpty(card.Name, card.ID)
			line := "  " + listTitleStyle.Render(name)
			if card.TypeLine != "" {
				line += mutedStyle.Render("  " + oneLine(card.TypeLine))
			}
			lines = append(lines, lipgloss.NewStyle().Width(width).Render(line))
		}
		lines = append(lines, "")
	}
	appendRelatedSection("Pitch Siblings", related.Siblings)
	appendRelatedSection("Referenced Cards", related.References)
	appendRelatedSection("Referenced By", related.ReferencedBy)
	if len(lines) == 0 {
		return mutedStyle.Render("No related card links found in the local database.")
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}

func renderPricesTab(rangeKey priceRange, productID string, current []clientv1.PriceEntry, history []clientv1.PriceEntry, loading bool, spinnerFrame, priceErr string, width, height int) string {
	entry, hasCurrent := priceForProduct(current, productID)
	chartHeight := min(16, max(7, height-9))
	noPriceRecorded := priceErr == noPriceRecordedMessage
	var chart string
	switch {
	case loading:
		chart = renderPriceMessageChart(max(24, width-2), chartHeight, strings.TrimSpace(spinnerFrame+" Loading prices..."))
	case noPriceRecorded && len(history) == 0:
		chart = renderPriceMessageChart(max(24, width-2), chartHeight, noPriceRecordedMessage)
	default:
		chart = renderPriceHistoryChart(history, max(24, width-2), chartHeight)
	}
	lines := []string{
		splitLine(detailSectionStyle.Render("Prices"), renderPriceRangeControls(rangeKey), width),
		chart,
		"",
	}
	if loading {
		lines = append(lines, mutedStyle.Render("Fetching latest TCGplayer prices..."))
	} else if noPriceRecorded {
		if hasCurrent {
			lines = append(lines, renderCurrentPriceSummary(entry, hasCurrent))
		}
	} else if priceErr != "" {
		lines = append(lines, errorStyle.Render("Pricing API: "+priceErr))
	} else if !hasCurrent && len(history) == 0 {
		lines = append(lines, mutedStyle.Render(noPriceRecordedMessage))
	} else {
		lines = append(lines, renderCurrentPriceSummary(entry, hasCurrent))
	}
	return strings.Join(lines, "\n")
}

func appendDetailSection(lines *[]string, title, body string) {
	body = strings.TrimSpace(body)
	if body == "" {
		return
	}
	*lines = append(*lines, "", detailSectionStyle.Render(title), body)
}

func contentWidth(width int) int {
	return max(20, width-2)
}

func splitLine(left, right string, width int) string {
	right = strings.TrimSpace(right)
	if right == "" || width <= 0 {
		return left
	}
	rightW := lipgloss.Width(right)
	leftW := lipgloss.Width(left)
	if leftW+rightW+2 > width {
		return strings.TrimSpace(left) + "\n" + right
	}
	return left + strings.Repeat(" ", max(1, width-leftW-rightW)) + right
}

func wrapBlock(text string, width int) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	return lipgloss.NewStyle().Width(width).Render(text)
}

func wrapIndentedText(text string, width int, indent string) []string {
	text = oneLine(text)
	if text == "" {
		return nil
	}
	width = max(1, width)
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	lines := []string{}
	current := ""
	for _, word := range words {
		if current == "" {
			current = word
			continue
		}
		if lipgloss.Width(current)+1+lipgloss.Width(word) > width {
			lines = append(lines, current)
			current = indent + word
			continue
		}
		current += " " + word
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func renderStatLines(d *cardsdb.CardDetail) string {
	parts := []string{}
	add := func(label, value string) {
		if strings.TrimSpace(value) != "" {
			parts = append(parts, label+" "+strings.TrimSpace(value))
		}
	}
	add("Pitch", d.Pitch)
	add("Cost", d.Cost)
	add("Attack", d.Power)
	add("Defense", d.Defense)
	add("Health", d.Health)
	add("Intellect", d.Intelligence)
	add("Arcane", d.Arcane)
	if len(d.Keywords) > 0 {
		parts = append(parts, "Keywords "+strings.Join(d.Keywords, ", "))
	}
	return strings.Join(parts, "\n")
}

func renderInlineStats(d *cardsdb.CardDetail, includeCostPitch bool) string {
	parts := []string{}
	add := func(label, value string) {
		if strings.TrimSpace(value) != "" {
			parts = append(parts, label+" "+strings.TrimSpace(value))
		}
	}
	if includeCostPitch {
		add("Cost", d.Cost)
		add("Pitch", d.Pitch)
	}
	add("Attack", d.Power)
	add("Defense", d.Defense)
	add("Health", d.Health)
	add("Intellect", d.Intelligence)
	add("Arcane", d.Arcane)
	if len(d.Keywords) > 0 {
		parts = append(parts, "Keywords "+strings.Join(d.Keywords, ", "))
	}
	return strings.Join(parts, "  ")
}

func renderRulesText(text string) string {
	replacer := strings.NewReplacer(
		"{br}", "\n",
		"{p}", "Power",
		"{i}", "Intellect",
		"{r}", "Resource",
		"{d}", "Defense",
		"**", "",
	)
	return strings.TrimSpace(replacer.Replace(text))
}

func renderLegalityGrid(raw string) string {
	entries := legalityEntries(raw)
	cells := make([]string, 0, len(entries))
	for _, entry := range entries {
		cells = append(cells, renderLegalityCell(entry.label, entry.status))
	}
	lines := []string{}
	for i := 0; i < len(cells); i += 2 {
		left := cells[i]
		right := ""
		if i+1 < len(cells) {
			right = cells[i+1]
		}
		lines = append(lines, fmt.Sprintf("%-36s %s", left, right))
	}
	return strings.Join(lines, "\n")
}

func renderLegalityColumn(raw string) string {
	entries := legalityEntries(raw)
	if len(entries) == 0 {
		return ""
	}
	lines := []string{detailSectionStyle.Render("Legality")}
	for _, entry := range entries {
		lines = append(lines, renderLegalityCell(entry.label, entry.status))
	}
	return strings.Join(lines, "\n")
}

func legalityEntries(raw string) []struct {
	label  string
	status string
} {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var data map[string]any
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return []struct {
			label  string
			status string
		}{{label: raw, status: ""}}
	}
	entries := []struct {
		keys  []string
		label string
	}{
		{[]string{"classicConstructed", "Classic Constructed", "cc"}, "Classic Constructed"},
		{[]string{"blitz", "Blitz"}, "Blitz"},
		{[]string{"commoner", "Commoner"}, "Commoner"},
		{[]string{"ultimatePitFight", "Ultimate Pit Fight", "upf"}, "Ultimate Pit Fight"},
		{[]string{"livingLegend", "Living Legend", "ll"}, "Living Legend"},
		{[]string{"projectBlue", "Silver Age", "Project Blue", "pb"}, "Silver Age"},
	}
	out := make([]struct {
		label  string
		status string
	}, 0, len(entries))
	for _, entry := range entries {
		value, ok := lookupLegality(data, entry.keys...)
		if !ok {
			continue
		}
		status := legalityStatus(value)
		out = append(out, struct {
			label  string
			status string
		}{label: entry.label, status: status})
	}
	return out
}

func renderLegalityCell(format, status string) string {
	icon := "?"
	style := mutedStyle
	switch strings.ToLower(status) {
	case "legal":
		icon = "✓"
		style = legalStyle
	case "not legal":
		icon = "○"
	case "banned":
		icon = "!"
		style = bannedStyle
	case "suspended", "restricted":
		icon = "!"
		style = warningStyle
	case "living legend":
		icon = "★"
		style = warningStyle
	}
	return style.Render(icon) + " " + format + mutedStyle.Render("  "+status)
}

func renderPriceRangeControls(active priceRange) string {
	parts := []string{}
	for _, option := range priceRangeOptions() {
		label := option.key + " " + option.label
		if option.value == active {
			parts = append(parts, detailTabActiveStyle.Render("["+label+"]"))
		} else {
			parts = append(parts, detailTabStyle.Render(label))
		}
	}
	return strings.Join(parts, "  ")
}

func renderPriceMessageChart(width, height int, message string) string {
	width = max(24, width)
	height = max(4, height)
	message = strings.TrimSpace(message)
	if message == "" {
		message = noPriceRecordedMessage
	}
	lines := make([]string, 0, height+2)
	messageRow := height / 2
	for row := 0; row < height; row++ {
		if row == messageRow {
			lines = append(lines, mutedStyle.Render(centerLine(message, width)))
			continue
		}
		lines = append(lines, strings.Repeat(" ", width))
	}
	return strings.Join(lines, "\n")
}

func renderPriceHistoryChart(entries []clientv1.PriceEntry, width, height int) string {
	width = max(24, width)
	height = max(4, height)
	points := make([]priceChartPoint, 0, len(entries))
	for _, entry := range entries {
		if entry.Price > 0 {
			points = append(points, priceChartPoint{value: entry.Price, at: entry.RecordAt})
		}
	}
	yLabelW := 9
	graphW := max(12, width-yLabelW)
	if len(points) == 0 {
		lines := make([]string, 0, height+2)
		for i := 0; i < height; i++ {
			if i == height-1 {
				lines = append(lines, mutedStyle.Render(strings.Repeat(" ", yLabelW)+strings.Repeat("─", graphW)))
			} else {
				lines = append(lines, mutedStyle.Render(strings.Repeat(" ", width)))
			}
		}
		lines = append(lines, mutedStyle.Render(strings.Repeat(" ", yLabelW)+strings.Repeat("─", graphW)))
		return strings.Join(lines, "\n")
	}
	values := make([]float64, 0, len(points))
	for _, point := range points {
		values = append(values, point.value)
	}
	minV, maxV := minMaxFloat(values)
	grid := make([][]rune, height)
	for row := range grid {
		grid[row] = []rune(strings.Repeat(" ", graphW))
	}
	if maxV == minV {
		row := height / 2
		for col := 0; col < graphW; col++ {
			grid[row][col] = '─'
		}
	} else {
		for col := 0; col < graphW; col++ {
			source := col * len(points) / graphW
			value := points[source].value
			row := height - 1 - int((value-minV)*float64(height-1)/(maxV-minV))
			row = min(max(row, 0), height-1)
			grid[row][col] = '█'
		}
	}
	yLabels := priceYAxisLabels(minV, maxV, height)
	lines := make([]string, 0, height+2)
	for row := 0; row < height; row++ {
		label := strings.Repeat(" ", yLabelW)
		if value, ok := yLabels[row]; ok {
			label = fmt.Sprintf("%8s ", value)
		}
		lines = append(lines, mutedStyle.Render(label)+warningStyle.Render(string(grid[row])))
	}
	lines = append(lines, mutedStyle.Render(strings.Repeat(" ", yLabelW)+strings.Repeat("─", graphW)))
	lines = append(lines, mutedStyle.Render(strings.Repeat(" ", yLabelW)+renderPriceXAxis(points, graphW)))
	return strings.Join(lines, "\n")
}

func priceYAxisLabels(minV, maxV float64, height int) map[int]string {
	count := min(5, max(2, height/2+1))
	labels := make(map[int]string, count)
	if count == 1 || height <= 1 {
		labels[0] = formatCompactMoney(maxV)
		return labels
	}
	for i := 0; i < count; i++ {
		row := i * (height - 1) / (count - 1)
		value := maxV
		if maxV != minV {
			value = maxV - ((maxV - minV) * float64(row) / float64(height-1))
		}
		labels[row] = formatCompactMoney(value)
	}
	return labels
}

func renderPriceXAxis(points []priceChartPoint, width int) string {
	if len(points) == 0 || width <= 0 {
		return ""
	}
	count := 5
	if width < 60 {
		count = 4
	}
	if width < 42 {
		count = 3
	}
	if width < 24 {
		count = 2
	}
	line := []rune(strings.Repeat(" ", width))
	lastEnd := -1
	for i := 0; i < count; i++ {
		col := i * (width - 1) / max(1, count-1)
		idx := i * (len(points) - 1) / max(1, count-1)
		label := formatPriceAxisDate(points[idx].at)
		if label == "" {
			continue
		}
		start := col - len([]rune(label))/2
		start = min(max(start, 0), max(0, width-len([]rune(label))))
		if start <= lastEnd {
			continue
		}
		for offset, r := range label {
			if start+offset >= 0 && start+offset < len(line) {
				line[start+offset] = r
			}
		}
		lastEnd = start + len([]rune(label))
	}
	return strings.TrimRight(string(line), " ")
}

func formatPriceAxisDate(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format("Jan 2")
}

func isPriceNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *clientv1.APIError
	if errors.As(err, &apiErr) {
		return apiErr.Metadata.StatusCode == http.StatusNotFound
	}
	return strings.Contains(err.Error(), "status 404")
}

func priceForProduct(entries []clientv1.PriceEntry, productID string) (clientv1.PriceEntry, bool) {
	productID = strings.TrimSpace(productID)
	for _, entry := range entries {
		if strings.TrimSpace(entry.ProductID) == productID {
			return entry, true
		}
	}
	if len(entries) == 1 {
		return entries[0], true
	}
	return clientv1.PriceEntry{}, false
}

func renderCurrentPriceSummary(entry clientv1.PriceEntry, ok bool) string {
	if !ok {
		return mutedStyle.Render("Current price unavailable.")
	}
	parts := []string{}
	if entry.Price2 > 0 {
		parts = append(parts, "Low "+formatMoney(entry.Price2, entry.Currency))
	}
	if entry.Price > 0 {
		parts = append(parts, "Market "+formatMoney(entry.Price, entry.Currency))
	}
	if entry.Price3 > 0 {
		parts = append(parts, "High "+formatMoney(entry.Price3, entry.Currency))
	}
	if len(parts) == 0 {
		parts = append(parts, "Price unavailable")
	}
	source := strings.TrimSpace(entry.Source)
	if source != "" {
		parts = append(parts, source)
	}
	if entry.RecordAt != nil {
		parts = append(parts, entry.RecordAt.Format("2006-01-02"))
	}
	return strings.Join(parts, "  ")
}

func renderPriceHistoryRange(entries []clientv1.PriceEntry) string {
	if len(entries) == 0 {
		return ""
	}
	first := entries[0].RecordAt
	last := entries[len(entries)-1].RecordAt
	if first == nil || last == nil {
		return fmt.Sprintf("%d price history points", len(entries))
	}
	return fmt.Sprintf("%d price history points  %s to %s", len(entries), first.Format("2006-01-02"), last.Format("2006-01-02"))
}

func formatMoney(value float64, currency string) string {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "" || currency == "USD" {
		return fmt.Sprintf("$%.2f", value)
	}
	return fmt.Sprintf("%.2f %s", value, currency)
}

type priceRangeOption struct {
	key   string
	label string
	value priceRange
	limit int32
	start func(time.Time) time.Time
}

type priceChartPoint struct {
	value float64
	at    *time.Time
}

func priceRangeOptions() []priceRangeOption {
	return []priceRangeOption{
		{key: "4", label: "1M", value: priceRange1M, limit: 45, start: func(now time.Time) time.Time { return now.AddDate(0, -1, 0) }},
		{key: "5", label: "3M", value: priceRange3M, limit: 90, start: func(now time.Time) time.Time { return now.AddDate(0, -3, 0) }},
		{key: "6", label: "6M", value: priceRange6M, limit: 180, start: func(now time.Time) time.Time { return now.AddDate(0, -6, 0) }},
		{key: "7", label: "1Y", value: priceRange1Y, limit: 365, start: func(now time.Time) time.Time { return now.AddDate(-1, 0, 0) }},
	}
}

func priceRangeConfig(value priceRange) priceRangeOption {
	for _, option := range priceRangeOptions() {
		if option.value == value {
			return option
		}
	}
	return priceRangeOptions()[1]
}

func priceRangeFromKey(keyText string) priceRange {
	for _, option := range priceRangeOptions() {
		if option.key == keyText {
			return option.value
		}
	}
	return ""
}

func minMaxFloat(points []float64) (float64, float64) {
	minV, maxV := points[0], points[0]
	for _, point := range points {
		if point < minV {
			minV = point
		}
		if point > maxV {
			maxV = point
		}
	}
	return minV, maxV
}

func formatCompactMoney(value float64) string {
	if value >= 1000 {
		return fmt.Sprintf("$%.1fk", value/1000)
	}
	return fmt.Sprintf("$%.2f", value)
}

func lookupLegality(data map[string]any, keys ...string) (any, bool) {
	normalized := map[string]any{}
	for key, value := range data {
		normalized[normalizeLegalityKey(key)] = value
	}
	for _, key := range keys {
		value, ok := normalized[normalizeLegalityKey(key)]
		if ok {
			return value, true
		}
	}
	return nil, false
}

func normalizeLegalityKey(key string) string {
	key = strings.ToLower(strings.TrimSpace(key))
	key = strings.NewReplacer(" ", "", "_", "", "-", "").Replace(key)
	return key
}

func legalityStatus(value any) string {
	entry, ok := value.(map[string]any)
	if !ok {
		return titleStatus(fmt.Sprint(value))
	}
	for _, key := range []string{"status", "legality"} {
		if raw, ok := entry[key]; ok {
			return titleStatus(fmt.Sprint(raw))
		}
	}
	if boolField(entry, "banned") {
		return "Banned"
	}
	if boolField(entry, "suspended") {
		return "Suspended"
	}
	if boolField(entry, "restricted") {
		return "Restricted"
	}
	if boolField(entry, "isLivingLegend") || boolField(entry, "livingLegend") {
		return "Living Legend"
	}
	if raw, ok := entry["isLegal"]; ok {
		if b, ok := raw.(bool); ok && b {
			return "Legal"
		}
		return "Not Legal"
	}
	return "Unknown"
}

func boolField(entry map[string]any, key string) bool {
	raw, ok := entry[key]
	if !ok {
		return false
	}
	switch value := raw.(type) {
	case bool:
		return value
	case float64:
		return value != 0
	case string:
		value = strings.ToLower(strings.TrimSpace(value))
		return value == "true" || value == "1" || value == "yes"
	default:
		return false
	}
}

func titleStatus(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Unknown"
	}
	value = strings.NewReplacer("_", " ", "-", " ").Replace(value)
	words := strings.Fields(strings.ToLower(value))
	for i, word := range words {
		words[i] = strings.ToUpper(word[:1]) + word[1:]
	}
	return strings.Join(words, " ")
}

func formatStats(cost, pitch, power, defense, health string) string {
	parts := []string{}
	add := func(label, value string) {
		if strings.TrimSpace(value) != "" {
			parts = append(parts, label+" "+strings.TrimSpace(value))
		}
	}
	add("Cost", cost)
	add("Pitch", pitch)
	add("Power", power)
	add("Defense", defense)
	add("Health", health)
	return strings.Join(parts, "  ")
}

func nonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func centerLine(value string, width int) string {
	value = oneLine(value)
	valueWidth := lipgloss.Width(value)
	if valueWidth >= width {
		return value
	}
	left := (width - valueWidth) / 2
	right := width - valueWidth - left
	return strings.Repeat(" ", left) + value + strings.Repeat(" ", right)
}

func oneLine(value string) string {
	return strings.TrimSpace(strings.NewReplacer("\r", " ", "\n", " ", "\t", " ").Replace(value))
}

func syntaxModal(width int) string {
	content := strings.Join([]string{
		sectionLabelStyle.Render("Search Syntax"),
		"",
		"Type plain text to search card names, ids, and rules text.",
		"",
		"Examples:",
		"  fai",
		"  red pitch",
		"  arcane barrier",
		"  MST131",
		"",
		"Navigation:",
		"  Tab      switch between search and card rows",
		"  /        focus search from card rows",
		"  Esc      focus card rows",
		"  Up/Down  move through card rows",
		"  Enter    select highlighted card",
		"  1/2/3    switch details, related, and prices tabs",
		"  4/5/6/7  switch price ranges on the prices tab",
		"  t        swap true rules text and printed text",
		"",
		"Press ?, Esc, Enter, or q to close this popup.",
	}, "\n")
	panelWidth := min(max(48, width-8), 78)
	if width <= 0 {
		panelWidth = 72
	}
	return modalStyle.Width(panelWidth).Render(content)
}

func overlayModal(width, height int, base, modal string) string {
	if width <= 0 {
		width = lipgloss.Width(base)
	}
	if height <= 0 {
		height = lipgloss.Height(base)
	}
	baseLines := strings.Split(base, "\n")
	for len(baseLines) < height {
		baseLines = append(baseLines, "")
	}
	if len(baseLines) > height {
		baseLines = baseLines[:height]
	}
	modalLines := strings.Split(modal, "\n")
	top := max(0, (height-len(modalLines))/2)
	for i, line := range modalLines {
		row := top + i
		if row >= len(baseLines) {
			break
		}
		baseLines[row] = lipgloss.PlaceHorizontal(width, lipgloss.Center, line)
	}
	return strings.Join(baseLines, "\n")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func wrapIndex(index, length int) int {
	if length <= 0 {
		return 0
	}
	index %= length
	if index < 0 {
		index += length
	}
	return index
}

func clampIndex(index, length int) int {
	if length <= 0 || index < 0 {
		return 0
	}
	if index >= length {
		return length - 1
	}
	return index
}

func printingLanguages(printings []cardsdb.CardPrinting) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, printing := range printings {
		lang := normalizeLanguage(printing.Language)
		if lang == "" || seen[lang] {
			continue
		}
		seen[lang] = true
		out = append(out, lang)
	}
	return out
}

func defaultPrintingLanguage(langs []string) string {
	if len(langs) == 0 {
		return ""
	}
	for _, lang := range langs {
		if lang == "en" {
			return lang
		}
	}
	return langs[0]
}

func normalizeLanguage(language string) string {
	language = strings.ToLower(strings.TrimSpace(language))
	if language == "" {
		return ""
	}
	if len(language) > 2 {
		return language[:2]
	}
	return language
}

func uniqueValues(values []string, limit int) []string {
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

var (
	titleStyle           = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#f8f8f2")).Background(lipgloss.Color("#2f5d50")).Padding(0, 1)
	inputStyle           = lipgloss.NewStyle().Padding(0, 1)
	panelStyle           = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#3f6f66")).Padding(0, 1)
	activePanelStyle     = panelStyle.BorderForeground(lipgloss.Color("#f2d16b"))
	sectionLabelStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#f2d16b"))
	modalStyle           = lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).BorderForeground(lipgloss.Color("#f2d16b")).Padding(1, 2)
	mutedStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("#8a9a96"))
	errorStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff6b6b"))
	legalStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("#10b981"))
	warningStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("#f2d16b"))
	bannedStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff6b6b"))
	detailTitleStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#f2d16b"))
	detailSectionStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#8dd7c7"))
	detailTabStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#8a9a96"))
	detailTabActiveStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#f8f8f2")).Background(lipgloss.Color("#3f6f66")).Padding(0, 1)
	listTitleStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#f8f8f2"))
	listMetaStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#8a9a96"))

	listSelectedTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#e778ff"))
	listSelectedMetaStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#d783e8"))
)
