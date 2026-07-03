package opencodego

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// ErrCookieExpired is returned by fetchDashboard when the session has
// expired and the server returns 401/403/302.
var ErrCookieExpired = errors.New("opencode-go: cookie expired")

// ErrDashboardMarkupChanged is returned by fetchDashboard and the
// parser when the dashboard HTML does not contain any parsable usage
// data.
var ErrDashboardMarkupChanged = errors.New("opencode-go: dashboard markup may have changed")

// ErrDashboardServerError is wrapped around a non-2xx response from
// the dashboard endpoint. The message carries the last seen HTTP
// status.
var ErrDashboardServerError = errors.New("opencode-go: dashboard server error")

// fetchDashboard GETs the dashboard URL with the supplied session
// cookie and the desktop-Chrome User-Agent. It returns the raw HTML
// body or one of: ErrAuthFailed, ErrCookieExpired.
func fetchDashboard(ctx context.Context, url string, session Session) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build dashboard request: %w", err)
	}

	req.Header.Set("Cookie", session.Cookie)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("User-Agent", UserAgent)

	client := &http.Client{Timeout: httpClientTimeout}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("opencode-go: dashboard: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return nil, fmt.Errorf("%w (HTTP %d)", ErrAuthFailed, resp.StatusCode)
	case resp.StatusCode >= 300 && resp.StatusCode < 400:
		return nil, fmt.Errorf("%w (HTTP %d redirect)", ErrCookieExpired, resp.StatusCode)
	case resp.StatusCode < 200 || resp.StatusCode >= 300:
		return nil, fmt.Errorf("%w: HTTP %d", ErrDashboardServerError, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read dashboard body: %w", err)
	}

	if !bytes.Contains(body, []byte("__next_f.push")) {
		return nil, fmt.Errorf("%w: no __next_f.push in body", ErrDashboardMarkupChanged)
	}

	return body, nil
}
