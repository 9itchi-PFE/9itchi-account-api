package services

import (
	"9itchi-account-api/core"
	"9itchi-account-api/pkg/google"
	"9itchi-account-api/pkg/linkedin"
	"9itchi-account-api/pkg/mailgun"
	"9itchi-account-api/pkg/notification"
	"9itchi-account-api/pkg/payments"
	"9itchi-account-api/repositories"
	"9itchi-account-api/utils"

	"go.uber.org/zap"
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
	var googleOAuth core.GoogleOAuth
	if config.GoogleClientID != "" && config.GoogleClientSecret != "" {
		googleOAuth = google.New(config.GoogleClientID, config.GoogleClientSecret, config.GoogleRedirectURL)
	}
	var linkedinOAuth core.OAuthProvider
	if config.LinkedInClientID != "" && config.LinkedInClientSecret != "" {
		linkedinOAuth = linkedin.New(config.LinkedInClientID, config.LinkedInClientSecret, config.LinkedInRedirectURL)
	}
	var mailer core.Mailer = core.NoopMailer{}
	if config.MailgunAPIKey != "" && config.MailgunDomain != "" {
		mailer = core.AsyncMailer{
			Mailer: mailgun.New(mailgun.Config{
				APIKey:    config.MailgunAPIKey,
				Domain:    config.MailgunDomain,
				FromEmail: config.MailgunFromEmail,
				APIBase:   config.MailgunAPIBase,
				RateLimit: config.MailgunRateLimit,
			}, resetCodeTTL),
			OnError: func(err error) { utils.Logger.Error("forgot mail", zap.Error(err)) },
		}
	} else {
		utils.Logger.Info("mailgun not configured: password reset mails are disabled")
	}
	return &Services{
		Auth: NewAuthService(config, repos.User, repos.Auth, repos.Store, mailer, googleOAuth, linkedinOAuth, core.NewPublisher(config.RabbitMQURL)),
		User: NewUserService(repos.User, repos.Auth, repos.Store),
		Clients: &Clients{
			Notification: notification.New(config.NotificationAPIURL, config.InternalAPIKey),
			Payments:     payments.New(config.PaymentsAPIURL, config.InternalAPIKey),
		},
	}
}
