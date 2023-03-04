package protocol

func Read(b []byte, fromClient bool) ([]Message, error) {
	messages := make([]Message, 0)
	p := Packet(b)

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
		case N_EDITVAR:
			message, err = UnmarshalMessage(&p, N_EDITVAR, &EditVar{})
		case N_EDITVSLOT:
			message, err = UnmarshalMessage(&p, N_EDITVSLOT, &EditVSlot{})
		case N_EXPLODE:
			message, err = UnmarshalMessage(&p, N_EXPLODE, &Explode{})
		case N_FORCEINTERMISSION:
			message, err = UnmarshalMessage(&p, N_FORCEINTERMISSION, &ForceIntermission{})
		case N_FROMAI:
			message, err = UnmarshalMessage(&p, N_FROMAI, &FromAI{})
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
		case N_REDO:
			message, err = UnmarshalMessage(&p, N_REDO, &PackData{})
		case N_SENDMAP:
			message, err = UnmarshalMessage(&p, N_SENDMAP, &SendMap{})
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
		case N_UNDO:
			message, err = UnmarshalMessage(&p, N_UNDO, &PackData{})
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
			message, err = UnmarshalMessage(&p, N_CLIPBOARD, &PackData{})
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
			message, err = UnmarshalMessage(&p, N_EDITENT, &EditEntity{})
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

		if !IsSpammyMessage(message.Type()) {
			log.Debug().Msgf("read message %s", message.Type().String())
		}

		if err != nil {
			return nil, err
		}

		messages = append(messages, message)
	}

	return messages, nil
}
