package app_test

import (
	"testing"

	"github.com/redjive2/Craftmine/app"
)

// TestNewIsRunning checks that a fresh Model is in the running state.
func TestNewIsRunning(t *testing.T) {
	m := app.New()
	if !m.Running() {
		t.Fatalf("app.New().Running() = false, want true")
	}
}

// TestSetRunningRoundTrip exercises the generic SetField-style mutator
// across both directions and verifies the original Model is not mutated.
func TestSetRunningRoundTrip(t *testing.T) {
	cases := []struct {
		name string
		in   bool
		set  bool
		want bool
	}{
		{"running to stopped", true, false, false},
		{"stopped to running", false, true, true},
		{"running to running", true, true, true},
		{"stopped to stopped", false, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := app.SetRunning(app.New(), tc.in)
			after := app.SetRunning(before, tc.set)
			if after.Running() != tc.want {
				t.Fatalf("after.Running() = %v, want %v", after.Running(), tc.want)
			}
			if before.Running() != tc.in {
				t.Fatalf("input Model mutated: before.Running() = %v, want %v", before.Running(), tc.in)
			}
		})
	}
}

// TestImplStopTransition checks the named transition and confirms Impl
// implements the App interface at the call site (not just the var _ check).
func TestImplStopTransition(t *testing.T) {
	var application app.App = app.Impl{}
	state := app.New()

	if !application.IsRunning(state) {
		t.Fatalf("IsRunning(New()) = false, want true")
	}

	stopped := application.Stop(state)
	if application.IsRunning(stopped) {
		t.Fatalf("IsRunning(Stop(state)) = true, want false")
	}
	if !application.IsRunning(state) {
		t.Fatalf("Stop mutated its input: IsRunning(state) = false, want true")
	}
}
