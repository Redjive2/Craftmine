// Command blocks-demo opens a g3n window showing one cube per registered block
// kind, laid out in a row. It is the visual acceptance check for the blocks
// module: a reviewer should see five colored cubes (grass, dirt, stone, wood,
// leaves) from left to right. ESC closes the window.
//
// Run with: go run ./cmd/blocks-demo
package main

import (
	"log"
	"time"

	"github.com/redjive2/Craftmine/blocks"

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
	var impl blocks.Blocks = blocks.Impl{}
	registry, err := blocks.NewWithDefaults(impl)
	if err != nil {
		log.Fatalf("blocks-demo: failed to build default registry: %v", err)
	}

	a := g3napp.App()
	scene := core.NewNode()

	all := impl.All(registry)
	log.Printf("blocks-demo: %d registered block kinds", len(all))
	for index, block := range all {
		red, green, blue := block.Color()
		log.Printf("  [%d] id=%d name=%-7s color=(%.2f,%.2f,%.2f) solid=%v",
			index, block.ID(), block.Name(), red, green, blue, block.Solid())
		scene.Add(buildCube(block, index))
	}

	scene.Add(light.NewAmbient(&math32.Color{R: 0.4, G: 0.4, B: 0.4}, 1.0))
	directional := light.NewDirectional(&math32.Color{R: 1.0, G: 1.0, B: 1.0}, 1.0)
	directional.SetPosition(0.5, 1.0, 0.7)
	scene.Add(directional)

	cam := camera.New(1)
	cam.SetPosition(float32(len(all)-1), 1.5, float32(len(all))+2)
	cam.LookAt(&math32.Vector3{X: float32(len(all) - 1), Y: 0, Z: 0}, &math32.Vector3{X: 0, Y: 1, Z: 0})
	scene.Add(cam)

	a.Gls().ClearColor(0.1, 0.12, 0.18, 1.0)

	a.Subscribe(window.OnKeyDown, func(_ string, ev interface{}) {
		if kev, ok := ev.(*window.KeyEvent); ok && kev.Key == window.KeyEscape {
			a.Exit()
		}
	})

	a.Run(func(rend *renderer.Renderer, _ time.Duration) {
		a.Gls().Clear(gls.DEPTH_BUFFER_BIT | gls.STENCIL_BUFFER_BIT | gls.COLOR_BUFFER_BIT)
		if err := rend.Render(scene, cam); err != nil {
			log.Printf("blocks-demo: render error: %v", err)
		}
	})
}

// buildCube wires one Block descriptor into a g3n renderable, positioned along
// the X axis so consecutive blocks line up left-to-right.
func buildCube(block blocks.Block, index int) *graphic.Mesh {
	red, green, blue := block.Color()
	geom := geometry.NewCube(1)
	mat := material.NewStandard(&math32.Color{R: red, G: green, B: blue})
	mesh := graphic.NewMesh(geom, mat)
	mesh.SetPosition(float32(index*2), 0, 0)
	return mesh
}
