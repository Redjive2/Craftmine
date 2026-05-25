package world_test

import (
	"strings"
	"testing"

	"github.com/redjive2/Craftmine/blocks"
	"github.com/redjive2/Craftmine/world"
)

// smallOptions is the test-friendly world size: small enough to iterate every
// block in a reasonable time, but big enough to actually contain trees and a
// few chunks. 64x64x64 yields 16 chunks (4x4) and ~25-50 trees typically.
func smallOptions() world.GenerateOptions {
	return world.GenerateOptions{Width: 64, Depth: 64, MaxHeight: 64, DirtDepth: 3}
}

func defaultRegistry(t *testing.T) (blocks.Model, blocks.Blocks) {
	t.Helper()
	var blocksImpl blocks.Blocks = blocks.Impl{}
	registry, err := blocks.NewWithDefaults(blocksImpl)
	if err != nil {
		t.Fatalf("blocks.NewWithDefaults: %v", err)
	}
	return registry, blocksImpl
}

// TestGenerateDeterministic verifies the headline purity guarantee from
// Vision.md: same seed + same options yields the same Model byte-for-byte
// (here checked via observable accessors).
func TestGenerateDeterministic(t *testing.T) {
	registry, blocksImpl := defaultRegistry(t)
	var worldImpl world.World = world.Impl{}

	a, err := worldImpl.Generate(12345, registry, blocksImpl, smallOptions())
	if err != nil {
		t.Fatalf("Generate A: %v", err)
	}
	b, err := worldImpl.Generate(12345, registry, blocksImpl, smallOptions())
	if err != nil {
		t.Fatalf("Generate B: %v", err)
	}

	if a.Width() != b.Width() || a.Depth() != b.Depth() || a.MaxHeight() != b.MaxHeight() {
		t.Fatalf("dimensions diverged between two generates with same seed")
	}
	if a.TreeCount() != b.TreeCount() {
		t.Fatalf("tree count diverged: %d vs %d", a.TreeCount(), b.TreeCount())
	}
	for x := 0; x < a.Width(); x++ {
		for z := 0; z < a.Depth(); z++ {
			if a.HeightAt(x, z) != b.HeightAt(x, z) {
				t.Fatalf("heightmap diverged at (%d, %d): %d vs %d",
					x, z, a.HeightAt(x, z), b.HeightAt(x, z))
			}
		}
	}
	treesA, treesB := a.Trees(), b.Trees()
	for i := range treesA {
		if treesA[i].X() != treesB[i].X() || treesA[i].Z() != treesB[i].Z() ||
			treesA[i].BaseHeight() != treesB[i].BaseHeight() ||
			treesA[i].TrunkHeight() != treesB[i].TrunkHeight() {
			t.Fatalf("tree[%d] diverged: %+v vs %+v", i, treesA[i], treesB[i])
		}
	}
}

// TestDifferentSeedsDifferTerrain confirms two different seeds yield
// non-identical heightmaps. Without this, a bug in seed plumbing could make
// every world look the same.
func TestDifferentSeedsDifferTerrain(t *testing.T) {
	registry, blocksImpl := defaultRegistry(t)
	var worldImpl world.World = world.Impl{}

	a, err := worldImpl.Generate(1, registry, blocksImpl, smallOptions())
	if err != nil {
		t.Fatalf("Generate seed=1: %v", err)
	}
	b, err := worldImpl.Generate(2, registry, blocksImpl, smallOptions())
	if err != nil {
		t.Fatalf("Generate seed=2: %v", err)
	}

	differences := 0
	for x := 0; x < a.Width(); x++ {
		for z := 0; z < a.Depth(); z++ {
			if a.HeightAt(x, z) != b.HeightAt(x, z) {
				differences++
			}
		}
	}
	// Vast majority of columns should differ; require at least 50% to guard
	// against the noise function ignoring its seed input.
	if differences < a.Width()*a.Depth()/2 {
		t.Fatalf("only %d / %d columns differ between seeds 1 and 2 — seed may be unused",
			differences, a.Width()*a.Depth())
	}
}

// TestHeightsWithinBounds checks every column's surface y stays inside
// [1, maxHeight-1]. Heights at 0 leave no stone floor; heights at maxHeight
// leave no room for trees.
func TestHeightsWithinBounds(t *testing.T) {
	registry, blocksImpl := defaultRegistry(t)
	var worldImpl world.World = world.Impl{}

	m, err := worldImpl.Generate(42, registry, blocksImpl, smallOptions())
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	for x := 0; x < m.Width(); x++ {
		for z := 0; z < m.Depth(); z++ {
			h := m.HeightAt(x, z)
			if h < 1 || h >= m.MaxHeight() {
				t.Fatalf("height at (%d, %d) = %d out of bounds [1, %d)",
					x, z, h, m.MaxHeight())
			}
		}
	}
}

// TestLayering walks every column and verifies the grass/dirt/stone layering
// holds (away from trees): the surface block is grass, the next DirtDepth
// blocks down are dirt, everything below is stone, everything above is air.
func TestLayering(t *testing.T) {
	registry, blocksImpl := defaultRegistry(t)
	var worldImpl world.World = world.Impl{}

	m, err := worldImpl.Generate(7, registry, blocksImpl, smallOptions())
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Build a set of (x, z) columns that contain a tree so we can skip them
	// when checking "above surface is air" — trunks legitimately occupy that
	// space.
	treeColumn := make(map[[2]int]bool)
	for _, tr := range m.Trees() {
		for dx := -tr.CanopyRadius() - 1; dx <= tr.CanopyRadius()+1; dx++ {
			for dz := -tr.CanopyRadius() - 1; dz <= tr.CanopyRadius()+1; dz++ {
				treeColumn[[2]int{tr.X() + dx, tr.Z() + dz}] = true
			}
		}
	}

	for x := 0; x < m.Width(); x++ {
		for z := 0; z < m.Depth(); z++ {
			h := m.HeightAt(x, z)

			if got := worldImpl.BlockAt(m, x, h, z); got != m.Grass() {
				t.Fatalf("surface at (%d, %d, %d) = %d, want grass (%d)",
					x, h, z, got, m.Grass())
			}
			for dy := 1; dy <= m.DirtDepth() && h-dy >= 0; dy++ {
				if got := worldImpl.BlockAt(m, x, h-dy, z); got != m.Dirt() {
					t.Fatalf("at (%d, %d, %d) y=h-%d = %d, want dirt (%d)",
						x, h-dy, z, dy, got, m.Dirt())
				}
			}
			deepY := h - m.DirtDepth() - 1
			if deepY >= 0 {
				if got := worldImpl.BlockAt(m, x, deepY, z); got != m.Stone() {
					t.Fatalf("deep (%d, %d, %d) = %d, want stone (%d)",
						x, deepY, z, got, m.Stone())
				}
			}
			if !treeColumn[[2]int{x, z}] {
				if got := worldImpl.BlockAt(m, x, h+1, z); got != world.Air {
					t.Fatalf("above-surface (%d, %d, %d) = %d, want Air (no tree nearby)",
						x, h+1, z, got)
				}
			}
		}
	}
}

// TestTreesGenerated checks at least a few trees were placed (the default
// world target is "forested hills") and every tree has wood at its base and
// at least some leaves above.
func TestTreesGenerated(t *testing.T) {
	registry, blocksImpl := defaultRegistry(t)
	var worldImpl world.World = world.Impl{}

	m, err := worldImpl.Generate(99, registry, blocksImpl, smallOptions())
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if m.TreeCount() < 5 {
		t.Fatalf("only %d trees generated in a 64x64 world; expected at least 5", m.TreeCount())
	}
	for i, tr := range m.Trees() {
		if got := worldImpl.BlockAt(m, tr.X(), tr.BaseHeight(), tr.Z()); got != m.Wood() {
			t.Fatalf("tree[%d] trunk base (%d, %d, %d) = %d, want wood (%d)",
				i, tr.X(), tr.BaseHeight(), tr.Z(), got, m.Wood())
		}
		if got := worldImpl.BlockAt(m, tr.X(), tr.TopY(), tr.Z()); got != m.Wood() {
			t.Fatalf("tree[%d] trunk top (%d, %d, %d) = %d, want wood (%d)",
				i, tr.X(), tr.TopY(), tr.Z(), got, m.Wood())
		}
		// A canopy block one above and to the side of the trunk top should
		// be leaves.
		if got := worldImpl.BlockAt(m, tr.X()+1, tr.TopY(), tr.Z()); got != m.Leaves() {
			t.Fatalf("tree[%d] canopy at (%d, %d, %d) = %d, want leaves (%d)",
				i, tr.X()+1, tr.TopY(), tr.Z(), got, m.Leaves())
		}
	}
}

// TestBlockAtOutOfBounds checks the documented out-of-bounds behavior: Air
// rather than panic. Renderers and any future block-placement code can probe
// edges without special-casing.
func TestBlockAtOutOfBounds(t *testing.T) {
	registry, blocksImpl := defaultRegistry(t)
	var worldImpl world.World = world.Impl{}

	m, err := worldImpl.Generate(1, registry, blocksImpl, smallOptions())
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	cases := []struct{ x, y, z int }{
		{-1, 10, 10},
		{10, -1, 10},
		{10, 10, -1},
		{m.Width(), 10, 10},
		{10, m.MaxHeight(), 10},
		{10, 10, m.Depth()},
	}
	for _, c := range cases {
		if got := worldImpl.BlockAt(m, c.x, c.y, c.z); got != world.Air {
			t.Fatalf("BlockAt(%d, %d, %d) = %d, want Air", c.x, c.y, c.z, got)
		}
	}
}

// TestChunkCounts verifies the chunk math agrees with the configured
// dimensions: width=64 -> 4 chunks of size 16.
func TestChunkCounts(t *testing.T) {
	registry, blocksImpl := defaultRegistry(t)
	var worldImpl world.World = world.Impl{}

	m, err := worldImpl.Generate(0, registry, blocksImpl, smallOptions())
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if m.ChunkCountX() != 4 || m.ChunkCountZ() != 4 {
		t.Fatalf("ChunkCountX/Z = %d/%d, want 4/4", m.ChunkCountX(), m.ChunkCountZ())
	}
}

// TestTreesInChunkCoversAllTrees confirms every tree shows up under at least
// one chunk's TreesInChunk result. Without this guarantee, a chunk-iterating
// renderer would silently drop trees that sit on chunk boundaries.
func TestTreesInChunkCoversAllTrees(t *testing.T) {
	registry, blocksImpl := defaultRegistry(t)
	var worldImpl world.World = world.Impl{}

	m, err := worldImpl.Generate(13, registry, blocksImpl, smallOptions())
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	seen := make(map[[2]int]bool)
	for cx := 0; cx < m.ChunkCountX(); cx++ {
		for cz := 0; cz < m.ChunkCountZ(); cz++ {
			for _, tr := range worldImpl.TreesInChunk(m, cx, cz) {
				seen[[2]int{tr.X(), tr.Z()}] = true
			}
		}
	}
	for _, tr := range m.Trees() {
		if !seen[[2]int{tr.X(), tr.Z()}] {
			t.Fatalf("tree at (%d, %d) is not reported by any chunk", tr.X(), tr.Z())
		}
	}
}

// TestTreesReturnsCopy guards against accidental aliasing of the internal
// tree slice: mutating the returned slice must not affect future queries.
func TestTreesReturnsCopy(t *testing.T) {
	registry, blocksImpl := defaultRegistry(t)
	var worldImpl world.World = world.Impl{}

	m, err := worldImpl.Generate(5, registry, blocksImpl, smallOptions())
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	trees := m.Trees()
	if len(trees) == 0 {
		t.Skip("seed produced no trees; cannot test aliasing")
	}
	trees[0] = world.NewTree(-999, -999, 0, 0, 0)
	if m.Trees()[0].X() == -999 {
		t.Fatalf("Trees() returned aliased slice — mutation leaked into Model")
	}
}

// TestGenerateValidation covers the input-validation guarantees of Generate.
// Bad dimensions, non-multiple sizes, and missing registry entries should all
// be rejected up front rather than producing a malformed Model.
func TestGenerateValidation(t *testing.T) {
	registry, blocksImpl := defaultRegistry(t)
	var worldImpl world.World = world.Impl{}

	cases := []struct {
		name    string
		opts    world.GenerateOptions
		wantSub string
	}{
		{"zero width", world.GenerateOptions{Width: 0, Depth: 16, MaxHeight: 16, DirtDepth: 3}, "must all be positive"},
		{"negative depth", world.GenerateOptions{Width: 16, Depth: -1, MaxHeight: 16, DirtDepth: 3}, "must all be positive"},
		{"non-chunk-multiple width", world.GenerateOptions{Width: 17, Depth: 16, MaxHeight: 16, DirtDepth: 3}, "multiples of ChunkSize"},
		{"negative dirtDepth", world.GenerateOptions{Width: 16, Depth: 16, MaxHeight: 16, DirtDepth: -1}, "dirtDepth"},
		{"dirtDepth >= maxHeight", world.GenerateOptions{Width: 16, Depth: 16, MaxHeight: 4, DirtDepth: 4}, "less than maxHeight"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := worldImpl.Generate(0, registry, blocksImpl, tc.opts)
			if err == nil {
				t.Fatalf("Generate(%+v) returned nil error, want %q", tc.opts, tc.wantSub)
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("Generate error = %q, want substring %q", err.Error(), tc.wantSub)
			}
		})
	}
}

// TestGenerateMissingBlockKind verifies Generate fails fast if the registry is
// missing one of the kinds the world needs. The error message must name the
// missing kind so a future modder knows what to register.
func TestGenerateMissingBlockKind(t *testing.T) {
	var blocksImpl blocks.Blocks = blocks.Impl{}
	registry := blocks.New()
	// Register all kinds except wood.
	for _, b := range []blocks.Block{
		blocks.NewBlock(blocks.IDGrass, "grass", 0, 1, 0, true),
		blocks.NewBlock(blocks.IDDirt, "dirt", 0.5, 0.3, 0.1, true),
		blocks.NewBlock(blocks.IDStone, "stone", 0.5, 0.5, 0.5, true),
		blocks.NewBlock(blocks.IDLeaves, "leaves", 0.2, 0.5, 0.2, true),
	} {
		next, err := blocksImpl.Register(registry, b)
		if err != nil {
			t.Fatalf("Register %s: %v", b.Name(), err)
		}
		registry = next
	}

	var worldImpl world.World = world.Impl{}
	_, err := worldImpl.Generate(0, registry, blocksImpl, smallOptions())
	if err == nil {
		t.Fatalf("Generate returned nil error with no 'wood' registered")
	}
	if !strings.Contains(err.Error(), "wood") {
		t.Fatalf("error = %q, want it to name the missing kind 'wood'", err.Error())
	}
}

// TestDefaultOptionsTargetVision pins the default config to the Vision.md
// numbers. If this test fails, either Vision.md moved or someone accidentally
// shrank the target world.
func TestDefaultOptionsTargetVision(t *testing.T) {
	opts := world.DefaultOptions()
	if opts.Width != 512 || opts.Depth != 512 || opts.MaxHeight != 256 {
		t.Fatalf("DefaultOptions = %+v, want 512x512x256 per Vision.md", opts)
	}
}

// TestGenerateAtVisionScale exercises Generate at the full 512x512x256 target
// so we know the module actually works at the size the menu's New Game button
// will request. Skipped under -short for quick test runs.
func TestGenerateAtVisionScale(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping full-scale world generation under -short")
	}
	registry, blocksImpl := defaultRegistry(t)
	var worldImpl world.World = world.Impl{}

	m, err := worldImpl.Generate(2026, registry, blocksImpl, world.DefaultOptions())
	if err != nil {
		t.Fatalf("Generate at default size: %v", err)
	}
	if m.Width() != 512 || m.Depth() != 512 || m.MaxHeight() != 256 {
		t.Fatalf("size = %dx%dx%d, want 512x512x256", m.Width(), m.Depth(), m.MaxHeight())
	}
	if m.ChunkCountX() != 32 || m.ChunkCountZ() != 32 {
		t.Fatalf("chunk counts = %dx%d, want 32x32", m.ChunkCountX(), m.ChunkCountZ())
	}
	if m.TreeCount() < 100 {
		t.Fatalf("tree count at default size = %d, want >= 100", m.TreeCount())
	}
}
