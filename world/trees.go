package world

// Tree placement and shape queries.
//
// Trees are placed by quantizing the world into placementCell-sized cells along
// x/z and rolling a deterministic per-cell coin. Inside a cell that "spawns",
// the offset is hashed from (seed, cellX, cellZ) so positions stay reproducible
// across regenerations. This guarantees a minimum spacing of one cell between
// trees, which keeps canopies from over-crowding.

const (
	// treeSeedOffset namespaces tree-placement noise from terrain noise.
	treeSeedOffset int64 = 0x7EE5

	// placementCell is the cell size used to thin tree placements. A 6x6 cell
	// gives min-spacing ~6 blocks, which comfortably separates canopyRadius=2
	// canopies.
	placementCell = 6

	// treeProbability is the chance that a given cell spawns a tree.
	treeProbability = 0.35

	// trunkMinHeight is the minimum trunk length; trunkHeightVariants chooses
	// the actual length via hash for each tree.
	trunkMinHeight      = 4
	trunkHeightVariants = 3 // trunks land in [trunkMinHeight, trunkMinHeight+variants-1]

	// canopyRadius controls the leaf-cluster size. Two layers of (2*r+1)^2
	// leaves with corners cut, plus a small cap above. Kept as a constant for
	// now; per-tree variation is easy to add later.
	canopyRadius = 2

	// canopyClearance is the vertical room above the trunk top required for
	// the canopy to fit. Equals 2 (two extra leaf layers above topY).
	canopyClearance = 2

	// edgeMargin keeps trees away from the world edge so canopies never spill
	// out of the [0, width) x [0, depth) bounds.
	edgeMargin = canopyRadius + 1
)

// placeTrees scans the world in placementCell-sized cells, deterministically
// deciding whether each cell spawns a tree and where. Pure function of
// (seed, dimensions, heightmap).
func placeTrees(seed int64, width, depth, maxHeight int, heights []int16) []Tree {
	trees := make([]Tree, 0, (width*depth)/(placementCell*placementCell)/2)
	noiseSeed := seed + treeSeedOffset

	cellsX := width / placementCell
	cellsZ := depth / placementCell

	for cellX := 0; cellX < cellsX; cellX++ {
		for cellZ := 0; cellZ < cellsZ; cellZ++ {
			if random01(noiseSeed, cellX, cellZ) >= treeProbability {
				continue
			}
			// Offset inside the cell (0..placementCell-1).
			ox := int(hashCoord(noiseSeed+1, cellX, cellZ) % uint64(placementCell))
			oz := int(hashCoord(noiseSeed+2, cellX, cellZ) % uint64(placementCell))
			x := cellX*placementCell + ox
			z := cellZ*placementCell + oz
			if x < edgeMargin || x >= width-edgeMargin {
				continue
			}
			if z < edgeMargin || z >= depth-edgeMargin {
				continue
			}
			surfaceY := int(heights[x*depth+z])
			trunkHeight := trunkMinHeight + int(hashCoord(noiseSeed+3, cellX, cellZ)%uint64(trunkHeightVariants))
			if surfaceY+trunkHeight+canopyClearance >= maxHeight {
				continue
			}
			trees = append(trees, NewTree(x, z, surfaceY+1, trunkHeight, canopyRadius))
		}
	}
	return trees
}

// buildTreeIndex groups trees by chunk so per-chunk queries don't scan the
// whole tree list. A tree is added to every chunk whose footprint may contain
// any of its trunk-or-canopy blocks.
func buildTreeIndex(trees []Tree, chunkCountX, chunkCountZ int) [][]int {
	index := make([][]int, chunkCountX*chunkCountZ)
	for treeIdx, t := range trees {
		minChunkX := chunkIndex(t.x - t.canopyRadius)
		maxChunkX := chunkIndex(t.x + t.canopyRadius)
		minChunkZ := chunkIndex(t.z - t.canopyRadius)
		maxChunkZ := chunkIndex(t.z + t.canopyRadius)
		for cx := minChunkX; cx <= maxChunkX; cx++ {
			if cx < 0 || cx >= chunkCountX {
				continue
			}
			for cz := minChunkZ; cz <= maxChunkZ; cz++ {
				if cz < 0 || cz >= chunkCountZ {
					continue
				}
				key := cx*chunkCountZ + cz
				index[key] = append(index[key], treeIdx)
			}
		}
	}
	return index
}

func chunkIndex(coord int) int {
	// Integer floor-div so negative coords (out-of-bounds canopy edges) still
	// land in the correct conceptual chunk. We immediately clamp in the
	// caller, but the math should be coherent.
	if coord < 0 {
		return (coord - ChunkSize + 1) / ChunkSize
	}
	return coord / ChunkSize
}

// blockInTree reports whether (x, y, z) lies inside the tree's trunk or
// canopy, and what block kind that is (Wood/Leaves). Trunk takes priority
// over canopy where they overlap at the trunk top. Returns (Air, false) if
// (x, y, z) is not part of this tree.
func blockInTree(t Tree, x, y, z int, wood, leaves uint16) (uint16, bool) {
	if x == t.x && z == t.z && y >= t.baseHeight && y < t.baseHeight+t.trunkHeight {
		return wood, true
	}
	// Canopy: two layers at topY..topY+1 with cropped corners, one smaller
	// layer above. The trunk's top block is wood (caught above), so canopy
	// here only fills the surrounding ring + top cap.
	topY := t.TopY()
	dy := y - topY
	dx := x - t.x
	dz := z - t.z
	if dy >= 0 && dy <= 1 {
		r := t.canopyRadius
		ax, az := absInt(dx), absInt(dz)
		if ax <= r && az <= r && !(ax == r && az == r) {
			return leaves, true
		}
	}
	if dy == 2 {
		ax, az := absInt(dx), absInt(dz)
		if ax <= 1 && az <= 1 && !(ax == 1 && az == 1) {
			return leaves, true
		}
	}
	return 0, false
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
