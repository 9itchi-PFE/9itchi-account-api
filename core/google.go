package core

import "context"

// OAuthProfile is the identity returned by an OAuth2/OpenID Connect provider's userinfo endpoint.
type OAuthProfile struct {
	Email         string
	EmailVerified bool
	Name          string
}

// OAuthProvider abstracts one sign-in provider (Google, LinkedIn) so services never touch the network in tests.
type OAuthProvider interface {
	// AuthURL returns the consent URL (scopes openid email profile) for the given state.
	AuthURL(state string) string
	// Exchange trades the authorization code for the user's profile. Implementations must never log the code or tokens.
	Exchange(ctx context.Context, code string) (OAuthProfile, error)
}

// GoogleProfile and GoogleOAuth keep the original Google names working.
type (
	GoogleProfile = OAuthProfile
	GoogleOAuth   = OAuthProvider
)
