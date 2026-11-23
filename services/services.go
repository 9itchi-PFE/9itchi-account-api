package services

import (
	"9itchi-account-api/core"
	"9itchi-account-api/pkg/notification"
	"9itchi-account-api/pkg/payments"
	"9itchi-account-api/repositories"
	"9itchi-account-api/utils"
)

// Services holds service implementations and the peer clients they call.
type Services struct {
	Auth    AuthService
	User    UserService
	Clients *Clients
}

// Clients holds one HTTP client per peer service (singletons, 50s timeout).
type Clients struct {
	Notification *notification.Client
	Payments     *payments.Client
}

func InitServices(config utils.AppConfig, repos *repositories.Repositories) *Services {
	return &Services{
		Auth: NewAuthService(config, repos.User, repos.Auth, repos.Store, core.NoopMailer{}),
		User: NewUserService(repos.User, repos.Auth, repos.Store),
		Clients: &Clients{
			Notification: notification.New(config.NotificationAPIURL, config.InternalAPIKey),
			Payments:     payments.New(config.PaymentsAPIURL, config.InternalAPIKey),
		},
	}
}
