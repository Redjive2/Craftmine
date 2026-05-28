package save_test

import (
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/redjive2/Craftmine/blocks"
	"github.com/redjive2/Craftmine/player"
	"github.com/redjive2/Craftmine/save"
	"github.com/redjive2/Craftmine/world"
)

// testWorld builds a small but realistic world.Model for round-tripping.
// Smaller than the menu's New Game default but big enough to exercise the
// heightmap, tree list, and per-chunk tree index.
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

func testPlayer() player.Model {
	p := player.New(player.NewVec3(10.5, 12, 8.25))
	p = player.SetLook(p, player.NewLook(1.2, -0.3))
	p = player.SetVelocity(p, player.NewVec3(0.5, -1.0, 0.25))
	p = player.SetOnGround(p, true)
	return p
}

func tempSaveModel(t *testing.T) save.Model {
	t.Helper()
	dir := t.TempDir()
	return save.New(filepath.Join(dir, "nested", "save.gob"))
}

// TestRoundTrip is the headline guarantee: writing a world+player and
// reading them back yields Models whose accessor outputs agree with the
// originals.
func TestRoundTrip(t *testing.T) {
	var impl save.Save = save.Impl{}
	m := tempSaveModel(t)
	w := testWorld(t)
	p := testPlayer()

	updated, err := impl.WriteWorld(m, w, p)
	if err != nil {
		t.Fatalf("WriteWorld: %v", err)
	}
	if updated.LastSaveTime().IsZero() {
		t.Fatalf("WriteWorld did not stamp LastSaveTime")
	}
	if !impl.Exists(updated) {
		t.Fatalf("Exists = false after WriteWorld; expected true")
	}

	gotWorld, gotPlayer, err := impl.ReadWorld(updated)
	if err != nil {
		t.Fatalf("ReadWorld: %v", err)
	}

	if gotWorld.Seed() != w.Seed() ||
		gotWorld.Width() != w.Width() ||
		gotWorld.Depth() != w.Depth() ||
		gotWorld.MaxHeight() != w.MaxHeight() ||
		gotWorld.DirtDepth() != w.DirtDepth() ||
		gotWorld.TreeCount() != w.TreeCount() {
		t.Fatalf("world Model fields diverged after round-trip: got %+v want %+v",
			summarizeWorld(gotWorld), summarizeWorld(w))
	}
	if gotWorld.Grass() != w.Grass() || gotWorld.Dirt() != w.Dirt() ||
		gotWorld.Stone() != w.Stone() || gotWorld.Wood() != w.Wood() ||
		gotWorld.Leaves() != w.Leaves() {
		t.Fatalf("cached block IDs diverged after round-trip")
	}
	for x := 0; x < w.Width(); x++ {
		for z := 0; z < w.Depth(); z++ {
			if gotWorld.HeightAt(x, z) != w.HeightAt(x, z) {
				t.Fatalf("HeightAt(%d,%d) = %d, want %d",
					x, z, gotWorld.HeightAt(x, z), w.HeightAt(x, z))
			}
		}
	}
	origTrees := w.Trees()
	gotTrees := gotWorld.Trees()
	for i := range origTrees {
		if gotTrees[i].X() != origTrees[i].X() ||
			gotTrees[i].Z() != origTrees[i].Z() ||
			gotTrees[i].BaseHeight() != origTrees[i].BaseHeight() ||
			gotTrees[i].TrunkHeight() != origTrees[i].TrunkHeight() ||
			gotTrees[i].CanopyRadius() != origTrees[i].CanopyRadius() {
			t.Fatalf("tree[%d] diverged: %+v vs %+v", i, gotTrees[i], origTrees[i])
		}
	}
	// Per-chunk index must still resolve every tree — it's rebuilt on
	// load, so a bug in rebuild would silently drop trees from queries.
	var worldImpl world.World = world.Impl{}
	indexed := 0
	for cx := 0; cx < gotWorld.ChunkCountX(); cx++ {
		for cz := 0; cz < gotWorld.ChunkCountZ(); cz++ {
			indexed += len(worldImpl.TreesInChunk(gotWorld, cx, cz))
		}
	}
	if indexed < gotWorld.TreeCount() {
		t.Fatalf("per-chunk index covers %d entries but %d trees exist", indexed, gotWorld.TreeCount())
	}

	if gotPlayer.Position() != p.Position() {
		t.Fatalf("player Position = %+v, want %+v", gotPlayer.Position(), p.Position())
	}
	if gotPlayer.Look().Yaw() != p.Look().Yaw() || gotPlayer.Look().Pitch() != p.Look().Pitch() {
		t.Fatalf("player Look diverged: %+v vs %+v", gotPlayer.Look(), p.Look())
	}
	if gotPlayer.Velocity() != p.Velocity() {
		t.Fatalf("player Velocity = %+v, want %+v", gotPlayer.Velocity(), p.Velocity())
	}
	if gotPlayer.OnGround() != p.OnGround() {
		t.Fatalf("player OnGround = %v, want %v", gotPlayer.OnGround(), p.OnGround())
	}
	if gotPlayer.EyeHeight() != p.EyeHeight() ||
		gotPlayer.HitboxWidth() != p.HitboxWidth() ||
		gotPlayer.HitboxHeight() != p.HitboxHeight() {
		t.Fatalf("player dimensions diverged after round-trip")
	}
}

// TestExistsAndReadMissing covers the "fresh install" path: no file on
// disk, so Exists is false and ReadWorld returns ErrMissingFile.
func TestExistsAndReadMissing(t *testing.T) {
	var impl save.Save = save.Impl{}
	m := tempSaveModel(t)

	if impl.Exists(m) {
		t.Fatalf("Exists = true with no file at %q", m.Path())
	}
	_, _, err := impl.ReadWorld(m)
	if !errors.Is(err, save.ErrMissingFile) {
		t.Fatalf("ReadWorld on missing file: err = %v, want ErrMissingFile", err)
	}
}

// TestExistsRejectsNonRegular guards the menu against a directory at the
// save path enabling Resume Game.
func TestExistsRejectsNonRegular(t *testing.T) {
	var impl save.Save = save.Impl{}
	dir := t.TempDir()
	// Make the "save file" be a directory.
	path := filepath.Join(dir, "save.gob")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	m := save.New(path)
	if impl.Exists(m) {
		t.Fatalf("Exists = true for a directory at the save path")
	}
}

// TestCorruptFile flips bytes in a valid save and verifies ReadWorld
// surfaces ErrCorrupt rather than returning a half-built Model.
func TestCorruptFile(t *testing.T) {
	var impl save.Save = save.Impl{}
	m := tempSaveModel(t)
	w := testWorld(t)
	p := testPlayer()
	updated, err := impl.WriteWorld(m, w, p)
	if err != nil {
		t.Fatalf("WriteWorld: %v", err)
	}

	data, err := os.ReadFile(updated.Path())
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(data) < 64 {
		t.Fatalf("save file unexpectedly small (%d bytes)", len(data))
	}
	// Mangle a chunk past the version word so the version check still
	// passes but gob decode (or the inner Deserialize) chokes.
	for i := 32; i < 64; i++ {
		data[i] ^= 0xFF
	}
	if err := os.WriteFile(updated.Path(), data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, _, err = impl.ReadWorld(updated)
	if !errors.Is(err, save.ErrCorrupt) {
		t.Fatalf("ReadWorld on corrupt file: err = %v, want ErrCorrupt", err)
	}
}

// TestTruncatedFile checks that a file too small to even hold the
// version word is reported as corrupt, not as missing.
func TestTruncatedFile(t *testing.T) {
	var impl save.Save = save.Impl{}
	dir := t.TempDir()
	path := filepath.Join(dir, "save.gob")
	if err := os.WriteFile(path, []byte{0x00}, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	m := save.New(path)
	_, _, err := impl.ReadWorld(m)
	if !errors.Is(err, save.ErrCorrupt) {
		t.Fatalf("ReadWorld on 1-byte file: err = %v, want ErrCorrupt", err)
	}
}

// TestVersionMismatch hand-rolls a save with a future version word and
// verifies the read fails fast with ErrIncompatibleVersion.
func TestVersionMismatch(t *testing.T) {
	var impl save.Save = save.Impl{}
	dir := t.TempDir()
	path := filepath.Join(dir, "save.gob")

	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, 999)
	// Append some plausible-looking garbage past the version word so the
	// failure is unambiguously the version check, not a truncated body.
	buf = append(buf, []byte("not-a-real-gob-blob")...)
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	m := save.New(path)
	_, _, err := impl.ReadWorld(m)
	if !errors.Is(err, save.ErrIncompatibleVersion) {
		t.Fatalf("ReadWorld on version=999 file: err = %v, want ErrIncompatibleVersion", err)
	}
}

// TestWriteIsAtomic verifies the tempfile-then-rename strategy by
// confirming no stray *.tmp files survive a successful write.
func TestWriteIsAtomic(t *testing.T) {
	var impl save.Save = save.Impl{}
	m := tempSaveModel(t)
	w := testWorld(t)
	p := testPlayer()
	updated, err := impl.WriteWorld(m, w, p)
	if err != nil {
		t.Fatalf("WriteWorld: %v", err)
	}
	entries, err := os.ReadDir(filepath.Dir(updated.Path()))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Fatalf("stray tempfile after WriteWorld: %s", e.Name())
		}
	}
}

// TestIntegrationWalkSaveLoad is the ticket's headline integration test:
// spawn the player, walk a few frames, save, "quit," reload, assert the
// position and look match what we saved.
func TestIntegrationWalkSaveLoad(t *testing.T) {
	var impl save.Save = save.Impl{}
	var playerImpl player.Player = player.Impl{}
	m := tempSaveModel(t)
	w := testWorld(t)

	// Spawn in the middle of the world, on the surface, looking at +Z.
	spawnX := float64(w.Width()) / 2
	spawnZ := float64(w.Depth()) / 2
	surface := w.HeightAt(int(spawnX), int(spawnZ))
	spawn := player.NewVec3(spawnX, float64(surface+1), spawnZ)
	p, err := playerImpl.SetPosition(player.New(spawn), spawn, w)
	if err != nil {
		t.Fatalf("SetPosition: %v", err)
	}
	p = player.SetLook(p, player.NewLook(0.75, -0.15))

	// Walk forward a few frames.
	const dt = 1.0 / 60
	for i := 0; i < 30; i++ {
		p = playerImpl.Tick(p, player.Input{Forward: true}, w, dt)
	}
	savedPos := p.Position()
	savedLook := p.Look()

	if _, err := impl.WriteWorld(m, w, p); err != nil {
		t.Fatalf("WriteWorld: %v", err)
	}

	// "Restart": fresh Model, reload from disk.
	reloaded := save.New(m.Path())
	gotWorld, gotPlayer, err := impl.ReadWorld(reloaded)
	if err != nil {
		t.Fatalf("ReadWorld: %v", err)
	}
	if gotPlayer.Position() != savedPos {
		t.Fatalf("Position after reload = %+v, want %+v", gotPlayer.Position(), savedPos)
	}
	if gotPlayer.Look().Yaw() != savedLook.Yaw() || gotPlayer.Look().Pitch() != savedLook.Pitch() {
		t.Fatalf("Look after reload = %+v, want %+v", gotPlayer.Look(), savedLook)
	}
	// And the reloaded world should still respond to player physics
	// without crashing — exercise one tick.
	_ = playerImpl.Tick(gotPlayer, player.Input{}, gotWorld, dt)
}

type worldSummary struct {
	Seed                                int64
	Width, Depth, MaxHeight, TreeCount  int
	DirtDepth                           int
	Grass, Dirt, Stone, Wood, Leaves    blocks.BlockID
}

func summarizeWorld(m world.Model) worldSummary {
	return worldSummary{
		Seed:      m.Seed(),
		Width:     m.Width(),
		Depth:     m.Depth(),
		MaxHeight: m.MaxHeight(),
		TreeCount: m.TreeCount(),
		DirtDepth: m.DirtDepth(),
		Grass:     m.Grass(),
		Dirt:      m.Dirt(),
		Stone:     m.Stone(),
		Wood:      m.Wood(),
		Leaves:    m.Leaves(),
	}
}
