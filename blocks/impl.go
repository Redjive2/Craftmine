package blocks

import "fmt"

// Blocks is the behavior interface for the block registry module.
//
// World gen and any other consumer should depend on Blocks rather than on the
// concrete Impl, so a test double can substitute without dragging in the real
// registry.
type Blocks interface {
	Register(m Model, b Block) (Model, error)
	Lookup(m Model, id BlockID) (Block, bool)
	LookupByName(m Model, name string) (Block, bool)
	All(m Model) []Block
	Count(m Model) int
}

// Impl is the zero-field implementation of Blocks. All state lives in Model.
type Impl struct{}

var _ Blocks = Impl{}

// Register validates b and returns a new Model with b appended.
//
// The input Model is not mutated. Validation rejects empty names and any ID or
// name that collides with an already-registered block. Adding a new block kind
// (including from a mod) is just a call to Register — no existing code needs
// to change.
func (Impl) Register(m Model, b Block) (Model, error) {
	if b.name == "" {
		return m, fmt.Errorf("blocks: block name must not be empty (id=%d)", b.id)
	}
	for _, existing := range m.blocks {
		if existing.id == b.id {
			return m, fmt.Errorf("blocks: duplicate id %d (name %q vs %q)", b.id, existing.name, b.name)
		}
		if existing.name == b.name {
			return m, fmt.Errorf("blocks: duplicate name %q (id %d vs %d)", b.name, existing.id, b.id)
		}
	}
	next := make([]Block, len(m.blocks)+1)
	copy(next, m.blocks)
	next[len(m.blocks)] = b
	return Model{blocks: next}, nil
}

// Lookup returns the registered block with the given ID, or (zero, false) if
// no such block exists.
func (Impl) Lookup(m Model, id BlockID) (Block, bool) {
	for _, b := range m.blocks {
		if b.id == id {
			return b, true
		}
	}
	return Block{}, false
}

// LookupByName returns the registered block with the given name, or
// (zero, false) if no such block exists.
func (Impl) LookupByName(m Model, name string) (Block, bool) {
	for _, b := range m.blocks {
		if b.name == name {
			return b, true
		}
	}
	return Block{}, false
}

// All returns a copy of the registered blocks in registration order. The slice
// is a fresh allocation; mutating it does not affect the registry.
func (Impl) All(m Model) []Block {
	out := make([]Block, len(m.blocks))
	copy(out, m.blocks)
	return out
}

// Count reports how many blocks are registered.
func (Impl) Count(m Model) int { return m.Count() }
