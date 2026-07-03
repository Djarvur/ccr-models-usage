package opencodego

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAuthenticate_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		_ = r.ParseForm()
		if r.FormValue("username") != "alice" || r.FormValue("password") != "secret" {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "cookie-value"})
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	session, err := Authenticate(context.Background(), server.URL, "alice", "secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.Cookie != "session=cookie-value" {
		t.Errorf("expected session cookie, got %q", session.Cookie)
	}
}

func TestAuthenticate_BadCreds(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/login", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	_, err := Authenticate(context.Background(), server.URL, "alice", "wrong")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, ErrAuthFailed) {
		t.Errorf("expected ErrAuthFailed, got %v", err)
	}
	if !strings.Contains(err.Error(), "auth failed") {
		t.Errorf("expected error to contain 'auth failed', got %q", err.Error())
	}
}

func TestAuthenticate_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/login", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	_, err := Authenticate(context.Background(), server.URL, "alice", "secret")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error to mention 500, got %q", err.Error())
	}
}
