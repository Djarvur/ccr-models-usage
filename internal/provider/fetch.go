package provider

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"
)

// DefaultFetchTimeout is the per-call timeout applied to each Adapter.Fetch
// invocation. Callers may pass a context with a tighter deadline to
// override it.
const DefaultFetchTimeout = 10 * time.Second

// MaxConcurrentFetches caps the number of in-flight Adapter.Fetch
// calls at any moment. The spec demands <=4; we hard-code 4 to match.
const MaxConcurrentFetches = 4

// Provider is a deduplicated provider entry that the fetcher can run
// against. The type lives here (not in internal/config) so the
// provider package has no dependency on internal/config.
type Provider struct {
	Name string
	Host string
	Key  string
}

// Result is what FetchAll emits per input provider: the provider
// itself plus the limits (if any), the optional level/tariff name
// (if any) and the error (if any) returned by the matching adapter.
type Result struct {
	Provider Provider
	Limits   Limits
	Level    string
	Err      error
}

// ErrNoAdapter is the sentinel error stored in Result.Err when the
// provider's host has no registered adapter.
var ErrNoAdapter = errors.New("no adapter")

// FetchAll queries every provider that has a matching adapter in
// parallel, capping concurrency at MaxConcurrentFetches. The output
// slice has one entry per input provider, in the same order.
//
// Errors are captured per-provider; one provider's failure MUST NOT
// affect the others.
func FetchAll(ctx context.Context, registry *Registry, providers []Provider) []Result {
	results := make([]Result, len(providers))
	sem := make(chan struct{}, MaxConcurrentFetches)

	var wg sync.WaitGroup
	for idx, prov := range providers {
		wg.Add(1)

		go func(idx int, prov Provider) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			results[idx] = fetchOne(ctx, registry, prov)
		}(idx, prov)
	}

	wg.Wait()

	return results
}

func fetchOne(ctx context.Context, registry *Registry, prov Provider) Result {
	adapter := registry.Match(prov.Host)
	if adapter == nil {
		return Result{Provider: prov, Limits: nil, Level: "", Err: ErrNoAdapter}
	}

	callCtx, cancel := context.WithTimeout(ctx, DefaultFetchTimeout)
	defer cancel()

	out, err := adapter.Fetch(callCtx, prov.Key)

	return Result{Provider: prov, Limits: out.Limits, Level: out.Level, Err: err}
}

// SortedResults returns results in a stable, predictable order: by
// provider name within each host. The implementation sorts a copy and
// leaves the input slice unchanged.
func SortedResults(in []Result) []Result {
	out := make([]Result, len(in))
	copy(out, in)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Provider.Host != out[j].Provider.Host {
			return out[i].Provider.Host < out[j].Provider.Host
		}

		return out[i].Provider.Name < out[j].Provider.Name
	})

	return out
}
