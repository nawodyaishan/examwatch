package ui

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"
)

func TestSparkline(t *testing.T) {
	tests := []struct {
		name string
		data []int64
		want string
	}{
		{"empty", []int64{}, ""},
		{"flat", []int64{10, 10, 10}, "   "}, // min=10, max=10 -> first char ' '
		{"ramp", []int64{0, 1, 2, 3, 4, 5, 6, 7}, " ▂▃▄▅▆▇█"},
		{"min-max", []int64{100, 200, 100}, " █ "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sparkline(tt.data)
			if got != tt.want {
				t.Errorf("sparkline() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHeader_Draw_NoColor(t *testing.T) {
	var buf bytes.Buffer
	h := &Header{out: &buf, useColor: false} // Force no color for simpler string matching

	state := State{
		Elapsed:        14*time.Minute + 32*time.Second,
		Total:          1 * time.Hour,
		RTT:            23,
		RTTBuffer:      []int64{10, 20, 10, 10, 30, 10, 20, 10, 10, 10},
		Loss:           0,
		LossStatus:     StatusOk,
		Jitter:         4,
		JitterStatus:   StatusOk,
		PublicIP:       "105.101.x.x",
		PublicIPStatus: "(unchanged)",
		MacPower:       "AC connected  98%",
		DNSLatency:     31,
		DNSStatus:      StatusOk,
	}

	h.Draw(state)
	out := buf.String()

	if !strings.HasPrefix(out, "\033[H") {
		t.Errorf("expected output to start with home cursor \\033[H, got: %q", out)
	}

	// Verify all 9 lines have clear-line sequence
	lines := strings.Split(strings.TrimPrefix(out, "\033[H"), "\n")
	if len(lines) < 9 {
		t.Fatalf("expected at least 9 lines, got %d", len(lines))
	}
	for i := 0; i < 9; i++ {
		if !strings.HasPrefix(lines[i], "\033[2K") {
			t.Errorf("expected line %d to start with \\033[2K, got: %q", i, lines[i])
		}
	}

	expectedTop := "\033[2K┌─ examwatch ── running 00:14:32 / 01:00:00 ──────────────┐"
	if lines[0] != expectedTop {
		t.Errorf("line 0 = %q, want %q", lines[0], expectedTop)
	}

	expectedBottom := "\033[2K└─────────────────────────────────────────────────────────┘"
	if lines[8] != expectedBottom {
		t.Errorf("line 8 = %q, want %q", lines[8], expectedBottom)
	}

	// Check inner content lengths - should be 59 without escape sequence
	for i := 0; i < 9; i++ {
		// stripped is the string without \033[2K
		stripped := strings.TrimPrefix(lines[i], "\033[2K")
		// count runes
		rc := len([]rune(stripped))
		if rc != 59 {
			t.Errorf("line %d has visible width %d, want 59: %q", i, rc, stripped)
		}
	}
}

func TestHeader_Color(t *testing.T) {
	var buf bytes.Buffer
	h := &Header{out: &buf, useColor: true}

	if got := h.colorStatus(StatusOk); got != "\033[32m[ok]\033[0m" {
		t.Errorf("StatusOk color = %q, want \\033[32m[ok]\\033[0m", got)
	}
	if got := h.colorStatus(StatusWarn); got != "\033[33m[warn]\033[0m" {
		t.Errorf("StatusWarn color = %q, want \\033[33m[warn]\\033[0m", got)
	}
	if got := h.colorStatus(StatusFail); got != "\033[31m[fail]\033[0m" {
		t.Errorf("StatusFail color = %q, want \\033[31m[fail]\\033[0m", got)
	}
}

func TestNewHeader_ColorDetection(t *testing.T) {
	// Normally we can't test terminal detection easily without mocking os.Stdout
	// But we can test that passing a normal bytes.Buffer disables color
	var buf bytes.Buffer
	h := NewHeader(&buf)
	if h.useColor != false {
		t.Errorf("useColor should be false for non-terminal")
	}

	// Setting NO_COLOR disables color even if it was a TTY, but since we can't mock TTY here,
	// just sanity check.
	_ = os.Setenv("NO_COLOR", "1")
	defer func() { _ = os.Unsetenv("NO_COLOR") }()
	h2 := NewHeader(&buf)
	if h2.useColor != false {
		t.Errorf("useColor should be false when NO_COLOR is set")
	}
}
