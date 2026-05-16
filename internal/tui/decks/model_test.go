package decks

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/pitchstack-gg/pitchstack-cli/internal/cardsdb"
	"github.com/pitchstack-gg/pitchstack-cli/internal/powersync"
	clientv1 "github.com/pitchstack-gg/pitchstack-go/client/v1"
)

func TestModelLoadsSearchesScopesAndDetails(t *testing.T) {
	t.Parallel()
	store := openTestStore(t)
	defer store.Close()
	if err := store.ApplyOperations(context.Background(), "cp-1", []powersync.Operation{
		{Bucket: "b1", OpID: "1", Op: "PUT", Table: "decks", ID: "deck-owned", Data: map[string]any{"id": "deck-owned", "user_id": "viewer", "name": "Bravo Blitz", "hero_id": "bravo", "format": "blitz", "active_version_id": "ver-1"}},
		{Bucket: "b1", OpID: "2", Op: "PUT", Table: "deck_versions", ID: "ver-1", Data: map[string]any{"id": "ver-1", "deck_id": "deck-owned", "version_name": "Current"}},
		{Bucket: "b1", OpID: "3", Op: "PUT", Table: "deck_cards", ID: "card-1", Data: map[string]any{"id": "card-1", "deck_version_id": "ver-1", "board_type": "mainboard", "quantity": 3}},
		{Bucket: "b1", OpID: "4", Op: "PUT", Table: "decks", ID: "deck-shared", Data: map[string]any{"id": "deck-shared", "user_id": "other", "name": "Azalea Control", "hero_id": "azalea", "format": "cc"}},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	model := New(Options{Store: store, ViewerUserID: "viewer"})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	model = updated.(Model)
	updated, _ = model.Update(loadDoneMsg{decks: mustListDecks(t, store, powersync.DeckListParams{ViewerUserID: "viewer"}), status: &powersync.Status{Rows: 5}})
	model = updated.(Model)
	if len(model.decks) != 2 {
		t.Fatalf("decks len = %d, want 2", len(model.decks))
	}

	updated, _ = model.Update(tea.KeyPressMsg{Text: "/"})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyPressMsg{Text: "b"})
	model = updated.(Model)
	updated, _ = model.Update(loadDoneMsg{decks: mustListDecks(t, store, powersync.DeckListParams{ViewerUserID: "viewer", Search: "b"}), status: &powersync.Status{Rows: 5}})
	model = updated.(Model)
	if len(model.decks) != 1 || model.decks[0].ID != "deck-owned" {
		t.Fatalf("searched decks = %#v", model.decks)
	}

	model.focus = focusList
	updated, _ = model.Update(tea.KeyPressMsg{Text: "o"})
	model = updated.(Model)
	if model.scope != powersync.DeckListScopeOwned {
		t.Fatalf("scope = %q, want owned", model.scope)
	}

	updated, _ = model.Update(loadDoneMsg{decks: mustListDecks(t, store, powersync.DeckListParams{Scope: powersync.DeckListScopeOwned, ViewerUserID: "viewer", Search: "b"}), status: &powersync.Status{Rows: 5}})
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyPressMsg{Text: "enter"})
	model = updated.(Model)
	if model.mode != modeDetail {
		t.Fatalf("mode = %q, want detail", model.mode)
	}
	details, err := store.GetDeckDetails(context.Background(), "deck-owned", "ver-1")
	if err != nil {
		t.Fatalf("details: %v", err)
	}
	updated, _ = model.Update(deckDetailsDoneMsg{details: details, cards: map[string]*cardsdb.CardDetail{}})
	model = updated.(Model)
	if model.details == nil {
		t.Fatalf("details = nil")
	}
	output := stripANSILite(model.renderDetailMode())
	if !strings.Contains(output, "████") || !strings.Contains(output, "Current") || !strings.Contains(output, "3") {
		t.Fatalf("detail output missing deck data:\n%s", output)
	}
	updated, _ = model.Update(tea.KeyPressMsg{Text: "esc"})
	model = updated.(Model)
	if model.mode != modeList {
		t.Fatalf("mode after esc = %q, want list", model.mode)
	}
}

func TestModelEmptyStates(t *testing.T) {
	t.Parallel()
	model := New(Options{MissingDB: true})
	model.loading = false
	if got := model.emptyMessage(); !strings.Contains(got, "Log in") {
		t.Fatalf("missing db message = %q", got)
	}
	model.options.MissingDB = false
	model.input.SetValue("azalea")
	if got := model.emptyMessage(); !strings.Contains(got, "No decks match") {
		t.Fatalf("search empty message = %q", got)
	}
	model.input.SetValue("")
	model.scope = powersync.DeckListScopeShared
	if got := model.emptyMessage(); !strings.Contains(got, "No shared decks") {
		t.Fatalf("shared empty message = %q", got)
	}
}

func TestModelResolvesSharedDeckOwnerFromAPI(t *testing.T) {
	t.Parallel()
	store := openTestStore(t)
	defer store.Close()
	ownerID := "720b07fe-3dd6-46da-bb98-c184d4b28958"
	if err := store.ApplyOperations(context.Background(), "cp-1", []powersync.Operation{
		{Bucket: "b1", OpID: "1", Op: "PUT", Table: "decks", ID: "deck-shared", Data: map[string]any{"id": "deck-shared", "user_id": ownerID, "name": "Shared Deck", "hero_id": "bravo", "format": "blitz", "active_version_id": "ver-1"}},
		{Bucket: "b1", OpID: "2", Op: "PUT", Table: "deck_versions", ID: "ver-1", Data: map[string]any{"id": "ver-1", "deck_id": "deck-shared", "version_name": "Current"}},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	model := New(Options{Store: store, ViewerUserID: "viewer", ProfileClient: fakeProfileClient{names: map[string]string{"u-" + ownerID: "friend"}}})
	msg := model.loadCmd()().(loadDoneMsg)
	updated, _ := model.Update(msg)
	model = updated.(Model)
	if len(model.decks) != 1 {
		t.Fatalf("decks len = %d, want 1", len(model.decks))
	}
	if got := model.decks[0].Author; got != "friend" {
		t.Fatalf("resolved author = %q, want friend", got)
	}
}

type fakeProfileClient struct {
	names map[string]string
}

func (f fakeProfileClient) GetProfile(_ context.Context, req *clientv1.GetProfileRequest, _ ...clientv1.RequestOpt) (*clientv1.GetProfileResponse, error) {
	name := f.names[req.UserID]
	return &clientv1.GetProfileResponse{Profile: &clientv1.UserProfile{Username: name}}, nil
}

func mustListDecks(t *testing.T, store *powersync.Store, params powersync.DeckListParams) []powersync.DeckSummary {
	t.Helper()
	decks, err := store.ListDecks(context.Background(), params)
	if err != nil {
		t.Fatalf("list decks: %v", err)
	}
	return decks
}

func openTestStore(t *testing.T) *powersync.Store {
	t.Helper()
	store, err := powersync.OpenStore(t.TempDir() + "/sync.sqlite")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return store
}

func stripANSILite(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		if inEsc {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEsc = false
			}
			continue
		}
		if r == '\x1b' {
			inEsc = true
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
