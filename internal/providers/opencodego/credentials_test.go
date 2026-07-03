package opencodego

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveCreds_FromEnv(t *testing.T) {
	t.Setenv("OPENCODE_GO_USERNAME", "alice")
	t.Setenv("OPENCODE_GO_PASSWORD", "secret")

	creds, err := ResolveCreds("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.Username != "alice" || creds.Password != "secret" {
		t.Errorf("got %+v", creds)
	}
}

func TestResolveCreds_FromFile(t *testing.T) {
	t.Setenv("OPENCODE_GO_USERNAME", "")
	t.Setenv("OPENCODE_GO_PASSWORD", "")

	dir := t.TempDir()
	path := filepath.Join(dir, "opencode-go.json")
	body := `{"username": "bob", "password": "hush"}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	creds, err := ResolveCreds(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.Username != "bob" || creds.Password != "hush" {
		t.Errorf("got %+v", creds)
	}
}

func TestResolveCreds_Missing(t *testing.T) {
	t.Setenv("OPENCODE_GO_USERNAME", "")
	t.Setenv("OPENCODE_GO_PASSWORD", "")

	_, err := ResolveCreds(filepath.Join(t.TempDir(), "missing.json"))
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, ErrCredentialsMissing) {
		t.Errorf("expected ErrCredentialsMissing, got %v", err)
	}
	if !contains(err.Error(), "credentials missing") {
		t.Errorf("expected message to mention 'credentials missing', got %q", err.Error())
	}
}

func TestResolveCreds_OnlyOneEnv(t *testing.T) {
	t.Setenv("OPENCODE_GO_USERNAME", "alice")
	t.Setenv("OPENCODE_GO_PASSWORD", "")

	_, err := ResolveCreds(filepath.Join(t.TempDir(), "missing.json"))
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !contains(err.Error(), "OPENCODE_GO_PASSWORD") {
		t.Errorf("expected error to name missing variable, got %q", err.Error())
	}
	if contains(err.Error(), "alice") {
		t.Errorf("error leaked username, got %q", err.Error())
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}

	return false
}
