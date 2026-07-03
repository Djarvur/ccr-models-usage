package render

import (
	"testing"
	"time"
)

func TestFormatResetAt_RelativeHours(t *testing.T) {
	now := time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC)
	at := now.Add(3*time.Hour + 42*time.Minute)

	got := formatResetAt(at, now)
	if got != "in 3h 42m" {
		t.Errorf("expected 'in 3h 42m', got %q", got)
	}
}

func TestFormatResetAt_RelativeDaysHours(t *testing.T) {
	now := time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC)
	at := now.Add(4*24*time.Hour + 12*time.Hour)

	got := formatResetAt(at, now)
	if got != "in 4d 12h" {
		t.Errorf("expected 'in 4d 12h', got %q", got)
	}
}

func TestFormatResetAt_RelativeDays(t *testing.T) {
	now := time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC)
	at := now.Add(16*24*time.Hour + 5*time.Hour)

	got := formatResetAt(at, now)
	if got != "in 16d 5h" {
		t.Errorf("expected 'in 16d 5h', got %q", got)
	}
}

func TestFormatResetAt_Absolute(t *testing.T) {
	now := time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC)
	loc, err := time.LoadLocation("Local")
	if err != nil {
		t.Skip("no Local timezone")
	}
	now = now.In(loc)
	at := now.Add(31 * 24 * time.Hour)

	got := formatResetAt(at, now)
	// YYYY-MM-DD HH:MM
	if len(got) < len("2026-08-03 12:00") {
		t.Errorf("expected absolute date, got %q", got)
	}
}

func TestFormatResetAt_NowForPast(t *testing.T) {
	now := time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC)
	at := now.Add(-1 * time.Hour)

	got := formatResetAt(at, now)
	if got != "now" {
		t.Errorf("expected 'now', got %q", got)
	}
}
