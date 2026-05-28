// Package menu holds the main-menu module.
//
// The menu module follows the Craftmine module pattern (see Vision.md): all
// state lives in Model with private fields and Field() accessors; all
// behavior lives on Impl (no fields) and is reached through the Menu
// interface; Model is passed to functions as an argument, never as a
// receiver for behavior.
//
// Rendering wiring (g3n widgets, layout, event subscription) is not the
// menu module's job — that lives in main.go. The menu module is pure
// state plus the named transitions over it.
package menu

// Choice identifies which menu entry has been chosen by the user.
//
// ChoiceNone is the zero value and means "no selection yet". Callers
// driving the menu loop should treat ChoiceNone as "stay in the menu"
// and any other value as "transition out".
type Choice int

const (
	ChoiceNone Choice = iota
	ChoiceNewGame
	ChoiceResumeGame
)

// Item is one menu entry. Items are constructed once in New and never
// mutated afterwards — they are the immutable part of the menu Model.
type Item struct {
	label   string
	choice  Choice
	enabled bool
}

// Label returns the user-visible text for this item.
func (it Item) Label() string { return it.label }

// Choice returns the Choice this item represents.
func (it Item) Choice() Choice { return it.choice }

// Enabled reports whether this item can be selected. A disabled item
// must still render (greyed out) but Select on it is a no-op.
func (it Item) Enabled() bool { return it.enabled }

// Model is the menu state.
//
// items is immutable per Model (set in New). highlighted and selected
// are mutable but only through SetHighlighted/SetSelected or the named
// transitions on Impl that wrap them with validation.
type Model struct {
	items       []Item
	highlighted int
	selected    Choice
}

// New returns a Model wired with the canonical Craftmine menu: a New
// Game entry (always enabled) and a Resume Game entry whose enabled
// state is the value of resumeAvailable. The model starts with no
// highlight and no selection.
//
// Resume-availability is a constructor argument — not a global, not a
// cached lookup — so the enable state is a pure function of the Model.
// Callers are expected to compute resumeAvailable by asking the save
// module (save.Impl{}.Exists(...)) and rebuild the Model when that
// answer might have changed (startup, return-to-menu).
func New(resumeAvailable bool) Model {
	return Model{
		items: []Item{
			{label: "New Game", choice: ChoiceNewGame, enabled: true},
			{label: "Resume Game", choice: ChoiceResumeGame, enabled: resumeAvailable},
		},
		highlighted: -1,
		selected:    ChoiceNone,
	}
}

// Items returns a copy of the menu items. A copy — not the underlying
// slice — so callers cannot reach in and mutate Model state from
// outside.
func (m Model) Items() []Item {
	out := make([]Item, len(m.items))
	copy(out, m.items)
	return out
}

// ItemCount returns the number of menu entries.
func (m Model) ItemCount() int { return len(m.items) }

// Highlighted returns the index of the currently highlighted item, or
// -1 if no item is highlighted.
func (m Model) Highlighted() int { return m.highlighted }

// Selected returns the user's chosen Choice, or ChoiceNone if no
// choice has been made yet.
func (m Model) Selected() Choice { return m.selected }

// SetHighlighted returns a new Model with highlighted set to idx.
// Callers should usually prefer the Highlight transition on Impl,
// which rejects out-of-range indices.
func SetHighlighted(m Model, idx int) Model {
	m.highlighted = idx
	return m
}

// SetSelected returns a new Model with selected set to c. Callers
// should usually prefer the Select transition on Impl, which rejects
// disabled or unknown choices.
func SetSelected(m Model, c Choice) Model {
	m.selected = c
	return m
}
