package packager

import (
	"github.com/cfoust/sour/pkg/assets"
)

// Mapping represents a source file -> target path pair.
// From is either "fs:/absolute/path" or "id:sha256hash".
// To is the target path in the game filesystem, e.g. "packages/base/complex.ogz".
type Mapping struct {
	From string
	To   string
}

// BuildParams controls how assets are processed.
type BuildParams struct {
	Roots          []assets.Root
	RootStrings    []string
	SkipRoot       string
	CompressImages bool
	DownloadAssets bool
	BuildWeb       bool
	BuildDesktop   bool
}
