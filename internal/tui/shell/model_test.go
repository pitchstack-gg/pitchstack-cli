package shell

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestModelSwitchesTabs(t *testing.T) {
	t.Parallel()
	model := New(Options{
		Cards:      stubModel{name: "cards"},
		Decks:      stubModel{name: "decks"},
		InitialTab: TabCards,
	})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model = updated.(Model)
	if !strings.Contains(model.View().Content, "cards") {
		t.Fatalf("initial view = %q", model.View().Content)
	}
	updated, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	model = updated.(Model)
	if model.active != TabDecks || !strings.Contains(model.View().Content, "decks") {
		t.Fatalf("after tab active=%q view=%q", model.active, model.View().Content)
	}
}

func TestParseTab(t *testing.T) {
	t.Parallel()
	if tab, err := ParseTab("decks"); err != nil || tab != TabDecks {
		t.Fatalf("ParseTab(decks) = %q, %v", tab, err)
	}
	if _, err := ParseTab("bad"); err == nil {
		t.Fatalf("ParseTab(bad) err = nil")
	}
}

type stubModel struct {
	name string
}

func (m stubModel) Init() tea.Cmd { return nil }

func (m stubModel) Update(tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m stubModel) View() tea.View {
	v := tea.NewView(m.name)
	v.AltScreen = true
	return v
}
