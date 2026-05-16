package decks

import (
	"context"
	"database/sql"
	"fmt"
	"io"
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
	"github.com/pitchstack-gg/pitchstack-cli/internal/powersync"
	cardstui "github.com/pitchstack-gg/pitchstack-cli/internal/tui/cards"
	clientv1 "github.com/pitchstack-gg/pitchstack-go/client/v1"
)

type ProfileClient interface {
	GetProfile(context.Context, *clientv1.GetProfileRequest, ...clientv1.RequestOpt) (*clientv1.GetProfileResponse, error)
}

type Options struct {
	Store         *powersync.Store
	Client        *powersync.Client
	ProfileClient ProfileClient
	ViewerUserID  string
	ViewerName    string
	MissingDB     bool
	AutoSync      bool
	InitialScope  powersync.DeckListScope
	Limit         int
	CardsRepo     *cardsdb.Repository
	ImageCacheDir string
}

type Model struct {
	options Options

	input   textinput.Model
	results list.Model
	detail  viewport.Model
	art     viewport.Model
	preview viewport.Model
	spinner spinner.Model
	help    help.Model

	width          int
	height         int
	focus          focusArea
	scope          powersync.DeckListScope
	decks          []powersync.DeckSummary
	details        *powersync.DeckDetails
	cardLookup     map[string]*cardsdb.CardDetail
	ownerLookup    map[string]string
	cardArt        string
	artCardID      string
	artWidth       int
	artHeight      int
	heroArt        string
	heroArtID      string
	heroArtW       int
	heroArtH       int
	status         *powersync.Status
	err            string
	syncErr        string
	loading        bool
	syncing        bool
	loadingArt     bool
	loadingHeroArt bool
	syncOnLoad     bool
	mode           deckMode
	deckID         string
	versionID      string
	cardIndex      int
}

type focusArea string
type deckMode string

const (
	focusSearch focusArea = "search"
	focusList   focusArea = "list"

	modeList   deckMode = "list"
	modeDetail deckMode = "detail"
)

type deckItem struct {
	powersync.DeckSummary
}

func (i deckItem) Title() string {
	return nonEmpty(i.Name, i.ID)
}

func (i deckItem) Description() string {
	parts := []string{}
	if i.HeroID != "" {
		parts = append(parts, prettyCardID(i.HeroID))
	}
	if i.Format != "" {
		parts = append(parts, prettyFormatName(i.Format))
	}
	if i.Author != "" && !looksLikeOpaqueID(i.Author) {
		parts = append(parts, "by "+i.Author)
	}
	if i.Ownership != "" {
		parts = append(parts, i.Ownership)
	}
	return strings.Join(parts, "  ")
}

func (i deckItem) FilterValue() string {
	return strings.Join(nonEmptyValues(i.Name, i.Author, i.HeroID, i.Format, i.ID), " ")
}

type deckDelegate struct{}

func (d deckDelegate) Height() int  { return 3 }
func (d deckDelegate) Spacing() int { return 1 }
func (d deckDelegate) Update(tea.Msg, *list.Model) tea.Cmd {
	return nil
}

func (d deckDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	deck, ok := item.(deckItem)
	if !ok {
		return
	}
	selected := index == m.Index() && m.FilterState() != list.Filtering
	prefix := "  "
	titleStyle := listTitleStyle
	metaStyle := listMetaStyle
	if selected {
		prefix = "| "
		titleStyle = listSelectedTitleStyle
		metaStyle = listSelectedMetaStyle
	}
	meta := []string{}
	if deck.HeroID != "" {
		meta = append(meta, prettyCardID(deck.HeroID))
	}
	if deck.Format != "" {
		meta = append(meta, prettyFormatName(deck.Format))
	}
	if deck.Author != "" && !looksLikeOpaqueID(deck.Author) {
		meta = append(meta, "by "+deck.Author)
	}
	if deck.Visibility != "" {
		meta = append(meta, prettyVisibility(deck.Visibility))
	}
	counts := fmt.Sprintf("%d cards", deck.TotalQuantity)
	if deck.CardRowCount != deck.TotalQuantity {
		counts = fmt.Sprintf("%d cards / %d rows", deck.TotalQuantity, deck.CardRowCount)
	}
	version := nonEmpty(deck.SelectedVersionName, deck.SelectedVersionID, deck.ActiveVersionName, deck.ActiveVersionID, "no version")
	updated := shortDate(nonEmpty(deck.UpdatedAt, deck.CreatedAt))
	lines := []string{
		titleStyle.Render(prefix + nonEmpty(deck.Name, deck.ID)),
		metaStyle.Render(prefix + oneLine(strings.Join(meta, "  "))),
		metaStyle.Render(prefix + oneLine(strings.Join(nonEmptyValues(deck.Ownership, version, counts, updated), "  "))),
	}
	_, _ = fmt.Fprint(w, strings.Join(lines, "\n"))
}

type keyMap struct {
	Enter   key.Binding
	Back    key.Binding
	Focus   key.Binding
	Search  key.Binding
	Scope   key.Binding
	Version key.Binding
	Reload  key.Binding
	Quit    key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Search, k.Enter, k.Back, k.Scope, k.Version, k.Reload, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Search, k.Enter, k.Back, k.Scope, k.Version, k.Reload, k.Quit}}
}

var keys = keyMap{
	Enter:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
	Back:    key.NewBinding(key.WithKeys("esc", "b"), key.WithHelp("esc/b", "back")),
	Focus:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "list")),
	Search:  key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
	Scope:   key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "scope")),
	Version: key.NewBinding(key.WithKeys("[", "]", "v"), key.WithHelp("[/]/v", "version")),
	Reload:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload")),
	Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

type loadDoneMsg struct {
	decks   []powersync.DeckSummary
	owners  map[string]string
	status  *powersync.Status
	synced  bool
	syncErr string
	err     error
}

type deckDetailsDoneMsg struct {
	details *powersync.DeckDetails
	cards   map[string]*cardsdb.CardDetail
	owners  map[string]string
	err     error
}

type deckArtDoneMsg struct {
	cardID string
	art    string
	width  int
	height int
	err    error
}

type deckHeroArtDoneMsg struct {
	cardID string
	art    string
	width  int
	height int
	err    error
}

func New(opts Options) Model {
	input := textinput.New()
	input.Placeholder = "Search local decks"
	input.Prompt = "Search*: "
	input.CharLimit = 120
	input.SetWidth(64)
	_ = input.Focus()

	results := list.New(nil, deckDelegate{}, 40, 12)
	results.Title = "Decks"
	results.SetShowTitle(false)
	results.SetShowFilter(false)
	results.SetShowHelp(false)
	results.SetShowStatusBar(false)
	results.SetShowPagination(false)
	results.DisableQuitKeybindings()

	detail := viewport.New()
	detail.SoftWrap = true
	detail.SetContent("Select a deck to view details.")
	art := viewport.New()
	art.SoftWrap = false
	art.SetContent("Select a card.")
	preview := viewport.New()
	preview.SoftWrap = true
	preview.SetContent("Select a card row.")

	scope := opts.InitialScope
	if scope == "" {
		scope = powersync.DeckListScopeAccessible
	}
	spin := spinner.New(spinner.WithSpinner(spinner.Line))
	return Model{
		options:     opts,
		input:       input,
		results:     results,
		detail:      detail,
		art:         art,
		preview:     preview,
		spinner:     spin,
		help:        help.New(),
		focus:       focusSearch,
		scope:       scope,
		ownerLookup: initialOwnerLookup(opts),
		loading:     true,
		syncing:     opts.AutoSync,
		syncOnLoad:  opts.AutoSync,
		mode:        modeList,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.input.Focus(), m.spinner.Tick, m.loadCmd())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
		if m.mode == modeDetail && m.details != nil {
			cmds = append(cmds, m.renderHeroBannerArtCmd(), m.renderSelectedCardArtCmd())
		}
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.focus == focusList || m.mode == modeDetail {
				return m, tea.Quit
			}
		case "esc", "b":
			if m.mode == modeDetail {
				m.mode = modeList
				m.err = ""
				m.resize()
				m.refreshDetailContent()
				return m, tea.Batch(cmds...)
			}
			if msg.String() == "esc" {
				m.focus = focusList
				m.input.Prompt = "Search: "
				m.input.Blur()
				return m, tea.Batch(cmds...)
			}
		case "/":
			if m.mode == modeDetail {
				break
			}
			m.focus = focusSearch
			m.input.Prompt = "Search*: "
			cmds = append(cmds, m.input.Focus())
			return m, tea.Batch(cmds...)
		case "down":
			if m.mode == modeDetail {
				m.moveCardSelection(1)
				m.refreshDetailContent()
				cmds = append(cmds, m.renderSelectedCardArtCmd())
				return m, tea.Batch(cmds...)
			}
			if m.focus == focusSearch {
				m.focus = focusList
				m.input.Prompt = "Search: "
				m.input.Blur()
			}
		case "up":
			if m.mode == modeDetail {
				m.moveCardSelection(-1)
				m.refreshDetailContent()
				cmds = append(cmds, m.renderSelectedCardArtCmd())
				return m, tea.Batch(cmds...)
			}
		case "o":
			if m.mode == modeList && m.focus == focusList {
				m.scope = nextScope(m.scope)
				m.loading = true
				cmds = append(cmds, m.loadCmd(), m.spinner.Tick)
				return m, tea.Batch(cmds...)
			}
		case "r":
			m.loading = true
			if m.options.AutoSync {
				m.syncOnLoad = true
				m.syncing = true
			}
			cmds = append(cmds, m.loadCmd(), m.spinner.Tick)
			return m, tea.Batch(cmds...)
		case "[", "]", "v":
			if m.mode == modeDetail && m.details != nil && len(m.details.Versions) > 1 {
				m.loading = true
				nextID := m.nextVersionID(msg.String())
				cmds = append(cmds, m.loadDeckDetailsCmd(m.deckID, nextID), m.spinner.Tick)
				return m, tea.Batch(cmds...)
			}
		case "enter":
			if m.mode == modeDetail {
				break
			}
			if m.focus == focusSearch {
				m.focus = focusList
				m.input.Prompt = "Search: "
				m.input.Blur()
				return m, tea.Batch(cmds...)
			}
			if item, ok := m.results.SelectedItem().(deckItem); ok {
				m.mode = modeDetail
				m.deckID = item.ID
				m.versionID = item.SelectedVersionID
				m.cardIndex = 0
				m.loading = true
				m.resize()
				m.detail.SetContent("Loading deck...")
				cmds = append(cmds, m.loadDeckDetailsCmd(m.deckID, m.versionID), m.spinner.Tick)
				return m, tea.Batch(cmds...)
			}
		}
	case loadDoneMsg:
		m.loading = false
		m.syncing = false
		m.syncErr = msg.syncErr
		if msg.synced {
			m.syncOnLoad = false
			m.options.MissingDB = false
		}
		if msg.err != nil {
			m.err = msg.err.Error()
			m.decks = nil
			cmds = append(cmds, m.results.SetItems(nil))
			m.detail.SetContent("Could not load local decks.")
			break
		}
		m.err = ""
		m.mergeOwnerLookup(msg.owners)
		m.decks = m.withResolvedDeckOwners(msg.decks)
		m.status = msg.status
		items := make([]list.Item, 0, len(m.decks))
		for _, deck := range m.decks {
			items = append(items, deckItem{DeckSummary: deck})
		}
		cmds = append(cmds, m.results.SetItems(items))
		if len(items) > 0 {
			m.results.Select(0)
		}
		if m.mode == modeDetail && m.deckID != "" {
			cmds = append(cmds, m.loadDeckDetailsCmd(m.deckID, m.versionID))
		} else {
			m.refreshDetailContent()
		}
	case deckDetailsDoneMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			m.details = nil
			m.cardLookup = nil
			m.detail.SetContent("Could not load deck details.")
			break
		}
		m.err = ""
		m.mergeOwnerLookup(msg.owners)
		m.details = msg.details
		if m.details != nil {
			m.details.Deck = m.withResolvedDeckOwner(m.details.Deck)
		}
		m.cardLookup = msg.cards
		m.cardArt = ""
		m.artCardID = ""
		m.artWidth = 0
		m.artHeight = 0
		m.heroArt = ""
		m.heroArtID = ""
		m.heroArtW = 0
		m.heroArtH = 0
		if msg.details != nil {
			m.deckID = msg.details.Deck.ID
			m.versionID = msg.details.SelectedVersionID
		}
		m.resize()
		m.clampCardSelection()
		m.refreshDetailContent()
		cmds = append(cmds, m.renderHeroBannerArtCmd(), m.renderSelectedCardArtCmd())
	case deckArtDoneMsg:
		m.loadingArt = false
		if msg.cardID != "" && msg.cardID != m.selectedCardID() {
			break
		}
		if msg.err != nil {
			m.cardArt = errorStyle.Render("image failed: " + msg.err.Error())
		} else {
			m.cardArt = msg.art
			m.artWidth = msg.width
			m.artHeight = msg.height
		}
		if strings.TrimSpace(m.cardArt) == "" {
			m.cardArt = "No image available for this card."
		}
		m.art.SetContent(m.cardArt)
	case deckHeroArtDoneMsg:
		m.loadingHeroArt = false
		if msg.cardID != "" && m.details != nil && msg.cardID != strings.TrimSpace(m.details.Deck.HeroID) {
			break
		}
		if msg.err != nil {
			m.heroArt = ""
		} else {
			m.heroArt = msg.art
			m.heroArtW = msg.width
			m.heroArtH = msg.height
		}
	}

	if m.loading || m.syncing || m.loadingArt || m.loadingHeroArt {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.mode == modeList && m.focus == focusSearch {
		old := m.input.Value()
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
		if m.input.Value() != old {
			m.loading = true
			cmds = append(cmds, m.loadCmd(), m.spinner.Tick)
		}
	} else if m.mode == modeList {
		var cmd tea.Cmd
		m.results, cmd = m.results.Update(msg)
		cmds = append(cmds, cmd)
		m.refreshDetailContent()
	}
	var detailCmd tea.Cmd
	m.detail, detailCmd = m.detail.Update(msg)
	cmds = append(cmds, detailCmd)
	var artCmd tea.Cmd
	m.art, artCmd = m.art.Update(msg)
	cmds = append(cmds, artCmd)
	var previewCmd tea.Cmd
	m.preview, previewCmd = m.preview.Update(msg)
	cmds = append(cmds, previewCmd)
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
	headerH := 4
	if m.err != "" {
		headerH++
	}
	footerH := 2
	bodyH := max(6, m.height-headerH-footerH)
	chromeH := panelStyle.GetVerticalFrameSize()
	chromeW := panelStyle.GetHorizontalFrameSize()
	if m.mode == modeDetail {
		if m.width >= 100 {
			boardW, sidebarW := m.detailColumnWidths()
			m.detail.SetWidth(max(24, boardW-chromeW))
			m.art.SetWidth(max(20, sidebarW-chromeW))
			m.preview.SetWidth(max(20, sidebarW-chromeW))
		} else {
			m.detail.SetWidth(max(24, m.width-chromeW))
			m.art.SetWidth(max(20, m.width-chromeW))
			m.preview.SetWidth(max(20, m.width-chromeW))
		}
		bannerOuterH := m.bannerOuterHeight()
		availableBodyH := max(10, bodyH-bannerOuterH)
		artOuterH := min(30, max(16, (availableBodyH*58)/100))
		m.detail.SetHeight(max(8, availableBodyH-chromeH))
		m.art.SetHeight(max(6, artOuterH-chromeH))
		m.preview.SetHeight(max(8, availableBodyH-artOuterH-chromeH))
	} else {
		m.results.SetSize(max(24, m.width-chromeW), max(5, bodyH-chromeH-1))
		m.detail.SetWidth(max(24, m.width-chromeW))
		m.detail.SetHeight(max(5, bodyH-chromeH))
	}
	m.input.SetWidth(max(20, m.width-10))
	m.help.SetWidth(m.width)
}

func (m Model) render() string {
	title := titleStyle.Render("Pitchstack")
	page := "Decks"
	if m.mode == modeDetail {
		page = "Decks / Inspect"
	}
	scope := sectionLabelStyle.Render(page) + " " + mutedStyle.Render("scope: "+string(m.scope))
	if m.syncing {
		scope += " " + m.spinner.View() + " " + mutedStyle.Render("syncing")
	} else if m.loading {
		scope += " " + m.spinner.View()
	}
	headerParts := []string{title, scope}
	if m.mode == modeList {
		search := sectionLabelStyle.Render("Search") + "\n" + inputStyle.Width(max(20, m.width)).Render(m.input.View())
		headerParts = append(headerParts, search)
	}
	header := lipgloss.JoinVertical(lipgloss.Left, headerParts...)
	if m.err != "" {
		header = lipgloss.JoinVertical(lipgloss.Left, header, errorStyle.Render(m.err))
	}
	if m.syncErr != "" {
		header = lipgloss.JoinVertical(lipgloss.Left, header, errorStyle.Render("sync failed: "+m.syncErr))
	}

	body := ""
	if m.mode == modeDetail {
		body = m.renderDetailMode()
	} else {
		listContent := m.emptyMessage()
		if len(m.decks) > 0 {
			listContent = m.results.View()
		}
		listPanel := panelStyle
		if m.focus == focusList {
			listPanel = activePanelStyle
		}
		body = renderPanel(listPanel, max(20, m.width-2), m.results.Height()+panelStyle.GetVerticalFrameSize()+1, sectionLabelStyle.Render("Local Decks")+"\n"+listContent)
	}

	footer := m.help.ShortHelpView(keys.ShortHelp())
	status := mutedStyle.Render(m.statusLine())
	return lipgloss.JoinVertical(lipgloss.Left, header, body, status, mutedStyle.Render(footer))
}

func (m Model) emptyMessage() string {
	if m.loading {
		return "Loading local decks..."
	}
	if m.options.MissingDB {
		if !m.options.AutoSync {
			return "No local PowerSync database found. Log in, then open Decks again to initialize and pull local sync."
		}
		return "No local PowerSync database found. Run `pitchstack sync local init && pitchstack sync local pull`."
	}
	if strings.TrimSpace(m.input.Value()) != "" {
		return "No decks match your search."
	}
	switch m.scope {
	case powersync.DeckListScopeOwned:
		return "No owned decks in local sync."
	case powersync.DeckListScopeShared:
		return "No shared decks in local sync."
	default:
		return "No local decks synced yet. Run `pitchstack sync local pull`."
	}
}

func (m Model) statusLine() string {
	if m.status == nil {
		return "local PowerSync cache"
	}
	parts := []string{fmt.Sprintf("%d synced rows", m.status.Rows)}
	if m.status.LastSuccessfulSync != nil {
		parts = append(parts, "synced "+m.status.LastSuccessfulSync.Local().Format("Jan 2 15:04"))
	}
	if m.status.PendingCrud > 0 {
		parts = append(parts, fmt.Sprintf("%d pending writes", m.status.PendingCrud))
	}
	if m.status.FailedCrud > 0 {
		parts = append(parts, fmt.Sprintf("%d failed writes", m.status.FailedCrud))
	}
	return strings.Join(parts, "  ")
}

func (m *Model) refreshDetailContent() {
	if m.mode != modeDetail {
		m.detail.SetContent("Select a deck to inspect.")
		return
	}
	if m.details == nil {
		m.detail.SetContent("Loading deck...")
		return
	}
	m.detail.SetContent(m.renderBoardSections())
	if line, ok := m.selectedCardLine(); ok {
		m.detail.EnsureVisible(line, 0, 0)
	}
}

func (m Model) renderDetailMode() string {
	if m.details == nil {
		return renderPanel(activePanelStyle, max(20, m.width-2), max(10, m.height-8), "Loading deck...")
	}
	banner := m.renderDeckBanner()
	boardW, sidebarW := m.detailColumnWidths()
	boardPanel := renderPanel(activePanelStyle, boardW, m.detail.Height()+panelStyle.GetVerticalFrameSize(), m.detail.View())
	m.preview.SetContent(m.renderCardPreview())
	m.preview.GotoTop()
	sidebar := lipgloss.JoinVertical(lipgloss.Left,
		renderPanel(panelStyle, sidebarW, m.art.Height()+panelStyle.GetVerticalFrameSize(), m.renderArtPanel()),
		renderPanel(panelStyle, sidebarW, m.preview.Height()+panelStyle.GetVerticalFrameSize(), m.preview.View()),
	)
	if m.width >= 100 {
		return lipgloss.JoinVertical(lipgloss.Left, banner, lipgloss.JoinHorizontal(lipgloss.Top, boardPanel, sidebar))
	}
	return lipgloss.JoinVertical(lipgloss.Left, banner, boardPanel, sidebar)
}

func (m Model) detailColumnWidths() (int, int) {
	if m.width < 100 {
		full := max(24, m.width-panelStyle.GetHorizontalFrameSize())
		return full, full
	}
	gapW := 0
	available := max(80, m.width-gapW-2)
	boardW := min(96, max(64, (available*48)/100))
	sidebarW := max(34, available-boardW)
	if boardW+sidebarW > available {
		sidebarW = max(34, available-boardW)
	}
	return boardW, sidebarW
}

func (m Model) bannerOuterHeight() int {
	if m.width >= 100 {
		return 14
	}
	return 8
}

func (m Model) heroBannerArtSize() (int, int) {
	if m.width < 100 {
		return 0, 0
	}
	contentW := max(20, m.width-bannerPanelStyle.GetHorizontalFrameSize()-2)
	heroW := min(88, max(44, (contentW*38)/100))
	heroH := 10
	return heroW, heroH
}

func (m Model) renderDeckBanner() string {
	d := m.details
	if d == nil {
		return ""
	}
	deck := d.Deck
	version := nonEmpty(d.SelectedVersionName, d.SelectedVersionID, "no version")
	owner := m.deckOwnerName(deck)
	bg := m.bannerBackgroundColor()
	fg := readableTextColor(bg)
	contentW := max(20, m.width-bannerPanelStyle.GetHorizontalFrameSize()-2)
	heroW, heroH := m.heroBannerArtSize()
	topH := heroH
	if topH <= 0 {
		topH = max(5, m.bannerOuterHeight()-bannerPanelStyle.GetVerticalFrameSize()-1)
	}
	leftW := contentW
	if heroW > 0 {
		leftW = max(28, contentW-heroW-2)
	}
	title := nonEmpty(deck.Name, deck.ID)
	metaStyle := bannerMetaStyle.Foreground(lipgloss.Color(fg))
	mutedBannerStyle := mutedStyle.Foreground(lipgloss.Color(fg))
	meta := []string{renderLargeDeckTitle(title, leftW, fg)}
	if owner != "" {
		meta = append(meta, metaStyle.Render("by "+owner))
	}
	meta = append(meta,
		"",
		"",
		metaStyle.Render(strings.Join(nonEmptyValues(m.heroDisplayName(deck.HeroID), prettyFormatName(deck.Format)), "  -  ")),
	)
	left := lipgloss.NewStyle().
		Width(leftW).
		Height(topH).
		Background(lipgloss.Color(bg)).
		Render(strings.Join(meta, "\n"))
	content := lipgloss.NewStyle().
		Width(contentW).
		Background(lipgloss.Color(bg)).
		Render(left)
	if heroW > 0 {
		spacer := lipgloss.NewStyle().Width(2).Height(topH).Background(lipgloss.Color(bg)).Render("")
		content = lipgloss.JoinHorizontal(lipgloss.Top, left, spacer, m.renderBannerHeroArt(heroW, heroH))
	}
	stats := lipgloss.NewStyle().
		Width(contentW).
		Background(lipgloss.Color(bg)).
		Render(renderDeckStats(d))
	bottomLeft := strings.Join(nonEmptyValues("version: "+version, fmt.Sprintf("%d versions", len(d.Versions)), prettyVisibility(deck.Visibility)), "  ")
	bottomRight := ""
	if updated := shortDateTime(nonEmpty(deck.UpdatedAt, deck.CreatedAt)); updated != "" {
		bottomRight = "Last updated: " + updated
	}
	bottomW := max(20, contentW-8)
	bottom := mutedBannerStyle.
		Width(contentW).
		Background(lipgloss.Color(bg)).
		Render(splitLine(bottomLeft, bottomRight, bottomW))
	content = lipgloss.JoinVertical(lipgloss.Left, content, stats, bottom)
	return bannerPanelStyle.
		Width(contentW).
		Height(m.bannerOuterHeight() - bannerPanelStyle.GetVerticalFrameSize()).
		Background(lipgloss.Color(bg)).
		BorderForeground(lipgloss.Color(bannerBorderColor(bg))).
		Render(content)
}

func (m Model) renderBannerHeroArt(width, height int) string {
	bg := m.bannerBackgroundColor()
	if m.loadingHeroArt {
		return bannerHeroArtStyle.Width(width).Height(height).Background(lipgloss.Color(bg)).Render(m.spinner.View() + " " + mutedStyle.Render("rendering hero"))
	}
	art := strings.TrimSpace(m.heroArt)
	if art == "" {
		art = renderDeckAccent(m.bannerSeed(), width)
	} else {
		art = paintTransparentArtCells(art, bg)
	}
	return bannerHeroArtStyle.
		Width(width).
		Height(height).
		Background(lipgloss.Color(bg)).
		Render(art)
}

func (m Model) bannerBackgroundColor() string {
	if m.details == nil {
		return "#1f2a2e"
	}
	if detail := m.cardLookup[strings.TrimSpace(m.details.Deck.HeroID)]; detail != nil {
		if color := normalizeHexColor(detail.PrimaryColor); color != "" {
			return color
		}
	}
	return "#1f2a2e"
}

func (m Model) renderBoardSections() string {
	if m.details == nil {
		return "Loading deck..."
	}
	sections := m.displaySections()
	lines := []string{}
	for _, section := range sections {
		if len(section.cards) == 0 && section.title != "Hero / Equipment" {
			continue
		}
		count := int64(0)
		for _, card := range section.cards {
			count += displayQuantity(card.Quantity)
		}
		lines = append(lines, sectionLabelStyle.Render(fmt.Sprintf("%s  %d", section.title, count)))
		if len(section.cards) == 0 {
			if m.details.Deck.HeroID != "" {
				lines = append(lines, mutedStyle.Render("  "+m.details.Deck.HeroID))
			} else {
				lines = append(lines, mutedStyle.Render("  no local cards"))
			}
		}
		for _, card := range section.cards {
			lines = append(lines, m.renderCardRow(card))
		}
		lines = append(lines, "")
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}

type deckDisplaySection struct {
	title string
	cards []powersync.DeckCardLine
}

func (m Model) displaySections() []deckDisplaySection {
	if m.details == nil {
		return nil
	}
	hero := append([]powersync.DeckCardLine{}, m.details.HeroEquipment...)
	main := append([]powersync.DeckCardLine{}, m.details.Mainboard...)
	if len(hero) == 0 {
		remaining := main[:0]
		for _, card := range main {
			if m.isHeroOrEquipment(card.CardID) {
				hero = append(hero, card)
				continue
			}
			remaining = append(remaining, card)
		}
		main = remaining
	}
	return []deckDisplaySection{
		{title: "Hero / Equipment", cards: hero},
		{title: "Mainboard", cards: main},
		{title: "Inventory", cards: m.details.Sideboard},
		{title: "Maybeboard", cards: m.details.Maybeboard},
		{title: "Other", cards: m.details.Other},
	}
}

func (m Model) renderCardRow(card powersync.DeckCardLine) string {
	name := m.cardName(card.CardID)
	prefix := "  "
	style := lipgloss.NewStyle()
	if selected, ok := m.selectedCard(); ok && selected.ID == card.ID {
		prefix = "| "
		style = listSelectedTitleStyle
	}
	meta := []string{}
	if detail := m.cardLookup[card.CardID]; detail != nil {
		meta = append(meta, detail.TypeLine)
	}
	row := fmt.Sprintf("%s%dx %s", prefix, displayQuantity(card.Quantity), name)
	if len(meta) > 0 {
		row += mutedStyle.Render("  " + oneLine(strings.Join(meta, "  ")))
	}
	return style.Render(row)
}

func (m Model) renderCardPreview() string {
	card, ok := m.selectedCard()
	if !ok {
		return mutedStyle.Render("Select a card row.")
	}
	detail := m.cardLookup[card.CardID]
	if detail == nil {
		return strings.Join([]string{
			sectionLabelStyle.Render("Card"),
			field("ID", card.CardID),
			field("Board", prettyBoardName(card.BoardType)),
			field("Quantity", fmt.Sprintf("%d", displayQuantity(card.Quantity))),
		}, "\n")
	}
	lines := []string{
		splitLine(cardTitleStyle.Render(nonEmpty(detail.Name, card.CardID)), mutedStyle.Render(renderDeckCardContext(card)), max(24, m.preview.Width()-2)),
	}
	cardStats := renderCardStats(detail)
	if detail.TypeLine != "" || cardStats != "" {
		lines = append(lines, splitLine(oneLine(detail.TypeLine), mutedStyle.Render(cardStats), max(24, m.preview.Width()-2)))
	}
	if len(detail.Classes) > 0 || len(detail.Talents) > 0 {
		lines = append(lines, mutedStyle.Render(strings.Join(classTalentValues(detail), "  ")))
	}
	text := nonEmpty(detail.Text, detail.PrintedText)
	if text != "" {
		lines = append(lines, "", sectionLabelStyle.Render("Rules"), wrapText(renderRulesText(text), max(24, m.preview.Width()-2)))
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderArtPanel() string {
	if m.loadingArt {
		return m.spinner.View() + " " + mutedStyle.Render("rendering card image")
	}
	if strings.TrimSpace(m.cardArt) != "" {
		return m.cardArt
	}
	return mutedStyle.Render("No image available.")
}

func (m Model) selectedCardID() string {
	card, ok := m.selectedCard()
	if !ok {
		return ""
	}
	return card.CardID
}

func (m Model) selectedCardLine() (int, bool) {
	selected, ok := m.selectedCard()
	if !ok {
		return 0, false
	}
	line := 0
	for _, section := range m.displaySections() {
		if len(section.cards) == 0 && section.title != "Hero / Equipment" {
			continue
		}
		line++
		if len(section.cards) == 0 {
			line += 2
			continue
		}
		for _, card := range section.cards {
			if card.ID == selected.ID {
				return line, true
			}
			line++
		}
		line++
	}
	return 0, false
}

func (m Model) bannerSeed() string {
	if m.details == nil {
		return "deck"
	}
	deck := m.details.Deck
	return nonEmpty(deck.Name, deck.HeroID, deck.ID, "deck")
}

func renderLargeDeckTitle(title string, width int, foreground string) string {
	title = oneLine(title)
	if title == "" {
		return ""
	}
	style := bannerLargeTitleStyle.Foreground(lipgloss.Color(foreground))
	block := renderBlockTitle(title)
	if block != "" && lipgloss.Width(firstLine(block)) <= width {
		return style.Render(block)
	}
	return style.Bold(true).Render(title)
}

func renderBlockTitle(title string) string {
	title = strings.ToUpper(strings.TrimSpace(title))
	if title == "" {
		return ""
	}
	rows := make([]string, 5)
	for _, r := range title {
		glyph, ok := largeTitleGlyphs[r]
		if !ok {
			if r == ' ' {
				glyph = [5]string{"   ", "   ", "   ", "   ", "   "}
			} else {
				return ""
			}
		}
		for i := range rows {
			if rows[i] != "" {
				rows[i] += " "
			}
			rows[i] += glyph[i]
		}
	}
	return strings.Join(rows, "\n")
}

func firstLine(value string) string {
	if idx := strings.IndexByte(value, '\n'); idx >= 0 {
		return value[:idx]
	}
	return value
}

var largeTitleGlyphs = map[rune][5]string{
	'A': {" ███ ", "█   █", "█████", "█   █", "█   █"},
	'B': {"████ ", "█   █", "████ ", "█   █", "████ "},
	'C': {" ████", "█    ", "█    ", "█    ", " ████"},
	'D': {"████ ", "█   █", "█   █", "█   █", "████ "},
	'E': {"█████", "█    ", "████ ", "█    ", "█████"},
	'F': {"█████", "█    ", "████ ", "█    ", "█    "},
	'G': {" ████", "█    ", "█ ███", "█   █", " ████"},
	'H': {"█   █", "█   █", "█████", "█   █", "█   █"},
	'I': {"█████", "  █  ", "  █  ", "  █  ", "█████"},
	'J': {"█████", "   █ ", "   █ ", "█  █ ", " ██  "},
	'K': {"█   █", "█  █ ", "███  ", "█  █ ", "█   █"},
	'L': {"█    ", "█    ", "█    ", "█    ", "█████"},
	'M': {"█   █", "██ ██", "█ █ █", "█   █", "█   █"},
	'N': {"█   █", "██  █", "█ █ █", "█  ██", "█   █"},
	'O': {" ███ ", "█   █", "█   █", "█   █", " ███ "},
	'P': {"████ ", "█   █", "████ ", "█    ", "█    "},
	'Q': {" ███ ", "█   █", "█ █ █", "█  █ ", " ██ █"},
	'R': {"████ ", "█   █", "████ ", "█  █ ", "█   █"},
	'S': {" ████", "█    ", " ███ ", "    █", "████ "},
	'T': {"█████", "  █  ", "  █  ", "  █  ", "  █  "},
	'U': {"█   █", "█   █", "█   █", "█   █", " ███ "},
	'V': {"█   █", "█   █", "█   █", " █ █ ", "  █  "},
	'W': {"█   █", "█   █", "█ █ █", "██ ██", "█   █"},
	'X': {"█   █", " █ █ ", "  █  ", " █ █ ", "█   █"},
	'Y': {"█   █", " █ █ ", "  █  ", "  █  ", "  █  "},
	'Z': {"█████", "   █ ", "  █  ", " █   ", "█████"},
	'0': {" ███ ", "█  ██", "█ █ █", "██  █", " ███ "},
	'1': {"  █  ", " ██  ", "  █  ", "  █  ", "█████"},
	'2': {"████ ", "    █", " ███ ", "█    ", "█████"},
	'3': {"████ ", "    █", " ███ ", "    █", "████ "},
	'4': {"█   █", "█   █", "█████", "    █", "    █"},
	'5': {"█████", "█    ", "████ ", "    █", "████ "},
	'6': {" ████", "█    ", "████ ", "█   █", " ███ "},
	'7': {"█████", "   █ ", "  █  ", " █   ", "█    "},
	'8': {" ███ ", "█   █", " ███ ", "█   █", " ███ "},
	'9': {" ███ ", "█   █", " ████", "    █", "████ "},
	'-': {"     ", "     ", "█████", "     ", "     "},
	'/': {"    █", "   █ ", "  █  ", " █   ", "█    "},
}

func renderDeckStats(d *powersync.DeckDetails) string {
	if d == nil {
		return ""
	}
	stats := []struct {
		label string
		value int64
	}{
		{"TOTAL", d.TotalQuantity},
		{"MAIN", d.MainboardCount},
		{"SIDE", d.SideboardCount},
		{"MAYBE", d.MaybeboardCount},
	}
	parts := []string{}
	for _, stat := range stats {
		parts = append(parts, bannerStatStyle.Render(fmt.Sprintf(" %s %d ", stat.label, stat.value)))
	}
	return strings.Join(parts, " ")
}

func renderDeckAccent(seed string, width int) string {
	width = max(20, width)
	palette := []string{"#274c43", "#8dd7c7", "#f2d16b", "#e778ff", "#476b60", "#1f2a2e"}
	offset := 0
	for _, r := range seed {
		offset += int(r)
	}
	var b strings.Builder
	for x := 0; x < width; x++ {
		idx := (x/4 + offset) % len(palette)
		if (x+offset)%19 == 0 {
			idx = (idx + 2) % len(palette)
		}
		b.WriteString(lipgloss.NewStyle().Background(lipgloss.Color(palette[idx])).Render(" "))
	}
	return b.String()
}

func (m Model) loadCmd() tea.Cmd {
	return func() tea.Msg {
		if m.options.Store == nil {
			return loadDoneMsg{err: fmt.Errorf("local PowerSync store is not configured")}
		}
		ctx := context.Background()
		synced := false
		if m.syncOnLoad && m.options.Client != nil {
			synced = true
			var syncErr error
			if _, err := m.options.Client.Initialize(ctx); err != nil {
				syncErr = err
			} else if err := m.options.Client.UploadPending(ctx); err != nil {
				syncErr = err
			} else if err := m.options.Client.PullOnce(ctx); err != nil {
				syncErr = err
			} else if incomplete, err := m.options.Store.HasIncompleteDeckData(ctx); err == nil && incomplete {
				status, _ := m.options.Store.Status(ctx)
				if status == nil || status.PendingCrud == 0 {
					if err := m.options.Store.ResetSyncState(ctx); err != nil {
						syncErr = err
					} else if err := m.options.Client.PullOnce(ctx); err != nil {
						syncErr = err
					}
				}
			}
			if syncErr != nil {
				decks, listErr := m.options.Store.ListDecks(ctx, powersync.DeckListParams{
					Scope:        m.scope,
					ViewerUserID: m.options.ViewerUserID,
					Search:       m.input.Value(),
					Limit:        m.options.Limit,
				})
				if listErr != nil {
					return loadDoneMsg{synced: true, err: listErr, syncErr: syncErr.Error()}
				}
				status, statusErr := m.options.Store.Status(ctx)
				if statusErr != nil {
					return loadDoneMsg{synced: true, err: statusErr, syncErr: syncErr.Error()}
				}
				owners := m.resolveDeckOwners(ctx, decks)
				return loadDoneMsg{decks: decks, owners: owners, status: status, synced: true, syncErr: syncErr.Error()}
			}
		}
		decks, err := m.options.Store.ListDecks(ctx, powersync.DeckListParams{
			Scope:        m.scope,
			ViewerUserID: m.options.ViewerUserID,
			Search:       m.input.Value(),
			Limit:        m.options.Limit,
		})
		if err != nil {
			return loadDoneMsg{err: err}
		}
		status, err := m.options.Store.Status(ctx)
		if err != nil {
			return loadDoneMsg{err: err}
		}
		owners := m.resolveDeckOwners(ctx, decks)
		return loadDoneMsg{decks: decks, owners: owners, status: status, synced: synced}
	}
}

func (m *Model) renderSelectedCardArtCmd() tea.Cmd {
	card, ok := m.selectedCard()
	if !ok {
		return func() tea.Msg {
			return deckArtDoneMsg{art: "Select a card row."}
		}
	}
	detail := m.cardLookup[card.CardID]
	imageURL := ""
	if detail != nil {
		imageURL = nonEmpty(detail.ArtURL, detail.ImageURL)
	}
	if m.artCardID == card.CardID && strings.TrimSpace(m.cardArt) != "" {
		if m.artWidth == m.art.Width() && m.artHeight == m.art.Height() {
			return nil
		}
	}
	m.artCardID = card.CardID
	m.loadingArt = strings.TrimSpace(imageURL) != ""
	m.art.SetContent("Rendering card image...")
	artW := max(20, m.art.Width())
	artH := max(10, m.art.Height())
	return func() tea.Msg {
		if strings.TrimSpace(imageURL) == "" || strings.TrimSpace(m.options.ImageCacheDir) == "" {
			return deckArtDoneMsg{
				cardID: card.CardID,
				art:    "No image available for this card.",
				width:  artW,
				height: artH,
			}
		}
		renderer := cardstui.ArtRenderer{CacheDir: m.options.ImageCacheDir}
		cardArt, err := renderer.Render(context.Background(), imageURL, artW, artH)
		if err != nil {
			return deckArtDoneMsg{cardID: card.CardID, width: artW, height: artH, err: err}
		}
		return deckArtDoneMsg{cardID: card.CardID, art: cardArt, width: artW, height: artH}
	}
}

func (m *Model) renderHeroBannerArtCmd() tea.Cmd {
	if m.details == nil {
		return nil
	}
	heroID := strings.TrimSpace(m.details.Deck.HeroID)
	if heroID == "" {
		m.loadingHeroArt = false
		m.heroArt = ""
		return nil
	}
	artW, artH := m.heroBannerArtSize()
	if artW <= 0 || artH <= 0 {
		return nil
	}
	detail := m.cardLookup[heroID]
	imageURL := ""
	if detail != nil {
		imageURL = nonEmpty(detail.ArtURL, detail.ImageURL)
	}
	if m.heroArtID == heroID && strings.TrimSpace(m.heroArt) != "" && m.heroArtW == artW && m.heroArtH == artH {
		return nil
	}
	m.heroArtID = heroID
	m.loadingHeroArt = strings.TrimSpace(imageURL) != ""
	if strings.TrimSpace(imageURL) == "" || strings.TrimSpace(m.options.ImageCacheDir) == "" {
		m.loadingHeroArt = false
		m.heroArt = ""
		return nil
	}
	return func() tea.Msg {
		renderer := cardstui.ArtRenderer{CacheDir: m.options.ImageCacheDir}
		heroArt, err := renderer.Render(context.Background(), imageURL, artW, artH)
		if err != nil {
			return deckHeroArtDoneMsg{cardID: heroID, width: artW, height: artH, err: err}
		}
		return deckHeroArtDoneMsg{cardID: heroID, art: heroArt, width: artW, height: artH}
	}
}

func (m Model) loadDeckDetailsCmd(deckID, versionID string) tea.Cmd {
	return func() tea.Msg {
		if m.options.Store == nil {
			return deckDetailsDoneMsg{err: fmt.Errorf("local PowerSync store is not configured")}
		}
		ctx := context.Background()
		details, err := m.options.Store.GetDeckDetails(ctx, deckID, versionID)
		if err != nil {
			return deckDetailsDoneMsg{err: err}
		}
		cards := map[string]*cardsdb.CardDetail{}
		if m.options.CardsRepo != nil {
			for _, card := range deckDetailCards(details) {
				if _, ok := cards[card.CardID]; ok || strings.TrimSpace(card.CardID) == "" {
					continue
				}
				detail, err := m.options.CardsRepo.GetCard(ctx, card.CardID)
				if err != nil && err != sql.ErrNoRows {
					continue
				}
				if detail != nil {
					cards[card.CardID] = detail
				}
			}
			heroID := strings.TrimSpace(details.Deck.HeroID)
			if heroID != "" {
				if _, ok := cards[heroID]; !ok {
					detail, err := m.options.CardsRepo.GetCard(ctx, heroID)
					if err == nil && detail != nil {
						cards[heroID] = detail
					}
				}
			}
		}
		owners := m.resolveDeckOwners(ctx, []powersync.DeckSummary{details.Deck})
		return deckDetailsDoneMsg{details: details, cards: cards, owners: owners}
	}
}

func deckDetailCards(details *powersync.DeckDetails) []powersync.DeckCardLine {
	if details == nil {
		return nil
	}
	out := []powersync.DeckCardLine{}
	out = append(out, details.HeroEquipment...)
	out = append(out, details.Mainboard...)
	out = append(out, details.Sideboard...)
	out = append(out, details.Maybeboard...)
	out = append(out, details.Other...)
	return out
}

func nextScope(scope powersync.DeckListScope) powersync.DeckListScope {
	switch scope {
	case powersync.DeckListScopeAccessible:
		return powersync.DeckListScopeOwned
	case powersync.DeckListScopeOwned:
		return powersync.DeckListScopeShared
	default:
		return powersync.DeckListScopeAccessible
	}
}

func (m *Model) moveCardSelection(delta int) {
	cards := m.visibleCards()
	if len(cards) == 0 {
		m.cardIndex = 0
		return
	}
	m.cardIndex += delta
	m.clampCardSelection()
}

func (m *Model) clampCardSelection() {
	cards := m.visibleCards()
	if len(cards) == 0 {
		m.cardIndex = 0
		return
	}
	if m.cardIndex < 0 {
		m.cardIndex = 0
	}
	if m.cardIndex >= len(cards) {
		m.cardIndex = len(cards) - 1
	}
}

func (m Model) selectedCard() (powersync.DeckCardLine, bool) {
	cards := m.visibleCards()
	if len(cards) == 0 || m.cardIndex < 0 || m.cardIndex >= len(cards) {
		return powersync.DeckCardLine{}, false
	}
	return cards[m.cardIndex], true
}

func (m Model) visibleCards() []powersync.DeckCardLine {
	sections := m.displaySections()
	out := []powersync.DeckCardLine{}
	for _, section := range sections {
		out = append(out, section.cards...)
	}
	return out
}

func (m Model) nextVersionID(key string) string {
	if m.details == nil || len(m.details.Versions) == 0 {
		return m.versionID
	}
	idx := 0
	for i, version := range m.details.Versions {
		if version.ID == m.details.SelectedVersionID {
			idx = i
			break
		}
	}
	if key == "[" {
		idx--
	} else {
		idx++
	}
	if idx < 0 {
		idx = len(m.details.Versions) - 1
	}
	if idx >= len(m.details.Versions) {
		idx = 0
	}
	return m.details.Versions[idx].ID
}

func (m Model) cardName(cardID string) string {
	if detail := m.cardLookup[cardID]; detail != nil && strings.TrimSpace(detail.Name) != "" {
		return detail.Name
	}
	return cardID
}

func (m Model) heroDisplayName(heroID string) string {
	heroID = strings.TrimSpace(heroID)
	if heroID == "" {
		return ""
	}
	if detail := m.cardLookup[heroID]; detail != nil && strings.TrimSpace(detail.Name) != "" {
		return detail.Name
	}
	return prettyCardID(heroID)
}

func (m Model) deckOwnerName(deck powersync.DeckSummary) string {
	author := strings.TrimSpace(deck.Author)
	if author != "" && !looksLikeOpaqueID(author) {
		return author
	}
	if m.isViewerDeck(deck) {
		if name := strings.TrimSpace(m.options.ViewerName); name != "" {
			return name
		}
		return "You"
	}
	if name := strings.TrimSpace(m.ownerLookup[ownerLookupKey(deckOwnerUserID(deck))]); name != "" {
		return name
	}
	return ""
}

func (m Model) withResolvedDeckOwners(decks []powersync.DeckSummary) []powersync.DeckSummary {
	out := make([]powersync.DeckSummary, 0, len(decks))
	for _, deck := range decks {
		out = append(out, m.withResolvedDeckOwner(deck))
	}
	return out
}

func (m Model) withResolvedDeckOwner(deck powersync.DeckSummary) powersync.DeckSummary {
	if strings.TrimSpace(deck.Author) != "" && !looksLikeOpaqueID(deck.Author) {
		return deck
	}
	if name := m.deckOwnerName(deck); name != "" && name != "You" {
		deck.Author = name
	}
	return deck
}

func (m *Model) mergeOwnerLookup(values map[string]string) {
	if len(values) == 0 {
		return
	}
	if m.ownerLookup == nil {
		m.ownerLookup = map[string]string{}
	}
	for id, name := range values {
		key := ownerLookupKey(id)
		name = strings.TrimSpace(name)
		if key == "" || name == "" {
			continue
		}
		m.ownerLookup[key] = name
	}
}

func (m Model) resolveDeckOwners(ctx context.Context, decks []powersync.DeckSummary) map[string]string {
	if m.options.ProfileClient == nil || len(decks) == 0 {
		return nil
	}
	out := map[string]string{}
	seen := map[string]struct{}{}
	for _, deck := range decks {
		if strings.TrimSpace(deck.Author) != "" && !looksLikeOpaqueID(deck.Author) {
			continue
		}
		userID := deckOwnerUserID(deck)
		key := ownerLookupKey(userID)
		if key == "" || sameUserID(key, m.options.ViewerUserID) {
			continue
		}
		if _, ok := m.ownerLookup[key]; ok {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		resp, err := m.options.ProfileClient.GetProfile(ctx, &clientv1.GetProfileRequest{UserID: apiUserID(userID)})
		if err != nil || resp == nil || resp.Profile == nil {
			continue
		}
		name := nonEmpty(resp.Profile.Username, resp.Profile.Name)
		if name != "" {
			out[key] = name
		}
	}
	return out
}

func initialOwnerLookup(opts Options) map[string]string {
	out := map[string]string{}
	viewerID := ownerLookupKey(opts.ViewerUserID)
	if viewerID != "" && strings.TrimSpace(opts.ViewerName) != "" {
		out[viewerID] = strings.TrimSpace(opts.ViewerName)
	}
	return out
}

func deckOwnerUserID(deck powersync.DeckSummary) string {
	if strings.TrimSpace(deck.UserID) != "" {
		return strings.TrimSpace(deck.UserID)
	}
	if looksLikeOpaqueID(deck.Author) {
		return strings.TrimSpace(deck.Author)
	}
	return ""
}

func ownerLookupKey(userID string) string {
	return strings.TrimPrefix(strings.TrimSpace(userID), "u-")
}

func apiUserID(userID string) string {
	userID = strings.TrimSpace(userID)
	if userID == "" || strings.HasPrefix(userID, "u-") {
		return userID
	}
	if looksLikeUUID(userID) {
		return "u-" + userID
	}
	return userID
}

func looksLikeUUID(value string) bool {
	value = strings.TrimSpace(value)
	return len(value) == 36 && strings.Count(value, "-") == 4
}

func (m Model) isViewerDeck(deck powersync.DeckSummary) bool {
	viewerID := strings.TrimSpace(m.options.ViewerUserID)
	if viewerID == "" {
		return false
	}
	return sameUserID(deck.UserID, viewerID) || sameUserID(deck.Author, viewerID)
}

func (m Model) isHeroOrEquipment(cardID string) bool {
	detail := m.cardLookup[cardID]
	if detail == nil {
		return false
	}
	typeLine := strings.ToLower(detail.TypeLine)
	return strings.Contains(typeLine, "hero") || strings.Contains(typeLine, "weapon") || strings.Contains(typeLine, "equipment")
}

func displayQuantity(quantity int64) int64 {
	if quantity <= 0 {
		return 1
	}
	return quantity
}

func renderCardStats(detail *cardsdb.CardDetail) string {
	if detail == nil {
		return ""
	}
	parts := []string{}
	add := func(label, value string) {
		if strings.TrimSpace(value) != "" {
			parts = append(parts, label+" "+strings.TrimSpace(value))
		}
	}
	for _, stat := range []struct {
		label string
		value string
	}{
		{"Cost", detail.Cost},
		{"Pitch", detail.Pitch},
		{"Attack", detail.Power},
		{"Defense", detail.Defense},
		{"Health", detail.Health},
		{"Intellect", detail.Intelligence},
		{"Arcane", detail.Arcane},
	} {
		add(stat.label, stat.value)
	}
	if len(detail.Keywords) > 0 {
		parts = append(parts, "Keywords "+strings.Join(detail.Keywords, ", "))
	}
	return strings.Join(parts, "  ")
}

func renderDeckCardContext(card powersync.DeckCardLine) string {
	quantity := fmt.Sprintf("x%d", displayQuantity(card.Quantity))
	board := prettyBoardName(card.BoardType)
	if board == "" {
		return quantity
	}
	return quantity + " - " + board
}

func prettyFormatName(value string) string {
	normalized := normalizeDisplayToken(value)
	switch normalized {
	case "":
		return ""
	case "cc", "classic", "classic constructed", "constructed":
		return "Classic Constructed"
	case "blitz":
		return "Blitz"
	case "commoner":
		return "Commoner"
	case "upf", "ultimate pit fight":
		return "Ultimate Pit Fight"
	case "ll", "living legend":
		return "Living Legend"
	case "project blue", "pb", "silver age":
		return "Silver Age"
	default:
		return titleWords(normalized)
	}
}

func prettyVisibility(value string) string {
	return titleWords(normalizeDisplayToken(value))
}

func prettyBoardName(value string) string {
	normalized := normalizeDisplayToken(value)
	switch normalized {
	case "":
		return ""
	case "main", "main board", "mainboard":
		return "Mainboard"
	case "side", "side board", "sideboard", "inventory":
		return "Sideboard"
	case "maybe", "maybe board", "maybeboard":
		return "Maybeboard"
	default:
		return titleWords(normalized)
	}
}

func prettyCardID(value string) string {
	return titleWords(normalizeDisplayToken(value))
}

func normalizeDisplayToken(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.NewReplacer("-", " ", "_", " ").Replace(value)
	value = strings.ToLower(strings.Join(strings.Fields(value), " "))
	for _, prefix := range []string{"deck format ", "format ", "visibility ", "board type "} {
		value = strings.TrimPrefix(value, prefix)
	}
	return value
}

func titleWords(value string) string {
	words := strings.Fields(value)
	for i, word := range words {
		if len(word) == 0 {
			continue
		}
		switch word {
		case "of", "the", "and", "or", "by":
			if i != 0 {
				words[i] = word
				continue
			}
		}
		words[i] = strings.ToUpper(word[:1]) + word[1:]
	}
	return strings.Join(words, " ")
}

func looksLikeOpaqueID(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	stripped := strings.TrimPrefix(value, "u-")
	if len(stripped) == 36 && strings.Count(stripped, "-") == 4 {
		return true
	}
	if strings.Contains(stripped, "-") && len(stripped) >= 24 {
		return true
	}
	return false
}

func sameUserID(a, b string) bool {
	a = strings.TrimPrefix(strings.TrimSpace(a), "u-")
	b = strings.TrimPrefix(strings.TrimSpace(b), "u-")
	return a != "" && b != "" && a == b
}

func normalizeHexColor(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if !strings.HasPrefix(value, "#") {
		value = "#" + value
	}
	if len(value) != 7 {
		return ""
	}
	var r, g, b uint8
	if _, err := fmt.Sscanf(value, "#%02x%02x%02x", &r, &g, &b); err != nil {
		return ""
	}
	return strings.ToLower(value)
}

func readableTextColor(background string) string {
	r, g, b, ok := parseHexRGB(background)
	if !ok {
		return "#f8f8f2"
	}
	luma := (299*int(r) + 587*int(g) + 114*int(b)) / 1000
	if luma > 140 {
		return "#11181b"
	}
	return "#f8f8f2"
}

func bannerBorderColor(background string) string {
	r, g, b, ok := parseHexRGB(background)
	if !ok {
		return "#f2d16b"
	}
	luma := (299*int(r) + 587*int(g) + 114*int(b)) / 1000
	if luma > 140 {
		return "#6f6642"
	}
	return "#f2d16b"
}

func parseHexRGB(value string) (uint8, uint8, uint8, bool) {
	value = normalizeHexColor(value)
	if value == "" {
		return 0, 0, 0, false
	}
	var r, g, b uint8
	if _, err := fmt.Sscanf(value, "#%02x%02x%02x", &r, &g, &b); err != nil {
		return 0, 0, 0, false
	}
	return r, g, b, true
}

func paintTransparentArtCells(art, background string) string {
	background = normalizeHexColor(background)
	if background == "" || strings.TrimSpace(art) == "" {
		return art
	}
	r, g, b, ok := parseHexRGB(background)
	if !ok {
		return art
	}
	bgCell := fmt.Sprintf("\x1b[0m\x1b[48;2;%d;%d;%dm \x1b[0m", r, g, b)
	return strings.ReplaceAll(art, "\x1b[0m ", bgCell)
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

func classTalentValues(detail *cardsdb.CardDetail) []string {
	if detail == nil {
		return nil
	}
	values := append([]string{}, detail.Classes...)
	values = append(values, detail.Talents...)
	return nonEmptyValues(values...)
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

func wrapText(value string, width int) string {
	width = max(20, width)
	words := strings.Fields(value)
	if len(words) == 0 {
		return ""
	}
	lines := []string{}
	line := ""
	for _, word := range words {
		if line == "" {
			line = word
			continue
		}
		if len(line)+1+len(word) > width {
			lines = append(lines, line)
			line = word
			continue
		}
		line += " " + word
	}
	if line != "" {
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func renderPanel(style lipgloss.Style, width, height int, content string) string {
	return style.Width(width).Height(height).MaxWidth(width).MaxHeight(height).Render(content)
}

func field(label, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = "-"
	}
	return mutedStyle.Render(label+": ") + value
}

func nonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func nonEmptyValues(values ...string) []string {
	out := []string{}
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, strings.TrimSpace(value))
		}
	}
	return out
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func shortDate(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		t, err := time.Parse(layout, value)
		if err == nil {
			return t.Local().Format("Jan 2")
		}
	}
	if len(value) > 10 {
		return value[:10]
	}
	return value
}

func shortDateTime(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		t, err := time.Parse(layout, value)
		if err == nil {
			return t.Local().Format("Jan 2 15:04")
		}
	}
	return shortDate(value)
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
