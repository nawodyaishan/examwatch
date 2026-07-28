// Package report generates the two end-of-run artifacts described in the
// tech spec (section 10, "Output Artifacts"):
//
//   - summary.json — machine-readable verdict, per-signature status with
//     evidence timestamp ranges, plus run metadata.
//   - report.md    — human-readable report: overall verdict banner, ASCII
//     sparkline charts, a chronological timeline of discrete events, and
//     the same per-signature breakdown as the JSON.
//
// This package intentionally defines its own input types rather than
// importing internal/probe or internal/rules directly. Those packages are
// being built in parallel under the same tech-spec sections; wiring real
// producer types into report.RunData happens in a later integration task.
// The shape below is deliberately small and stable so that wiring is a
// matter of populating these fields, not redesigning them.
package report

import "time"

// Status is the verdict of a single failure signature, or of the run as a
// whole. Severity ordering (worst to best) is FAIL > WARN > PASS, per
// section 7 of the tech spec ("Final verdict = worst individual signature").
type Status string

// Recognized status values, in ascending order of severity precedence when
// compared with Status.Worse.
const (
	StatusPass Status = "PASS"
	StatusWarn Status = "WARN"
	StatusFail Status = "FAIL"
)

// rank returns the severity rank of a status: higher is worse. Unknown
// values rank alongside PASS so malformed input degrades safely rather than
// silently winning the overall verdict.
func (s Status) rank() int {
	switch s {
	case StatusFail:
		return 2
	case StatusWarn:
		return 1
	default:
		return 0
	}
}

// Worse returns the more severe of s and other.
func (s Status) Worse(other Status) Status {
	if other.rank() > s.rank() {
		return other
	}
	return s
}

// EvidenceWindow is a timestamp range supporting a signature's verdict. For
// signatures with no evidence (e.g. a clean PASS), Start and End are the
// zero time.
type EvidenceWindow struct {
	Start time.Time
	End   time.Time
}

// IsZero reports whether the window carries no evidence.
func (w EvidenceWindow) IsZero() bool {
	return w.Start.IsZero() && w.End.IsZero()
}

// SignatureResult is the verdict for a single rules-engine signature (see
// tech spec section 7: SUSTAINED_LOSS, IP_CHURN, JITTER_SPIKE, AC_DROP,
// DNS_STALL), plus the evidence window and a short human-readable detail
// string (e.g. "6.2s at 100% loss").
type SignatureResult struct {
	Name     string
	Status   Status
	Evidence EvidenceWindow
	Detail   string
}

// RunMeta is run-level metadata common to both artifacts.
type RunMeta struct {
	StartTime time.Time
	Duration  time.Duration
	Interval  time.Duration
	Hostname  string
}

// TimelineEvent is a single discrete, timestamped occurrence surfaced in
// report.md's chronological timeline (e.g. an AC_DROP transition or a loss
// window boundary). Events are rendered in the order given by RunData —
// callers are responsible for chronological sorting.
type TimelineEvent struct {
	Time    time.Time
	Message string
}

// SeriesPoint is a single sample in a named metric series (RTT, loss %,
// jitter, ...) used to render sparkline charts across the full run.
type SeriesPoint struct {
	Time  time.Time
	Value float64
}

// Series is a named, unit-labeled sequence of samples spanning the full
// run, rendered as one ASCII sparkline line in report.md.
type Series struct {
	Name   string // e.g. "RTT"
	Unit   string // e.g. "ms"
	Points []SeriesPoint
}

// RunData is the complete input to both WriteSummary and WriteMarkdown: a
// completed rehearsal's data, independent of how it was produced (live
// probe run or replay of a checked-in log.jsonl).
type RunData struct {
	Meta       RunMeta
	Signatures []SignatureResult
	Timeline   []TimelineEvent
	Series     []Series
}

// OverallStatus computes the run's overall verdict as the worst individual
// signature status, per tech spec section 7. A run with no signatures at
// all is reported PASS.
func (d RunData) OverallStatus() Status {
	overall := StatusPass
	for _, sig := range d.Signatures {
		overall = overall.Worse(sig.Status)
	}
	return overall
}
