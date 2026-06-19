package world_test

import (
	"bytes"
	"encoding/gob"
	"strings"
	"testing"

	"github.com/redjive2/Craftmine/blocks"
	"github.com/redjive2/Craftmine/world"
)

// wireSnapshot mirrors world's unexported worldSnapshot by exported field
// name so gob can decode a value of this type straight into the internal
// snapshot. Only the fields the validation paths touch are present; gob
// matches by name and leaves the omitted (block-id, tree) fields zeroed,
// which is fine because Deserialize rejects these blobs before it ever
// consults them.
type wireSnapshot struct {
	Seed      int64
	Width     int
	Depth     int
	MaxHeight int
	Heights   []int16
	DirtDepth int
}

// encodeWire gob-encodes any value into a blob suitable for handing to
// world.Deserialize. Encoding an unrelated type (e.g. a string) is the
// reliable way to manufacture a "looks like gob but isn't a snapshot"
// decode failure.
func encodeWire(t *testing.T, v any) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(v); err != nil {
		t.Fatalf("gob encode: %v", err)
	}
	return buf.Bytes()
}

// heightsOf returns a heightmap of length n with every column set to val.
func heightsOf(n int, val int16) []int16 {
	out := make([]int16, n)
	for i := range out {
		out[i] = val
	}
	return out
}

// testWorld builds a small, valid world.Model for the round-trip check.
// Mirrors save/save_test.go's helper so the two stay recognizably the same.
func testWorld(t *testing.T) world.Model {
	t.Helper()
	var blocksImpl blocks.Blocks = blocks.Impl{}
	registry, err := blocks.NewWithDefaults(blocksImpl)
	if err != nil {
		t.Fatalf("blocks.NewWithDefaults: %v", err)
	}
	var worldImpl world.World = world.Impl{}
	m, err := worldImpl.Generate(2026, registry, blocksImpl, world.GenerateOptions{
		Width: 32, Depth: 32, MaxHeight: 32, DirtDepth: 3,
	})
	if err != nil {
		t.Fatalf("world.Generate: %v", err)
	}
	return m
}

// TestDeserializeRejects is the batch guard over every error branch in
// world.Deserialize: empty input, an undecodable blob, each validateOptions
// rule, the heights length mismatch, and per-column range checks. One table
// exercises them all so a regression that drops any guard fails loudly here.
func TestDeserializeRejects(t *testing.T) {
	// A valid heightmap for a 16x16 world is 256 columns in [0, MaxHeight).
	validHeights := heightsOf(16*16, 1)

	outOfRangeHigh := heightsOf(16*16, 1)
	outOfRangeHigh[5] = 32 // == MaxHeight, so >= MaxHeight: rejected

	outOfRangeLow := heightsOf(16*16, 1)
	outOfRangeLow[7] = -1 // < 0: rejected

	cases := []struct {
		name    string
		data    []byte
		wantSub string
	}{
		{
			name:    "empty data",
			data:    nil,
			wantSub: "empty data",
		},
		{
			name:    "undecodable gob",
			data:    encodeWire(t, "not a world snapshot"),
			wantSub: "decode snapshot",
		},
		{
			name: "width not multiple of ChunkSize",
			data: encodeWire(t, wireSnapshot{
				Width: 17, Depth: 16, MaxHeight: 32, DirtDepth: 3,
			}),
			wantSub: "multiples of ChunkSize",
		},
		{
			name: "depth not multiple of ChunkSize",
			data: encodeWire(t, wireSnapshot{
				Width: 16, Depth: 17, MaxHeight: 32, DirtDepth: 3,
			}),
			wantSub: "multiples of ChunkSize",
		},
		{
			name: "non-positive maxHeight",
			data: encodeWire(t, wireSnapshot{
				Width: 16, Depth: 16, MaxHeight: 0, DirtDepth: 3,
			}),
			wantSub: "must all be positive",
		},
		{
			name: "negative dirtDepth",
			data: encodeWire(t, wireSnapshot{
				Width: 16, Depth: 16, MaxHeight: 32, DirtDepth: -1,
			}),
			wantSub: "must be non-negative",
		},
		{
			name: "dirtDepth not less than maxHeight",
			data: encodeWire(t, wireSnapshot{
				Width: 16, Depth: 16, MaxHeight: 32, DirtDepth: 32,
			}),
			wantSub: "must be less than maxHeight",
		},
		{
			name: "heights length mismatch",
			data: encodeWire(t, wireSnapshot{
				Width: 16, Depth: 16, MaxHeight: 32, DirtDepth: 3,
				Heights: []int16{1, 2, 3},
			}),
			wantSub: "heights len",
		},
		{
			name: "height at or above maxHeight",
			data: encodeWire(t, wireSnapshot{
				Width: 16, Depth: 16, MaxHeight: 32, DirtDepth: 3,
				Heights: outOfRangeHigh,
			}),
			wantSub: "outside [0,",
		},
		{
			name: "negative height",
			data: encodeWire(t, wireSnapshot{
				Width: 16, Depth: 16, MaxHeight: 32, DirtDepth: 3,
				Heights: outOfRangeLow,
			}),
			wantSub: "outside [0,",
		},
		{
			// Sanity anchor: the "valid" heights are only valid once paired
			// with matching dimensions; on their own with mismatched dims
			// they must still be rejected, not silently accepted.
			name: "valid heights but wrong dimensions",
			data: encodeWire(t, wireSnapshot{
				Width: 32, Depth: 16, MaxHeight: 32, DirtDepth: 3,
				Heights: validHeights,
			}),
			wantSub: "heights len",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := world.Deserialize(c.data)
			if err == nil {
				t.Fatalf("Deserialize(%s) = nil error, want error containing %q", c.name, c.wantSub)
			}
			if !strings.Contains(err.Error(), c.wantSub) {
				t.Fatalf("Deserialize(%s) error = %q, want substring %q", c.name, err.Error(), c.wantSub)
			}
		})
	}
}

// TestSerializeRoundTrip is the positive counterpart: a Generate'd Model
// survives Serialize -> Deserialize with every accessor agreeing, including
// the heightmap and the tree list rebuilt from the blob.
func TestSerializeRoundTrip(t *testing.T) {
	w := testWorld(t)

	data, err := w.Serialize()
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	if len(data) == 0 {
		t.Fatalf("Serialize returned empty blob")
	}

	got, err := world.Deserialize(data)
	if err != nil {
		t.Fatalf("Deserialize: %v", err)
	}

	if got.Seed() != w.Seed() ||
		got.Width() != w.Width() ||
		got.Depth() != w.Depth() ||
		got.MaxHeight() != w.MaxHeight() ||
		got.DirtDepth() != w.DirtDepth() ||
		got.TreeCount() != w.TreeCount() {
		t.Fatalf("scalar fields diverged: got seed=%d %dx%d maxH=%d dirt=%d trees=%d want seed=%d %dx%d maxH=%d dirt=%d trees=%d",
			got.Seed(), got.Width(), got.Depth(), got.MaxHeight(), got.DirtDepth(), got.TreeCount(),
			w.Seed(), w.Width(), w.Depth(), w.MaxHeight(), w.DirtDepth(), w.TreeCount())
	}
	if got.Grass() != w.Grass() || got.Dirt() != w.Dirt() ||
		got.Stone() != w.Stone() || got.Wood() != w.Wood() ||
		got.Leaves() != w.Leaves() {
		t.Fatalf("cached block IDs diverged after round-trip")
	}
	for x := 0; x < w.Width(); x++ {
		for z := 0; z < w.Depth(); z++ {
			if got.HeightAt(x, z) != w.HeightAt(x, z) {
				t.Fatalf("HeightAt(%d,%d) = %d, want %d", x, z, got.HeightAt(x, z), w.HeightAt(x, z))
			}
		}
	}
	origTrees := w.Trees()
	gotTrees := got.Trees()
	if len(gotTrees) != len(origTrees) {
		t.Fatalf("tree count = %d, want %d", len(gotTrees), len(origTrees))
	}
	for i := range origTrees {
		if gotTrees[i].X() != origTrees[i].X() ||
			gotTrees[i].Z() != origTrees[i].Z() ||
			gotTrees[i].BaseHeight() != origTrees[i].BaseHeight() ||
			gotTrees[i].TrunkHeight() != origTrees[i].TrunkHeight() ||
			gotTrees[i].CanopyRadius() != origTrees[i].CanopyRadius() {
			t.Fatalf("tree[%d] diverged: %+v vs %+v", i, gotTrees[i], origTrees[i])
		}
	}
}
