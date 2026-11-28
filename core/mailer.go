// Package core holds interfaces for pluggable parts.
package core

import (
	"context"
	"errors"
	"time"
)

// Mailer delivers account e-mails in the user's language (fr, en, ar; anything else is French). Implementations must
// never log the code.
type Mailer interface {
	SendPasswordResetCode(ctx context.Context, email, lang, code string) error
	// SendVerificationCode sends the code that proves a new account owns its email.
	SendVerificationCode(ctx context.Context, email, lang, code string) error
}

// NoopMailer is the default Mailer: it drops the message.
type NoopMailer struct{}

func (NoopMailer) SendPasswordResetCode(context.Context, string, string, string) error { return nil }
func (NoopMailer) SendVerificationCode(context.Context, string, string, string) error  { return nil }

// AsyncMailer delivers in the background so the forgot-password response time does not reveal
// whether the account exists. Delivery errors go to OnError (it must not log the code or address).
type AsyncMailer struct {
	Mailer  Mailer
	OnError func(error)
}

// asyncSendTimeout bounds each background delivery so goroutines cannot pile up on a hung provider.
const asyncSendTimeout = 30 * time.Second

func (a AsyncMailer) SendPasswordResetCode(ctx context.Context, email, lang, code string) error {
	return a.inBackground(ctx, func(ctx context.Context) error { return a.Mailer.SendPasswordResetCode(ctx, email, lang, code) })
}

func (a AsyncMailer) SendVerificationCode(ctx context.Context, email, lang, code string) error {
	return a.inBackground(ctx, func(ctx context.Context) error { return a.Mailer.SendVerificationCode(ctx, email, lang, code) })
}

// inBackground runs send outside the request and returns at once.
func (a AsyncMailer) inBackground(ctx context.Context, send func(ctx context.Context) error) error {
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
		if err := send(ctx); err != nil {
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
