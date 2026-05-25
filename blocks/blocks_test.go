package blocks_test

import (
	"strings"
	"testing"

	"github.com/redjive2/Craftmine/blocks"
)

// TestNewIsEmpty checks that a fresh Model carries no registrations.
func TestNewIsEmpty(t *testing.T) {
	m := blocks.New()
	if got := m.Count(); got != 0 {
		t.Fatalf("New().Count() = %d, want 0", got)
	}
	var impl blocks.Blocks = blocks.Impl{}
	if got := impl.All(m); len(got) != 0 {
		t.Fatalf("All(New()) = %v, want empty", got)
	}
}

// TestRegisterReturnsNewModel confirms registration is non-mutating.
func TestRegisterReturnsNewModel(t *testing.T) {
	var impl blocks.Blocks = blocks.Impl{}
	before := blocks.New()
	after, err := impl.Register(before, blocks.NewBlock(42, "test", 0.5, 0.5, 0.5, true))
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if before.Count() != 0 {
		t.Fatalf("input Model mutated: before.Count() = %d, want 0", before.Count())
	}
	if after.Count() != 1 {
		t.Fatalf("after.Count() = %d, want 1", after.Count())
	}
}

// TestRegisterValidation covers the validation guarantees of the registry:
// empty names, duplicate IDs, and duplicate names are all rejected.
func TestRegisterValidation(t *testing.T) {
	var impl blocks.Blocks = blocks.Impl{}
	base, err := impl.Register(blocks.New(), blocks.NewBlock(1, "grass", 0, 1, 0, true))
	if err != nil {
		t.Fatalf("seed Register failed: %v", err)
	}

	cases := []struct {
		name    string
		block   blocks.Block
		wantSub string
	}{
		{"empty name", blocks.NewBlock(2, "", 0, 0, 0, true), "must not be empty"},
		{"duplicate id", blocks.NewBlock(1, "dirt", 0, 0, 0, true), "duplicate id"},
		{"duplicate name", blocks.NewBlock(2, "grass", 0, 0, 0, true), "duplicate name"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := impl.Register(base, tc.block)
			if err == nil {
				t.Fatalf("Register(%v) returned nil error, want error containing %q", tc.block, tc.wantSub)
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("Register error = %q, want substring %q", err.Error(), tc.wantSub)
			}
		})
	}
}

// TestLookup covers both ID and name lookup, hit and miss.
func TestLookup(t *testing.T) {
	var impl blocks.Blocks = blocks.Impl{}
	m, err := blocks.NewWithDefaults(impl)
	if err != nil {
		t.Fatalf("NewWithDefaults: %v", err)
	}

	if b, ok := impl.Lookup(m, blocks.IDStone); !ok || b.Name() != "stone" {
		t.Fatalf("Lookup(IDStone) = (%v, %v), want stone/true", b, ok)
	}
	if _, ok := impl.Lookup(m, blocks.BlockID(9999)); ok {
		t.Fatalf("Lookup(unknown id) returned ok=true, want false")
	}
	if b, ok := impl.LookupByName(m, "wood"); !ok || b.ID() != blocks.IDWood {
		t.Fatalf("LookupByName(wood) = (%v, %v), want IDWood/true", b, ok)
	}
	if _, ok := impl.LookupByName(m, "obsidian"); ok {
		t.Fatalf("LookupByName(obsidian) returned ok=true, want false")
	}
}

// TestDefaultsRegistersAllFive is the headline acceptance check: the registry
// produced by NewWithDefaults contains the five vanilla block kinds with the
// expected names and unique IDs.
func TestDefaultsRegistersAllFive(t *testing.T) {
	var impl blocks.Blocks = blocks.Impl{}
	m, err := blocks.NewWithDefaults(impl)
	if err != nil {
		t.Fatalf("NewWithDefaults: %v", err)
	}

	wantNames := []string{"grass", "dirt", "stone", "wood", "leaves"}
	if impl.Count(m) != len(wantNames) {
		t.Fatalf("Count = %d, want %d", impl.Count(m), len(wantNames))
	}

	all := impl.All(m)
	seenIDs := make(map[blocks.BlockID]bool, len(all))
	for i, name := range wantNames {
		got := all[i]
		if got.Name() != name {
			t.Fatalf("all[%d].Name() = %q, want %q", i, got.Name(), name)
		}
		if seenIDs[got.ID()] {
			t.Fatalf("duplicate ID %d in defaults", got.ID())
		}
		seenIDs[got.ID()] = true
		r, g, b := got.Color()
		if r < 0 || r > 1 || g < 0 || g > 1 || b < 0 || b > 1 {
			t.Fatalf("%s color out of range: (%v, %v, %v)", name, r, g, b)
		}
		if !got.Solid() {
			t.Fatalf("%s expected to be solid in defaults", name)
		}
	}
}

// TestAllReturnsCopy guards against accidental aliasing — mutating the slice
// returned by All must not leak back into the Model.
func TestAllReturnsCopy(t *testing.T) {
	var impl blocks.Blocks = blocks.Impl{}
	m, err := blocks.NewWithDefaults(impl)
	if err != nil {
		t.Fatalf("NewWithDefaults: %v", err)
	}
	all := impl.All(m)
	all[0] = blocks.NewBlock(999, "tampered", 0, 0, 0, false)

	first := impl.All(m)[0]
	if first.Name() != "grass" {
		t.Fatalf("registry leaked aliased slice: first.Name() = %q, want grass", first.Name())
	}
}

// TestColorClampedToUnitRange verifies that NewBlock clamps placeholder colors
// so renderers downstream can treat them as in-range without re-validating.
func TestColorClampedToUnitRange(t *testing.T) {
	b := blocks.NewBlock(1, "test", -0.5, 2.0, 0.5, true)
	r, g, bl := b.Color()
	if r != 0 || g != 1 || bl != 0.5 {
		t.Fatalf("Color() = (%v, %v, %v), want (0, 1, 0.5)", r, g, bl)
	}
}

// TestModAddsBlockWithoutEditingDefaults is the mod-extensibility acceptance
// check: a sixth block kind can be added through Register alone, without
// touching coreBlocks or any other existing function.
func TestModAddsBlockWithoutEditingDefaults(t *testing.T) {
	var impl blocks.Blocks = blocks.Impl{}
	m, err := blocks.NewWithDefaults(impl)
	if err != nil {
		t.Fatalf("NewWithDefaults: %v", err)
	}
	m, err = impl.Register(m, blocks.NewBlock(256, "redstone", 0.9, 0.1, 0.1, true))
	if err != nil {
		t.Fatalf("mod Register failed: %v", err)
	}
	if impl.Count(m) != 6 {
		t.Fatalf("Count after mod register = %d, want 6", impl.Count(m))
	}
	if b, ok := impl.LookupByName(m, "redstone"); !ok || b.ID() != 256 {
		t.Fatalf("LookupByName(redstone) = (%v, %v), want id=256/true", b, ok)
	}
}
