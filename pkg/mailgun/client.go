// Package mailgun is the real core.Mailer implementation (Mailgun HTTP API).
package mailgun

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"9itchi-account-api/utils"

	sdk "github.com/mailgun/mailgun-go/v4"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

const (
	maxAttempts    = 3
	retryDelayBase = 500 * time.Millisecond
	attemptTimeout = 5 * time.Second
	callTimeout    = 15 * time.Second
)

// Config is the Mailgun setup. APIBase is empty for the US region ("https://api.eu.mailgun.net" for EU).
type Config struct {
	APIKey    string
	Domain    string
	FromEmail string
	APIBase   string
	RateLimit int
}

// sender is the minimal slice of the Mailgun SDK used here, so tests can fake it.
type sender interface {
	Send(ctx context.Context, from, to, subject, text, htmlBody string) error
}

type Client struct {
	sender  sender
	from    string
	limiter *rate.Limiter
	ttl     time.Duration
	retryIn func(attempt int) time.Duration
}

// New builds a Client. resetTTL is the lifetime of the codes (reset and verification), shown in the message.
func New(cfg Config, resetTTL time.Duration) *Client {
	mg := sdk.NewMailgun(cfg.Domain, cfg.APIKey)
	if cfg.APIBase != "" {
		mg.SetAPIBase(normalizeBase(cfg.APIBase))
	}
	from := cfg.FromEmail
	if from == "" {
		from = "9itchi <no-reply@" + cfg.Domain + ">"
	}
	limit := cfg.RateLimit
	if limit < 1 {
		limit = 10
	}
	return &Client{
		sender:  sdkSender{mg: mg},
		from:    from,
		limiter: rate.NewLimiter(rate.Limit(limit), limit),
		ttl:     resetTTL,
		retryIn: func(attempt int) time.Duration { return retryDelayBase * time.Duration(attempt) },
	}
}

// normalizeBase appends the /v3 path the SDK expects.
func normalizeBase(base string) string {
	base = strings.TrimRight(base, "/")
	if !strings.HasSuffix(base, "/v3") {
		base += "/v3"
	}
	return base
}

// SendPasswordResetCode sends the reset code; errors never contain the code, the address or the API key.
func (c *Client) SendPasswordResetCode(ctx context.Context, email, lang, code string) error {
	return c.sendCode(ctx, kindReset, email, lang, code)
}

// SendVerificationCode sends the code that confirms a new account's email.
func (c *Client) SendVerificationCode(ctx context.Context, email, lang, code string) error {
	return c.sendCode(ctx, kindVerify, email, lang, code)
}

func (c *Client) sendCode(ctx context.Context, kind, email, lang, code string) error {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()

	subject, text, htmlBody, ok := renderCode(kind, lang, code, int(c.ttl.Minutes()))
	if !ok {
		return fmt.Errorf("unknown email kind %q", kind)
	}

	class := ""
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := c.limiter.Wait(ctx); err != nil {
			return fmt.Errorf("mailgun rate limiter: %w", ctxCause(ctx, err))
		}
		attemptCtx, attemptCancel := context.WithTimeout(ctx, attemptTimeout)
		err := c.sender.Send(attemptCtx, c.from, email, subject, text, htmlBody)
		attemptCancel()
		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return fmt.Errorf("mailgun send aborted: %w", ctx.Err())
		}
		var retry bool
		class, retry = classify(err)
		utils.Logger.Warn("mailgun send failed", zap.Int("attempt", attempt), zap.String("class", class))
		if !retry || attempt == maxAttempts {
			break
		}
		select {
		case <-time.After(c.retryIn(attempt)):
		case <-ctx.Done():
			return fmt.Errorf("mailgun send aborted: %w", ctx.Err())
		}
	}
	return fmt.Errorf("mailgun send failed: %s", class)
}

func ctxCause(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}

// classify returns a generic error class and whether the failure is worth retrying (5xx or network).
func classify(err error) (string, bool) {
	var respErr *sdk.UnexpectedResponseError
	if errors.As(err, &respErr) {
		if respErr.Actual >= 500 {
			return "http_5xx", true
		}
		return "http_4xx", false
	}
	return "network", true
}

type sdkSender struct{ mg *sdk.MailgunImpl }

func (s sdkSender) Send(ctx context.Context, from, to, subject, text, htmlBody string) error {
	msg := sdk.NewMessage(from, subject, text, to)
	msg.SetHTML(htmlBody)
	_, _, err := s.mg.Send(ctx, msg)
	return err
}
