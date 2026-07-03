package opencodego

import (
	"context"
	"os"
	"path/filepath"

	"github.com/Djarvur/ccr-models-usage/internal/provider"
)

// Adapter is the OpenCode Go implementation of provider.Adapter.
//
// It authenticates with a (username, password) pair resolved from
// env vars or a JSON config file, fetches the dashboard HTML, and
// parses out the rolling/weekly/monthly usage windows.
type Adapter struct {
	credsPath string
	authURL   string
	dashURL   string
}

// New returns a new OpenCode Go Adapter.
func New() *Adapter {
	return &Adapter{
		authURL: AuthURL,
		dashURL: "https://opencode.ai/dashboard",
	}
}

// Host returns the canonical hostname the adapter services.
func (a *Adapter) Host() string { return "opencode.ai" }

// NeedsSessionCreds reports that the OpenCode Go adapter needs
// (username, password) on top of the API key.
func (a *Adapter) NeedsSessionCreds() bool { return true }

// Fetch resolves credentials, authenticates, fetches the dashboard,
// parses the usage windows and returns them.
func (a *Adapter) Fetch(ctx context.Context, _ string) (provider.FetchResult, error) {
	creds, err := ResolveCreds(a.credsPath)
	if err != nil {
		return provider.FetchResult{}, err
	}

	session, err := Authenticate(ctx, a.authURL, creds.Username, creds.Password)
	if err != nil {
		return provider.FetchResult{}, err
	}

	body, err := fetchDashboard(ctx, a.dashURL, session)
	if err != nil {
		return provider.FetchResult{}, err
	}

	limits, err := parseUsage(body)
	if err != nil {
		return provider.FetchResult{}, err
	}

	return provider.FetchResult{Limits: limits, Level: "Go"}, nil
}

// WithCredsPath returns a copy of the adapter that reads credentials
// from the given JSON file path (overriding the default lookup).
func (a *Adapter) WithCredsPath(path string) *Adapter {
	clone := *a
	clone.credsPath = path

	return &clone
}

// DefaultCredsPath is the default location of the credentials file
// (used by the CLI's main package).
func DefaultCredsPath() string {
	if env := os.Getenv("XDG_CONFIG_HOME"); env != "" {
		return filepath.Join(env, "ccr-models-usage", "opencode-go.json")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, ".config", "ccr-models-usage", "opencode-go.json")
}
