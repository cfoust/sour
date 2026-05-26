package preview

import "github.com/cfoust/sour/pkg/maps"

// OctreeStats collects depth statistics from an octree.
type OctreeStats struct {
	LeafCounts map[int]int // depth → number of non-empty leaves
	EmptyCounts map[int]int // depth → number of empty leaves
	MaxDepth   int
}

func CollectOctreeStats(root *maps.Cube, worldSize int) OctreeStats {
	stats := OctreeStats{
		LeafCounts:  make(map[int]int),
		EmptyCounts: make(map[int]int),
	}
	if root == nil {
		return stats
	}
	for _, child := range root.Children {
		if child != nil {
			collectStats(child, 1, &stats)
		}
	}
	return stats
}

func collectStats(c *maps.Cube, depth int, stats *OctreeStats) {
	if len(c.Children) > 0 {
		for _, child := range c.Children {
			if child != nil {
				collectStats(child, depth+1, stats)
			}
		}
		return
	}
	if depth > stats.MaxDepth {
		stats.MaxDepth = depth
	}
	if c.IsEmpty() {
		stats.EmptyCounts[depth]++
	} else {
		stats.LeafCounts[depth]++
	}
}
