package core

import "context"

// GoogleProfile is the verified-or-not identity returned by Google's userinfo endpoint.
type GoogleProfile struct {
	Email         string
	EmailVerified bool
	Name          string
}

// GoogleOAuth abstracts Google OAuth2 so services never touch the network in tests.
type GoogleOAuth interface {
	// AuthURL returns the consent URL (scopes openid email profile) for the given state.
	AuthURL(state string) string
	// Exchange trades the authorization code for the user's profile. Implementations must never log the code or tokens.
	Exchange(ctx context.Context, code string) (GoogleProfile, error)
}
