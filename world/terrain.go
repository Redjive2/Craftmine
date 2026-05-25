package world

// terrainSeedOffset namespaces the noise used for the heightmap so it can be
// varied independently from tree placement and any future generators sharing
// the same seed.
const terrainSeedOffset int64 = 0xA17E7

// generateHeights returns a width*depth heightmap, row-major as [x*depth + z].
//
// The map is a single fractal-noise field scaled into a y range that always
// leaves room for a tree canopy above the tallest surface and keeps the
// stone floor above zero. baseHeight is the lowest surface y; amplitude is
// the peak-to-trough range.
//
// Pure function of (seed, dimensions): same inputs -> same heightmap.
func generateHeights(seed int64, width, depth, maxHeight int) []int16 {
	heights := make([]int16, width*depth)

	// Reserve room above for tree canopy (~6 blocks) and below for stone
	// (DefaultDirtDepth + a few). Amplitude is bounded so the surface stays
	// inside [baseHeight, baseHeight + amplitude].
	baseHeight := maxHeight / 4
	amplitude := maxHeight / 3
	if baseHeight < DefaultDirtDepth+1 {
		baseHeight = DefaultDirtDepth + 1
	}
	if baseHeight+amplitude > maxHeight-8 {
		amplitude = maxHeight - 8 - baseHeight
		if amplitude < 1 {
			amplitude = 1
		}
	}

	noiseSeed := seed + terrainSeedOffset
	for x := 0; x < width; x++ {
		for z := 0; z < depth; z++ {
			// Four octaves give gentle rolling hills without sharp ridges:
			// base frequency 1/64 means features ~64 blocks wide, with finer
			// detail layered on top.
			n := fbm2D(noiseSeed, float64(x), float64(z), 4, 1.0/64.0, 2.0, 0.5)
			y := baseHeight + int(n*float64(amplitude))
			if y < 1 {
				y = 1
			}
			if y >= maxHeight {
				y = maxHeight - 1
			}
			heights[x*depth+z] = int16(y)
		}
	}
	return heights
}
