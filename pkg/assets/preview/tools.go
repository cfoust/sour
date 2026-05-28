package preview

import (
	"os/exec"
	"sync"

	"github.com/rs/zerolog/log"
)

// Tools holds paths to external programs used by the preview system.
// Empty string means the tool is not available and features depending
// on it are gracefully disabled.
type Tools struct {
	Img2webp       string // animated WebP assembly (brew install webp)
	ImageMagick    string // image compression (brew install imagemagick)
}

// ExternalTools is populated once by DetectTools.
var ExternalTools Tools

var detectOnce sync.Once

// DetectTools checks for external programs and logs their availability.
// Safe to call multiple times — detection runs only once.
func DetectTools() {
	detectOnce.Do(func() {
		check := func(name, installHint string) string {
			p, err := exec.LookPath(name)
			if err != nil {
				log.Warn().Msgf("preview: %s not found — %s", name, installHint)
				return ""
			}
			log.Info().Msgf("preview: %s available", name)
			return p
		}

		ExternalTools.Img2webp = check("img2webp", "WebP animation disabled (install with: brew install webp)")
		// ImageMagick: try v7 "magick" first, fall back to v6 "convert"
		if p, err := exec.LookPath("magick"); err == nil {
			ExternalTools.ImageMagick = p
			log.Info().Msgf("preview: magick available")
		} else if p, err := exec.LookPath("convert"); err == nil {
			ExternalTools.ImageMagick = p
			log.Info().Msgf("preview: convert available (ImageMagick 6)")
		} else {
			log.Warn().Msg("preview: ImageMagick not found — image compression disabled (install with: brew install imagemagick)")
		}
	})
}
