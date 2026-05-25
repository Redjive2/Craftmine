// Command Craftmine opens into the main menu.
//
// The wiring here is intentionally thin: every piece of state lives in a
// module Model (app.Model, menu.Model) and every transition is named on
// the corresponding Impl. main.go's job is to translate g3n window/GUI
// events into those named transitions and to drive the render loop.
package main

import (
	"fmt"
	"time"

	"github.com/redjive2/Craftmine/app"
	"github.com/redjive2/Craftmine/menu"

	g3napp "github.com/g3n/engine/app"
	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/core"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/gui"
	"github.com/g3n/engine/math32"
	"github.com/g3n/engine/renderer"
	"github.com/g3n/engine/window"
)

// Visual constants for the menu layout. Pixel values, not world units.
const (
	buttonWidth  float32 = 240
	buttonHeight float32 = 56
	buttonGap    float32 = 20
	titleSizePt  float64 = 48
)

func main() {
	a := g3napp.App()
	scene := core.NewNode()

	// Route gui events (mouse/keyboard) into panels under this scene.
	gui.Manager().Set(scene)

	cam := camera.New(1)
	scene.Add(cam)

	// Simplified minecraft-style sky backdrop.
	a.Gls().ClearColor(0.45, 0.65, 0.85, 1.0)

	var application app.App = app.Impl{}
	state := app.New()

	var menuApp menu.Menu = menu.Impl{}
	menuState := menu.New()

	title := gui.NewLabel("Craftmine")
	title.SetFontSize(titleSizePt)
	title.SetColor4(&math32.Color4{R: 1, G: 1, B: 1, A: 1})
	scene.Add(title)

	// One button per menu item. Each button captures its item's Choice
	// in the closure so the OnClick handler can route through the
	// menu Impl's Select transition (which rejects disabled choices).
	items := menuApp.Items(menuState)
	buttons := make([]*gui.Button, len(items))
	for i, it := range items {
		b := gui.NewButton(it.Label())
		b.SetSize(buttonWidth, buttonHeight)
		choice := it.Choice()
		b.Subscribe(gui.OnClick, func(_ string, _ interface{}) {
			menuState = menuApp.Select(menuState, choice)
		})
		if !it.Enabled() {
			b.SetEnabled(false)
		}
		buttons[i] = b
		scene.Add(b)
	}

	// layout recomputes positions for the title and button stack so
	// they stay centered as the window resizes. Called once at startup
	// and again on every OnWindowSize event.
	layout := func() {
		width, height := a.GetSize()
		fw, fh := float32(width), float32(height)
		a.Gls().Viewport(0, 0, int32(width), int32(height))
		cam.SetAspect(fw / fh)

		title.SetPosition((fw-title.Width())/2, fh*0.18)

		topY := fh*0.20 + 120
		for i, b := range buttons {
			x := (fw - buttonWidth) / 2
			y := topY + float32(i)*(buttonHeight+buttonGap)
			b.SetPosition(x, y)
		}
	}
	a.Subscribe(window.OnWindowSize, func(_ string, _ interface{}) { layout() })
	layout()

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
		if menuApp.IsDone(menuState) {
			// Stub world-creation. mg-7522 will replace this with a
			// real transition into the world module.
			switch menuApp.Selected(menuState) {
			case menu.ChoiceNewGame:
				fmt.Println("would create world")
			}
			state = application.Stop(state)
			a.Exit()
			return
		}
		a.Gls().Clear(gls.DEPTH_BUFFER_BIT | gls.STENCIL_BUFFER_BIT | gls.COLOR_BUFFER_BIT)
		rend.Render(scene, cam)
	})
}
