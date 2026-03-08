# Editorial Artists

LoreForge models artists as editorial entities inside the universe.

They are not diegetic characters by default. They should shape voice, framing, prompting, and presentation without becoming participants in the fiction.

## Layout

```text
universes/example-universe/
  artists/
    ash-chorister/
      artist.md
      assets.yaml
      portrait.png
      signature-mark.png
```

## `artist.md`

Required frontmatter fields:

- `id`
- `name`
- `role`
- `summary`
- `non_diegietic`
- `voice`
- `mission`
- `prompting`
- `presentation`

Example:

```yaml
id: ash-chorister
name: The Ash Chorister
title: Chronicler of the Ember Archive
role: chronicler
summary: An editorial voice that records the universe with ritual gravity.
non_diegietic: true

voice:
  register: elevated
  cadence: ritual
  diction: ceremonial
  stance: observant
  perspective: editorial
  intensity: medium

mission:
  purpose: Preserve canon through reflective narration.
  priorities:
    - clarity of canon
    - continuity over spectacle

prompting:
  system_identity: You are The Ash Chorister.
  system_rules:
    - Never contradict canon.
    - Never appear as an in-world character.
  tonal_biases:
    - ritual
    - restrained
  lexical_cues:
    - ember
    - oath
  forbidden:
    - internet slang

presentation:
  enabled: true
  signature_mode: presentation_only
  signature_text: Filed by The Ash Chorister.
  framing_mode: intro_outro
  intro_template: From the Ember Archive:
  outro_template: Filed by The Ash Chorister.
  allowed_channels:
    - filesystem
    - twitter

future:
  memory_mode: reserved
```

The markdown body is freeform long-form editorial context. LoreForge uses it as part of the artist snapshot and future extension point.

## Runtime binding

`config.yaml` binds runtime generators to artist profiles:

```yaml
artists:
  - id: short-story-artist
    profile_id: ash-chorister
    type: short_story
```

- `id`: runtime generator/job id
- `profile_id`: editorial profile id from the universe

## Presentation model

Supported `signature_mode` values:

- `none`
- `presentation_only`
- `append`
- `prepend`

Supported `framing_mode` values:

- `none`
- `intro`
- `outro`
- `intro_outro`

Presentation is applied per channel as a separate layer from the generated work itself.

## Artist assets

Artists support the same `assets.yaml` pattern as other entities.

Common usages:

- `artist_portrait`
- `editorial_brand`
- `signature_mark`
- `style_reference`
- `mood_reference`

In the current MVP:

- artist assets can influence prompting and reference selection
- image/video outputs can use artist assets as style context
- signature assets are not embedded into generated binaries automatically

## Priority rules

LoreForge applies these priorities in prompting:

1. Universe canon
2. Template
3. Format rules
4. Artist lens
5. References and continuity
6. Provider constraints

This prevents the artist from overriding the universe.
