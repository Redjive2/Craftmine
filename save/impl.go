package save

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/redjive2/Craftmine/player"
	"github.com/redjive2/Craftmine/world"
)

// SaveFormatVersion is the on-disk schema version. Bump on any breaking
// change to the envelope or to a nested Model's serialization. Readers
// reject any blob whose leading version word doesn't match this value;
// the menu treats that as "no save exists" so an old or future save
// quietly falls back to the New Game path rather than crashing.
const SaveFormatVersion uint32 = 1

// Sentinel errors. ReadWorld returns one of these wrapped via fmt.Errorf
// so callers can use errors.Is for typed checks while still reading a
// descriptive message in logs.
var (
	// ErrMissingFile is returned by ReadWorld when the save path does not
	// exist. Distinguishing it from corrupt-file errors lets the menu
	// treat "no save yet" differently from "save broke; tell the user".
	ErrMissingFile = errors.New("save: file does not exist")

	// ErrIncompatibleVersion is returned when the leading version word
	// of a save file does not match SaveFormatVersion. The menu treats
	// this as "no usable save"; surfacing the typed error lets a future
	// migration step short-circuit version mismatches deliberately.
	ErrIncompatibleVersion = errors.New("save: incompatible format version")

	// ErrCorrupt is returned for any other read failure that suggests
	// the file is not a valid save (gob decode error, truncated body,
	// validation failure inside world/player Deserialize, etc.).
	ErrCorrupt = errors.New("save: corrupt save data")
)

// Save is the behavior interface for the save module. Callers depend on
// Save rather than the concrete Impl so a test double can be substituted
// (e.g. an in-memory implementation for tests that don't want to touch
// disk).
type Save interface {
	WriteWorld(m Model, w world.Model, p player.Model) (Model, error)
	ReadWorld(m Model) (world.Model, player.Model, error)
	Exists(m Model) bool
}

// Impl is the zero-field implementation of Save. All state lives in
// Model; all behavior takes Model as an argument.
type Impl struct{}

// Compile-time check that Impl satisfies Save.
var _ Save = Impl{}

// envelopePayload is the gob-encoded body that follows the version word.
// Keeping the version word outside the gob blob lets readers cheaply
// reject incompatible saves without paying for a full decode.
type envelopePayload struct {
	World  []byte
	Player []byte
}

// WriteWorld atomically writes (w, p) to m's path. Tempfile-then-rename
// keeps a half-written save from clobbering a known-good one if the
// process dies mid-write. Returns a new Model with the last-save
// timestamp updated.
func (Impl) WriteWorld(m Model, w world.Model, p player.Model) (Model, error) {
	if m.path == "" {
		return m, fmt.Errorf("save: path is empty")
	}
	worldBytes, err := w.Serialize()
	if err != nil {
		return m, fmt.Errorf("save: serialize world: %w", err)
	}
	playerBytes, err := p.Serialize()
	if err != nil {
		return m, fmt.Errorf("save: serialize player: %w", err)
	}

	var body bytes.Buffer
	if err := gob.NewEncoder(&body).Encode(envelopePayload{
		World:  worldBytes,
		Player: playerBytes,
	}); err != nil {
		return m, fmt.Errorf("save: encode envelope: %w", err)
	}

	dir := filepath.Dir(m.path)
	if dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return m, fmt.Errorf("save: create dir %q: %w", dir, err)
		}
	}

	tmp, err := os.CreateTemp(dir, "save-*.tmp")
	if err != nil {
		return m, fmt.Errorf("save: create temp: %w", err)
	}
	tmpPath := tmp.Name()
	// On any failure below, drop the half-written tempfile rather than
	// leaving litter next to the real save.
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}

	if err := binary.Write(tmp, binary.BigEndian, SaveFormatVersion); err != nil {
		cleanup()
		return m, fmt.Errorf("save: write version: %w", err)
	}
	if _, err := tmp.Write(body.Bytes()); err != nil {
		cleanup()
		return m, fmt.Errorf("save: write body: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		cleanup()
		return m, fmt.Errorf("save: sync temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return m, fmt.Errorf("save: close temp: %w", err)
	}
	if err := os.Rename(tmpPath, m.path); err != nil {
		_ = os.Remove(tmpPath)
		return m, fmt.Errorf("save: rename %q -> %q: %w", tmpPath, m.path, err)
	}
	return SetLastSaveTime(m, time.Now()), nil
}

// ReadWorld decodes the file at m's path and returns the world and player
// Models contained in it. Returns one of the typed sentinel errors
// (wrapped) on every failure path so callers can distinguish missing /
// incompatible / corrupt from one another.
func (Impl) ReadWorld(m Model) (world.Model, player.Model, error) {
	if m.path == "" {
		return world.Model{}, player.Model{}, fmt.Errorf("%w: empty path", ErrMissingFile)
	}
	data, err := os.ReadFile(m.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return world.Model{}, player.Model{}, fmt.Errorf("%w: %s", ErrMissingFile, m.path)
		}
		return world.Model{}, player.Model{}, fmt.Errorf("save: read %q: %w", m.path, err)
	}
	if len(data) < 4 {
		return world.Model{}, player.Model{}, fmt.Errorf("%w: file %q is %d bytes (too small for version word)",
			ErrCorrupt, m.path, len(data))
	}
	version := binary.BigEndian.Uint32(data[:4])
	if version != SaveFormatVersion {
		return world.Model{}, player.Model{}, fmt.Errorf("%w: file %q has version %d, want %d",
			ErrIncompatibleVersion, m.path, version, SaveFormatVersion)
	}

	var payload envelopePayload
	if err := gob.NewDecoder(bytes.NewReader(data[4:])).Decode(&payload); err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
			return world.Model{}, player.Model{}, fmt.Errorf("%w: truncated body in %q: %v",
				ErrCorrupt, m.path, err)
		}
		return world.Model{}, player.Model{}, fmt.Errorf("%w: decode envelope %q: %v",
			ErrCorrupt, m.path, err)
	}
	w, err := world.Deserialize(payload.World)
	if err != nil {
		return world.Model{}, player.Model{}, fmt.Errorf("%w: world body %q: %v",
			ErrCorrupt, m.path, err)
	}
	p, err := player.Deserialize(payload.Player)
	if err != nil {
		return world.Model{}, player.Model{}, fmt.Errorf("%w: player body %q: %v",
			ErrCorrupt, m.path, err)
	}
	return w, p, nil
}

// Exists reports whether a regular file exists at m's path. False for any
// non-regular entry (directory, symlink that doesn't resolve, etc.) so a
// stray directory at the save path doesn't trick the menu into enabling
// Resume Game.
func (Impl) Exists(m Model) bool {
	if m.path == "" {
		return false
	}
	info, err := os.Stat(m.path)
	if err != nil {
		return false
	}
	return info.Mode().IsRegular()
}

// DefaultPath returns the canonical save path under the user's home
// directory: ~/.craftmine/save.gob. Returns an error only if the home
// directory cannot be resolved, in which case callers should fall back to
// the current working directory or refuse to start the save module.
//
// The directory is not created here — WriteWorld does that on first save
// so a read-only "is there a save?" probe doesn't leave directories
// behind on disk.
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("save: resolve home dir: %w", err)
	}
	return filepath.Join(home, ".craftmine", "save.gob"), nil
}
