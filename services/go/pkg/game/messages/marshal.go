package messages

import (
	"fmt"
	"math"
	"reflect"
	"strconv"

	"github.com/cfoust/sour/pkg/game"

	"github.com/rs/zerolog/log"
)

type Message interface {
	Type() game.MessageCode
	Data() interface{}
}

type RawMessage struct {
	code game.MessageCode
	data interface{}
}

func (m RawMessage) Type() game.MessageCode {
	return m.code
}

func (m RawMessage) Data() interface{} {
	return m.data
}

func getComponent(p *game.Packet, flags uint32, k uint32) float64 {
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

func vecFromYawPitch(yaw float64, pitch float64, move int, strafe int) game.Vector {
	m := game.Vector{}
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

func readPhysics(p *game.Packet) game.PhysicsState {
	d := game.PhysicsState{}

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

	falling := game.Vector{}
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
			falling = game.Vector{
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

func unmarshalStruct(p *game.Packet, type_ reflect.Type, value reflect.Value) error {
	if value.Kind() != reflect.Struct {
		return fmt.Errorf("cannot unmarshal non-struct")
	}

	if type_ == reflect.TypeOf(game.PhysicsState{}) {
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
					peekable := game.Packet(*p)

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

func Unmarshal(p *game.Packet, code game.MessageCode, message interface{}) (Message, error) {
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

func Read(b []byte) ([]Message, error) {
	messages := make([]Message, 0)
	p := game.Packet(b)

	for len(p) > 0 {
		type_, ok := p.GetInt()
		if !ok {
			return nil, fmt.Errorf("failed to read message")
		}

		code := game.MessageCode(type_)

		if code >= game.NUMMSG {
			return nil, fmt.Errorf("code %d is not in range of messages", code)
		}

		var message Message
		var err error
		switch code {
		case game.N_CONNECT:
			message, err = Unmarshal(&p, game.N_CONNECT, &Connect{})
		case game.N_SERVINFO:
			message, err = Unmarshal(&p, game.N_SERVINFO, &ServerInfo{})
		case game.N_WELCOME:
			message, err = Unmarshal(&p, game.N_WELCOME, &Welcome{})
		case game.N_AUTHCHAL:
			message, err = Unmarshal(&p, game.N_AUTHCHAL, &AuthChallenge{})
		case game.N_PONG:
			message, err = Unmarshal(&p, game.N_PONG, &Pong{})
		case game.N_POS:
			message, err = Unmarshal(&p, game.N_POS, &Pos{})
		case game.N_SERVMSG:
			message, err = Unmarshal(&p, game.N_SERVMSG, &ServerMessage{})
		case game.N_PAUSEGAME:
			message, err = Unmarshal(&p, game.N_PAUSEGAME, &PauseGame{})
		case game.N_TIMEUP:
			message, err = Unmarshal(&p, game.N_TIMEUP, &TimeUp{})
		case game.N_ANNOUNCE:
			message, err = Unmarshal(&p, game.N_ANNOUNCE, &Announce{})
		case game.N_MASTERMODE:
			message, err = Unmarshal(&p, game.N_MASTERMODE, &MasterMode{})
		case game.N_CDIS:
			message, err = Unmarshal(&p, game.N_CDIS, &ClientDisconnected{})
		case game.N_JUMPPAD:
			message, err = Unmarshal(&p, game.N_JUMPPAD, &JumpPad{})
		case game.N_TELEPORT:
			message, err = Unmarshal(&p, game.N_TELEPORT, &Teleport{})
		case game.N_SPECTATOR:
			message, err = Unmarshal(&p, game.N_SPECTATOR, &Spectator{})
		case game.N_SETTEAM:
			message, err = Unmarshal(&p, game.N_SETTEAM, &SetTeam{})
		case game.N_CURRENTMASTER:
			message, err = Unmarshal(&p, game.N_CURRENTMASTER, &CurrentMaster{})
		case game.N_MAPCHANGE:
			message, err = Unmarshal(&p, game.N_MAPCHANGE, &MapChange{})
		case game.N_TEAMINFO:
			message, err = Unmarshal(&p, game.N_TEAMINFO, &TeamInfo{})
		case game.N_INITCLIENT:
			message, err = Unmarshal(&p, game.N_INITCLIENT, &InitClient{})
		case game.N_SPAWNSTATE:
			message, err = Unmarshal(&p, game.N_SPAWNSTATE, &SpawnState{})
		case game.N_RESUME:
			message, err = Unmarshal(&p, game.N_RESUME, &Resume{})
		case game.N_INITFLAGS:
			message, err = Unmarshal(&p, game.N_INITFLAGS, &InitFlags{})
		case game.N_DROPFLAG:
			message, err = Unmarshal(&p, game.N_DROPFLAG, &DropFlag{})
		case game.N_SCOREFLAG:
			message, err = Unmarshal(&p, game.N_SCOREFLAG, &ScoreFlag{})
		case game.N_RETURNFLAG:
			message, err = Unmarshal(&p, game.N_RETURNFLAG, &ReturnFlag{})
		case game.N_TAKEFLAG:
			message, err = Unmarshal(&p, game.N_TAKEFLAG, &TakeFlag{})
		case game.N_RESETFLAG:
			message, err = Unmarshal(&p, game.N_RESETFLAG, &ResetFlag{})
		case game.N_INVISFLAG:
			message, err = Unmarshal(&p, game.N_INVISFLAG, &InvisFlag{})
		case game.N_BASES:
			message, err = Unmarshal(&p, game.N_BASES, &Bases{})
		case game.N_BASEINFO:
			message, err = Unmarshal(&p, game.N_BASEINFO, &BaseInfo{})
		case game.N_BASESCORE:
			message, err = Unmarshal(&p, game.N_BASESCORE, &BaseScore{})
		case game.N_REPAMMO:
			message, err = Unmarshal(&p, game.N_REPAMMO, &ReplenishAmmo{})
		case game.N_TRYSPAWN:
			message, err = Unmarshal(&p, game.N_TRYSPAWN, &TrySpawn{})
		case game.N_BASEREGEN:
			message, err = Unmarshal(&p, game.N_BASEREGEN, &BaseRegen{})
		case game.N_INITTOKENS:
			message, err = Unmarshal(&p, game.N_INITTOKENS, &InitTokens{})
		case game.N_TAKETOKEN:
			message, err = Unmarshal(&p, game.N_TAKETOKEN, &TakeToken{})
		case game.N_EXPIRETOKENS:
			message, err = Unmarshal(&p, game.N_EXPIRETOKENS, &ExpireTokens{})
		case game.N_DROPTOKENS:
			message, err = Unmarshal(&p, game.N_DROPTOKENS, &DropTokens{})
		case game.N_STEALTOKENS:
			message, err = Unmarshal(&p, game.N_STEALTOKENS, &StealTokens{})
		case game.N_DEPOSITTOKENS:
			message, err = Unmarshal(&p, game.N_DEPOSITTOKENS, &DepositTokens{})
		case game.N_ITEMLIST:
			message, err = Unmarshal(&p, game.N_ITEMLIST, &ItemList{})
		case game.N_ITEMSPAWN:
			message, err = Unmarshal(&p, game.N_ITEMSPAWN, &ItemSpawn{})
		case game.N_ITEMACC:
			message, err = Unmarshal(&p, game.N_ITEMACC, &ItemAck{})
		case game.N_CLIPBOARD:
			message, err = Unmarshal(&p, game.N_CLIPBOARD, &Clipboard{})
		case game.N_EDITF:
			message, err = Unmarshal(&p, game.N_EDITF, &Editf{})
		case game.N_EDITT:
			message, err = Unmarshal(&p, game.N_EDITT, &Editt{})
		case game.N_EDITM:
			message, err = Unmarshal(&p, game.N_EDITM, &Editm{})
		case game.N_FLIP:
			message, err = Unmarshal(&p, game.N_FLIP, &Flip{})
		case game.N_COPY:
			message, err = Unmarshal(&p, game.N_COPY, &Copy{})
		case game.N_PASTE:
			message, err = Unmarshal(&p, game.N_PASTE, &Paste{})
		case game.N_ROTATE:
			message, err = Unmarshal(&p, game.N_ROTATE, &Rotate{})
		case game.N_REPLACE:
			message, err = Unmarshal(&p, game.N_REPLACE, &Replace{})
		case game.N_DELCUBE:
			message, err = Unmarshal(&p, game.N_DELCUBE, &Delcube{})
		case game.N_REMIP:
			message, err = Unmarshal(&p, game.N_REMIP, &Remip{})
		case game.N_EDITENT:
			message, err = Unmarshal(&p, game.N_EDITENT, &EditEnt{})
		case game.N_HITPUSH:
			message, err = Unmarshal(&p, game.N_HITPUSH, &HitPush{})
		case game.N_SHOTFX:
			message, err = Unmarshal(&p, game.N_SHOTFX, &ShotFX{})
		case game.N_EXPLODEFX:
			message, err = Unmarshal(&p, game.N_EXPLODEFX, &ExplodeFX{})
		case game.N_DAMAGE:
			message, err = Unmarshal(&p, game.N_DAMAGE, &Damage{})
		case game.N_DIED:
			message, err = Unmarshal(&p, game.N_DIED, &Died{})
		case game.N_FORCEDEATH:
			message, err = Unmarshal(&p, game.N_FORCEDEATH, &ForceDeath{})
		case game.N_NEWMAP:
			message, err = Unmarshal(&p, game.N_NEWMAP, &NewMap{})
		case game.N_REQAUTH:
			message, err = Unmarshal(&p, game.N_REQAUTH, &ReqAuth{})
		case game.N_INITAI:
			message, err = Unmarshal(&p, game.N_INITAI, &InitAI{})
		case game.N_SENDDEMOLIST:
			message, err = Unmarshal(&p, game.N_SENDDEMOLIST, &SendDemoList{})
		case game.N_SENDDEMO:
			message, err = Unmarshal(&p, game.N_SENDDEMO, &SendDemo{})
		case game.N_CLIENT:
			message, err = Unmarshal(&p, game.N_CLIENT, &ClientInfo{})
		case game.N_SPAWN:
			message, err = Unmarshal(&p, game.N_SPAWN, &Spawn{})
		case game.N_SOUND:
			message, err = Unmarshal(&p, game.N_SOUND, &Sound{})
		case game.N_CLIENTPING:
			message, err = Unmarshal(&p, game.N_CLIENTPING, &ClientPing{})
		case game.N_TAUNT:
			message, err = Unmarshal(&p, game.N_TAUNT, &Taunt{})
		case game.N_GUNSELECT:
			message, err = Unmarshal(&p, game.N_GUNSELECT, &GunSelect{})
		case game.N_TEXT:
			message, err = Unmarshal(&p, game.N_TEXT, &Text{})
		case game.N_SAYTEAM:
			message, err = Unmarshal(&p, game.N_SAYTEAM, &SayTeam{})
		case game.N_SWITCHNAME:
			message, err = Unmarshal(&p, game.N_SWITCHNAME, &SwitchName{})
		case game.N_SWITCHMODEL:
			message, err = Unmarshal(&p, game.N_SWITCHMODEL, &SwitchModel{})
		case game.N_EDITMODE:
			message, err = Unmarshal(&p, game.N_EDITMODE, &EditMode{})
		default:
			return nil, fmt.Errorf("unhandled code %s", code.String())
		}

		if err != nil {
			return nil, err
		}

		messages = append(messages, message)
	}

	return messages, nil
}
