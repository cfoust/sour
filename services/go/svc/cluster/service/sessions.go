package service

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cfoust/sour/pkg/game"
)

type RecordedPacket struct {
	From   bool
	Time   time.Time
	Packet game.GamePacket
}

func NewPacket(from bool, packet game.GamePacket) RecordedPacket {
	return RecordedPacket{
		From:   from,
		Time:   time.Now(),
		Packet: packet,
	}
}

func Compress(data []byte) ([]byte, error) {
	var buffer bytes.Buffer
	gz := gzip.NewWriter(&buffer)
	_, err := gz.Write(data)
	if err != nil {
		return nil, err
	}
	gz.Close()

	return buffer.Bytes(), nil
}

func WriteFile(path string, data []byte) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = out.Write(data)
	if err != nil {
		return err
	}

	return nil
}

func RecordSession(ctx context.Context, directory string, user *User) error {
	logger := user.Logger()
	to, from := user.ReceiveIntercept()

	messages := make([]RecordedPacket, 0)
	start := time.Now()

	path := filepath.Join(
		directory,
		fmt.Sprintf(
			"%s-%s.dmo",
			start.Format("2006.01.02.03.04.05"),
			user.Connection.Host(),
		),
	)

	shouldSave := len(directory) > 0

	if shouldSave {
		logger.Info().Str("path", path).Msg("logging client session")
	}

Outer:
	for {
		select {
		case <-ctx.Done():
			logger.Info().Msg("client done")
			break Outer
		case msg := <-to:
			messages = append(messages, NewPacket(false, msg))
		case msg := <-from:
			messages = append(messages, NewPacket(true, msg))
		}
	}

	logger.Info().Msg("client done")

	if !shouldSave {
		return nil
	}

	// Write a valid demo
	p := game.Buffer{}

	// The header
	p.Put(
		[]byte(game.DEMO_MAGIC),
		int32(game.DEMO_VERSION),
		int32(game.PROTOCOL_VERSION),
	)

	for _, message := range messages {
		packet := message.Packet
		millis := int32(message.Time.Sub(start).Round(time.Millisecond).Milliseconds())
		p.Put(
			int32(millis),
			int32(packet.Channel),
			int32(len(packet.Data)),
			packet.Data,
		)
	}

	compressed, err := Compress(p)
	if err != nil {
		return err
	}

	err = WriteFile(path, compressed)
	if err != nil {
		return err
	}

	logger.Info().Str("path", path).Msg("saved client session")

	return nil
}
