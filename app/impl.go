package app

// App is the behavior interface for the application module.
//
// Callers depend on App, not on the concrete Impl, so the module is swappable
// (e.g. a test double can satisfy App without dragging in the real one).
type App interface {
	IsRunning(m Model) bool
	Stop(m Model) Model
}

// Impl is the zero-field implementation of App. All behavior hangs off Impl;
// state lives in Model and is passed as an argument to every method.
type Impl struct{}

// Compile-time check that Impl satisfies App.
var _ App = Impl{}

// IsRunning is a pure query over Model.
func (Impl) IsRunning(m Model) bool {
	return m.Running()
}

// Stop encodes the running -> stopped transition. It returns a new Model;
// the input is not mutated.
func (Impl) Stop(m Model) Model {
	return SetRunning(m, false)
}
