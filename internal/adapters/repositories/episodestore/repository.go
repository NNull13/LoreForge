package episodestore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"loreforge/internal/domain/episode"

	_ "github.com/mattn/go-sqlite3"
)

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

func (r *Repository) Save(_ context.Context, record episode.Record) (episode.StoredRecord, error) {
	path := snapshotPath(r.baseDir, record.Manifest.CreatedAt, record.Manifest.EpisodeID)
	if err := r.writeSnapshot(path, record); err != nil {
		return episode.StoredRecord{}, err
	}
	if err := r.indexRecord(path, record); err != nil {
		_ = os.RemoveAll(path)
		return episode.StoredRecord{}, err
	}
	return episode.StoredRecord{Path: path, Manifest: record.Manifest}, nil
}

func (r *Repository) FindByID(_ context.Context, id string) (episode.StoredRecord, error) {
	if err := r.ensureDB(); err != nil {
		return episode.StoredRecord{}, err
	}
	row := r.db.QueryRow(`SELECT path FROM episodes WHERE id = ?`, id)
	var path string
	if err := row.Scan(&path); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return episode.StoredRecord{}, fmt.Errorf("%w: %s", episode.ErrEpisodeNotFound, id)
		}
		return episode.StoredRecord{}, err
	}
	content, err := os.ReadFile(filepath.Join(path, "manifest.json"))
	if err != nil {
		return episode.StoredRecord{}, err
	}
	var manifest episode.Manifest
	if err := json.Unmarshal(content, &manifest); err != nil {
		return episode.StoredRecord{}, err
	}
	return episode.StoredRecord{Path: path, Manifest: manifest}, nil
}

func (r *Repository) RecentCombos(_ context.Context, limit int) ([]episode.Combo, error) {
	if limit <= 0 {
		return nil, nil
	}
	if err := r.ensureDB(); err != nil {
		return nil, err
	}
	rows, err := r.db.Query(`
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

func (r *Repository) RecentCombosByGenerator(_ context.Context, generatorID string, limit int) ([]episode.Combo, error) {
	if limit <= 0 || strings.TrimSpace(generatorID) == "" {
		return nil, nil
	}
	if err := r.ensureDB(); err != nil {
		return nil, err
	}
	rows, err := r.db.Query(`
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
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(path, "manifest.json"), record.Manifest); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(path, "context.json"), record.Context); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(path, "prompt.txt"), []byte(record.Prompt), 0o644); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(path, "provider_request.json"), record.ProviderRequest); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(path, "provider_response.json"), record.ProviderResponse); err != nil {
		return err
	}
	if len(record.Manifest.Scores) > 0 {
		if err := writeJSON(filepath.Join(path, "score.json"), record.Manifest.Scores); err != nil {
			return err
		}
	}
	if len(record.Publish) > 0 {
		if err := writeJSON(filepath.Join(path, "publish.json"), record.Publish); err != nil {
			return err
		}
	}
	if record.OutputText != "" {
		if err := os.WriteFile(filepath.Join(path, "output.txt"), []byte(record.OutputText), 0o644); err != nil {
			return err
		}
	}
	if record.OutputAssetPath != "" {
		target := filepath.Join(path, filepath.Base(record.OutputAssetPath))
		content, err := os.ReadFile(record.OutputAssetPath)
		if err == nil {
			if err := os.WriteFile(target, content, 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Repository) indexRecord(path string, record episode.Record) error {
	if err := r.ensureDB(); err != nil {
		return err
	}
	characters, err := json.Marshal(record.Manifest.CharacterIDs)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(`
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

func (r *Repository) ensureDB() error {
	r.initOnce.Do(func() {
		if err := os.MkdirAll(filepath.Dir(r.dsn), 0o755); err != nil {
			r.initErr = err
			return
		}
		db, err := sql.Open("sqlite3", r.dsn)
		if err != nil {
			r.initErr = err
			return
		}
		if _, err := db.Exec(`
			CREATE TABLE IF NOT EXISTS episodes (
				id TEXT PRIMARY KEY,
				path TEXT NOT NULL,
				generator_id TEXT,
				created_at DATETIME NOT NULL,
				world_id TEXT,
				character_ids TEXT,
				event_id TEXT
			);`); err != nil {
			_ = db.Close()
			r.initErr = err
			return
		}
		_, _ = db.Exec(`ALTER TABLE episodes ADD COLUMN path TEXT;`)
		_, _ = db.Exec(`ALTER TABLE episodes ADD COLUMN generator_id TEXT;`)
		_, _ = db.Exec(`UPDATE episodes SET generator_id = artist_id WHERE generator_id IS NULL AND artist_id IS NOT NULL;`)
		if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_episodes_created_at ON episodes(created_at DESC);`); err != nil {
			_ = db.Close()
			r.initErr = err
			return
		}
		if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_episodes_generator_created_at ON episodes(generator_id, created_at DESC);`); err != nil {
			_ = db.Close()
			r.initErr = err
			return
		}
		if err := r.backfillPaths(db); err != nil {
			_ = db.Close()
			r.initErr = err
			return
		}
		_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_episodes_id_path ON episodes(id, path);`)
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
		ts, err := time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			ts, _ = time.Parse(time.RFC3339, createdAt)
		}
		updates = append(updates, struct {
			id   string
			path string
		}{
			id:   id,
			path: snapshotPath(r.baseDir, ts, id),
		})
	}
	for _, update := range updates {
		if _, err := db.Exec(`UPDATE episodes SET path = ? WHERE id = ?`, update.path, update.id); err != nil {
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
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')
	return os.WriteFile(path, content, 0o644)
}

func firstOrEmpty(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func BaseDirFromDSN(dsn string) string {
	normalized := normalizeDSN(dsn)
	if normalized == "./data/universe.db" || filepath.Base(normalized) == normalized {
		return "./data"
	}
	return filepath.Dir(normalized)
}

func normalizeDSN(dsn string) string {
	if strings.TrimSpace(dsn) == "" {
		return "./data/universe.db"
	}
	return dsn
}
