package mailgun

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	sdk "github.com/mailgun/mailgun-go/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

type sent struct{ from, to, subject, text, html string }

type fakeSender struct {
	results []error
	calls   []sent
	onCall  func()
}

func (f *fakeSender) Send(_ context.Context, from, to, subject, text, html string) error {
	f.calls = append(f.calls, sent{from, to, subject, text, html})
	if f.onCall != nil {
		f.onCall()
	}
	if len(f.results) == 0 {
		return nil
	}
	err := f.results[0]
	f.results = f.results[1:]
	return err
}

func newTestClient(f *fakeSender) *Client {
	return &Client{
		sender:  f,
		from:    "9itchi <no-reply@example.com>",
		limiter: rate.NewLimiter(rate.Inf, 1),
		ttl:     15 * time.Minute,
		retryIn: func(int) time.Duration { return time.Millisecond },
	}
}

func respErr(status int) error {
	return &sdk.UnexpectedResponseError{Actual: status, URL: "https://secret-url", Data: []byte("secret-body 123456")}
}

func TestSendPasswordResetCode_Success(t *testing.T) {
	f := &fakeSender{}
	require.NoError(t, newTestClient(f).SendPasswordResetCode(context.Background(), "user@example.com", "123456"))
	require.Len(t, f.calls, 1)
	m := f.calls[0]
	assert.Equal(t, "user@example.com", m.to)
	assert.Equal(t, "Your 9itchi password reset code", m.subject)
	assert.Contains(t, m.text, "123456")
	assert.Contains(t, m.text, "15 minutes")
	assert.Contains(t, m.html, "123456")
	assert.Contains(t, m.html, "15 minutes")
}

func TestSendPasswordResetCode_RetriesOn5xx(t *testing.T) {
	f := &fakeSender{results: []error{respErr(503), errors.New("connection reset")}}
	require.NoError(t, newTestClient(f).SendPasswordResetCode(context.Background(), "user@example.com", "123456"))
	assert.Len(t, f.calls, 3)
}

func TestSendPasswordResetCode_GivesUpAfterThreeAttempts(t *testing.T) {
	f := &fakeSender{results: []error{respErr(500), respErr(500), respErr(500), respErr(500)}}
	err := newTestClient(f).SendPasswordResetCode(context.Background(), "user@example.com", "123456")
	require.Error(t, err)
	assert.Len(t, f.calls, 3)
}

func TestSendPasswordResetCode_NoRetryOn4xx(t *testing.T) {
	f := &fakeSender{results: []error{respErr(400)}}
	err := newTestClient(f).SendPasswordResetCode(context.Background(), "user@example.com", "123456")
	require.Error(t, err)
	assert.Len(t, f.calls, 1)
	for _, secret := range []string{"123456", "user@example.com", "secret-url", "secret-body"} {
		assert.False(t, strings.Contains(err.Error(), secret), "error leaks %q", secret)
	}
}

func TestSendPasswordResetCode_ContextCancelStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	f := &fakeSender{results: []error{respErr(503), respErr(503), respErr(503)}, onCall: cancel}
	c := newTestClient(f)
	c.retryIn = func(int) time.Duration { return time.Hour }
	err := c.SendPasswordResetCode(ctx, "user@example.com", "123456")
	require.ErrorIs(t, err, context.Canceled)
	assert.Len(t, f.calls, 1)
	assert.NotContains(t, err.Error(), "123456")
}

func TestNormalizeBase(t *testing.T) {
	assert.Equal(t, "https://api.eu.mailgun.net/v3", normalizeBase("https://api.eu.mailgun.net"))
	assert.Equal(t, "https://api.eu.mailgun.net/v3", normalizeBase("https://api.eu.mailgun.net/v3/"))
}
