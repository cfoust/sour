package game

import (
	"fmt"
)

type VariableType byte

const (
	VariableTypeInt    VariableType = iota
	VariableTypeFloat               = iota
	VariableTypeString              = iota
)

type Variable interface {
	Type() VariableType
}

type IntVariable int32

func (v IntVariable) Type() VariableType {
	return VariableTypeInt
}

type FloatVariable float32

func (v FloatVariable) Type() VariableType {
	return VariableTypeFloat
}

const MAXSTRLEN = 260

type StringVariable string

func (v StringVariable) Type() VariableType {
	return VariableTypeString
}

// Constrain variables to their range of valid values
type VariableConstraint interface {
	Type() VariableType
}

type IntConstraint struct {
	Min     int32
	Default int32
	Max     int32
}

func (v IntConstraint) Type() VariableType {
	return VariableTypeInt
}

type FloatConstraint struct {
	Min     float32
	Default float32
	Max     float32
}

func (v FloatConstraint) Type() VariableType {
	return VariableTypeFloat
}

type StringConstraint struct {
	Default string
}

func (v StringConstraint) Type() VariableType {
	return VariableTypeString
}

var DEFAULT_VARIABLES = map[string]VariableConstraint{
	"ambient":           IntConstraint{1, 0x191919, 0xFFFFFF},
	"atmo":              IntConstraint{0, 0, 1},
	"atmoalpha":         FloatConstraint{0, 1, 1},
	"atmobright":        FloatConstraint{0, 1, 16},
	"atmodensity":       FloatConstraint{0, 1, 16},
	"atmohaze":          FloatConstraint{0, 0.1, 16},
	"atmoheight":        FloatConstraint{0.001, 1, 1000},
	"atmoozone":         FloatConstraint{0, 1, 16},
	"atmoplanetsize":    FloatConstraint{0.001, 1, 1000},
	"atmosundisk":       IntConstraint{0, 0, 0xFFFFFF},
	"atmosundiskbright": FloatConstraint{0, 1, 16},
	"atmosundiskcorona": FloatConstraint{0, 0.4, 1},
	"atmosundisksize":   FloatConstraint{0, 12, 90},
	"atmosunlight":      IntConstraint{0, 0, 0xFFFFFF},
	"atmosunlightscale": FloatConstraint{0, 1, 16},
	"blurlms":           IntConstraint{0, 0, 2},
	"blurskylight":      IntConstraint{0, 0, 2},
	"bumperror":         IntConstraint{1, 3, 16},
	"causticcontrast":   FloatConstraint{0, 0.6, 1},
	"causticmillis":     IntConstraint{0, 75, 1000},
	"causticscale":      IntConstraint{0, 50, 10000},
	"cloudalpha":        FloatConstraint{0, 1, 1},
	"cloudbox":          StringConstraint{""},
	"cloudboxalpha":     FloatConstraint{0, 1, 1},
	"cloudboxcolour":    IntConstraint{0, 0xFFFFFF, 0xFFFFFF},
	"cloudclip":         FloatConstraint{0, 0.5, 1},
	"cloudcolour":       IntConstraint{0, 0xFFFFFF, 0xFFFFFF},
	"cloudfade":         FloatConstraint{0, 0.2, 1},
	"cloudheight":       FloatConstraint{-1, 0.2, 1},
	"cloudlayer":        StringConstraint{""},
	"cloudoffsetx":      FloatConstraint{0, 0, 1},
	"cloudoffsety":      FloatConstraint{0, 0, 1},
	"cloudscale":        FloatConstraint{0.001, 1, 64},
	"cloudscrollx":      FloatConstraint{-16, 0, 16},
	"cloudscrolly":      FloatConstraint{-16, 0, 16},
	"cloudsubdiv":       IntConstraint{4, 16, 64},
	"envmapbb":          IntConstraint{0, 0, 1},
	"envmapradius":      IntConstraint{0, 128, 10000},
	"fog":               IntConstraint{16, 4000, 1000024},
	"fogcolour":         IntConstraint{0, 0x8099B3, 0xFFFFFF},
	"fogdomecap":        IntConstraint{0, 1, 1},
	"fogdomeclip":       FloatConstraint{0, 1, 1},
	"fogdomeclouds":     IntConstraint{0, 1, 1},
	"fogdomecolour":     IntConstraint{0, 0, 0xFFFFFF},
	"fogdomeheight":     FloatConstraint{-1, -0.5, 1},
	"fogdomemax":        FloatConstraint{0, 0, 1},
	"fogdomemin":        FloatConstraint{0, 0, 1},
	"glass2colour":      IntConstraint{0, 0x2080C0, 0xFFFFFF},
	"glass3colour":      IntConstraint{0, 0x2080C0, 0xFFFFFF},
	"glass4colour":      IntConstraint{0, 0x2080C0, 0xFFFFFF},
	"glasscolour":       IntConstraint{0, 0x2080C0, 0xFFFFFF},
	"grassalpha":        FloatConstraint{0, 1, 1},
	"grassanimmillis":   IntConstraint{0, 3000, 60000},
	"grassanimscale":    FloatConstraint{0, 0.03, 1},
	"grasscolour":       IntConstraint{0, 0xFFFFFF, 0xFFFFFF},
	"grassscale":        IntConstraint{1, 2, 64},
	"lava2colour":       IntConstraint{0, 0xFF4000, 0xFFFFFF},
	"lava2fog":          IntConstraint{0, 50, 10000},
	"lava3colour":       IntConstraint{0, 0xFF4000, 0xFFFFFF},
	"lava3fog":          IntConstraint{0, 50, 10000},
	"lava4colour":       IntConstraint{0, 0xFF4000, 0xFFFFFF},
	"lava4fog":          IntConstraint{0, 50, 10000},
	"lavacolour":        IntConstraint{0, 0xFF4000, 0xFFFFFF},
	"lavafog":           IntConstraint{0, 50, 10000},
	"lerpangle":         IntConstraint{0, 44, 180},
	"lerpsubdiv":        IntConstraint{0, 2, 4},
	"lerpsubdivsize":    IntConstraint{4, 4, 128},
	"lighterror":        IntConstraint{1, 8, 16},
	"lightlod":          IntConstraint{0, 0, 10},
	"lightprecision":    IntConstraint{1, 32, 1024},
	"maptitle":          StringConstraint{"Untitled Map by Unknown"},
	"mapversion":        IntConstraint{1, MAP_VERSION, 0},
	"minimapclip":       IntConstraint{0, 0, 1},
	"minimapcolour":     IntConstraint{0, 0, 0xFFFFFF},
	"minimapheight":     IntConstraint{0, 0, 2 << 16},
	"refractclear":      IntConstraint{0, 0, 1},
	"refractsky":        IntConstraint{0, 0, 1},
	"shadowmapambient":  IntConstraint{0, 0, 0xFFFFFF},
	"shadowmapangle":    IntConstraint{0, 0, 360},
	"skybox":            StringConstraint{""},
	"skyboxcolour":      IntConstraint{0, 0xFFFFFF, 0xFFFFFF},
	"skylight":          IntConstraint{0, 0, 0xFFFFFF},
	"skytexturelight":   IntConstraint{0, 1, 1},
	"spincloudlayer":    FloatConstraint{-720, 0, 720},
	"spinclouds":        FloatConstraint{-720, 0, 720},
	"spinsky":           FloatConstraint{-720, 0, 720},
	"skytexture":        IntConstraint{0, 0, 1},
	"sunlight":          IntConstraint{0, 0, 0xFFFFFF},
	"sunlightpitch":     IntConstraint{-90, 90, 90},
	"sunlightscale":     FloatConstraint{0, 1, 16},
	"sunlightyaw":       IntConstraint{0, 0, 360},
	"water2colour":      IntConstraint{0, 0x144650, 0xFFFFFF},
	"water2fallcolour":  IntConstraint{0, 0, 0xFFFFFF},
	"water2fog":         IntConstraint{0, 150, 10000},
	"water2spec":        IntConstraint{0, 150, 1000},
	"water3colour":      IntConstraint{0, 0x144650, 0xFFFFFF},
	"water3fallcolour":  IntConstraint{0, 0, 0xFFFFFF},
	"water3fog":         IntConstraint{0, 150, 10000},
	"water3spec":        IntConstraint{0, 150, 1000},
	"water4colour":      IntConstraint{0, 0x144650, 0xFFFFFF},
	"water4fallcolour":  IntConstraint{0, 0, 0xFFFFFF},
	"water4fog":         IntConstraint{0, 150, 10000},
	"water4spec":        IntConstraint{0, 150, 1000},
	"watercolour":       IntConstraint{0, 0x144650, 0xFFFFFF},
	"waterfallcolour":   IntConstraint{0, 0, 0xFFFFFF},
	"waterfog":          IntConstraint{0, 150, 10000},
	"waterspec":         IntConstraint{0, 150, 1000},
	"yawcloudlayer":     IntConstraint{0, 0, 360},
	"yawclouds":         IntConstraint{0, 0, 360},
	"yawsky":            IntConstraint{0, 0, 360},
}

type Variables map[string]Variable

func (v Variables) SetInt(name string, value int32) error {
	constraint, ok := DEFAULT_VARIABLES[name]
	if !ok {
		return fmt.Errorf("variable '%s' is not a valid map variable", name)
	}

	if constraint.Type() != VariableTypeInt {
		return fmt.Errorf("variable '%s' is not an int", name)
	}

	intConstraint := constraint.(IntConstraint)

	clean := value
	if value < intConstraint.Min {
		clean = intConstraint.Min
	} else if value > intConstraint.Max {
		clean = intConstraint.Max
	}

	v[name] = IntVariable(clean)

	return nil
}

func (v Variables) SetFloat(name string, value float32) error {
	constraint, ok := DEFAULT_VARIABLES[name]
	if !ok {
		return fmt.Errorf("variable '%s' is not a valid map variable", name)
	}

	if constraint.Type() != VariableTypeFloat {
		return fmt.Errorf("variable '%s' is not a float", name)
	}

	floatConstraint := constraint.(FloatConstraint)

	clean := value
	if value < floatConstraint.Min {
		clean = floatConstraint.Min
	} else if value > floatConstraint.Max {
		clean = floatConstraint.Max
	}

	v[name] = FloatVariable(clean)

	return nil
}

func (v Variables) SetString(name string, value string) error {
	constraint, ok := DEFAULT_VARIABLES[name]
	if !ok {
		return fmt.Errorf("variable '%s' is not a valid map variable", name)
	}

	if constraint.Type() != VariableTypeString {
		return fmt.Errorf("variable '%s' is not a string", name)
	}

	clean := value
	if len(value) > MAXSTRLEN {
		clean = value[:MAXSTRLEN]
	}

	v[name] = StringVariable(clean)

	return nil
}

func (v Variables) Set(name string, value Variable) error {
	switch value.Type() {
	case VariableTypeInt:
		var_ := value.(IntVariable)
		return v.SetInt(name, int32(var_))
	case VariableTypeFloat:
		var_ := value.(FloatVariable)
		return v.SetFloat(name, float32(var_))
	case VariableTypeString:
		var_ := value.(StringVariable)
		return v.SetString(name, string(var_))
	}

	return fmt.Errorf("attempt to set invalid variable")
}
