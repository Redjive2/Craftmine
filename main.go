// Command Craftmine opens into the main menu and, on New Game, transitions
// into a 3D world view with a first-person player camera.
//
// The wiring here is intentionally thin: every piece of state lives in a
// module Model (app.Model, menu.Model, world.Model, worldview.Model,
// player.Model) and every transition is named on the corresponding Impl.
// main.go's job is to translate g3n window/GUI events into those named
// transitions and to drive the render loop. The render loop has two modes
// — menu and world — and switches once Menu.IsDone reports the user picked
// New Game.
//
// World mode is driven by the player module: each frame we collect WASD
// input from the keyboard state, accumulated mouse delta from OnCursor,
// and feed them into player.Impl.Tick. The g3n camera is repositioned
// from the resulting eye / look-target every frame.
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/redjive2/Craftmine/app"
	"github.com/redjive2/Craftmine/blocks"
	"github.com/redjive2/Craftmine/menu"
	"github.com/redjive2/Craftmine/player"
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
	"github.com/go-gl/glfw/v3.3/glfw"
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

// Mouse-look sensitivity, in radians per pixel of cursor movement. Tuned
// by feel: a full screen-width sweep yields roughly a half-turn.
const mouseSensitivity = 0.0035

func main() {
	a := g3napp.App()

	// Open fullscreen on the primary monitor. App() created an 800x600
	// windowed surface; switching now (before any layout / handlers) means
	// the very first frame is drawn at the monitor's native resolution.
	gw := a.IWindow.(*window.GlfwWindow)
	gw.SetFullscreen(true)

	menuScene := core.NewNode()
	worldScene := core.NewNode()

	// Route gui events (mouse/keyboard) into panels under the menu scene
	// for now. On the menu->world transition we point the manager at the
	// (panel-less) world scene so menu buttons stop receiving clicks.
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
	var playerImpl player.Player = player.Impl{}

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

	// World-mode state. worldCam is the active 3D camera; worldModel and
	// playerState hold the world / player Models for the active session.
	// worldStarted gates the one-shot transition.
	var worldCam *camera.Camera
	var worldModel world.Model
	var playerState player.Model
	worldStarted := false

	// Mouse-look delta accumulator. The OnCursor handler fills this; the
	// per-frame Tick consumes and resets it. Tracking the previous cursor
	// position lets us derive a delta even though g3n's OnCursor reports
	// absolute positions.
	var pendingYaw, pendingPitch float64
	var lastCursorX, lastCursorY float32
	cursorSeeded := false

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

	// OnCursor fires for every cursor movement. We only update look while
	// in world mode; the menu still uses the GUI's own mouse routing.
	// Seeding lastCursor on the first event prevents an initial garbage
	// delta when the window opens with the cursor in an arbitrary place.
	a.Subscribe(window.OnCursor, func(_ string, ev interface{}) {
		cev := ev.(*window.CursorEvent)
		if !worldStarted {
			lastCursorX = cev.Xpos
			lastCursorY = cev.Ypos
			cursorSeeded = true
			return
		}
		if !cursorSeeded {
			lastCursorX = cev.Xpos
			lastCursorY = cev.Ypos
			cursorSeeded = true
			return
		}
		dx := cev.Xpos - lastCursorX
		dy := cev.Ypos - lastCursorY
		lastCursorX = cev.Xpos
		lastCursorY = cev.Ypos
		// Yaw decreases when the mouse moves right (turn right), pitch
		// increases when the mouse moves up (look up). dy is positive
		// downward in screen space, hence the second negation.
		pendingYaw += -float64(dx) * mouseSensitivity
		pendingPitch += -float64(dy) * mouseSensitivity
	})

	a.Run(func(rend *renderer.Renderer, deltaTime time.Duration) {
		if !application.IsRunning(state) {
			a.Exit()
			return
		}
		if !worldStarted && menuApp.IsDone(menuState) {
			switch menuApp.Selected(menuState) {
			case menu.ChoiceNewGame:
				cam, wm, ps, err := startWorld(worldScene, viewImpl, worldImpl, playerImpl, a)
				if err != nil {
					log.Printf("craftmine: failed to start world: %v", err)
					state = application.Stop(state)
					a.Exit()
					return
				}
				worldCam = cam
				worldModel = wm
				playerState = ps
				gui.Manager().Set(worldScene)
				// Disable the OS cursor and lock its position to the
				// window so first-person mouse-look gets raw, unbounded
				// deltas instead of fighting the cursor against the
				// window edge. The OnCursor subscriber already handles
				// the post-disable position jump via cursorSeeded.
				gw.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
				worldStarted = true
				cursorSeeded = false
				layout()
			}
		}
		a.Gls().Clear(gls.DEPTH_BUFFER_BIT | gls.STENCIL_BUFFER_BIT | gls.COLOR_BUFFER_BIT)
		if worldStarted {
			in := buildInput(a.KeyState(), pendingYaw, pendingPitch)
			pendingYaw, pendingPitch = 0, 0
			playerState = playerImpl.Tick(playerState, in, worldModel, deltaTime.Seconds())
			updateCamera(worldCam, playerImpl, playerState)
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
// stand up the camera and player. Returns the world camera, the world
// Model (needed by the per-frame player Tick for bounds), and the
// spawned player state.
func startWorld(scene *core.Node, viewImpl worldview.View, worldImpl world.World, playerImpl player.Player, a *g3napp.Application) (*camera.Camera, world.Model, player.Model, error) {
	var blocksImpl blocks.Blocks = blocks.Impl{}
	registry, err := blocks.NewWithDefaults(blocksImpl)
	if err != nil {
		return nil, world.Model{}, player.Model{}, fmt.Errorf("blocks registry: %w", err)
	}

	opts := world.GenerateOptions{
		Width:     newGameWidth,
		Depth:     newGameDepth,
		MaxHeight: newGameMaxHeight,
		DirtDepth: world.DefaultDirtDepth,
	}
	model, err := worldImpl.Generate(newGameSeed, registry, blocksImpl, opts)
	if err != nil {
		return nil, world.Model{}, player.Model{}, fmt.Errorf("world generate: %w", err)
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

	cam := camera.New(1)
	cam.SetFar(float32(model.Width()) * 4)
	scene.Add(cam)

	width, height := a.GetSize()
	cam.SetAspect(float32(width) / float32(height))

	// Spawn the player one block above the surface at the world's center
	// column. SetPosition exists to validate against world bounds; if the
	// computed spawn is somehow rejected (off-by-one at the edges of the
	// configured world), fall back to the world center floor — being
	// visibly inside terrain is better than a startup crash.
	spawnX := float64(model.Width()) / 2
	spawnZ := float64(model.Depth()) / 2
	surface := model.HeightAt(int(spawnX), int(spawnZ))
	spawnY := float64(surface + 1)
	if spawnY > float64(model.MaxHeight()) {
		spawnY = float64(model.MaxHeight())
	}
	spawn := player.NewVec3(spawnX, spawnY, spawnZ)
	ps := player.New(spawn)
	ps, perr := playerImpl.SetPosition(ps, spawn, model)
	if perr != nil {
		log.Printf("craftmine: spawn position rejected (%v); falling back to (0, 0, 0)", perr)
		ps = player.New(player.NewVec3(0, 0, 0))
	}
	log.Printf("craftmine: player spawned at (%.1f, %.1f, %.1f), surface=%d", spawnX, spawnY, spawnZ, surface)

	// Initial camera placement so the first rendered frame is sensible
	// even before Tick runs.
	updateCamera(cam, playerImpl, ps)

	return cam, model, ps, nil
}

// buildInput maps the current keyboard state into a per-tick Input struct
// for the player module. The yaw/pitch deltas come from the mouse-look
// accumulator filled by the OnCursor subscriber.
func buildInput(keys *window.KeyState, yawDelta, pitchDelta float64) player.Input {
	if keys == nil {
		return player.Input{YawDelta: yawDelta, PitchDelta: pitchDelta}
	}
	return player.Input{
		Forward:    keys.Pressed(window.KeyW),
		Back:       keys.Pressed(window.KeyS),
		Left:       keys.Pressed(window.KeyA),
		Right:      keys.Pressed(window.KeyD),
		Jump:       keys.Pressed(window.KeySpace),
		YawDelta:   yawDelta,
		PitchDelta: pitchDelta,
	}
}

// updateCamera positions the g3n camera at the player's eye and aims it
// at the player's LookTarget. The player Model owns the camera math; the
// camera is a dumb consumer that gets repositioned each frame.
func updateCamera(cam *camera.Camera, playerImpl player.Player, ps player.Model) {
	eye := playerImpl.EyePosition(ps)
	target := playerImpl.LookTarget(ps)
	cam.SetPosition(float32(eye.X()), float32(eye.Y()), float32(eye.Z()))
	cam.LookAt(
		&math32.Vector3{X: float32(target.X()), Y: float32(target.Y()), Z: float32(target.Z())},
		&math32.Vector3{X: 0, Y: 1, Z: 0},
	)
}
