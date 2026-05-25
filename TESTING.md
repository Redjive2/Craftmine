# Testing Craftmine

This document lists every runnable entry point in the repo. Each command is
copy-pasteable from the project root.

## Main game

```
go run .
```

Opens the main menu. Currently:

- **New Game** prints `would create world` to stdout and exits. The real
  world-creation wiring is a separate ticket.
- **Resume Game** is disabled (no saved world to resume).
- **ESC** closes the window.

## World demo

```
go run ./cmd/world-demo
```

Visual acceptance check for the world module (mg-7522). Renders hills of
grass-topped cubes plus scattered trees (wood + leaves) under an orbit
camera. Drag with the left mouse to rotate, right mouse to pan, scroll to
zoom. **ESC** closes.

Flags:

- `-seed N` — world generation seed (default `2026`).
- `-size N` — world width and depth in blocks; must be a multiple of `16`
  (default `96`).
- `-height N` — maximum vertical extent (default `48`).

Example with a different seed and a larger world:

```
go run ./cmd/world-demo -seed 7 -size 128 -height 64
```

## Blocks demo

```
go run ./cmd/blocks-demo
```

Visual acceptance check for the blocks module (mg-0114). Renders one cube
per registered block kind in a row: grass, dirt, stone, wood, leaves.
**ESC** closes.

## Unit and integration tests

```
go test ./...
```

Runs every test across all modules.

## Pre-commit Progress.md auto-stamp

Install once per clone (or per worktree):

```
bash tools/install-hooks.sh
```

This drops a `pre-commit` hook into the local `.git/hooks/` dir that delegates
to the tracked script at `tools/pre-commit-progress.sh`. On every subsequent
`git commit`, the script:

- reads the last `Version: X.Y.Z` line from `Progress.md` (default `0.0.0`),
- bumps the patch number,
- rewrites the `<!-- auto-stamp:start --> ... <!-- auto-stamp:end -->` block
  near the top of `Progress.md` with the new version and current UTC
  timestamp,
- `git add`s `Progress.md` so the stamp is part of the commit being made.

The hook is a no-op when:

- the commit message starts with `WIP:` (read from
  `$(git rev-parse --git-dir)/COMMIT_EDITMSG`), or
- the caller passed `--no-verify` to git (git skips the hook entirely).

Stamp failures never block a commit — they log a warning to stderr and exit
`0`. Re-running `install-hooks.sh` is safe; it overwrites the hook file in
place.

## Enforcer

The Enforcer is a pogo crew agent that periodically re-reads `Vision.md` and
scans `~/Dev/Craftmine` for style/architecture violations. Small violations
(≤10 lines of net change) it fixes directly on a fresh
`enforcer-YYYYMMDD-HHMMSS` branch and submits to the refinery. Bigger ones it
files as `mg` tickets and mails `dispatch-ready: <id>` to the mayor.

The prompt source of truth lives at `tools/enforcer-prompt.md` in this repo.
The installer copies it to `~/.pogo/agents/enforcer.md` and registers the
hourly review-cycle schedule:

```
bash tools/install-enforcer.sh
```

The installer is idempotent — re-running overwrites the prompt and replaces
the schedule entry in place. After install, activate the agent without
waiting for the next pogod restart:

```
pogo agent start enforcer
```

Verify with `pogo agent list` (enforcer should show running) and
`pogo schedule list --agent enforcer` (the `review-cycle-enforcer` entry
should be present).

The Enforcer never edits `Vision.md` or `Progress.md`, never pushes to
`main` directly, never enforces outside `~/Dev/Craftmine`, and never
auto-reformats code (that's the pre-commit hook's job).
