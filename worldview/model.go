// Package worldview owns the g3n rendering of a world.Model.
//
// Per Vision.md this module follows the Craftmine module pattern: a Model
// holding all rendering state (cached materials, the shared cube geometry,
// and the assembled scene-node groups), an Impl carrying behavior with no
// fields, and a View interface that Impl satisfies. Build is pure given
// (world model, registry, world impl) — the resulting Model holds pointers
// into the g3n scene graph because the engine requires them, but those
// pointers are treated as read-only by the rest of the codebase.
//
// The rendering strategy matches cmd/world-demo: one grass cube per column
// surface, plus per-block cubes for every tree's trunk and canopy. Sharing
// the cube geometry and per-kind materials across every mesh keeps GPU
// memory bounded even at the Vision.md 512x512 target.
package worldview

import (
	"github.com/redjive2/Craftmine/blocks"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/material"
)

// Model holds the assembled world-rendering state.
//
// All fields are private; read them through the accessor methods. The
// returned nodes are owned by the Model — callers attach them to a scene
// exactly once and should not mutate them further.
type Model struct {
	materials map[blocks.BlockID]*material.Standard
	cube      *geometry.Geometry
	surfaces  *core.Node
	trees     *core.Node
}

// Surfaces returns the node containing one grass cube per world column.
func (m Model) Surfaces() *core.Node { return m.surfaces }

// Trees returns the node containing the wood and leaf cubes of every tree.
func (m Model) Trees() *core.Node { return m.trees }

// Cube returns the shared cube geometry used by every mesh in the model.
// Exposed mostly so tests can confirm mesh-geometry reuse.
func (m Model) Cube() *geometry.Geometry { return m.cube }

// Material returns the standard material registered for blockID, or nil
// if blockID is not in the registry the Model was built from.
func (m Model) Material(id blocks.BlockID) *material.Standard {
	return m.materials[id]
}
