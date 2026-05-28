package packager

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cfoust/sour/pkg/assets/preview"
)

var imageExtensions = map[string]bool{
	".dds": true,
	".jpg": true,
	".png": true,
}

const minCompressSize = 128000

// IsImageCompressible returns true if the file is a compressible image above the size threshold.
func IsImageCompressible(path string, minSize int64) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if !imageExtensions[ext] {
		return false
	}

	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.Size() >= minSize
}

// CompressImage uses ImageMagick to resize an image to 25% (two 50% passes).
// Returns nil if ImageMagick is not installed (skips silently).
func CompressImage(src, dst string) error {
	preview.DetectTools()
	convertPath := preview.ExternalTools.ImageMagick
	if convertPath == "" {
		return nil
	}

	cmd := exec.Command(convertPath, src, "-resize", "50%", dst)
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command(convertPath, dst, "-resize", "50%", dst)
	return cmd.Run()
}
