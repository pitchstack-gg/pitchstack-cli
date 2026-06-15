package commands

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pitchstack-gg/pitchstack-cli/internal/paths"

	_ "modernc.org/sqlite"
)

func TestCardsSearchCommandUsesLocalDatabaseOffline(t *testing.T) {
	cfgPath := setupCommandTestProfile(t, "http://127.0.0.1:1")
	installSimpleCommandCardsDB(t, paths.CardsDBPath("test"))

	var stdout, stderr bytes.Buffer
	root := NewRootCommand(strings.NewReader(""), &stdout, &stderr)
	err := root.Run(context.Background(), []string{
		"pitchstack", "--config", cfgPath, "--profile", "test",
		"cards", "search", "--offline", "Alpha",
	})
	if err != nil {
		t.Fatalf("run command: %v; stderr=%s", err, stderr.String())
	}

	var got struct {
		Summaries []struct {
			Identifier string `json:"identifier"`
			Name       string `json:"name"`
		} `json:"summaries"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode stdout %q: %v", stdout.String(), err)
	}
	if len(got.Summaries) != 1 || got.Summaries[0].Identifier != "card-alpha" {
		t.Fatalf("summaries = %#v, stdout=%s", got.Summaries, stdout.String())
	}
	if strings.Contains(stderr.String(), "http://") {
		t.Fatalf("stderr suggests network use: %s", stderr.String())
	}
}

func TestCardsSearchCommandAcceptsShortQueryFlag(t *testing.T) {
	cfgPath := setupCommandTestProfile(t, "http://127.0.0.1:1")
	installSimpleCommandCardsDB(t, paths.CardsDBPath("test"))

	var stdout, stderr bytes.Buffer
	root := NewRootCommand(strings.NewReader(""), &stdout, &stderr)
	err := root.Run(context.Background(), []string{
		"pitchstack", "--config", cfgPath, "--profile", "test",
		"cards", "search", "--offline", "-q", "Alpha",
	})
	if err != nil {
		t.Fatalf("run command: %v; stderr=%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "card-alpha") {
		t.Fatalf("stdout = %q, want card-alpha", stdout.String())
	}
}

func TestCardsSearchCommandRejectsEmptySearch(t *testing.T) {
	cfgPath := setupCommandTestProfile(t, "http://127.0.0.1:1")

	var stdout, stderr bytes.Buffer
	root := NewRootCommand(strings.NewReader(""), &stdout, &stderr)
	err := root.Run(context.Background(), []string{
		"pitchstack", "--config", cfgPath, "--profile", "test",
		"cards", "search",
	})
	if err == nil {
		t.Fatalf("run command error = nil, want empty search error")
	}
	if !strings.Contains(err.Error(), "requires a query") {
		t.Fatalf("error = %v, want query requirement", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
}

func installSimpleCommandCardsDB(t *testing.T, dbPath string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatal(err)
	}
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
);`)
	if err != nil {
		t.Fatal(err)
	}
}
