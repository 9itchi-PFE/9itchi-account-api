// Package core holds interfaces for pluggable parts.
package core

import "context"

// Mailer delivers account e-mails. Implementations must never log the code.
type Mailer interface {
	SendPasswordResetCode(ctx context.Context, email, code string) error
}

// NoopMailer is the default Mailer: it drops the message.
type NoopMailer struct{}

func (NoopMailer) SendPasswordResetCode(context.Context, string, string) error { return nil }
