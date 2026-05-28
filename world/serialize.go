package world

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/redjive2/Craftmine/blocks"
)

// treeSnapshot mirrors Tree's private fields with exported names so gob can
// round-trip them without GobEncoder gymnastics. Kept in lockstep with Tree.
type treeSnapshot struct {
	X            int
	Z            int
	BaseHeight   int
	TrunkHeight  int
	CanopyRadius int
}

// worldSnapshot is the on-disk shape of world.Model. Exported fields so gob
// can read them; treesByChunk is omitted because it is derived from Trees and
// rebuilt on load.
type worldSnapshot struct {
	Seed      int64
	Width     int
	Depth     int
	MaxHeight int
	Heights   []int16
	Trees     []treeSnapshot
	Grass     blocks.BlockID
	Dirt      blocks.BlockID
	Stone     blocks.BlockID
	Wood      blocks.BlockID
	Leaves    blocks.BlockID
	DirtDepth int
}

// Serialize encodes the world Model to a gob blob. The blob is self-contained:
// Deserialize can rebuild a working Model (including the per-chunk tree index)
// from it alone, given that the dimensions remain multiples of ChunkSize.
func (m Model) Serialize() ([]byte, error) {
	snap := worldSnapshot{
		Seed:      m.seed,
		Width:     m.width,
		Depth:     m.depth,
		MaxHeight: m.maxHeight,
		Heights:   append([]int16(nil), m.heights...),
		Trees:     treesToSnapshots(m.trees),
		Grass:     m.grass,
		Dirt:      m.dirt,
		Stone:     m.stone,
		Wood:      m.wood,
		Leaves:    m.leaves,
		DirtDepth: m.dirtDepth,
	}
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(snap); err != nil {
		return nil, fmt.Errorf("world: encode snapshot: %w", err)
	}
	return buf.Bytes(), nil
}

// Deserialize decodes a gob blob produced by Serialize and returns a fresh
// Model. The decoded dimensions are validated against the same rules Generate
// enforces, so a corrupted blob with non-multiple-of-ChunkSize sizes is
// rejected up front rather than producing a malformed Model.
func Deserialize(data []byte) (Model, error) {
	if len(data) == 0 {
		return Model{}, fmt.Errorf("world: deserialize: empty data")
	}
	var snap worldSnapshot
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&snap); err != nil {
		return Model{}, fmt.Errorf("world: decode snapshot: %w", err)
	}
	if err := validateOptions(GenerateOptions{
		Width:     snap.Width,
		Depth:     snap.Depth,
		MaxHeight: snap.MaxHeight,
		DirtDepth: snap.DirtDepth,
	}); err != nil {
		return Model{}, fmt.Errorf("world: deserialize: %w", err)
	}
	expectedHeights := snap.Width * snap.Depth
	if len(snap.Heights) != expectedHeights {
		return Model{}, fmt.Errorf("world: deserialize: heights len=%d, want %d (%dx%d)",
			len(snap.Heights), expectedHeights, snap.Width, snap.Depth)
	}
	for i, h := range snap.Heights {
		if h < 0 || int(h) >= snap.MaxHeight {
			return Model{}, fmt.Errorf("world: deserialize: heights[%d]=%d outside [0, %d)",
				i, h, snap.MaxHeight)
		}
	}
	trees := snapshotsToTrees(snap.Trees)
	chunkCountX := snap.Width / ChunkSize
	chunkCountZ := snap.Depth / ChunkSize
	return Model{
		seed:         snap.Seed,
		width:        snap.Width,
		depth:        snap.Depth,
		maxHeight:    snap.MaxHeight,
		heights:      snap.Heights,
		trees:        trees,
		treesByChunk: buildTreeIndex(trees, chunkCountX, chunkCountZ),
		grass:        snap.Grass,
		dirt:         snap.Dirt,
		stone:        snap.Stone,
		wood:         snap.Wood,
		leaves:       snap.Leaves,
		dirtDepth:    snap.DirtDepth,
	}, nil
}

func treesToSnapshots(trees []Tree) []treeSnapshot {
	out := make([]treeSnapshot, len(trees))
	for i, t := range trees {
		out[i] = treeSnapshot{
			X:            t.x,
			Z:            t.z,
			BaseHeight:   t.baseHeight,
			TrunkHeight:  t.trunkHeight,
			CanopyRadius: t.canopyRadius,
		}
	}
	return out
}

func snapshotsToTrees(snaps []treeSnapshot) []Tree {
	out := make([]Tree, len(snaps))
	for i, s := range snaps {
		out[i] = Tree{
			x:            s.X,
			z:            s.Z,
			baseHeight:   s.BaseHeight,
			trunkHeight:  s.TrunkHeight,
			canopyRadius: s.CanopyRadius,
		}
	}
	return out
}
