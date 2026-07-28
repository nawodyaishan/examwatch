package report

import (
	"encoding/json"
	"io"
	"time"
)

// SummaryJSON is the exact on-disk shape of summary.json. Field names use
// JSON tags matching the tech spec's description of the artifact (section
// 10): per-signature status with evidence timestamp ranges, plus run
// metadata (start time, duration, interval, hostname).
type SummaryJSON struct {
	Verdict Status                `json:"verdict"`
	Meta    SummaryMetaJSON       `json:"meta"`
	Results []SignatureResultJSON `json:"signatures"`
}

// SummaryMetaJSON is the run-metadata block of summary.json.
type SummaryMetaJSON struct {
	StartTime string `json:"start_time"` // RFC3339
	Duration  string `json:"duration"`   // e.g. "1h0m0s"
	Interval  string `json:"interval"`   // e.g. "1s"
	Hostname  string `json:"hostname"`
}

// SignatureResultJSON is one signature's entry in summary.json.
type SignatureResultJSON struct {
	Name     string      `json:"name"`
	Status   Status      `json:"status"`
	Detail   string      `json:"detail,omitempty"`
	Evidence *WindowJSON `json:"evidence,omitempty"`
}

// WindowJSON is an evidence timestamp range, omitted entirely when the
// signature has no evidence (e.g. a clean PASS).
type WindowJSON struct {
	Start string `json:"start"` // RFC3339
	End   string `json:"end"`   // RFC3339
}

// BuildSummary converts a RunData into the SummaryJSON shape written by
// WriteSummary. Exported separately from WriteSummary so callers that need
// the struct (e.g. for further processing or a different sink) don't have
// to round-trip through JSON bytes.
func BuildSummary(data RunData) SummaryJSON {
	out := SummaryJSON{
		Verdict: data.OverallStatus(),
		Meta: SummaryMetaJSON{
			StartTime: data.Meta.StartTime.UTC().Format(time.RFC3339),
			Duration:  data.Meta.Duration.String(),
			Interval:  data.Meta.Interval.String(),
			Hostname:  data.Meta.Hostname,
		},
		Results: make([]SignatureResultJSON, 0, len(data.Signatures)),
	}
	for _, sig := range data.Signatures {
		entry := SignatureResultJSON{
			Name:   sig.Name,
			Status: sig.Status,
			Detail: sig.Detail,
		}
		if !sig.Evidence.IsZero() {
			entry.Evidence = &WindowJSON{
				Start: sig.Evidence.Start.UTC().Format(time.RFC3339),
				End:   sig.Evidence.End.UTC().Format(time.RFC3339),
			}
		}
		out.Results = append(out.Results, entry)
	}
	return out
}

// WriteSummary generates summary.json for the given run data and writes it
// to w, pretty-printed (two-space indent) with a trailing newline so the
// file is diff-friendly and terminates cleanly.
func WriteSummary(w io.Writer, data RunData) error {
	summary := BuildSummary(data)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(summary)
}
