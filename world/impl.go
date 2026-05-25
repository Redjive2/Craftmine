package world

import (
	"fmt"

	"github.com/redjive2/Craftmine/blocks"
)

// World is the behavior interface for the world module.
//
// Callers depend on World, not on the concrete Impl, so a test double can be
// substituted (e.g. a fixed-world for camera/menu tests).
type World interface {
	Generate(seed int64, registry blocks.Model, registryImpl blocks.Blocks, opts GenerateOptions) (Model, error)
	BlockAt(m Model, x, y, z int) blocks.BlockID
	TreesInChunk(m Model, chunkX, chunkZ int) []Tree
}

// Impl is the zero-field implementation of World. All behavior hangs off Impl;
// state lives in Model and is passed as an argument to every method.
type Impl struct{}

// Compile-time check that Impl satisfies World.
var _ World = Impl{}

// GenerateOptions controls world dimensions and a few generation knobs.
//
// All fields have sensible defaults via DefaultOptions; this struct exists so
// tests can use small worlds without polluting the public API with many
// arguments.
type GenerateOptions struct {
	Width     int
	Depth     int
	MaxHeight int
	DirtDepth int
}

// DefaultOptions returns the Vision.md target: 512x512 blocks, 256 tall.
func DefaultOptions() GenerateOptions {
	return GenerateOptions{
		Width:     DefaultWidth,
		Depth:     DefaultDepth,
		MaxHeight: DefaultMaxHeight,
		DirtDepth: DefaultDirtDepth,
	}
}

// Generate builds a Model from (seed, registry, opts). Pure: same inputs
// always produce the same Model.
//
// Registry lookups are by name (per Vision.md "consume the registry"); the
// resolved BlockIDs are cached in the Model so block-id queries downstream
// don't need to re-traverse the registry.
func (Impl) Generate(seed int64, registry blocks.Model, registryImpl blocks.Blocks, opts GenerateOptions) (Model, error) {
	if err := validateOptions(opts); err != nil {
		return Model{}, err
	}
	grass, err := requireBlock(registry, registryImpl, "grass")
	if err != nil {
		return Model{}, err
	}
	dirt, err := requireBlock(registry, registryImpl, "dirt")
	if err != nil {
		return Model{}, err
	}
	stone, err := requireBlock(registry, registryImpl, "stone")
	if err != nil {
		return Model{}, err
	}
	wood, err := requireBlock(registry, registryImpl, "wood")
	if err != nil {
		return Model{}, err
	}
	leaves, err := requireBlock(registry, registryImpl, "leaves")
	if err != nil {
		return Model{}, err
	}

	heights := generateHeights(seed, opts.Width, opts.Depth, opts.MaxHeight)
	trees := placeTrees(seed, opts.Width, opts.Depth, opts.MaxHeight, heights)

	chunkCountX := opts.Width / ChunkSize
	chunkCountZ := opts.Depth / ChunkSize
	index := buildTreeIndex(trees, chunkCountX, chunkCountZ)

	return Model{
		seed:         seed,
		width:        opts.Width,
		depth:        opts.Depth,
		maxHeight:    opts.MaxHeight,
		heights:      heights,
		trees:        trees,
		treesByChunk: index,
		grass:        grass,
		dirt:         dirt,
		stone:        stone,
		wood:         wood,
		leaves:       leaves,
		dirtDepth:    opts.DirtDepth,
	}, nil
}

// BlockAt returns the BlockID at (x, y, z). Out-of-bounds queries return Air.
//
// Resolution order:
//  1. Trees rooted in the same chunk as (x, z) may override otherwise-empty
//     space with Wood (trunk) or Leaves (canopy).
//  2. Terrain: at y == surface -> Grass; (surface-DirtDepth) <= y < surface
//     -> Dirt; y < that -> Stone; y > surface and no tree -> Air.
func (Impl) BlockAt(m Model, x, y, z int) blocks.BlockID {
	if x < 0 || x >= m.width || z < 0 || z >= m.depth || y < 0 || y >= m.maxHeight {
		return Air
	}
	surface := int(m.heights[x*m.depth+z])
	if y <= surface {
		switch {
		case y == surface:
			// Tree trunks root at surface+1, so the surface block stays grass
			// even directly under a tree.
			return m.grass
		case y >= surface-m.dirtDepth:
			return m.dirt
		default:
			return m.stone
		}
	}
	// Above-surface: trees only.
	if hit, ok := treeBlockAt(m, x, y, z); ok {
		return hit
	}
	return Air
}

// TreesInChunk returns the trees whose footprint touches chunk (chunkX, chunkZ).
// Tree blocks (trunk + canopy) for any tree NOT in this list are guaranteed
// to fall outside the chunk's column range, so renderers can iterate per-chunk
// without scanning every tree.
//
// Out-of-range chunk coords return an empty slice.
func (Impl) TreesInChunk(m Model, chunkX, chunkZ int) []Tree {
	if chunkX < 0 || chunkX >= m.ChunkCountX() || chunkZ < 0 || chunkZ >= m.ChunkCountZ() {
		return nil
	}
	key := chunkX*m.ChunkCountZ() + chunkZ
	indices := m.treesByChunk[key]
	out := make([]Tree, len(indices))
	for i, idx := range indices {
		out[i] = m.trees[idx]
	}
	return out
}

// treeBlockAt looks up whether (x, y, z) lies inside any tree's trunk or
// canopy. Uses the per-chunk index so the scan is O(trees in chunk), not
// O(total trees).
func treeBlockAt(m Model, x, y, z int) (blocks.BlockID, bool) {
	chunkX := x / ChunkSize
	chunkZ := z / ChunkSize
	key := chunkX*m.ChunkCountZ() + chunkZ
	if key < 0 || key >= len(m.treesByChunk) {
		return Air, false
	}
	wood := uint16(m.wood)
	leaves := uint16(m.leaves)
	for _, idx := range m.treesByChunk[key] {
		if hit, ok := blockInTree(m.trees[idx], x, y, z, wood, leaves); ok {
			return blocks.BlockID(hit), true
		}
	}
	return Air, false
}

func validateOptions(opts GenerateOptions) error {
	if opts.Width <= 0 || opts.Depth <= 0 || opts.MaxHeight <= 0 {
		return fmt.Errorf("world: width=%d depth=%d maxHeight=%d must all be positive",
			opts.Width, opts.Depth, opts.MaxHeight)
	}
	if opts.Width%ChunkSize != 0 || opts.Depth%ChunkSize != 0 {
		return fmt.Errorf("world: width=%d depth=%d must be multiples of ChunkSize=%d",
			opts.Width, opts.Depth, ChunkSize)
	}
	if opts.DirtDepth < 0 {
		return fmt.Errorf("world: dirtDepth=%d must be non-negative", opts.DirtDepth)
	}
	if opts.DirtDepth >= opts.MaxHeight {
		return fmt.Errorf("world: dirtDepth=%d must be less than maxHeight=%d",
			opts.DirtDepth, opts.MaxHeight)
	}
	return nil
}

func requireBlock(registry blocks.Model, registryImpl blocks.Blocks, name string) (blocks.BlockID, error) {
	b, ok := registryImpl.LookupByName(registry, name)
	if !ok {
		return 0, fmt.Errorf("world: block registry missing required kind %q", name)
	}
	return b.ID(), nil
}
