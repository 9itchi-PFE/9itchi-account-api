// Package linkedin is the real core.OAuthProvider for "Sign In with LinkedIn using OpenID Connect".
package linkedin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"9itchi-account-api/core"

	"golang.org/x/oauth2"
)

const (
	authURL     = "https://www.linkedin.com/oauth/v2/authorization"
	tokenURL    = "https://www.linkedin.com/oauth/v2/accessToken"
	userInfoURL = "https://api.linkedin.com/v2/userinfo"
	maxBody     = 1 << 20
)

type Client struct {
	cfg         *oauth2.Config
	userInfoURL string
	http        *http.Client
}

func New(clientID, clientSecret, redirectURL string) *Client {
	return newClient(clientID, clientSecret, redirectURL, authURL, tokenURL, userInfoURL)
}

func newClient(clientID, clientSecret, redirectURL, auth, token, userInfo string) *Client {
	return &Client{
		cfg: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"openid", "profile", "email"},
			// LinkedIn expects the client credentials in the form body, not in a Basic header.
			Endpoint: oauth2.Endpoint{AuthURL: auth, TokenURL: token, AuthStyle: oauth2.AuthStyleInParams},
		},
		userInfoURL: userInfo,
		http:        &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) AuthURL(state string) string {
	return c.cfg.AuthCodeURL(state)
}

func (c *Client) Exchange(ctx context.Context, code string) (core.OAuthProfile, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, c.http)
	token, err := c.cfg.Exchange(ctx, code)
	if err != nil {
		return core.OAuthProfile{}, errors.New("linkedin code exchange failed")
	}
	resp, err := c.cfg.Client(ctx, token).Get(c.userInfoURL)
	if err != nil {
		return core.OAuthProfile{}, errors.New("linkedin userinfo request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return core.OAuthProfile{}, fmt.Errorf("linkedin userinfo status %d", resp.StatusCode)
	}
	var info struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBody)).Decode(&info); err != nil {
		return core.OAuthProfile{}, errors.New("linkedin userinfo decode failed")
	}
	return core.OAuthProfile{Email: info.Email, EmailVerified: info.EmailVerified, Name: info.Name}, nil
}
