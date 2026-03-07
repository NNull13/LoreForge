# LoreForge (MVP)

A Go engine to generate autonomous text and video episodes from a universe defined in Markdown + YAML frontmatter.

## Commands

```bash
go run ./cmd/loreforge run --config ./universes/config/config.yaml
go run ./cmd/loreforge validate --config ./universes/config/config.yaml
go run ./cmd/loreforge generate once --agent text --config ./universes/config/config.yaml
go run ./cmd/loreforge episode show <episode-id> --config ./universes/config/config.yaml
go run ./cmd/loreforge universe lint ./universes/universes/example-universe
go run ./cmd/loreforge scheduler next-run --config ./universes/config/config.yaml
```

## Structure

- `cmd/loreforge`: CLI.
- `internal/universe`: universe loading and validation.
- `internal/planner`: narrative brief with weights and anti-repetition.
- `internal/agents`: text and video agents.
- `internal/providers`: decoupled providers (mock in MVP).
- `internal/channels`: filesystem channel.
- `internal/memory`: episode persistence and scheduler state.
- `internal/core`: cycle orchestration.

## Episode Persistence

Episodes are stored in:

- `data/episodes/{yyyy}/{mm}/{episode-id}/manifest.json`
- `context.json`, `prompt.txt`, `provider_request.json`, `provider_response.json`, `output.txt|*.mp4`, `score.json`, `publish.json`
