package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
	"strconv"

	"github.com/rs/zerolog/log"
)

type Message interface {
	Type() MessageCode
	Contents() interface{}
	Data() []byte
}

type RawMessage struct {
	code    MessageCode
	message interface{}
	data    []byte
}

func (m RawMessage) Type() MessageCode {
	return m.code
}

func (m RawMessage) Contents() interface{} {
	return m.message
}

func (m RawMessage) Data() []byte {
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

func unmarshalValue(p *Packet, type_ reflect.Type, value reflect.Value) error {
	switch type_.Kind() {
	case reflect.Int32:
		fallthrough
	case reflect.Int:
		readValue, ok := p.GetInt()
		if !ok {
			return fmt.Errorf("error reading int")
		}
		value.SetInt(int64(readValue))
	case reflect.Uint8:
		readValue, ok := p.GetByte()
		if !ok {
			return fmt.Errorf("error reading byte")
		}
		value.SetUint(uint64(readValue))
	case reflect.Bool:
		readValue, ok := p.GetInt()
		if !ok {
			return fmt.Errorf("error reading bool")
		}
		if readValue == 1 {
			value.SetBool(true)
		} else {
			value.SetBool(false)
		}
	case reflect.Uint:
		readValue, ok := p.GetUint()
		if !ok {
			return fmt.Errorf("error reading uint")
		}
		value.SetUint(uint64(readValue))
	case reflect.String:
		readValue, ok := p.GetString()
		if !ok {
			return fmt.Errorf("error reading string")
		}
		value.SetString(readValue)
	case reflect.Struct:
		err := unmarshalStruct(p, type_, value)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unimplemented type: %s", type_.String())
	}

	return nil
}

func unmarshalRawValue(p *Packet, type_ reflect.Type, value interface{}) error {
	var err error
	switch v := value.(type) {
	default:
		err = binary.Read(p, binary.LittleEndian, v)
	}

	if err != nil {
		return err
	}

	return nil
}

func Unmarshal(p *Packet, compressed bool, pieces ...interface{}) error {
	for _, piece := range pieces {
		type_ := reflect.TypeOf(piece).Elem()
		value := reflect.ValueOf(piece).Elem()

		var err error
		if compressed {
			err = unmarshalValue(p, type_, value)
		} else {
			err = unmarshalRawValue(p, type_, piece)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func marshalValue(p *Packet, type_ reflect.Type, value reflect.Value) error {
	switch type_.Kind() {
	case reflect.Int32:
		fallthrough
	case reflect.Int:
		p.PutInt(int32(value.Int()))
	case reflect.Uint8:
		p.PutInt(int32(value.Uint()))
	case reflect.Bool:
		boolean := value.Bool()
		if boolean {
			p.PutInt(1)
		} else {
			p.PutInt(0)
		}
	case reflect.Uint:
		p.PutUint(uint32(value.Uint()))
	case reflect.String:
		p.PutString(value.String())
	case reflect.Struct:
		for i := 0; i < type_.NumField(); i++ {
			field := type_.Field(i)
			fieldValue := value.Field(i)
			marshalValue(p, field.Type, fieldValue)
		}
	default:
		return fmt.Errorf("unimplemented type: %s", type_.String())
	}

	return nil
}

func marshalRawValue(p *Packet, type_ reflect.Type, value interface{}) error {
	var buffer bytes.Buffer

	var err error
	switch v := value.(type) {
	default:
		err = binary.Write(&buffer, binary.LittleEndian, v)
	}

	if err != nil {
		return err
	}

	*p = append(*p, buffer.Bytes()...)

	return nil
}

func Marshal(p *Packet, compressed bool, pieces ...interface{}) error {
	for _, piece := range pieces {
		type_ := reflect.TypeOf(piece)
		value := reflect.ValueOf(piece)

		var err error
		if compressed {
			err = marshalValue(p, type_, value)
		} else {
			err = marshalRawValue(p, type_, piece)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func UnmarshalMessage(p *Packet, code MessageCode, message interface{}) (Message, error) {
	before := *p

	// Throw away the type information (we got it already)
	p.GetInt()

	raw := RawMessage{
		code:    code,
		message: message,
	}

	type_ := reflect.TypeOf(message)
	value := reflect.ValueOf(message)

	if value.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("can't unmarshal non-pointer")
	}

	err := unmarshalStruct(p, type_.Elem(), value.Elem())

	after := *p

	raw.data = before[:len(before)-len(after)]

	return raw, err
}

func Read(b []byte, fromClient bool) ([]Message, error) {
	messages := make([]Message, 0)
	p := Packet(b)

	log.Debug().Msgf("reading messages from %d bytes", len(b))

	for len(p) > 0 {
		// We just want to peek this so that the message type int gets into the RawMessage
		q := Packet(p)
		type_, ok := q.GetInt()
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
		case N_ADDBOT:
			message, err = UnmarshalMessage(&p, N_ADDBOT, &AddBot{})
		case N_AUTHANS:
			message, err = UnmarshalMessage(&p, N_AUTHANS, &AuthAns{})
		case N_AUTHKICK:
			message, err = UnmarshalMessage(&p, N_AUTHKICK, &AuthKick{})
		case N_AUTHTRY:
			message, err = UnmarshalMessage(&p, N_AUTHTRY, &AuthTry{})
		case N_BOTBALANCE:
			message, err = UnmarshalMessage(&p, N_BOTBALANCE, &BotBalance{})
		case N_BOTLIMIT:
			message, err = UnmarshalMessage(&p, N_BOTLIMIT, &BotLimit{})
		case N_CHECKMAPS:
			message, err = UnmarshalMessage(&p, N_CHECKMAPS, &CheckMaps{})
		case N_CLEARBANS:
			message, err = UnmarshalMessage(&p, N_CLEARBANS, &ClearBans{})
		case N_CLEARDEMOS:
			message, err = UnmarshalMessage(&p, N_CLEARDEMOS, &ClearDemos{})
		case N_DELBOT:
			message, err = UnmarshalMessage(&p, N_DELBOT, &DelBot{})
		case N_DEMOPACKET:
			message, err = UnmarshalMessage(&p, N_DEMOPACKET, &DemoPacket{})
		case N_DEMOPLAYBACK:
			message, err = UnmarshalMessage(&p, N_DEMOPLAYBACK, &DemoPlayback{})
		//case N_EDITVAR:
		//message, err = UnmarshalMessage(&p, N_EDITVAR, &Editvar{})
		case N_EDITVSLOT:
			message, err = UnmarshalMessage(&p, N_EDITVSLOT, &EditVSlot{})
		case N_EXPLODE:
			message, err = UnmarshalMessage(&p, N_EXPLODE, &Explode{})
		case N_FORCEINTERMISSION:
			message, err = UnmarshalMessage(&p, N_FORCEINTERMISSION, &ForceIntermission{})
		//case N_FROMAI:
		//message, err = UnmarshalMessage(&p, N_FROMAI, &Fromai{})
		case N_GAMESPEED:
			message, err = UnmarshalMessage(&p, N_GAMESPEED, &GameSpeed{})
		case N_GETDEMO:
			message, err = UnmarshalMessage(&p, N_GETDEMO, &GetDemo{})
		case N_GETMAP:
			message, err = UnmarshalMessage(&p, N_GETMAP, &GetMap{})
		case N_ITEMPICKUP:
			message, err = UnmarshalMessage(&p, N_ITEMPICKUP, &ItemPickup{})
		case N_KICK:
			message, err = UnmarshalMessage(&p, N_KICK, &Kick{})
		case N_LISTDEMOS:
			message, err = UnmarshalMessage(&p, N_LISTDEMOS, &ListDemos{})
		case N_MAPCRC:
			message, err = UnmarshalMessage(&p, N_MAPCRC, &MapCRC{})
		case N_MAPVOTE:
			message, err = UnmarshalMessage(&p, N_MAPVOTE, &MapVote{})
		case N_RECORDDEMO:
			message, err = UnmarshalMessage(&p, N_RECORDDEMO, &RecordDemo{})
		//case N_REDO:
		//message, err = UnmarshalMessage(&p, N_REDO, &Redo{})
		//case N_SENDMAP:
		//message, err = UnmarshalMessage(&p, N_SENDMAP, &Sendmap{})
		case N_SERVCMD:
			message, err = UnmarshalMessage(&p, N_SERVCMD, &ServCMD{})
		case N_SETMASTER:
			message, err = UnmarshalMessage(&p, N_SETMASTER, &SetMaster{})
		case N_SHOOT:
			message, err = UnmarshalMessage(&p, N_SHOOT, &Shoot{})
		case N_STOPDEMO:
			message, err = UnmarshalMessage(&p, N_STOPDEMO, &StopDemo{})
		case N_SUICIDE:
			message, err = UnmarshalMessage(&p, N_SUICIDE, &Suicide{})
		case N_SWITCHTEAM:
			message, err = UnmarshalMessage(&p, N_SWITCHTEAM, &SwitchTeam{})
		case N_TRYDROPFLAG:
			message, err = UnmarshalMessage(&p, N_TRYDROPFLAG, &TryDropFlag{})
		//case N_UNDO:
		//message, err = UnmarshalMessage(&p, N_UNDO, &Undo{})
		case N_CONNECT:
			message, err = UnmarshalMessage(&p, N_CONNECT, &Connect{})
		case N_SERVINFO:
			message, err = UnmarshalMessage(&p, N_SERVINFO, &ServerInfo{})
		case N_WELCOME:
			message, err = UnmarshalMessage(&p, N_WELCOME, &Welcome{})
		case N_AUTHCHAL:
			message, err = UnmarshalMessage(&p, N_AUTHCHAL, &AuthChallenge{})
		case N_PONG:
			message, err = UnmarshalMessage(&p, N_PONG, &Pong{})
		case N_PING:
			message, err = UnmarshalMessage(&p, N_PING, &Ping{})
		case N_POS:
			message, err = UnmarshalMessage(&p, N_POS, &Pos{})
		case N_SERVMSG:
			message, err = UnmarshalMessage(&p, N_SERVMSG, &ServerMessage{})
		case N_PAUSEGAME:
			message, err = UnmarshalMessage(&p, N_PAUSEGAME, &PauseGame{})
		case N_TIMEUP:
			message, err = UnmarshalMessage(&p, N_TIMEUP, &TimeUp{})
		case N_ANNOUNCE:
			message, err = UnmarshalMessage(&p, N_ANNOUNCE, &Announce{})
		case N_MASTERMODE:
			message, err = UnmarshalMessage(&p, N_MASTERMODE, &MasterMode{})
		case N_CDIS:
			message, err = UnmarshalMessage(&p, N_CDIS, &ClientDisconnected{})
		case N_JUMPPAD:
			message, err = UnmarshalMessage(&p, N_JUMPPAD, &JumpPad{})
		case N_TELEPORT:
			message, err = UnmarshalMessage(&p, N_TELEPORT, &Teleport{})
		case N_SPECTATOR:
			message, err = UnmarshalMessage(&p, N_SPECTATOR, &Spectator{})
		case N_SETTEAM:
			message, err = UnmarshalMessage(&p, N_SETTEAM, &SetTeam{})
		case N_CURRENTMASTER:
			message, err = UnmarshalMessage(&p, N_CURRENTMASTER, &CurrentMaster{})
		case N_MAPCHANGE:
			message, err = UnmarshalMessage(&p, N_MAPCHANGE, &MapChange{})
		case N_TEAMINFO:
			message, err = UnmarshalMessage(&p, N_TEAMINFO, &TeamInfo{})
		case N_INITCLIENT:
			message, err = UnmarshalMessage(&p, N_INITCLIENT, &InitClient{})
		case N_SPAWNSTATE:
			message, err = UnmarshalMessage(&p, N_SPAWNSTATE, &SpawnState{})
		case N_RESUME:
			message, err = UnmarshalMessage(&p, N_RESUME, &Resume{})
		case N_INITFLAGS:
			message, err = UnmarshalMessage(&p, N_INITFLAGS, &InitFlags{})
		case N_DROPFLAG:
			message, err = UnmarshalMessage(&p, N_DROPFLAG, &DropFlag{})
		case N_SCOREFLAG:
			message, err = UnmarshalMessage(&p, N_SCOREFLAG, &ScoreFlag{})
		case N_RETURNFLAG:
			message, err = UnmarshalMessage(&p, N_RETURNFLAG, &ReturnFlag{})
		case N_TAKEFLAG:
			message, err = UnmarshalMessage(&p, N_TAKEFLAG, &TakeFlag{})
		case N_RESETFLAG:
			message, err = UnmarshalMessage(&p, N_RESETFLAG, &ResetFlag{})
		case N_INVISFLAG:
			message, err = UnmarshalMessage(&p, N_INVISFLAG, &InvisFlag{})
		case N_BASES:
			message, err = UnmarshalMessage(&p, N_BASES, &Bases{})
		case N_BASEINFO:
			message, err = UnmarshalMessage(&p, N_BASEINFO, &BaseInfo{})
		case N_BASESCORE:
			message, err = UnmarshalMessage(&p, N_BASESCORE, &BaseScore{})
		case N_REPAMMO:
			message, err = UnmarshalMessage(&p, N_REPAMMO, &ReplenishAmmo{})
		case N_TRYSPAWN:
			message, err = UnmarshalMessage(&p, N_TRYSPAWN, &TrySpawn{})
		case N_BASEREGEN:
			message, err = UnmarshalMessage(&p, N_BASEREGEN, &BaseRegen{})
		case N_INITTOKENS:
			message, err = UnmarshalMessage(&p, N_INITTOKENS, &InitTokens{})
		case N_TAKETOKEN:
			message, err = UnmarshalMessage(&p, N_TAKETOKEN, &TakeToken{})
		case N_EXPIRETOKENS:
			message, err = UnmarshalMessage(&p, N_EXPIRETOKENS, &ExpireTokens{})
		case N_DROPTOKENS:
			message, err = UnmarshalMessage(&p, N_DROPTOKENS, &DropTokens{})
		case N_STEALTOKENS:
			message, err = UnmarshalMessage(&p, N_STEALTOKENS, &StealTokens{})
		case N_DEPOSITTOKENS:
			message, err = UnmarshalMessage(&p, N_DEPOSITTOKENS, &DepositTokens{})
		case N_ITEMLIST:
			message, err = UnmarshalMessage(&p, N_ITEMLIST, &ItemList{})
		case N_ITEMSPAWN:
			message, err = UnmarshalMessage(&p, N_ITEMSPAWN, &ItemSpawn{})
		case N_ITEMACC:
			message, err = UnmarshalMessage(&p, N_ITEMACC, &ItemAck{})
		case N_CLIPBOARD:
			message, err = UnmarshalMessage(&p, N_CLIPBOARD, &Clipboard{})
		case N_EDITF:
			message, err = UnmarshalMessage(&p, N_EDITF, &Editf{})
		case N_EDITT:
			message, err = UnmarshalMessage(&p, N_EDITT, &Editt{})
		case N_EDITM:
			message, err = UnmarshalMessage(&p, N_EDITM, &Editm{})
		case N_FLIP:
			message, err = UnmarshalMessage(&p, N_FLIP, &Flip{})
		case N_COPY:
			message, err = UnmarshalMessage(&p, N_COPY, &Copy{})
		case N_PASTE:
			message, err = UnmarshalMessage(&p, N_PASTE, &Paste{})
		case N_ROTATE:
			message, err = UnmarshalMessage(&p, N_ROTATE, &Rotate{})
		case N_REPLACE:
			message, err = UnmarshalMessage(&p, N_REPLACE, &Replace{})
		case N_DELCUBE:
			message, err = UnmarshalMessage(&p, N_DELCUBE, &Delcube{})
		case N_REMIP:
			message, err = UnmarshalMessage(&p, N_REMIP, &Remip{})
		case N_EDITENT:
			message, err = UnmarshalMessage(&p, N_EDITENT, &EditEnt{})
		case N_HITPUSH:
			message, err = UnmarshalMessage(&p, N_HITPUSH, &HitPush{})
		case N_SHOTFX:
			message, err = UnmarshalMessage(&p, N_SHOTFX, &ShotFX{})
		case N_EXPLODEFX:
			message, err = UnmarshalMessage(&p, N_EXPLODEFX, &ExplodeFX{})
		case N_DAMAGE:
			message, err = UnmarshalMessage(&p, N_DAMAGE, &Damage{})
		case N_DIED:
			message, err = UnmarshalMessage(&p, N_DIED, &Died{})
		case N_FORCEDEATH:
			message, err = UnmarshalMessage(&p, N_FORCEDEATH, &ForceDeath{})
		case N_NEWMAP:
			message, err = UnmarshalMessage(&p, N_NEWMAP, &NewMap{})
		case N_REQAUTH:
			message, err = UnmarshalMessage(&p, N_REQAUTH, &ReqAuth{})
		case N_INITAI:
			message, err = UnmarshalMessage(&p, N_INITAI, &InitAI{})
		case N_SENDDEMOLIST:
			message, err = UnmarshalMessage(&p, N_SENDDEMOLIST, &SendDemoList{})
		case N_SENDDEMO:
			message, err = UnmarshalMessage(&p, N_SENDDEMO, &SendDemo{})
		case N_CLIENT:
			message, err = UnmarshalMessage(&p, N_CLIENT, &ClientInfo{})
		case N_SOUND:
			message, err = UnmarshalMessage(&p, N_SOUND, &Sound{})
		case N_CLIENTPING:
			message, err = UnmarshalMessage(&p, N_CLIENTPING, &ClientPing{})
		case N_TAUNT:
			message, err = UnmarshalMessage(&p, N_TAUNT, &Taunt{})
		case N_GUNSELECT:
			message, err = UnmarshalMessage(&p, N_GUNSELECT, &GunSelect{})
		case N_TEXT:
			message, err = UnmarshalMessage(&p, N_TEXT, &Text{})
		case N_SAYTEAM:
			message, err = UnmarshalMessage(&p, N_SAYTEAM, &SayTeam{})
		case N_SWITCHNAME:
			message, err = UnmarshalMessage(&p, N_SWITCHNAME, &SwitchName{})
		case N_SWITCHMODEL:
			message, err = UnmarshalMessage(&p, N_SWITCHMODEL, &SwitchModel{})
		case N_EDITMODE:
			message, err = UnmarshalMessage(&p, N_EDITMODE, &EditMode{})
		default:
			if code == N_SPAWN {
				if fromClient {
					message, err = UnmarshalMessage(&p, N_SPAWN, &SpawnRequest{})
				} else {
					message, err = UnmarshalMessage(&p, N_SPAWN, &SpawnResponse{})
				}
			} else {
				return nil, fmt.Errorf("unhandled code %s", code.String())
			}
		}

		log.Debug().Msgf("read message %s", message.Type().String())

		if err != nil {
			return nil, err
		}

		messages = append(messages, message)
	}

	return messages, nil
}
