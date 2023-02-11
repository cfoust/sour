package service

import (
	"context"
	"fmt"
	"time"

	"github.com/cfoust/sour/pkg/game"
)

func sendClient(user *User, data []byte, channel int) <-chan bool {
	return user.Send(game.GamePacket{
		Channel: uint8(channel),
		Data:    data,
	})
}

func sendClientSync(user *User, data []byte, channel int) error {
	if !<-sendClient(user, data, channel) {
		return fmt.Errorf("client never acknowledged message")
	}
	return nil
}

func (c *Cluster) waitForMapConsent(ctx context.Context, user *User) error {
	timeout, cancel := context.WithTimeout(user.Context(), 60*time.Second)
	defer cancel()

	check := time.NewTicker(250 * time.Millisecond)
	warn := time.NewTicker(10 * time.Second)
	serverCtx := user.ServerSessionContext()

	message := "you are missing assets. run '/do (getservauth)' to allow the server to automatically send maps and assets you are missing"
	user.SendServerMessage(message)

	for {
		select {
		case <-timeout.Done():
			go c.RunCommand(ctx, "go lobby", user)
			return fmt.Errorf("user never consented")
		case <-serverCtx.Done():
			return fmt.Errorf("user left the server")
		case <-check.C:
			if !user.HasCubeScript() {
				continue
			}
			return nil
		case <-warn.C:
			user.SendServerMessage(message)
		}
	}
}

func sendRawMap(ctx context.Context, user *User, data []byte) error {
	p := game.Packet{}
	p.Put(game.N_SENDMAP)
	p = append(p, data...)
	done := user.Send(game.GamePacket{
		Channel: 2,
		Data:    p,
	})

	sendCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	select {
	case <-sendCtx.Done():
		cancel()
		return fmt.Errorf("user failed to download map")
	case <-done:
		cancel()
		break
	}

	return nil
}

const RUN_WAIT_TIMEOUT = 15 * time.Second

func runScriptAndWait(ctx context.Context, user *User, type_ game.MessageCode, code string) (game.Message, error) {
	csError := make(chan error)
	msgChan := make(chan game.Message)
	go func() {
		csError <- user.RunCubeScript(ctx, code)
	}()

	go func() {
		msg, err := user.From.NextTimeout(ctx, RUN_WAIT_TIMEOUT, type_)
		if err != nil {
			msgChan <- nil
			return
		}
		msgChan <- msg
	}()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err := <-csError:
			if err != nil {
				return nil, err
			}
			continue
		case msg := <-msgChan:
			if msg == nil {
				return nil, fmt.Errorf("waiting for message failed")
			}
			return msg, nil
		}
	}
}

func sendBundle(serverCtx context.Context, user *User, id string, data []byte) error {
	logger := user.Logger()

	fileName := id[:20]

	user.SendServerMessage("downloading map assets...")
	msg, err := runScriptAndWait(serverCtx, user, game.N_GETDEMO, fmt.Sprintf(`
demodir sour
getdemo 0 %s
`, fileName))
	if err != nil {
		return err
	}

	getDemo := msg.Contents().(*game.GetDemo)
	tag := getDemo.Tag

	p := game.Packet{}
	p.Put(
		game.N_SENDDEMO,
		tag,
	)
	p = append(p, data...)
	err = sendClientSync(user, p, 2)
	if err != nil {
		return err
	}

	timeout, cancel := context.WithTimeout(user.Context(), 120*time.Second)
	defer cancel()

	for {
		msg, err = runScriptAndWait(timeout, user, game.N_SERVCMD, fmt.Sprintf(`
if (= (findfile sour/%s.dmo) 1) [servcmd ok] [servcmd missing]
`, fileName))
		if err != nil {
			return err
		}

		cmd := msg.Contents().(*game.ServCMD)
		if cmd.Command != "ok" {
			logger.Info().Msg("demo missing")
			time.Sleep(1 * time.Second)
			continue
		}
		break
	}

	user.SendServerMessage("mounting asset layer...")
	msg, err = runScriptAndWait(serverCtx, user, game.N_SERVCMD, fmt.Sprintf(`
addzip sour/%s.dmo
demodir demo
servcmd ok
`, fileName))
	if err != nil {
		return err
	}

	cmd := msg.Contents().(*game.ServCMD)
	if cmd.Command != "ok" {
		return fmt.Errorf("user never ack'd demo")
	}

	user.SendServerMessage("map download complete")

	return nil
}

func (c *Cluster) SendMap(ctx context.Context, user *User, name string) error {
	user.Mutex.RLock()
	isSending := user.sendingMap
	user.Mutex.RUnlock()

	if isSending {
		return nil
	}

	user.Mutex.Lock()
	user.sendingMap = true
	user.Mutex.Unlock()

	defer func() {
		user.Mutex.Lock()
		user.sendingMap = false
		user.Mutex.Unlock()
	}()

	server := user.GetServer()
	instance := c.spaces.FindInstance(server)

	if instance != nil && instance.Editing != nil {
		e := instance.Editing
		err := e.Checkpoint(ctx)
		if err != nil {
			return err
		}

		data, err := e.Map.LoadMapData(ctx)
		if err != nil {
			return err
		}

		p := game.Packet{}
		p.Put(game.N_SENDMAP)
		p = append(p, data...)

		user.Send(game.GamePacket{
			Channel: 2,
			Data:    p,
		})

		return nil
	}

	server.Mutex.RLock()
	mode := server.Mode
	mapName := server.Map
	server.Mutex.RUnlock()

	found := c.assets.FindMap(mapName)
	// Server might be used for something else e.g. general coopedit
	if found == nil {
		return nil
	}

	map_ := found.Map

	logger := user.Logger()
	logger.Info().Str("map", map_.Name).Msg("sending map to client")

	// Specifically in this case we don't need CS
	if mode == game.MODE_COOP && !map_.HasCFG {
		data, err := found.GetOGZ(ctx)
		if err != nil {
			return err
		}

		return sendRawMap(ctx, user, data)
	}

	send := func(data []byte, channel uint8) {
		user.Send(game.GamePacket{
			Data:    data,
			Channel: channel,
		})
	}

	// You can't SENDMAP outside of coopedit, change to it
	if mode != game.MODE_COOP {
		p := game.Packet{}
		p.Put(
			game.N_MAPCHANGE,
			game.MapChange{
				Name:     "",
				Mode:     int(game.MODE_COOP),
				HasItems: 0,
			},
		)
		send(p, 1)
		user.From.Take(ctx, game.N_MAPCRC)
	}

	// Go to purgatory
	err := sendRawMap(ctx, user, PURGATORY)
	if err != nil {
		return err
	}

	// Otherwise we always need CS
	if !user.HasCubeScript() {
		err := c.waitForMapConsent(ctx, user)
		if err != nil {
			return err
		}
	}

	user.SendServerMessage("packaging data...")
	data, err := found.GetBundle(ctx)
	if err != nil {
		return err
	}

	err = sendBundle(ctx, user, map_.Bundle, data)
	if err != nil {
		return err
	}

	// Then change back
	p := game.Packet{}
	p.Put(
		game.N_MAPCHANGE,
		game.MapChange{
			Name:     map_.Name,
			Mode:     int(mode),
			HasItems: 1,
		},
	)
	send(p, 1)
	user.From.Take(ctx, game.N_MAPCRC)
	logger.Info().Msgf("downloaded map %s (%d)", name, len(data))

	// Changing maps causes the gamelimit to disappear, so the server has
	// to resend it
	server.SendCommand("sendtime")

	num := user.GetClientNum()
	server.SendCommand(fmt.Sprintf("refreshwelcome %d", num))

	return nil
}
