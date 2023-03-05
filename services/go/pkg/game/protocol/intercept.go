package protocol

import (
	"context"
	"time"

	"github.com/sasha-s/go-deadlock"
)

type ProxiedMessage struct {
	Message Message
	Channel uint8
	drop    chan bool
	replace chan Message
}

func (p *ProxiedMessage) Drop() {
	p.drop <- true
}

func (p *ProxiedMessage) Pass() {
	p.drop <- false
}

func (p *ProxiedMessage) Replace(message Message) {
	p.replace <- message
}

type Handler struct {
	handles func(code MessageCode) bool
	recv    chan ProxiedMessage
	proxy   *MessageProxy
}

func (h *Handler) Receive() <-chan ProxiedMessage {
	return h.recv
}

func makeCodeSetCheck(codes []MessageCode) func(code MessageCode) bool {
	return func(code MessageCode) bool {
		for _, otherCode := range codes {
			if code == otherCode {
				return true
			}
		}
		return false
	}
}

func (h *Handler) Handles(code MessageCode) bool {
	return h.handles(code)
}

func (h *Handler) Remove() {
	h.proxy.Remove(h)
}

type MessageProxy struct {
	handlers   []*Handler
	mutex      deadlock.Mutex
	fromClient bool
}

func (m *MessageProxy) Process(ctx context.Context, channel uint8, message Message) (Message, error) {
	current := message
	drop := make(chan bool)
	replace := make(chan Message)
	m.mutex.Lock()
	handlers := m.handlers
	m.mutex.Unlock()
	for _, handler := range handlers {
		if !handler.Handles(message.Type()) {
			continue
		}

		handler.recv <- ProxiedMessage{
			Message: current,
			Channel: channel,
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
			current = data
		}
	}

	return current, nil
}

func (m *MessageProxy) InterceptWith(check func(MessageCode) bool) *Handler {
	handler := Handler{
		handles: check,
		recv:    make(chan ProxiedMessage),
		proxy:   m,
	}
	m.mutex.Lock()
	m.handlers = append([]*Handler{&handler}, m.handlers...)
	m.mutex.Unlock()
	return &handler
}

func (m *MessageProxy) Intercept(codes ...MessageCode) *Handler {
	return m.InterceptWith(makeCodeSetCheck(codes))
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

func (m *MessageProxy) getNext(ctx context.Context, shouldSwallow bool, codes ...MessageCode) (Message, error) {
	handler := m.Intercept(codes...)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case msg := <-handler.Receive():
		m.Remove(handler)
		if shouldSwallow {
			msg.Drop()
		} else {
			msg.Pass()
		}
		return msg.Message, nil
	}
}

type nextResult struct {
	Message Message
	Err     error
}

func (m *MessageProxy) getNextTimeout(
	ctx context.Context,
	shouldSwallow bool,
	timeout time.Duration,
	codes ...MessageCode,
) (Message, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	out := make(chan nextResult)

	go func() {
		msg, err := m.getNext(
			ctx,
			shouldSwallow,
			codes...,
		)

		out <- nextResult{
			Message: msg,
			Err:     err,
		}
	}()

	select {
	case <-timeoutCtx.Done():
		return nil, timeoutCtx.Err()
	case result := <-out:
		return result.Message, result.Err
	}
}

// Wait for a message and drop it.
func (m *MessageProxy) Next(ctx context.Context, codes ...MessageCode) (Message, error) {
	return m.getNext(ctx, true, codes...)
}

func (m *MessageProxy) NextTimeout(
	ctx context.Context,
	timeout time.Duration,
	codes ...MessageCode,
) (Message, error) {
	return m.getNextTimeout(ctx, true, timeout, codes...)
}

const (
	DEFAULT_TIMEOUT = 5 * time.Second
)

func (m *MessageProxy) Take(
	ctx context.Context,
	codes ...MessageCode,
) (Message, error) {
	return m.getNextTimeout(ctx, true, DEFAULT_TIMEOUT, codes...)
}

// Wait for a message, but don't prevent it from being transmitted.
func (m *MessageProxy) Wait(ctx context.Context, codes ...MessageCode) (Message, error) {
	return m.getNext(ctx, false, codes...)
}

func (m *MessageProxy) WaitTimeout(
	ctx context.Context,
	timeout time.Duration,
	codes ...MessageCode,
) (Message, error) {
	return m.getNextTimeout(ctx, false, timeout, codes...)
}

func NewMessageProxy(fromClient bool) *MessageProxy {
	return &MessageProxy{
		handlers:   make([]*Handler, 0),
		fromClient: fromClient,
	}
}
