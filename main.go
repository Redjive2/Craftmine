// Command Craftmine opens into the main menu and, on New Game, transitions
// into a 3D world view.
//
// The wiring here is intentionally thin: every piece of state lives in a
// module Model (app.Model, menu.Model, world.Model, worldview.Model) and
// every transition is named on the corresponding Impl. main.go's job is
// to translate g3n window/GUI events into those named transitions and to
// drive the render loop. The render loop has two modes — menu and world
// — and switches once Menu.IsDone reports the user picked New Game.
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/redjive2/Craftmine/app"
	"github.com/redjive2/Craftmine/blocks"
	"github.com/redjive2/Craftmine/menu"
	"github.com/redjive2/Craftmine/world"
	"github.com/redjive2/Craftmine/worldview"

	g3napp "github.com/g3n/engine/app"
	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/core"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/gui"
	"github.com/g3n/engine/light"
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

// World-generation defaults used when the user clicks New Game. The size
// is intentionally smaller than the Vision.md 512x512 target so the menu-
// to-world transition feels instant; users wanting the full-size view can
// run cmd/world-demo with -size=512.
const (
	newGameSeed      int64 = 2026
	newGameWidth           = 96
	newGameDepth           = 96
	newGameMaxHeight       = 48
)

func main() {
	a := g3napp.App()
	menuScene := core.NewNode()
	worldScene := core.NewNode()

	// Route gui events (mouse/keyboard) into panels under the menu scene
	// for now. On the menu->world transition we point the manager at the
	// (panel-less) world scene so menu buttons stop receiving clicks and
	// the orbit camera gets unhandled mouse events.
	gui.Manager().Set(menuScene)

	menuCam := camera.New(1)
	menuScene.Add(menuCam)

	a.Gls().ClearColor(0.45, 0.65, 0.85, 1.0)

	var application app.App = app.Impl{}
	state := app.New()

	var menuApp menu.Menu = menu.Impl{}
	menuState := menu.New()

	var worldImpl world.World = world.Impl{}
	var viewImpl worldview.View = worldview.Impl{}

	title := gui.NewLabel("Craftmine")
	title.SetFontSize(titleSizePt)
	title.SetColor4(&math32.Color4{R: 1, G: 1, B: 1, A: 1})
	menuScene.Add(title)

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
		menuScene.Add(b)
	}

	// World-mode state. worldCam is the active 3D camera; worldOrbit is
	// kept alive in this scope so its event subscriptions don't get
	// dropped by GC. worldStarted gates the one-shot transition.
	var worldCam *camera.Camera
	var worldOrbit *camera.OrbitControl
	worldStarted := false

	// layout recomputes positions for the title and button stack so
	// they stay centered as the window resizes. It also updates the
	// world camera's aspect ratio once the world is up.
	//
	// Two coordinate systems are at play. The GL viewport is in physical
	// pixels (GetFramebufferSize), since glViewport addresses the
	// framebuffer directly. GUI panel positions are in logical pixels
	// (GetSize), since gui.Panel.SetModelMatrix multiplies by the window
	// DPI scale to convert. Mixing them up — e.g. setting the viewport
	// to logical size on Retina — leaves the world rendered into a
	// quadrant of the framebuffer and the menu drawn at 2x its intended
	// size with hitboxes still in the correct (logical) place.
	layout := func() {
		width, height := a.GetSize()
		fbw, fbh := a.GetFramebufferSize()
		fw, fh := float32(width), float32(height)
		a.Gls().Viewport(0, 0, int32(fbw), int32(fbh))
		menuCam.SetAspect(fw / fh)
		if worldCam != nil {
			worldCam.SetAspect(fw / fh)
		}

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
		if !worldStarted && menuApp.IsDone(menuState) {
			switch menuApp.Selected(menuState) {
			case menu.ChoiceNewGame:
				cam, orbit, err := startWorld(worldScene, viewImpl, worldImpl, a)
				if err != nil {
					log.Printf("craftmine: failed to start world: %v", err)
					state = application.Stop(state)
					a.Exit()
					return
				}
				worldCam = cam
				worldOrbit = orbit
				_ = worldOrbit
				gui.Manager().Set(worldScene)
				worldStarted = true
				layout()
			}
		}
		a.Gls().Clear(gls.DEPTH_BUFFER_BIT | gls.STENCIL_BUFFER_BIT | gls.COLOR_BUFFER_BIT)
		if worldStarted {
			if err := rend.Render(worldScene, worldCam); err != nil {
				log.Printf("craftmine: render error: %v", err)
			}
			return
		}
		if err := rend.Render(menuScene, menuCam); err != nil {
			log.Printf("craftmine: render error: %v", err)
		}
	})
}

// startWorld wires up the world scene: build a block registry, generate
// the world, build meshes through the worldview Impl, add lighting, and
// stand up the orbit camera. Returns the world camera and orbit control
// so main can update their aspect on window resize and keep the orbit
// reachable for its event subscriptions.
func startWorld(scene *core.Node, viewImpl worldview.View, worldImpl world.World, a *g3napp.Application) (*camera.Camera, *camera.OrbitControl, error) {
	var blocksImpl blocks.Blocks = blocks.Impl{}
	registry, err := blocks.NewWithDefaults(blocksImpl)
	if err != nil {
		return nil, nil, fmt.Errorf("blocks registry: %w", err)
	}

	opts := world.GenerateOptions{
		Width:     newGameWidth,
		Depth:     newGameDepth,
		MaxHeight: newGameMaxHeight,
		DirtDepth: world.DefaultDirtDepth,
	}
	model, err := worldImpl.Generate(newGameSeed, registry, blocksImpl, opts)
	if err != nil {
		return nil, nil, fmt.Errorf("world generate: %w", err)
	}
	log.Printf("craftmine: generated %dx%dx%d world, seed=%d, %d trees",
		model.Width(), model.Depth(), model.MaxHeight(), newGameSeed, model.TreeCount())

	view := viewImpl.Build(model, worldImpl, registry, blocksImpl)
	scene.Add(view.Surfaces())
	scene.Add(view.Trees())

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

	width, height := a.GetSize()
	cam.SetAspect(float32(width) / float32(height))

	orbit := camera.NewOrbitControl(cam)
	orbit.SetTarget(*center)

	return cam, orbit, nil
}
