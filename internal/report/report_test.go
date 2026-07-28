package report

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var update = flag.Bool("update", false, "update golden files")

func sampleRunData() RunData {
	start := time.Date(2026, 7, 28, 14, 0, 0, 0, time.UTC)
	return RunData{
		Meta: RunMeta{
			StartTime: start,
			Duration:  time.Hour,
			Interval:  time.Second,
			Hostname:  "test-macbook",
		},
		Signatures: []SignatureResult{
			{
				Name:   "SUSTAINED_LOSS",
				Status: StatusFail,
				Evidence: EvidenceWindow{
					Start: start.Add(10 * time.Minute),
					End:   start.Add(11 * time.Minute),
				},
				Detail: "100% packet loss sustained for consecutive samples",
			},
			{
				Name:   "AC_DROP",
				Status: StatusWarn,
				Evidence: EvidenceWindow{
					Start: start.Add(5 * time.Minute),
					End:   start.Add(15 * time.Minute),
				},
				Detail: "AC power disconnected during the run",
			},
			{
				Name:   "IP_CHURN",
				Status: StatusPass,
			},
		},
		Timeline: []TimelineEvent{
			{Time: start.Add(5 * time.Minute), Message: "AC_DROP (WARN) started: AC power disconnected during the run"},
			{Time: start.Add(10 * time.Minute), Message: "SUSTAINED_LOSS (FAIL) started: 100% packet loss sustained for consecutive samples"},
			{Time: start.Add(11 * time.Minute), Message: "SUSTAINED_LOSS (FAIL) ended"},
			{Time: start.Add(15 * time.Minute), Message: "AC_DROP (WARN) ended"},
		},
		Series: []Series{
			{
				Name: "RTT",
				Unit: "ms",
				Points: []SeriesPoint{
					{Time: start, Value: 20},
					{Time: start.Add(time.Minute), Value: 25},
					{Time: start.Add(2 * time.Minute), Value: 22},
					{Time: start.Add(3 * time.Minute), Value: 150},
					{Time: start.Add(4 * time.Minute), Value: 20},
				},
			},
			{
				Name: "Loss",
				Unit: "%",
				Points: []SeriesPoint{
					{Time: start, Value: 0},
					{Time: start.Add(10 * time.Minute), Value: 100},
					{Time: start.Add(11 * time.Minute), Value: 0},
				},
			},
		},
	}
}

func TestWriteSummary(t *testing.T) {
	data := sampleRunData()
	var buf bytes.Buffer
	if err := WriteSummary(&buf, data); err != nil {
		t.Fatalf("WriteSummary failed: %v", err)
	}

	golden := filepath.Join("testdata", "summary.golden.json")
	if *update {
		if err := os.MkdirAll("testdata", 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, buf.Bytes(), 0644); err != nil {
			t.Fatal(err)
		}
	}

	expected, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("Failed to read golden file %s: %v. Run with -update to generate.", golden, err)
	}

	if !bytes.Equal(buf.Bytes(), expected) {
		t.Errorf("WriteSummary output does not match %s. Run with -update to regenerate.\nGot:\n%s", golden, buf.String())
	}
}

func TestWriteMarkdown(t *testing.T) {
	data := sampleRunData()
	var buf bytes.Buffer
	if err := WriteMarkdown(&buf, data); err != nil {
		t.Fatalf("WriteMarkdown failed: %v", err)
	}

	golden := filepath.Join("testdata", "report.golden.md")
	if *update {
		if err := os.MkdirAll("testdata", 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, buf.Bytes(), 0644); err != nil {
			t.Fatal(err)
		}
	}

	expected, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("Failed to read golden file %s: %v. Run with -update to generate.", golden, err)
	}

	if !bytes.Equal(buf.Bytes(), expected) {
		t.Errorf("WriteMarkdown output does not match %s. Run with -update to regenerate.\nGot:\n%s", golden, buf.String())
	}
}
