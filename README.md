# LoreForge

LoreForge is a Go story engine for generating autonomous text, image, and video episodes from a universe defined in Markdown with YAML frontmatter.

It is designed around three ideas:

- the universe is content, not code
- artists are editorial identities with their own voice and presentation
- generation, publishing, and traceability are first-class

## Why It Exists

LoreForge is for building procedural narrative systems that still feel authored.

You define:

- a universe
- its characters, worlds, events, rules, and templates
- editorial artists that interpret that universe
- providers and channels for generation and distribution

LoreForge then picks a brief, generates an episode, publishes it, and stores the full trace of what happened.

## What It Can Generate

Text:

- `tweet_short`
- `tweet_thread`
- `short_story`
- `long_story`
- `poem`
- `song_lyrics`
- `screenplay_series`

Visual:

- `image`
- `video`

## Quick Start

### 1. Validate the bundled example

```bash
go run ./cmd/loreforge validate --config ./universes/config.yaml
go run ./cmd/loreforge universe lint ./universes/example-universe
```

### 2. Inspect the configured artists

```bash
go run ./cmd/loreforge artists list --config ./universes/config.yaml
```

### 3. Generate something

```bash
go run ./cmd/loreforge generate once --artist short-story-artist --config ./universes/config.yaml
go run ./cmd/loreforge generate once --artist tweet-thread-artist --config ./universes/config.yaml
go run ./cmd/loreforge generate once --artist image-artist --config ./universes/config.yaml
```

### 4. Let LoreForge choose the next due artist

```bash
go run ./cmd/loreforge run --config ./universes/config.yaml
```

## Core Commands

| Command | What it does |
| --- | --- |
| `loreforge validate --config ...` | Validates config, universe loading, and runtime wiring. |
| `loreforge universe lint <path>` | Validates only the universe folder structure and schema. |
| `loreforge artists list --config ...` | Lists active runtime artists, linked profiles, providers, and next scheduled run. Artists with `scheduler.enabled: false` show `next_run=disabled`. |
| `loreforge generate once --artist <id> --config ...` | Generates one episode for a specific artist. |
| `loreforge run --config ...` | Generates one episode for the next due artist. Disabled schedulers are ignored, and future artists are not executed early. |
| `loreforge scheduler next-run --artist <id> --config ...` | Shows the next scheduled run for one artist or the nearest overall. Disabled schedulers are excluded from the overall result. |
| `loreforge episode show <episode-id> --config ...` | Shows the stored manifest for one episode. |
| `loreforge config refresh --config ...` | Reconciles config with persisted scheduler state without resetting existing schedules or memory. |

## Config Refresh

`config refresh` is the operational command for adopting config changes safely.

It does this:

- reloads the current config and universe
- preserves scheduler state for existing artists
- creates scheduler state for newly added artists with scheduling enabled
- keeps orphaned scheduler state files untouched instead of deleting them

It does not:

- wipe episode history
- reset artist continuity
- rewrite existing scheduler state for active artists

Example:

```bash
go run ./cmd/loreforge config refresh --config ./universes/config.yaml
```

## Universe Model

LoreForge uses a folder-per-entity universe.

```text
universes/example-universe/
  universe/
    universe.md
    assets.yaml
    skyline.png
  artists/
    ash-chorister/
      artist.md
      assets.yaml
      portrait.png
  characters/
    red-wanderer/
      red-wanderer.md
      assets.yaml
      red-wanderer-base.png
  worlds/
    glass-kingdom/
      glass-kingdom.md
      assets.yaml
  events/
    lost-artifact/
      lost-artifact.md
  templates/
    short-story/
      short-story.md
    tweet-thread/
      tweet-thread.md
  rules/
    global-rules/
      global-rules.md
```

Every entity is content-addressable and can carry its own assets.

## Editorial Artists

Artists are now part of the universe, not just runtime config.

An artist profile defines:

- voice
- mission
- prompting rules
- framing and signature policy
- optional visual/editorial assets

Runtime config binds a generator job to that profile through `profile_id`.

Runtime ids are intentionally strict:

- `id` and `profile_id` must use only letters, numbers, `_`, or `-`
- path separators and `..` are rejected

Example:

```yaml
artists:
  - id: short-story-artist
    profile_id: ash-chorister
    type: short_story
    provider:
      driver: openai_text
      model: gpt-5-mini
```

The important rule is simple:

`universe canon > template > format rules > artist lens > references > provider constraints`

That keeps artists expressive without letting them break continuity.

## Minimal Config Shape

```yaml
app:
  name: loreforge
  env: dev

universe:
  path: ./universes/example-universe

providers:
  text:
    driver: mock
    model: mock-text-v1
  image:
    driver: mock
    model: mock-image-v1
  video:
    driver: mock
    model: mock-video-v1

artists:
  - id: short-story-artist
    profile_id: ash-chorister
    type: short_story
    provider:
      driver: mock
      model: mock-text-v1
    publish_targets: [filesystem]
```

The full working example lives at `universes/config.yaml`.

Path handling is relative to the config file location, not the current shell directory. That applies to:

- `universe.path`
- `memory.dsn`
- `channels.filesystem.output_dir`

## Provider Drivers

| Capability | Drivers |
| --- | --- |
| Text | `mock`, `openai_text`, `lmstudio_text` |
| Image | `mock`, `openai_image`, `vertex_imagen` |
| Video | `mock`, `vertex_veo`, `runway_gen4` |

### Notes

- `lmstudio_text` lets you run local models through LM Studio's OpenAI-compatible server.
- `runway_gen4` is wired as `image_to_video`.
- `vertex_veo` and `runway_gen4` support async generation with polling.
- The bundled example config uses `mock` providers so you can validate the full flow without external credentials.

## Persistence

Each episode stores a full trace under `data/episodes/...`, including:

- `manifest.json`
- `context.json`
- `prompt.txt`
- `provider_request.json`
- `provider_response.json`
- `output.txt` or generated asset
- `output_parts.json` when the output is multipart
- `publish.json`
- `presentation.json`
- `artist_snapshot.json`

Manifest state now distinguishes:

- `generated`: content was generated and no publish target ran
- `published`: at least one publish target succeeded
- `publish_failed`: publish targets existed but none succeeded

This makes LoreForge useful not only as a generator, but also as an auditable narrative pipeline.

## Docs

- [Universe assets](./docs/universe-assets.md)
- [Editorial artists](./docs/artists.md)
- [OpenAI image](./docs/providers/openai-image.md)
- [Vertex Imagen](./docs/providers/vertex-imagen.md)
- [Vertex Veo](./docs/providers/vertex-veo.md)
- [Runway Gen-4](./docs/providers/runway-gen4.md)

## Current Status

LoreForge is still pre-`1.0.0`.

That is deliberate. The project is still moving fast, and the current architecture is optimized for:

- clean boundaries
- provider experimentation
- editorial flexibility
- operational traceability

If you want a starting point, use the example universe, run `artists list`, then generate one text piece and one visual piece to see the full loop.

--------------------

Crafted with ❤️ by NoName13

Questions? Open an issue • Want updates? Star the repo ⭐
