// Package app holds the top-level application module.
//
// It is the canonical example of the Craftmine module pattern (see Vision.md):
// every module owns a Model in model.go with private fields and Field()
// accessors, an Impl struct in impl.go with no fields, and an interface that
// Impl satisfies. Behavior lives on Impl as functions that take Model as an
// argument and return a new Model; Model is never a receiver for behavior.
package app

// Model is the application state.
//
// All fields are private. Read them through the accessor methods below
// (Running, not GetRunning). The only methods allowed on Model are these
// trivial field accessors — every other function belongs on Impl and takes
// Model as an argument.
type Model struct {
	running bool
}

// New returns a Model with the application marked as running.
func New() Model {
	return Model{running: true}
}

// Running reports whether the application loop should continue.
func (m Model) Running() bool {
	return m.running
}

// SetRunning returns a new Model with the running flag set to running.
// Prefer named transitions (e.g. Stop) when they express a valid state change.
func SetRunning(m Model, running bool) Model {
	m.running = running
	return m
}
