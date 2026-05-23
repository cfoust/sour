package preview

// PreviewEntityType represents a gameplay-relevant entity type in the preview.
type PreviewEntityType byte

const (
	EntityPlayerStart PreviewEntityType = iota
	EntityFlag
	EntityBase
	EntityTeleport
	EntityJumpPad
	EntityHealth
	EntityArmour
	EntityAmmo
	EntityQuad
)

// VoxelFlags encodes cube properties and voxel size.
// Bits 0-1: geometry type (solid/normal)
// Bits 2-4: material (water/lava/glass/clip)
// Bits 5-7: log2(size) — voxel spans 2^n grid cells per axis
const (
	FlagSolid  byte = 1 << 0
	FlagNormal byte = 1 << 1
	// Material bits 2-4
	FlagWater byte = 1 << 2
	FlagLava  byte = 2 << 2
	FlagGlass byte = 3 << 2
	FlagClip  byte = 4 << 2

	FlagSizeShift = 5
	FlagSizeMask  = 0x7 << FlagSizeShift
)

// Size returns the number of grid cells this voxel spans per axis.
func (v Voxel) Size() int {
	return 1 << ((v.Flags >> FlagSizeShift) & 0x7)
}

// Voxel uses uint16 coordinates to support grids up to 1024³.
// Binary layout: X(2) Y(2) Z(2) PaletteIndex(1) Flags(1) AO(1) = 9 bytes
type Voxel struct {
	X, Y, Z      uint16
	PaletteIndex uint8
	Flags        byte
	AO           byte
}

// PreviewEntity uses uint16 coordinates.
// Binary layout: X(2) Y(2) Z(2) Type(1) = 7 bytes
type PreviewEntity struct {
	X, Y, Z uint16
	Type    PreviewEntityType
}

type MapPreview struct {
	MaxDepth    uint8
	GridSize    uint16
	WorldSize   uint32
	Palette     [][3]uint8
	Voxels      []Voxel
	Entities    []PreviewEntity
	SkyTop      [3]uint8
	SkyHorizon  [3]uint8
	Ambient     [3]uint8
	Sunlight    [3]uint8
	FocusX      uint16 // orbit target in grid coords (Sauer X)
	FocusY      uint16 // orbit target in grid coords (Sauer Y)
	FocusZ      uint16 // orbit target in grid coords (Sauer Z)
	FocusRadius uint16 // suggested orbit distance in grid units
}
