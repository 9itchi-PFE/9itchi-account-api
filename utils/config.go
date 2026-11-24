package utils

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type AppConfig struct {
	Port        int
	Environment string
	// DatabaseURL is the PostgreSQL connection string (env DATABASE_URL). Never log it.
	DatabaseURL string
	// SecretKey signs JWTs (HS256), EncryptionKey wraps them with AES-GCM. Shared by every service.
	SecretKey     string
	EncryptionKey string
	// InternalAPIKey is the shared secret for service-to-service calls (env INTERNAL_API_KEY).
	InternalAPIKey string
	// InternalAPIKeyRequired (env INTERNAL_API_KEY_REQUIRED, default true); false = log and allow.
	InternalAPIKeyRequired bool
	// Peer service base URLs.
	NotificationAPIURL string
	PaymentsAPIURL     string
	// AdminEmail/AdminPassword optionally bootstrap the first admin at startup (env ADMIN_EMAIL, ADMIN_PASSWORD). Never log them.
	AdminEmail    string
	AdminPassword string
	// Google sign-in is disabled (503) while GoogleClientID or GoogleClientSecret is empty. Never log the secret.
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	// LinkedIn sign-in is disabled (503) while LinkedInClientID or LinkedInClientSecret is empty. Never log the secret.
	LinkedInClientID     string
	LinkedInClientSecret string
	LinkedInRedirectURL  string
	// FrontendURL is where the Google and LinkedIn callbacks sends the browser (tokens in the URL fragment).
	FrontendURL string
	// Mailgun sends password reset mails; disabled (NoopMailer) while the API key or domain is empty. Never log the key.
	MailgunAPIKey    string
	MailgunDomain    string
	MailgunFromEmail string
	MailgunAPIBase   string
	MailgunRateLimit int
}

// LoadConfig loads configuration from environment variables.
func LoadConfig() AppConfig {
	defer lockConfig()()
	viper.AutomaticEnv()
	return AppConfig{
		Port:                   getIntWithDefault("PORT", 8081),
		Environment:            getStringWithDefault("ENVIRONMENT", "development"),
		DatabaseURL:            getStringWithDefault("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/users?sslmode=disable"),
		SecretKey:              getStringWithDefault("ITCHI_SECRET_KEY", ""),
		EncryptionKey:          getStringWithDefault("ITCHI_ENCRYPTION_KEY", ""),
		InternalAPIKey:         getStringWithDefault("INTERNAL_API_KEY", ""),
		InternalAPIKeyRequired: getBoolWithDefault("INTERNAL_API_KEY_REQUIRED", true),
		NotificationAPIURL:     getStringWithDefault("NOTIFICATION_API_URL", "http://9itchi-notification-api:8087"),
		PaymentsAPIURL:         getStringWithDefault("PAYMENTS_API_URL", "http://9itchi-payments-api:8089"),
		AdminEmail:             getStringWithDefault("ADMIN_EMAIL", ""),
		AdminPassword:          getStringWithDefault("ADMIN_PASSWORD", ""),
		GoogleClientID:         getStringWithDefault("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret:     getStringWithDefault("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:      getStringWithDefault("GOOGLE_REDIRECT_URL", "http://localhost:8081/auth/google/callback"),
		LinkedInClientID:       getStringWithDefault("LINKEDIN_CLIENT_ID", ""),
		LinkedInClientSecret:   getStringWithDefault("LINKEDIN_CLIENT_SECRET", ""),
		LinkedInRedirectURL:    getStringWithDefault("LINKEDIN_REDIRECT_URL", "http://localhost:8081/auth/linkedin/callback"),
		FrontendURL:            getStringWithDefault("FRONTEND_URL", "http://localhost:3000"),
		MailgunAPIKey:          getStringWithDefault("MAILGUN_API_KEY", ""),
		MailgunDomain:          getStringWithDefault("MAILGUN_DOMAIN", ""),
		MailgunFromEmail:       getStringWithDefault("MAILGUN_FROM_EMAIL", ""),
		MailgunAPIBase:         getStringWithDefault("MAILGUN_API_BASE", ""),
		MailgunRateLimit:       getIntWithDefault("MAILGUN_RATE_LIMIT", 10),
	}
}

// Validate fails fast on an unusable config. Only variable names are reported, never values.
func (c AppConfig) Validate() error {
	if c.Port < 1 {
		return fmt.Errorf("invalid PORT: must be a positive integer")
	}
	var bad []string
	if len(c.SecretKey) < 32 {
		bad = append(bad, "ITCHI_SECRET_KEY (min 32 chars)")
	}
	if len(c.EncryptionKey) < 32 {
		bad = append(bad, "ITCHI_ENCRYPTION_KEY (min 32 chars)")
	}
	if c.DatabaseURL == "" {
		bad = append(bad, "DATABASE_URL")
	}
	if c.InternalAPIKeyRequired && c.InternalAPIKey == "" {
		bad = append(bad, "INTERNAL_API_KEY")
	}
	if len(bad) > 0 {
		return fmt.Errorf("required environment variables are missing or invalid: %s", strings.Join(bad, ", "))
	}
	return nil
}

func getStringWithDefault(key, defaultValue string) string {
	viper.SetDefault(key, defaultValue)
	return viper.GetString(key)
}

func getIntWithDefault(key string, defaultValue int) int {
	viper.SetDefault(key, defaultValue)
	return viper.GetInt(key)
}

func getBoolWithDefault(key string, defaultValue bool) bool {
	viper.SetDefault(key, defaultValue)
	return viper.GetBool(key)
}
