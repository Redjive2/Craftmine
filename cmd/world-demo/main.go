// Command world-demo opens a g3n window showing a generated world: hills as a
// blanket of grass-topped cubes, scattered trees of wood + leaves. It is the
// visual acceptance check for the world module (mg-7522).
//
// Controls:
//
//	left mouse + drag : rotate the orbit camera around the world center
//	right mouse + drag: pan
//	scroll wheel       : zoom
//	ESC                : close
//
// Run with: go run ./cmd/world-demo [-seed N] [-size N]
//
// Notes on performance: g3n issues one draw call per Mesh, so rendering every
// block of a 512x512 world would be unworkable. The demo renders the surface
// block of each column plus all tree blocks; that is enough to verify the
// hills-and-trees acceptance visually. Block geometry and per-kind materials
// are shared across meshes to keep GPU memory bounded.
package main

import (
	"flag"
	"log"
	"time"

	"github.com/redjive2/Craftmine/blocks"
	"github.com/redjive2/Craftmine/world"

	g3napp "github.com/g3n/engine/app"
	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/core"
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/light"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"
	"github.com/g3n/engine/renderer"
	"github.com/g3n/engine/window"
)

func main() {
	seed := flag.Int64("seed", 2026, "world generation seed")
	size := flag.Int("size", 96, "world width/depth (must be a multiple of 16)")
	maxY := flag.Int("height", 48, "world max vertical extent")
	flag.Parse()

	registry, blocksImpl := buildRegistry()

	var worldImpl world.World = world.Impl{}
	opts := world.GenerateOptions{
		Width:     *size,
		Depth:     *size,
		MaxHeight: *maxY,
		DirtDepth: world.DefaultDirtDepth,
	}
	model, err := worldImpl.Generate(*seed, registry, blocksImpl, opts)
	if err != nil {
		log.Fatalf("world-demo: Generate failed: %v", err)
	}
	log.Printf("world-demo: generated %dx%dx%d world, seed=%d, %d trees",
		model.Width(), model.Depth(), model.MaxHeight(), *seed, model.TreeCount())

	a := g3napp.App()
	scene := core.NewNode()

	materials := buildMaterials(registry, blocksImpl)
	cube := geometry.NewCube(1)
	scene.Add(buildSurfaceMeshes(model, cube, materials))
	scene.Add(buildTreeMeshes(model, worldImpl, cube, materials))

	scene.Add(light.NewAmbient(&math32.Color{R: 0.45, G: 0.45, B: 0.45}, 1.0))
	sun := light.NewDirectional(&math32.Color{R: 1.0, G: 0.95, B: 0.85}, 1.1)
	sun.SetPosition(0.5, 1.0, 0.6)
	scene.Add(sun)

	center := &math32.Vector3{
		X: float32(model.Width()) / 2,
		Y: float32(model.MaxHeight()) / 4,
		Z: float32(model.Depth()) / 2,
	}
	cam := camera.New(1)
	cam.SetFar(float32(model.Width()) * 4)
	cam.SetPosition(center.X, center.Y+float32(model.MaxHeight()), center.Z+float32(model.Width()))
	cam.LookAt(center, &math32.Vector3{X: 0, Y: 1, Z: 0})
	scene.Add(cam)

	orbit := camera.NewOrbitControl(cam)
	orbit.SetTarget(*center)
	_ = orbit

	a.Gls().ClearColor(0.50, 0.70, 0.95, 1.0)

	a.Subscribe(window.OnKeyDown, func(_ string, ev interface{}) {
		if kev, ok := ev.(*window.KeyEvent); ok && kev.Key == window.KeyEscape {
			a.Exit()
		}
	})

	a.Run(func(rend *renderer.Renderer, _ time.Duration) {
		a.Gls().Clear(gls.DEPTH_BUFFER_BIT | gls.STENCIL_BUFFER_BIT | gls.COLOR_BUFFER_BIT)
		if err := rend.Render(scene, cam); err != nil {
			log.Printf("world-demo: render error: %v", err)
		}
	})
}

func buildRegistry() (blocks.Model, blocks.Blocks) {
	var blocksImpl blocks.Blocks = blocks.Impl{}
	registry, err := blocks.NewWithDefaults(blocksImpl)
	if err != nil {
		log.Fatalf("world-demo: build registry: %v", err)
	}
	return registry, blocksImpl
}

// buildMaterials returns one shared *material.Standard per registered block
// kind, keyed by BlockID. Sharing materials across meshes keeps GPU state
// changes per frame low.
func buildMaterials(registry blocks.Model, blocksImpl blocks.Blocks) map[blocks.BlockID]*material.Standard {
	out := make(map[blocks.BlockID]*material.Standard)
	for _, b := range blocksImpl.All(registry) {
		r, g, blu := b.Color()
		out[b.ID()] = material.NewStandard(&math32.Color{R: r, G: g, B: blu})
	}
	return out
}

// buildSurfaceMeshes adds one grass cube at the top of every column. It uses
// a shared geometry so per-mesh memory stays small.
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

// buildTreeMeshes iterates each tree's trunk and canopy blocks and adds them
// as individual cubes. Going through World.BlockAt would also work but is
// slower because of per-voxel chunk lookups; iterating the tree shape directly
// keeps the demo startup snappy.
func buildTreeMeshes(m world.Model, impl world.World, cube *geometry.Geometry, mats map[blocks.BlockID]*material.Standard) *core.Node {
	group := core.NewNode()
	woodMat := mats[m.Wood()]
	leavesMat := mats[m.Leaves()]
	for cx := 0; cx < m.ChunkCountX(); cx++ {
		for cz := 0; cz < m.ChunkCountZ(); cz++ {
			for _, tr := range impl.TreesInChunk(m, cx, cz) {
				if tr.X()/world.ChunkSize != cx || tr.Z()/world.ChunkSize != cz {
					continue // tree is from a neighbor chunk via canopy overlap
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

type voxel struct{ x, y, z int }

// canopyPositions enumerates the leaf-block positions of a tree, matching the
// shape used by world.blockInTree. Kept in sync with that function by
// construction: the same two-cropped-corner layers plus a smaller top cap.
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
					continue // trunk top — already wood
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
