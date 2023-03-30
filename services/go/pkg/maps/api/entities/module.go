package entities

import (
	"encoding/json"
	//"reflect"

	C "github.com/cfoust/sour/pkg/game/constants"
)

type EntityInfo interface {
	Type() C.EntityType
}

type Color struct {
	R byte
	G byte
	B byte
}

type Vector struct {
	X float32
	Y float32
	Z float32
}

type Entity struct {
	Position Vector
	Info     EntityInfo
}

func (e *Entity) MarshalJSON() ([]byte, error) {
	result := make(map[string]interface{})
	info := e.Info
	result["type"] = e.Info.Type().String()
	result["position"] = e.Position

	infoData, err := json.Marshal(info)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(infoData, &result)
	if err != nil {
		return nil, err
	}

	return json.Marshal(result)
}

func (e *Entity) UnmarshalJSON(data []byte) error {
	var obj map[string]*json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}

	err := json.Unmarshal(*obj["position"], &e.Position)
	if err != nil {
		return err
	}

	var type_ C.EntityType
	err = json.Unmarshal(*obj["type"], &type_)
	if err != nil {
		return err
	}

	var info EntityInfo = nil
	switch type_ {
	case C.EntityTypeLight:
		info = &Light{}
	case C.EntityTypeMapModel:
		info = &MapModel{}
	case C.EntityTypePlayerStart:
		info = &PlayerStart{}
	case C.EntityTypeEnvMap:
		info = &EnvMap{}
	case C.EntityTypeParticles:
		info = &Particles{}
	case C.EntityTypeSound:
		info = &Sound{}
	case C.EntityTypeSpotlight:
		info = &Spotlight{}
	case C.EntityTypeShells:
		info = &Shells{}
	case C.EntityTypeBullets:
		info = &Bullets{}
	case C.EntityTypeRockets:
		info = &Rockets{}
	case C.EntityTypeRounds:
		info = &Rounds{}
	case C.EntityTypeGrenades:
		info = &Grenades{}
	case C.EntityTypeCartridges:
		info = &Cartridges{}
	case C.EntityTypeHealth:
		info = &Health{}
	case C.EntityTypeBoost:
		info = &Boost{}
	case C.EntityTypeGreenArmour:
		info = &GreenArmour{}
	case C.EntityTypeYellowArmour:
		info = &YellowArmour{}
	case C.EntityTypeQuad:
		info = &Quad{}
	case C.EntityTypeTeleport:
		info = &Teleport{}
	case C.EntityTypeTeledest:
		info = &Teledest{}
	case C.EntityTypeMonster:
		info = &Monster{}
	case C.EntityTypeCarrot:
		info = &Carrot{}
	case C.EntityTypeJumpPad:
		info = &JumpPad{}
	case C.EntityTypeBase:
		info = &Base{}
	case C.EntityTypeRespawnPoint:
		info = &RespawnPoint{}
	case C.EntityTypeBox:
		info = &Box{}
	case C.EntityTypeBarrel:
		info = &Barrel{}
	case C.EntityTypePlatform:
		info = &Platform{}
	case C.EntityTypeElevator:
		info = &Elevator{}
	case C.EntityTypeFlag:
		info = &Flag{}
	}

	err = json.Unmarshal(data, info)
	if err != nil {
		return err
	}

	return nil
}

type Light struct {
	Radius int16
	Color  Color
}

func (e *Light) Type() C.EntityType { return C.EntityTypeLight }

type MapModel struct {
	Angle int16
	Index int16
}

func (m *MapModel) Type() C.EntityType { return C.EntityTypeMapModel }

type PlayerStart struct {
	Angle int16
	Tag   int16
}

func (e *PlayerStart) Type() C.EntityType { return C.EntityTypePlayerStart }

type EnvMap struct {
	Radius int16
	Size   int16
	Blur   int16
}

func (e *EnvMap) Type() C.EntityType { return C.EntityTypeEnvMap }

type Sound struct {
	Index int16
}

func (e *Sound) Type() C.EntityType { return C.EntityTypeSound }

type Spotlight struct {
	Radius int16
	Color  Color
}

func (e *Spotlight) Type() C.EntityType { return C.EntityTypeSpotlight }

type Powerup struct{}
type Shells Powerup
type Bullets Powerup
type Rockets Powerup
type Rounds Powerup
type Grenades Powerup
type Cartridges Powerup
type Health Powerup
type Boost Powerup
type GreenArmour Powerup
type YellowArmour Powerup
type Quad Powerup

func (e *Shells) Type() C.EntityType       { return C.EntityTypeShells }
func (e *Bullets) Type() C.EntityType      { return C.EntityTypeBullets }
func (e *Rockets) Type() C.EntityType      { return C.EntityTypeRockets }
func (e *Rounds) Type() C.EntityType       { return C.EntityTypeRounds }
func (e *Grenades) Type() C.EntityType     { return C.EntityTypeGrenades }
func (e *Cartridges) Type() C.EntityType   { return C.EntityTypeCartridges }
func (e *Health) Type() C.EntityType       { return C.EntityTypeHealth }
func (e *Boost) Type() C.EntityType        { return C.EntityTypeBoost }
func (e *GreenArmour) Type() C.EntityType  { return C.EntityTypeGreenArmour }
func (e *YellowArmour) Type() C.EntityType { return C.EntityTypeYellowArmour }
func (e *Quad) Type() C.EntityType         { return C.EntityTypeQuad }

type Teleport struct {
	Index int16
	Model int16
	Tag   int16
	Sound int16
}

func (e *Teleport) Type() C.EntityType { return C.EntityTypeTeleport }

type Teledest struct {
	Angle int16
	Tag   int16
}

func (e *Teledest) Type() C.EntityType { return C.EntityTypeTeledest }

type Monster struct {
	Angle int16
	Kind  int16
}

func (e *Monster) Type() C.EntityType { return C.EntityTypeMonster }

type Carrot struct {
	Tag  int16
	Kind int16
}

func (e *Carrot) Type() C.EntityType { return C.EntityTypeCarrot }

type JumpPad struct {
	PushZ int16
	PushX int16
	PushY int16
	Sound int16
}

func (e *JumpPad) Type() C.EntityType { return C.EntityTypeJumpPad }

type Base struct {
	Ammo int16
	Tag  int16
}

func (e *Base) Type() C.EntityType { return C.EntityTypeBase }

type RespawnPoint struct {
	Angle int16
	Spin  int16
}

func (e *RespawnPoint) Type() C.EntityType { return C.EntityTypeRespawnPoint }

type Box struct {
	Angle  int16
	Model  int16
	Weight int16
}

func (e *Box) Type() C.EntityType { return C.EntityTypeBox }

type Barrel struct {
	Angle  int16
	Model  int16
	Weight int16
	Health int16
}

func (e *Barrel) Type() C.EntityType { return C.EntityTypeBarrel }

type Platform struct {
	Angle int16
	Model int16
	Tag   int16
	Speed int16
}

func (e *Platform) Type() C.EntityType { return C.EntityTypePlatform }

type Elevator struct {
	Angle int16
	Model int16
	Tag   int16
	Speed int16
}

func (e *Elevator) Type() C.EntityType { return C.EntityTypeElevator }

type Flag struct {
	Angle int16
	Team  int16
}

func (e *Flag) Type() C.EntityType { return C.EntityTypeFlag }

var ENTITY_TYPES = []EntityInfo{
	&Light{},
	&MapModel{},
	&PlayerStart{},
	&EnvMap{},
	&Particles{},
	&Sound{},
	&Spotlight{},
	&Shells{},
	&Bullets{},
	&Rockets{},
	&Rounds{},
	&Grenades{},
	&Cartridges{},
	&Health{},
	&Boost{},
	&GreenArmour{},
	&YellowArmour{},
	&Quad{},
	&Teleport{},
	&Teledest{},
	&Carrot{},
	&JumpPad{},
	&Base{},
	&RespawnPoint{},
	&Box{},
	&Barrel{},
	&Platform{},
	&Elevator{},
	&Flag{},
}

var ENTITY_TYPE_MAP = map[C.EntityType]EntityInfo{}

func init() {
	for _, type_ := range ENTITY_TYPES {
		ENTITY_TYPE_MAP[type_.Type()] = type_
	}
}
