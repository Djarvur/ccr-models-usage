package render

import (
	"fmt"
	"time"
)

// formatResetAt renders a time as a relative ("in 3h 42m") or
// absolute ("2026-08-15 14:00") label. now is the reference
// "current" time; callers may pass time.Now() but tests can pass a
// fixed value.
//
// Rules:
//   - if at is in the past, returns "now".
//   - if at is less than 30 days in the future, returns "in Xd Yh"
//     or "in Yh Zm".
//   - otherwise returns the local-time "YYYY-MM-DD HH:MM".
const relativeCutoff = 30 * 24 * time.Hour

const hoursPerDay = 24

func formatResetAt(at, now time.Time) string {
	delta := at.Sub(now)
	if delta <= 0 {
		return "now"
	}

	if delta < relativeCutoff {
		return relative(delta)
	}

	return absolute(at)
}

func relative(d time.Duration) string {
	totalHours := int(d / time.Hour)
	days := totalHours / hoursPerDay
	hours := totalHours % hoursPerDay
	minutes := int((d - time.Duration(totalHours)*time.Hour) / time.Minute)

	if days > 0 {
		return fmt.Sprintf("in %dd %dh", days, hours)
	}

	if hours > 0 {
		return fmt.Sprintf("in %dh %dm", hours, minutes)
	}

	// Less than an hour: minutes only.
	mins := int(d / time.Minute)

	return fmt.Sprintf("in %dm", mins)
}

func absolute(at time.Time) string {
	return at.Format("2006-01-02 15:04")
}
