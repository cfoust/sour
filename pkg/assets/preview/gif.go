package preview

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"math"
	"os/exec"

	xdraw "golang.org/x/image/draw"
)

// RenderGIF renders an animated GIF of a full 360° orbit around the preview.
// Uses 24 frames at 15° yaw increments, fixed pitch from the preview's
// computed camera angle. Each frame is rendered with the software renderer
// at the given width/height, then quantized to 256 colors for GIF encoding.
// The result is a seamlessly looping animation suitable for thumbnails.
func RenderGIF(p *MapPreview, width, height int) []byte {
	const (
		numFrames = 120
		yawStep   = 360.0 / numFrames // 3° per frame
		delay     = 6                 // centiseconds per frame (~17 fps, full orbit in ~7.2s)
	)

	baseYaw := float64(p.CameraYaw) / 10.0 // starting yaw in degrees

	frames := make([]*image.Paletted, numFrames)
	delays := make([]int, numFrames)

	for i := 0; i < numFrames; i++ {
		yawDeg := math.Mod(baseYaw+float64(i)*yawStep, 360)

		// Create a copy of the preview with this frame's camera angle
		frame := *p
		frame.CameraYaw = uint16(yawDeg * 10)

		rendered := RenderPreview(&frame, width, height)

		// Downscale if the renderer produced a larger image (GPU renders at
		// fixed framebuffer size). This gives free supersampled antialiasing.
		if rendered.Bounds().Dx() != width || rendered.Bounds().Dy() != height {
			scaled := image.NewRGBA(image.Rect(0, 0, width, height))
			xdraw.BiLinear.Scale(scaled, scaled.Bounds(), rendered, rendered.Bounds(), xdraw.Over, nil)
			rendered = scaled
		}

		frames[i] = quantizeImage(rendered)
		delays[i] = delay
	}

	anim := &gif.GIF{
		Image:     frames,
		Delay:     delays,
		LoopCount: 0, // loop forever
	}

	var buf bytes.Buffer
	gif.EncodeAll(&buf, anim)
	raw := buf.Bytes()

	// Compress with gifsicle if available
	if gifsicle, err := exec.LookPath("gifsicle"); err == nil {
		cmd := exec.Command(gifsicle, "-O3", "--lossy=80")
		cmd.Stdin = bytes.NewReader(raw)
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err == nil && out.Len() > 0 {
			return out.Bytes()
		}
	}

	return raw
}

// quantizeImage converts an RGBA image to a 256-color paletted image
// using a simple median-cut-like approach via popularity.
func quantizeImage(img *image.RGBA) *image.Paletted {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	// Count color frequencies (quantized to 5 bits per channel)
	type rgb5 struct{ r, g, b uint8 }
	freq := make(map[rgb5]int)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := img.RGBAAt(x+bounds.Min.X, y+bounds.Min.Y)
			k := rgb5{c.R >> 3, c.G >> 3, c.B >> 3}
			freq[k]++
		}
	}

	// Pick the 255 most frequent colors + transparent
	type colorFreq struct {
		c rgb5
		n int
	}
	sorted := make([]colorFreq, 0, len(freq))
	for c, n := range freq {
		sorted = append(sorted, colorFreq{c, n})
	}
	// Simple selection sort for top 255 (palette is small)
	for i := 0; i < len(sorted) && i < 255; i++ {
		best := i
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].n > sorted[best].n {
				best = j
			}
		}
		sorted[i], sorted[best] = sorted[best], sorted[i]
	}

	paletteSize := 256
	if len(sorted) < paletteSize {
		paletteSize = len(sorted)
	}

	palette := make(color.Palette, paletteSize)
	for i := 0; i < paletteSize; i++ {
		c := sorted[i].c
		palette[i] = color.RGBA{c.r << 3, c.g << 3, c.b << 3, 255}
	}

	paletted := image.NewPaletted(bounds, palette)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := img.RGBAAt(x+bounds.Min.X, y+bounds.Min.Y)
			paletted.SetColorIndex(x+bounds.Min.X, y+bounds.Min.Y,
				uint8(palette.Index(c)))
		}
	}

	return paletted
}
