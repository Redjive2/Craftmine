# Craftmine Progress

Tracking implementation against [Vision.md](./Vision.md). Maintained by the mayor.

## Status

- **Repo state**: initialized as git repo on 2026-05-24 (root commit added `Vision.md` and `projects.json`).
- **Architect agent**: blocked — `pogo agent start architect` fails until pogod is restarted with an updated build. Once pogod is restarted, retry `pogo agent start architect`.

## Work Items

### In flight

| ID | Title | Polecat | Notes |
|----|-------|---------|-------|
| mg-99eb | scaffold Craftmine Go module with g3n | scaffold-99eb | Bootstrap `go.mod`, add `g3n/engine`, minimal main window that exits on ESC. No menu/world/blocks yet. |

### Planned (not yet filed)

Derived from `Vision.md` — file these as separate work items once the scaffold lands:

- Main menu screen with `New Game` (always enabled) and `Resume Game` (disabled when no save).
- World generation: 512×512 block world, height 0–256, simple terrain.
- Block types: grass, dirt, stone, wood, leaves.
- Hills + trees via simple terrain generator.
- Save/load (needed to make `Resume Game` meaningful).

### Done

_None yet._

## Design Notes

- **Performance + modularity first** (per Vision.md). Features that aren't core engine should be deferrable to mods later — keep gameplay code thin and behind clear seams.
- Each work item targets a single concern. Avoid bundling menu work into the scaffold, etc.
