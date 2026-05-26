package preview

import "container/heap"

// SimplifyVoxels reduces a voxel set to at most targetCount voxels using
// error-driven octree collapse, inspired by Garland & Heckbert (1997),
// "Surface Simplification Using Quadric Error Metrics."
//
// The algorithm starts with the full-detail voxel set and iteratively merges
// the cheapest octree sibling groups. Eight sibling voxels sharing a parent
// position collapse into one voxel at double the size. The collapse error is
// the number of empty child cells that become filled — a solid 2×2×2 block
// (8/8 children) collapses for free (error=0), while a thin bridge slab
// (1/8 children) has error=7 and is preserved until the very end.
//
// This gives a single tuning parameter (targetCount) that automatically
// allocates detail to visually important features.
type voxelKey struct {
	x, y, z int
	size    int
}

func SimplifyVoxels(voxels []Voxel, targetCount int) []Voxel {
	if len(voxels) <= targetCount {
		return voxels
	}

	// Use a map from position to voxel for the live set.
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

	// Build initial collapse candidates.
	// A collapse merges all voxels that share a parent octree cell.
	// Parent of (x, y, z, size) is at (x & ^(2*size-1), ..., 2*size).
	pq := &collapseHeap{}
	heap.Init(pq)

	buildCandidates := func() {
		// Group voxels by parent position
		type parentKey struct {
			x, y, z int
			size    int // parent size = child size * 2
		}
		groups := make(map[parentKey][]voxelKey)

		for k := range live {
			parentSize := k.size * 2
			mask := ^(parentSize - 1)
			pk := parentKey{k.x & mask, k.y & mask, k.z & mask, parentSize}
			groups[pk] = append(groups[pk], k)
		}

		pq = &collapseHeap{}
		for pk, children := range groups {
			if len(children) < 2 {
				continue // need at least 2 children to save voxels
			}
			// All children must be the same size
			childSize := children[0].size
			allSame := true
			for _, c := range children {
				if c.size != childSize {
					allSame = false
					break
				}
			}
			if !allSame {
				continue
			}

			// Collapse error = empty cells that get filled
			emptyCount := 8 - len(children)
			saved := len(children) - 1

			// Pick the most common palette index among children
			palCounts := make(map[uint8]int)
			var bestFlags byte
			for _, ck := range children {
				v := live[ck]
				palCounts[v.PaletteIndex]++
				bestFlags = v.Flags
			}
			bestPal := uint8(0)
			bestPalCount := 0
			for p, c := range palCounts {
				if c > bestPalCount {
					bestPalCount = c
					bestPal = p
				}
			}

			cand := &collapseCandidate{
				parentX:    pk.x,
				parentY:    pk.y,
				parentZ:    pk.z,
				parentSize: pk.size,
				childKeys:  children,
				error:      emptyCount,
				saved:      saved,
				palIdx:     bestPal,
				flags:      bestFlags,
			}
			heap.Push(pq, cand)
		}
	}

	buildCandidates()

	for count > targetCount && pq.Len() > 0 {
		cand := heap.Pop(pq).(*collapseCandidate)

		// Verify all children still exist (may have been collapsed already)
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
		mask := ^(grandparentSize - 1)
		gpX, gpY, gpZ := cand.parentX&mask, cand.parentY&mask, cand.parentZ&mask

		// Find siblings of the new parent
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

		if len(siblings) >= 2 {
			emptyCount := 8 - len(siblings)
			saved := len(siblings) - 1

			palCounts := make(map[uint8]int)
			var bestFlags byte
			for _, sk := range siblings {
				v := live[sk]
				palCounts[v.PaletteIndex]++
				bestFlags = v.Flags
			}
			bestPal := uint8(0)
			bestPalCount := 0
			for p, c := range palCounts {
				if c > bestPalCount {
					bestPalCount = c
					bestPal = p
				}
			}

			newCand := &collapseCandidate{
				parentX:    gpX,
				parentY:    gpY,
				parentZ:    gpZ,
				parentSize: grandparentSize,
				childKeys:  siblings,
				error:      emptyCount,
				saved:      saved,
				palIdx:     bestPal,
				flags:      bestFlags,
			}
			heap.Push(pq, newCand)
		}
	}

	// Collect results
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
	error                     int // empty cells filled by merge (0-7)
	saved                     int // voxels saved by merge (children - 1)
	palIdx                    uint8
	flags                     byte
	index                     int // heap index
}

// collapseHeap is a min-heap of collapse candidates, ordered by error.
// At equal error, prefer merges that save more voxels.
type collapseHeap []*collapseCandidate

func (h collapseHeap) Len() int { return len(h) }
func (h collapseHeap) Less(i, j int) bool {
	if h[i].error != h[j].error {
		return h[i].error < h[j].error
	}
	return h[i].saved > h[j].saved // prefer saving more voxels at equal error
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
