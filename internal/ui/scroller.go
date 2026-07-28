package ui

import (
	"fmt"
	"io"
	"time"
)

type Scroller struct {
	out io.Writer
}

func NewScroller(out io.Writer) *Scroller {
	return &Scroller{out: out}
}

func (s *Scroller) LogEvent(t time.Time, msg string) {
	_, _ = fmt.Fprintf(s.out, "[%s] %s\n", t.Format("15:04:05"), msg)
}
