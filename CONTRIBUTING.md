# Contributing To LoreForge

## Quick Start
1. Fork the repository (if you do not have write access).
2. Create a branch from `main`:
   - `git checkout -b codex/short-description`
3. Make focused changes with tests.
4. Run:
   - `go test ./...`
5. Push your branch and open a Pull Request.

## Branch And Commit Guidelines
- Branch naming: `codex/<topic>`
- Keep PRs focused and small.
- Use clear commit messages describing intent.

## Pull Request Expectations
- Fill out the PR template completely.
- Link related issue(s) when possible.
- Include tests for behavior changes.
- Update docs when CLI/config behavior changes.

## From A Fork
1. Sync your fork with upstream `main` before starting.
2. Push your branch to your fork.
3. Open PR from `your-fork:branch` into `LoreForge:main`.
4. Ensure no secrets/config credentials are included.

## Code Style
- Follow existing project structure under `cmd/` and `internal/`.
- Prefer simple, explicit names and avoid unnecessary aliases.
- Keep changes backward-compatible unless explicitly discussed.
