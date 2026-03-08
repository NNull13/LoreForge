# Vertex Veo

- Driver: `vertex_veo`
- Recommended model: `veo-3.1-fast-generate-001`
- Required config: `project_id_env`, `location`, `bucket_uri`, `poll_interval`, `timeout`
- Auth implementation: `GOOGLE_CLOUD_ACCESS_TOKEN` or `VERTEX_AI_ACCESS_TOKEN`

LoreForge uses the long-running prediction flow and polls until completion. The final video is downloaded from GCS before episode persistence.
