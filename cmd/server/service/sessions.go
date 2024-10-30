package service

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"time"

	C "github.com/cfoust/sour/pkg/game/constants"
	"github.com/cfoust/sour/pkg/game/io"
	P "github.com/cfoust/sour/pkg/game/protocol"

	"github.com/go-redis/redis/v9"
)

type RecordedPacket struct {
	From   bool
	Time   time.Time
	Packet io.RawPacket
}

func NewPacket(from bool, packet io.RawPacket) RecordedPacket {
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
	p := io.Buffer{}

	// The header
	p.Put(
		[]byte(C.DEMO_MAGIC),
		int32(C.DEMO_VERSION),
		int32(P.PROTOCOL_VERSION),
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

func EncodeSession(startTime time.Time, messages []RecordedPacket) ([]byte, error) {
	p := io.Buffer{}

	for _, message := range messages {
		packet := message.Packet
		millis := int32(message.Time.Sub(startTime).Round(time.Millisecond).Milliseconds())
		p.Put(
			message.From,
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
	to := user.RawTo.Subscribe()
	from := user.RawFrom.Subscribe()

	start := time.Now()

	allMsg := make([]RecordedPacket, 0)
	toMsg := make([]RecordedPacket, 0)

Outer:
	for {
		select {
		case <-ctx.Done():
			break Outer
		case msg := <-to.Recv():
			toMsg = append(toMsg, NewPacket(false, msg))
			allMsg = append(allMsg, NewPacket(false, msg))
		case msg := <-from.Recv():
			allMsg = append(allMsg, NewPacket(true, msg))
		}
	}

	if !shouldSave {
		return nil
	}

	session, err := EncodeSession(start, allMsg)
	if err != nil {
		return err
	}

	toDemo, err := EncodeDemo(start, toMsg)
	if err != nil {
		return err
	}

	key := user.GetSessionID()

	pipe := redis.Pipeline()
	pipe.Set(context.Background(), fmt.Sprintf(DEMO_KEY, key+"-demo"), toDemo, DEMO_TTL)
	pipe.Set(context.Background(), fmt.Sprintf(DEMO_KEY, key), session, DEMO_TTL)
	_, err = pipe.Exec(context.Background())
	if err != nil {
		return err
	}

	return nil
}
