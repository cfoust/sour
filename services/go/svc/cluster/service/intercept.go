package service

import (
	"context"
	"fmt"
	"github.com/cfoust/sour/pkg/game"

	"github.com/sasha-s/go-deadlock"
)

type ProxiedMessage struct {
	Message game.Message
	drop    chan bool
	replace chan []byte
}

func (p *ProxiedMessage) Drop() {
	p.drop <- true
}

func (p *ProxiedMessage) Pass() {
	p.drop <- false
}

func (p *ProxiedMessage) Replace(data []byte) {
	p.replace <- data
}

type Handler struct {
	codes []game.MessageCode
	recv  chan ProxiedMessage
}

func (h *Handler) Receive() <-chan ProxiedMessage {
	return h.recv
}

func (h *Handler) Handles(code game.MessageCode) bool {
	for _, otherCode := range h.codes {
		if code == otherCode {
			return true
		}
	}
	return false
}

type MessageProxy struct {
	handlers   []*Handler
	mutex      deadlock.Mutex
	fromClient bool
}

func (m *MessageProxy) Process(ctx context.Context, message game.Message) ([]byte, error) {

	current := message
	drop := make(chan bool)
	replace := make(chan []byte)
	m.mutex.Lock()
	for _, handler := range m.handlers {
		if !handler.Handles(message.Type()) {
			continue
		}

		handler.recv <- ProxiedMessage{
			Message: current,
			drop:    drop,
			replace: replace,
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case shouldDrop := <-drop:
			if shouldDrop {
				return nil, nil
			}
			continue
		case data := <-replace:
			messages, err := game.Read(data, m.fromClient)
			if err != nil {
				return nil, err
			}
			if len(messages) != 1 {
				return nil, fmt.Errorf("handler returned invalid number of messages")
			}
			current = messages[0]
		}
	}
	m.mutex.Unlock()

	return nil, nil
}

func (m *MessageProxy) Intercept(codes ...game.MessageCode) *Handler {
	handler := Handler{
		codes: codes,
		recv:  make(chan ProxiedMessage),
	}
	m.mutex.Lock()
	m.handlers = append(m.handlers, &handler)
	m.mutex.Unlock()
	return &handler
}

func (m *MessageProxy) Remove(handler *Handler) {
	newHandlers := make([]*Handler, 0)
	m.mutex.Lock()
	for _, otherHandler := range m.handlers {
		if handler == otherHandler {
			continue
		}
		newHandlers = append(newHandlers, otherHandler)
	}
	m.handlers = newHandlers
	m.mutex.Unlock()
}

func (m *MessageProxy) getNext(ctx context.Context, shouldSwallow bool, codes ...game.MessageCode) (game.Message, error) {
	handler := m.Intercept(codes...)
	defer m.Remove(handler)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case msg := <-handler.Receive():
		if shouldSwallow {
			msg.Drop()
		} else {
			msg.Pass()
		}
		return msg.Message, nil
	}
}

func (m *MessageProxy) Next(ctx context.Context, codes ...game.MessageCode) (game.Message, error) {
	return m.getNext(ctx, true, codes...)
}

func (m *MessageProxy) Wait(ctx context.Context, codes ...game.MessageCode) (game.Message, error) {
	return m.getNext(ctx, false, codes...)
}

func NewMessageProxy(fromClient bool) *MessageProxy {
	return &MessageProxy{
		handlers:   make([]*Handler, 0),
		fromClient: fromClient,
	}
}
