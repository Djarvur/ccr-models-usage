package provider

import (
	"context"
	"testing"
)

type fakeAdapter struct {
	host      string
	needsAuth bool
	limits    Limits
}

func (a *fakeAdapter) Host() string            { return a.host }
func (a *fakeAdapter) NeedsSessionCreds() bool { return a.needsAuth }
func (a *fakeAdapter) Fetch(_ context.Context, _ string) (FetchResult, error) {
	return FetchResult{Limits: a.limits}, nil
}

func TestRegistry_MatchByHost(t *testing.T) {
	r := NewRegistry()
	a := &fakeAdapter{host: "api.z.ai"}
	r.Register(a)

	if got := r.Match("api.z.ai"); got != a {
		t.Errorf("expected adapter, got %v", got)
	}
}

func TestRegistry_MatchUnknownReturnsNil(t *testing.T) {
	r := NewRegistry()
	if got := r.Match("nope.example.com"); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestRegistry_TwoProvidersSameHostShareAdapter(t *testing.T) {
	r := NewRegistry()
	a := &fakeAdapter{host: "opencode.ai"}
	r.Register(a)

	got1 := r.Match("opencode.ai")
	got2 := r.Match("opencode.ai")
	if got1 != a || got2 != a {
		t.Errorf("expected same adapter instance, got %v and %v", got1, got2)
	}
}
