package ui

import (
	"bytes"
	"testing"
	"time"
)

func TestScroller(t *testing.T) {
	var buf bytes.Buffer
	s := NewScroller(&buf)

	ts := time.Date(2026, 7, 28, 14, 32, 1, 0, time.UTC)
	s.LogEvent(ts, "AC_DROP detected — mac switched to battery")

	out := buf.String()
	want := "[14:32:01] AC_DROP detected — mac switched to battery\n"
	if out != want {
		t.Errorf("Scroller.LogEvent() = %q, want %q", out, want)
	}
}
