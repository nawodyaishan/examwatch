package ui

import (
	"fmt"
	"io"
	"time"
)

type Flat struct {
	out io.Writer
}

func NewFlat(out io.Writer) *Flat {
	return &Flat{out: out}
}

func (f *Flat) LogEvent(t time.Time, msg string) {
	_, _ = fmt.Fprintf(f.out, "[%s] %s\n", t.Format(time.RFC3339), msg)
}

func (f *Flat) Draw(t time.Time, state State) {
	_, _ = fmt.Fprintf(f.out, "[%s] STATE: RTT=%dms Loss=%.0f%% Jitter=%dms IP=%s Power=%q DNS=%dms\n",
		t.Format(time.RFC3339), state.RTT, state.Loss, state.Jitter, state.PublicIP, state.MacPower, state.DNSLatency)
}
