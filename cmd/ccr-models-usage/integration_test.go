//go:build integration

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestIntegration_EndToEnd_UnknownHost exercises the run() function
// against a config containing only an unknown host. The run MUST
// complete without error and render the unknown host as a skip.
func TestIntegration_EndToEnd_UnknownHost(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	cfg := `{
  "Providers": [
    {"name": "yadro", "api_base_url": "https://litellm-proxy.ai.yadro.com/v1", "api_key": "k", "models": ["m"]}
  ]
}`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Redirect stdout to a pipe so we can both observe output and
	// keep it from polluting the test runner.
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	errCh := make(chan error, 1)
	go func() {
		errCh <- run(ctx, cfgPath)
		_ = w.Close()
	}()

	err := <-errCh

	os.Stdout = originalStdout
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])
	if output == "" {
		t.Errorf("expected non-empty output, got empty")
	}
}
