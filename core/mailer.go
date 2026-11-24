// Package core holds interfaces for pluggable parts.
package core

import (
	"context"
	"errors"
	"time"
)

// Mailer delivers account e-mails. Implementations must never log the code.
type Mailer interface {
	SendPasswordResetCode(ctx context.Context, email, code string) error
}

// NoopMailer is the default Mailer: it drops the message.
type NoopMailer struct{}

func (NoopMailer) SendPasswordResetCode(context.Context, string, string) error { return nil }

// AsyncMailer delivers in the background so the forgot-password response time does not reveal
// whether the account exists. Delivery errors go to OnError (it must not log the code or address).
type AsyncMailer struct {
	Mailer  Mailer
	OnError func(error)
}

// asyncSendTimeout bounds each background delivery so goroutines cannot pile up on a hung provider.
const asyncSendTimeout = 30 * time.Second

func (a AsyncMailer) SendPasswordResetCode(ctx context.Context, email, code string) error {
	// The request context is cancelled once the response is written; delivery must outlive it.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), asyncSendTimeout)
	go func() {
		defer cancel()
		// A panic in a background goroutine would take the whole service down; the value may hold the code, so it is dropped.
		defer func() {
			if recover() != nil {
				a.report(errors.New("mailer panicked"))
			}
		}()
		if err := a.Mailer.SendPasswordResetCode(ctx, email, code); err != nil {
			a.report(err)
		}
	}()
	return nil
}

func (a AsyncMailer) report(err error) {
	if a.OnError != nil {
		a.OnError(err)
	}
}
