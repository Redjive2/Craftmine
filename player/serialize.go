package player

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math"
)

// playerSnapshot is the on-disk shape of player.Model. Exported fields so gob
// can encode them without needing GobEncoder methods on every nested type.
type playerSnapshot struct {
	PositionX, PositionY, PositionZ float64
	Yaw                             float64
	Pitch                           float64
	VelocityX, VelocityY, VelocityZ float64
	OnGround                        bool
	EyeHeight                       float64
	HitboxWidth                     float64
	HitboxHeight                    float64
}

// Serialize encodes the player Model to a gob blob. Pure: same Model always
// produces the same bytes (subject to gob's stability guarantees).
func (m Model) Serialize() ([]byte, error) {
	snap := playerSnapshot{
		PositionX:    m.position.x,
		PositionY:    m.position.y,
		PositionZ:    m.position.z,
		Yaw:          m.look.yaw,
		Pitch:        m.look.pitch,
		VelocityX:    m.velocity.x,
		VelocityY:    m.velocity.y,
		VelocityZ:    m.velocity.z,
		OnGround:     m.onGround,
		EyeHeight:    m.eyeHeight,
		HitboxWidth:  m.hitboxWidth,
		HitboxHeight: m.hitboxHeight,
	}
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(snap); err != nil {
		return nil, fmt.Errorf("player: encode snapshot: %w", err)
	}
	return buf.Bytes(), nil
}

// Deserialize decodes a gob blob produced by Serialize and returns a fresh
// Model. Aggressively validates numeric components — a corrupt save that
// snuck NaN or an out-of-range pitch through gob should not be honored
// silently by the rest of the codebase.
func Deserialize(data []byte) (Model, error) {
	if len(data) == 0 {
		return Model{}, fmt.Errorf("player: deserialize: empty data")
	}
	var snap playerSnapshot
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&snap); err != nil {
		return Model{}, fmt.Errorf("player: decode snapshot: %w", err)
	}
	if err := validateFloats(snap); err != nil {
		return Model{}, err
	}
	if snap.EyeHeight <= 0 || snap.HitboxWidth <= 0 || snap.HitboxHeight <= 0 {
		return Model{}, fmt.Errorf("player: deserialize: non-positive hitbox/eye (%v, %v, %v)",
			snap.EyeHeight, snap.HitboxWidth, snap.HitboxHeight)
	}
	// NewLook re-normalizes yaw and clamps pitch back into range, so a
	// slightly-out-of-range value from a hand-edited save still loads.
	look := NewLook(snap.Yaw, snap.Pitch)
	return Model{
		position:     Vec3{x: snap.PositionX, y: snap.PositionY, z: snap.PositionZ},
		look:         look,
		velocity:     Vec3{x: snap.VelocityX, y: snap.VelocityY, z: snap.VelocityZ},
		onGround:     snap.OnGround,
		eyeHeight:    snap.EyeHeight,
		hitboxWidth:  snap.HitboxWidth,
		hitboxHeight: snap.HitboxHeight,
	}, nil
}

func validateFloats(snap playerSnapshot) error {
	floats := []struct {
		name string
		v    float64
	}{
		{"position.x", snap.PositionX},
		{"position.y", snap.PositionY},
		{"position.z", snap.PositionZ},
		{"yaw", snap.Yaw},
		{"pitch", snap.Pitch},
		{"velocity.x", snap.VelocityX},
		{"velocity.y", snap.VelocityY},
		{"velocity.z", snap.VelocityZ},
		{"eyeHeight", snap.EyeHeight},
		{"hitboxWidth", snap.HitboxWidth},
		{"hitboxHeight", snap.HitboxHeight},
	}
	for _, f := range floats {
		if math.IsNaN(f.v) {
			return fmt.Errorf("player: deserialize: %s is NaN", f.name)
		}
		if math.IsInf(f.v, 0) {
			return fmt.Errorf("player: deserialize: %s is infinite", f.name)
		}
	}
	return nil
}
