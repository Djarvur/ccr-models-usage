package opencodego

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/Djarvur/ccr-models-usage/internal/provider"
)

const (
	labelRolling = "rolling (5h)"
	labelWeekly  = "weekly"
	labelMonthly = "monthly"
)

// escapedJSONRe matches a single __next_f.push([N, "..."]) call and
// captures the JSON string. The pattern is greedy enough to handle
// strings that contain escaped quotes.
var escapedJSONRe = regexp.MustCompile(`__next_f\.push\(\[\s*\d+\s*,\s*"((?:[^"\\]|\\.)*)"\s*\]\)`)

// dollarRReUsageFirst matches the rollingUsage/weeklyUsage/monthlyUsage
// windows in the `$R\d = { ... usagePercent: X, resetInSec: N ... }`
// form (usagePercent before resetInSec).
var dollarRReUsageFirst = regexp.MustCompile(
	`(rollingUsage|weeklyUsage|monthlyUsage)\s*:\s*\$R\d+\s*=\s*\{[^{}]*?usagePercent:\s*([\d.]+)[^{}]*?resetInSec:\s*(\d+)`,
)

// dollarRReResetFirst matches the alternative order (resetInSec first,
// usagePercent after).
var dollarRReResetFirst = regexp.MustCompile(
	`(rollingUsage|weeklyUsage|monthlyUsage)\s*:\s*\$R\d+\s*=\s*\{[^{}]*?resetInSec:\s*(\d+)[^{}]*?usagePercent:\s*([\d.]+)`,
)

type usageWindow struct {
	UsagePercent float64 `json:"usagePercent"`
	ResetInSec   int     `json:"resetInSec"`
}

// parseUsage extracts all known usage windows from the dashboard
// body. The result contains one provider.Limit per found window, in
// the order (rolling, weekly, monthly). If no window is parseable,
// an error wrapping ErrDashboardMarkupChanged is returned.
func parseUsage(body []byte) (provider.Limits, error) {
	windows := collectWindows(body)
	if len(windows) == 0 {
		return nil, fmt.Errorf("%w: no usage windows in body", ErrDashboardMarkupChanged)
	}

	now := time.Now()
	order := []string{"rollingUsage", "weeklyUsage", "monthlyUsage"}
	labels := map[string]string{
		"rollingUsage": labelRolling,
		"weeklyUsage":  labelWeekly,
		"monthlyUsage": labelMonthly,
	}

	out := make(provider.Limits, 0, len(windows))
	for _, key := range order {
		w, ok := windows[key]
		if !ok {
			continue
		}
		if w.UsagePercent == 0 && w.ResetInSec == 0 {
			continue
		}
		ts := now.Add(time.Duration(w.ResetInSec) * time.Second)
		out = append(out, provider.Limit{
			Label:   labels[key],
			UsedPct: w.UsagePercent,
			ResetAt: &ts,
		})
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("%w: no usage windows in body", ErrDashboardMarkupChanged)
	}

	return out, nil
}

func collectWindows(body []byte) map[string]usageWindow {
	windows := map[string]usageWindow{}
	now := time.Now()
	_ = now // referenced for future per-window "now" if needed

	// Form 1: __next_f.push([1, "{\"rollingUsage\":{...},...}"]).
	for _, match := range escapedJSONRe.FindAllSubmatch(body, -1) {
		rawJSON, unquoteErr := strconv.Unquote(`"` + string(match[1]) + `"`)
		if unquoteErr != nil {
			continue
		}
		var envelope struct {
			RollingUsage *usageWindow `json:"rollingUsage"`
			WeeklyUsage  *usageWindow `json:"weeklyUsage"`
			MonthlyUsage *usageWindow `json:"monthlyUsage"`
		}

		unmarshalErr := json.Unmarshal([]byte(rawJSON), &envelope)
		if unmarshalErr != nil {
			continue
		}
		if envelope.RollingUsage != nil {
			windows["rollingUsage"] = *envelope.RollingUsage
		}
		if envelope.WeeklyUsage != nil {
			windows["weeklyUsage"] = *envelope.WeeklyUsage
		}
		if envelope.MonthlyUsage != nil {
			windows["monthlyUsage"] = *envelope.MonthlyUsage
		}
	}

	// Form 2: $R[N] = { ... rollingUsage: $R[M] = { usagePercent: X, resetInSec: N } ... }.
	// Two field orders are accepted.
	for _, match := range dollarRReUsageFirst.FindAllSubmatch(body, -1) {
		key := string(match[1])
		pct, perr := strconv.ParseFloat(string(match[2]), 64)
		if perr != nil {
			continue
		}
		sec, serr := strconv.Atoi(string(match[3]))
		if serr != nil {
			continue
		}
		windows[key] = usageWindow{UsagePercent: pct, ResetInSec: sec}
	}
	for _, match := range dollarRReResetFirst.FindAllSubmatch(body, -1) {
		key := string(match[1])
		sec, serr := strconv.Atoi(string(match[2]))
		if serr != nil {
			continue
		}
		pct, perr := strconv.ParseFloat(string(match[3]), 64)
		if perr != nil {
			continue
		}
		windows[key] = usageWindow{UsagePercent: pct, ResetInSec: sec}
	}

	return windows
}
