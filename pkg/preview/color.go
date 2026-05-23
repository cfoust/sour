package preview

import (
	"bytes"
	"context"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"path/filepath"

	"github.com/cfoust/sour/pkg/maps"
	"github.com/cfoust/sour/pkg/min"

	"github.com/rs/zerolog/log"
)

// SampleTextureColor reads a texture image and computes its average RGB color.
// For performance, it subsamples large images.
func SampleTextureColor(ctx context.Context, ref *min.Reference) ([3]uint8, error) {
	data, err := ref.ReadFile(ctx)
	if err != nil {
		return [3]uint8{128, 128, 128}, err
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return [3]uint8{128, 128, 128}, err
	}

	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	// Subsample: take at most ~1024 samples.
	stepX := w / 32
	stepY := h / 32
	if stepX < 1 {
		stepX = 1
	}
	if stepY < 1 {
		stepY = 1
	}

	var rSum, gSum, bSum, count uint64
	for y := bounds.Min.Y; y < bounds.Max.Y; y += stepY {
		for x := bounds.Min.X; x < bounds.Max.X; x += stepX {
			r, g, b, _ := img.At(x, y).RGBA()
			rSum += uint64(r >> 8)
			gSum += uint64(g >> 8)
			bSum += uint64(b >> 8)
			count++
		}
	}

	if count == 0 {
		return [3]uint8{128, 128, 128}, nil
	}

	return [3]uint8{
		uint8(rSum / count),
		uint8(gSum / count),
		uint8(bSum / count),
	}, nil
}

// CollectUsedVSlots walks the octree and returns all unique VSlot texture
// indices referenced by non-empty cubes.
func CollectUsedVSlots(root *maps.Cube) map[uint16]struct{} {
	used := make(map[uint16]struct{})
	if root == nil {
		return used
	}
	collectVSlotsRecursive(root, used)
	return used
}

func collectVSlotsRecursive(c *maps.Cube, used map[uint16]struct{}) {
	if c == nil {
		return
	}
	if len(c.Children) > 0 {
		for _, child := range c.Children {
			collectVSlotsRecursive(child, used)
		}
		return
	}
	if c.IsEmpty() {
		return
	}
	for _, t := range c.Texture {
		if t != maps.DEFAULT_SKY {
			used[t] = struct{}{}
		}
	}
}

// BuildPalette samples colors from all texture slots referenced by the octree.
// Returns the palette array and a mapping from VSlot index to palette index.
func BuildPalette(
	ctx context.Context,
	processor *min.Processor,
	usedVSlots map[uint16]struct{},
) ([][3]uint8, map[uint16]uint8) {
	// Index 0 = default gray
	palette := [][3]uint8{{128, 128, 128}}
	paletteMap := make(map[uint16]uint8)

	// Map each Slot to its sampled color, so VSlots sharing the same
	// Slot share the same palette entry.
	slotColors := make(map[int32]uint8) // Slot.Index -> palette index

	for vsIdx := range usedVSlots {
		if int(vsIdx) >= len(processor.VSlots) {
			continue
		}
		vs := processor.VSlots[vsIdx]
		if vs == nil || vs.Slot == nil {
			continue
		}
		slot := vs.Slot

		// Check if we've already sampled this slot.
		if palIdx, ok := slotColors[slot.Index]; ok {
			paletteMap[vsIdx] = palIdx
			continue
		}

		color := sampleSlotColor(ctx, processor, slot)

		// Apply VSlot color scale if present.
		if vs.Changed&(1<<maps.VSLOT_COLOR) != 0 {
			color[0] = clampByte(float32(color[0]) * vs.ColorScale.X)
			color[1] = clampByte(float32(color[1]) * vs.ColorScale.Y)
			color[2] = clampByte(float32(color[2]) * vs.ColorScale.Z)
		}

		if len(palette) >= 255 {
			// Palette full, map to closest existing entry.
			paletteMap[vsIdx] = findClosestPalette(palette, color)
			continue
		}

		palIdx := uint8(len(palette))
		palette = append(palette, color)
		slotColors[slot.Index] = palIdx
		paletteMap[vsIdx] = palIdx
	}

	return palette, paletteMap
}

func sampleSlotColor(ctx context.Context, processor *min.Processor, slot *maps.Slot) [3]uint8 {
	if len(slot.Sts) == 0 {
		return [3]uint8{128, 128, 128}
	}

	// Use the first (diffuse) texture.
	texName := min.NormalizeTexture(slot.Sts[0].Name)
	ref := processor.FindTexture(ctx, texName)
	if ref == nil {
		// Try without "packages/" prefix normalization.
		ref = processor.SearchFile(ctx, texName)
	}
	if ref == nil {
		log.Debug().Msgf("preview: texture not found: %s", texName)
		return [3]uint8{128, 128, 128}
	}

	color, err := SampleTextureColor(ctx, ref)
	if err != nil {
		log.Debug().Err(err).Msgf("preview: failed to sample texture %s", texName)
		return [3]uint8{128, 128, 128}
	}

	return color
}

func clampByte(v float32) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

func findClosestPalette(palette [][3]uint8, color [3]uint8) uint8 {
	bestIdx := uint8(0)
	bestDist := 3 * 256 * 256 // max possible distance
	for i, c := range palette {
		dr := int(color[0]) - int(c[0])
		dg := int(color[1]) - int(c[1])
		db := int(color[2]) - int(c[2])
		dist := dr*dr + dg*dg + db*db
		if dist < bestDist {
			bestDist = dist
			bestIdx = uint8(i)
		}
	}
	return bestIdx
}

func findTextureRef(ctx context.Context, processor *min.Processor, name string) *min.Reference {
	normalized := min.NormalizeTexture(name)
	ref := processor.FindTexture(ctx, normalized)
	if ref != nil {
		return ref
	}
	// Some textures are stored with full path including extension.
	for _, ext := range []string{"png", "jpg"} {
		ref = processor.SearchFile(ctx, filepath.Join("packages", normalized+"."+ext))
		if ref != nil {
			return ref
		}
	}
	return nil
}
