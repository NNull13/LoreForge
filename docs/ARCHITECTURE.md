# MVP Architecture

Principles:

- Universe-first.
- Provider-agnostic.
- Agents and channels decoupled through interfaces.
- Full traceability per episode.

Flow:

1. Load `config.yaml`.
2. Load/validate the Markdown universe.
3. The planner generates an `EpisodeBrief` with weights (`text/video`) and recent anti-repetition.
4. An agent invokes a provider (text/video).
5. Minimal output validation + retries.
6. Full episode persistence.
7. Publish to the filesystem channel.
8. Compute `next_run` with the scheduler (`random_window` or `fixed_interval`).
