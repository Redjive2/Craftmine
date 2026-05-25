# Craftmine Progress

<!-- auto-stamp:start -->
Version: 0.0.7
Last commit: 2026-05-25T20:18:18Z
<!-- auto-stamp:end -->

Tracking implementation against [Vision.md](./Vision.md). Maintained by the mayor.

## Status

- **Repo state**: initialized as git repo on 2026-05-24 (root commit added `Vision.md` and `projects.json`).
- **GitHub**: not yet set up. Target is `redjive2/Craftmine` (public). Waiting on `gh auth login`.
- **Architect agent**: blocked — `pogo agent start architect` fails until pogod is restarted with an updated build. Once pogod is restarted, retry `pogo agent start architect`.
- **Polecat cap**: 5 concurrent (per directive).

## Work Items

### In flight

| ID | Title | Polecat | Notes |
|----|-------|---------|-------|
| mg-104a | block types (grass, dirt, stone, wood, leaves) | blocks-104a | Use a registry pattern so mods can add types later. |
| mg-f8e6 | main menu (New Game + Resume Game) | menu-f8e6 | Resume Game stays hard-coded disabled until save/load lands. |

### Pending (dependencies not met)

| ID | Title | Depends on | Notes |
|----|-------|------------|-------|
| mg-4901 | 512×512 world with terrain generation | mg-104a | Chunk the world (e.g. 16×16) for basic perf. Scaffold dep met. |

### Planned (not yet filed)

- Save/load (needed before `Resume Game` can be enabled in mg-f8e6).
- Player camera/controls beyond fly-around (block place/destroy, etc.).

### Done

| ID | Title | Notes |
|----|-------|-------|
| mg-99eb | scaffold Go module + g3n window | Polecat work landed via manual merge (refinery couldn't push — no remote yet). Commit `eca3288`, merged at `39af928`. |

## Known Issues

- **Polecat spawn requires manual attach.** Spawned polecats receive the prompt pasted into their Claude Code input but Enter is never sent — they sit idle until a human attaches (`pogo agent attach <name>`) and submits. Same root cause as the architect start failure. Workaround until pogod is rebuilt.
- **Refinery merges fail at push step.** No git remote configured yet (GitHub repo not created). Mayor manually merges polecat branches into main as a workaround. This will continue for `mg-104a`, `mg-f8e6`, `mg-4901` until `redjive2/Craftmine` is set up on GitHub.

## Design Notes

- **Performance + modularity first** (per Vision.md). Features that aren't core engine should be deferrable to mods later — keep gameplay code thin and behind clear seams.
- Each work item targets a single concern. Avoid bundling menu work into the scaffold, etc.
