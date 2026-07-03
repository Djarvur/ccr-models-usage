// Package render produces the plain-text report of per-provider usage.
package render

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Djarvur/ccr-models-usage/internal/provider"
)

// Row is one provider's worth of output. If Skip is non-empty, the
// row is rendered as `<Header> skip (<Skip>)`. Otherwise the row's
// Limits are rendered as separate lines under the header.
type Row struct {
	Header string
	Limits provider.Limits
	Skip   string
}

// Write writes rows to w in a stable format. now is the reference
// time for relative time formatting; tests can pass a fixed value.
func Write(w io.Writer, rows []Row, now time.Time) error {
	for i, row := range rows {
		if i > 0 {
			_, err := fmt.Fprintln(w)
			if err != nil {
				return fmt.Errorf("render separator: %w", err)
			}
		}

		var err error
		if row.Skip != "" {
			err = writeSkip(w, row)
		} else {
			err = writeBlock(w, row, now)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func writeSkip(w io.Writer, row Row) error {
	header := strings.TrimSuffix(row.Header, " ")

	_, err := fmt.Fprintf(w, "%s skip (%s)\n", header, row.Skip)
	if err != nil {
		return fmt.Errorf("render skip: %w", err)
	}

	return nil
}

func writeBlock(w io.Writer, row Row, now time.Time) error {
	_, err := fmt.Fprintf(w, "%s\n", row.Header)
	if err != nil {
		return fmt.Errorf("render header: %w", err)
	}

	for _, lim := range row.Limits {
		writeErr := writeLimit(w, lim, now)
		if writeErr != nil {
			return writeErr
		}
	}

	return nil
}

func writeLimit(w io.Writer, lim provider.Limit, now time.Time) error {
	line := formatLimit(lim, now)
	_, err := fmt.Fprintln(w, line)
	if err != nil {
		return fmt.Errorf("render limit: %w", err)
	}

	return nil
}

const labelWidth = 14

// formatLimit renders one limit as a single line. Exposed for tests.
func formatLimit(lim provider.Limit, now time.Time) string {
	var sb strings.Builder
	sb.WriteString("  ")
	sb.WriteString(padRight(lim.Label, labelWidth))
	fmt.Fprintf(&sb, " %3s%%", formatPct(lim.UsedPct))

	if lim.Detail != "" {
		fmt.Fprintf(&sb, "  %s", lim.Detail)
	}

	if lim.ResetAt != nil {
		fmt.Fprintf(&sb, "  resets %s", formatResetAt(*lim.ResetAt, now))
	}

	return sb.String()
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}

	return s + strings.Repeat(" ", width-len(s))
}

func formatPct(pct float64) string {
	if pct == float64(int(pct)) {
		return fmt.Sprintf("%d", int(pct))
	}

	return fmt.Sprintf("%.1f", pct)
}
