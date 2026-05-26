package assets

import (
	"context"
	"fmt"
	"image/jpeg"
	"os"
	"path/filepath"

	pkgassets "github.com/cfoust/sour/pkg/assets"
	"github.com/cfoust/sour/pkg/assets/packager"
	"github.com/cfoust/sour/pkg/assets/preview"

	"github.com/rs/zerolog/log"
)

// PreviewCmd generates .svox and .jpg preview files for maps.
type PreviewCmd struct {
	Outdir string   `arg:"" help:"Output directory for preview files."`
	Maps   []string `arg:"" optional:"" help:"Map names to generate previews for (default: all)."`
}

func (cmd *PreviewCmd) Run(parent *Cmd) error {
	ctx, roots, _, err := parent.LoadRoots()
	if err != nil {
		return err
	}

	preview.NewRenderService()
	errCh := make(chan error, 1)
	go func() {
		errCh <- cmd.run(ctx, roots)
		preview.Shutdown()
	}()
	preview.Poll(1024, 1024)
	return <-errCh
}

func (cmd *PreviewCmd) run(ctx context.Context, roots []pkgassets.Root) error {
	os.MkdirAll(cmd.Outdir, 0755)

	// Enumerate available maps
	allFiles, err := packager.GetRootFiles(ctx, roots)
	if err != nil {
		return fmt.Errorf("enumerating root files: %w", err)
	}
	mapNames := packager.EnumerateMapNames(allFiles)

	// Filter if specific maps requested
	if len(cmd.Maps) > 0 {
		filter := make(map[string]bool)
		for _, m := range cmd.Maps {
			filter[m] = true
		}
		var filtered []string
		for _, name := range mapNames {
			if filter[name] {
				filtered = append(filtered, name)
			}
		}
		mapNames = filtered
	}

	if len(mapNames) == 0 {
		return fmt.Errorf("no maps found")
	}

	built := 0
	for _, name := range mapNames {
		ogzFile := fmt.Sprintf("packages/base/%s.ogz", name)

		// Read the .ogz data
		var mapData []byte
		for _, root := range roots {
			if root.Exists(ctx, ogzFile) {
				data, err := root.ReadFile(ctx, ogzFile)
				if err == nil {
					mapData = data
					break
				}
			}
		}
		if mapData == nil {
			log.Warn().Msgf("could not read %s", ogzFile)
			continue
		}

		// Generate SVOX
		svoxData, err := preview.Generate(ctx, roots, mapData, ogzFile)
		if err != nil {
			log.Warn().Err(err).Msgf("failed to generate preview for %s", name)
			continue
		}

		// Write .svox
		svoxPath := filepath.Join(cmd.Outdir, name+".svox")
		if err := os.WriteFile(svoxPath, svoxData, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", svoxPath, err)
		}

		// Render still
		prev, err := preview.Decode(svoxData)
		if err != nil {
			log.Warn().Err(err).Msgf("failed to decode preview for %s", name)
			continue
		}

		img := preview.RenderPreview(prev, 1024, 1024)
		jpgPath := filepath.Join(cmd.Outdir, name+".jpg")
		f, err := os.Create(jpgPath)
		if err != nil {
			return fmt.Errorf("creating %s: %w", jpgPath, err)
		}
		if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 85}); err != nil {
			f.Close()
			return fmt.Errorf("encoding %s: %w", jpgPath, err)
		}
		f.Close()

		log.Info().Msgf("preview: %s (%d bytes svox, %s)",
			name, len(svoxData), humanSize(jpgPath))
		built++
	}

	log.Info().Msgf("generated %d previews", built)
	return nil
}

func humanSize(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return "?"
	}
	size := info.Size()
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	return fmt.Sprintf("%.1f KB", float64(size)/1024)
}

