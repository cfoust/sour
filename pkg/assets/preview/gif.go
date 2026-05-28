package preview

import (
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"

	xdraw "golang.org/x/image/draw"
)

// RenderWebP renders an animated WebP of a full 360° orbit around the preview.
// Uses 120 frames at 3° yaw increments, fixed pitch from the preview's
// computed camera angle. Frames are rendered via RenderPreview (GPU if
// available), downscaled to the target size, and assembled into a looping
// animated WebP using img2webp.
func RenderWebP(p *MapPreview, width, height int) ([]byte, error) {
	const (
		numFrames = 120
		yawStep   = 360.0 / numFrames
		delayMs   = 60 // milliseconds per frame (~17 fps, full orbit in ~7.2s)
	)

	baseYaw := float64(p.CameraYaw) / 10.0

	// Write frames to a temp directory
	tmpDir, err := os.MkdirTemp("", "webp-frames-")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	for i := 0; i < numFrames; i++ {
		yawDeg := math.Mod(baseYaw+float64(i)*yawStep, 360)

		frame := *p
		frame.CameraYaw = uint16(yawDeg * 10)

		rendered := RenderPreview(&frame, width, height)

		// Downscale if GPU rendered at larger size (free supersampling)
		if rendered.Bounds().Dx() != width || rendered.Bounds().Dy() != height {
			scaled := image.NewRGBA(image.Rect(0, 0, width, height))
			xdraw.BiLinear.Scale(scaled, scaled.Bounds(), rendered, rendered.Bounds(), xdraw.Over, nil)
			rendered = scaled
		}

		framePath := filepath.Join(tmpDir, fmt.Sprintf("frame%03d.png", i))
		f, err := os.Create(framePath)
		if err != nil {
			return nil, fmt.Errorf("creating frame %d: %w", i, err)
		}
		if err := png.Encode(f, rendered); err != nil {
			f.Close()
			return nil, fmt.Errorf("encoding frame %d: %w", i, err)
		}
		f.Close()
	}

	// Assemble with img2webp
	outPath := filepath.Join(tmpDir, "output.webp")

	args := []string{"-lossy", "-q", "60", "-loop", "0"}
	for i := 0; i < numFrames; i++ {
		args = append(args, "-d", fmt.Sprintf("%d", delayMs))
		args = append(args, filepath.Join(tmpDir, fmt.Sprintf("frame%03d.png", i)))
	}
	args = append(args, "-o", outPath)

	cmd := exec.Command("img2webp", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("img2webp failed: %w\n%s", err, string(out))
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		return nil, fmt.Errorf("reading webp: %w", err)
	}

	return data, nil
}
