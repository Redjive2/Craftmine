# Craftmine Progress

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
| mg-99eb | scaffold Craftmine Go module with g3n | scaffold-99eb | Bootstrap `go.mod`, add `g3n/engine`, minimal main window that exits on ESC. Required for everything else. |

### Pending (dependencies not met)

| ID | Title | Depends on | Notes |
|----|-------|------------|-------|
| mg-f8e6 | main menu (New Game + Resume Game) | mg-99eb | Resume Game stays hard-coded disabled until save/load lands. |
| mg-104a | block types (grass, dirt, stone, wood, leaves) | mg-99eb | Use a registry pattern so mods can add types later. |
| mg-4901 | 512×512 world with terrain generation | mg-99eb, mg-104a | Chunk the world (e.g. 16×16) for basic perf. |

When `mg-99eb` lands, `mg-f8e6` and `mg-104a` can dispatch in parallel (within 5-polecat cap). `mg-4901` waits for `mg-104a`.

### Planned (not yet filed)

- Save/load (needed before `Resume Game` can be enabled in mg-f8e6).
- Player camera/controls beyond fly-around (block place/destroy, etc.).

### Done

_None yet._

## Design Notes

- **Performance + modularity first** (per Vision.md). Features that aren't core engine should be deferrable to mods later — keep gameplay code thin and behind clear seams.
- Each work item targets a single concern. Avoid bundling menu work into the scaffold, etc.
