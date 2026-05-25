// Package player owns the first-person player character.
//
// Per Vision.md this module follows the Craftmine module pattern: a Model
// holding all player state (position, look, velocity, ground flag, and the
// immutable eye-height / hitbox dimensions), an Impl carrying behavior with
// no fields, and a Player interface that Impl satisfies. Every behavioral
// function takes Model as an argument and returns a new Model when state
// changes; nothing is hung off the receiver.
//
// The player is a primary module: it consumes a sub-interface of the world
// module for bounds clamping (see WorldBounds in impl.go) but otherwise is
// independent. Rendering / camera wiring lives at the top level (main.go).
package player

import "math"

// Vec3 is a plain 3D point or vector. Fields are private to keep the type
// trivially serializable through the Model accessors and to forbid in-place
// mutation; build a new Vec3 instead.
//
// The package convention is Y-up, with yaw=0 looking down the -Z axis and
// yaw rotating around Y. See ForwardDirection in impl.go for the full
// camera-space convention.
type Vec3 struct {
	x float64
	y float64
	z float64
}

// NewVec3 returns a Vec3 with the given coordinates.
func NewVec3(x, y, z float64) Vec3 { return Vec3{x: x, y: y, z: z} }

// X, Y, Z return the components of v.
func (v Vec3) X() float64 { return v.x }
func (v Vec3) Y() float64 { return v.y }
func (v Vec3) Z() float64 { return v.z }

// Add returns v + other. v is not modified.
func (v Vec3) Add(other Vec3) Vec3 { return Vec3{v.x + other.x, v.y + other.y, v.z + other.z} }

// Scale returns v multiplied by s. v is not modified.
func (v Vec3) Scale(s float64) Vec3 { return Vec3{v.x * s, v.y * s, v.z * s} }

// Look is the camera orientation: yaw (rotation around Y) and pitch (around
// the local X axis), both in radians. Fields are private; build via NewLook.
type Look struct {
	yaw   float64
	pitch float64
}

// NewLook returns a Look with the given yaw and pitch (radians). The pitch
// is clamped to [-pitchLimit, pitchLimit] to keep the camera basis well
// defined at all times.
func NewLook(yaw, pitch float64) Look {
	return Look{yaw: wrapAngle(yaw), pitch: clampPitch(pitch)}
}

// Yaw is the rotation around the world Y axis in radians.
func (l Look) Yaw() float64 { return l.yaw }

// Pitch is the rotation around the camera-local X axis in radians.
// Negative pitch looks down, positive pitch looks up. Always within
// [-pitchLimit, pitchLimit].
func (l Look) Pitch() float64 { return l.pitch }

// Default physical dimensions for the player. They mirror Minecraft's
// nominal values closely enough that the camera height feels right.
const (
	DefaultEyeHeight    = 1.62
	DefaultHitboxWidth  = 0.6
	DefaultHitboxHeight = 1.8

	// pitchLimit is the maximum absolute pitch in radians. Slightly less
	// than pi/2 so the forward vector never collapses onto the Y axis.
	pitchLimit = math.Pi/2 - 0.001
)

// Model is the player's mutable + immutable state.
//
// All fields are private. Read them through the accessor methods below
// (Position, not GetPosition). Mutating helpers (SetPosition, SetLook,
// SetVelocity, SetOnGround) live below as package-level functions and
// return a fresh Model rather than receiving on *Model — this matches
// the Vision.md "Model is data, Impl is behavior" split.
type Model struct {
	position Vec3
	look     Look
	velocity Vec3
	onGround bool

	eyeHeight    float64
	hitboxWidth  float64
	hitboxHeight float64
}

// New returns a Model spawned at the given world position with default
// hitbox / eye-height dimensions, no velocity, and facing yaw=0 / pitch=0
// (looking down -Z).
//
// The spawn position is taken on trust here; callers that want validation
// against world bounds should route through Impl.SetPosition after New.
func New(spawn Vec3) Model {
	return Model{
		position:     spawn,
		look:         Look{},
		velocity:     Vec3{},
		onGround:     false,
		eyeHeight:    DefaultEyeHeight,
		hitboxWidth:  DefaultHitboxWidth,
		hitboxHeight: DefaultHitboxHeight,
	}
}

// Position returns the player's world-space foot position.
func (m Model) Position() Vec3 { return m.position }

// Look returns the player's camera orientation.
func (m Model) Look() Look { return m.look }

// Velocity returns the player's per-second velocity vector in world space.
func (m Model) Velocity() Vec3 { return m.velocity }

// OnGround reports whether the player's feet are resting on a surface.
// Jump input is only honored when OnGround is true.
func (m Model) OnGround() bool { return m.onGround }

// EyeHeight is the offset from the player's foot position to the camera
// eye position, in blocks. Immutable for the life of the Model.
func (m Model) EyeHeight() float64 { return m.eyeHeight }

// HitboxWidth is the side length of the (axis-aligned) player hitbox in
// the X/Z plane. Immutable for the life of the Model.
func (m Model) HitboxWidth() float64 { return m.hitboxWidth }

// HitboxHeight is the vertical extent of the player hitbox in blocks.
// Immutable for the life of the Model.
func (m Model) HitboxHeight() float64 { return m.hitboxHeight }

// SetPosition returns a new Model with position set to p. The input Model
// is not mutated. Validation lives on Impl.SetPosition; this raw setter is
// the building block that named transitions compose on top of.
func SetPosition(m Model, p Vec3) Model {
	m.position = p
	return m
}

// SetLook returns a new Model with look set to l. l is normalized through
// NewLook so an out-of-range pitch supplied by a caller is corrected
// before storage.
func SetLook(m Model, l Look) Model {
	m.look = NewLook(l.yaw, l.pitch)
	return m
}

// SetVelocity returns a new Model with velocity set to v.
func SetVelocity(m Model, v Vec3) Model {
	m.velocity = v
	return m
}

// SetOnGround returns a new Model with the on-ground flag set to grounded.
func SetOnGround(m Model, grounded bool) Model {
	m.onGround = grounded
	return m
}

// wrapAngle reduces a yaw value modulo 2*pi into (-pi, pi]. Keeps
// accumulated yaw bounded across long sessions without changing the
// observable camera direction.
func wrapAngle(a float64) float64 {
	twoPi := 2 * math.Pi
	a = math.Mod(a, twoPi)
	if a > math.Pi {
		a -= twoPi
	} else if a <= -math.Pi {
		a += twoPi
	}
	return a
}

// clampPitch confines pitch to [-pitchLimit, pitchLimit] so the forward
// basis vector never degenerates at the zenith / nadir.
func clampPitch(p float64) float64 {
	if p > pitchLimit {
		return pitchLimit
	}
	if p < -pitchLimit {
		return -pitchLimit
	}
	return p
}
