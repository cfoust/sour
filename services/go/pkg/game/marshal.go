package game

import (
	"fmt"
	"math"
	"reflect"
	"strconv"

	"github.com/rs/zerolog/log"
)

type Message interface {
	Type() MessageCode
	Data() interface{}
}

type RawMessage struct {
	code MessageCode
	data interface{}
}

func (m RawMessage) Type() MessageCode {
	return m.code
}

func (m RawMessage) Data() interface{} {
	return m.data
}

func getComponent(p *Packet, flags uint32, k uint32) float64 {
	r, _ := p.GetByte()
	n := int(r)
	r, _ = p.GetByte()
	n |= int(r) << 8
	if flags&(1<<k) > 0 {
		r, _ = p.GetByte()
		n |= int(r) << 16
		if n&0x800000 > 0 {
			n |= -1 << 24
		}
	}

	return float64(n)
}

func clamp(a int, b int, c int) int {
	if a < b {
		return b
	}
	if a > c {
		return c
	}

	return a
}

const RAD = math.Pi / 180.0

func vecFromYawPitch(yaw float64, pitch float64, move int, strafe int) Vector {
	m := Vector{}
	if move > 0 {
		m.X = float64(move) * -math.Sin(RAD*yaw)
		m.Y = float64(move) * math.Cos(RAD*yaw)
	} else {
		m.X = 0
		m.Y = 0
	}

	if pitch > 0 {
		m.X *= math.Cos(RAD * pitch)
		m.Y *= math.Cos(RAD * pitch)
		m.Z = float64(move) * math.Sin(RAD*pitch)
	} else {
		m.Z = 0
	}

	if strafe > 0 {
		m.X += float64(strafe) * math.Cos(RAD*yaw)
		m.Y += float64(strafe) * math.Sin(RAD*yaw)
	}
	return m
}

func readPhysics(p *Packet) PhysicsState {
	d := PhysicsState{}

	r, _ := p.GetByte()
	state := r
	flags, _ := p.GetUint()

	d.O.X = getComponent(p, flags, 0)
	d.O.Y = getComponent(p, flags, 1)
	d.O.Z = getComponent(p, flags, 2)

	r, _ = p.GetByte()
	dir := int(r)
	r, _ = p.GetByte()
	dir |= int(r) << 8
	yaw := dir % 360
	pitch := clamp(dir/360, 0, 180) - 90
	r, _ = p.GetByte()
	roll := clamp(int(r), 0, 180) - 90
	r, _ = p.GetByte()
	mag := int(r)
	if flags&(1<<3) > 0 {
		r, _ = p.GetByte()
		mag |= int(r) << 8
	}
	r, _ = p.GetByte()
	dir = int(r)
	r, _ = p.GetByte()
	dir |= int(r) << 8

	d.Velocity = vecFromYawPitch(float64(dir%360), float64(clamp(dir/360, 0, 180)-90), 1, 0)

	falling := Vector{}
	if flags&(1<<4) > 0 {
		r, _ = p.GetByte()
		mag := int(r)
		if flags&(1<<5) > 0 {
			r, _ = p.GetByte()
			mag |= int(r) << 8
		}

		if flags&(1<<6) > 0 {
			r, _ = p.GetByte()
			dir = int(r)
			r, _ = p.GetByte()
			dir |= int(r) << 8
			falling = vecFromYawPitch(float64(dir%360), float64(clamp(dir/360, 0, 180)-90), 1, 0)
		} else {
			falling = Vector{
				X: 0,
				Y: 0,
				Z: -1,
			}
		}
	}

	d.Falling = falling

	d.Yaw = yaw
	d.Pitch = pitch
	d.Roll = roll

	if (state>>4)&2 > 0 {
		d.Move = -1
	} else {
		d.Move = (int(state) >> 4) & 1
	}

	if (state>>6)&2 > 0 {
		d.Strafe = -1
	} else {
		d.Strafe = (int(state) >> 6) & 1
	}

	d.State = state & 7

	return d
}

func unmarshalStruct(p *Packet, type_ reflect.Type, value reflect.Value) error {
	if value.Kind() != reflect.Struct {
		return fmt.Errorf("cannot unmarshal non-struct")
	}

	if type_ == reflect.TypeOf(PhysicsState{}) {
		state := readPhysics(p)
		value.Set(reflect.ValueOf(state))
		return nil
	}

	for i := 0; i < type_.NumField(); i++ {
		field := type_.Field(i)
		fieldValue := value.Field(i)
		ref := fmt.Sprintf("%s.%s", type_.Name(), field.Name)

		log.Debug().Str("ref", ref).Str("kind", field.Type.Kind().String()).Msg("parsing field")

		switch field.Type.Kind() {
		case reflect.Slice:
			element := field.Type.Elem()
			tag := field.Tag
			if len(tag) == 0 {
				return fmt.Errorf("all arrays must specify tag")
			}

			endType, haveType := field.Tag.Lookup("type")
			if !haveType {
				return fmt.Errorf("all arrays must specify a type in the tag")
			}

			slice := reflect.MakeSlice(field.Type, 0, 0)

			switch endType {
			// There is some condition that indicates the array is done
			case "term":
				cmp, haveCmp := field.Tag.Lookup("cmp")
				if !haveCmp {
					return fmt.Errorf("term tags must specify end condition")
				}

				for {
					peekable := Packet(*p)

					done := false
					switch cmp {
					case "gez":
						endValue, ok := peekable.GetInt()
						if !ok {
							return fmt.Errorf("failed to read int condition")
						}

						if endValue < 0 {
							p.GetInt()
							done = true
							break
						}
					case "len":
						endValue, ok := peekable.GetString()
						if !ok {
							return fmt.Errorf("failed to read string condition")
						}

						if len(endValue) == 0 {
							p.GetString()
							done = true
							break
						}
					}

					if done {
						break
					}

					entry := reflect.New(element)
					err := unmarshalStruct(p, element, entry.Elem())
					if err != nil {
						return err
					}

					reflect.Append(slice, entry.Elem())
				}
			case "count":
				number, haveConst := field.Tag.Lookup("const")
				var numElements int
				if haveConst {
					numElements, _ = strconv.Atoi(number)
				} else {
					readElements, ok := p.GetInt()
					if !ok {
						return fmt.Errorf("failed to read number of elements")
					}
					numElements = int(readElements)
				}

				for i := 0; i < numElements; i++ {
					entry := reflect.New(element)
					err := unmarshalStruct(p, element, entry.Elem())
					if err != nil {
						return err
					}

					reflect.Append(slice, entry.Elem())
				}
				break
			default:
				return fmt.Errorf("unhandled end type: %s", endType)
			}

			fieldValue.Set(slice)

		case reflect.Int:
			readValue, ok := p.GetInt()
			if !ok {
				return fmt.Errorf("error reading int %s", ref)
			}
			fieldValue.SetInt(int64(readValue))
		case reflect.String:
			readValue, ok := p.GetString()
			if !ok {
				return fmt.Errorf("error reading string %s", ref)
			}
			fieldValue.SetString(readValue)
		case reflect.Struct:
			err := unmarshalStruct(p, field.Type, fieldValue)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func Unmarshal(p *Packet, code MessageCode, message interface{}) (Message, error) {
	raw := RawMessage{
		code: code,
		data: message,
	}

	type_ := reflect.TypeOf(message)
	value := reflect.ValueOf(message)

	if value.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("can't unmarshal non-pointer")
	}

	log.Debug().Str("type", type_.Elem().Name()).Msg("parsing message")
	err := unmarshalStruct(p, type_.Elem(), value.Elem())

	return raw, err
}

func Read(b []byte) (*[]Message, error) {
	messages := make([]Message, 0)
	p := Packet(b)

	for len(p) > 0 {
		type_, ok := p.GetInt()
		if !ok {
			return nil, fmt.Errorf("failed to read message")
		}

		code := MessageCode(type_)

		if code >= NUMMSG {
			return nil, fmt.Errorf("code %d is not in range of messages", code)
		}

		var message Message
		var err error
		switch code {
		case N_CONNECT:
			message, err = Unmarshal(&p, N_CONNECT, &Connect{})
		case N_SERVINFO:
			message, err = Unmarshal(&p, N_SERVINFO, &ServerInfo{})
		case N_WELCOME:
			message, err = Unmarshal(&p, N_WELCOME, &Welcome{})
		case N_AUTHCHAL:
			message, err = Unmarshal(&p, N_AUTHCHAL, &AuthChallenge{})
		case N_PONG:
			message, err = Unmarshal(&p, N_PONG, &Pong{})
		case N_POS:
			message, err = Unmarshal(&p, N_POS, &Pos{})
		case N_SERVMSG:
			message, err = Unmarshal(&p, N_SERVMSG, &ServerMessage{})
		case N_PAUSEGAME:
			message, err = Unmarshal(&p, N_PAUSEGAME, &PauseGame{})
		case N_TIMEUP:
			message, err = Unmarshal(&p, N_TIMEUP, &TimeUp{})
		case N_ANNOUNCE:
			message, err = Unmarshal(&p, N_ANNOUNCE, &Announce{})
		case N_MASTERMODE:
			message, err = Unmarshal(&p, N_MASTERMODE, &MasterMode{})
		case N_CDIS:
			message, err = Unmarshal(&p, N_CDIS, &ClientDisconnected{})
		case N_JUMPPAD:
			message, err = Unmarshal(&p, N_JUMPPAD, &JumpPad{})
		case N_TELEPORT:
			message, err = Unmarshal(&p, N_TELEPORT, &Teleport{})
		case N_SPECTATOR:
			message, err = Unmarshal(&p, N_SPECTATOR, &Spectator{})
		case N_SETTEAM:
			message, err = Unmarshal(&p, N_SETTEAM, &SetTeam{})
		case N_CURRENTMASTER:
			message, err = Unmarshal(&p, N_CURRENTMASTER, &CurrentMaster{})
		case N_MAPCHANGE:
			message, err = Unmarshal(&p, N_MAPCHANGE, &MapChange{})
		case N_TEAMINFO:
			message, err = Unmarshal(&p, N_TEAMINFO, &TeamInfo{})
		case N_INITCLIENT:
			message, err = Unmarshal(&p, N_INITCLIENT, &InitClient{})
		case N_SPAWNSTATE:
			message, err = Unmarshal(&p, N_SPAWNSTATE, &SpawnState{})
		case N_RESUME:
			message, err = Unmarshal(&p, N_RESUME, &Resume{})
		case N_INITFLAGS:
			message, err = Unmarshal(&p, N_INITFLAGS, &InitFlags{})
		case N_DROPFLAG:
			message, err = Unmarshal(&p, N_DROPFLAG, &DropFlag{})
		case N_SCOREFLAG:
			message, err = Unmarshal(&p, N_SCOREFLAG, &ScoreFlag{})
		case N_RETURNFLAG:
			message, err = Unmarshal(&p, N_RETURNFLAG, &ReturnFlag{})
		case N_TAKEFLAG:
			message, err = Unmarshal(&p, N_TAKEFLAG, &TakeFlag{})
		case N_RESETFLAG:
			message, err = Unmarshal(&p, N_RESETFLAG, &ResetFlag{})
		case N_INVISFLAG:
			message, err = Unmarshal(&p, N_INVISFLAG, &InvisFlag{})
		case N_BASES:
			message, err = Unmarshal(&p, N_BASES, &Bases{})
		case N_BASEINFO:
			message, err = Unmarshal(&p, N_BASEINFO, &BaseInfo{})
		case N_BASESCORE:
			message, err = Unmarshal(&p, N_BASESCORE, &BaseScore{})
		case N_REPAMMO:
			message, err = Unmarshal(&p, N_REPAMMO, &ReplenishAmmo{})
		case N_TRYSPAWN:
			message, err = Unmarshal(&p, N_TRYSPAWN, &TrySpawn{})
		case N_BASEREGEN:
			message, err = Unmarshal(&p, N_BASEREGEN, &BaseRegen{})
		case N_INITTOKENS:
			message, err = Unmarshal(&p, N_INITTOKENS, &InitTokens{})
		case N_TAKETOKEN:
			message, err = Unmarshal(&p, N_TAKETOKEN, &TakeToken{})
		case N_EXPIRETOKENS:
			message, err = Unmarshal(&p, N_EXPIRETOKENS, &ExpireTokens{})
		case N_DROPTOKENS:
			message, err = Unmarshal(&p, N_DROPTOKENS, &DropTokens{})
		case N_STEALTOKENS:
			message, err = Unmarshal(&p, N_STEALTOKENS, &StealTokens{})
		case N_DEPOSITTOKENS:
			message, err = Unmarshal(&p, N_DEPOSITTOKENS, &DepositTokens{})
		case N_ITEMLIST:
			message, err = Unmarshal(&p, N_ITEMLIST, &ItemList{})
		case N_ITEMSPAWN:
			message, err = Unmarshal(&p, N_ITEMSPAWN, &ItemSpawn{})
		case N_ITEMACC:
			message, err = Unmarshal(&p, N_ITEMACC, &ItemAck{})
		case N_CLIPBOARD:
			message, err = Unmarshal(&p, N_CLIPBOARD, &Clipboard{})
		case N_EDITF:
			message, err = Unmarshal(&p, N_EDITF, &Editf{})
		case N_EDITT:
			message, err = Unmarshal(&p, N_EDITT, &Editt{})
		case N_EDITM:
			message, err = Unmarshal(&p, N_EDITM, &Editm{})
		case N_FLIP:
			message, err = Unmarshal(&p, N_FLIP, &Flip{})
		case N_COPY:
			message, err = Unmarshal(&p, N_COPY, &Copy{})
		case N_PASTE:
			message, err = Unmarshal(&p, N_PASTE, &Paste{})
		case N_ROTATE:
			message, err = Unmarshal(&p, N_ROTATE, &Rotate{})
		case N_REPLACE:
			message, err = Unmarshal(&p, N_REPLACE, &Replace{})
		case N_DELCUBE:
			message, err = Unmarshal(&p, N_DELCUBE, &Delcube{})
		case N_REMIP:
			message, err = Unmarshal(&p, N_REMIP, &Remip{})
		case N_EDITENT:
			message, err = Unmarshal(&p, N_EDITENT, &EditEnt{})
		case N_HITPUSH:
			message, err = Unmarshal(&p, N_HITPUSH, &HitPush{})
		case N_SHOTFX:
			message, err = Unmarshal(&p, N_SHOTFX, &ShotFX{})
		case N_EXPLODEFX:
			message, err = Unmarshal(&p, N_EXPLODEFX, &ExplodeFX{})
		case N_DAMAGE:
			message, err = Unmarshal(&p, N_DAMAGE, &Damage{})
		case N_DIED:
			message, err = Unmarshal(&p, N_DIED, &Died{})
		case N_FORCEDEATH:
			message, err = Unmarshal(&p, N_FORCEDEATH, &ForceDeath{})
		case N_NEWMAP:
			message, err = Unmarshal(&p, N_NEWMAP, &NewMap{})
		case N_REQAUTH:
			message, err = Unmarshal(&p, N_REQAUTH, &ReqAuth{})
		case N_INITAI:
			message, err = Unmarshal(&p, N_INITAI, &InitAI{})
		case N_SENDDEMOLIST:
			message, err = Unmarshal(&p, N_SENDDEMOLIST, &SendDemoList{})
		case N_SENDDEMO:
			message, err = Unmarshal(&p, N_SENDDEMO, &SendDemo{})
		case N_CLIENT:
			message, err = Unmarshal(&p, N_CLIENT, &ClientInfo{})
		case N_SPAWN:
			message, err = Unmarshal(&p, N_SPAWN, &Spawn{})
		case N_SOUND:
			message, err = Unmarshal(&p, N_SOUND, &Sound{})
		case N_CLIENTPING:
			message, err = Unmarshal(&p, N_CLIENTPING, &ClientPing{})
		case N_TAUNT:
			message, err = Unmarshal(&p, N_TAUNT, &Taunt{})
		case N_GUNSELECT:
			message, err = Unmarshal(&p, N_GUNSELECT, &GunSelect{})
		case N_TEXT:
			message, err = Unmarshal(&p, N_TEXT, &Text{})
		case N_SAYTEAM:
			message, err = Unmarshal(&p, N_SAYTEAM, &SayTeam{})
		case N_SWITCHNAME:
			message, err = Unmarshal(&p, N_SWITCHNAME, &SwitchName{})
		case N_SWITCHMODEL:
			message, err = Unmarshal(&p, N_SWITCHMODEL, &SwitchModel{})
		case N_EDITMODE:
			message, err = Unmarshal(&p, N_EDITMODE, &EditMode{})
		default:
			return nil, fmt.Errorf("unhandled code %s", code.String())
		}

		if err != nil {
			return nil, err
		}

		messages = append(messages, message)
	}

	return &messages, nil
}
