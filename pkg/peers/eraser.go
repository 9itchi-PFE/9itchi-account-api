// Package peers calls the route every data-owning service offers to erase what it keeps about one user.
// Base URLs come from env, never hard-coded.
package peers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Eraser is one service that keeps data about users.
type Eraser struct {
	// Service is the name used in logs and in the answer when the service fails ("resume", "email"…).
	Service string
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

// NewEraser builds the eraser of one service. The timeout is short: seven of them run in a row.
func NewEraser(service, baseURL, apiKey string) *Eraser {
	return &Eraser{Service: service, BaseURL: baseURL, APIKey: apiKey, HTTP: &http.Client{Timeout: 15 * time.Second}}
}

func (e *Eraser) Name() string { return e.Service }

// EraseUser calls DELETE /internal/users/:id. Only 204 means the data is gone: a service that does not know the
// route yet (404) must not be taken for a service with nothing to erase.
func (e *Eraser) EraseUser(ctx context.Context, userID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, strings.TrimRight(e.BaseURL, "/")+"/internal/users/"+url.PathEscape(userID), nil)
	if err != nil {
		return err
	}
	req.Header.Set("x-api-key", e.APIKey)
	resp, err := e.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("%s: unreachable", e.Service)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("%s: status %d", e.Service, resp.StatusCode)
	}
	return nil
}
