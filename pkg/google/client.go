// Package google is the real core.GoogleOAuth implementation (OAuth2 authorization code flow).
package google

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
	googleoauth "golang.org/x/oauth2/google"
)

const (
	userInfoURL = "https://openidconnect.googleapis.com/v1/userinfo"
	maxBody     = 1 << 20
)

type Client struct {
	cfg  *oauth2.Config
	http *http.Client
}

func New(clientID, clientSecret, redirectURL string) *Client {
	return &Client{
		cfg: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint:     googleoauth.Endpoint,
		},
		http: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) AuthURL(state string) string {
	return c.cfg.AuthCodeURL(state)
}

func (c *Client) Exchange(ctx context.Context, code string) (core.GoogleProfile, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, c.http)
	token, err := c.cfg.Exchange(ctx, code)
	if err != nil {
		return core.GoogleProfile{}, errors.New("google code exchange failed")
	}
	resp, err := c.cfg.Client(ctx, token).Get(userInfoURL)
	if err != nil {
		return core.GoogleProfile{}, errors.New("google userinfo request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return core.GoogleProfile{}, fmt.Errorf("google userinfo status %d", resp.StatusCode)
	}
	var info struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBody)).Decode(&info); err != nil {
		return core.GoogleProfile{}, errors.New("google userinfo decode failed")
	}
	return core.GoogleProfile{Email: info.Email, EmailVerified: info.EmailVerified, Name: info.Name}, nil
}
