package episodestore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"loreforge/internal/domain/episode"

	_ "github.com/mattn/go-sqlite3"
)

const (
	defaultDSN               = "./data/universe.db"
	defaultBaseDir           = "./data"
	fileMode                 = 0o600
	dirMode                  = 0o755
	summaryLimit             = 280
	outputTextFileName       = "output.txt"
	promptFileName           = "prompt.txt"
	manifestFileName         = "manifest.json"
	contextFileName          = "context.json"
	providerRequestFileName  = "provider_request.json"
	providerResponseFileName = "provider_response.json"
	scoreFileName            = "score.json"
	publishFileName          = "publish.json"
	presentationFileName     = "presentation.json"
	artistSnapshotFileName   = "artist_snapshot.json"
	outputPartsFileName      = "output_parts.json"
)

var sqliteInitStatements = []string{
	`PRAGMA busy_timeout = 5000;`,
	`PRAGMA journal_mode = WAL;`,
	`PRAGMA synchronous = NORMAL;`,
	`CREATE TABLE IF NOT EXISTS episodes (
		id TEXT PRIMARY KEY,
		path TEXT NOT NULL,
		generator_id TEXT,
		created_at DATETIME NOT NULL,
		world_id TEXT,
		character_ids TEXT,
		event_id TEXT
	);`,
	`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at DATETIME NOT NULL
	);`,
	`CREATE INDEX IF NOT EXISTS idx_episodes_created_at ON episodes(created_at DESC);`,
	`CREATE INDEX IF NOT EXISTS idx_episodes_generator_created_at ON episodes(generator_id, created_at DESC);`,
	`CREATE INDEX IF NOT EXISTS idx_episodes_id_path ON episodes(id, path);`,
}

type Repository struct {
	dsn     string
	baseDir string

	initOnce sync.Once
	initErr  error
	db       *sql.DB
}

func New(dsn string) *Repository {
	return &Repository{dsn: normalizeDSN(dsn), baseDir: BaseDirFromDSN(dsn)}
}

func (r *Repository) Save(ctx context.Context, record episode.Record) (episode.StoredRecord, error) {
	if err := ctx.Err(); err != nil {
		return episode.StoredRecord{}, err
	}
	path := snapshotPath(r.baseDir, record.Manifest.CreatedAt, record.Manifest.EpisodeID)
	if err := r.writeSnapshot(path, record); err != nil {
		return episode.StoredRecord{}, err
	}
	if err := r.indexRecord(ctx, path, record); err != nil {
		_ = os.RemoveAll(path)
		return episode.StoredRecord{}, err
	}
	return episode.StoredRecord{Path: path, Manifest: record.Manifest}, nil
}

func (r *Repository) FindByID(ctx context.Context, id string) (episode.StoredRecord, error) {
	if err := r.ensureDB(); err != nil {
		return episode.StoredRecord{}, err
	}
	row := r.db.QueryRowContext(ctx, `SELECT path FROM episodes WHERE id = ?`, id)
	var path string
	if err := row.Scan(&path); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return episode.StoredRecord{}, fmt.Errorf("%w: %s", episode.ErrEpisodeNotFound, id)
		}
		return episode.StoredRecord{}, err
	}
	manifest, err := readManifest(path)
	if err != nil {
		return episode.StoredRecord{}, err
	}
	return episode.StoredRecord{Path: path, Manifest: manifest}, nil
}

func (r *Repository) RecentCombos(ctx context.Context, limit int) ([]episode.Combo, error) {
	if limit <= 0 {
		return nil, nil
	}
	if err := r.ensureDB(); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT world_id, character_ids, event_id
		FROM episodes
		ORDER BY created_at DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCombos(rows, limit)
}

func (r *Repository) RecentReferencesByGenerator(ctx context.Context, generatorID string, limit int) ([]episode.ContinuityReference, error) {
	generatorID = strings.TrimSpace(generatorID)
	if limit <= 0 || generatorID == "" {
		return nil, nil
	}
	if err := r.ensureDB(); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, path, created_at
		FROM episodes
		WHERE generator_id = ?
		ORDER BY created_at DESC
		LIMIT ?`, generatorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]episode.ContinuityReference, 0, limit)
	for rows.Next() {
		var id, path, createdAt string
		if err := rows.Scan(&id, &path, &createdAt); err != nil {
			return nil, err
		}
		ts, err := parseStoredTime(createdAt)
		if err != nil {
			return nil, err
		}
		ref := episode.ContinuityReference{
			EpisodeID:   id,
			GeneratorID: generatorID,
			CreatedAt:   ts,
		}
		if prompt, err := os.ReadFile(filepath.Join(path, promptFileName)); err == nil {
			ref.Prompt = string(prompt)
		}
		if outputText, err := os.ReadFile(filepath.Join(path, outputTextFileName)); err == nil {
			ref.OutputText = string(outputText)
			ref.Summary = summarizeText(ref.OutputText)
		}
		if ref.OutputAssetPath == "" {
			if assetPath := findOutputAsset(path); assetPath != "" {
				ref.OutputAssetPath = assetPath
				if ref.Summary == "" {
					ref.Summary = "Visual output from previous episode " + id
				}
			}
		}
		out = append(out, ref)
	}
	return out, rows.Err()
}

func (r *Repository) RecentCombosByGenerator(ctx context.Context, generatorID string, limit int) ([]episode.Combo, error) {
	generatorID = strings.TrimSpace(generatorID)
	if limit <= 0 || generatorID == "" {
		return nil, nil
	}
	if err := r.ensureDB(); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT world_id, character_ids, event_id
		FROM episodes
		WHERE generator_id = ?
		ORDER BY created_at DESC
		LIMIT ?`, generatorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCombos(rows, limit)
}

func (r *Repository) writeSnapshot(path string, record episode.Record) error {
	// Write to a temp directory first, then atomically rename to the final path
	// to avoid leaving partial snapshots on disk if the process is interrupted.
	tmpPath := path + ".tmp"
	if err := os.RemoveAll(tmpPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := r.writeSnapshotTo(tmpPath, record); err != nil {
		_ = os.RemoveAll(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.RemoveAll(tmpPath)
		return err
	}
	return nil
}

func (r *Repository) writeSnapshotTo(path string, record episode.Record) error {
	if err := os.MkdirAll(path, dirMode); err != nil {
		return err
	}
	requiredJSON := []struct {
		name  string
		value any
	}{
		{name: manifestFileName, value: record.Manifest},
		{name: contextFileName, value: record.Context},
		{name: providerRequestFileName, value: record.ProviderRequest},
		{name: providerResponseFileName, value: record.ProviderResponse},
	}
	for _, item := range requiredJSON {
		if err := writeJSON(filepath.Join(path, item.name), item.value); err != nil {
			return err
		}
	}
	if err := writeTextFile(filepath.Join(path, promptFileName), record.Prompt); err != nil {
		return err
	}
	if err := writeOptionalJSON(filepath.Join(path, scoreFileName), record.Manifest.Scores); err != nil {
		return err
	}
	if err := writeOptionalJSON(filepath.Join(path, publishFileName), record.Publish); err != nil {
		return err
	}
	if err := writeOptionalJSON(filepath.Join(path, presentationFileName), record.Presentation); err != nil {
		return err
	}
	if err := writeOptionalJSON(filepath.Join(path, artistSnapshotFileName), record.ArtistSnapshot); err != nil {
		return err
	}
	if err := writeOptionalText(filepath.Join(path, outputTextFileName), record.OutputText); err != nil {
		return err
	}
	if err := writeOptionalJSON(filepath.Join(path, outputPartsFileName), record.OutputParts); err != nil {
		return err
	}
	if record.OutputAssetPath != "" {
		target := filepath.Join(path, filepath.Base(record.OutputAssetPath))
		if err := copyFile(record.OutputAssetPath, target); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (r *Repository) indexRecord(ctx context.Context, path string, record episode.Record) error {
	if err := r.ensureDB(); err != nil {
		return err
	}
	characters, err := json.Marshal(record.Manifest.CharacterIDs)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO episodes(id, path, generator_id, created_at, world_id, character_ids, event_id)
		VALUES(?, ?, ?, ?, ?, ?, ?)`,
		record.Manifest.EpisodeID,
		path,
		record.Manifest.ArtistID,
		record.Manifest.CreatedAt.UTC().Format(time.RFC3339Nano),
		firstOrEmpty(record.Manifest.WorldIDs),
		string(characters),
		record.Manifest.EventID,
	)
	return err
}

func applyMigrationIfNeeded(db *sql.DB, version int, query string) error {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, version).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	if _, err := db.Exec(query); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return err
	}
	_, err := db.Exec(`INSERT INTO schema_migrations(version, applied_at) VALUES(?, ?)`,
		version, time.Now().UTC().Format(time.RFC3339))
	return err
}

func (r *Repository) ensureDB() error {
	r.initOnce.Do(func() {
		if err := os.MkdirAll(filepath.Dir(r.dsn), dirMode); err != nil {
			r.initErr = err
			return
		}
		db, err := sql.Open("sqlite3", r.dsn)
		if err != nil {
			r.initErr = err
			return
		}
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
		db.SetConnMaxLifetime(0)
		for _, stmt := range sqliteInitStatements {
			if _, err := db.Exec(stmt); err != nil {
				_ = db.Close()
				r.initErr = err
				return
			}
		}
		if err := applyMigrationIfNeeded(db, 1, `ALTER TABLE episodes ADD COLUMN path TEXT`); err != nil {
			_ = db.Close()
			r.initErr = err
			return
		}
		if err := applyMigrationIfNeeded(db, 2, `ALTER TABLE episodes ADD COLUMN generator_id TEXT`); err != nil {
			_ = db.Close()
			r.initErr = err
			return
		}
		// Best-effort: propagate artist_id → generator_id for legacy rows.
		// The artist_id column may not exist in all schemas; errors are intentionally ignored.
		_, _ = db.Exec(`UPDATE episodes SET generator_id = artist_id WHERE generator_id IS NULL AND artist_id IS NOT NULL`)
		if err := r.backfillPaths(db); err != nil {
			_ = db.Close()
			r.initErr = err
			return
		}
		r.db = db
	})
	return r.initErr
}

func (r *Repository) backfillPaths(db *sql.DB) error {
	rows, err := db.Query(`SELECT id, created_at, path FROM episodes`)
	if err != nil {
		return err
	}
	defer rows.Close()
	var updates []struct {
		id   string
		path string
	}
	for rows.Next() {
		var id, createdAt string
		var path sql.NullString
		if err := rows.Scan(&id, &createdAt, &path); err != nil {
			return err
		}
		if path.Valid && strings.TrimSpace(path.String) != "" {
			continue
		}
		ts, err := parseStoredTime(createdAt)
		if err != nil {
			return err
		}
		updates = append(updates, struct {
			id   string
			path string
		}{
			id:   id,
			path: snapshotPath(r.baseDir, ts, id),
		})
	}
	stmt, err := db.Prepare(`UPDATE episodes SET path = ? WHERE id = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, update := range updates {
		if _, err := stmt.Exec(update.path, update.id); err != nil {
			return err
		}
	}
	return rows.Err()
}

func scanCombos(rows *sql.Rows, limit int) ([]episode.Combo, error) {
	out := make([]episode.Combo, 0, limit)
	for rows.Next() {
		var worldID, charJSON, eventID string
		if err := rows.Scan(&worldID, &charJSON, &eventID); err != nil {
			return nil, err
		}
		var characters []string
		if strings.TrimSpace(charJSON) != "" {
			if err := json.Unmarshal([]byte(charJSON), &characters); err != nil {
				return nil, err
			}
		}
		out = append(out, episode.Combo{
			WorldID:      worldID,
			CharacterIDs: characters,
			EventID:      eventID,
		})
	}
	return out, rows.Err()
}

func snapshotPath(baseDir string, ts time.Time, episodeID string) string {
	return filepath.Join(baseDir, "episodes", ts.Format("2006"), ts.Format("01"), episodeID)
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), dirMode); err != nil {
		return err
	}
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')
	return os.WriteFile(path, content, fileMode)
}

func writeOptionalJSON(path string, value any) error {
	if isEmptyValue(value) {
		return nil
	}
	return writeJSON(path, value)
}

func writeTextFile(path, value string) error {
	return os.WriteFile(path, []byte(value), fileMode)
}

func writeOptionalText(path, value string) error {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return writeTextFile(path, value)
}

func readManifest(path string) (episode.Manifest, error) {
	content, err := os.ReadFile(filepath.Join(path, manifestFileName))
	if err != nil {
		return episode.Manifest{}, err
	}
	var manifest episode.Manifest
	if err := json.Unmarshal(content, &manifest); err != nil {
		return episode.Manifest{}, err
	}
	return manifest, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileMode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Sync(); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func summarizeText(value string) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= summaryLimit {
		return value
	}
	return string(runes[:summaryLimit])
}

func findOutputAsset(path string) string {
	entries, err := os.ReadDir(path)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.EqualFold(name, outputTextFileName) || strings.HasSuffix(strings.ToLower(name), ".json") || strings.EqualFold(name, promptFileName) || strings.EqualFold(name, manifestFileName) {
			continue
		}
		ext := strings.ToLower(filepath.Ext(name))
		switch ext {
		case ".png", ".jpg", ".jpeg", ".webp", ".mp4", ".mov", ".webm":
			return filepath.Join(path, name)
		}
	}
	return ""
}

func firstOrEmpty(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func BaseDirFromDSN(dsn string) string {
	normalized := normalizeDSN(dsn)
	// Use the default data dir for the default DSN or bare filenames with no directory component
	if normalized == defaultDSN || filepath.Base(normalized) == normalized {
		return defaultBaseDir
	}
	return filepath.Dir(normalized)
}

func normalizeDSN(dsn string) string {
	if strings.TrimSpace(dsn) == "" {
		return defaultDSN
	}
	return dsn
}

func parseStoredTime(value string) (time.Time, error) {
	if ts, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return ts, nil
	}
	return time.Parse(time.RFC3339, value)
}

func isEmptyValue(value any) bool {
	switch v := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(v) == ""
	case []string:
		return len(v) == 0
	case map[string]any:
		return len(v) == 0
	default:
		return false
	}
}
