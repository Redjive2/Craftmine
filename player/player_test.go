package player_test

import (
	"math"
	"strings"
	"testing"

	"github.com/redjive2/Craftmine/player"
)

// epsilon is the float tolerance for direct math comparisons. Movement
// integration accumulates a little dt-quantization error, so longer
// integrations use loose() instead.
const epsilon = 1e-9

// fakeBounds is a tiny WorldBounds that lets tests dial in dimensions
// directly. The production world.Model satisfies the same interface.
type fakeBounds struct {
	width, depth, height int
}

func (b fakeBounds) Width() int     { return b.width }
func (b fakeBounds) Depth() int     { return b.depth }
func (b fakeBounds) MaxHeight() int { return b.height }

func defaultBounds() fakeBounds { return fakeBounds{width: 64, depth: 64, height: 64} }

func approx(t *testing.T, name string, got, want, tol float64) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Fatalf("%s = %v, want %v (±%v)", name, got, want, tol)
	}
}

// TestNewDefaults pins the Vision.md / Minecraft-ish dimensions.
func TestNewDefaults(t *testing.T) {
	m := player.New(player.NewVec3(8, 5, 8))
	if m.Position().X() != 8 || m.Position().Y() != 5 || m.Position().Z() != 8 {
		t.Fatalf("Position = %+v, want (8, 5, 8)", m.Position())
	}
	if m.EyeHeight() != player.DefaultEyeHeight {
		t.Fatalf("EyeHeight = %v, want %v", m.EyeHeight(), player.DefaultEyeHeight)
	}
	if m.HitboxWidth() != player.DefaultHitboxWidth {
		t.Fatalf("HitboxWidth = %v, want %v", m.HitboxWidth(), player.DefaultHitboxWidth)
	}
	if m.HitboxHeight() != player.DefaultHitboxHeight {
		t.Fatalf("HitboxHeight = %v, want %v", m.HitboxHeight(), player.DefaultHitboxHeight)
	}
	if m.Velocity() != (player.Vec3{}) {
		t.Fatalf("initial velocity = %+v, want zero", m.Velocity())
	}
	if m.OnGround() {
		t.Fatalf("OnGround = true on fresh spawn; expected false until first Tick lands")
	}
}

// TestVec3Math covers the few helpers used by Impl. Cheap to test, easy
// to regress if someone "simplifies" the helpers later.
func TestVec3Math(t *testing.T) {
	a := player.NewVec3(1, 2, 3)
	b := player.NewVec3(10, -5, 0.5)
	sum := a.Add(b)
	if sum.X() != 11 || sum.Y() != -3 || sum.Z() != 3.5 {
		t.Fatalf("Add = %+v, want (11, -3, 3.5)", sum)
	}
	scaled := a.Scale(2)
	if scaled.X() != 2 || scaled.Y() != 4 || scaled.Z() != 6 {
		t.Fatalf("Scale = %+v, want (2, 4, 6)", scaled)
	}
	// Input must not be mutated by either operation.
	if a.X() != 1 || a.Y() != 2 || a.Z() != 3 {
		t.Fatalf("Vec3.Add mutated receiver: a = %+v", a)
	}
}

// TestLookClampsPitch confirms NewLook's pitch clamp keeps the forward
// basis well defined no matter what a caller passes in.
func TestLookClampsPitch(t *testing.T) {
	cases := []struct {
		name      string
		yawIn     float64
		pitchIn   float64
		wantPitch float64
	}{
		{"above limit", 0, 10, math.Pi/2 - 0.001},
		{"below limit", 0, -10, -(math.Pi/2 - 0.001)},
		{"inside limit", 0, 0.3, 0.3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l := player.NewLook(tc.yawIn, tc.pitchIn)
			approx(t, "pitch", l.Pitch(), tc.wantPitch, epsilon)
		})
	}
}

// TestForwardDirection ties the camera convention down: yaw=0/pitch=0
// looks at -Z, and the basis vectors swap correctly under rotation. The
// expected values are derived from the documented convention in impl.go
// (-sin yaw cos pitch, sin pitch, -cos yaw cos pitch).
func TestForwardDirection(t *testing.T) {
	var impl player.Player = player.Impl{}
	cases := []struct {
		name  string
		yaw   float64
		pitch float64
		want  player.Vec3
	}{
		{"identity", 0, 0, player.NewVec3(0, 0, -1)},
		{"yaw 90 right", math.Pi / 2, 0, player.NewVec3(-1, 0, 0)},
		{"yaw 180", math.Pi, 0, player.NewVec3(0, 0, 1)},
		{"pitch up", 0, math.Pi / 4, player.NewVec3(0, math.Sin(math.Pi/4), -math.Cos(math.Pi/4))},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := player.SetLook(player.New(player.NewVec3(0, 0, 0)), player.NewLook(tc.yaw, tc.pitch))
			got := impl.ForwardDirection(m)
			approx(t, "x", got.X(), tc.want.X(), 1e-9)
			approx(t, "y", got.Y(), tc.want.Y(), 1e-9)
			approx(t, "z", got.Z(), tc.want.Z(), 1e-9)
		})
	}
}

// TestHorizontalForwardIgnoresPitch — looking up should not change the
// horizontal movement direction, otherwise looking at the sky would lift
// the player off the ground.
func TestHorizontalForwardIgnoresPitch(t *testing.T) {
	var impl player.Player = player.Impl{}
	m := player.SetLook(player.New(player.NewVec3(0, 0, 0)), player.NewLook(0, math.Pi/4))
	h := impl.HorizontalForward(m)
	if h.Y() != 0 {
		t.Fatalf("HorizontalForward.Y = %v, want 0 (pitch must not bleed in)", h.Y())
	}
	approx(t, "x", h.X(), 0, epsilon)
	approx(t, "z", h.Z(), -1, epsilon)
}

// TestEyePositionOffsetsByEyeHeight — the camera should sit at foot+eye.
func TestEyePositionOffsetsByEyeHeight(t *testing.T) {
	var impl player.Player = player.Impl{}
	m := player.New(player.NewVec3(3, 10, 4))
	eye := impl.EyePosition(m)
	approx(t, "x", eye.X(), 3, epsilon)
	approx(t, "y", eye.Y(), 10+player.DefaultEyeHeight, epsilon)
	approx(t, "z", eye.Z(), 4, epsilon)
}

// TestLookTargetIsEyePlusForward — main.go feeds this into camera.LookAt.
func TestLookTargetIsEyePlusForward(t *testing.T) {
	var impl player.Player = player.Impl{}
	m := player.New(player.NewVec3(0, 0, 0))
	target := impl.LookTarget(m)
	eye := impl.EyePosition(m)
	fwd := impl.ForwardDirection(m)
	approx(t, "x", target.X(), eye.X()+fwd.X(), epsilon)
	approx(t, "y", target.Y(), eye.Y()+fwd.Y(), epsilon)
	approx(t, "z", target.Z(), eye.Z()+fwd.Z(), epsilon)
}

// TestApplyLookAccumulatesAndClamps walks several mouse deltas through
// the named transition and checks both accumulation and pitch clamping.
func TestApplyLookAccumulatesAndClamps(t *testing.T) {
	var impl player.Player = player.Impl{}
	m := player.New(player.NewVec3(0, 0, 0))
	m = impl.ApplyLook(m, 0.5, 0.2)
	approx(t, "yaw", m.Look().Yaw(), 0.5, epsilon)
	approx(t, "pitch", m.Look().Pitch(), 0.2, epsilon)
	m = impl.ApplyLook(m, 0.5, 5.0) // should saturate pitch
	approx(t, "yaw 2", m.Look().Yaw(), 1.0, epsilon)
	approx(t, "pitch saturated", m.Look().Pitch(), math.Pi/2-0.001, epsilon)
}

// TestSetPositionValidation covers every named error path: NaN, Inf, and
// each out-of-bounds axis. SetPosition is the spawn / teleport entry
// point, so its validation is load-bearing.
func TestSetPositionValidation(t *testing.T) {
	var impl player.Player = player.Impl{}
	bounds := defaultBounds()
	m := player.New(player.NewVec3(0, 0, 0))

	bad := []struct {
		name    string
		p       player.Vec3
		wantSub string
	}{
		{"NaN x", player.NewVec3(math.NaN(), 5, 5), "NaN"},
		{"NaN y", player.NewVec3(5, math.NaN(), 5), "NaN"},
		{"NaN z", player.NewVec3(5, 5, math.NaN()), "NaN"},
		{"+Inf x", player.NewVec3(math.Inf(1), 5, 5), "infinite"},
		{"-Inf y", player.NewVec3(5, math.Inf(-1), 5), "infinite"},
		{"x negative", player.NewVec3(-1, 5, 5), "x="},
		{"x at width", player.NewVec3(float64(bounds.Width()), 5, 5), "x="},
		{"z negative", player.NewVec3(5, 5, -0.1), "z="},
		{"z at depth", player.NewVec3(5, 5, float64(bounds.Depth())), "z="},
		{"y above max", player.NewVec3(5, float64(bounds.MaxHeight()+1), 5), "y="},
		{"y negative", player.NewVec3(5, -1, 5), "y="},
	}
	for _, tc := range bad {
		t.Run(tc.name, func(t *testing.T) {
			_, err := impl.SetPosition(m, tc.p, bounds)
			if err == nil {
				t.Fatalf("SetPosition(%+v) returned nil error", tc.p)
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("err = %q, want substring %q", err.Error(), tc.wantSub)
			}
		})
	}

	good := player.NewVec3(10, 5, 10)
	out, err := impl.SetPosition(m, good, bounds)
	if err != nil {
		t.Fatalf("SetPosition(valid) = %v, want nil", err)
	}
	if out.Position() != good {
		t.Fatalf("Position = %+v, want %+v", out.Position(), good)
	}
	// Verify input was not mutated.
	if m.Position() == good {
		t.Fatalf("SetPosition mutated its input Model")
	}
}

// TestSetPositionRequiresBounds is a separate case because the error path
// runs before validation and shouldn't be conflated with bound failures.
func TestSetPositionRequiresBounds(t *testing.T) {
	var impl player.Player = player.Impl{}
	m := player.New(player.NewVec3(0, 0, 0))
	if _, err := impl.SetPosition(m, player.NewVec3(1, 1, 1), nil); err == nil {
		t.Fatalf("SetPosition with nil bounds returned no error")
	}
}

// TestTickIgnoresBadDt covers the two guards at the top of Tick: a zero
// or negative dt is a no-op, and an absurdly large dt is capped.
func TestTickIgnoresBadDt(t *testing.T) {
	var impl player.Player = player.Impl{}
	bounds := defaultBounds()
	m := player.New(player.NewVec3(10, 20, 10))

	for _, dt := range []float64{0, -1, math.NaN()} {
		out := impl.Tick(m, player.Input{}, bounds, dt)
		if out.Position() != m.Position() {
			t.Fatalf("Tick(dt=%v) moved player: %+v -> %+v", dt, m.Position(), out.Position())
		}
		if out.Velocity() != m.Velocity() {
			t.Fatalf("Tick(dt=%v) changed velocity: %+v -> %+v", dt, m.Velocity(), out.Velocity())
		}
	}
}

// TestGravityPullsAirborneToGround integrates a few seconds of physics
// with no input and asserts the player ends up rested at y=0 with the
// onGround flag set. This is the headline "drop test".
func TestGravityPullsAirborneToGround(t *testing.T) {
	var impl player.Player = player.Impl{}
	bounds := defaultBounds()
	m := player.New(player.NewVec3(10, 30, 10))

	dt := 1.0 / 60
	for i := 0; i < 600; i++ { // 10 seconds — plenty to fall from y=30
		m = impl.Tick(m, player.Input{}, bounds, dt)
	}
	if !m.OnGround() {
		t.Fatalf("after free fall, OnGround = false; pos=%+v vel=%+v", m.Position(), m.Velocity())
	}
	approx(t, "rested y", m.Position().Y(), 0, 1e-6)
	approx(t, "rested vy", m.Velocity().Y(), 0, 1e-6)
}

// TestJumpLeavesGroundAndLandsBack confirms the jump impulse adds upward
// velocity, the player clears the ground, and gravity returns them. The
// apex must be strictly above the starting Y.
func TestJumpLeavesGroundAndLandsBack(t *testing.T) {
	var impl player.Player = player.Impl{}
	bounds := defaultBounds()
	// Start grounded by stepping once at y=0.
	m := player.New(player.NewVec3(10, 0, 10))
	m = impl.Tick(m, player.Input{}, bounds, 1.0/60)
	if !m.OnGround() {
		t.Fatalf("setup: expected grounded after one tick at y=0, got %+v", m)
	}

	m = impl.Tick(m, player.Input{Jump: true}, bounds, 1.0/60)
	if m.OnGround() {
		t.Fatalf("after jump tick, OnGround = true; should be airborne")
	}
	if m.Velocity().Y() <= 0 {
		t.Fatalf("after jump, vy = %v, want > 0", m.Velocity().Y())
	}

	apex := m.Position().Y()
	for i := 0; i < 240; i++ { // 4 seconds — easily long enough to land
		m = impl.Tick(m, player.Input{}, bounds, 1.0/60)
		if m.Position().Y() > apex {
			apex = m.Position().Y()
		}
		if m.OnGround() {
			break
		}
	}
	if !m.OnGround() {
		t.Fatalf("after jump+wait, never landed: pos=%+v vel=%+v", m.Position(), m.Velocity())
	}
	if apex <= 0.05 {
		t.Fatalf("jump apex = %v, want clearly above 0", apex)
	}
}

// TestJumpIgnoredAirborne — once airborne, a jump press should not
// re-launch (no infinite hover). Critical for movement feel.
func TestJumpIgnoredAirborne(t *testing.T) {
	var impl player.Player = player.Impl{}
	bounds := defaultBounds()
	m := player.New(player.NewVec3(10, 10, 10))
	m = impl.Tick(m, player.Input{}, bounds, 1.0/60) // start falling

	before := m.Velocity().Y()
	m = impl.Tick(m, player.Input{Jump: true}, bounds, 1.0/60)
	// Velocity should have decreased (gravity), not jumped to JumpVelocity.
	if m.Velocity().Y() > before {
		t.Fatalf("airborne jump raised vy: before=%v after=%v", before, m.Velocity().Y())
	}
	if m.Velocity().Y() > 1 {
		t.Fatalf("airborne jump triggered: vy=%v, want falling (negative)", m.Velocity().Y())
	}
}

// TestInputForwardMovesAlongMinusZ — at yaw=0, pressing Forward must
// move the player along -Z at MoveSpeed.
func TestInputForwardMovesAlongMinusZ(t *testing.T) {
	var impl player.Player = player.Impl{}
	bounds := defaultBounds()
	m := player.New(player.NewVec3(10, 0, 10))
	// One settling tick to ground the player.
	m = impl.Tick(m, player.Input{}, bounds, 1.0/60)

	// One tick of forward input.
	m = impl.Tick(m, player.Input{Forward: true}, bounds, 1.0/60)
	approx(t, "vx", m.Velocity().X(), 0, 1e-9)
	if m.Velocity().Z() >= 0 {
		t.Fatalf("forward velocity z = %v, want < 0 (moving along -Z)", m.Velocity().Z())
	}
	approx(t, "|vz|", math.Abs(m.Velocity().Z()), player.MoveSpeed, 1e-9)
}

// TestDiagonalMovementNormalized — W+D should not be faster than W
// alone. Otherwise diagonal strafe gives free sprint speed.
func TestDiagonalMovementNormalized(t *testing.T) {
	var impl player.Player = player.Impl{}
	bounds := defaultBounds()
	m := player.New(player.NewVec3(10, 0, 10))
	m = impl.Tick(m, player.Input{}, bounds, 1.0/60)

	m = impl.Tick(m, player.Input{Forward: true, Right: true}, bounds, 1.0/60)
	speed := math.Hypot(m.Velocity().X(), m.Velocity().Z())
	approx(t, "horizontal speed", speed, player.MoveSpeed, 1e-9)
}

// TestNoInputZerosHorizontalVelocity — releasing keys must brake instantly
// (no inertia carry). Movement feel tuning.
func TestNoInputZerosHorizontalVelocity(t *testing.T) {
	var impl player.Player = player.Impl{}
	bounds := defaultBounds()
	m := player.New(player.NewVec3(10, 0, 10))
	m = impl.Tick(m, player.Input{}, bounds, 1.0/60)
	m = impl.Tick(m, player.Input{Forward: true}, bounds, 1.0/60)
	if m.Velocity().Z() >= 0 {
		t.Fatalf("setup: expected forward velocity, got %v", m.Velocity().Z())
	}
	m = impl.Tick(m, player.Input{}, bounds, 1.0/60)
	approx(t, "vx after release", m.Velocity().X(), 0, 1e-9)
	approx(t, "vz after release", m.Velocity().Z(), 0, 1e-9)
}

// TestBoundsClampPreventsLeavingWorld — walking into a wall should stop
// at the wall, not warp through it, and shouldn't accumulate negative
// velocity that would later make you teleport on release.
func TestBoundsClampPreventsLeavingWorld(t *testing.T) {
	var impl player.Player = player.Impl{}
	bounds := fakeBounds{width: 8, depth: 8, height: 32}
	// Spawn near the +X wall facing yaw=-pi/2 (which looks down +X) and
	// hold Forward; the player should hit the wall and stop.
	m := player.New(player.NewVec3(7, 0, 4))
	m = player.SetLook(m, player.NewLook(-math.Pi/2, 0))
	m = impl.Tick(m, player.Input{}, bounds, 1.0/60)
	for i := 0; i < 120; i++ {
		m = impl.Tick(m, player.Input{Forward: true}, bounds, 1.0/60)
	}
	if m.Position().X() < 0 || m.Position().X() >= float64(bounds.Width()) {
		t.Fatalf("player escaped world: x=%v, bounds.Width=%v", m.Position().X(), bounds.Width())
	}
	if m.Position().X() < float64(bounds.Width())-1 {
		t.Fatalf("player did not reach +X wall: x=%v", m.Position().X())
	}
}

// TestYawMouseLookFedThroughTick — mouse deltas applied via Input.YawDelta
// must change look as expected. The look update is applied BEFORE the
// movement step, so a single tick with both yaw delta and Forward should
// move along the new direction.
func TestYawMouseLookFedThroughTick(t *testing.T) {
	var impl player.Player = player.Impl{}
	bounds := defaultBounds()
	m := player.New(player.NewVec3(10, 0, 10))
	m = impl.Tick(m, player.Input{}, bounds, 1.0/60)

	// Rotate yaw to pi/2 (forward becomes -X) and step forward.
	m = impl.Tick(m, player.Input{Forward: true, YawDelta: math.Pi / 2}, bounds, 1.0/60)
	approx(t, "yaw", m.Look().Yaw(), math.Pi/2, 1e-9)
	if m.Velocity().X() >= 0 {
		t.Fatalf("after yaw 90, vx = %v, want < 0", m.Velocity().X())
	}
	approx(t, "vz", m.Velocity().Z(), 0, 1e-9)
}

// TestIntegrationWalkAFewFrames is the headline ticket-acceptance check:
// spawn the player, walk forward 30 frames at 1/60s, and assert the
// final position matches the analytical expectation (within float wiggle
// from ground settle).
func TestIntegrationWalkAFewFrames(t *testing.T) {
	var impl player.Player = player.Impl{}
	bounds := defaultBounds()
	m := player.New(player.NewVec3(20, 0, 20))

	dt := 1.0 / 60
	// One settling tick brings onGround = true and zeros vy.
	m = impl.Tick(m, player.Input{}, bounds, dt)

	const frames = 30
	for i := 0; i < frames; i++ {
		m = impl.Tick(m, player.Input{Forward: true}, bounds, dt)
	}
	wantZ := 20 - player.MoveSpeed*float64(frames)*dt
	approx(t, "final x", m.Position().X(), 20, 1e-6)
	approx(t, "final z", m.Position().Z(), wantZ, 1e-6)
	approx(t, "final y", m.Position().Y(), 0, 1e-6)
	if !m.OnGround() {
		t.Fatalf("after walk, OnGround = false")
	}
}
