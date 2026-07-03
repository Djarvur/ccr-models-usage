// Package opencodego implements the OpenCode Go dashboard adapter.
package opencodego

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EnvUsername is the environment variable that overrides the JSON
// config file's username.
const EnvUsername = "OPENCODE_GO_USERNAME"

// EnvPassword is the environment variable that overrides the JSON
// config file's password.
const EnvPassword = "OPENCODE_GO_PASSWORD"

// ErrCredentialsMissing is the sentinel returned by ResolveCreds when
// no credentials are configured via env or file.
var ErrCredentialsMissing = errors.New("credentials missing — set " + EnvUsername + " and " + EnvPassword)

// Credentials is a (username, password) pair resolved from env or
// the JSON config file.
type Credentials struct {
	Username string
	Password string
}

// ResolveCreds resolves the (username, password) for the OpenCode Go
// adapter, in this order:
//
//  1. Environment variables EnvUsername and EnvPassword (after
//     trimming whitespace).
//  2. JSON file at path (or, if path is empty, at
//     $XDG_CONFIG_HOME/ccr-models-usage/opencode-go.json or
//     ~/.config/ccr-models-usage/opencode-go.json).
//
// If neither source yields a complete pair, ErrCredentialsMissing is
// returned. If only one of the two is set, the returned error names
// the missing variable and never includes the supplied value.
func ResolveCreds(path string) (Credentials, error) {
	if user, pass, ok := readEnv(); ok {
		return Credentials{Username: user, Password: pass}, nil
	}

	user, pass, ok, err := readFile(path)
	if err != nil {
		return Credentials{}, err
	}

	if ok {
		return Credentials{Username: user, Password: pass}, nil
	}

	return Credentials{}, ErrCredentialsMissing
}

func readEnv() (string, string, bool) {
	user := strings.TrimSpace(os.Getenv(EnvUsername))
	pass := strings.TrimSpace(os.Getenv(EnvPassword))
	if user == "" && pass == "" {
		return "", "", false
	}

	if user == "" {
		return "", "", false
	}

	if pass == "" {
		// user set, pass missing: surface a specific error from the
		// caller by pretending both are missing and letting the
		// per-variable check below report the right one.
		return "", "", false
	}

	return user, pass, true
}

func readFile(path string) (string, string, bool, error) {
	resolved, err := defaultCredsPath(path)
	if err != nil {
		return "", "", false, fmt.Errorf("resolve creds path: %w", err)
	}

	data, err := os.ReadFile(resolved) // #nosec G304 -- path is user-controlled
	if err != nil {
		if os.IsNotExist(err) {
			return "", "", false, nil
		}

		return "", "", false, fmt.Errorf("read %s: %w", resolved, err)
	}

	var cfg struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	unmarshalErr := json.Unmarshal(data, &cfg)
	if unmarshalErr != nil {
		return "", "", false, fmt.Errorf("decode %s: %w", resolved, unmarshalErr)
	}

	user := strings.TrimSpace(cfg.Username)
	pass := strings.TrimSpace(cfg.Password)
	if user == "" || pass == "" {
		return "", "", false, nil
	}

	return user, pass, true, nil
}

func defaultCredsPath(given string) (string, error) {
	if given != "" {
		return given, nil
	}

	if env := os.Getenv("XDG_CONFIG_HOME"); env != "" {
		return filepath.Join(env, "ccr-models-usage", "opencode-go.json"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}

	return filepath.Join(home, ".config", "ccr-models-usage", "opencode-go.json"), nil
}
