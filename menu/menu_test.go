package menu_test

import (
	"testing"

	"github.com/redjive2/Craftmine/menu"
)

// TestNewHasCanonicalEntries verifies New seeds the model with exactly
// the two items the vision calls for: New Game (enabled) and Resume
// Game (disabled, pending save/load).
func TestNewHasCanonicalEntries(t *testing.T) {
	m := menu.New()
	items := m.Items()
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].Label() != "New Game" || items[0].Choice() != menu.ChoiceNewGame || !items[0].Enabled() {
		t.Fatalf("items[0] = {label=%q, choice=%v, enabled=%v}; want New Game/ChoiceNewGame/enabled",
			items[0].Label(), items[0].Choice(), items[0].Enabled())
	}
	if items[1].Label() != "Resume Game" || items[1].Choice() != menu.ChoiceResumeGame || items[1].Enabled() {
		t.Fatalf("items[1] = {label=%q, choice=%v, enabled=%v}; want Resume Game/ChoiceResumeGame/disabled",
			items[1].Label(), items[1].Choice(), items[1].Enabled())
	}
}

// TestNewStartsUnselectedAndUnhighlighted verifies the initial Model
// is in the "menu just opened, user has not interacted" state.
func TestNewStartsUnselectedAndUnhighlighted(t *testing.T) {
	m := menu.New()
	if m.Selected() != menu.ChoiceNone {
		t.Fatalf("Selected() = %v, want ChoiceNone", m.Selected())
	}
	if m.Highlighted() != -1 {
		t.Fatalf("Highlighted() = %d, want -1", m.Highlighted())
	}
	var impl menu.Menu = menu.Impl{}
	if impl.IsDone(m) {
		t.Fatalf("IsDone(New()) = true, want false")
	}
}

// TestHighlight exercises every branch of Impl.Highlight in a single
// batch — accepted indices, the clear-highlight value -1, and the
// rejection paths for out-of-range indices.
func TestHighlight(t *testing.T) {
	impl := menu.Impl{}
	base := menu.New()
	// Pre-seed a known highlighted=0 so the rejection cases have
	// something specific to be "unchanged from".
	base = impl.Highlight(base, 0)

	cases := []struct {
		name    string
		idx     int
		wantHi  int
		wantKey string // "set" or "unchanged"
	}{
		{"set to first item", 0, 0, "set"},
		{"set to second item", 1, 1, "set"},
		{"clear highlight", -1, -1, "set"},
		{"out-of-range high rejected", 99, 0, "unchanged"},
		{"out-of-range low rejected", -2, 0, "unchanged"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			after := impl.Highlight(base, tc.idx)
			if after.Highlighted() != tc.wantHi {
				t.Fatalf("Highlighted() = %d, want %d (%s)", after.Highlighted(), tc.wantHi, tc.wantKey)
			}
			if base.Highlighted() != 0 {
				t.Fatalf("Highlight mutated its input: base.Highlighted() = %d, want 0", base.Highlighted())
			}
		})
	}
}

// TestSelectEnabledIsAccepted verifies Select on an enabled choice
// records the selection and flips IsDone, without mutating the input.
func TestSelectEnabledIsAccepted(t *testing.T) {
	impl := menu.Impl{}
	m := menu.New()
	after := impl.Select(m, menu.ChoiceNewGame)
	if after.Selected() != menu.ChoiceNewGame {
		t.Fatalf("Selected() = %v, want ChoiceNewGame", after.Selected())
	}
	if !impl.IsDone(after) {
		t.Fatalf("IsDone(after) = false, want true once a choice is made")
	}
	if impl.IsDone(m) {
		t.Fatalf("Select mutated its input: IsDone(m) = true, want false")
	}
}

// TestSelectDisabledIsRejected is the load-bearing test for the
// "Resume Game stays disabled until save/load lands" guarantee. If
// Select ever lets ChoiceResumeGame through, the menu will hand a
// half-implemented resume path to the rest of the game.
func TestSelectDisabledIsRejected(t *testing.T) {
	impl := menu.Impl{}
	m := menu.New()
	after := impl.Select(m, menu.ChoiceResumeGame)
	if after.Selected() != menu.ChoiceNone {
		t.Fatalf("Selected() = %v, want ChoiceNone — Resume Game is disabled", after.Selected())
	}
	if impl.IsDone(after) {
		t.Fatalf("IsDone(after) = true, want false — disabled item must not advance the menu")
	}
}

// TestSelectUnknownChoiceIsRejected guards against future Choice
// constants slipping through Select without an items entry.
func TestSelectUnknownChoiceIsRejected(t *testing.T) {
	impl := menu.Impl{}
	m := menu.New()
	after := impl.Select(m, menu.Choice(999))
	if after.Selected() != menu.ChoiceNone {
		t.Fatalf("Selected() = %v, want ChoiceNone for unknown Choice(999)", after.Selected())
	}
}

// TestItemsIsACopy verifies Model.Items returns an isolated slice so
// callers cannot reach in and mutate the immutable items.
func TestItemsIsACopy(t *testing.T) {
	m := menu.New()
	items := m.Items()
	items[0] = menu.Item{}
	again := m.Items()
	if again[0].Choice() != menu.ChoiceNewGame || !again[0].Enabled() {
		t.Fatalf("Items() shared its backing slice — mutating the returned slice changed the Model")
	}
}

// TestImplSatisfiesMenuInterface is a compile-time-style assertion at
// the call site, matching the pattern in app_test.go.
func TestImplSatisfiesMenuInterface(t *testing.T) {
	var _ menu.Menu = menu.Impl{}
}
