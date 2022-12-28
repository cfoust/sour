package maps

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"io"
	"os"

	"github.com/cfoust/sour/pkg/game"
	"github.com/rs/zerolog/log"
)

type Header struct {
	Version    int32
	HeaderSize int32
	WorldSize  int32
	LightMaps  int32
	BlendMap   int32
	NumVars    int32
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
	Type     byte
	Reserved byte
}

type SurfaceInfo struct {
	Lmid     [2]byte
	Verts    byte
	NumVerts byte
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

const MAP_VERSION = 33

type VariableType byte

const (
	VariableTypeInt    VariableType = iota
	VariableTypeFloat               = iota
	VariableTypeString              = iota
)

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

type Cube struct {
	Children    *[]Cube
	SurfaceInfo [6]SurfaceInfo
	Edges       [12]byte
	Texture     [6]uint16
	Material    uint16
	Merged      byte
	Escaped     byte
}

type VSlot struct {
	Index   int32
	Changed int32
	Layer   int32
}

type VSlotData struct {
	Slots    []*VSlot
	Previous []int32
}

type IntVariable int32

func (v IntVariable) Type() VariableType {
	return VariableTypeInt
}

type FloatVariable float32

func (v FloatVariable) Type() VariableType {
	return VariableTypeFloat
}

type StringVariable string

func (v StringVariable) Type() VariableType {
	return VariableTypeString
}

type Variable interface {
	Type() VariableType
}

func GetDefaultVariables() map[string]Variable {
	return map[string]Variable{
		"ambient":           IntVariable(0x191919), // 1 -> 0xFFFFFF
		"atmo":              IntVariable(0),        // 0 -> 1
		"atmoalpha":         FloatVariable(1),      // 0 -> 1
		"atmobright":        FloatVariable(1),      // 0 -> 16
		"atmodensity":       FloatVariable(1),      // 0 -> 16
		"atmohaze":          FloatVariable(0.1),    // 0 -> 16
		"atmoheight":        FloatVariable(1),      // 1e-3f -> 1e3f
		"atmoozone":         FloatVariable(1),      // 0 -> 16
		"atmoplanetsize":    FloatVariable(1),      // 1e-3f -> 1e3f
		"atmosundisk":       IntVariable(0),        // 0 -> 0xFFFFFF
		"atmosundiskbright": FloatVariable(1),      // 0 -> 16
		"atmosundiskcorona": FloatVariable(0.4),    // 0 -> 1
		"atmosundisksize":   FloatVariable(12),     // 0 -> 90
		"atmosunlight":      IntVariable(0),        // 0 -> 0xFFFFFF
		"atmosunlightscale": FloatVariable(1),      // 0 -> 16
		"blurlms":           IntVariable(0),        // 0 -> 2
		"blurskylight":      IntVariable(0),        // 0 -> 2
		"bumperror":         IntVariable(3),        // 1 -> 16
		"causticcontrast":   FloatVariable(0.6),    // 0 -> 1
		"causticmillis":     IntVariable(75),       // 0 -> 1000
		"causticscale":      IntVariable(50),       // 0 -> 10000
		"cloudalpha":        FloatVariable(1),      // 0 -> 1
		"cloudbox":          StringVariable(""),
		"cloudboxalpha":     FloatVariable(1),      // 0 -> 1
		"cloudboxcolour":    IntVariable(0xFFFFFF), // 0 -> 0xFFFFFF
		"cloudclip":         FloatVariable(0.5),    // 0 -> 1
		"cloudcolour":       IntVariable(0xFFFFFF), // 0 -> 0xFFFFFF
		"cloudfade":         FloatVariable(0.2),    // 0 -> 1
		"cloudheight":       FloatVariable(0.2),    // -1 -> 1
		"cloudlayer":        StringVariable(""),
		"cloudoffsetx":      FloatVariable(0),      // 0 -> 1
		"cloudoffsety":      FloatVariable(0),      // 0 -> 1
		"cloudscale":        FloatVariable(1),      // 0.001 -> 64
		"cloudscrollx":      FloatVariable(0),      // -16 -> 16
		"cloudscrolly":      FloatVariable(0),      // -16 -> 16
		"cloudsubdiv":       IntVariable(16),       // 4 -> 64
		"envmapbb":          IntVariable(0),        // 0 -> 1
		"envmapradius":      IntVariable(128),      // 0 -> 10000
		"fog":               IntVariable(4000),     // 16 -> 1000024
		"fogcolour":         IntVariable(0x8099B3), // 0 -> 0xFFFFFF
		"fogdomecap":        IntVariable(1),        // 0 -> 1
		"fogdomeclip":       FloatVariable(1),      // 0 -> 1
		"fogdomeclouds":     IntVariable(1),        // 0 -> 1
		"fogdomecolour":     IntVariable(0),        // 0 -> 0xFFFFFF
		"fogdomeheight":     FloatVariable(-0.5),   // -1 -> 1
		"fogdomemax":        FloatVariable(0),      // 0 -> 1
		"fogdomemin":        FloatVariable(0),      // 0 -> 1
		"glass2colour":      IntVariable(0x2080C0), // 0 -> 0xFFFFFF
		"glass3colour":      IntVariable(0x2080C0), // 0 -> 0xFFFFFF
		"glass4colour":      IntVariable(0x2080C0), // 0 -> 0xFFFFFF
		"glasscolour":       IntVariable(0x2080C0), // 0 -> 0xFFFFFF
		"grassalpha":        FloatVariable(1),      // 0 -> 1
		"grassanimmillis":   IntVariable(3000),     // 0 -> 60000
		"grassanimscale":    FloatVariable(0.03),   // 0 -> 1
		"grasscolour":       IntVariable(0xFFFFFF), // 0 -> 0xFFFFFF
		"grassscale":        IntVariable(2),        // 1 -> 64
		"lava2colour":       IntVariable(0xFF4000), // 0 -> 0xFFFFFF
		"lava2fog":          IntVariable(50),       // 0 -> 10000
		"lava3colour":       IntVariable(0xFF4000), // 0 -> 0xFFFFFF
		"lava3fog":          IntVariable(50),       // 0 -> 10000
		"lava4colour":       IntVariable(0xFF4000), // 0 -> 0xFFFFFF
		"lava4fog":          IntVariable(50),       // 0 -> 10000
		"lavacolour":        IntVariable(0xFF4000), // 0 -> 0xFFFFFF
		"lavafog":           IntVariable(50),       // 0 -> 10000
		"lerpangle":         IntVariable(44),       // 0 -> 180
		"lerpsubdiv":        IntVariable(2),        // 0 -> 4
		"lerpsubdivsize":    IntVariable(4),        // 4 -> 128
		"lighterror":        IntVariable(8),        // 1 -> 16
		"lightlod":          IntVariable(0),        // 0 -> 10
		"lightprecision":    IntVariable(32),       // 1 -> 1024
		"maptitle":          StringVariable("Untitled Map by Unknown"),
		"mapversion":        IntVariable(MAP_VERSION), // 1 -> 0
		"minimapclip":       IntVariable(0),           // 0 -> 1
		"minimapcolour":     IntVariable(0),           // 0 -> 0xFFFFFF
		"minimapheight":     IntVariable(0),           // 0 -> 2<<16
		"refractclear":      IntVariable(0),           // 0 -> 1
		"refractsky":        IntVariable(0),           // 0 -> 1
		"shadowmapambient":  IntVariable(0),           // 0 -> 0xFFFFFF
		"shadowmapangle":    IntVariable(0),           // 0 -> 360
		"skybox":            StringVariable(""),
		"skyboxcolour":      IntVariable(0xFFFFFF), // 0 -> 0xFFFFFF
		"skylight":          IntVariable(0),        // 0 -> 0xFFFFFF
		"skytexturelight":   IntVariable(1),        // 0 -> 1
		"spincloudlayer":    FloatVariable(0),      // -720 -> 720
		"spinclouds":        FloatVariable(0),      // -720 -> 720
		"spinsky":           FloatVariable(0),      // -720 -> 720
		"sunlight":          IntVariable(0),        // 0 -> 0xFFFFFF
		"sunlightpitch":     IntVariable(90),       // -90 -> 90
		"sunlightscale":     FloatVariable(1),      // 0 -> 16
		"sunlightyaw":       IntVariable(0),        // 0 -> 360
		"water2colour":      IntVariable(0x144650), // 0 -> 0xFFFFFF
		"water2fallcolour":  IntVariable(0),        // 0 -> 0xFFFFFF
		"water2fog":         IntVariable(150),      // 0 -> 10000
		"water2spec":        IntVariable(150),      // 0 -> 1000
		"water3colour":      IntVariable(0x144650), // 0 -> 0xFFFFFF
		"water3fallcolour":  IntVariable(0),        // 0 -> 0xFFFFFF
		"water3fog":         IntVariable(150),      // 0 -> 10000
		"water3spec":        IntVariable(150),      // 0 -> 1000
		"water4colour":      IntVariable(0x144650), // 0 -> 0xFFFFFF
		"water4fallcolour":  IntVariable(0),        // 0 -> 0xFFFFFF
		"water4fog":         IntVariable(150),      // 0 -> 10000
		"water4spec":        IntVariable(150),      // 0 -> 1000
		"watercolour":       IntVariable(0x144650), // 0 -> 0xFFFFFF
		"waterfallcolour":   IntVariable(0),        // 0 -> 0xFFFFFF
		"waterfog":          IntVariable(150),      // 0 -> 10000
		"waterspec":         IntVariable(150),      // 0 -> 1000
		"yawcloudlayer":     IntVariable(0),        // 0 -> 360
		"yawclouds":         IntVariable(0),        // 0 -> 360
		"yawsky":            IntVariable(0),        // 0 -> 360
	}
}

type GameMap struct {
	Header   Header
	Entities []Entity
	Vars     map[string]Variable
	Cubes    []Cube
	VSlots   VSlotData
}

func NewMap() *GameMap {
	return &GameMap{
		Header: Header{
			Version: MAP_VERSION,
			HeaderSize: 40,
			WorldSize: 1024,
		},
		Entities: make([]Entity, 0),
		Cubes:    make([]Cube, 8),
		Vars:     make(map[string]Variable),
		VSlots: VSlotData{
			Slots:    make([]*VSlot, 0),
			Previous: make([]int32, 0),
		},
	}
}

func (m *GameMap) Encode() ([]byte, error) {
	p := game.Packet{}

	err := p.PutRaw(
		FileHeader{
			Magic:      [4]byte{byte('O'), byte('C'), byte('T'), byte('A')},
			Version:    MAP_VERSION,
			HeaderSize: 40,
			WorldSize:  m.Header.WorldSize,
			NumEnts:    int32(len(m.Entities)),
			// TODO
			NumPVs:    0,
			LightMaps: 0,
		},
		NewFooter{
			BlendMap: 0,
			NumVars:  int32(len(m.Vars)),
			// TODO
			NumVSlots: 0,
		},
	)
	if err != nil {
		return p, err
	}

	defaults := GetDefaultVariables()

	for key, variable := range m.Vars {
		defaultValue, defaultExists := defaults[key]
		if !defaultExists || defaultValue.Type() != variable.Type() {
			log.
				Warn().
				Msgf("variable %s is not a valid map variable or invalid type", key)
		}
		err = p.PutRaw(
			byte(variable.Type()),
			uint16(len(key)),
			[]byte(key),
		)
		if err != nil {
			return p, err
		}

		switch variable.Type() {
		case VariableTypeInt:
			err = p.PutRaw(variable.(IntVariable))
		case VariableTypeFloat:
			err = p.PutRaw(variable.(FloatVariable))
		case VariableTypeString:
			value := variable.(StringVariable)
			err = p.PutRaw(
				uint16(len(value)),
				[]byte(value),
			)
		}
		if err != nil {
			return p, err
		}
	}

	err = p.PutRaw(
		// game type (almost always FPS)
		byte(len(m.Header.GameType)),
		[]byte(m.Header.GameType),
		byte(0), // null terminated

		uint16(0), // eif
		uint16(0), // extras

		uint16(0), // texture MRU
	)
	if err != nil {
		return p, err
	}

	for _, entity := range m.Entities {
		err = p.PutRaw(entity)
		if err != nil {
			return p, err
		}
	}

	return p, nil
}

func (m *GameMap) EncodeOGZ() ([]byte, error) {
	data, err := m.Encode()
	if err != nil {
		return data, err
	}

	var buffer bytes.Buffer
	gz := gzip.NewWriter(&buffer)
	_, err = gz.Write(data)
	if err != nil {
		return nil, err
	}
	gz.Close()

	return buffer.Bytes(), nil
}

type Unpacker struct {
	Reader *bytes.Reader
}

func NewUnpacker(reader *bytes.Reader) *Unpacker {
	unpacker := Unpacker{}
	unpacker.Reader = reader
	return &unpacker
}

func (unpack *Unpacker) Read(data any) error {
	return binary.Read(unpack.Reader, binary.LittleEndian, data)
}

func (unpack *Unpacker) Float() float32 {
	var value float32
	unpack.Read(&value)
	return value
}

func (unpack *Unpacker) Int() int32 {
	var value int32
	unpack.Read(&value)
	return value
}

func (unpack *Unpacker) Char() byte {
	var value byte
	unpack.Read(&value)
	return value
}

func (unpack *Unpacker) Short() uint16 {
	var value uint16
	unpack.Read(&value)
	return value
}

func (unpack *Unpacker) String() string {
	bytes := unpack.Short()
	value := make([]byte, bytes)
	unpack.Read(value)
	return string(value)
}

func (unpack *Unpacker) StringByte() string {
	var bytes byte
	unpack.Read(&bytes)
	value := make([]byte, bytes+1)
	unpack.Read(value)
	return string(value)
}

func (unpack *Unpacker) Skip(bytes int64) {
	unpack.Reader.Seek(bytes, io.SeekCurrent)
}

func (unpack *Unpacker) Tell() int64 {
	pos, _ := unpack.Reader.Seek(0, io.SeekCurrent)
	return pos
}

func LoadCube(unpack *Unpacker, cube *Cube, mapVersion int32) error {
	//log.Printf("pos=%d", unpack.Tell())

	var hasChildren = false
	octsav := unpack.Char()

	//fmt.Printf("pos %d octsav %d\n", unpack.Tell(), octsav&0x7)

	switch octsav & 0x7 {
	case OCTSAV_CHILDREN:
		children, err := LoadChildren(unpack, mapVersion)
		if err != nil {
			return err
		}
		cube.Children = &children
		return nil
	case OCTSAV_LODCUB:
		hasChildren = true
		break
	case OCTSAV_EMPTY:
		// TODO emptyfaces
		break
	case OCTSAV_SOLID:
		// TODO solidfaces
		break
	case OCTSAV_NORMAL:
		unpack.Read(&cube.Edges)
		break
	}

	if (octsav & 0x7) > 4 {
		log.Fatal().Msg("Map had invalid octsav")
		return errors.New("Map had invalid octsav")
	}

	for i := 0; i < 6; i++ {
		if mapVersion < 14 {
			texture := unpack.Char()
			cube.Texture[i] = uint16(texture)
		} else {
			texture := unpack.Short()
			cube.Texture[i] = texture
		}
		//log.Printf("Texture[%d]=%d", i, cube.Texture[i])
	}

	if mapVersion < 7 {
		unpack.Skip(3)
	} else if mapVersion <= 31 {
		mask := unpack.Char()

		if (mask & 0x80) > 0 {
			unpack.Skip(1)
		}

		surfaces := make([]SurfaceCompat, 12)
		normals := make([]NormalsCompat, 6)
		merges := make([]MergeCompat, 6)

		var numSurfaces = 6
		if (mask & 0x3F) > 0 {
			for i := 0; i < numSurfaces; i++ {
				if i >= 6 || mask&(1<<i) > 0 {
					unpack.Read(&surfaces[i])
					if i < 6 {
						if (mask & 0x40) > 0 {
							unpack.Read(&normals[i])
						}
						if (surfaces[i].Layer & 2) > 0 {
							numSurfaces++
						}
					}
				}
			}
		}

		if mapVersion >= 20 && (octsav&0x80) > 0 {
			merged := unpack.Char()
			cube.Merged = merged & 0x3F
			if (merged & 0x80) > 0 {
				mask := unpack.Char()
				if mask > 0 {
					for i := 0; i < 6; i++ {
						if (mask & (1 << i)) > 0 {
							unpack.Read(&merges[i])
						}
					}
				}
			}
		}
	} else {
		// TODO material
		if (octsav & 0x40) > 0 {
			if mapVersion <= 32 {
				unpack.Char()
			} else {
				unpack.Short()
			}
		}

		//fmt.Printf("a %d\n", unpack.Tell())

		// TODO merged
		if (octsav & 0x80) > 0 {
			unpack.Char()
		}

		if (octsav & 0x20) > 0 {
			surfMask := unpack.Char()
			unpack.Char() // totalVerts

			surfaces := make([]SurfaceInfo, 6)
			var offset byte
			offset = 0
			for i := 0; i < 6; i++ {
				if surfMask&(1<<i) == 0 {
					continue
				}

				unpack.Read(&surfaces[i])
				//fmt.Printf("%d %d %d %d\n", surfaces[i].Lmid[0], surfaces[i].Lmid[1], surfaces[i].Verts, surfaces[i].NumVerts)
				vertMask := surfaces[i].Verts
				numVerts := surfaces[i].TotalVerts()

				if numVerts == 0 {
					surfaces[i].Verts = 0
					continue
				}

				surfaces[i].Verts = offset
				offset += numVerts

				layerVerts := surfaces[i].NumVerts & MAXFACEVERTS
				hasXYZ := (vertMask & 0x04) != 0
				hasUV := (vertMask & 0x40) != 0
				hasNorm := (vertMask & 0x80) != 0

				//fmt.Printf("%d %t %t %t\n", vertMask, hasXYZ, hasUV, hasNorm)
				//fmt.Printf("b %d\n", unpack.Tell())

				if layerVerts == 4 {
					if hasXYZ && (vertMask&0x01) > 0 {
						unpack.Short()
						unpack.Short()
						unpack.Short()
						unpack.Short()
						hasXYZ = false
					}

					//fmt.Printf("b-1 %d\n", unpack.Tell())
					if hasUV && (vertMask&0x02) > 0 {
						unpack.Short()
						unpack.Short()
						unpack.Short()
						unpack.Short()

						if (surfaces[i].NumVerts & LAYER_DUP) > 0 {
							unpack.Short()
							unpack.Short()
							unpack.Short()
							unpack.Short()
						}

						hasUV = false
					}
					//fmt.Printf("c-2 %d\n", unpack.Tell())
				}

				//fmt.Printf("c %d\n", unpack.Tell())

				if hasNorm && (vertMask&0x08) > 0 {
					unpack.Short()
					hasNorm = false
				}

				if hasXYZ || hasUV || hasNorm {
					for k := 0; k < int(layerVerts); k++ {
						if hasXYZ {
							unpack.Short()
							unpack.Short()
						}

						if hasUV {
							unpack.Short()
							unpack.Short()
						}

						if hasNorm {
							unpack.Short()
						}
					}
				}

				if (surfaces[i].NumVerts & LAYER_DUP) > 0 {
					for k := 0; k < int(layerVerts); k++ {
						if hasUV {
							unpack.Short()
							unpack.Short()
						}
					}
				}
			}
		}
	}

	if hasChildren {
		children, _ := LoadChildren(unpack, mapVersion)
		cube.Children = &children
	}

	return nil
}

func LoadChildren(unpack *Unpacker, mapVersion int32) ([]Cube, error) {
	children := make([]Cube, CUBE_FACTOR)

	for i := 0; i < CUBE_FACTOR; i++ {
		err := LoadCube(unpack, &children[i], mapVersion)
		if err != nil {
			return nil, err
		}
	}

	return children, nil
}

func LoadVSlot(unpack *Unpacker, slot *VSlot, changed int32) error {
	slot.Changed = changed
	if (changed & (1 << VSLOT_SHPARAM)) > 0 {
		numParams := unpack.Short()

		for i := 0; i < int(numParams); i++ {
			_ = unpack.StringByte()
			// TODO vslots
			for k := 0; k < 4; k++ {
				unpack.Float()
			}
		}
	}

	if (changed & (1 << VSLOT_SCALE)) > 0 {
		unpack.Float()
	}

	if (changed & (1 << VSLOT_ROTATION)) > 0 {
		unpack.Int()
	}

	if (changed & (1 << VSLOT_OFFSET)) > 0 {
		unpack.Int()
		unpack.Int()
	}

	if (changed & (1 << VSLOT_SCROLL)) > 0 {
		unpack.Float()
		unpack.Float()
	}

	if (changed & (1 << VSLOT_LAYER)) > 0 {
		slot.Layer = unpack.Int()
	}

	if (changed & (1 << VSLOT_ALPHA)) > 0 {
		unpack.Float()
		unpack.Float()
	}

	if (changed & (1 << VSLOT_COLOR)) > 0 {
		for k := 0; k < 3; k++ {
			unpack.Float()
		}
	}

	return nil
}

func LoadVSlots(unpack *Unpacker, numVSlots int32) (VSlotData, error) {
	leftToRead := numVSlots

	vslots := make([]*VSlot, 0)
	prev := make([]int32, numVSlots)

	addSlot := func() *VSlot {
		vslot := VSlot{}
		vslot.Index = int32(len(vslots))
		vslots = append(vslots, &vslot)
		return &vslot
	}

	for leftToRead > 0 {
		changed := unpack.Int()
		if changed < 0 {
			for i := 0; i < int(-1*changed); i++ {
				addSlot()
			}
			leftToRead += changed
		} else {
			prev[len(vslots)] = unpack.Int()
			slot := addSlot()
			LoadVSlot(unpack, slot, changed)
			leftToRead--
		}
	}

	return VSlotData{
		Slots:    vslots,
		Previous: prev,
	}, nil
}

func InsideWorld(size int32, vector Vector) bool {
	return vector.X >= 0 && vector.X < float32(size) && vector.Y >= 0 && vector.Y < float32(size) && vector.Z >= 0 && vector.Z < float32(size)
}

func LoadMap(filename string) (*GameMap, error) {
	file, err := os.Open(filename)

	if err != nil {
		log.Fatal().Err(err).Msg("could not open file")
	}

	gz, err := gzip.NewReader(file)

	if err != nil {
		log.Fatal().Err(err).Msg("could not read gzip")
	}

	defer file.Close()
	defer gz.Close()

	gameMap := GameMap{}

	// Read the entire file into memory -- maps are small
	buffer, err := io.ReadAll(gz)

	if err == gzip.ErrChecksum {
		log.Printf("Map file had invalid checksum")
	} else if err != nil {
		return nil, err
	}

	reader := bytes.NewReader(buffer)

	unpack := NewUnpacker(reader)

	header := FileHeader{}
	err = unpack.Read(&header)
	if err != nil {
		return nil, err
	}

	newFooter := NewFooter{}
	oldFooter := OldFooter{}
	if header.Version <= 28 {
		reader.Seek(28, io.SeekStart) // 7 * 4, like in worldio.cpp
		err = unpack.Read(&oldFooter)
		if err != nil {
			return nil, err
		}

		newFooter.BlendMap = int32(oldFooter.BlendMap)
		newFooter.NumVars = 0
		newFooter.NumVSlots = 0
	} else {
		unpack.Read(&newFooter)

		if header.Version <= 29 {
			newFooter.NumVSlots = 0
		}

		// v29 had one fewer field
		if header.Version == 29 {
			reader.Seek(-4, io.SeekCurrent)
		}
	}

	mapHeader := Header{}

	mapHeader.Version = header.Version
	mapHeader.HeaderSize = header.HeaderSize
	mapHeader.WorldSize = header.WorldSize
	mapHeader.LightMaps = header.LightMaps
	mapHeader.BlendMap = newFooter.BlendMap
	mapHeader.NumVars = newFooter.NumVars
	mapHeader.NumVSlots = newFooter.NumVSlots

	gameMap.Header = mapHeader

	log.Printf("Version %d", header.Version)
	gameMap.Vars = make(map[string]Variable)

	// These are apparently arbitrary Sauerbraten variables a map can set
	for i := 0; i < int(newFooter.NumVars); i++ {
		_type := unpack.Char()
		name := unpack.String()

		switch VariableType(_type) {
		case VariableTypeInt:
			value := unpack.Int()
			gameMap.Vars[name] = IntVariable(value)
			//log.Printf("%s=%d", name, value)
		case VariableTypeFloat:
			value := unpack.Float()
			gameMap.Vars[name] = FloatVariable(value)
			//log.Printf("%s=%f", name, value)
		case VariableTypeString:
			value := unpack.String()
			gameMap.Vars[name] = StringVariable(value)
			//log.Printf("%s=%s", name, value)
		}
	}

	gameType := "fps"
	if header.Version >= 16 {
		gameType = unpack.StringByte()
	}
	//log.Printf("GameType %s", gameType)

	mapHeader.GameType = gameType

	// We just skip extras
	var eif uint16 = 0
	if header.Version >= 16 {
		eif = unpack.Short()
		extraBytes := unpack.Short()
		unpack.Skip(int64(extraBytes))
	}

	// Also skip the texture MRU
	if header.Version < 14 {
		unpack.Skip(256)
	} else {
		numMRUBytes := unpack.Short()
		unpack.Skip(int64(numMRUBytes * 2))
	}

	entities := make([]Entity, header.NumEnts)

	// Load entities
	for i := 0; i < int(header.NumEnts); i++ {
		entity := Entity{}

		unpack.Read(&entity)

		if gameType != "fps" {
			if eif > 0 {
				unpack.Skip(int64(eif))
			}
		}

		if !InsideWorld(header.WorldSize, entity.Position) {
			log.Printf("Entity outside of world")
			log.Printf("entity type %d", entity.Type)
			log.Printf("entity pos x=%f,y=%f,z=%f", entity.Position.X, entity.Position.Y, entity.Position.Z)
		}

		if header.Version <= 14 && entity.Type == ET_MAPMODEL {
			entity.Position.Z += float32(entity.Attr3)
			entity.Attr3 = 0

			if entity.Attr4 > 0 {
				log.Printf("warning: mapmodel ent (index %d) uses texture slot %d", i, entity.Attr4)
			}

			entity.Attr4 = 0
		}

		entities[i] = entity
	}

	gameMap.Entities = entities

	vSlotData, err := LoadVSlots(unpack, newFooter.NumVSlots)
	gameMap.VSlots = vSlotData

	cube, err := LoadChildren(unpack, header.Version)
	if err != nil {
		return nil, err
	}

	gameMap.Cubes = cube

	return &gameMap, nil
}
