package relay

import (
	"github.com/sauerbraten/waiter/internal/net/packet"
)

// Publisher provides methods to send updates to all subscribers of a certain topic.
type Publisher struct {
	cn          uint32
	notifyRelay chan<- uint32
	updates     chan<- []byte
}

func newPublisher(cn uint32, notifyRelay chan<- uint32) (*Publisher, <-chan []byte) {
	updates := make(chan []byte)

	p := &Publisher{
		cn:          cn,
		notifyRelay: notifyRelay,
		updates:     updates,
	}

	return p, updates
}

// Publish notifies p's broker that there is an update on p's topic and blocks until the broker received the notification.
// Publish then blocks until the broker received the update. Calling Publish() after Close() returns immediately. Use p's
// Stop channel to know when the broker stopped listening.
func (p *Publisher) Publish(args ...interface{}) {
	p.notifyRelay <- p.cn
	p.updates <- packet.Encode(args...)
}

// Close tells the broker there will be no more updates coming from p. Calling Publish() after Close() returns immediately.
// Calling Close() makes the broker unsubscribe all subscribers and telling them updates on the topic have ended.
func (p *Publisher) Close() {
	close(p.updates)
}
