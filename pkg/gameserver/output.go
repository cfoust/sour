package gameserver

import P "github.com/cfoust/sour/pkg/game/protocol"

// OutputBuffer collects output packets during a Step() call. It replaces
// the channel-based sending pattern where client.Send() and
// Clients.Broadcast() wrote to the unbuffered outgoing channel.
type OutputBuffer struct {
	packets []ServerPacket
}

// Send queues a packet to a specific client on channel 1 (game messages).
func (o *OutputBuffer) Send(session uint32, messages ...P.Message) {
	if len(messages) == 0 {
		return
	}
	o.packets = append(o.packets, ServerPacket{
		Session:  session,
		Channel:  1,
		Messages: messages,
	})
}

// SendChan queues a packet to a specific client on a specific channel.
func (o *OutputBuffer) SendChan(session uint32, channel uint8, messages ...P.Message) {
	if len(messages) == 0 {
		return
	}
	o.packets = append(o.packets, ServerPacket{
		Session:  session,
		Channel:  channel,
		Messages: messages,
	})
}

// Drain returns all collected packets and resets the buffer.
func (o *OutputBuffer) Drain() []ServerPacket {
	result := o.packets
	o.packets = nil
	return result
}
