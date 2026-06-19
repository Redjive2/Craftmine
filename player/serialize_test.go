package player_test

import (
	"bytes"
	"encoding/gob"
	"math"
	"strings"
	"testing"

	"github.com/redjive2/Craftmine/player"
)

// wireSnapshot mirrors player's unexported playerSnapshot by exported field
// name so gob decodes a value of this type straight into the internal
// snapshot. Replicating every field keeps a baseline blob fully valid; tests
// then corrupt a single field to drive one error branch at a time.
type wireSnapshot struct {
	PositionX, PositionY, PositionZ float64
	Yaw                             float64
	Pitch                           float64
	VelocityX, VelocityY, VelocityZ float64
	OnGround                        bool
	EyeHeight                       float64
	HitboxWidth                     float64
	HitboxHeight                    float64
}

// validSnap returns a wireSnapshot that round-trips cleanly: all components
// finite and the eye/hitbox dimensions strictly positive.
func validSnap() wireSnapshot {
	return wireSnapshot{
		PositionX: 10.5, PositionY: 12, PositionZ: 8.25,
		Yaw: 1.2, Pitch: -0.3,
		VelocityX: 0.5, VelocityY: -1.0, VelocityZ: 0.25,
		OnGround:  true,
		EyeHeight: 1.62, HitboxWidth: 0.6, HitboxHeight: 1.8,
	}
}

func encodeWire(t *testing.T, v any) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(v); err != nil {
		t.Fatalf("gob encode: %v", err)
	}
	return buf.Bytes()
}

// floatComponents lists every float field validateFloats inspects, paired
// with a setter so a single table can poison each one with NaN and Inf.
var floatComponents = []struct {
	name string
	set  func(*wireSnapshot, float64)
}{
	{"position.x", func(s *wireSnapshot, v float64) { s.PositionX = v }},
	{"position.y", func(s *wireSnapshot, v float64) { s.PositionY = v }},
	{"position.z", func(s *wireSnapshot, v float64) { s.PositionZ = v }},
	{"yaw", func(s *wireSnapshot, v float64) { s.Yaw = v }},
	{"pitch", func(s *wireSnapshot, v float64) { s.Pitch = v }},
	{"velocity.x", func(s *wireSnapshot, v float64) { s.VelocityX = v }},
	{"velocity.y", func(s *wireSnapshot, v float64) { s.VelocityY = v }},
	{"velocity.z", func(s *wireSnapshot, v float64) { s.VelocityZ = v }},
	{"eyeHeight", func(s *wireSnapshot, v float64) { s.EyeHeight = v }},
	{"hitboxWidth", func(s *wireSnapshot, v float64) { s.HitboxWidth = v }},
	{"hitboxHeight", func(s *wireSnapshot, v float64) { s.HitboxHeight = v }},
}

// TestDeserializeRejectsNonFinite batch-checks that every float component
// rejects NaN and both infinities. validateFloats runs before the
// dimension-positivity check, so even the eye/hitbox fields report the
// NaN/infinite message rather than "non-positive" here.
func TestDeserializeRejectsNonFinite(t *testing.T) {
	bad := []struct {
		label   string
		value   float64
		wantSub string
	}{
		{"NaN", math.NaN(), "is NaN"},
		{"+Inf", math.Inf(1), "is infinite"},
		{"-Inf", math.Inf(-1), "is infinite"},
	}
	for _, comp := range floatComponents {
		for _, b := range bad {
			t.Run(comp.name+"/"+b.label, func(t *testing.T) {
				snap := validSnap()
				comp.set(&snap, b.value)
				_, err := player.Deserialize(encodeWire(t, snap))
				if err == nil {
					t.Fatalf("Deserialize with %s=%s: nil error, want %q", comp.name, b.label, b.wantSub)
				}
				if !strings.Contains(err.Error(), b.wantSub) {
					t.Fatalf("Deserialize with %s=%s: error = %q, want substring %q",
						comp.name, b.label, err.Error(), b.wantSub)
				}
				if !strings.Contains(err.Error(), comp.name) {
					t.Fatalf("Deserialize with %s=%s: error = %q, want it to name %q",
						comp.name, b.label, err.Error(), comp.name)
				}
			})
		}
	}
}

// TestDeserializeRejectsBadShape covers the non-float guards: empty input,
// an undecodable blob, and each non-positive dimension.
func TestDeserializeRejectsBadShape(t *testing.T) {
	zeroEye := validSnap()
	zeroEye.EyeHeight = 0
	negEye := validSnap()
	negEye.EyeHeight = -1
	zeroWidth := validSnap()
	zeroWidth.HitboxWidth = 0
	zeroHeight := validSnap()
	zeroHeight.HitboxHeight = 0

	cases := []struct {
		name    string
		data    []byte
		wantSub string
	}{
		{"empty data", nil, "empty data"},
		{"undecodable gob", encodeWire(t, "not a player snapshot"), "decode snapshot"},
		{"zero eyeHeight", encodeWire(t, zeroEye), "non-positive"},
		{"negative eyeHeight", encodeWire(t, negEye), "non-positive"},
		{"zero hitboxWidth", encodeWire(t, zeroWidth), "non-positive"},
		{"zero hitboxHeight", encodeWire(t, zeroHeight), "non-positive"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := player.Deserialize(c.data)
			if err == nil {
				t.Fatalf("Deserialize(%s) = nil error, want substring %q", c.name, c.wantSub)
			}
			if !strings.Contains(err.Error(), c.wantSub) {
				t.Fatalf("Deserialize(%s) error = %q, want substring %q", c.name, err.Error(), c.wantSub)
			}
		})
	}
}

// TestDeserializeRenormalizesLook checks the deliberate leniency in
// Deserialize: a hand-edited save with out-of-range yaw/pitch still loads,
// because Deserialize funnels them back through NewLook (which wraps yaw and
// clamps pitch) rather than rejecting them.
func TestDeserializeRenormalizesLook(t *testing.T) {
	snap := validSnap()
	snap.Yaw = 10.0  // outside (-pi, pi], must wrap
	snap.Pitch = 3.0 // beyond the pitch limit (~1.57), must clamp

	got, err := player.Deserialize(encodeWire(t, snap))
	if err != nil {
		t.Fatalf("Deserialize: %v", err)
	}

	want := player.NewLook(snap.Yaw, snap.Pitch)
	// Guard the test itself: these inputs must genuinely be out of range, or
	// the assertion below would pass without exercising re-normalization.
	if want.Yaw() == snap.Yaw {
		t.Fatalf("test setup: yaw %v was not wrapped by NewLook", snap.Yaw)
	}
	if want.Pitch() == snap.Pitch {
		t.Fatalf("test setup: pitch %v was not clamped by NewLook", snap.Pitch)
	}
	if got.Look().Yaw() != want.Yaw() {
		t.Fatalf("loaded yaw = %v, want re-normalized %v", got.Look().Yaw(), want.Yaw())
	}
	if got.Look().Pitch() != want.Pitch() {
		t.Fatalf("loaded pitch = %v, want clamped %v", got.Look().Pitch(), want.Pitch())
	}
}

// TestSerializeRoundTrip is the positive guarantee: NewVec3/NewLook values
// set on a Model survive Serialize -> Deserialize with every accessor
// agreeing.
func TestSerializeRoundTrip(t *testing.T) {
	p := player.New(player.NewVec3(10.5, 12, 8.25))
	p = player.SetLook(p, player.NewLook(1.2, -0.3))
	p = player.SetVelocity(p, player.NewVec3(0.5, -1.0, 0.25))
	p = player.SetOnGround(p, true)

	data, err := p.Serialize()
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	if len(data) == 0 {
		t.Fatalf("Serialize returned empty blob")
	}

	got, err := player.Deserialize(data)
	if err != nil {
		t.Fatalf("Deserialize: %v", err)
	}

	if got.Position() != p.Position() {
		t.Fatalf("Position = %+v, want %+v", got.Position(), p.Position())
	}
	if got.Look().Yaw() != p.Look().Yaw() || got.Look().Pitch() != p.Look().Pitch() {
		t.Fatalf("Look = %+v, want %+v", got.Look(), p.Look())
	}
	if got.Velocity() != p.Velocity() {
		t.Fatalf("Velocity = %+v, want %+v", got.Velocity(), p.Velocity())
	}
	if got.OnGround() != p.OnGround() {
		t.Fatalf("OnGround = %v, want %v", got.OnGround(), p.OnGround())
	}
	if got.EyeHeight() != p.EyeHeight() ||
		got.HitboxWidth() != p.HitboxWidth() ||
		got.HitboxHeight() != p.HitboxHeight() {
		t.Fatalf("dimensions diverged after round-trip")
	}
}
