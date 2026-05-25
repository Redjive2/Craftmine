package worldview

import (
	"testing"

	"github.com/redjive2/Craftmine/blocks"
	"github.com/redjive2/Craftmine/world"
)

// smallOptions matches the world package's test-friendly size: small
// enough to iterate every block in a reasonable time but big enough to
// contain several chunks and a handful of trees.
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

func generate(t *testing.T, seed int64) (world.Model, world.World, blocks.Model, blocks.Blocks) {
	t.Helper()
	registry, blocksImpl := defaultRegistry(t)
	var worldImpl world.World = world.Impl{}
	m, err := worldImpl.Generate(seed, registry, blocksImpl, smallOptions())
	if err != nil {
		t.Fatalf("world.Generate(seed=%d): %v", seed, err)
	}
	return m, worldImpl, registry, blocksImpl
}

// TestBuildSurfaceMeshCount verifies Build produces exactly one surface
// cube per world column. If this count drifts, either we're skipping
// columns or double-rendering them — both of which would be visible at
// runtime as terrain gaps or z-fighting.
func TestBuildSurfaceMeshCount(t *testing.T) {
	wm, wImpl, reg, bImpl := generate(t, 1)
	view := Impl{}.Build(wm, wImpl, reg, bImpl)
	want := wm.Width() * wm.Depth()
	got := len(view.Surfaces().Children())
	if got != want {
		t.Fatalf("surface mesh count = %d, want %d (width*depth)", got, want)
	}
}

// TestBuildTreeMeshCount verifies every tree contributes exactly its
// trunk blocks plus its canopy blocks, and no tree is double-counted
// across chunks even when its canopy overlaps a neighbor chunk.
func TestBuildTreeMeshCount(t *testing.T) {
	wm, wImpl, reg, bImpl := generate(t, 1)
	view := Impl{}.Build(wm, wImpl, reg, bImpl)

	want := 0
	for _, tr := range wm.Trees() {
		want += tr.TrunkHeight()
		want += len(canopyPositions(tr))
	}
	got := len(view.Trees().Children())
	if got != want {
		t.Fatalf("tree mesh count = %d, want %d (sum of trunk + canopy per tree)", got, want)
	}
}

// TestMaterialPerRegisteredBlock verifies Build registers a Standard
// material for every block kind in the registry. Without this, mesh
// construction silently uses a nil material — which would crash later
// during render rather than fail fast at build time.
func TestMaterialPerRegisteredBlock(t *testing.T) {
	wm, wImpl, reg, bImpl := generate(t, 1)
	view := Impl{}.Build(wm, wImpl, reg, bImpl)

	for _, b := range bImpl.All(reg) {
		if view.Material(b.ID()) == nil {
			t.Fatalf("Material(%d %q) = nil, want non-nil material", b.ID(), b.Name())
		}
	}
}

// TestCubeIsShared verifies the cube geometry is constructed once and
// reused across meshes. The test is indirect: we only check Cube() is
// non-nil, since walking mesh.GetGraphic().GetGeometry() requires the
// graphic interface we don't want to depend on here. The per-mesh
// memory math relies on this sharing.
func TestCubeIsShared(t *testing.T) {
	wm, wImpl, reg, bImpl := generate(t, 1)
	view := Impl{}.Build(wm, wImpl, reg, bImpl)
	if view.Cube() == nil {
		t.Fatalf("Cube() = nil, want a shared cube geometry")
	}
}

// TestCanopyPositionsMatchBlockAt is the load-bearing parity test
// between the renderer's canopy enumeration and the world module's
// per-voxel BlockAt. If the two ever drift, the rendered tree shape
// will not match the simulated tree shape — block-place/destroy code
// downstream would silently miss canopy voxels.
func TestCanopyPositionsMatchBlockAt(t *testing.T) {
	wm, wImpl, _, _ := generate(t, 1)
	if len(wm.Trees()) == 0 {
		t.Fatalf("seed produced no trees; cannot test canopy parity")
	}
	for i, tr := range wm.Trees() {
		for _, p := range canopyPositions(tr) {
			if p.x < 0 || p.x >= wm.Width() || p.z < 0 || p.z >= wm.Depth() ||
				p.y < 0 || p.y >= wm.MaxHeight() {
				continue // out-of-world canopy fringe; world doesn't store it
			}
			got := wImpl.BlockAt(wm, p.x, p.y, p.z)
			if got != wm.Leaves() && got != wm.Wood() {
				t.Fatalf("tree[%d] canopy block at (%d, %d, %d) = %d, want leaves(%d) or wood(%d)",
					i, p.x, p.y, p.z, got, wm.Leaves(), wm.Wood())
			}
		}
	}
}

// TestBuildIsDeterministic verifies Build of the same world twice
// produces the same mesh counts. World.Generate is deterministic; the
// renderer must be too so screenshots / replays stay reproducible.
func TestBuildIsDeterministic(t *testing.T) {
	wm, wImpl, reg, bImpl := generate(t, 42)

	a := Impl{}.Build(wm, wImpl, reg, bImpl)
	b := Impl{}.Build(wm, wImpl, reg, bImpl)

	if len(a.Surfaces().Children()) != len(b.Surfaces().Children()) {
		t.Fatalf("surface counts differ across two Builds: %d vs %d",
			len(a.Surfaces().Children()), len(b.Surfaces().Children()))
	}
	if len(a.Trees().Children()) != len(b.Trees().Children()) {
		t.Fatalf("tree counts differ across two Builds: %d vs %d",
			len(a.Trees().Children()), len(b.Trees().Children()))
	}
}

// TestImplSatisfiesViewInterface mirrors the compile-time-style
// assertion used in the other modules' tests.
func TestImplSatisfiesViewInterface(t *testing.T) {
	var _ View = Impl{}
}
