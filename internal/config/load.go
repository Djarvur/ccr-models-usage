package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// DefaultConfigPath returns the default path to the CCR config file:
// $HOME/.claude-code-router/config.json.
func DefaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot resolve home dir: %w", err)
	}

	return filepath.Join(home, ".claude-code-router", "config.json"), nil
}

// ErrUnreadable is the sentinel returned by Load on read or decode
// failure. Callers map it to a non-zero exit code.
var ErrUnreadable = errors.New("config unreadable")

// Load reads, parses, and returns the RawConfig stored at path. Any
// problems (missing file, bad JSON) are written as a one-line message
// to stderr and returned wrapped around ErrUnreadable.
func Load(path string, stderr io.Writer) (RawConfig, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- path is supplied by the user via -config
	if err != nil {
		writeStderrf(stderr, "config: cannot read %s: %v\n", path, err)

		return RawConfig{}, fmt.Errorf("%w: %w", ErrUnreadable, err)
	}

	var raw RawConfig

	err = json.Unmarshal(data, &raw)
	if err != nil {
		writeStderrf(stderr, "config: %s: %v\n", path, err)

		return RawConfig{}, fmt.Errorf("%w: %w", ErrUnreadable, err)
	}

	return raw, nil
}

// IsUnreadable reports whether err originated from a read or decode
// failure in Load.
func IsUnreadable(err error) bool {
	return errors.Is(err, ErrUnreadable)
}

// writeStderrf writes to stderr, ignoring errors (the writer is best-
// effort anyway; we cannot do anything useful with a failure here).
func writeStderrf(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}
