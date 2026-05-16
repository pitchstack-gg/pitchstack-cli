package shell

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type Tab string

const (
	TabCards Tab = "cards"
	TabDecks Tab = "decks"
)

type Options struct {
	Cards      tea.Model
	Decks      tea.Model
	InitialTab Tab
}

type Model struct {
	cards tea.Model
	decks tea.Model

	active      Tab
	initialized map[Tab]bool
	width       int
	height      int
}

func New(opts Options) Model {
	active := opts.InitialTab
	if active == "" {
		active = TabCards
	}
	return Model{
		cards:       opts.Cards,
		decks:       opts.Decks,
		active:      active,
		initialized: map[Tab]bool{},
	}
}

func ParseTab(value string) (Tab, error) {
	switch Tab(strings.ToLower(strings.TrimSpace(value))) {
	case "", TabCards:
		return TabCards, nil
	case TabDecks:
		return TabDecks, nil
	default:
		return "", fmt.Errorf("tab must be cards or decks")
	}
}

func (m Model) Init() tea.Cmd {
	return m.initActive()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyPressMsg:
		switch msg.String() {
		case "tab":
			cmds = append(cmds, m.switchTo(m.nextTab()))
			return m, tea.Batch(cmds...)
		case "shift+tab":
			cmds = append(cmds, m.switchTo(m.prevTab()))
			return m, tea.Batch(cmds...)
		case "ctrl+c":
			return m, tea.Quit
		}
		child, cmd := m.updateActive(msg)
		m.setActiveModel(child)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}

	for _, tab := range []Tab{TabCards, TabDecks} {
		if !m.initialized[tab] {
			continue
		}
		child, cmd := m.updateTab(tab, msg)
		m.setTabModel(tab, child)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m Model) View() tea.View {
	view := m.activeModel().View()
	content := lipgloss.JoinVertical(lipgloss.Left, m.renderTabs(), view.Content)
	out := tea.NewView(content)
	out.AltScreen = true
	out.Cursor = view.Cursor
	out.OnMouse = view.OnMouse
	return out
}

func (m *Model) switchTo(tab Tab) tea.Cmd {
	if tab == m.active {
		return nil
	}
	m.active = tab
	return m.initActive()
}

func (m *Model) initActive() tea.Cmd {
	if m.initialized == nil {
		m.initialized = map[Tab]bool{}
	}
	if m.initialized[m.active] {
		if m.width > 0 && m.height > 1 {
			child, cmd := m.updateActive(tea.WindowSizeMsg{Width: m.width, Height: m.height - 1})
			m.setActiveModel(child)
			return cmd
		}
		return nil
	}
	m.initialized[m.active] = true
	var cmds []tea.Cmd
	if m.width > 0 && m.height > 1 {
		child, cmd := m.updateActive(tea.WindowSizeMsg{Width: m.width, Height: m.height - 1})
		m.setActiveModel(child)
		cmds = append(cmds, cmd)
	}
	cmds = append(cmds, m.activeModel().Init())
	return tea.Batch(cmds...)
}

func (m Model) activeModel() tea.Model {
	if m.active == TabDecks {
		return m.decks
	}
	return m.cards
}

func (m *Model) setActiveModel(model tea.Model) {
	m.setTabModel(m.active, model)
}

func (m *Model) setTabModel(tab Tab, model tea.Model) {
	if model == nil {
		return
	}
	if tab == TabDecks {
		m.decks = model
		return
	}
	m.cards = model
}

func (m Model) updateActive(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m.updateTab(m.active, msg)
}

func (m Model) updateTab(tab Tab, msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok && size.Height > 1 {
		msg = tea.WindowSizeMsg{Width: size.Width, Height: size.Height - 1}
	}
	if tab == TabDecks {
		return m.decks.Update(msg)
	}
	return m.cards.Update(msg)
}

func (m Model) nextTab() Tab {
	if m.active == TabCards {
		return TabDecks
	}
	return TabCards
}

func (m Model) prevTab() Tab {
	return m.nextTab()
}

func (m Model) renderTabs() string {
	labels := []string{
		m.tabLabel(TabCards, "Cards"),
		m.tabLabel(TabDecks, "Decks"),
	}
	help := mutedStyle.Render("tab switch")
	width := m.width
	if width <= 0 {
		width = 80
	}
	line := strings.Join(labels, " ")
	return lipgloss.PlaceHorizontal(width, lipgloss.Left, line+"  "+help)
}

func (m Model) tabLabel(tab Tab, label string) string {
	if m.active == tab {
		return activeTabStyle.Render(" " + label + " ")
	}
	return tabStyle.Render(" " + label + " ")
}
