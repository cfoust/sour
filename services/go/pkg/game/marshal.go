package game

import (
	"fmt"
	"reflect"

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

func unmarshalStruct(p *Packet, type_ reflect.Type, value reflect.Value) error {
	if value.Kind() != reflect.Struct {
		return fmt.Errorf("cannot unmarshal non-struct")
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

					switch cmp {
					case "gez":
						endValue, ok := peekable.GetInt()
						if !ok {
							return fmt.Errorf("failed to read int condition")
						}

						if endValue < 0 {
							p.GetInt()
							break
						}
					case "len":
						endValue, ok := peekable.GetString()
						if !ok {
							return fmt.Errorf("failed to read string condition")
						}

						if len(endValue) == 0 {
							p.GetString()
							break
						}
					}

					entry := reflect.New(element)
					log.Info().Msgf("entry kind %s", entry.Elem().Kind().String())
					err := unmarshalStruct(p, element, entry.Elem())
					if err != nil {
						return err
					}
				}
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
