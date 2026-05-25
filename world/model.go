// Package world owns voxel terrain generation and storage for one game world.
//
// Per Vision.md this module follows the Craftmine module pattern: a Model
// holding all state (heightmap, trees, cached block IDs), an Impl carrying
// behavior with no fields, and a World interface that Impl satisfies. World
// generation is pure — given the same (seed, block registry, options) it
// returns the same Model.
//
// The world is partitioned into ChunkSize x ChunkSize columns along x/z so
// renderers can iterate one neighborhood at a time. Storage is column-oriented:
// the Model keeps a heightmap and a list of tree placements, and the actual
// per-voxel block id is derived on demand from those (see BlockAt in impl.go).
// This keeps memory bounded even at the Vision.md target of 512x512x256.
package world

import "github.com/redjive2/Craftmine/blocks"

const (
	// Air is the sentinel BlockID returned for "no block here". Registered
	// blocks always have ids >= 1, so 0 is safe to reserve.
	Air blocks.BlockID = 0

	// DefaultWidth, DefaultDepth, DefaultMaxHeight match the Vision.md target.
	// Tests use smaller worlds; the menu's New Game path uses these defaults.
	DefaultWidth     = 512
	DefaultDepth     = 512
	DefaultMaxHeight = 256

	// ChunkSize is the side length of one chunk in blocks. The world's x and
	// z extents must be multiples of ChunkSize. Vertical extent is not chunked
	// — Vision.md's 0..256 fits comfortably in one vertical slab.
	ChunkSize = 16

	// DefaultDirtDepth is the dirt-layer thickness directly under the grass
	// surface; everything deeper is stone.
	DefaultDirtDepth = 3
)

// Tree is one fully-resolved tree placement in the world. Fields are private;
// accessors below expose them. Construct via NewTree so future validation has
// a single entry point.
type Tree struct {
	x            int
	z            int
	baseHeight   int
	trunkHeight  int
	canopyRadius int
}

// NewTree builds a Tree at (x, z). baseHeight is the y of the first trunk
// block (typically heightmap[x,z]+1). trunkHeight is the number of wood blocks
// stacked. canopyRadius controls the leaf-cluster size.
func NewTree(x, z, baseHeight, trunkHeight, canopyRadius int) Tree {
	return Tree{
		x:            x,
		z:            z,
		baseHeight:   baseHeight,
		trunkHeight:  trunkHeight,
		canopyRadius: canopyRadius,
	}
}

// X and Z are the trunk's column position.
func (t Tree) X() int { return t.x }
func (t Tree) Z() int { return t.z }

// BaseHeight is the y of the lowest trunk block.
func (t Tree) BaseHeight() int { return t.baseHeight }

// TrunkHeight is the number of wood blocks in the trunk.
func (t Tree) TrunkHeight() int { return t.trunkHeight }

// CanopyRadius controls the leaf-cluster size around the trunk top.
func (t Tree) CanopyRadius() int { return t.canopyRadius }

// TopY is the y of the topmost trunk block (canopy center).
func (t Tree) TopY() int { return t.baseHeight + t.trunkHeight - 1 }

// Model holds one generated world.
//
// All fields are private; accessors below expose what callers need. The Model
// is effectively immutable once Generate returns: there are no mutators on
// Impl yet, and the heights slice is treated as read-only. (Future block
// place/destroy will go through an explicit transition function.)
type Model struct {
	seed      int64
	width     int
	depth     int
	maxHeight int

	// heights[x*depth + z] is the y of the topmost solid block of that column
	// (i.e. the grass-surface block). int16 keeps the heightmap to ~512KB at
	// the 512x512 target — well below worrying.
	heights []int16

	// trees is the list of placements in deterministic generation order.
	trees []Tree

	// treesByChunk[cx*chunkCountZ + cz] indexes into the trees slice. A tree
	// is added to every chunk whose footprint its trunk-or-canopy may touch,
	// so per-chunk queries don't need to scan all trees. Aliasing across
	// Model copies is fine because nothing mutates these slices.
	treesByChunk [][]int

	// Cached block ids from the registry so BlockAt doesn't need a registry
	// argument. They were resolved by name during Generate.
	grass  blocks.BlockID
	dirt   blocks.BlockID
	stone  blocks.BlockID
	wood   blocks.BlockID
	leaves blocks.BlockID

	dirtDepth int
}

// Seed returns the seed used to generate this world.
func (m Model) Seed() int64 { return m.seed }

// Width returns the world's x extent in blocks.
func (m Model) Width() int { return m.width }

// Depth returns the world's z extent in blocks.
func (m Model) Depth() int { return m.depth }

// MaxHeight returns the world's vertical extent in blocks. Valid y values are
// 0..MaxHeight-1.
func (m Model) MaxHeight() int { return m.maxHeight }

// ChunkCountX returns the number of chunks along x.
func (m Model) ChunkCountX() int { return m.width / ChunkSize }

// ChunkCountZ returns the number of chunks along z.
func (m Model) ChunkCountZ() int { return m.depth / ChunkSize }

// HeightAt returns the surface y at column (x, z). Out-of-bounds queries
// return 0 (treated as "bedrock floor") rather than panicking, so renderers
// can probe edges without special-casing.
func (m Model) HeightAt(x, z int) int {
	if x < 0 || x >= m.width || z < 0 || z >= m.depth {
		return 0
	}
	return int(m.heights[x*m.depth+z])
}

// TreeCount is the number of trees in the world.
func (m Model) TreeCount() int { return len(m.trees) }

// Trees returns a defensive copy of the tree list. Mutating the returned
// slice does not affect the Model.
func (m Model) Trees() []Tree {
	out := make([]Tree, len(m.trees))
	copy(out, m.trees)
	return out
}

// Grass, Dirt, Stone, Wood, Leaves return the BlockIDs cached at generation
// time. They are part of the Model so BlockAt and renderers don't need a
// registry argument.
func (m Model) Grass() blocks.BlockID  { return m.grass }
func (m Model) Dirt() blocks.BlockID   { return m.dirt }
func (m Model) Stone() blocks.BlockID  { return m.stone }
func (m Model) Wood() blocks.BlockID   { return m.wood }
func (m Model) Leaves() blocks.BlockID { return m.leaves }

// DirtDepth is the dirt-layer thickness directly below the grass surface.
func (m Model) DirtDepth() int { return m.dirtDepth }
