package core

import (
	"context"

	"9itchi-account-api/utils"

	"go.uber.org/zap"
)

// Event names exchanged over RabbitMQ (exchange "9itchi.events"). Never rename a published name.
const (
	EventUserRegistered = "user.registered"
)

// Publisher sends domain events. LogPublisher without RABBITMQ_URL, RabbitPublisher with it.
type Publisher interface {
	Publish(ctx context.Context, event string, payload any) error
}

// LogPublisher logs the event name only (never the payload).
type LogPublisher struct{}

func (LogPublisher) Publish(_ context.Context, event string, _ any) error {
	utils.Logger.Info("event", zap.String("name", event))
	return nil
}

// UserEraser is one service that keeps data about users and erases its part when an account is deleted.
type UserEraser interface {
	// Name is the service, for logs and for the answer when it fails.
	Name() string
	EraseUser(ctx context.Context, userID string) error
}
