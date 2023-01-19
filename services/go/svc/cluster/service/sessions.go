package service

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/cfoust/sour/pkg/game"

	"github.com/go-redis/redis/v9"
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

func EncodeDemo(startTime time.Time, messages []RecordedPacket) ([]byte, error) {
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
		millis := int32(message.Time.Sub(startTime).Round(time.Millisecond).Milliseconds())
		p.Put(
			int32(millis),
			int32(packet.Channel),
			int32(len(packet.Data)),
			packet.Data,
		)
	}

	compressed, err := Compress(p)
	if err != nil {
		return nil, err
	}
	return compressed, nil
}

const (
	DEMO_KEY = "demo-%s"
	DEMO_TTL = time.Duration(48 * time.Hour)
)

func (c *Cluster) GetDemo(ctx context.Context, id string) ([]byte, error) {
	return c.redis.Get(ctx, fmt.Sprintf(DEMO_KEY, id)).Bytes()
}

func RecordSession(ctx context.Context, redis *redis.Client, shouldSave bool, user *User) error {
	logger := user.Logger()
	to, from := user.ReceiveIntercept()

	start := time.Now()

	allMsg := make([]RecordedPacket, 0)
	toMsg := make([]RecordedPacket, 0)

Outer:
	for {
		select {
		case <-ctx.Done():
			break Outer
		case msg := <-to:
			toMsg = append(toMsg, NewPacket(false, msg))
			allMsg = append(allMsg, NewPacket(false, msg))
		case msg := <-from:
			allMsg = append(allMsg, NewPacket(true, msg))
		}
	}

	if !shouldSave {
		return nil
	}

	allDemo, err := EncodeDemo(start, allMsg)
	if err != nil {
		return err
	}

	toDemo, err := EncodeDemo(start, toMsg)
	if err != nil {
		return err
	}

	key := user.Session

	pipe := redis.Pipeline()
	pipe.Set(context.Background(), fmt.Sprintf(DEMO_KEY, key), toDemo, DEMO_TTL)
	pipe.Set(context.Background(), fmt.Sprintf(DEMO_KEY, key+"-all"), allDemo, DEMO_TTL)
	_, err = pipe.Exec(context.Background())
	if err != nil {
		return err
	}

	logger.Info().Msg("saved session")

	return nil
}
