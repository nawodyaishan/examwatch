package probe

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type PowerEvent struct {
	Timestamp    time.Time `json:"timestamp"`
	ACConnected  bool      `json:"ac_connected"`
	BatteryPct   int       `json:"battery_percent"`
	Charging     bool      `json:"charging"`
	TimeRemaining string   `json:"time_remaining,omitempty"`
}

type BattReader interface {
	Read(ctx context.Context) (string, error)
}

type defaultBattReader struct{}

func (d *defaultBattReader) Read(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "pmset", "-g", "batt")
	out, err := cmd.Output()
	return string(out), err
}

type PowerProbe struct {
	Reader   BattReader
	Interval time.Duration
}

func NewPowerProbe(interval time.Duration) *PowerProbe {
	return &PowerProbe{
		Reader:   &defaultBattReader{},
		Interval: interval,
	}
}

func (p *PowerProbe) Start(ctx context.Context, out chan<- interface{}) {
	ticker := time.NewTicker(p.Interval)
	defer ticker.Stop()

	var lastACState *bool

	for {
		select {
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			output, err := p.Reader.Read(ctx)
			if err != nil {
				continue
			}

			event, err := ParsePmset(output)
			if err != nil {
				continue
			}
			event.Timestamp = t

			// Emit event ONLY on transition
			if lastACState == nil || *lastACState != event.ACConnected {
				// State changed
				select {
				case out <- event:
				case <-ctx.Done():
					return
				}
				v := event.ACConnected
				lastACState = &v
			}
		}
	}
}

var pctRegex = regexp.MustCompile(`(\d+)%`)
var remainRegex = regexp.MustCompile(`(\d+:\d+) remaining`)

func ParsePmset(output string) (PowerEvent, error) {
	var event PowerEvent
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return event, fmt.Errorf("empty pmset output")
	}

	// Parse AC power state
	switch {
	case strings.Contains(lines[0], "'AC Power'"):
		event.ACConnected = true
	case strings.Contains(lines[0], "'Battery Power'"):
		event.ACConnected = false
	default:
		return event, fmt.Errorf("unknown power source in pmset: %s", lines[0])
	}

	if len(lines) > 1 {
		detail := lines[1]
		if m := pctRegex.FindStringSubmatch(detail); len(m) > 1 {
			pct, _ := strconv.Atoi(m[1])
			event.BatteryPct = pct
		}
		if strings.Contains(detail, "charging") || strings.Contains(detail, "charged") {
			event.Charging = !strings.Contains(detail, "discharging")
		}
		if m := remainRegex.FindStringSubmatch(detail); len(m) > 1 {
			event.TimeRemaining = m[1]
		}
	}

	return event, nil
}
