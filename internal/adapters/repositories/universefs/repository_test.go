package universefs

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadUniverseWithEntityDirectoriesAndAssets(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeEntityFile(t, filepath.Join(root, "universe", "universe.md"), `---
id: no-name-universe
type: universe
name: No Name Universe
---
Root body.`)
	writeFile(t, filepath.Join(root, "universe", "assets.yaml"), `assets:
  - file: skyline.png
    id: skyline
    media_type: image
    usage: environment_reference
    weight: 90
`)
	writeFile(t, filepath.Join(root, "universe", "skyline.png"), "placeholder")
	writeEntityFile(t, filepath.Join(root, "characters", "panda", "panda.md"), `---
id: panda
type: character
display_name: Panda
---
Character body.`)
	writeFile(t, filepath.Join(root, "characters", "panda", "assets.yaml"), `assets:
  - file: panda.png
    id: panda-base
    media_type: image
    usage: video_prompt_image
    weight: 100
`)
	writeFile(t, filepath.Join(root, "characters", "panda", "panda.png"), "placeholder")
	writeEntityFile(t, filepath.Join(root, "worlds", "bamboo-forest", "bamboo-forest.md"), `---
id: bamboo-forest
type: world
---
World body.`)
	writeFile(t, filepath.Join(root, "worlds", "bamboo-forest", "moodboard.png"), "placeholder")
	writeEntityFile(t, filepath.Join(root, "events", "moon-festival", "moon-festival.md"), `---
id: moon-festival
type: event
---
Event body.`)
	writeEntityFile(t, filepath.Join(root, "templates", "short-story", "short-story.md"), `---
id: short-story
type: template
output_type: text
---
Template body.`)
	writeEntityFile(t, filepath.Join(root, "rules", "global-rules", "global-rules.md"), `---
id: global-rules
type: rule
target: all
---
Rule body.`)

	repo := Repository{Root: root}
	u, err := repo.Load(context.Background())
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if u.Universe.ID != "no-name-universe" {
		t.Fatalf("unexpected universe id: %s", u.Universe.ID)
	}
	if len(u.Universe.Assets.Items) != 1 {
		t.Fatalf("unexpected root assets: %d", len(u.Universe.Assets.Items))
	}
	if len(u.Characters["panda"].Assets.Items) != 1 {
		t.Fatalf("unexpected character assets: %d", len(u.Characters["panda"].Assets.Items))
	}
	if len(u.Worlds["bamboo-forest"].Assets.Items) != 1 {
		t.Fatalf("expected autodiscovered world asset")
	}
}

func TestLoadRejectsFolderIdMismatch(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeEntityFile(t, filepath.Join(root, "universe", "universe.md"), `---
id: no-name-universe
type: universe
---
Root body.`)
	writeEntityFile(t, filepath.Join(root, "characters", "panda", "panda.md"), `---
id: hero
type: character
---
Character body.`)
	writeEntityFile(t, filepath.Join(root, "worlds", "bamboo-forest", "bamboo-forest.md"), `---
id: bamboo-forest
type: world
---
World body.`)
	writeEntityFile(t, filepath.Join(root, "events", "moon-festival", "moon-festival.md"), `---
id: moon-festival
type: event
---
Event body.`)
	writeEntityFile(t, filepath.Join(root, "templates", "short-story", "short-story.md"), `---
id: short-story
type: template
output_type: text
---
Template body.`)
	writeEntityFile(t, filepath.Join(root, "rules", "global-rules", "global-rules.md"), `---
id: global-rules
type: rule
---
Rule body.`)

	repo := Repository{Root: root}
	if _, err := repo.Load(context.Background()); err == nil {
		t.Fatal("expected id mismatch error")
	}
}

func writeEntityFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
