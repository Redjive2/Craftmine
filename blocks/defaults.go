package blocks

// Core block IDs. Mods reserve IDs at 256 and above by convention; the core
// range is left wide so additional vanilla blocks can slot in without
// renumbering. Adding a new core block kind is two lines: a constant here and
// a NewBlock call in coreBlocks.
const (
	IDGrass  BlockID = 1
	IDDirt   BlockID = 2
	IDStone  BlockID = 3
	IDWood   BlockID = 4
	IDLeaves BlockID = 5
)

// coreBlocks lists the vanilla block kinds. The colors are placeholders chosen
// to be visually distinct in the demo viewer; replacing them with proper
// textures is a separate concern.
func coreBlocks() []Block {
	return []Block{
		NewBlock(IDGrass, "grass", 0.30, 0.65, 0.20, true),
		NewBlock(IDDirt, "dirt", 0.55, 0.35, 0.18, true),
		NewBlock(IDStone, "stone", 0.55, 0.55, 0.55, true),
		NewBlock(IDWood, "wood", 0.45, 0.27, 0.10, true),
		NewBlock(IDLeaves, "leaves", 0.20, 0.55, 0.18, true),
	}
}

// NewWithDefaults returns a Model with the five core block kinds registered.
//
// The Blocks argument is the interface (Impl in production), so test doubles
// can observe or rewrite the registration sequence. Errors surface registration
// failures (duplicate IDs, etc.) up front rather than silently dropping blocks.
func NewWithDefaults(impl Blocks) (Model, error) {
	m := New()
	for _, b := range coreBlocks() {
		next, err := impl.Register(m, b)
		if err != nil {
			return Model{}, err
		}
		m = next
	}
	return m, nil
}
