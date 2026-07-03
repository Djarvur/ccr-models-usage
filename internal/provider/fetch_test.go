package provider

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type slowAdapter struct {
	host     string
	delay    time.Duration
	limits   Limits
	err      error
	doneAtMu sync.Mutex
	doneAt   time.Time
}

func (a *slowAdapter) Host() string            { return a.host }
func (a *slowAdapter) NeedsSessionCreds() bool { return false }
func (a *slowAdapter) Fetch(ctx context.Context, _ string) (FetchResult, error) {
	if a.delay > 0 {
		select {
		case <-time.After(a.delay):
		case <-ctx.Done():
			a.markDone()

			return FetchResult{}, ctx.Err()
		}
	}
	a.markDone()

	return FetchResult{Limits: a.limits}, a.err
}

func (a *slowAdapter) Done() time.Time {
	a.doneAtMu.Lock()
	defer a.doneAtMu.Unlock()

	return a.doneAt
}

func (a *slowAdapter) markDone() {
	a.doneAtMu.Lock()
	defer a.doneAtMu.Unlock()
	a.doneAt = time.Now()
}

func TestFetch_SlowProviderDoesNotBlockFast(t *testing.T) {
	fast := &slowAdapter{host: "fast", delay: 50 * time.Millisecond, limits: Limits{{Label: "L", UsedPct: 1}}}
	slow := &slowAdapter{host: "slow", delay: 500 * time.Millisecond, limits: Limits{{Label: "L", UsedPct: 2}}}

	registry := NewRegistry()
	registry.Register(slow)
	registry.Register(fast)

	providers := []Provider{
		{Name: "slow", Host: "slow", Key: "k1"},
		{Name: "fast", Host: "fast", Key: "k1"},
	}

	results := FetchAll(context.Background(), registry, providers)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	fastDone := fast.Done()
	slowDone := slow.Done()
	if fastDone.IsZero() || slowDone.IsZero() {
		t.Fatalf("expected both adapters to have completed")
	}

	gap := slowDone.Sub(fastDone)
	if gap < 300*time.Millisecond {
		t.Errorf("expected slow to finish at least 300ms after fast, got gap=%v", gap)
	}
}

func TestFetch_AdapterErrorDoesNotAbort(t *testing.T) {
	good := &slowAdapter{host: "good", limits: Limits{{Label: "L", UsedPct: 1}}}
	bad := &slowAdapter{host: "bad", err: errors.New("boom")}

	registry := NewRegistry()
	registry.Register(good)
	registry.Register(bad)

	providers := []Provider{
		{Name: "good", Host: "good", Key: "k"},
		{Name: "bad", Host: "bad", Key: "k"},
	}
	results := FetchAll(context.Background(), registry, providers)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	seen := map[string]error{}
	for _, r := range results {
		seen[r.Provider.Name] = r.Err
	}

	if seen["good"] != nil {
		t.Errorf("expected nil err for good, got %v", seen["good"])
	}

	if seen["bad"] == nil {
		t.Errorf("expected err for bad, got nil")
	}
}

func TestFetch_TimeoutShowsSkipTimeout(t *testing.T) {
	slow := &slowAdapter{host: "slow", delay: 5 * time.Second, limits: Limits{{Label: "L", UsedPct: 1}}}
	registry := NewRegistry()
	registry.Register(slow)

	providers := []Provider{{Name: "slow", Host: "slow", Key: "k"}}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	results := FetchAll(ctx, registry, providers)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Err == nil {
		t.Errorf("expected timeout error, got nil")
	}
}
