# Vertex Imagen

- Driver: `vertex_imagen`
- Recommended model: `imagen-4.0-fast-generate-001`
- Required config: `project_id_env`, `location`
- Auth implementation: `GOOGLE_CLOUD_ACCESS_TOKEN` or `VERTEX_AI_ACCESS_TOKEN`

LoreForge currently calls the Vertex AI REST API directly and writes the returned image bytes to a local temp file before episode persistence.
