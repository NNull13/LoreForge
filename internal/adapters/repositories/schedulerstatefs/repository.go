package schedulerstatefs

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

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
