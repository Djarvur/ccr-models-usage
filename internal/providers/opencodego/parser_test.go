package opencodego

import (
	"strings"
	"testing"
	"time"

	"github.com/Djarvur/ccr-models-usage/internal/provider"
)

func TestParseUsage_EscapedJSONAllWindows(t *testing.T) {
	body := `self.__next_f.push([1, "{\"rollingUsage\":{\"usagePercent\":12.5,\"resetInSec\":3600},\"weeklyUsage\":{\"usagePercent\":25,\"resetInSec\":7200},\"monthlyUsage\":{\"usagePercent\":50,\"resetInSec\":10800}}"])`
	before := time.Now()
	limits, err := parseUsage([]byte(body))
	after := time.Now()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(limits) != 3 {
		t.Fatalf("expected 3 limits, got %d", len(limits))
	}
	if limits[0].Label != "rolling (5h)" {
		t.Errorf("expected rolling (5h), got %q", limits[0].Label)
	}
	if limits[0].UsedPct != 12.5 {
		t.Errorf("expected 12.5%%, got %v", limits[0].UsedPct)
	}
	if limits[0].ResetAt == nil {
		t.Fatalf("expected resetAt to be set")
	}
	resetAt := *limits[0].ResetAt
	expected := before.Add(time.Hour).Add(-after.Sub(before))
	if resetAt.Before(expected) || resetAt.After(after.Add(time.Hour)) {
		t.Errorf("resetAt %v not within [%v, %v]", resetAt, expected, after.Add(time.Hour))
	}
}

func TestParseUsage_OnlyRolling(t *testing.T) {
	body := `__next_f.push([1, "{\"rollingUsage\":{\"usagePercent\":40,\"resetInSec\":13320}}"])`
	limits, err := parseUsage([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(limits) != 1 {
		t.Fatalf("expected 1 limit, got %d", len(limits))
	}
	if limits[0].Label != "rolling (5h)" {
		t.Errorf("expected rolling (5h), got %q", limits[0].Label)
	}
}

func TestParseUsage_DollarRForm(t *testing.T) {
	body := `some:function($R1, $R2 = { mine: true, rollingUsage: $R3 = { status: "ok", resetInSec: 3600, usagePercent: 12.5 }, weeklyUsage: $R4 = { resetInSec: 7200, usagePercent: 25 } })`
	limits, err := parseUsage([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(limits) != 2 {
		t.Fatalf("expected 2 limits, got %d", len(limits))
	}
}

func TestParseUsage_NoWindows(t *testing.T) {
	body := `__next_f.push([1, "{\"unrelated\":true}"])`
	_, err := parseUsage([]byte(body))
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "dashboard markup may have changed") {
		t.Errorf("expected markup error, got %q", err.Error())
	}
}

func TestParseUsage_Cloudflare(t *testing.T) {
	body := `<html>cloudflare challenge</html>`
	_, err := parseUsage([]byte(body))
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "dashboard markup may have changed") {
		t.Errorf("expected markup error, got %q", err.Error())
	}
}

func TestParseUsage_EmptyLimitsIsError(t *testing.T) {
	// If the body contains the marker but no parseable windows, we
	// still want to return an error so the renderer can show "skip".
	_, err := parseUsage([]byte("__next_f.push([1, '{\"unrelated\":true}'])"))
	if err == nil {
		t.Fatalf("expected error for empty limits, got nil")
	}
	if !strings.Contains(err.Error(), "dashboard markup may have changed") {
		t.Errorf("expected markup error, got %q", err.Error())
	}
}

// silence unused imports if test file shrinks.
var _ = provider.Limit{}
