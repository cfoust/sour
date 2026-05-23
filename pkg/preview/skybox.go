package preview

import (
	"bytes"
	"context"
	"image"
	_ "image/jpeg"
	_ "image/png"

	V "github.com/cfoust/sour/pkg/game/variables"
	"github.com/cfoust/sour/pkg/min"

	"github.com/rs/zerolog/log"
)

// Default sky colors when no skybox is available.
var (
	defaultSkyTop     = [3]uint8{30, 40, 80}
	defaultSkyHorizon = [3]uint8{80, 100, 140}
	defaultAmbient    = [3]uint8{25, 25, 25}
	defaultSunlight   = [3]uint8{200, 200, 200}
)

// ExtractSkyboxColors reads the skybox cubemap and samples colors for the
// sky gradient. Returns (skyTop, skyHorizon).
func ExtractSkyboxColors(ctx context.Context, processor *min.Processor, vars V.Variables) ([3]uint8, [3]uint8) {
	skyboxVar, ok := vars["skybox"]
	if !ok {
		return defaultSkyTop, defaultSkyHorizon
	}

	skyboxName, ok := skyboxVar.(V.StringVariable)
	if !ok {
		return defaultSkyTop, defaultSkyHorizon
	}

	sides := processor.FindCubemap(ctx, min.NormalizeTexture(string(skyboxName)))
	if len(sides) == 0 {
		return defaultSkyTop, defaultSkyHorizon
	}

	// Try to find "up" side (index 5) and "ft" side (index 2).
	// FindCubemap returns sides in order: lf, rt, ft, bk, dn, up
	// but may skip missing sides, so we sample what we have.

	skyTop := defaultSkyTop
	skyHorizon := defaultSkyHorizon

	// Sample the last side (likely "up") for sky top color.
	if len(sides) >= 6 {
		// "up" is the 6th side
		color, err := sampleImageCenter(ctx, sides[5])
		if err == nil {
			skyTop = color
		}
	}

	// Sample the 3rd side (likely "ft") bottom row for horizon.
	if len(sides) >= 3 {
		color, err := sampleImageBottom(ctx, sides[2])
		if err == nil {
			skyHorizon = color
		}
	}

	return skyTop, skyHorizon
}

// ExtractLightingColors reads ambient and sunlight colors from map variables.
func ExtractLightingColors(vars V.Variables) ([3]uint8, [3]uint8) {
	ambient := defaultAmbient
	sunlight := defaultSunlight

	if v, ok := vars["ambient"]; ok {
		if iv, ok := v.(V.IntVariable); ok {
			packed := int32(iv)
			ambient = unpackColor(packed)
		}
	}

	if v, ok := vars["sunlight"]; ok {
		if iv, ok := v.(V.IntVariable); ok {
			packed := int32(iv)
			sunlight = unpackColor(packed)
		}
	}

	return ambient, sunlight
}

func unpackColor(packed int32) [3]uint8 {
	r := uint8((packed >> 16) & 0xFF)
	g := uint8((packed >> 8) & 0xFF)
	b := uint8(packed & 0xFF)
	return [3]uint8{r, g, b}
}

func sampleImageCenter(ctx context.Context, ref *min.Reference) ([3]uint8, error) {
	data, err := ref.ReadFile(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("preview: failed to read skybox side")
		return defaultSkyTop, err
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return defaultSkyTop, err
	}

	bounds := img.Bounds()
	cx := (bounds.Min.X + bounds.Max.X) / 2
	cy := (bounds.Min.Y + bounds.Max.Y) / 2

	// Sample a small region around center.
	var rSum, gSum, bSum, count uint64
	radius := bounds.Dx() / 8
	if radius < 1 {
		radius = 1
	}
	for y := cy - radius; y <= cy+radius; y++ {
		for x := cx - radius; x <= cx+radius; x++ {
			if x < bounds.Min.X || x >= bounds.Max.X || y < bounds.Min.Y || y >= bounds.Max.Y {
				continue
			}
			r, g, b, _ := img.At(x, y).RGBA()
			rSum += uint64(r >> 8)
			gSum += uint64(g >> 8)
			bSum += uint64(b >> 8)
			count++
		}
	}

	if count == 0 {
		return defaultSkyTop, nil
	}
	return [3]uint8{uint8(rSum / count), uint8(gSum / count), uint8(bSum / count)}, nil
}

func sampleImageBottom(ctx context.Context, ref *min.Reference) ([3]uint8, error) {
	data, err := ref.ReadFile(ctx)
	if err != nil {
		return defaultSkyHorizon, err
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return defaultSkyHorizon, err
	}

	bounds := img.Bounds()
	bottomY := bounds.Max.Y - 1

	var rSum, gSum, bSum, count uint64
	step := bounds.Dx() / 32
	if step < 1 {
		step = 1
	}
	for x := bounds.Min.X; x < bounds.Max.X; x += step {
		r, g, b, _ := img.At(x, bottomY).RGBA()
		rSum += uint64(r >> 8)
		gSum += uint64(g >> 8)
		bSum += uint64(b >> 8)
		count++
	}

	if count == 0 {
		return defaultSkyHorizon, nil
	}
	return [3]uint8{uint8(rSum / count), uint8(gSum / count), uint8(bSum / count)}, nil
}
