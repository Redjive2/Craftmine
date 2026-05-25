package worldview

import (
	"github.com/redjive2/Craftmine/blocks"
	"github.com/redjive2/Craftmine/world"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"
)

// View is the behavior interface for the world-view module.
//
// Callers depend on View, not on the concrete Impl, so a test double can
// substitute (e.g. a renderer that counts meshes without touching the
// real graphic pipeline).
type View interface {
	Build(w world.Model, worldImpl world.World, registry blocks.Model, blocksImpl blocks.Blocks) Model
}

// Impl is the zero-field implementation of View. All behavior hangs off
// Impl; state lives in Model and is passed as an argument or returned by
// each method.
type Impl struct{}

// Compile-time check that Impl satisfies View.
var _ View = Impl{}

// Build assembles a render Model for w. Materials and the shared cube
// geometry are constructed once and reused across every mesh so GPU state
// changes per frame stay bounded.
func (Impl) Build(w world.Model, worldImpl world.World, registry blocks.Model, blocksImpl blocks.Blocks) Model {
	materials := buildMaterials(registry, blocksImpl)
	cube := geometry.NewCube(1)
	return Model{
		materials: materials,
		cube:      cube,
		surfaces:  buildSurfaceMeshes(w, cube, materials),
		trees:     buildTreeMeshes(w, worldImpl, cube, materials),
	}
}

// buildMaterials returns one shared *material.Standard per registered
// block kind, keyed by BlockID.
func buildMaterials(registry blocks.Model, blocksImpl blocks.Blocks) map[blocks.BlockID]*material.Standard {
	out := make(map[blocks.BlockID]*material.Standard)
	for _, b := range blocksImpl.All(registry) {
		r, g, blu := b.Color()
		out[b.ID()] = material.NewStandard(&math32.Color{R: r, G: g, B: blu})
	}
	return out
}

// buildSurfaceMeshes adds one grass cube at the top of every column. The
// shared cube geometry keeps per-mesh memory small.
func buildSurfaceMeshes(m world.Model, cube *geometry.Geometry, mats map[blocks.BlockID]*material.Standard) *core.Node {
	group := core.NewNode()
	grassMat := mats[m.Grass()]
	for x := 0; x < m.Width(); x++ {
		for z := 0; z < m.Depth(); z++ {
			y := m.HeightAt(x, z)
			mesh := graphic.NewMesh(cube, grassMat)
			mesh.SetPosition(float32(x), float32(y), float32(z))
			group.Add(mesh)
		}
	}
	return group
}

// buildTreeMeshes iterates each tree's trunk and canopy blocks and adds
// them as individual cubes. Iterating the tree shape directly is faster
// than scanning World.BlockAt because it avoids per-voxel chunk lookups.
//
// The chunk-bounds check matches cmd/world-demo: TreesInChunk reports a
// tree under every chunk its canopy touches, so we filter to the chunk
// rooted by the trunk to avoid drawing each tree multiple times.
func buildTreeMeshes(m world.Model, impl world.World, cube *geometry.Geometry, mats map[blocks.BlockID]*material.Standard) *core.Node {
	group := core.NewNode()
	woodMat := mats[m.Wood()]
	leavesMat := mats[m.Leaves()]
	for cx := 0; cx < m.ChunkCountX(); cx++ {
		for cz := 0; cz < m.ChunkCountZ(); cz++ {
			for _, tr := range impl.TreesInChunk(m, cx, cz) {
				if tr.X()/world.ChunkSize != cx || tr.Z()/world.ChunkSize != cz {
					continue
				}
				for ty := tr.BaseHeight(); ty < tr.BaseHeight()+tr.TrunkHeight(); ty++ {
					mesh := graphic.NewMesh(cube, woodMat)
					mesh.SetPosition(float32(tr.X()), float32(ty), float32(tr.Z()))
					group.Add(mesh)
				}
				for _, p := range canopyPositions(tr) {
					mesh := graphic.NewMesh(cube, leavesMat)
					mesh.SetPosition(float32(p.x), float32(p.y), float32(p.z))
					group.Add(mesh)
				}
			}
		}
	}
	return group
}

// voxel is a plain (x, y, z) triple used by canopyPositions. Kept package-
// private since worldview is the only caller.
type voxel struct{ x, y, z int }

// canopyPositions enumerates the leaf-block positions of a tree, matching
// the shape used by world.blockInTree: two cropped-corner layers at TopY
// and TopY+1 plus a smaller plus-shaped cap at TopY+2.
func canopyPositions(t world.Tree) []voxel {
	var out []voxel
	topY := t.TopY()
	r := t.CanopyRadius()
	for dy := 0; dy <= 1; dy++ {
		for dx := -r; dx <= r; dx++ {
			for dz := -r; dz <= r; dz++ {
				ax, az := absInt(dx), absInt(dz)
				if ax == r && az == r {
					continue
				}
				if dx == 0 && dz == 0 {
					continue
				}
				out = append(out, voxel{t.X() + dx, topY + dy, t.Z() + dz})
			}
		}
	}
	for dx := -1; dx <= 1; dx++ {
		for dz := -1; dz <= 1; dz++ {
			if absInt(dx) == 1 && absInt(dz) == 1 {
				continue
			}
			out = append(out, voxel{t.X() + dx, topY + 2, t.Z() + dz})
		}
	}
	return out
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
