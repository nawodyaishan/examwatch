package report

import (
	"fmt"
	"io"
	"math"
	"strings"
)

// sparkBlocks are the block characters used to render a series as a single
// line of text, scaled to the series' own min/max — the same conceptual
// approach as the live TTY sparkline (tech spec section 8), reimplemented
// locally here since report generation is a distinct concern (offline,
// full-run rendering into Markdown) from the live terminal UI's ring-buffer
// display. Duplicating ~10 lines of scaling logic is cheaper than coupling
// these two packages.
var sparkBlocks = []rune("▁▂▃▄▅▆▇█")

// sparkline renders values as a single line of block characters scaled to
// the slice's own min/max. An empty input renders as an empty string; a
// constant series renders as a flat mid-height line.
func sparkline(values []float64) string {
	if len(values) == 0 {
		return ""
	}
	min, max := values[0], values[0]
	for _, v := range values {
		min = math.Min(min, v)
		max = math.Max(max, v)
	}
	span := max - min
	var b strings.Builder
	for _, v := range values {
		var idx int
		if span == 0 {
			idx = len(sparkBlocks) / 2
		} else {
			frac := (v - min) / span
			idx = int(math.Round(frac * float64(len(sparkBlocks)-1)))
			if idx < 0 {
				idx = 0
			}
			if idx >= len(sparkBlocks) {
				idx = len(sparkBlocks) - 1
			}
		}
		b.WriteRune(sparkBlocks[idx])
	}
	return b.String()
}

// statusBadge renders a status as a bracketed, upper-case badge, e.g.
// "[PASS]", matching the style of the live UI's "[ok]"/"[warn]"/"[fail]"
// brackets without depending on that package.
func statusBadge(s Status) string {
	return "[" + string(s) + "]"
}

const timeLayout = "2006-01-02 15:04:05 MST"

// WriteMarkdown generates report.md for the given run data and writes it to
// w: an overall verdict banner, ASCII sparkline charts for each series
// across the full run, a chronological timeline of discrete events, and
// the per-signature breakdown (mirroring summary.json).
func WriteMarkdown(w io.Writer, data RunData) error {
	var b strings.Builder

	overall := data.OverallStatus()
	fmt.Fprintf(&b, "# examwatch run report\n\n")
	fmt.Fprintf(&b, "## Verdict: %s %s\n\n", overall, statusBadge(overall))

	fmt.Fprintf(&b, "| Field | Value |\n|---|---|\n")
	fmt.Fprintf(&b, "| Start time | %s |\n", data.Meta.StartTime.UTC().Format(timeLayout))
	fmt.Fprintf(&b, "| Duration | %s |\n", data.Meta.Duration)
	fmt.Fprintf(&b, "| Interval | %s |\n", data.Meta.Interval)
	fmt.Fprintf(&b, "| Hostname | %s |\n", data.Meta.Hostname)
	fmt.Fprintf(&b, "\n")

	fmt.Fprintf(&b, "## Charts\n\n")
	if len(data.Series) == 0 {
		fmt.Fprintf(&b, "_No series data recorded for this run._\n\n")
	} else {
		fmt.Fprintf(&b, "```\n")
		for _, series := range data.Series {
			values := make([]float64, len(series.Points))
			for i, p := range series.Points {
				values[i] = p.Value
			}
			label := series.Name
			if series.Unit != "" {
				label = fmt.Sprintf("%s (%s)", series.Name, series.Unit)
			}
			fmt.Fprintf(&b, "%-16s %s\n", label, sparkline(values))
		}
		fmt.Fprintf(&b, "```\n\n")
	}

	fmt.Fprintf(&b, "## Timeline\n\n")
	if len(data.Timeline) == 0 {
		fmt.Fprintf(&b, "_No discrete events recorded for this run._\n\n")
	} else {
		for _, ev := range data.Timeline {
			fmt.Fprintf(&b, "- `%s` %s\n", ev.Time.UTC().Format(timeLayout), ev.Message)
		}
		fmt.Fprintf(&b, "\n")
	}

	fmt.Fprintf(&b, "## Signatures\n\n")
	if len(data.Signatures) == 0 {
		fmt.Fprintf(&b, "_No signatures evaluated for this run._\n\n")
	} else {
		fmt.Fprintf(&b, "| Signature | Status | Evidence | Detail |\n|---|---|---|---|\n")
		for _, sig := range data.Signatures {
			evidence := "—"
			if !sig.Evidence.IsZero() {
				evidence = fmt.Sprintf("%s → %s",
					sig.Evidence.Start.UTC().Format(timeLayout),
					sig.Evidence.End.UTC().Format(timeLayout))
			}
			detail := sig.Detail
			if detail == "" {
				detail = "—"
			}
			fmt.Fprintf(&b, "| %s | %s %s | %s | %s |\n", sig.Name, sig.Status, statusBadge(sig.Status), evidence, detail)
		}
		fmt.Fprintf(&b, "\n")
	}

	_, err := io.WriteString(w, b.String())
	return err
}
