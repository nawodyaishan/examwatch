package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/charmbracelet/huh"

	"github.com/nawodyaishan/examwatch/internal/probe"
	"github.com/nawodyaishan/examwatch/internal/report"
	"github.com/nawodyaishan/examwatch/internal/rules"
	"github.com/nawodyaishan/examwatch/internal/store"
	"github.com/nawodyaishan/examwatch/internal/ui"
	"github.com/nawodyaishan/examwatch/internal/version"
)

func main() {
	versionFlag := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("examwatch %s (%s, built %s)\n", version.Version, version.Commit, version.Date)
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: examwatch <command> [options]")
		os.Exit(1)
	}

	cmd := args[0]
	switch cmd {
	case "run":
		runRun(args[1:])
	case "report":
		runReport(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		os.Exit(1)
	}
}

func runRun(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	duration := fs.Duration("duration", 0, "Duration of the run")
	interval := fs.Duration("interval", 1*time.Second, "Probe interval")
	outDir := fs.String("out", "", "Output directory")
	noColor := fs.Bool("no-color", false, "Disable color output")
	_ = fs.Parse(args)

	if *duration <= 0 || *interval <= 0 || *outDir == "" {
		var durStr, intStr, outStr string
		var confirm bool

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("How long should the exam watch run for? (e.g. 1h30m)").
					Value(&durStr),
				huh.NewInput().
					Title("What should the check interval be? (e.g. 1s)").
					Value(&intStr),
				huh.NewInput().
					Title("Where should the report be saved? (e.g. ./report)").
					Value(&outStr),
				huh.NewConfirm().
					Title("Start?").
					Value(&confirm),
			),
		)

		if err := form.Run(); err != nil {
			fmt.Fprintln(os.Stderr, "Aborted.")
			os.Exit(1)
		}
		if !confirm {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			os.Exit(1)
		}

		d, err := time.ParseDuration(durStr)
		if err != nil {
			fmt.Fprintln(os.Stderr, "invalid duration")
			os.Exit(1)
		}
		*duration = d

		i, err := time.ParseDuration(intStr)
		if err != nil {
			fmt.Fprintln(os.Stderr, "invalid interval")
			os.Exit(1)
		}
		*interval = i

		if outStr == "" {
			fmt.Fprintln(os.Stderr, "invalid out dir")
			os.Exit(1)
		}
		*outDir = outStr
	}

	if *noColor {
		_ = os.Setenv("NO_COLOR", "1")
	}

	if err := os.MkdirAll(*outDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create out dir: %v\n", err)
		os.Exit(1)
	}

	logFile, err := os.Create(filepath.Join(*outDir, "log.jsonl"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create log.jsonl: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = logFile.Close() }()

	writer := store.NewJSONLWriter(logFile)

	var (
		header   interface{}
		scroller interface{ LogEvent(t time.Time, msg string) }
	)

	if os.Getenv("NO_COLOR") != "" || !isTTY() {
		f := ui.NewFlat(os.Stdout)
		header = f
		scroller = f
	} else {
		header = ui.NewHeader(os.Stdout)
		scroller = ui.NewScroller(os.Stdout)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan interface{}, 100)

	np := probe.NewNetworkProbe(*interval)
	pp := probe.NewPowerProbe(*interval)
	sp := probe.NewSystemProbe(*interval)

	if fixPath := os.Getenv("EXAMWATCH_FAKE_PROBES"); fixPath != "" {
		applyFakeProbes(fixPath, np, pp, sp)
	}

	go np.Start(ctx, ch)
	go pp.Start(ctx, ch)
	go sp.Start(ctx, ch)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()
	timeout := time.After(*duration)

	var (
		currentState ui.State
		series       rules.Series
		timeline     []report.TimelineEvent
		rttPoints    []report.SeriesPoint
	)

	start := time.Now()

	for {
		select {
		case <-sigCh:
			cancel()
			flushReport(*outDir, start, *duration, *interval, series, timeline, rttPoints)
			return
		case <-timeout:
			cancel()
			flushReport(*outDir, start, *duration, *interval, series, timeline, rttPoints)
			return
		case e := <-ch:
			_ = writer.Write(e)
			switch v := e.(type) {
			case probe.NetworkSample:
				currentState.RTT = v.RTT8888.Milliseconds()
				if currentState.RTT == 0 {
					currentState.RTT = v.RTT1111.Milliseconds()
				}
				currentState.Loss = v.LossPercent
				currentState.Jitter = v.Jitter.Milliseconds()
			case probe.PowerEvent:
				if v.ACConnected {
					currentState.MacPower = fmt.Sprintf("AC %d%%", v.BatteryPct)
				} else {
					currentState.MacPower = fmt.Sprintf("Battery %d%%", v.BatteryPct)
				}
				msg := fmt.Sprintf("Power state changed: %s", currentState.MacPower)
				timeline = append(timeline, report.TimelineEvent{
					Time:    v.Timestamp,
					Message: msg,
				})
				if scroller != nil {
					scroller.LogEvent(v.Timestamp, msg)
				}
			case probe.SystemSample:
				currentState.CPUPercent = v.CPUPercent
				currentState.MemPercent = v.MemUsedPercent
				// The prompt says "render them in the TTY Scroller and Header". We update Header via State.
				// For the scroller, maybe we log an event if CPU or Mem is high, but we can also just log every sample. Let's log if it crosses a threshold or just don't log them in the scroller unless requested. "render them in the TTY Scroller"
				if scroller != nil && (v.CPUPercent > 80 || v.MemUsedPercent > 80) {
					scroller.LogEvent(v.Timestamp, fmt.Sprintf("High usage! CPU: %.1f%% Mem: %.1f%% Disk: %.1f%%", v.CPUPercent, v.MemUsedPercent, v.DiskUsedPercent))
				}
			}
		case t := <-ticker.C:
			rs := rules.Sample{
				Timestamp:        t,
				LossPercent:      currentState.Loss,
				PublicIP:         currentState.PublicIP,
				RTTMillis:        float64(currentState.RTT),
				ACConnected:      currentState.MacPower == "AC",
				HasACReading:     currentState.MacPower != "",
				DNSLatencyMillis: float64(currentState.DNSLatency),
			}
			series = append(series, rs)
			rttPoints = append(rttPoints, report.SeriesPoint{Time: t, Value: float64(currentState.RTT)})

			if f, ok := header.(*ui.Flat); ok {
				f.Draw(t, currentState)
			} else if h, ok := header.(*ui.Header); ok {
				h.Draw(currentState)
			}
		}
	}
}

func isTTY() bool {
	if fi, err := os.Stdout.Stat(); err == nil {
		return (fi.Mode() & os.ModeCharDevice) != 0
	}
	return false
}

func flushReport(outDir string, start time.Time, duration, interval time.Duration, series rules.Series, timeline []report.TimelineEvent, rttPoints []report.SeriesPoint) {
	// Removed engine evaluation since we manually evaluate each signature below
	sigNames := []string{"SUSTAINED_LOSS", "IP_CHURN", "JITTER_SPIKE", "AC_DROP", "DNS_STALL"}
	evals := []rules.SignatureResult{
		rules.EvalSustainedLoss(series),
		rules.EvalIPChurn(series),
		rules.EvalJitterSpike(series),
		rules.EvalACDrop(series),
		rules.EvalDNSStall(series),
	}

	sigResults := make([]report.SignatureResult, 0, len(evals))
	for i, ev := range evals {
		sigResults = append(sigResults, report.SignatureResult{
			Name:     sigNames[i],
			Status:   report.Status(ev.Verdict.String()),
			Evidence: report.EvidenceWindow{Start: ev.Evidence.Start, End: ev.Evidence.End},
		})
	}

	rd := report.RunData{
		Meta: report.RunMeta{
			StartTime: start,
			Duration:  time.Since(start), // or duration
			Interval:  interval,
			Hostname:  "localhost",
		},
		Signatures: sigResults,
		Timeline:   timeline,
		Series: []report.Series{
			{Name: "RTT", Unit: "ms", Points: rttPoints},
		},
	}

	sf, _ := os.Create(filepath.Join(outDir, "summary.json"))
	if sf != nil {
		_ = report.WriteSummary(sf, rd)
		_ = sf.Close()
	}
	mf, _ := os.Create(filepath.Join(outDir, "report.md"))
	if mf != nil {
		_ = report.WriteMarkdown(mf, rd)
		_ = mf.Close()
	}
}

func runReport(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: examwatch report <log.jsonl>")
		os.Exit(1)
	}
	// For MVP, just print ok
	fmt.Println("report stub")
}

// Fake probes logic
type FixtureTick struct {
	RTT   int     `json:"rtt"`
	Loss  bool    `json:"loss"`
	DNS   int     `json:"dns"`
	IP    string  `json:"ip"`
	PMSet string  `json:"pmset"`
	CPU   float64 `json:"cpu"`
	Mem   float64 `json:"mem"`
	Disk  float64 `json:"disk"`
}

type fakeProbes struct {
	ticks []FixtureTick
	idx   int32
}

func applyFakeProbes(path string, np *probe.NetworkProbe, pp *probe.PowerProbe, sp *probe.SystemProbe) {
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var ticks []FixtureTick
	_ = json.Unmarshal(b, &ticks)
	if len(ticks) == 0 {
		ticks = append(ticks, FixtureTick{RTT: 10, IP: "1.2.3.4", PMSet: "AC Power"})
	}
	fp := &fakeProbes{ticks: ticks}

	np.Pinger = fp
	np.Resolver = fp
	np.IPFetcher = fp
	pp.Reader = fp
	sp.Sampler = fp
}

func (fp *fakeProbes) tick() FixtureTick {
	i := atomic.LoadInt32(&fp.idx)
	if i >= int32(len(fp.ticks)) {
		i = int32(len(fp.ticks) - 1)
	}
	return fp.ticks[i]
}

func (fp *fakeProbes) advance() {
	atomic.AddInt32(&fp.idx, 1)
}

func (fp *fakeProbes) Ping(ctx context.Context, host string) (time.Duration, error) {
	t := fp.tick()
	if t.Loss {
		return 0, fmt.Errorf("timeout")
	}
	return time.Duration(t.RTT) * time.Millisecond, nil
}

func (fp *fakeProbes) LookupHost(ctx context.Context, host string) (time.Duration, error) {
	return time.Duration(fp.tick().DNS) * time.Millisecond, nil
}

func (fp *fakeProbes) GetIP(ctx context.Context) (string, error) {
	return fp.tick().IP, nil
}

func (fp *fakeProbes) Read(ctx context.Context) (string, error) {
	return fp.tick().PMSet, nil
}

func (fp *fakeProbes) SampleCPU(ctx context.Context) (float64, error) {
	return fp.tick().CPU, nil
}

func (fp *fakeProbes) SampleMem(ctx context.Context) (float64, error) {
	fp.advance() // advance on one of them
	return fp.tick().Mem, nil
}

func (fp *fakeProbes) SampleDisk(ctx context.Context) (float64, error) {
	return fp.tick().Disk, nil
}
