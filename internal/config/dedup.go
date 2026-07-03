package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
)

// RawConfig matches the on-disk shape of ~/.claude-code-router/config.json.
// Unknown fields are ignored.
type RawConfig struct {
	Providers []RawProvider `json:"Providers"`
}

// RawProvider is one entry in Providers. JSON tags follow the CCR file
// casing: snake_case keys with capitalised first letter for the
// `Providers` array.
type RawProvider struct {
	Name       string   `json:"name"`
	APIBaseURL string   `json:"api_base_url"`
	APIKey     string   `json:"api_key"`
	Models     []string `json:"models"`
}

// Provider is a deduplicated provider entry, ready to be matched against
// adapters and rendered.
type Provider struct {
	Name   string
	Host   string
	Key    string
	Models []string
}

// ErrNoHostname is returned by Dedup when a provider's api_base_url has
// no parseable hostname.
var ErrNoHostname = errors.New("api_base_url has no hostname")

// ErrBadAPIBaseURL is returned when api_base_url cannot be parsed as a
// URL at all.
var ErrBadAPIBaseURL = errors.New("api_base_url is unparseable")

// Dedup collapses RawConfig.Providers by (hostname, api_key). The first
// occurrence wins on Name; Models is the union of all matching entries
// in source order, deduplicated.
//
// Entries with an unparseable api_base_url are silently skipped (with a
// warning printed to stderr); they MUST NOT abort the rest of the run.
func Dedup(raw RawConfig) []Provider {
	seen := make(map[string]Provider)
	order := make([]string, 0, len(raw.Providers))

	for _, provider := range raw.Providers {
		host, err := hostname(provider.APIBaseURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skipping provider %q: %v\n", provider.Name, err)

			continue
		}

		id := host + "|" + provider.APIKey
		if existing, ok := seen[id]; ok {
			existing.Models = unionStrings(existing.Models, provider.Models)
			seen[id] = existing

			continue
		}

		seen[id] = Provider{
			Name:   provider.Name,
			Host:   host,
			Key:    provider.APIKey,
			Models: copyStrings(provider.Models),
		}
		order = append(order, id)
	}

	out := make([]Provider, 0, len(order))
	for _, key := range order {
		out = append(out, seen[key])
	}

	return out
}

func hostname(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("%w: %q: %w", ErrBadAPIBaseURL, rawURL, err)
	}

	h := u.Hostname()
	if h == "" {
		return "", fmt.Errorf("%w: %q", ErrNoHostname, rawURL)
	}

	return strings.ToLower(h), nil
}

func unionStrings(left, right []string) []string {
	seen := make(map[string]struct{}, len(left))
	out := make([]string, 0, len(left)+len(right))

	for _, item := range left {
		if _, ok := seen[item]; ok {
			continue
		}

		seen[item] = struct{}{}
		out = append(out, item)
	}

	for _, item := range right {
		if _, ok := seen[item]; ok {
			continue
		}

		seen[item] = struct{}{}
		out = append(out, item)
	}

	return out
}

func copyStrings(in []string) []string {
	if in == nil {
		return nil
	}

	out := make([]string, len(in))
	copy(out, in)

	return out
}
