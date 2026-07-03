package zai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Djarvur/ccr-models-usage/internal/provider"
)

func newTestAdapter(serverURL string) *Adapter {
	return &Adapter{
		client:    &http.Client{Timeout: 5 * time.Second},
		endpoints: []string{serverURL + "/international", serverURL + "/cn"},
	}
}

func fetchLimits(t *testing.T, server *httptest.Server, key string) (provider.Limits, string) {
	t.Helper()
	adapter := newTestAdapter(server.URL)
	res, err := adapter.Fetch(context.Background(), key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	return res.Limits, res.Level
}

func TestAdapter_SuccessSingleLimit(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/international", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"code": 200,
			"msg": "ok",
			"data": {
				"level": "Pro",
				"limits": [
					{"type": "TIME_LIMIT", "percentage": 0, "remaining": 100, "nextResetTime": 1745000000000}
				]
			}
		}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	limits, level := fetchLimits(t, server, "test-key")
	if len(limits) != 1 {
		t.Fatalf("expected 1 limit, got %d", len(limits))
	}
	if limits[0].Label != "TIME_LIMIT" {
		t.Errorf("expected TIME_LIMIT, got %q", limits[0].Label)
	}
	if limits[0].UsedPct != 0 {
		t.Errorf("expected 0%%, got %v", limits[0].UsedPct)
	}
	if limits[0].Detail != "remaining 100" {
		t.Errorf("expected detail 'remaining 100', got %q", limits[0].Detail)
	}
	if level != "Pro" {
		t.Errorf("expected level Pro, got %q", level)
	}
}

func TestAdapter_SuccessMultipleLimits(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/international", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": {
				"level": "Pro",
				"limits": [
					{"type": "TIME_LIMIT", "percentage": 0, "remaining": 100, "nextResetTime": 1745000000000},
					{"type": "TOKENS_LIMIT", "percentage": 18, "nextResetTime": 1745003600000}
				]
			}
		}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	limits, _ := fetchLimits(t, server, "k")
	if len(limits) != 2 {
		t.Fatalf("expected 2 limits, got %d", len(limits))
	}
	if limits[1].Label != "TOKENS_LIMIT" {
		t.Errorf("expected TOKENS_LIMIT, got %q", limits[1].Label)
	}
}

func TestAdapter_401AuthFailed(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/international", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	adapter := newTestAdapter(server.URL)
	_, err := adapter.Fetch(context.Background(), "k")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "auth failed") {
		t.Errorf("expected error to contain 'auth failed', got %q", err.Error())
	}
}

func TestAdapter_404FallsBackToCN(t *testing.T) {
	var cnCalled bool
	mux := http.NewServeMux()
	mux.HandleFunc("/international", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("/cn", func(w http.ResponseWriter, _ *http.Request) {
		cnCalled = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"level":"Pro","limits":[{"type":"TIME_LIMIT","percentage":50}]}}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	adapter := newTestAdapter(server.URL)
	res, err := adapter.Fetch(context.Background(), "k")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cnCalled {
		t.Errorf("expected CN endpoint to be called after 404 on international")
	}
	if len(res.Limits) != 1 {
		t.Errorf("expected 1 limit, got %d", len(res.Limits))
	}
}

func TestAdapter_BothEndpoints5xxReturnsLastStatus(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/international", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("/cn", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	adapter := newTestAdapter(server.URL)
	_, err := adapter.Fetch(context.Background(), "k")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error to mention 500, got %q", err.Error())
	}
}

func TestAdapter_BadJSONReturnsDecodeError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/international", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`not json at all`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	adapter := newTestAdapter(server.URL)
	_, err := adapter.Fetch(context.Background(), "k")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "decode") {
		t.Errorf("expected error to contain 'decode', got %q", err.Error())
	}
}
