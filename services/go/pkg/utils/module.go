package utils

import (
	"context"
)

type Session struct {
	context context.Context
	cancel  context.CancelFunc
}

func NewSession(ctx context.Context) Session {
	ctx, cancel := context.WithCancel(ctx)
	return Session{
		context: ctx,
		cancel:  cancel,
	}
}

func (s *Session) Ctx() context.Context {
	return s.context
}

func (s *Session) IsDone() bool {
	return s.context.Err() != nil
}

func (s *Session) Cancel() {
	s.cancel()
}
