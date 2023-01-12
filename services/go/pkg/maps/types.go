package maps

import (
	_ "embed"
	"encoding/binary"
	"fmt"
	"unsafe"

	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/pkg/maps/worldio"

	"github.com/rs/zerolog/log"
)

type CCube worldio.Cube

type Header struct {
	Version    int32
	HeaderSize int32
	WorldSize  int32
	LightMaps  int32
	BlendMap   int32
	NumVars    int32
	NumPVs     int32
	NumVSlots  int32
	GameType   string
}

func NewHeader() *Header {
	return &Header{
		GameType: "fps",
	}
}

type FileHeader struct {
	Magic      [4]byte
	Version    int32
	HeaderSize int32
	WorldSize  int32
	NumEnts    int32
	NumPVs     int32
	LightMaps  int32
}

type NewFooter struct {
	BlendMap  int32
	NumVars   int32
	NumVSlots int32
}

// For versions <=28
type OldFooter struct {
	LightPrecision int32
	LightError     int32
	LightLOD       int32
	Ambient        byte
	WaterColor     [3]byte
	BlendMap       byte
	LerpAngle      byte
	LerpSubDiv     byte
	LerpSubDivSize byte
	BumpError      byte
	SkyLight       [3]byte
	LavaColor      [3]byte
	WaterfallColor [3]byte
	Reserved       [10]byte
	MapTitle       [128]byte
}

type Vector struct {
	X float32
	Y float32
	Z float32
}

type Entity struct {
	Position Vector
	Attr1    int16
	Attr2    int16
	Attr3    int16
	Attr4    int16
	Attr5    int16
	Type     game.EntityType
	Reserved byte
}

const (
	LMID_AMBIENT byte = 0
	LMID_AMBIENT1
	LMID_BRIGHT
	LMID_BRIGHT1
	LMID_DARK
	LMID_DARK1
	LMID_RESERVED
)

type SurfaceInfo struct {
	Lmid     [2]byte
	Verts    byte
	NumVerts byte
}

func InsideWorld(size int32, vector Vector) bool {
	return vector.X >= 0 && vector.X < float32(size) && vector.Y >= 0 && vector.Y < float32(size) && vector.Z >= 0 && vector.Z < float32(size)
}

const (
	LAYER_TOP    byte = (1 << 5)
	LAYER_BOTTOM      = (1 << 6)
	LAYER_DUP         = (1 << 7)
	LAYER_BLEND       = LAYER_TOP | LAYER_BOTTOM
	MAXFACEVERTS      = 15
)

func (surface *SurfaceInfo) TotalVerts() byte {
	if (surface.NumVerts & LAYER_DUP) > 0 {
		return (surface.NumVerts & MAXFACEVERTS) * 2
	}
	return surface.NumVerts & MAXFACEVERTS
}

type SurfaceCompat struct {
	TexCoords [8]byte
	Width     byte
	Height    byte
	X         uint16
	Y         uint16
	Lmid      byte
	Layer     byte
}

type BVec struct {
	X byte
	Y byte
	Z byte
}

type NormalsCompat struct {
	Normals [4]BVec
}

type MergeCompat struct {
	U1 uint16
	U2 uint16
	V1 uint16
	V2 uint16
}

const CUBE_FACTOR = 8

const (
	ET_EMPTY        byte = iota
	ET_LIGHT             = iota
	ET_MAPMODEL          = iota
	ET_PLAYERSTART       = iota
	ET_ENVMAP            = iota
	ET_PARTICLES         = iota
	ET_SOUND             = iota
	ET_SPOTLIGHT         = iota
	ET_GAMESPECIFIC      = iota
)

const (
	OCTSAV_CHILDREN byte = iota
	OCTSAV_EMPTY         = iota
	OCTSAV_SOLID         = iota
	OCTSAV_NORMAL        = iota
	OCTSAV_LODCUB        = iota
)

const (
	VSLOT_SHPARAM  byte = iota
	VSLOT_SCALE         = iota
	VSLOT_ROTATION      = iota
	VSLOT_OFFSET        = iota
	VSLOT_SCROLL        = iota
	VSLOT_LAYER         = iota
	VSLOT_ALPHA         = iota
	VSLOT_COLOR         = iota
	VSLOT_NUM           = iota
)

const (
	MATF_INDEX_SHIFT  = 0
	MATF_VOLUME_SHIFT = 2
	MATF_CLIP_SHIFT   = 5
	MATF_FLAG_SHIFT   = 8

	MATF_INDEX  = 3 << MATF_INDEX_SHIFT
	MATF_VOLUME = 7 << MATF_VOLUME_SHIFT
	MATF_CLIP   = 7 << MATF_CLIP_SHIFT
	MATF_FLAGS  = 0xFF << MATF_FLAG_SHIFT
	MAT_AIR     = 0                      // the default, fill the empty space with air
	MAT_WATER   = 1 << MATF_VOLUME_SHIFT // fill with water, showing waves at the surface
	MAT_LAVA    = 2 << MATF_VOLUME_SHIFT // fill with lava
	MAT_GLASS   = 3 << MATF_VOLUME_SHIFT // behaves like clip but is blended blueish

	MAT_NOCLIP   = 1 << MATF_CLIP_SHIFT // collisions always treat cube as empty
	MAT_CLIP     = 2 << MATF_CLIP_SHIFT // collisions always treat cube as solid
	MAT_GAMECLIP = 3 << MATF_CLIP_SHIFT // game specific clip material

	MAT_DEATH = 1 << MATF_FLAG_SHIFT // force player suicide
	MAT_ALPHA = 4 << MATF_FLAG_SHIFT // alpha blended
)

const (
	F_EMPTY uint32 = 0
	F_SOLID        = 0x80808080
)

// hardcoded texture numbers
const (
	DEFAULT_SKY = iota
	DEFAULT_GEOM
)

type Cube struct {
	Children    []*Cube
	SurfaceInfo [6]SurfaceInfo
	Edges       [12]byte
	Texture     [6]uint16
	Material    uint16
	Merged      byte
	Escaped     byte
}

func (c *Cube) Print() {
	log.Debug().Msgf("%+v", c)
	for _, child := range c.Children {
		child.Print()
	}
}

func (c *Cube) Count() uint {
	var children uint = 0
	for _, child := range c.Children {
		children += child.Count()
	}
	return 1 + children
}

func NewCubes(face uint32, mat uint16) *Cube {
	cubes := make([]*Cube, CUBE_FACTOR)
	for i, _ := range cubes {
		cube := Cube{
			Children: make([]*Cube, 0),
		}
		cube.SetFaces(face)
		for i, _ := range cube.Texture {
			cube.Texture[i] = DEFAULT_GEOM
		}
		cube.Material = mat
		cubes[i] = &cube
	}

	return &Cube{
		Children: cubes,
	}
}

func (c *Cube) GetFace(n int) uint32 {
	i := n * 4
	return uint32(c.Edges[i])<<3 +
		uint32(c.Edges[i+1])<<2 +
		uint32(c.Edges[i+2])<<1 +
		uint32(c.Edges[i+3])
}

func (c *Cube) SetFace(n int, val uint32) {
	a := make([]byte, 4)
	binary.BigEndian.PutUint32(a, val)

	i := n * 4
	for j := 0; j < 4; j++ {
		c.Edges[i+j] = a[j]
	}
}

func (c *Cube) IsEmpty() bool {
	return c.GetFace(0) == F_EMPTY
}

func (c *Cube) IsEntirelySolid() bool {
	return c.GetFace(0) == F_SOLID &&
		c.GetFace(1) == F_SOLID &&
		c.GetFace(2) == F_SOLID
}

func (c *Cube) SetFaces(val uint32) {
	c.SetFace(0, val)
	c.SetFace(1, val)
	c.SetFace(2, val)
}

func (c *Cube) SolidFaces() {
	c.SetFaces(F_SOLID)
}

func (c *Cube) EmptyFaces() {
	c.SetFaces(F_EMPTY)
}

type SlotShaderParam struct {
	Name string
	Loc  int32
	Val  [4]float32
}

type Vec2 struct {
	X float32
	Y float32
}

type IVec2 struct {
	X int32
	Y int32
}

type TexSlot struct {
	Name string
}

type Slot struct {
	Index    int32
	Sts      []TexSlot
	Variants *VSlot
	Loaded   bool
}

func NewSlot() *Slot {
	newSlot := Slot{}
	newSlot.Sts = make([]TexSlot, 0)
	newSlot.Loaded = false
	return &newSlot
}

func (slot *Slot) AddSts(name string) *TexSlot {
	sts := TexSlot{}
	sts.Name = name
	slot.Sts = append(slot.Sts, sts)
	return &slot.Sts[len(slot.Sts)-1]
}

type VSlot struct {
	Index      int32
	Changed    int32
	Layer      int32
	Next       *VSlot
	Slot       *Slot
	Params     []SlotShaderParam
	Linked     bool
	Scale      float32
	Rotation   int32
	Offset     IVec2
	Scroll     Vec2
	AlphaFront float32
	AlphaBack  float32
	ColorScale Vector
	GlowColor  Vector
}

func (vslot *VSlot) AddVariant(slot *Slot) {
	if slot.Variants == nil {
		slot.Variants = vslot
	} else {
		prev := slot.Variants
		for prev != nil {
			prev = prev.Next
		}
		prev.Next = vslot
	}
}

func NewVSlot(owner *Slot, index int32) *VSlot {
	vslot := VSlot{
		Index: index,
		Slot:  owner,
	}
	if owner != nil {
		vslot.AddVariant(owner)
	}
	return &vslot
}

type GameMap struct {
	Header    Header
	Entities  []Entity
	Vars      game.Variables
	WorldRoot *Cube
	VSlots    []*VSlot
	C         worldio.MapState
}

func (m *GameMap) Destroy() {
	worldio.M.Lock()
	worldio.Free_state(m.C)
	worldio.M.Unlock()
}

// This is generated using:
// sourdump -type cfg -index default.textures data/default_map_settings.cfg
// (that is not the full command, you should be able to infer it though)
//go:embed default.textures
var DEFAULT_MAP_SLOTS []byte

func LoadDefaultSlots(map_ *GameMap) error {
	worldio.M.Lock()
	loadOk := worldio.Load_texture_index(
		uintptr(unsafe.Pointer(&(DEFAULT_MAP_SLOTS)[0])),
		int64(len(DEFAULT_MAP_SLOTS)),
		map_.C,
	)
	worldio.M.Unlock()
	if !loadOk {
		return fmt.Errorf("failed to load texture index")
	}

	return nil
}

func NewMap() (*GameMap, error) {
	worldio.M.Lock()
	c := worldio.Empty_world(12)
	worldio.M.Unlock()
	map_ := GameMap{
		Header: Header{
			Version:    game.MAP_VERSION,
			GameType:   "fps",
			HeaderSize: 40,
			WorldSize:  1024,
		},
		Entities:  make([]Entity, 0),
		WorldRoot: EmptyMap(1024),
		Vars:      make(map[string]game.Variable),
		VSlots:    make([]*VSlot, 0),
		C:         c,
	}

	if err := LoadDefaultSlots(&map_); err != nil {
		return nil, err
	}

	return &map_, nil
}
