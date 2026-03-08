# LoreForge

A Go engine to generate autonomous text and video episodes from a universe defined in Markdown + YAML frontmatter.

## Commands

```bash
go run ./cmd/loreforge run --config ./universes/config.yaml
go run ./cmd/loreforge validate --config ./universes/config.yaml
go run ./cmd/loreforge generate once --artist text-artist --config ./universes/config.yaml
go run ./cmd/loreforge generate once --agent text --config ./universes/config.yaml
go run ./cmd/loreforge generate once --artist image-artist --config ./universes/config.yaml
go run ./cmd/loreforge episode show <episode-id> --config ./universes/config.yaml
go run ./cmd/loreforge universe lint ./universes/example-universe
go run ./cmd/loreforge scheduler next-run --artist text-artist --config ./universes/config.yaml
```

The bundled example config is wired to `mock` providers, but LoreForge now supports these real provider drivers:

- `openai_image`
- `vertex_imagen`
- `vertex_veo`
- `runway_gen4`

## Structure

- `cmd/loreforge`: CLI.
- `internal/domain`: core business types and rules.
- `internal/application`: use cases and ports.
- `internal/adapters`: repositories, providers, publishers, and generators.
- `internal/planner`: narrative brief with weights and anti-repetition.
- `internal/platform`: infrastructure helpers such as IDs, hashing, and clocks.

## Episode Persistence

Episodes are stored in:

- `data/episodes/{yyyy}/{mm}/{episode-id}/manifest.json`
- `context.json`, `prompt.txt`, `provider_request.json`, `provider_response.json`, `output.txt|*.mp4`, `score.json`, `publish.json`

## Provider Drivers

### Google Imagen (Vertex AI)

```yaml
providers:
  image:
    driver: vertex_imagen
    model: imagen-4.0-fast-generate-001
    project_id_env: GOOGLE_CLOUD_PROJECT
    location: us-central1
    timeout: 2m
```

Auth in the current implementation is resolved from `GOOGLE_CLOUD_ACCESS_TOKEN` or `VERTEX_AI_ACCESS_TOKEN`.

### OpenAI GPT Image

```yaml
providers:
  image:
    driver: openai_image
    model: gpt-image-1.5
    api_key_env: OPENAI_API_KEY
    timeout: 2m
    options:
      response_format: b64_json
      quality: auto
```

`dall-e-2` and `dall-e-3` are still accepted as configured models, but OpenAI documents them as deprecated and supported only until May 12, 2026.

### Google Veo (Vertex AI)

```yaml
providers:
  video:
    driver: vertex_veo
    model: veo-3.1-fast-generate-001
    project_id_env: GOOGLE_CLOUD_PROJECT
    location: us-central1
    bucket_uri: gs://my-loreforge-assets
    poll_interval: 10s
    timeout: 10m
```

### Runway Gen-4

```yaml
artists:
  - id: video-artist
    type: video
    provider:
      driver: runway_gen4
      model: gen4_turbo
      api_key_env: RUNWAY_API_KEY
      version: 2024-11-06
      poll_interval: 5s
      timeout: 10m
    options:
      bootstrap_image_provider: vertex_imagen
```

Runway is wired as `image_to_video`. If you choose `runway_gen4`, configure either `options.bootstrap_image_generator` or `options.bootstrap_image_provider` so LoreForge can generate a bootstrap image before creating the video.

See [openai-image.md](/docs/providers/openai-image.md), [vertex-imagen.md](/docs/providers/vertex-imagen.md), [vertex-veo.md](/docs/providers/vertex-veo.md), and [runway-gen4.md](/docs/providers/runway-gen4.md) for provider-specific notes.
