package provider

import (
	"context"
	"time"
)

// Adapter is implemented by every provider integration (z.ai,
// OpenCode Go, ...). Adapters are stateless apart from any internal
// HTTP client configuration and may be shared across providers that
// share a host.
type Adapter interface {
	// Host returns the hostname (lower-case, no port) that the
	// adapter services, e.g. "api.z.ai".
	Host() string

	// Fetch queries the provider's usage API using the API key
	// supplied by the CCR config and returns the usage limits
	// (plus optional metadata, like a tariff name).
	// Implementations MUST honour ctx cancellation and any
	// per-call timeout derived from it.
	Fetch(ctx context.Context, apiKey string) (FetchResult, error)

	// NeedsSessionCreds reports whether the adapter needs extra
	// credentials (env vars or a config file) on top of the API
	// key. The OpenCode Go adapter returns true; z.ai returns
	// false.
	NeedsSessionCreds() bool
}

// FetchResult bundles the limits returned by an adapter with
// optional metadata the renderer can use (e.g. the tariff name).
type FetchResult struct {
	Limits Limits
	Level  string
}

// Limit is one usage bucket returned by an Adapter. UsedPct is in the
// closed range [0, 100] and represents the amount used (not the
// amount remaining). ResetAt may be nil if the provider does not
// advertise a reset time for the limit.
type Limit struct {
	Label   string
	UsedPct float64
	ResetAt *time.Time
	Detail  string
}

// Limits is a list of Limit values, in the order the adapter produced
// them.
type Limits []Limit
