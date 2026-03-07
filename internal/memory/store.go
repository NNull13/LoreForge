package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"loreforge/internal/planner"
	"loreforge/pkg/contracts"
)

type Store struct {
	baseDir string
}

type HistoryEntry struct {
	EpisodeID    string    `json:"episode_id"`
	CreatedAt    time.Time `json:"created_at"`
	WorldID      string    `json:"world_id"`
	CharacterIDs []string  `json:"character_ids"`
	EventID      string    `json:"event_id"`
}

type SchedulerState struct {
	LastRunAt *time.Time `json:"last_run_at,omitempty"`
	NextRunAt time.Time  `json:"next_run_at"`
}

func New(baseDir string) *Store {
	if baseDir == "" {
		baseDir = "./data"
	}
	return &Store{baseDir: baseDir}
}

func (s *Store) SaveEpisode(r contracts.EpisodeRecord) (string, error) {
	ts := r.Manifest.CreatedAt
	path := filepath.Join(s.baseDir, "episodes", ts.Format("2006"), ts.Format("01"), r.Manifest.EpisodeID)
	if err := os.MkdirAll(path, 0o755); err != nil {
		return "", err
	}
	if err := writeJSON(filepath.Join(path, "manifest.json"), r.Manifest); err != nil {
		return "", err
	}
	if err := writeJSON(filepath.Join(path, "context.json"), r.Context); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(path, "prompt.txt"), []byte(r.Prompt), 0o644); err != nil {
		return "", err
	}
	if err := writeJSON(filepath.Join(path, "provider_request.json"), r.ProviderRequest); err != nil {
		return "", err
	}
	if err := writeJSON(filepath.Join(path, "provider_response.json"), r.ProviderResponse); err != nil {
		return "", err
	}
	if len(r.Manifest.Scores) > 0 {
		if err := writeJSON(filepath.Join(path, "score.json"), r.Manifest.Scores); err != nil {
			return "", err
		}
	}
	if len(r.Publish) > 0 {
		if err := writeJSON(filepath.Join(path, "publish.json"), r.Publish); err != nil {
			return "", err
		}
	}
	if r.OutputText != "" {
		if err := os.WriteFile(filepath.Join(path, "output.txt"), []byte(r.OutputText), 0o644); err != nil {
			return "", err
		}
	}
	if r.OutputAssetPath != "" {
		target := filepath.Join(path, filepath.Base(r.OutputAssetPath))
		b, err := os.ReadFile(r.OutputAssetPath)
		if err == nil {
			if err := os.WriteFile(target, b, 0o644); err != nil {
				return "", err
			}
		}
	}
	if err := s.appendHistory(HistoryEntry{
		EpisodeID:    r.Manifest.EpisodeID,
		CreatedAt:    r.Manifest.CreatedAt,
		WorldID:      firstOrEmpty(r.Manifest.WorldIDs),
		CharacterIDs: r.Manifest.CharacterIDs,
		EventID:      r.Manifest.EventID,
	}); err != nil {
		return "", err
	}
	return path, nil
}

func (s *Store) FindEpisode(episodeID string) (string, contracts.EpisodeManifest, error) {
	root := filepath.Join(s.baseDir, "episodes")
	var manifestPath string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() != "manifest.json" {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		var m contracts.EpisodeManifest
		if json.Unmarshal(b, &m) == nil && m.EpisodeID == episodeID {
			manifestPath = path
			return fmt.Errorf("found")
		}
		return nil
	})
	if manifestPath == "" {
		if err != nil && err.Error() != "found" {
			return "", contracts.EpisodeManifest{}, err
		}
		return "", contracts.EpisodeManifest{}, fmt.Errorf("episode not found: %s", episodeID)
	}
	b, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", contracts.EpisodeManifest{}, err
	}
	var m contracts.EpisodeManifest
	if err := json.Unmarshal(b, &m); err != nil {
		return "", contracts.EpisodeManifest{}, err
	}
	return filepath.Dir(manifestPath), m, nil
}

func (s *Store) RecentCombos(limit int) ([]planner.HistoryCombo, error) {
	entries, err := s.readHistory()
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].CreatedAt.Before(entries[j].CreatedAt) })
	if limit > 0 && len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}
	out := make([]planner.HistoryCombo, 0, len(entries))
	for _, e := range entries {
		out = append(out, planner.HistoryCombo{WorldID: e.WorldID, CharacterIDs: e.CharacterIDs, EventID: e.EventID})
	}
	return out, nil
}

func (s *Store) SaveSchedulerState(st SchedulerState) error {
	return writeJSON(filepath.Join(s.baseDir, "scheduler_state.json"), st)
}

func (s *Store) LoadSchedulerState() (SchedulerState, error) {
	var st SchedulerState
	b, err := os.ReadFile(filepath.Join(s.baseDir, "scheduler_state.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return st, nil
		}
		return st, err
	}
	if err := json.Unmarshal(b, &st); err != nil {
		return st, err
	}
	return st, nil
}

func (s *Store) appendHistory(entry HistoryEntry) error {
	items, err := s.readHistory()
	if err != nil {
		return err
	}
	items = append(items, entry)
	return writeJSON(filepath.Join(s.baseDir, "history.json"), items)
}

func (s *Store) readHistory() ([]HistoryEntry, error) {
	b, err := os.ReadFile(filepath.Join(s.baseDir, "history.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var items []HistoryEntry
	if err := json.Unmarshal(b, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o644)
}

func firstOrEmpty(in []string) string {
	if len(in) == 0 {
		return ""
	}
	return in[0]
}

func SanitizeSecrets(m map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range m {
		lk := strings.ToLower(k)
		if strings.Contains(lk, "key") || strings.Contains(lk, "token") || strings.Contains(lk, "secret") {
			out[k] = "***"
			continue
		}
		out[k] = v
	}
	return out
}
