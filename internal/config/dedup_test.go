package config

import "testing"

func TestDedup_SameHostSameKeyCollapses(t *testing.T) {
	raw := RawConfig{Providers: []RawProvider{
		{Name: "opencode", APIBaseURL: "https://opencode.ai/zen/go/v1/chat/completions", APIKey: "k1", Models: []string{"a"}},
		{Name: "opencode-a", APIBaseURL: "https://opencode.ai/zen/go/v1/messages", APIKey: "k1", Models: []string{"b"}},
	}}

	providers := Dedup(raw)
	if len(providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(providers))
	}
	p := providers[0]
	if p.Host != "opencode.ai" {
		t.Errorf("expected host opencode.ai, got %q", p.Host)
	}
	if p.Name != "opencode" {
		t.Errorf("expected first-seen name, got %q", p.Name)
	}
	want := []string{"a", "b"}
	if !equalStrings(p.Models, want) {
		t.Errorf("expected models %v, got %v", want, p.Models)
	}
}

func TestDedup_DifferentKeysStaySeparate(t *testing.T) {
	raw := RawConfig{Providers: []RawProvider{
		{Name: "yadro", APIBaseURL: "https://litellm-proxy.ai.yadro.com/v1", APIKey: "k1", Models: []string{"a"}},
		{Name: "yadev", APIBaseURL: "https://litellm-proxy.ai.yadro.com/v1", APIKey: "k2", Models: []string{"b"}},
	}}

	providers := Dedup(raw)
	if len(providers) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(providers))
	}
}

func TestDedup_DifferentHostsSameKeyStaySeparate(t *testing.T) {
	raw := RawConfig{Providers: []RawProvider{
		{Name: "zai", APIBaseURL: "https://api.z.ai/v1", APIKey: "k1", Models: []string{"a"}},
		{Name: "opencode", APIBaseURL: "https://opencode.ai/v1", APIKey: "k1", Models: []string{"b"}},
	}}

	providers := Dedup(raw)
	if len(providers) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(providers))
	}
}

func TestDedup_BadAPIBaseURLSkipped(t *testing.T) {
	raw := RawConfig{Providers: []RawProvider{
		{Name: "good", APIBaseURL: "https://api.z.ai/v1", APIKey: "k1", Models: []string{"a"}},
		{Name: "bad", APIBaseURL: "://not a url", APIKey: "k2", Models: []string{"b"}},
	}}

	providers := Dedup(raw)
	if len(providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(providers))
	}
	if providers[0].Name != "good" {
		t.Errorf("expected good, got %q", providers[0].Name)
	}
}

func TestDedup_EmptyProviders(t *testing.T) {
	raw := RawConfig{Providers: nil}
	providers := Dedup(raw)
	if len(providers) != 0 {
		t.Fatalf("expected 0 providers, got %d", len(providers))
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
