package ui

import (
	"bytes"
	"testing"
	"time"
)

func TestFlat(t *testing.T) {
	var buf bytes.Buffer
	f := NewFlat(&buf)

	ts := time.Date(2026, 7, 28, 14, 32, 1, 0, time.UTC)

	t.Run("LogEvent", func(t *testing.T) {
		buf.Reset()
		f.LogEvent(ts, "system offline")
		out := buf.String()
		want := "[2026-07-28T14:32:01Z] system offline\n"
		if out != want {
			t.Errorf("Flat.LogEvent() = %q, want %q", out, want)
		}
	})

	t.Run("Draw", func(t *testing.T) {
		buf.Reset()
		state := State{
			RTT:        25,
			Loss:       10,
			Jitter:     5,
			PublicIP:   "1.1.1.1",
			MacPower:   "AC Power",
			DNSLatency: 15,
		}
		f.Draw(ts, state)
		out := buf.String()
		want := "[2026-07-28T14:32:01Z] STATE: RTT=25ms Loss=10% Jitter=5ms IP=1.1.1.1 Power=\"AC Power\" DNS=15ms\n"
		if out != want {
			t.Errorf("Flat.Draw() = %q, want %q", out, want)
		}
	})
}
