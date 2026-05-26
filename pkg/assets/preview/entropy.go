package preview

import "math"

// entropySampleRes is the ray grid resolution for viewpoint entropy.
// 48×48 = 2304 rays per candidate angle — enough for stable estimates.
const entropySampleRes = 48

// viewpointEntropy casts a grid of rays from a candidate camera position and
// computes the Shannon entropy of the projected area distribution of visible
// faces. Background (sky/miss) is included as one face per Vázquez, Feixas,
// Sbert & Heidrich (2001), "Viewpoint Selection using Viewpoint Entropy".
//
// Higher entropy = more informative view showing diverse geometry.
func viewpointEntropy(
	camX, camY, camZ float64,
	focusX, focusY, focusZ float64,
	solidLk *voxelLookup,
	gridSize int,
) float64 {
	fwd := normalize([3]float64{focusX - camX, focusY - camY, focusZ - camZ})
	worldUp := [3]float64{0, 0, 1}
	right := normalize(cross(worldUp, fwd))
	up := cross(fwd, right)

	fov := 45.0 * math.Pi / 180.0
	halfTan := math.Tan(fov / 2)

	counts := make(map[int]int)
	totalHits := 0
	totalRays := entropySampleRes * entropySampleRes

	for py := 0; py < entropySampleRes; py++ {
		for px := 0; px < entropySampleRes; px++ {
			u := (2.0*float64(px)/float64(entropySampleRes) - 1.0) * halfTan
			v := (1.0 - 2.0*float64(py)/float64(entropySampleRes)) * halfTan

			dir := normalize([3]float64{
				fwd[0] + right[0]*u + up[0]*v,
				fwd[1] + right[1]*u + up[1]*v,
				fwd[2] + right[2]*u + up[2]*v,
			})

			idx, _ := ddaMarch(camX, camY, camZ, dir[0], dir[1], dir[2], solidLk, gridSize)
			if idx >= 0 {
				counts[idx]++
				totalHits++
			}
		}
	}

	// Shannon entropy including background as one face
	total := float64(totalRays)
	entropy := 0.0

	skyPixels := totalRays - totalHits
	if skyPixels > 0 {
		p := float64(skyPixels) / total
		entropy -= p * math.Log2(p)
	}

	for _, count := range counts {
		p := float64(count) / total
		entropy -= p * math.Log2(p)
	}

	return entropy
}
