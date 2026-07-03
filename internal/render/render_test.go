package render

import (
	"strings"
	"testing"
	"time"

	"github.com/Djarvur/ccr-models-usage/internal/provider"
)

func TestWrite_ZaiTwoLimits(t *testing.T) {
	now := time.Date(2026, 4, 16, 6, 0, 0, 0, time.UTC)
	reset := time.Date(2026, 4, 16, 10, 31, 0, 0, time.UTC)
	rows := []Row{
		{
			Header: "zai api.z.ai Pro",
			Limits: provider.Limits{
				{Label: "TIME_LIMIT", UsedPct: 0, Detail: "remaining 100", ResetAt: &reset},
				{Label: "TOKENS_LIMIT", UsedPct: 18, ResetAt: timePtr(now.Add(3 * time.Hour))},
			},
		},
	}
	out := renderRowsAt(t, rows, now)
	if !strings.Contains(out, "zai api.z.ai Pro") {
		t.Errorf("expected header, got: %s", out)
	}
	if !strings.Contains(out, "TIME_LIMIT") {
		t.Errorf("expected TIME_LIMIT, got: %s", out)
	}
	if !strings.Contains(out, "0%") {
		t.Errorf("expected 0%%, got: %s", out)
	}
	if !strings.Contains(out, "remaining 100") {
		t.Errorf("expected 'remaining 100', got: %s", out)
	}
}

func TestWrite_OpenCodeAllWindows(t *testing.T) {
	now := time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC)
	rows := []Row{
		{
			Header: "opencode-go opencode.ai Go",
			Limits: provider.Limits{
				{Label: "rolling (5h)", UsedPct: 40, ResetAt: timePtr(now.Add(3*time.Hour + 42*time.Minute))},
				{Label: "weekly", UsedPct: 31, ResetAt: timePtr(now.Add(4*24*time.Hour + 12*time.Hour))},
				{Label: "monthly", UsedPct: 21, ResetAt: timePtr(now.Add(16*24*time.Hour + 5*time.Hour))},
			},
		},
	}
	out := renderRowsAt(t, rows, now)
	for _, want := range []string{"rolling (5h)", "weekly", "monthly", "in 3h 42m", "in 4d 12h", "in 16d 5h"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got: %s", want, out)
		}
	}
}

func TestWrite_UnknownHost(t *testing.T) {
	rows := []Row{
		{Header: "yadro litellm-proxy.ai.yadro.com", Skip: "no adapter"},
	}
	out := renderRowsAt(t, rows, time.Now())
	if !strings.Contains(out, "yadro litellm-proxy.ai.yadro.com") {
		t.Errorf("expected yadro header, got: %s", out)
	}
	if !strings.Contains(out, "skip (no adapter)") {
		t.Errorf("expected 'skip (no adapter)', got: %s", out)
	}
}

func TestWrite_AdapterError(t *testing.T) {
	rows := []Row{
		{Header: "api.z.ai", Skip: "auth failed"},
	}
	out := renderRowsAt(t, rows, time.Now())
	if !strings.Contains(out, "skip (auth failed)") {
		t.Errorf("expected 'skip (auth failed)', got: %s", out)
	}
}

func TestWrite_OrderStable(t *testing.T) {
	rows := []Row{
		{Header: "zai api.z.ai Pro", Limits: provider.Limits{{Label: "TIME_LIMIT", UsedPct: 0}}},
		{Header: "opencode-go opencode.ai Go", Limits: provider.Limits{{Label: "rolling (5h)", UsedPct: 0}}},
		{Header: "yadro litellm-proxy.ai.yadro.com", Skip: "no adapter"},
	}
	first := renderRowsAt(t, rows, time.Now())
	second := renderRowsAt(t, rows, time.Now())
	if first != second {
		t.Errorf("expected stable output, got first:\n%s\nsecond:\n%s", first, second)
	}
}

func renderRowsAt(t *testing.T, rows []Row, now time.Time) string {
	t.Helper()

	var sb strings.Builder
	if err := Write(&sb, rows, now); err != nil {
		t.Fatalf("Write: %v", err)
	}

	return sb.String()
}

func timePtr(at time.Time) *time.Time {
	return &at
}
