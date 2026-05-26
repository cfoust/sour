package preview

import "container/heap"

type voxelKey struct {
	x, y, z int
	size    int
}

// SimplifyVoxels reduces a voxel set to at most targetCount voxels using
// error-driven octree collapse, inspired by Garland & Heckbert (1997),
// "Surface Simplification Using Quadric Error Metrics."
//
// The collapse error for merging 8 sibling voxels into one parent is:
//
//	error = emptyCells × parentSize² × proximityWeight
//
// This captures three key principles:
//   - emptyCells: solid blocks (0 empty) merge for free; thin features (7 empty) are expensive
//   - parentSize²: large blocks have proportionally more visual impact (screen-space area)
//   - proximityWeight: play-area geometry is 10× more expensive to simplify
//
// The result: small, distant, solid merges happen first. Large, near, thin
// features are preserved until the very end.
func SimplifyVoxels(voxels []Voxel, targetCount int, playArea *PlayAreaGrid) []Voxel {
	if len(voxels) <= targetCount {
		return voxels
	}

	// Maximum size of a merged block. Prevents cascading merges from
	// creating absurdly large cubes that dominate the scene.
	const maxMergeSize = 32

	proximityWeight := func(x, y, z, size int) int {
		if playArea == nil {
			return 1
		}
		cx, cy, cz := x+size/2, y+size/2, z+size/2
		if playArea.Get(cx, cy, cz) {
			return 10
		}
		for _, corner := range [8][3]int{
			{x, y, z}, {x + size, y, z}, {x, y + size, z}, {x + size, y + size, z},
			{x, y, z + size}, {x + size, y, z + size}, {x, y + size, z + size}, {x + size, y + size, z + size},
		} {
			if playArea.Get(corner[0], corner[1], corner[2]) {
				return 10
			}
		}
		return 1
	}

	collapseError := func(emptyCount, parentSize, weight int) int {
		return emptyCount * parentSize * parentSize * weight
	}

	// Build the live voxel set.
	live := make(map[voxelKey]*Voxel, len(voxels))
	for i := range voxels {
		v := &voxels[i]
		k := voxelKey{int(v.X), int(v.Y), int(v.Z), v.Size()}
		live[k] = v
	}

	count := len(live)
	if count <= targetCount {
		return voxels
	}

	// Compute dominant palette index and flags for a group of children.
	groupPalette := func(keys []voxelKey) (uint8, byte) {
		palCounts := make(map[uint8]int)
		var bestFlags byte
		for _, ck := range keys {
			v := live[ck]
			palCounts[v.PaletteIndex]++
			bestFlags = v.Flags
		}
		bestPal := uint8(0)
		bestCount := 0
		for p, c := range palCounts {
			if c > bestCount {
				bestCount = c
				bestPal = p
			}
		}
		return bestPal, bestFlags
	}

	// Build a collapse candidate for a parent position with given children.
	makeCandidate := func(px, py, pz, parentSize int, children []voxelKey) *collapseCandidate {
		if len(children) < 2 || parentSize > maxMergeSize {
			return nil
		}
		// All children must be the same size
		childSize := children[0].size
		for _, c := range children {
			if c.size != childSize {
				return nil
			}
		}

		emptyCount := 8 - len(children)
		weight := proximityWeight(px, py, pz, parentSize)
		palIdx, flags := groupPalette(children)

		return &collapseCandidate{
			parentX:    px,
			parentY:    py,
			parentZ:    pz,
			parentSize: parentSize,
			childKeys:  children,
			error:      collapseError(emptyCount, parentSize, weight),
			saved:      len(children) - 1,
			palIdx:     palIdx,
			flags:      flags,
		}
	}

	// Build initial collapse candidates.
	type parentKey struct {
		x, y, z int
		size    int
	}
	groups := make(map[parentKey][]voxelKey)
	for k := range live {
		parentSize := k.size * 2
		if parentSize > maxMergeSize {
			continue
		}
		mask := ^(parentSize - 1)
		pk := parentKey{k.x & mask, k.y & mask, k.z & mask, parentSize}
		groups[pk] = append(groups[pk], k)
	}

	pq := &collapseHeap{}
	heap.Init(pq)
	for pk, children := range groups {
		cand := makeCandidate(pk.x, pk.y, pk.z, pk.size, children)
		if cand != nil {
			heap.Push(pq, cand)
		}
	}

	for count > targetCount && pq.Len() > 0 {
		cand := heap.Pop(pq).(*collapseCandidate)

		// Verify all children still exist
		valid := true
		for _, ck := range cand.childKeys {
			if _, ok := live[ck]; !ok {
				valid = false
				break
			}
		}
		if !valid {
			continue
		}

		// Perform collapse: remove children, add parent
		for _, ck := range cand.childKeys {
			delete(live, ck)
		}

		sizeLog2 := 0
		for s := cand.parentSize; s > 1; s >>= 1 {
			sizeLog2++
		}

		parentVoxel := &Voxel{
			X:            uint16(cand.parentX),
			Y:            uint16(cand.parentY),
			Z:            uint16(cand.parentZ),
			PaletteIndex: cand.palIdx,
			Flags:        (cand.flags & 0x1F) | byte(sizeLog2<<FlagSizeShift),
		}

		pk := voxelKey{cand.parentX, cand.parentY, cand.parentZ, cand.parentSize}
		live[pk] = parentVoxel
		count -= cand.saved

		// Check if the new parent can form a group with its siblings
		grandparentSize := cand.parentSize * 2
		if grandparentSize <= maxMergeSize {
			mask := ^(grandparentSize - 1)
			gpX, gpY, gpZ := cand.parentX&mask, cand.parentY&mask, cand.parentZ&mask

			var siblings []voxelKey
			for dx := 0; dx < 2; dx++ {
				for dy := 0; dy < 2; dy++ {
					for dz := 0; dz < 2; dz++ {
						sk := voxelKey{
							gpX + dx*cand.parentSize,
							gpY + dy*cand.parentSize,
							gpZ + dz*cand.parentSize,
							cand.parentSize,
						}
						if _, ok := live[sk]; ok {
							siblings = append(siblings, sk)
						}
					}
				}
			}

			newCand := makeCandidate(gpX, gpY, gpZ, grandparentSize, siblings)
			if newCand != nil {
				heap.Push(pq, newCand)
			}
		}
	}

	result := make([]Voxel, 0, len(live))
	for _, v := range live {
		result = append(result, *v)
	}
	return result
}

// collapseCandidate represents a potential merge of sibling voxels.
type collapseCandidate struct {
	parentX, parentY, parentZ int
	parentSize                int
	childKeys                 []voxelKey
	error                     int
	saved                     int
	palIdx                    uint8
	flags                     byte
	index                     int
}

// collapseHeap is a min-heap ordered by error (ascending),
// then by saved (descending) to prefer more efficient merges.
type collapseHeap []*collapseCandidate

func (h collapseHeap) Len() int { return len(h) }
func (h collapseHeap) Less(i, j int) bool {
	if h[i].error != h[j].error {
		return h[i].error < h[j].error
	}
	return h[i].saved > h[j].saved
}
func (h collapseHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}
func (h *collapseHeap) Push(x interface{}) {
	n := len(*h)
	item := x.(*collapseCandidate)
	item.index = n
	*h = append(*h, item)
}
func (h *collapseHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*h = old[:n-1]
	return item
}
