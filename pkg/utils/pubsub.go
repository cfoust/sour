package utils

import (
	"github.com/sasha-s/go-deadlock"
)

type Topic[T any] struct {
	subscribers map[chan T]struct{}
	mutex       deadlock.Mutex
}

func NewTopic[T any]() *Topic[T] {
	return &Topic[T]{
		subscribers: make(map[chan T]struct{}),
	}
}

func (t *Topic[T]) Publish(value T) {
	t.mutex.Lock()
	for subscriber := range t.subscribers {
		subscriber <- value
	}
	t.mutex.Unlock()
}

type Subscriber[T any] struct {
	channel chan T
	topic   *Topic[T]
}

func (t *Topic[T]) Subscribe() *Subscriber[T] {
	channel := make(chan T)
	t.mutex.Lock()
	t.subscribers[channel] = struct{}{}
	t.mutex.Unlock()

	return &Subscriber[T]{channel, t}
}

func (t *Subscriber[T]) Recv() <-chan T {
	return t.channel
}

func (t *Subscriber[T]) Done() {
	topic := t.topic
	topic.mutex.Lock()
	delete(topic.subscribers, t.channel)
	topic.mutex.Unlock()
}
