package powersync

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	clientv1 "github.com/pitchstack-gg/pitchstack-go/client/v1"
)

func TestClientPullOnceConsumesStreamFixture(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := openTestStore(t)
	defer store.Close()

	stream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/sync/stream" {
			t.Fatalf("path = %s, want /sync/stream", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Token token-1" {
			t.Fatalf("authorization = %q, want Token token-1", got)
		}
		if got := r.Header.Get("Accept"); got != "application/x-ndjson" {
			t.Fatalf("accept = %q, want application/x-ndjson", got)
		}
		var payload struct {
			Buckets         []map[string]string `json:"buckets"`
			IncludeChecksum bool                `json:"include_checksum"`
			RawData         bool                `json:"raw_data"`
			ClientID        string              `json:"client_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !payload.IncludeChecksum || !payload.RawData || payload.ClientID == "" {
			t.Fatalf("payload = %#v", payload)
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = w.Write([]byte(`{"checkpoint":{"last_op_id":"cp-1","buckets":[]}}` + "\n"))
		_, _ = w.Write([]byte(`{"data":{"bucket":"b1","data":[{"op_id":1,"op":"PUT","object_type":"collections","object_id":"col-1","data":{"id":"col-1","name":"Binder"}},{"op_id":2,"op":"PUT","object_type":"collection_items","object_id":"item-1","data":{"id":"item-1","collectionId":"col-1","cardId":"card-1","quantity":3}}]}}` + "\n"))
		_, _ = w.Write([]byte(`{"checkpoint_complete":{"last_op_id":"cp-1"}}` + "\n"))
	}))
	defer stream.Close()

	client := &Client{
		Store:      store,
		Connector:  &fixtureConnector{creds: &Credentials{Endpoint: stream.URL, Token: "token-1", SyncEpoch: "epoch-1"}},
		HTTPClient: stream.Client(),
	}
	if err := client.PullOnce(ctx); err != nil {
		t.Fatalf("pull once: %v", err)
	}
	counts, err := store.CollectionCounts(ctx)
	if err != nil {
		t.Fatalf("counts: %v", err)
	}
	if len(counts) != 1 || counts[0].CollectionID != "col-1" || counts[0].QuantityCount != 3 {
		t.Fatalf("counts = %#v", counts)
	}
	status, err := store.Status(ctx)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.LastCheckpoint != "cp-1" || status.Rows != 2 {
		t.Fatalf("status = %#v", status)
	}
}

func TestStreamEndpointNormalizesBaseURL(t *testing.T) {
	t.Parallel()
	if got, want := streamEndpoint("https://sync.example.test/"), "https://sync.example.test/sync/stream"; got != want {
		t.Fatalf("streamEndpoint(base slash) = %q, want %q", got, want)
	}
	if got, want := streamEndpoint("https://sync.example.test/powersync/"), "https://sync.example.test/powersync/sync/stream"; got != want {
		t.Fatalf("streamEndpoint(path) = %q, want %q", got, want)
	}
	if got, want := streamEndpoint("https://sync.example.test/sync/stream/"), "https://sync.example.test/sync/stream"; got != want {
		t.Fatalf("streamEndpoint(full path) = %q, want %q", got, want)
	}
}

func TestOperationFromMapUsesEmbeddedDataAndSkipsProtocolFields(t *testing.T) {
	t.Parallel()
	op := operationFromMap(map[string]any{
		"op_id":       "1",
		"op":          "PUT",
		"object_type": "decks",
		"object_id":   "deck-1",
		"subkey":      "bucket-key",
		"data":        `{"name":"Bravo Blitz","active_version_id":"ver-1"}`,
	}, "b1", "")
	if op.Table != "decks" || op.ID != "deck-1" {
		t.Fatalf("op identity = %#v", op)
	}
	if op.Data["name"] != "Bravo Blitz" || op.Data["active_version_id"] != "ver-1" {
		t.Fatalf("op data = %#v", op.Data)
	}
	if _, ok := op.Data["object_type"]; ok {
		t.Fatalf("protocol object_type leaked into data: %#v", op.Data)
	}
}

type fixtureConnector struct {
	creds    *Credentials
	uploaded []clientv1.CrudEntry
}

func (c *fixtureConnector) FetchCredentials(context.Context) (*Credentials, error) {
	return c.creds, nil
}

func (c *fixtureConnector) UploadCrud(_ context.Context, _ string, entries []clientv1.CrudEntry) (*clientv1.UploadCrudResponse, error) {
	c.uploaded = append(c.uploaded, entries...)
	results := make([]clientv1.UploadCrudResult, 0, len(entries))
	for _, entry := range entries {
		results = append(results, clientv1.UploadCrudResult{OpID: "1", Status: clientv1.SyncStatusOK})
		_ = entry
	}
	return &clientv1.UploadCrudResponse{Results: results, WriteCheckpoint: "wcp-1"}, nil
}
