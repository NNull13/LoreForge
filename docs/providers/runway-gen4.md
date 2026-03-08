# Runway Gen-4

- Driver: `runway_gen4`
- Recommended model: `gen4_turbo`
- Required config: `api_key_env`, `version`, `poll_interval`, `timeout`
- Header used: `X-Runway-Version: 2024-11-06`

LoreForge integrates Runway as `image_to_video`. Configure either `bootstrap_image_generator` or `bootstrap_image_provider` on the video artist so the system can create a bootstrap image before invoking Runway.
