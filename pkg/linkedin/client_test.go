package linkedin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"9itchi-account-api/core"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fakeLinkedIn(t *testing.T, userInfoStatus int, userInfo string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "the-code", r.PostForm.Get("code"))
		assert.Equal(t, "cid", r.PostForm.Get("client_id"), "credentials go in the body")
		assert.Equal(t, "secret", r.PostForm.Get("client_secret"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"at","token_type":"Bearer","expires_in":3600}`))
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer at", r.Header.Get("Authorization"))
		w.WriteHeader(userInfoStatus)
		_, _ = w.Write([]byte(userInfo))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestAuthURL(t *testing.T) {
	c := New("cid", "secret", "http://localhost:8081/auth/linkedin/callback")
	u, err := url.Parse(c.AuthURL("st"))
	require.NoError(t, err)
	assert.Equal(t, "www.linkedin.com", u.Host)
	q := u.Query()
	assert.Equal(t, "st", q.Get("state"))
	assert.Equal(t, "code", q.Get("response_type"))
	assert.Equal(t, "openid profile email", q.Get("scope"))
	assert.Equal(t, "http://localhost:8081/auth/linkedin/callback", q.Get("redirect_uri"))
}

func TestExchange(t *testing.T) {
	t.Run("returns the verified profile", func(t *testing.T) {
		srv := fakeLinkedIn(t, http.StatusOK, `{"email":"ann@example.com","email_verified":true,"name":"Ann L"}`)
		c := newClient("cid", "secret", "http://cb", srv.URL+"/auth", srv.URL+"/token", srv.URL+"/userinfo")
		p, err := c.Exchange(context.Background(), "the-code")
		require.NoError(t, err)
		assert.Equal(t, core.OAuthProfile{Email: "ann@example.com", EmailVerified: true, Name: "Ann L"}, p)
	})

	t.Run("userinfo error status", func(t *testing.T) {
		srv := fakeLinkedIn(t, http.StatusUnauthorized, `{}`)
		c := newClient("cid", "secret", "http://cb", srv.URL+"/auth", srv.URL+"/token", srv.URL+"/userinfo")
		_, err := c.Exchange(context.Background(), "the-code")
		assert.EqualError(t, err, "linkedin userinfo status 401")
	})

	t.Run("bad userinfo body", func(t *testing.T) {
		srv := fakeLinkedIn(t, http.StatusOK, `not json`)
		c := newClient("cid", "secret", "http://cb", srv.URL+"/auth", srv.URL+"/token", srv.URL+"/userinfo")
		_, err := c.Exchange(context.Background(), "the-code")
		assert.EqualError(t, err, "linkedin userinfo decode failed")
	})

	t.Run("token endpoint failure never leaks the code", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}))
		defer srv.Close()
		c := newClient("cid", "secret", "http://cb", srv.URL+"/auth", srv.URL+"/token", srv.URL+"/userinfo")
		_, err := c.Exchange(context.Background(), "the-code")
		assert.EqualError(t, err, "linkedin code exchange failed")
	})
}
