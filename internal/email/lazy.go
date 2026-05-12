package email

import (
	"context"
	"sync"
)

type LazyService struct {
	mu      sync.RWMutex
	resolve func() Service
	cached  Service
}

func NewLazyService() *LazyService {
	return &LazyService{}
}

func (l *LazyService) SetResolver(resolve func() Service) {
	l.mu.Lock()
	l.resolve = resolve
	l.cached = nil
	l.mu.Unlock()
}

func (l *LazyService) getService() Service {
	l.mu.RLock()
	if l.cached != nil {
		l.mu.RUnlock()
		return l.cached
	}
	l.mu.RUnlock()

	l.mu.Lock()
	defer l.mu.Unlock()
	if l.cached != nil {
		return l.cached
	}
	if l.resolve != nil {
		l.cached = l.resolve()
	}
	return l.cached
}

func (l *LazyService) SendMagicLink(ctx context.Context, to, token, link string) error {
	if s := l.getService(); s != nil {
		return s.SendMagicLink(ctx, to, token, link)
	}
	return nil
}

func (l *LazyService) SendVerificationEmail(ctx context.Context, to, token, link string) error {
	if s := l.getService(); s != nil {
		return s.SendVerificationEmail(ctx, to, token, link)
	}
	return nil
}

func (l *LazyService) SendPasswordReset(ctx context.Context, to, token, link string) error {
	if s := l.getService(); s != nil {
		return s.SendPasswordReset(ctx, to, token, link)
	}
	return nil
}

func (l *LazyService) SendInvitationEmail(ctx context.Context, to, inviterName, inviteLink string) error {
	if s := l.getService(); s != nil {
		return s.SendInvitationEmail(ctx, to, inviterName, inviteLink)
	}
	return nil
}

func (l *LazyService) Send(ctx context.Context, to, subject, body string) error {
	if s := l.getService(); s != nil {
		return s.Send(ctx, to, subject, body)
	}
	return nil
}

func (l *LazyService) IsConfigured() bool {
	if s := l.getService(); s != nil {
		return s.IsConfigured()
	}
	return false
}
