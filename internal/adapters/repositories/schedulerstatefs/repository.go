package schedulerstatefs

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
	path, err := statePath(r.BaseDir, generatorID)
	if err != nil {
		return state, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		legacyPath := legacyStatePath(r.BaseDir, generatorID)
		if os.IsNotExist(err) && legacyPath != path {
			content, err = os.ReadFile(legacyPath)
		}
		if err != nil {
			if os.IsNotExist(err) {
				return state, nil
			}
			return state, err
		}
	}
	if err := json.Unmarshal(content, &state); err != nil {
		return state, err
	}
	return state, nil
}

func (r Repository) Save(_ context.Context, generatorID string, state scheduling.State) error {
	path, err := statePath(r.BaseDir, generatorID)
	if err != nil {
		return err
	}
	return writeJSON(path, state)
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
		if decoded, ok := decodeGeneratorID(id); ok {
			out = append(out, decoded)
			continue
		}
		if strings.TrimSpace(id) != "" {
			out = append(out, id)
		}
	}
	seen := make(map[string]bool, len(out))
	unique := make([]string, 0, len(out))
	for _, id := range out {
		if seen[id] {
			continue
		}
		seen[id] = true
		unique = append(unique, id)
	}
	sort.Strings(unique)
	return unique, nil
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

func statePath(baseDir, generatorID string) (string, error) {
	generatorID = strings.TrimSpace(generatorID)
	if generatorID == "" {
		return "", fmt.Errorf("generator id is required")
	}
	return filepath.Join(baseDir, "scheduler_state_"+hex.EncodeToString([]byte(generatorID))+".json"), nil
}

func legacyStatePath(baseDir, generatorID string) string {
	return filepath.Join(baseDir, "scheduler_state_"+strings.TrimSpace(generatorID)+".json")
}

func decodeGeneratorID(value string) (string, bool) {
	raw, err := hex.DecodeString(strings.TrimSpace(value))
	if err != nil || len(raw) == 0 {
		return "", false
	}
	return string(raw), true
}
