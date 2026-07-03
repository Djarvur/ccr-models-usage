package opencodego

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// AuthURL is the base URL of the OpenCode Go login endpoint.
const AuthURL = "https://opencode.ai"

// ErrAuthFailed is returned when the auth endpoint rejects the
// supplied credentials.
var ErrAuthFailed = errors.New("opencode-go: auth failed")

// ErrAuthServerError is wrapped around a non-2xx response from the
// auth endpoint. The message carries the last seen HTTP status.
var ErrAuthServerError = errors.New("opencode-go: auth server error")

// Session represents the result of a successful Authenticate call.
type Session struct {
	Cookie string
}

// UserAgent is the Chrome-on-macOS UA we send for dashboard requests.
// Real Chrome is required because Cloudflare rejects the default
// Go UA with a challenge page.
const UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 " +
	"(KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"

// httpClientTimeout is the timeout for the HTTP client used to
// authenticate and fetch the dashboard.
const httpClientTimeout = 15 * time.Second

// Authenticate exchanges a (username, password) pair for a session
// cookie. The authURL is the base URL of the OpenCode Go login
// endpoint; the implementation POSTs to authURL + "/login".
func Authenticate(ctx context.Context, authURL, username, password string) (Session, error) {
	form := url.Values{}
	form.Set("username", username)
	form.Set("password", password)

	client := &http.Client{Timeout: httpClientTimeout}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, authURL+"/login", strings.NewReader(form.Encode()))
	if err != nil {
		return Session{}, fmt.Errorf("build auth request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json,text/html")
	req.Header.Set("User-Agent", UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return Session{}, fmt.Errorf("opencode-go: auth: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return Session{}, ErrAuthFailed
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, resp.Body)

		return Session{}, fmt.Errorf("%w: HTTP %d", ErrAuthServerError, resp.StatusCode)
	}

	cookie := extractSessionCookie(resp.Cookies())
	if cookie == "" {
		return Session{}, fmt.Errorf("%w: no session cookie", ErrAuthFailed)
	}

	return Session{Cookie: cookie}, nil
}

func extractSessionCookie(cookies []*http.Cookie) string {
	for _, c := range cookies {
		if c.Name == "session" {
			return c.Name + "=" + c.Value
		}
	}

	return ""
}
