// Package blocks owns the registry of block kinds.
//
// It follows the Craftmine module pattern (see Vision.md): a Model holding all
// state (here, the registry of registered Block descriptors), an Impl carrying
// behavior with no fields, and a Blocks interface that Impl satisfies. Every
// behavioral function takes Model as an argument and returns a new Model when
// state changes — registration is the only state change supported, and it is
// expected to happen during initialization.
package blocks

// Model is the block registry.
//
// The slice field is private; callers reach the contents through Count / All /
// Lookup* methods on Impl. Registration returns a fresh Model rather than
// mutating in place, so a built Model is effectively immutable from the
// caller's perspective.
type Model struct {
	blocks []Block
}

// New returns an empty registry. Use Impl.Register to add blocks, or
// NewWithDefaults to obtain a registry pre-populated with the core block kinds.
func New() Model {
	return Model{}
}

// Count reports how many blocks are currently registered.
func (m Model) Count() int { return len(m.blocks) }
