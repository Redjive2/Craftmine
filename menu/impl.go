package menu

// Menu is the behavior interface for the menu module.
//
// Callers depend on Menu rather than the concrete Impl so the module
// can be swapped in tests or replaced wholesale by a different
// implementation later.
type Menu interface {
	Items(m Model) []Item
	ItemCount(m Model) int
	Highlighted(m Model) int
	Selected(m Model) Choice
	IsDone(m Model) bool
	Highlight(m Model, idx int) Model
	Select(m Model, c Choice) Model
}

// Impl is the zero-field implementation of Menu. All behavior hangs
// off Impl; state lives in Model and is passed in as an argument.
type Impl struct{}

// Compile-time check that Impl satisfies Menu.
var _ Menu = Impl{}

// Items returns the menu entries for m (as a copy).
func (Impl) Items(m Model) []Item { return m.Items() }

// ItemCount returns the number of menu entries for m.
func (Impl) ItemCount(m Model) int { return m.ItemCount() }

// Highlighted returns the highlighted index of m, or -1 for none.
func (Impl) Highlighted(m Model) int { return m.Highlighted() }

// Selected returns the chosen Choice of m, or ChoiceNone for none.
func (Impl) Selected(m Model) Choice { return m.Selected() }

// IsDone reports whether the menu has reached a terminal state — i.e.
// the user has selected an enabled entry. The render loop should stop
// drawing the menu once IsDone returns true and act on Selected.
func (Impl) IsDone(m Model) bool { return m.Selected() != ChoiceNone }

// Highlight returns a new Model with highlighted set to idx. idx must
// be -1 (clear the highlight) or a valid index into the items slice.
// Out-of-range values leave m unchanged so a stray hover event cannot
// corrupt state.
func (Impl) Highlight(m Model, idx int) Model {
	if idx == -1 {
		return SetHighlighted(m, -1)
	}
	if idx < 0 || idx >= m.ItemCount() {
		return m
	}
	return SetHighlighted(m, idx)
}

// Select returns a new Model with selected set to c, provided c maps
// to an enabled item in m. Selecting a disabled or unknown Choice is
// a no-op — the UI is responsible for not even firing the click, but
// this guard is the single source of truth.
func (Impl) Select(m Model, c Choice) Model {
	for _, it := range m.Items() {
		if it.Choice() == c && it.Enabled() {
			return SetSelected(m, c)
		}
	}
	return m
}
