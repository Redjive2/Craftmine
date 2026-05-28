// Package save persists a single Craftmine world+player pair to disk and
// reads it back. It follows the Craftmine module pattern (Vision.md): all
// state lives in a Model with private fields and Field() accessors, and
// behavior lives on an Impl with no fields, reached through the Save
// interface.
//
// The on-disk format is a length-prefixed envelope:
//
//	uint32 format version | gob(envelopePayload)
//
// where envelopePayload contains the gob-encoded world and player blobs.
// Bumping saveFormatVersion is the single point of control for schema
// migrations; readers reject anything that does not match.
package save

import "time"

// Model holds the configuration of a save slot: where on disk it lives and
// when (if at all) we last wrote to it. Fields are private — callers reach
// them through accessors below.
type Model struct {
	path         string
	lastSaveTime time.Time
}

// New returns a Model that targets the given file path. The file does not
// need to exist yet; WriteWorld creates the parent directory on first save.
func New(path string) Model {
	return Model{path: path}
}

// Path returns the on-disk path of the save file.
func (m Model) Path() string { return m.path }

// LastSaveTime returns the wall-clock time of the most recent successful
// WriteWorld through this Model, or the zero time if none.
func (m Model) LastSaveTime() time.Time { return m.lastSaveTime }

// SetLastSaveTime returns a new Model with the last-save timestamp set to t.
// Callers should usually route through WriteWorld on Impl, which stamps
// this automatically.
func SetLastSaveTime(m Model, t time.Time) Model {
	m.lastSaveTime = t
	return m
}
