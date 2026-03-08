# Universe Assets

LoreForge uses a single universe format: each entity lives in its own directory, and assets sit beside the entity markdown file.

## Layout

```text
universes/example-universe/
  universe/
    universe.md
    assets.yaml
    skyline.png

  characters/
    red-wanderer/
      red-wanderer.md
      assets.yaml
      red-wanderer-base.png
      red-wanderer-run.png

  worlds/
    glass-kingdom/
      glass-kingdom.md
      assets.yaml
      glass-kingdom-moodboard.png
```

Naming is strict:

- directory name must match the entity `id`
- markdown file must be `<id>.md`
- any extra markdown file in the entity directory is rejected

## assets.yaml

`assets.yaml` is optional. If it exists, LoreForge merges the declared entries with autodiscovered visual files.

```yaml
assets:
  - file: red-wanderer-base.png
    id: red-wanderer-base
    media_type: image
    usage: character_reference
    description: Primary look reference with the red scarf and dust coat.
    tags: [portrait, default]
    weight: 100
    optional: false
    model_roles:
      runway_gen4: prompt_image
      vertex_veo: asset
```

Supported `usage` values:

- `character_reference`
- `style_reference`
- `environment_reference`
- `prop_reference`
- `pose_reference`
- `continuity_reference`
- `video_prompt_image`

Supported `model_roles` keys:

- `mock`
- `openai_image`
- `vertex_imagen`
- `vertex_veo`
- `runway_gen4`

`video_prompt_image` is only valid for `media_type: image`.

`file` is intentionally restricted:

- it must be a basename of a file that lives beside the entity markdown
- absolute paths are rejected
- subdirectories are rejected
- `..` is rejected

This keeps asset declarations contained to the entity directory and prevents path traversal through config content.

## Autodiscovery

If an entity directory contains images or videos and no `assets.yaml`, LoreForge creates default asset records:

- `id`: filename without extension
- `media_type`: inferred from extension
- `usage`:
  - `character_reference` for characters
  - `environment_reference` for worlds
  - `continuity_reference` for universe, events, templates, and rules
- `weight`: `50`
- `optional`: `true`

## Reference Selection

At generation time LoreForge selects references from:

1. root universe assets
2. world assets
3. character assets
4. event assets
5. template assets
6. recent outputs from the same artist

The final mix is controlled per artist with:

- `reference_mode`
- `max_asset_references`
- `max_continuity_items`
- `asset_usage_allowlist`
- `include_text_memories`

For `runway_gen4`, a selected asset with `model_role=prompt_image` wins over bootstrap generation. If there is no selected prompt image, LoreForge can still fall back to `bootstrap_image_generator` or `bootstrap_image_provider`.
