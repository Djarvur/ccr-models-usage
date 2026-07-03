package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfigPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := DefaultConfigPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(home, ".claude-code-router", "config.json")
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestLoad_FileMissing(t *testing.T) {
	dir := t.TempDir()
	stderr := &bytes.Buffer{}
	cfg, err := Load(filepath.Join(dir, "nope.json"), stderr)
	if err == nil {
		t.Fatalf("expected error, got cfg=%v", cfg)
	}
	if !strings.Contains(err.Error(), "config") {
		t.Errorf("expected error to mention config, got %q", err.Error())
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("{not valid"), 0o600); err != nil {
		t.Fatal(err)
	}
	stderr := &bytes.Buffer{}
	_, err := Load(path, stderr)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestLoad_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ok.json")
	body := `{"Providers":[{"name":"zai","api_base_url":"https://api.z.ai/v1","api_key":"k","models":["a"]}]}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(cfg.Providers))
	}
}
