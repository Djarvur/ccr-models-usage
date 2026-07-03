package opencodego

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchDashboard_Authed(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") == "" {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}
		_, _ = w.Write([]byte(`<html>__next_f.push([1, "ok"])</html>`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	body, err := fetchDashboard(context.Background(), server.URL+"/dashboard", Session{Cookie: "session=abc"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(body), "__next_f.push") {
		t.Errorf("expected __next_f.push, got %q", string(body))
	}
}

func TestFetchDashboard_401(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/dashboard", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	_, err := fetchDashboard(context.Background(), server.URL+"/dashboard", Session{Cookie: "session=abc"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "auth failed") && !strings.Contains(err.Error(), "cookie expired") {
		t.Errorf("expected auth failed / cookie expired, got %q", err.Error())
	}
}

func TestFetchDashboard_Cloudflare(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/dashboard", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html>cloudflare challenge</html>`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	_, err := fetchDashboard(context.Background(), server.URL+"/dashboard", Session{Cookie: "session=abc"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "dashboard markup may have changed") {
		t.Errorf("expected dashboard markup error, got %q", err.Error())
	}
}
