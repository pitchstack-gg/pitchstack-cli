package powersync

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	clientv1 "github.com/pitchstack-gg/pitchstack-go/client/v1"
	_ "modernc.org/sqlite"
)

type Store struct {
	path string
	db   *sql.DB
}

func OpenStore(path string) (*Store, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("store path must not be empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create sync cache dir: %w", err)
	}
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	s := &Store{path: path, db: db}
	if err := s.Migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Path() string { return s.path }

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) Migrate(ctx context.Context) error {
	if s == nil || s.db == nil {
		return errors.New("store is not open")
	}
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS ps_meta (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS ps_buckets (
			bucket_id TEXT PRIMARY KEY,
			last_op_id TEXT NOT NULL DEFAULT '',
			checksum TEXT NOT NULL DEFAULT '',
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS ps_rows (
			table_name TEXT NOT NULL,
			row_id TEXT NOT NULL,
			data TEXT NOT NULL,
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (table_name, row_id)
		)`,
		`CREATE TABLE IF NOT EXISTS ps_row_buckets (
			table_name TEXT NOT NULL,
			row_id TEXT NOT NULL,
			bucket_id TEXT NOT NULL,
			PRIMARY KEY (table_name, row_id, bucket_id)
		)`,
		`CREATE TABLE IF NOT EXISTS ps_crud (
			op_id INTEGER PRIMARY KEY AUTOINCREMENT,
			tx_id INTEGER,
			op TEXT NOT NULL,
			table_name TEXT NOT NULL,
			row_id TEXT NOT NULL,
			data TEXT,
			old TEXT,
			metadata TEXT,
			status TEXT NOT NULL DEFAULT 'pending',
			error TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS ps_rows_table_idx ON ps_rows(table_name)`,
		`CREATE INDEX IF NOT EXISTS ps_crud_status_idx ON ps_crud(status, op_id)`,
		`DROP VIEW IF EXISTS collections`,
		`DROP VIEW IF EXISTS collection_items`,
		`DROP VIEW IF EXISTS decks`,
		`DROP VIEW IF EXISTS deck_versions`,
		`DROP VIEW IF EXISTS deck_version_cards`,
		`CREATE VIEW IF NOT EXISTS collections AS
			SELECT row_id AS id,
				COALESCE(json_extract(data, '$.name'), '') AS name,
				COALESCE(json_extract(data, '$.description'), '') AS description,
				COALESCE(json_extract(data, '$.ownerId'), json_extract(data, '$.owner_id'), '') AS owner_id,
				COALESCE(json_extract(data, '$.collectionType'), json_extract(data, '$.collection_type'), '') AS collection_type,
				COALESCE(json_extract(data, '$.visibility'), '') AS visibility,
				COALESCE(json_extract(data, '$.createdAt'), json_extract(data, '$.created_at'), '') AS created_at,
				COALESCE(json_extract(data, '$.updatedAt'), json_extract(data, '$.updated_at'), '') AS updated_at,
				data AS raw_json
			FROM ps_rows WHERE table_name = 'collections'`,
		`CREATE VIEW IF NOT EXISTS collection_items AS
			SELECT row_id AS id,
				COALESCE(json_extract(data, '$.collectionId'), json_extract(data, '$.collection_id'), '') AS collection_id,
				COALESCE(json_extract(data, '$.ownerId'), json_extract(data, '$.owner_id'), '') AS owner_id,
				COALESCE(json_extract(data, '$.productId'), json_extract(data, '$.product_id'), '') AS product_id,
				CAST(COALESCE(json_extract(data, '$.quantity'), 0) AS INTEGER) AS quantity,
				COALESCE(json_extract(data, '$.condition'), '') AS condition,
				COALESCE(json_extract(data, '$.cardId'), json_extract(data, '$.card_id'), '') AS card_id,
				COALESCE(json_extract(data, '$.printingId'), json_extract(data, '$.printing_id'), '') AS printing_id,
				COALESCE(json_extract(data, '$.createdAt'), json_extract(data, '$.created_at'), '') AS created_at,
				COALESCE(json_extract(data, '$.updatedAt'), json_extract(data, '$.updated_at'), '') AS updated_at,
				data AS raw_json
			FROM ps_rows WHERE table_name = 'collection_items'`,
		`CREATE VIEW IF NOT EXISTS decks AS
			SELECT row_id AS id,
				COALESCE(json_extract(data, '$.userId'), json_extract(data, '$.user_id'), json_extract(data, '$.data.userId'), json_extract(data, '$.data.user_id'), json_extract(data, '$.values.userId'), json_extract(data, '$.values.user_id'), '') AS user_id,
				COALESCE(json_extract(data, '$.name'), json_extract(data, '$.data.name'), json_extract(data, '$.values.name'), '') AS name,
				COALESCE(json_extract(data, '$.author'), json_extract(data, '$.data.author'), json_extract(data, '$.values.author'), '') AS author,
				COALESCE(json_extract(data, '$.heroId'), json_extract(data, '$.hero_id'), json_extract(data, '$.data.heroId'), json_extract(data, '$.data.hero_id'), json_extract(data, '$.values.heroId'), json_extract(data, '$.values.hero_id'), '') AS hero_id,
				COALESCE(json_extract(data, '$.format'), json_extract(data, '$.data.format'), json_extract(data, '$.values.format'), '') AS format,
				COALESCE(json_extract(data, '$.visibility'), json_extract(data, '$.data.visibility'), json_extract(data, '$.values.visibility'), '') AS visibility,
				COALESCE(json_extract(data, '$.activeDeckVersionId'), json_extract(data, '$.active_deck_version_id'), json_extract(data, '$.activeVersionId'), json_extract(data, '$.active_version_id'), json_extract(data, '$.data.activeDeckVersionId'), json_extract(data, '$.data.active_deck_version_id'), json_extract(data, '$.data.activeVersionId'), json_extract(data, '$.data.active_version_id'), json_extract(data, '$.values.activeVersionId'), json_extract(data, '$.values.active_version_id'), '') AS active_deck_version_id,
				COALESCE(json_extract(data, '$.deckKind'), json_extract(data, '$.deck_kind'), json_extract(data, '$.data.deckKind'), json_extract(data, '$.data.deck_kind'), json_extract(data, '$.values.deckKind'), json_extract(data, '$.values.deck_kind'), '') AS deck_kind,
				COALESCE(json_extract(data, '$.sourceKind'), json_extract(data, '$.source_kind'), json_extract(data, '$.data.sourceKind'), json_extract(data, '$.data.source_kind'), json_extract(data, '$.values.sourceKind'), json_extract(data, '$.values.source_kind'), '') AS source_kind,
				COALESCE(json_extract(data, '$.sourceReference'), json_extract(data, '$.source_reference'), json_extract(data, '$.data.sourceReference'), json_extract(data, '$.data.source_reference'), json_extract(data, '$.values.sourceReference'), json_extract(data, '$.values.source_reference'), '') AS source_reference,
				COALESCE(json_extract(data, '$.createdAt'), json_extract(data, '$.created_at'), json_extract(data, '$.data.createdAt'), json_extract(data, '$.data.created_at'), json_extract(data, '$.values.createdAt'), json_extract(data, '$.values.created_at'), '') AS created_at,
				COALESCE(json_extract(data, '$.updatedAt'), json_extract(data, '$.updated_at'), json_extract(data, '$.data.updatedAt'), json_extract(data, '$.data.updated_at'), json_extract(data, '$.values.updatedAt'), json_extract(data, '$.values.updated_at'), '') AS updated_at,
				data AS raw_json
			FROM ps_rows WHERE table_name = 'decks'`,
		`CREATE VIEW IF NOT EXISTS deck_versions AS
			SELECT row_id AS id,
				COALESCE(json_extract(data, '$.deckId'), json_extract(data, '$.deck_id'), json_extract(data, '$.data.deckId'), json_extract(data, '$.data.deck_id'), json_extract(data, '$.values.deckId'), json_extract(data, '$.values.deck_id'), '') AS deck_id,
				COALESCE(json_extract(data, '$.name'), json_extract(data, '$.versionName'), json_extract(data, '$.version_name'), json_extract(data, '$.data.name'), json_extract(data, '$.data.versionName'), json_extract(data, '$.data.version_name'), json_extract(data, '$.values.versionName'), json_extract(data, '$.values.version_name'), '') AS name,
				COALESCE(json_extract(data, '$.notes'), json_extract(data, '$.data.notes'), json_extract(data, '$.values.notes'), '') AS notes,
				COALESCE(json_extract(data, '$.createdAt'), json_extract(data, '$.created_at'), json_extract(data, '$.data.createdAt'), json_extract(data, '$.data.created_at'), json_extract(data, '$.values.createdAt'), json_extract(data, '$.values.created_at'), '') AS created_at,
				COALESCE(json_extract(data, '$.updatedAt'), json_extract(data, '$.updated_at'), json_extract(data, '$.data.updatedAt'), json_extract(data, '$.data.updated_at'), json_extract(data, '$.values.updatedAt'), json_extract(data, '$.values.updated_at'), '') AS updated_at,
				data AS raw_json
			FROM ps_rows WHERE table_name = 'deck_versions'`,
		`CREATE VIEW IF NOT EXISTS deck_version_cards AS
			SELECT row_id AS id,
				COALESCE(json_extract(data, '$.deckId'), json_extract(data, '$.deck_id'), json_extract(data, '$.data.deckId'), json_extract(data, '$.data.deck_id'), json_extract(data, '$.values.deckId'), json_extract(data, '$.values.deck_id'), '') AS deck_id,
				COALESCE(json_extract(data, '$.deckVersionId'), json_extract(data, '$.deck_version_id'), json_extract(data, '$.data.deckVersionId'), json_extract(data, '$.data.deck_version_id'), json_extract(data, '$.values.deckVersionId'), json_extract(data, '$.values.deck_version_id'), '') AS deck_version_id,
				COALESCE(json_extract(data, '$.cardId'), json_extract(data, '$.card_id'), json_extract(data, '$.data.cardId'), json_extract(data, '$.data.card_id'), json_extract(data, '$.values.cardId'), json_extract(data, '$.values.card_id'), '') AS card_id,
				COALESCE(json_extract(data, '$.boardType'), json_extract(data, '$.board_type'), json_extract(data, '$.board'), json_extract(data, '$.data.boardType'), json_extract(data, '$.data.board_type'), json_extract(data, '$.data.board'), json_extract(data, '$.values.boardType'), json_extract(data, '$.values.board_type'), '') AS board_type,
				CAST(COALESCE(json_extract(data, '$.quantity'), json_extract(data, '$.data.quantity'), json_extract(data, '$.values.quantity'), 0) AS INTEGER) AS quantity,
				COALESCE(json_extract(data, '$.createdAt'), json_extract(data, '$.created_at'), json_extract(data, '$.data.createdAt'), json_extract(data, '$.data.created_at'), json_extract(data, '$.values.createdAt'), json_extract(data, '$.values.created_at'), '') AS created_at,
				COALESCE(json_extract(data, '$.updatedAt'), json_extract(data, '$.updated_at'), json_extract(data, '$.data.updatedAt'), json_extract(data, '$.data.updated_at'), json_extract(data, '$.values.updatedAt'), json_extract(data, '$.values.updated_at'), '') AS updated_at,
				data AS raw_json
			FROM ps_rows WHERE table_name IN ('deck_version_cards', 'deck_cards')`,
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, stmt := range stmts {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO ps_meta(key, value) VALUES('schema_version', ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, schemaVersion); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) EnsureDeviceID(ctx context.Context) (string, error) {
	if v, err := s.meta(ctx, "device_id"); err != nil {
		return "", err
	} else if strings.TrimSpace(v) != "" {
		return v, nil
	}
	id := "cli-" + uuid.NewString()
	return id, s.setMeta(ctx, "device_id", id)
}

func (s *Store) EnsureSyncEpoch(ctx context.Context, epoch string) (bool, error) {
	epoch = strings.TrimSpace(epoch)
	if epoch == "" {
		return false, nil
	}
	cur, err := s.meta(ctx, "sync_epoch")
	if err != nil {
		return false, err
	}
	if strings.TrimSpace(cur) == "" {
		return false, s.setMeta(ctx, "sync_epoch", epoch)
	}
	if cur == epoch {
		return false, nil
	}
	if err := s.ResetSyncState(ctx); err != nil {
		return false, err
	}
	return true, s.setMeta(ctx, "sync_epoch", epoch)
}

func (s *Store) ResetSyncState(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, stmt := range []string{
		`DELETE FROM ps_buckets`,
		`DELETE FROM ps_row_buckets`,
		`DELETE FROM ps_rows`,
		`DELETE FROM ps_crud`,
		`DELETE FROM ps_meta WHERE key IN ('last_checkpoint', 'last_write_checkpoint', 'last_successful_sync')`,
	} {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) BucketPositions(ctx context.Context) ([]BucketPosition, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT bucket_id, last_op_id FROM ps_buckets ORDER BY bucket_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BucketPosition
	for rows.Next() {
		var p BucketPosition
		if err := rows.Scan(&p.Bucket, &p.OpID); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) ApplyOperations(ctx context.Context, checkpoint string, ops []Operation) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, op := range ops {
		if err := applyOperation(ctx, tx, op); err != nil {
			return err
		}
	}
	if strings.TrimSpace(checkpoint) != "" {
		if _, err := tx.ExecContext(ctx, `INSERT INTO ps_meta(key, value) VALUES('last_checkpoint', ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, checkpoint); err != nil {
			return err
		}
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx, `INSERT INTO ps_meta(key, value) VALUES('last_successful_sync', ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, now); err != nil {
		return err
	}
	return tx.Commit()
}

func applyOperation(ctx context.Context, tx *sql.Tx, op Operation) error {
	table := normalizeTable(op.Table)
	id := strings.TrimSpace(op.ID)
	if table == "" || id == "" {
		return nil
	}
	bucket := strings.TrimSpace(op.Bucket)
	switch strings.ToUpper(strings.TrimSpace(op.Op)) {
	case "PUT", "PATCH", "UPSERT", "":
		data := op.Data
		if data == nil {
			data = map[string]any{}
		}
		if _, ok := data["id"]; !ok {
			data["id"] = id
		}
		raw, err := json.Marshal(data)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO ps_rows(table_name, row_id, data, updated_at) VALUES(?, ?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT(table_name, row_id) DO UPDATE SET data = excluded.data, updated_at = CURRENT_TIMESTAMP`, table, id, string(raw)); err != nil {
			return err
		}
		if bucket != "" {
			if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO ps_row_buckets(table_name, row_id, bucket_id) VALUES(?, ?, ?)`, table, id, bucket); err != nil {
				return err
			}
		}
	case "REMOVE", "DELETE":
		if bucket != "" {
			if _, err := tx.ExecContext(ctx, `DELETE FROM ps_row_buckets WHERE table_name = ? AND row_id = ? AND bucket_id = ?`, table, id, bucket); err != nil {
				return err
			}
			var remaining int
			if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM ps_row_buckets WHERE table_name = ? AND row_id = ?`, table, id).Scan(&remaining); err != nil {
				return err
			}
			if remaining > 0 {
				break
			}
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM ps_rows WHERE table_name = ? AND row_id = ?`, table, id); err != nil {
			return err
		}
	}
	if bucket != "" {
		if _, err := tx.ExecContext(ctx, `INSERT INTO ps_buckets(bucket_id, last_op_id, checksum, updated_at) VALUES(?, ?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT(bucket_id) DO UPDATE SET last_op_id = excluded.last_op_id, checksum = COALESCE(NULLIF(excluded.checksum, ''), ps_buckets.checksum), updated_at = CURRENT_TIMESTAMP`,
			bucket, strings.TrimSpace(op.OpID), strings.TrimSpace(op.Checksum)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) PutLocal(ctx context.Context, table, id string, data map[string]any) error {
	table = normalizeTable(table)
	id = strings.TrimSpace(id)
	if table == "" || id == "" {
		return errors.New("table and id are required")
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `INSERT INTO ps_rows(table_name, row_id, data, updated_at) VALUES(?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(table_name, row_id) DO UPDATE SET data = excluded.data, updated_at = CURRENT_TIMESTAMP`, table, id, string(raw)); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO ps_crud(op, table_name, row_id, data) VALUES('PUT', ?, ?, ?)`, table, id, string(raw)); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) DeleteLocal(ctx context.Context, table, id string) error {
	table = normalizeTable(table)
	id = strings.TrimSpace(id)
	if table == "" || id == "" {
		return errors.New("table and id are required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM ps_rows WHERE table_name = ? AND row_id = ?`, table, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO ps_crud(op, table_name, row_id) VALUES('DELETE', ?, ?)`, table, id); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) NextCrudBatch(ctx context.Context, limit int) ([]clientv1.CrudEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `SELECT op_id, tx_id, op, table_name, row_id, data, old, metadata FROM ps_crud WHERE status = 'pending' ORDER BY COALESCE(tx_id, op_id), op_id LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []clientv1.CrudEntry
	var firstGroup string
	for rows.Next() {
		var opID int64
		var txID sql.NullInt64
		var op, table, rowID string
		var dataRaw, oldRaw, metadata sql.NullString
		if err := rows.Scan(&opID, &txID, &op, &table, &rowID, &dataRaw, &oldRaw, &metadata); err != nil {
			return nil, err
		}
		group := strconv.FormatInt(opID, 10)
		if txID.Valid {
			group = "tx:" + strconv.FormatInt(txID.Int64, 10)
		}
		if firstGroup == "" {
			firstGroup = group
		} else if group != firstGroup {
			break
		}
		entry := clientv1.CrudEntry{
			OpID: opID,
			Op:   op,
			Type: table,
			ID:   rowID,
		}
		if txID.Valid {
			v := txID.Int64
			entry.TxID = &v
		}
		if dataRaw.Valid && strings.TrimSpace(dataRaw.String) != "" {
			_ = json.Unmarshal([]byte(dataRaw.String), &entry.Data)
		}
		if oldRaw.Valid && strings.TrimSpace(oldRaw.String) != "" {
			_ = json.Unmarshal([]byte(oldRaw.String), &entry.Old)
		}
		if metadata.Valid {
			v := metadata.String
			entry.Metadata = &v
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (s *Store) MarkCrudUploaded(ctx context.Context, results []clientv1.UploadCrudResult, writeCheckpoint string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, result := range results {
		opID := strings.TrimSpace(result.OpID)
		if opID == "" {
			continue
		}
		switch result.Status {
		case clientv1.SyncStatusOK:
			if _, err := tx.ExecContext(ctx, `UPDATE ps_crud SET status = 'uploaded', error = '', updated_at = CURRENT_TIMESTAMP WHERE op_id = ?`, opID); err != nil {
				return err
			}
		case clientv1.SyncStatusConflict, clientv1.SyncStatusError:
			if _, err := tx.ExecContext(ctx, `UPDATE ps_crud SET status = 'failed', error = ?, updated_at = CURRENT_TIMESTAMP WHERE op_id = ?`, strings.TrimSpace(result.ErrorMessage), opID); err != nil {
				return err
			}
		}
	}
	if strings.TrimSpace(writeCheckpoint) != "" {
		if _, err := tx.ExecContext(ctx, `INSERT INTO ps_meta(key, value) VALUES('last_write_checkpoint', ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, strings.TrimSpace(writeCheckpoint)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) CollectionCounts(ctx context.Context) ([]CollectionCount, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT ci.collection_id, COALESCE(c.name, ''), COUNT(ci.id), COALESCE(SUM(ci.quantity), 0), COUNT(DISTINCT NULLIF(ci.card_id, ''))
		FROM collection_items ci
		LEFT JOIN collections c ON c.id = ci.collection_id
		GROUP BY ci.collection_id, c.name
		ORDER BY c.name COLLATE NOCASE, ci.collection_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CollectionCount
	for rows.Next() {
		var c CollectionCount
		if err := rows.Scan(&c.CollectionID, &c.Name, &c.ItemCount, &c.QuantityCount, &c.UniqueCardCount); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) ListDecks(ctx context.Context, params DeckListParams) ([]DeckSummary, error) {
	scope := params.Scope
	if scope == "" {
		scope = DeckListScopeAccessible
	}
	aliases := normalizedUserAliases(params.ViewerUserID, params.ViewerUserAliases)

	where := []string{
		`d.id <> ''`,
		`UPPER(COALESCE(NULLIF(d.deck_kind, ''), 'DECK_KIND_USER')) NOT IN ('DECK_KIND_REFERENCE', 'REFERENCE')`,
	}
	args := []any{}
	switch scope {
	case DeckListScopeOwned:
		if len(aliases) == 0 {
			return []DeckSummary{}, nil
		}
		where = append(where, `d.user_id IN (`+placeholders(len(aliases))+`)`)
		for _, alias := range aliases {
			args = append(args, alias)
		}
	case DeckListScopeShared:
		if len(aliases) == 0 {
			return []DeckSummary{}, nil
		}
		where = append(where, `d.user_id <> '' AND d.user_id NOT IN (`+placeholders(len(aliases))+`)`)
		for _, alias := range aliases {
			args = append(args, alias)
		}
	case DeckListScopeAccessible:
	default:
		return nil, fmt.Errorf("unsupported deck list scope %q", scope)
	}

	search := strings.ToLower(strings.TrimSpace(params.Search))
	if search != "" {
		where = append(where, `(LOWER(d.name) LIKE ? OR LOWER(d.author) LIKE ? OR LOWER(d.hero_id) LIKE ? OR LOWER(d.format) LIKE ?)`)
		like := "%" + search + "%"
		args = append(args, like, like, like, like)
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 500
	}
	query := `WITH ranked_versions AS (
			SELECT id, deck_id, name,
				CASE WHEN deck_id LIKE 'd-%' THEN substr(deck_id, 3) ELSE deck_id END AS deck_key,
				ROW_NUMBER() OVER (
					PARTITION BY CASE WHEN deck_id LIKE 'd-%' THEN substr(deck_id, 3) ELSE deck_id END
					ORDER BY NULLIF(updated_at, '') DESC, NULLIF(created_at, '') DESC, name COLLATE NOCASE ASC, id ASC
				) AS rn
			FROM deck_versions
		), version_counts AS (
			SELECT CASE WHEN deck_id LIKE 'd-%' THEN substr(deck_id, 3) ELSE deck_id END AS deck_key, COUNT(*) AS version_count
			FROM deck_versions
			GROUP BY deck_key
		), resolved_decks AS (
			SELECT d.*,
				av.id AS active_version_actual_id,
				COALESCE(av.id, rv.id, '') AS selected_version_id
			FROM decks d
			LEFT JOIN ranked_versions rv ON rv.deck_key = CASE WHEN d.id LIKE 'd-%' THEN substr(d.id, 3) ELSE d.id END AND rv.rn = 1
			LEFT JOIN deck_versions av ON av.id = d.active_deck_version_id
				OR ('dv-' || av.id) = d.active_deck_version_id
				OR av.id = CASE WHEN d.active_deck_version_id LIKE 'dv-%' THEN substr(d.active_deck_version_id, 4) ELSE d.active_deck_version_id END
		), active_counts AS (
			SELECT deck_version_id, COUNT(*) AS card_row_count, COALESCE(SUM(CASE WHEN quantity > 0 THEN quantity ELSE 1 END), 0) AS total_quantity
			FROM deck_version_cards
			GROUP BY deck_version_id
		)
		SELECT d.id, d.user_id, d.name, d.author, d.hero_id, d.format, d.visibility,
			d.active_deck_version_id, COALESCE(av.name, ''), d.selected_version_id, COALESCE(sv.name, ''), d.deck_kind, d.source_kind, d.source_reference,
			d.created_at, d.updated_at, COALESCE(vc.version_count, 0), COALESCE(ac.card_row_count, 0), COALESCE(ac.total_quantity, 0)
		FROM resolved_decks d
		LEFT JOIN deck_versions av ON av.id = d.active_version_actual_id
		LEFT JOIN deck_versions sv ON sv.id = d.selected_version_id
		LEFT JOIN version_counts vc ON vc.deck_key = CASE WHEN d.id LIKE 'd-%' THEN substr(d.id, 3) ELSE d.id END
		LEFT JOIN active_counts ac ON ac.deck_version_id = d.selected_version_id
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY NULLIF(d.updated_at, '') DESC, NULLIF(d.created_at, '') DESC, d.name COLLATE NOCASE ASC, d.id ASC
		LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []DeckSummary
	activeVersionIDs := []string{}
	for rows.Next() {
		var deck DeckSummary
		if err := rows.Scan(
			&deck.ID,
			&deck.UserID,
			&deck.Name,
			&deck.Author,
			&deck.HeroID,
			&deck.Format,
			&deck.Visibility,
			&deck.ActiveVersionID,
			&deck.ActiveVersionName,
			&deck.SelectedVersionID,
			&deck.SelectedVersionName,
			&deck.DeckKind,
			&deck.SourceKind,
			&deck.SourceReference,
			&deck.CreatedAt,
			&deck.UpdatedAt,
			&deck.VersionCount,
			&deck.CardRowCount,
			&deck.TotalQuantity,
		); err != nil {
			return nil, err
		}
		if userIDInAliases(deck.UserID, aliases) {
			deck.Ownership = "owned"
		} else if strings.TrimSpace(deck.UserID) != "" {
			deck.Ownership = "shared"
		}
		out = append(out, deck)
		if strings.TrimSpace(deck.SelectedVersionID) != "" {
			activeVersionIDs = append(activeVersionIDs, deck.SelectedVersionID)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	boards, err := s.deckBoardCounts(ctx, activeVersionIDs)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].ActiveVersionBoards = boards[out[i].SelectedVersionID]
	}
	return out, nil
}

func (s *Store) deckBoardCounts(ctx context.Context, activeVersionIDs []string) (map[string][]DeckBoardCount, error) {
	ids := uniqueStrings(activeVersionIDs)
	if len(ids) == 0 {
		return map[string][]DeckBoardCount{}, nil
	}
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT deck_version_id,
			CASE
				WHEN LOWER(board_type) LIKE '%side%' THEN 'sideboard'
				WHEN LOWER(board_type) LIKE '%maybe%' THEN 'maybeboard'
				WHEN TRIM(board_type) = '' THEN 'mainboard'
				ELSE LOWER(board_type)
			END AS normalized_board,
			COUNT(*) AS card_row_count,
			COALESCE(SUM(CASE WHEN quantity > 0 THEN quantity ELSE 1 END), 0) AS total_quantity
		FROM deck_version_cards
		WHERE deck_version_id IN (`+placeholders(len(ids))+`)
		GROUP BY deck_version_id, normalized_board
		ORDER BY deck_version_id,
			CASE normalized_board WHEN 'mainboard' THEN 0 WHEN 'sideboard' THEN 1 WHEN 'maybeboard' THEN 2 ELSE 3 END,
			normalized_board`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]DeckBoardCount{}
	for rows.Next() {
		var versionID string
		var count DeckBoardCount
		if err := rows.Scan(&versionID, &count.BoardType, &count.CardRowCount, &count.TotalQuantity); err != nil {
			return nil, err
		}
		out[versionID] = append(out[versionID], count)
	}
	return out, rows.Err()
}

func (s *Store) GetDeckDetails(ctx context.Context, deckID, selectedVersionID string) (*DeckDetails, error) {
	deckID = strings.TrimSpace(deckID)
	if deckID == "" {
		return nil, errors.New("deck id is required")
	}
	deckAliases := resourceIDAliases(deckID, "d")
	args := make([]any, 0, len(deckAliases))
	for _, alias := range deckAliases {
		args = append(args, alias)
	}
	var deck DeckSummary
	err := s.db.QueryRowContext(ctx, `SELECT id, user_id, name, author, hero_id, format, visibility,
			active_deck_version_id, deck_kind, source_kind, source_reference, created_at, updated_at
		FROM decks
		WHERE id IN (`+placeholders(len(deckAliases))+`)
		ORDER BY CASE WHEN id = ? THEN 0 ELSE 1 END
		LIMIT 1`, append(args, deckID)...).Scan(
		&deck.ID,
		&deck.UserID,
		&deck.Name,
		&deck.Author,
		&deck.HeroID,
		&deck.Format,
		&deck.Visibility,
		&deck.ActiveVersionID,
		&deck.DeckKind,
		&deck.SourceKind,
		&deck.SourceReference,
		&deck.CreatedAt,
		&deck.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	versions, err := s.deckVersions(ctx, resourceIDAliases(deck.ID, "d"))
	if err != nil {
		return nil, err
	}
	deck.VersionCount = int64(len(versions))
	selected := resolveDeckVersion(versions, selectedVersionID, deck.ActiveVersionID)
	if selected.ID != "" {
		deck.SelectedVersionID = selected.ID
		deck.SelectedVersionName = selected.Name
		if strings.TrimSpace(deck.ActiveVersionID) == selected.ID {
			deck.ActiveVersionName = selected.Name
		}
	}

	details := &DeckDetails{
		Deck:                deck,
		Versions:            versions,
		SelectedVersionID:   selected.ID,
		SelectedVersionName: selected.Name,
	}
	if selected.ID == "" {
		return details, nil
	}

	cards, err := s.deckCardsForVersion(ctx, resourceIDAliases(selected.ID, "dv"))
	if err != nil {
		return nil, err
	}
	for _, card := range cards {
		if card.Quantity <= 0 {
			card.Quantity = 1
		}
		details.CardRowCount++
		details.TotalQuantity += card.Quantity
		switch normalizeDeckBoardType(card.BoardType) {
		case "hero_equipment":
			details.HeroEquipment = append(details.HeroEquipment, card)
			details.HeroEquipmentCount += card.Quantity
		case "sideboard":
			details.Sideboard = append(details.Sideboard, card)
			details.SideboardCount += card.Quantity
		case "maybeboard":
			details.Maybeboard = append(details.Maybeboard, card)
			details.MaybeboardCount += card.Quantity
		case "mainboard":
			details.Mainboard = append(details.Mainboard, card)
			details.MainboardCount += card.Quantity
		default:
			details.Other = append(details.Other, card)
			details.OtherCount += card.Quantity
		}
	}
	details.Deck.CardRowCount = details.CardRowCount
	details.Deck.TotalQuantity = details.TotalQuantity
	details.Deck.ActiveVersionBoards = []DeckBoardCount{
		{BoardType: "hero/equipment", CardRowCount: int64(len(details.HeroEquipment)), TotalQuantity: details.HeroEquipmentCount},
		{BoardType: "mainboard", CardRowCount: int64(len(details.Mainboard)), TotalQuantity: details.MainboardCount},
		{BoardType: "sideboard", CardRowCount: int64(len(details.Sideboard)), TotalQuantity: details.SideboardCount},
		{BoardType: "maybeboard", CardRowCount: int64(len(details.Maybeboard)), TotalQuantity: details.MaybeboardCount},
	}
	return details, nil
}

func (s *Store) deckVersions(ctx context.Context, deckAliases []string) ([]DeckVersionSummary, error) {
	deckAliases = uniqueStrings(deckAliases)
	if len(deckAliases) == 0 {
		return nil, nil
	}
	args := make([]any, 0, len(deckAliases))
	for _, alias := range deckAliases {
		args = append(args, alias)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, deck_id, name, notes, created_at, updated_at
		FROM deck_versions
		WHERE deck_id IN (`+placeholders(len(deckAliases))+`)
		ORDER BY NULLIF(updated_at, '') DESC, NULLIF(created_at, '') DESC, name COLLATE NOCASE ASC, id ASC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DeckVersionSummary
	for rows.Next() {
		var v DeckVersionSummary
		if err := rows.Scan(&v.ID, &v.DeckID, &v.Name, &v.Notes, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) deckCardsForVersion(ctx context.Context, versionAliases []string) ([]DeckCardLine, error) {
	versionAliases = uniqueStrings(versionAliases)
	if len(versionAliases) == 0 {
		return nil, nil
	}
	args := make([]any, 0, len(versionAliases))
	for _, alias := range versionAliases {
		args = append(args, alias)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, deck_id, deck_version_id, card_id, board_type, quantity, created_at, updated_at
		FROM deck_version_cards
		WHERE deck_version_id IN (`+placeholders(len(versionAliases))+`)
		ORDER BY
			CASE
				WHEN LOWER(board_type) LIKE '%hero%' OR LOWER(board_type) LIKE '%equipment%' THEN 0
				WHEN LOWER(board_type) LIKE '%main%' OR TRIM(board_type) = '' THEN 1
				WHEN LOWER(board_type) LIKE '%inventory%' OR LOWER(board_type) LIKE '%side%' THEN 2
				WHEN LOWER(board_type) LIKE '%maybe%' THEN 3
				ELSE 4
			END,
			card_id COLLATE NOCASE ASC, id ASC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DeckCardLine
	for rows.Next() {
		var c DeckCardLine
		if err := rows.Scan(&c.ID, &c.DeckID, &c.DeckVersionID, &c.CardID, &c.BoardType, &c.Quantity, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func resolveDeckVersion(versions []DeckVersionSummary, requestedID, activeID string) DeckVersionSummary {
	for _, candidate := range []string{requestedID, activeID} {
		aliases := resourceIDAliases(candidate, "dv")
		for _, version := range versions {
			for _, alias := range aliases {
				if version.ID == alias {
					return version
				}
			}
		}
	}
	if len(versions) > 0 {
		return versions[0]
	}
	return DeckVersionSummary{}
}

func normalizeDeckBoardType(board string) string {
	board = strings.ToLower(strings.TrimSpace(board))
	board = strings.ReplaceAll(board, "-", "_")
	switch {
	case board == "":
		return "mainboard"
	case strings.Contains(board, "hero") || strings.Contains(board, "equipment"):
		return "hero_equipment"
	case strings.Contains(board, "inventory") || strings.Contains(board, "side"):
		return "sideboard"
	case strings.Contains(board, "maybe"):
		return "maybeboard"
	case strings.Contains(board, "main"):
		return "mainboard"
	default:
		return board
	}
}

func resourceIDAliases(id, prefix string) []string {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	prefix = strings.Trim(strings.TrimSpace(prefix), "-")
	candidates := []string{id}
	if prefix != "" {
		prefixValue := prefix + "-"
		if strings.HasPrefix(id, prefixValue) {
			candidates = append(candidates, strings.TrimPrefix(id, prefixValue))
		} else {
			candidates = append(candidates, prefixValue+id)
		}
	}
	return uniqueStrings(candidates)
}

func (s *Store) HasIncompleteDeckData(ctx context.Context) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*)
		FROM ps_rows
		WHERE table_name = 'decks'
			AND COALESCE(json_extract(data, '$.name'), json_extract(data, '$.data.name'), json_extract(data, '$.values.name'), '') = ''
			AND COALESCE(json_extract(data, '$.heroId'), json_extract(data, '$.hero_id'), json_extract(data, '$.data.heroId'), json_extract(data, '$.data.hero_id'), '') = ''
			AND COALESCE(json_extract(data, '$.object_type'), json_extract(data, '$.objectType'), '') <> ''`).Scan(&count)
	return count > 0, err
}

func (s *Store) Status(ctx context.Context) (*Status, error) {
	out := &Status{Path: s.path}
	var err error
	out.SchemaVersion, err = s.meta(ctx, "schema_version")
	if err != nil {
		return nil, err
	}
	out.DeviceID, _ = s.meta(ctx, "device_id")
	out.SyncEpoch, _ = s.meta(ctx, "sync_epoch")
	out.LastCheckpoint, _ = s.meta(ctx, "last_checkpoint")
	out.LastWriteCheckpoint, _ = s.meta(ctx, "last_write_checkpoint")
	if raw, _ := s.meta(ctx, "last_successful_sync"); raw != "" {
		if t, parseErr := time.Parse(time.RFC3339Nano, raw); parseErr == nil {
			out.LastSuccessfulSync = &t
		}
	}
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ps_rows`).Scan(&out.Rows)
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ps_buckets`).Scan(&out.Buckets)
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ps_crud WHERE status = 'pending'`).Scan(&out.PendingCrud)
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ps_crud WHERE status = 'failed'`).Scan(&out.FailedCrud)
	failed, err := s.failedCrudEntries(ctx, 20)
	if err != nil {
		return nil, err
	}
	out.FailedEntries = failed
	return out, nil
}

func (s *Store) failedCrudEntries(ctx context.Context, limit int) ([]CrudError, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT op_id, op, table_name, row_id, error, updated_at FROM ps_crud WHERE status = 'failed' ORDER BY updated_at DESC, op_id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CrudError
	for rows.Next() {
		var entry CrudError
		if err := rows.Scan(&entry.OpID, &entry.Op, &entry.Table, &entry.ID, &entry.Error, &entry.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, entry)
	}
	return out, rows.Err()
}

func (s *Store) meta(ctx context.Context, key string) (string, error) {
	var value string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM ps_meta WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

func (s *Store) setMeta(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO ps_meta(key, value) VALUES(?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

func normalizeTable(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	v = strings.ReplaceAll(v, "-", "_")
	switch v {
	case "collection", "collectionitems", "collection_item":
		if v == "collection" {
			return "collections"
		}
		return "collection_items"
	case "deck":
		return "decks"
	case "deckversion", "deck_version":
		return "deck_versions"
	case "deckcard", "deck_cards", "deck_version_card":
		return "deck_version_cards"
	default:
		return v
	}
}

func normalizedUserAliases(userID string, extra []string) []string {
	values := append([]string{userID}, extra...)
	var out []string
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		for _, candidate := range []string{value, strings.TrimPrefix(value, "u-")} {
			candidate = strings.TrimSpace(candidate)
			if candidate == "" {
				continue
			}
			if _, ok := seen[candidate]; ok {
				continue
			}
			seen[candidate] = struct{}{}
			out = append(out, candidate)
		}
	}
	return out
}

func userIDInAliases(userID string, aliases []string) bool {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return false
	}
	for _, alias := range aliases {
		if userID == alias {
			return true
		}
	}
	return false
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimRight(strings.Repeat("?,", n), ",")
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
