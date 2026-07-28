package ui

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

type Status string

const (
	StatusOk   Status = "ok"
	StatusWarn Status = "warn"
	StatusFail Status = "fail"
)

type State struct {
	Elapsed        time.Duration
	Total          time.Duration
	RTT            int64
	RTTBuffer      []int64
	Loss           float64
	LossStatus     Status
	Jitter         int64
	JitterStatus   Status
	PublicIP       string
	PublicIPStatus string // e.g. "(unchanged)"
	MacPower       string // e.g. "AC connected  98%"
	DNSLatency     int64
	DNSStatus      Status
}

type Header struct {
	out      io.Writer
	useColor bool
}

func NewHeader(out io.Writer) *Header {
	useColor := false
	if f, ok := out.(interface{ Fd() uintptr }); ok {
		if term.IsTerminal(int(f.Fd())) && os.Getenv("NO_COLOR") == "" {
			useColor = true
		}
	}
	return &Header{
		out:      out,
		useColor: useColor,
	}
}

func (h *Header) colorStatus(s Status) string {
	if !h.useColor {
		return fmt.Sprintf("[%s]", s)
	}
	switch s {
	case StatusOk:
		return "\033[32m[ok]\033[0m"
	case StatusWarn:
		return "\033[33m[warn]\033[0m"
	case StatusFail:
		return "\033[31m[fail]\033[0m"
	default:
		return fmt.Sprintf("[%s]", s)
	}
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func sparkline(data []int64) string {
	if len(data) == 0 {
		return ""
	}
	min := data[0]
	max := data[0]
	for _, v := range data {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	chars := []rune(" ▂▃▄▅▆▇█") // 8 levels
	var buf bytes.Buffer
	for _, v := range data {
		if max == min {
			buf.WriteRune(chars[0])
		} else {
			idx := int(float64(v-min) / float64(max-min) * float64(len(chars)-1))
			buf.WriteRune(chars[idx])
		}
	}
	return buf.String()
}

func (h *Header) formatLine(innerVisible string, innerColored string) string {
	visibleLen := len([]rune(innerVisible))
	padding := 55 - visibleLen
	if padding < 0 {
		padding = 0
	}
	return fmt.Sprintf("\033[2K│ %s%s │\n", innerColored, strings.Repeat(" ", padding))
}

func (h *Header) Draw(state State) {
	var buf bytes.Buffer
	buf.WriteString("\033[H") // Home cursor

	// Line 1: Top border
	top := fmt.Sprintf("┌─ examwatch ── running %s / %s ", formatDuration(state.Elapsed), formatDuration(state.Total))
	topLen := len([]rune(top))
	pad := 58 - topLen
	if pad < 0 {
		pad = 0
	}
	fmt.Fprintf(&buf, "\033[2K%s%s┐\n", top, strings.Repeat("─", pad))

	// Line 2: RTT
	rttStr := fmt.Sprintf("%dms", state.RTT)
	sl := sparkline(state.RTTBuffer)
	rttVis := fmt.Sprintf("%-15s %-6s %s", "RTT (1.1.1.1)", rttStr, sl)
	buf.WriteString(h.formatLine(rttVis, rttVis))

	// Line 3: Loss
	lossStr := fmt.Sprintf("%.0f%%", state.Loss)
	lossVis := fmt.Sprintf("%-15s %-6s [%s]", "Loss (10s)", lossStr, state.LossStatus)
	lossCol := fmt.Sprintf("%-15s %-6s %s", "Loss (10s)", lossStr, h.colorStatus(state.LossStatus))
	buf.WriteString(h.formatLine(lossVis, lossCol))

	// Line 4: Jitter
	jitterStr := fmt.Sprintf("%dms", state.Jitter)
	jitterVis := fmt.Sprintf("%-15s %-6s [%s]", "Jitter", jitterStr, state.JitterStatus)
	jitterCol := fmt.Sprintf("%-15s %-6s %s", "Jitter", jitterStr, h.colorStatus(state.JitterStatus))
	buf.WriteString(h.formatLine(jitterVis, jitterCol))

	// Line 5: Public IP
	ipVis := fmt.Sprintf("%-15s %-13s %s", "Public IP", state.PublicIP, state.PublicIPStatus)
	buf.WriteString(h.formatLine(ipVis, ipVis))

	// Line 6: Mac Power
	pwrVis := fmt.Sprintf("%-15s %s", "Mac Power", state.MacPower)
	buf.WriteString(h.formatLine(pwrVis, pwrVis))

	// Line 7: DNS latency
	dnsStr := fmt.Sprintf("%dms", state.DNSLatency)
	dnsVis := fmt.Sprintf("%-15s %-6s [%s]", "DNS latency", dnsStr, state.DNSStatus)
	dnsCol := fmt.Sprintf("%-15s %-6s %s", "DNS latency", dnsStr, h.colorStatus(state.DNSStatus))
	buf.WriteString(h.formatLine(dnsVis, dnsCol))

	// Line 8: Bottom border
	buf.WriteString("\033[2K└─────────────────────────────────────────────────────────┘\n")

	_, _ = h.out.Write(buf.Bytes())
}
