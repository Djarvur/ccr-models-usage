//go:build integration

package config

import (
	"bytes"
	"os"
	"testing"
)

// TestIntegration_RealCCRConfig exercises Load against the user's real
// ~/.claude-code-router/config.json. It runs only when the
// `integration` build tag is supplied AND CCR_CONFIG_TEST is set to a
// non-empty value; otherwise it is skipped.
func TestIntegration_RealCCRConfig(t *testing.T) {
	if os.Getenv("CCR_CONFIG_TEST") == "" {
		t.Skip("CCR_CONFIG_TEST not set; skipping integration test")
	}
	path, err := DefaultConfigPath()
	if err != nil {
		t.Fatalf("DefaultConfigPath: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Skipf("config file %s not present: %v", path, err)
	}
	cfg, err := Load(path, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Providers) == 0 {
		t.Errorf("expected at least one provider, got 0")
	}
	providers := Dedup(cfg)
	if len(providers) == 0 {
		t.Errorf("Dedup produced 0 providers")
	}
}
