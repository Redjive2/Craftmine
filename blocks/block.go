package blocks

// BlockID is the stable numeric identifier of a registered block kind.
// Core blocks reserve a low ID range; mods are expected to claim a disjoint
// range to avoid collisions (see defaults.go for the core assignments).
type BlockID uint16

// Block is the immutable descriptor of one block kind.
//
// Fields are private to keep instances effectively read-only after construction;
// all data is reached through the accessor methods below. Construct via
// NewBlock so future validation has a single entry point.
type Block struct {
	id    BlockID
	name  string
	red   float32
	green float32
	blue  float32
	solid bool
}

// NewBlock builds a Block descriptor. Color channels are placeholder material
// values in the 0..1 range; the rendering layer (cmd/blocks-demo, future world
// gen) decides how to map them to textures or shaders.
func NewBlock(id BlockID, name string, red, green, blue float32, solid bool) Block {
	return Block{
		id:    id,
		name:  name,
		red:   clamp01(red),
		green: clamp01(green),
		blue:  clamp01(blue),
		solid: solid,
	}
}

// ID returns the block's stable identifier.
func (b Block) ID() BlockID { return b.id }

// Name returns the block's human-readable name (also its unique registry key).
func (b Block) Name() string { return b.name }

// Color returns the block's placeholder material color as (red, green, blue),
// each in the 0..1 range. Returned as separate scalars to avoid leaking a
// mutable array or pointer to caller code.
func (b Block) Color() (float32, float32, float32) { return b.red, b.green, b.blue }

// Solid reports whether the block is a full opaque cube for collision and
// occlusion purposes. Leaves are solid for now; transparency is out of scope.
func (b Block) Solid() bool { return b.solid }

func clamp01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
