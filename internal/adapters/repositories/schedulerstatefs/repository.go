package schedulerstatefs

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"loreforge/internal/domain/scheduling"
)

type Repository struct {
	BaseDir string
}

func (r Repository) Load(_ context.Context, generatorID string) (scheduling.State, error) {
	var state scheduling.State
	content, err := os.ReadFile(filepath.Join(r.BaseDir, "scheduler_state_"+generatorID+".json"))
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return state, err
	}
	if err := json.Unmarshal(content, &state); err != nil {
		return state, err
	}
	return state, nil
}

func (r Repository) Save(_ context.Context, generatorID string, state scheduling.State) error {
	return writeJSON(filepath.Join(r.BaseDir, "scheduler_state_"+generatorID+".json"), state)
}

func (r Repository) ListGeneratorIDs(_ context.Context) ([]string, error) {
	entries, err := os.ReadDir(r.BaseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "scheduler_state_") || !strings.HasSuffix(name, ".json") {
			continue
		}
		id := strings.TrimSuffix(strings.TrimPrefix(name, "scheduler_state_"), ".json")
		if strings.TrimSpace(id) != "" {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out, nil
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
