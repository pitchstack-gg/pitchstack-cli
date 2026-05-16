package powersync

import (
	"context"
	"testing"

	clientv1 "github.com/pitchstack-gg/pitchstack-go/client/v1"
)

func TestStoreApplyOperationsCountsAndBucketRemoval(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := openTestStore(t)
	defer store.Close()

	err := store.ApplyOperations(ctx, "cp-1", []Operation{
		{Bucket: "b1", OpID: "1", Op: "PUT", Table: "collections", ID: "col-1", Data: map[string]any{"id": "col-1", "name": "Binder"}},
		{Bucket: "b1", OpID: "2", Op: "PUT", Table: "collection_items", ID: "item-1", Data: map[string]any{"id": "item-1", "collectionId": "col-1", "cardId": "card-1", "quantity": 2}},
		{Bucket: "b1", OpID: "3", Op: "PUT", Table: "collection_items", ID: "item-2", Data: map[string]any{"id": "item-2", "collectionId": "col-1", "cardId": "card-1", "quantity": 1}},
		{Bucket: "b2", OpID: "1", Op: "PUT", Table: "collection_items", ID: "shared-item", Data: map[string]any{"id": "shared-item", "collectionId": "col-1", "cardId": "card-2", "quantity": 4}},
		{Bucket: "b3", OpID: "1", Op: "PUT", Table: "collection_items", ID: "shared-item", Data: map[string]any{"id": "shared-item", "collectionId": "col-1", "cardId": "card-2", "quantity": 4}},
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	counts, err := store.CollectionCounts(ctx)
	if err != nil {
		t.Fatalf("counts: %v", err)
	}
	if len(counts) != 1 {
		t.Fatalf("counts len = %d, want 1", len(counts))
	}
	if counts[0].ItemCount != 3 || counts[0].QuantityCount != 7 || counts[0].UniqueCardCount != 2 {
		t.Fatalf("counts = %#v, want item=3 quantity=7 unique=2", counts[0])
	}

	err = store.ApplyOperations(ctx, "cp-2", []Operation{{Bucket: "b2", OpID: "2", Op: "REMOVE", Table: "collection_items", ID: "shared-item"}})
	if err != nil {
		t.Fatalf("remove b2: %v", err)
	}
	counts, err = store.CollectionCounts(ctx)
	if err != nil {
		t.Fatalf("counts after remove b2: %v", err)
	}
	if counts[0].ItemCount != 3 {
		t.Fatalf("item count after first bucket remove = %d, want 3", counts[0].ItemCount)
	}

	err = store.ApplyOperations(ctx, "cp-3", []Operation{{Bucket: "b3", OpID: "2", Op: "REMOVE", Table: "collection_items", ID: "shared-item"}})
	if err != nil {
		t.Fatalf("remove b3: %v", err)
	}
	counts, err = store.CollectionCounts(ctx)
	if err != nil {
		t.Fatalf("counts after remove b3: %v", err)
	}
	if counts[0].ItemCount != 2 || counts[0].QuantityCount != 3 || counts[0].UniqueCardCount != 1 {
		t.Fatalf("counts after final remove = %#v", counts[0])
	}
}

func TestStoreEpochResetAndOutboxLifecycle(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := openTestStore(t)
	defer store.Close()

	if _, err := store.EnsureSyncEpoch(ctx, "epoch-1"); err != nil {
		t.Fatalf("epoch 1: %v", err)
	}
	if err := store.PutLocal(ctx, "collections", "col-1", map[string]any{"id": "col-1", "name": "Local"}); err != nil {
		t.Fatalf("put local: %v", err)
	}
	batch, err := store.NextCrudBatch(ctx, 100)
	if err != nil {
		t.Fatalf("next crud: %v", err)
	}
	if len(batch) != 1 || batch[0].Op != "PUT" || batch[0].Type != "collections" || batch[0].ID != "col-1" {
		t.Fatalf("batch = %#v", batch)
	}
	if err := store.MarkCrudUploaded(ctx, []clientv1.UploadCrudResult{{OpID: "1", Status: clientv1.SyncStatusOK}}, "wcp-1"); err != nil {
		t.Fatalf("mark uploaded: %v", err)
	}
	status, err := store.Status(ctx)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.PendingCrud != 0 || status.LastWriteCheckpoint != "wcp-1" {
		t.Fatalf("status after upload = %#v", status)
	}
	if err := store.DeleteLocal(ctx, "collections", "col-1"); err != nil {
		t.Fatalf("delete local: %v", err)
	}
	if err := store.MarkCrudUploaded(ctx, []clientv1.UploadCrudResult{{OpID: "2", Status: clientv1.SyncStatusError, ErrorMessage: "denied"}}, ""); err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	status, err = store.Status(ctx)
	if err != nil {
		t.Fatalf("status after failed upload: %v", err)
	}
	if status.FailedCrud != 1 || len(status.FailedEntries) != 1 || status.FailedEntries[0].Error != "denied" {
		t.Fatalf("status failed entries = %#v", status)
	}
	reset, err := store.EnsureSyncEpoch(ctx, "epoch-2")
	if err != nil {
		t.Fatalf("epoch 2: %v", err)
	}
	if !reset {
		t.Fatalf("epoch reset = false, want true")
	}
	status, err = store.Status(ctx)
	if err != nil {
		t.Fatalf("status after reset: %v", err)
	}
	if status.Rows != 0 || status.PendingCrud != 0 || status.SyncEpoch != "epoch-2" {
		t.Fatalf("status after epoch reset = %#v", status)
	}
}

func TestStoreListDecksFiltersSearchesAndAggregates(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := openTestStore(t)
	defer store.Close()

	err := store.ApplyOperations(ctx, "cp-1", []Operation{
		{Bucket: "b1", OpID: "1", Op: "PUT", Table: "decks", ID: "deck-owned", Data: map[string]any{
			"id": "deck-owned", "user_id": "viewer", "deck_kind": "DECK_KIND_USER", "name": "Bravo Blitz", "author": "Me", "hero_id": "bravo", "format": "blitz", "active_version_id": "ver-owned", "visibility": "private", "created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-03T00:00:00Z",
		}},
		{Bucket: "b1", OpID: "2", Op: "PUT", Table: "deck_versions", ID: "ver-owned", Data: map[string]any{"id": "ver-owned", "deck_id": "deck-owned", "version_name": "Current"}},
		{Bucket: "b1", OpID: "3", Op: "PUT", Table: "deck_versions", ID: "ver-old", Data: map[string]any{"id": "ver-old", "deck_id": "deck-owned", "version_name": "Old"}},
		{Bucket: "b1", OpID: "4", Op: "PUT", Table: "deck_cards", ID: "card-1", Data: map[string]any{"id": "card-1", "deck_id": "deck-owned", "deck_version_id": "ver-owned", "card_id": "c1", "board_type": "mainboard", "quantity": 3}},
		{Bucket: "b1", OpID: "5", Op: "PUT", Table: "deck_cards", ID: "card-2", Data: map[string]any{"id": "card-2", "deck_id": "deck-owned", "deck_version_id": "ver-owned", "card_id": "c2", "board_type": "sideboard", "quantity": 2}},
		{Bucket: "b1", OpID: "6", Op: "PUT", Table: "decks", ID: "deck-shared", Data: map[string]any{
			"id": "deck-shared", "userId": "other", "deckKind": "DECK_KIND_USER", "name": "Azalea Control", "author": "Friend", "heroId": "azalea", "format": "cc", "activeDeckVersionId": "ver-shared", "createdAt": "2026-01-02T00:00:00Z", "updatedAt": "2026-01-04T00:00:00Z",
		}},
		{Bucket: "b1", OpID: "7", Op: "PUT", Table: "decks", ID: "deck-ref", Data: map[string]any{
			"id": "deck-ref", "user_id": "viewer", "deck_kind": "DECK_KIND_REFERENCE", "name": "Public Reference", "updated_at": "2026-01-05T00:00:00Z",
		}},
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	all, err := store.ListDecks(ctx, DeckListParams{ViewerUserID: "u-viewer"})
	if err != nil {
		t.Fatalf("list accessible: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("accessible len = %d, want 2: %#v", len(all), all)
	}
	if all[0].ID != "deck-shared" || all[1].ID != "deck-owned" {
		t.Fatalf("sort order = %#v", all)
	}

	owned, err := store.ListDecks(ctx, DeckListParams{Scope: DeckListScopeOwned, ViewerUserID: "u-viewer"})
	if err != nil {
		t.Fatalf("list owned: %v", err)
	}
	if len(owned) != 1 || owned[0].ID != "deck-owned" || owned[0].Ownership != "owned" {
		t.Fatalf("owned = %#v", owned)
	}
	if owned[0].ActiveVersionID != "ver-owned" || owned[0].ActiveVersionName != "Current" || owned[0].VersionCount != 2 {
		t.Fatalf("owned version fields = %#v", owned[0])
	}
	if owned[0].CardRowCount != 2 || owned[0].TotalQuantity != 5 || len(owned[0].ActiveVersionBoards) != 2 {
		t.Fatalf("owned card counts = %#v", owned[0])
	}

	shared, err := store.ListDecks(ctx, DeckListParams{Scope: DeckListScopeShared, ViewerUserID: "viewer", Search: "azalea"})
	if err != nil {
		t.Fatalf("list shared search: %v", err)
	}
	if len(shared) != 1 || shared[0].ID != "deck-shared" || shared[0].Ownership != "shared" {
		t.Fatalf("shared = %#v", shared)
	}
}

func TestStoreDeckDetailsResolvesVersionAndGroupsCards(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := openTestStore(t)
	defer store.Close()

	err := store.ApplyOperations(ctx, "cp-1", []Operation{
		{Bucket: "b1", OpID: "1", Op: "PUT", Table: "decks", ID: "deck-1", Data: map[string]any{
			"id": "deck-1", "user_id": "viewer", "name": "Fallback Kano", "hero_id": "kano", "format": "blitz",
		}},
		{Bucket: "b1", OpID: "2", Op: "PUT", Table: "deck_versions", ID: "ver-new", Data: map[string]any{"id": "ver-new", "deck_id": "deck-1", "version_name": "New", "updated_at": "2026-01-02T00:00:00Z"}},
		{Bucket: "b1", OpID: "3", Op: "PUT", Table: "deck_versions", ID: "ver-old", Data: map[string]any{"id": "ver-old", "deck_id": "deck-1", "version_name": "Old", "updated_at": "2026-01-01T00:00:00Z"}},
		{Bucket: "b1", OpID: "4", Op: "PUT", Table: "deck_cards", ID: "new-main", Data: map[string]any{"id": "new-main", "deck_id": "deck-1", "deck_version_id": "ver-new", "card_id": "c-main", "board_type": "mainboard", "quantity": 3}},
		{Bucket: "b1", OpID: "5", Op: "PUT", Table: "deck_cards", ID: "new-side", Data: map[string]any{"id": "new-side", "deck_id": "deck-1", "deck_version_id": "ver-new", "card_id": "c-side", "board_type": "inventory", "quantity": 2}},
		{Bucket: "b1", OpID: "6", Op: "PUT", Table: "deck_cards", ID: "old-main", Data: map[string]any{"id": "old-main", "deck_id": "deck-1", "deck_version_id": "ver-old", "card_id": "c-old", "board_type": "mainboard", "quantity": 10}},
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	decks, err := store.ListDecks(ctx, DeckListParams{ViewerUserID: "viewer"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(decks) != 1 {
		t.Fatalf("decks len = %d, want 1", len(decks))
	}
	if decks[0].SelectedVersionID != "ver-new" || decks[0].SelectedVersionName != "New" {
		t.Fatalf("selected version = %#v, want ver-new/New", decks[0])
	}
	if decks[0].TotalQuantity != 5 || decks[0].CardRowCount != 2 {
		t.Fatalf("list counts = %#v, want selected version only", decks[0])
	}

	details, err := store.GetDeckDetails(ctx, "deck-1", "")
	if err != nil {
		t.Fatalf("details: %v", err)
	}
	if details.SelectedVersionID != "ver-new" || details.TotalQuantity != 5 || details.MainboardCount != 3 || details.SideboardCount != 2 {
		t.Fatalf("details = %#v", details)
	}

	old, err := store.GetDeckDetails(ctx, "deck-1", "ver-old")
	if err != nil {
		t.Fatalf("old details: %v", err)
	}
	if old.SelectedVersionID != "ver-old" || old.TotalQuantity != 10 || len(old.Mainboard) != 1 {
		t.Fatalf("old details = %#v", old)
	}
}

func TestStoreViewsAcceptNestedPowerSyncDataAliases(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := openTestStore(t)
	defer store.Close()

	err := store.ApplyOperations(ctx, "cp-1", []Operation{
		{Bucket: "b1", OpID: "1", Op: "PUT", Table: "decks", ID: "deck-1", Data: map[string]any{"data": map[string]any{"name": "Nested Bravo", "active_version_id": "ver-1", "hero_id": "bravo"}}},
		{Bucket: "b1", OpID: "2", Op: "PUT", Table: "deck_versions", ID: "ver-1", Data: map[string]any{"values": map[string]any{"deck_id": "deck-1", "version_name": "Current"}}},
		{Bucket: "b1", OpID: "3", Op: "PUT", Table: "deck_cards", ID: "card-1", Data: map[string]any{"data": map[string]any{"deck_version_id": "ver-1", "card_id": "c1", "board_type": "mainboard", "quantity": 1}}},
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	details, err := store.GetDeckDetails(ctx, "deck-1", "")
	if err != nil {
		t.Fatalf("details: %v", err)
	}
	if details.Deck.Name != "Nested Bravo" || details.SelectedVersionName != "Current" || details.TotalQuantity != 1 {
		t.Fatalf("nested details = %#v", details)
	}
}

func openTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := OpenStore(t.TempDir() + "/sync.sqlite")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return store
}
