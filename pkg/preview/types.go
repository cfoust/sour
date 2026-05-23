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

// VoxelFlags encodes cube properties.
const (
	FlagSolid  byte = 1 << 0
	FlagNormal byte = 1 << 1
	// Material bits 2-4
	FlagWater byte = 1 << 2
	FlagLava  byte = 2 << 2
	FlagGlass byte = 3 << 2
	FlagClip  byte = 4 << 2
)

type Voxel struct {
	X, Y, Z      uint8
	PaletteIndex uint8
	Flags        byte
	AO           byte
}

type PreviewEntity struct {
	X, Y, Z uint8
	Type    PreviewEntityType
}

type MapPreview struct {
	MaxDepth   uint8
	GridSize   uint16
	WorldSize  uint32
	Palette    [][3]uint8
	Voxels     []Voxel
	Entities   []PreviewEntity
	SkyTop     [3]uint8
	SkyHorizon [3]uint8
	Ambient    [3]uint8
	Sunlight   [3]uint8
}
