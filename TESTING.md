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
