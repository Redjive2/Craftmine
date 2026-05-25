// Command Craftmine opens an empty g3n window. ESC closes it.
//
// This binary is intentionally tiny: it wires the g3n event loop to the
// example app module (see app/), which demonstrates the Model / Impl /
// interface pattern that every Craftmine module follows.
package main

import (
	"time"

	"github.com/redjive2/Craftmine/app"

	g3napp "github.com/g3n/engine/app"
	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/core"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/renderer"
	"github.com/g3n/engine/window"
)

func main() {
	a := g3napp.App()
	scene := core.NewNode()

	cam := camera.New(1)
	scene.Add(cam)

	a.Gls().ClearColor(0.1, 0.1, 0.1, 1.0)

	// Application state lives in app.Model. main depends on the app.App
	// interface, not the concrete app.Impl, so the module is swappable.
	var application app.App = app.Impl{}
	state := app.New()

	a.Subscribe(window.OnKeyDown, func(_ string, ev interface{}) {
		kev := ev.(*window.KeyEvent)
		if kev.Key == window.KeyEscape {
			state = application.Stop(state)
			a.Exit()
		}
	})

	a.Run(func(rend *renderer.Renderer, _ time.Duration) {
		if !application.IsRunning(state) {
			a.Exit()
			return
		}
		a.Gls().Clear(gls.DEPTH_BUFFER_BIT | gls.STENCIL_BUFFER_BIT | gls.COLOR_BUFFER_BIT)
		rend.Render(scene, cam)
	})
}
