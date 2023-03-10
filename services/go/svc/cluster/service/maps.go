package service

import (
	"context"
	"fmt"
	"time"

	C "github.com/cfoust/sour/pkg/game/constants"
	P "github.com/cfoust/sour/pkg/game/protocol"
)

func (c *Cluster) waitForMapConsent(ctx context.Context, user *User) error {
	timeout, cancel := context.WithTimeout(user.Ctx(), 60*time.Second)
	defer cancel()

	check := time.NewTicker(250 * time.Millisecond)
	warn := time.NewTicker(10 * time.Second)
	serverCtx := user.ServerSessionContext()

	message := "you are missing assets. run '/do (getservauth)' to allow the server to securely send maps and assets you are missing"
	user.Message(message)

	for {
		select {
		case <-timeout.Done():
			go c.runCommand(ctx, user, "go lobby")
			return fmt.Errorf("user never consented")
		case <-serverCtx.Done():
			return fmt.Errorf("user left the server")
		case <-check.C:
			if !user.HasCubeScript() {
				continue
			}
			return nil
		case <-warn.C:
			user.Message(message)
		}
	}
}

func sendRawMap(ctx context.Context, user *User, data []byte) error {
	done := user.SendChannel(
		2,
		P.SendMap{
			Map: data,
		},
	)

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

func runScriptAndWait(ctx context.Context, user *User, type_ P.MessageCode, code string) (P.Message, error) {
	csError := make(chan error)
	msgChan := make(chan P.Message)
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

	user.Message("downloading map assets...")
	msg, err := runScriptAndWait(serverCtx, user, P.N_GETDEMO, fmt.Sprintf(`
getdemo 0 %s
`, fileName))
	if err != nil {
		return err
	}

	getDemo := msg.(P.GetDemo)
	tag := getDemo.Tag

	err = user.SendChannelSync(
		2,
		P.SendDemo{
			Tag:  tag,
			Data: data,
		},
	)
	if err != nil {
		return err
	}

	timeout, cancel := context.WithTimeout(user.Ctx(), 120*time.Second)
	defer cancel()

	for {
		msg, err = runScriptAndWait(timeout, user, P.N_SERVCMD, fmt.Sprintf(`
if (= (findfile demo/%s.dmo) 1) [servcmd ok] [servcmd missing]
`, fileName))
		if err != nil {
			return err
		}

		cmd := msg.(P.ServCMD)
		if cmd.Command != "ok" {
			logger.Info().Msg("demo missing")
			time.Sleep(1 * time.Second)
			continue
		}
		break
	}

	user.Message("mounting asset layer...")
	msg, err = runScriptAndWait(serverCtx, user, P.N_SERVCMD, fmt.Sprintf(`
addzip demo/%s.dmo
servcmd ok
`, fileName))
	if err != nil {
		return err
	}

	cmd := msg.(P.ServCMD)
	if cmd.Command != "ok" {
		return fmt.Errorf("user never ack'd demo")
	}

	user.Message("map download complete")

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

		user.SendChannel(
			2,
			P.SendMap{
				Map: data,
			},
		)

		return nil
	}

	server.Mutex.RLock()
	mode := int32(server.GameMode.ID())
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
	if mode == C.MODE_COOP && !map_.HasCFG {
		data, err := found.GetOGZ(ctx)
		if err != nil {
			return err
		}

		return sendRawMap(ctx, user, data)
	}

	// You can't SENDMAP outside of coopedit, change to it
	if mode != C.MODE_COOP {
		user.Send(P.MapChange{
			Name:     "",
			Mode:     int32(C.MODE_COOP),
			HasItems: false,
		})
		user.From.Take(ctx, P.N_MAPCRC)
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

	user.Message("packaging data...")
	data, err := found.GetBundle(ctx)
	if err != nil {
		return err
	}

	err = sendBundle(ctx, user, map_.Bundle, data)
	if err != nil {
		return err
	}

	// Then change back
	user.Send(P.MapChange{
		Name:     map_.Name,
		Mode:     int32(mode),
		HasItems: false,
	})
	user.From.Take(ctx, P.N_MAPCRC)
	logger.Info().Msgf("downloaded map %s (%d)", name, len(data))

	// Changing maps causes the gamelimit to disappear, so the server has
	// to resend it
	server.RefreshTime()
	user.ServerClient.RefreshWelcome()

	return nil
}
