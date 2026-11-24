package core

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type blockingMailer struct {
	release chan struct{}
	done    chan error
}

func (m blockingMailer) SendPasswordResetCode(context.Context, string, string) error {
	<-m.release
	err := errors.New("send failed")
	m.done <- err
	return err
}

type funcMailer func(ctx context.Context) error

func (f funcMailer) SendPasswordResetCode(ctx context.Context, _, _ string) error { return f(ctx) }

func TestAsyncMailerSurvivesCancelledRequestAndPanics(t *testing.T) {
	reported := make(chan error, 1)
	ctxErr := make(chan error, 1)
	var calls int
	inner := funcMailer(func(ctx context.Context) error {
		calls++
		if calls == 1 {
			ctxErr <- ctx.Err()
			return nil
		}
		panic("code 123456")
	})
	mailer := AsyncMailer{Mailer: inner, OnError: func(err error) { reported <- err }}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	require.NoError(t, mailer.SendPasswordResetCode(cancelled, "a@b.c", "123456"))
	select {
	case err := <-ctxErr:
		require.NoError(t, err, "delivery must not inherit the request cancellation")
	case <-time.After(2 * time.Second):
		t.Fatal("mailer was not called")
	}

	require.NoError(t, mailer.SendPasswordResetCode(context.Background(), "a@b.c", "123456"))
	select {
	case err := <-reported:
		require.EqualError(t, err, "mailer panicked")
	case <-time.After(2 * time.Second):
		t.Fatal("panic was not reported")
	}
}

func TestAsyncMailerReturnsBeforeDeliveryAndReportsErrors(t *testing.T) {
	m := blockingMailer{release: make(chan struct{}), done: make(chan error, 1)}
	reported := make(chan error, 1)
	mailer := AsyncMailer{Mailer: m, OnError: func(err error) { reported <- err }}

	require.NoError(t, mailer.SendPasswordResetCode(context.Background(), "a@b.c", "123456"))
	close(m.release)

	select {
	case err := <-reported:
		require.Error(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("delivery error was not reported")
	}
}
