package player

import (
	"fmt"
	"math"
)

// Player is the behavior interface for the player module.
//
// Callers depend on Player rather than the concrete Impl so the module is
// swappable (e.g. a recorded test double can satisfy Player without
// running real physics).
type Player interface {
	Tick(m Model, in Input, bounds WorldBounds, dt float64) Model
	ApplyLook(m Model, deltaYaw, deltaPitch float64) Model
	SetPosition(m Model, p Vec3, bounds WorldBounds) (Model, error)
	EyePosition(m Model) Vec3
	ForwardDirection(m Model) Vec3
	HorizontalForward(m Model) Vec3
	LookTarget(m Model) Vec3
}

// Impl is the zero-field implementation of Player. All behavior hangs off
// Impl; state lives in Model and is passed as an argument to every method.
type Impl struct{}

// Compile-time check that Impl satisfies Player.
var _ Player = Impl{}

// WorldBounds is the sub-interface of the world Model that the player needs
// to keep itself inside the playable area. world.Model satisfies this
// already (see world/model.go), so wiring at main.go is a direct pass.
//
// Defining the interface here — rather than importing world directly — keeps
// the player module independent of any other primary module, per Vision.md.
type WorldBounds interface {
	Width() int
	Depth() int
	MaxHeight() int
}

// Input is the per-tick command surface from main.go. Booleans are "is
// this key currently held"; YawDelta / PitchDelta are the mouse-look
// deltas to apply this tick (radians).
//
// Fields are public because Input is transient transport, not Model state.
type Input struct {
	Forward    bool
	Back       bool
	Left       bool
	Right      bool
	Jump       bool
	YawDelta   float64
	PitchDelta float64
}

// Physics tuning. All values are per-second so the same numbers work at
// any frame rate; Tick scales them by dt.
const (
	// Gravity is the downward acceleration in blocks/sec². Higher than
	// real-world ~9.8 to make jumps land quickly — feels more arcade,
	// matches Minecraft's snappier physics.
	Gravity = 32.0

	// JumpVelocity is the upward velocity imparted at the start of a
	// jump. Sized so the apex is ~1.25 blocks above the launch point at
	// the chosen Gravity, which matches a one-block hop.
	JumpVelocity = 9.0

	// MoveSpeed is the horizontal walking speed in blocks/sec. Applied
	// directly to horizontal velocity each tick (no acceleration ramp).
	MoveSpeed = 4.5

	// MaxDelta is the largest dt Tick will honor. A long pause (debugger
	// break, GC stall, window dragged) shouldn't produce a single
	// catastrophic integration step.
	MaxDelta = 0.1

	// minFloorY is the lowest Y the player can occupy. y=0 is the
	// terrain floor for the first-pass world; gravity stops here until
	// real block collision lands in a follow-up ticket.
	minFloorY = 0.0
)

// Tick advances the player physics by dt seconds.
//
// Input is folded in at block boundaries: look deltas first (so movement
// uses the post-look-update yaw), then desired horizontal velocity, then
// jump, then gravity, then integration, then ground / bounds resolution.
// Internal scratch values live as locals; the Model is rewritten as a
// final assignment at the bottom — keeping mutation at the block edges.
//
// dt outside (0, MaxDelta] is clamped and logged-style absorbed: a zero
// or negative dt returns m unchanged, a too-large dt is capped at MaxDelta.
func (Impl) Tick(m Model, in Input, bounds WorldBounds, dt float64) Model {
	if !(dt > 0) {
		return m
	}
	if dt > MaxDelta {
		dt = MaxDelta
	}
	if bounds == nil {
		return m
	}

	// 1. Fold mouse-look first so movement uses the updated yaw.
	look := NewLook(m.look.yaw+in.YawDelta, m.look.pitch+in.PitchDelta)

	// 2. Desired horizontal velocity from WASD-style input.
	desiredH := desiredHorizontalVelocity(in, look.yaw)
	velocity := Vec3{x: desiredH.x, y: m.velocity.y, z: desiredH.z}

	// 3. Jump only when grounded so airborne jump-spam doesn't lift you.
	onGround := m.onGround
	if in.Jump && onGround {
		velocity.y = JumpVelocity
		onGround = false
	}

	// 4. Gravity is always applied; ground resolution below cancels it
	//    once we've landed.
	velocity.y -= Gravity * dt

	// 5. Integrate.
	position := Vec3{
		x: m.position.x + velocity.x*dt,
		y: m.position.y + velocity.y*dt,
		z: m.position.z + velocity.z*dt,
	}

	// 6. Resolve ground and world bounds.
	position, velocity, onGround = resolveBounds(position, velocity, bounds)

	out := m
	out.position = position
	out.velocity = velocity
	out.look = look
	out.onGround = onGround
	return out
}

// ApplyLook adds the supplied deltas to the player's look. Pitch is
// clamped inside NewLook so an extreme drag cannot flip the camera. The
// returned Model is a fresh copy; m is untouched.
func (Impl) ApplyLook(m Model, deltaYaw, deltaPitch float64) Model {
	m.look = NewLook(m.look.yaw+deltaYaw, m.look.pitch+deltaPitch)
	return m
}

// SetPosition validates p against the world bounds and stores it on a
// fresh copy of m. Rejected inputs (NaN, Inf, outside bounds) return the
// input Model and a descriptive error so callers can log and fall back.
//
// Validation is the single source of truth for "is this position legal";
// Tick uses resolveBounds which clamps rather than rejects, so explicit
// teleports / spawn placement should go through SetPosition.
func (Impl) SetPosition(m Model, p Vec3, bounds WorldBounds) (Model, error) {
	if bounds == nil {
		return m, fmt.Errorf("player: SetPosition requires non-nil WorldBounds")
	}
	if err := validatePosition(p, bounds); err != nil {
		return m, err
	}
	m.position = p
	return m, nil
}

// EyePosition is the world-space camera eye: the foot position plus the
// player's eye height along +Y.
func (Impl) EyePosition(m Model) Vec3 {
	return Vec3{x: m.position.x, y: m.position.y + m.eyeHeight, z: m.position.z}
}

// ForwardDirection is the unit vector the camera is currently aimed at.
// Convention: yaw=0, pitch=0 -> (0, 0, -1). Positive pitch looks up.
func (Impl) ForwardDirection(m Model) Vec3 {
	return forwardFromLook(m.look)
}

// HorizontalForward is the yaw-only forward (pitch ignored, Y component
// zero). Used for movement input so looking up doesn't lift the player.
func (Impl) HorizontalForward(m Model) Vec3 {
	yaw := m.look.yaw
	return Vec3{x: -math.Sin(yaw), y: 0, z: -math.Cos(yaw)}
}

// LookTarget is a point one block ahead of the eye in the look direction.
// Convenient for camera.LookAt(target, up).
func (Impl) LookTarget(m Model) Vec3 {
	eye := Impl{}.EyePosition(m)
	fwd := forwardFromLook(m.look)
	return eye.Add(fwd)
}

// desiredHorizontalVelocity translates WASD-style booleans into a
// horizontal velocity vector at MoveSpeed magnitude in the yaw-relative
// frame.
//
// Diagonal motion is normalized so pressing W+D is no faster than W
// alone — otherwise the player would sprint diagonally for free.
func desiredHorizontalVelocity(in Input, yaw float64) Vec3 {
	var f, s float64
	if in.Forward {
		f += 1
	}
	if in.Back {
		f -= 1
	}
	if in.Right {
		s += 1
	}
	if in.Left {
		s -= 1
	}
	if f == 0 && s == 0 {
		return Vec3{}
	}
	mag := math.Hypot(f, s)
	f /= mag
	s /= mag

	// Yaw-relative basis. yaw=0 looks along -Z, so forward = (-sin, 0, -cos)
	// and right = (cos, 0, -sin). Composing the two gives the desired
	// world-space horizontal velocity.
	sinY := math.Sin(yaw)
	cosY := math.Cos(yaw)
	vx := MoveSpeed * (-sinY*f + cosY*s)
	vz := MoveSpeed * (-cosY*f - sinY*s)
	return Vec3{x: vx, z: vz}
}

// resolveBounds applies the simple ground collision (y >= 0) and clamps
// horizontal position to the playable area. Out-of-bound collisions zero
// the relevant velocity component so the player doesn't accumulate
// energy by being pushed back each frame.
func resolveBounds(pos, vel Vec3, bounds WorldBounds) (Vec3, Vec3, bool) {
	onGround := false
	maxX := float64(bounds.Width()) - 0.001
	maxZ := float64(bounds.Depth()) - 0.001
	maxY := float64(bounds.MaxHeight()) - 0.001
	if pos.x < 0 {
		pos.x = 0
		if vel.x < 0 {
			vel.x = 0
		}
	}
	if pos.x > maxX {
		pos.x = maxX
		if vel.x > 0 {
			vel.x = 0
		}
	}
	if pos.z < 0 {
		pos.z = 0
		if vel.z < 0 {
			vel.z = 0
		}
	}
	if pos.z > maxZ {
		pos.z = maxZ
		if vel.z > 0 {
			vel.z = 0
		}
	}
	if pos.y > maxY {
		pos.y = maxY
		if vel.y > 0 {
			vel.y = 0
		}
	}
	if pos.y <= minFloorY {
		pos.y = minFloorY
		if vel.y < 0 {
			vel.y = 0
		}
		onGround = true
	}
	return pos, vel, onGround
}

// validatePosition rejects positions that would corrupt the Model: NaN,
// Inf, or coords outside [0, width) / [0, maxHeight] / [0, depth). The
// error message names the failure so logs are useful.
func validatePosition(p Vec3, bounds WorldBounds) error {
	if math.IsNaN(p.x) || math.IsNaN(p.y) || math.IsNaN(p.z) {
		return fmt.Errorf("player: position has NaN component (%v, %v, %v)", p.x, p.y, p.z)
	}
	if math.IsInf(p.x, 0) || math.IsInf(p.y, 0) || math.IsInf(p.z, 0) {
		return fmt.Errorf("player: position has infinite component (%v, %v, %v)", p.x, p.y, p.z)
	}
	w := float64(bounds.Width())
	d := float64(bounds.Depth())
	h := float64(bounds.MaxHeight())
	if p.x < 0 || p.x >= w {
		return fmt.Errorf("player: x=%v outside [0, %v)", p.x, w)
	}
	if p.z < 0 || p.z >= d {
		return fmt.Errorf("player: z=%v outside [0, %v)", p.z, d)
	}
	if p.y < 0 || p.y > h {
		return fmt.Errorf("player: y=%v outside [0, %v]", p.y, h)
	}
	return nil
}

// forwardFromLook is the canonical yaw+pitch -> unit-forward conversion.
// Kept package-private so the camera-space convention has a single home.
func forwardFromLook(l Look) Vec3 {
	cosP := math.Cos(l.pitch)
	return Vec3{
		x: -math.Sin(l.yaw) * cosP,
		y: math.Sin(l.pitch),
		z: -math.Cos(l.yaw) * cosP,
	}
}
