# `examwatch` — Technical Specification

**Version:** 0.1.1 (draft)
**Author:** Nawodya
**License:** MIT
**Status:** Pre-implementation spec — ready for Claude Code scaffolding

---

## 1. Problem Statement

Home-proctored certification exams (Certiverse, PSI Bridge, Pearson VUE OnVUE)
enforce zero-tolerance policies on network drops, audio anomalies, and webcam
disruptions. Candidates in grid-unstable regions have no tool to *quantify*
whether their internet + power backup setup will survive a full exam window
before they are financially and professionally committed to an attempt.

`examwatch` is a pre-exam rehearsal instrument. It runs during a simulated
outage (UPS test, scheduled maintenance, or a live unannounced cut), collects
network and system telemetry at 1-second resolution, and produces a
deterministic PASS/WARN/FAIL verdict against named failure signatures derived
from documented proctoring-platform behavior — not a vague "looks fine" data
dump.

**Explicit non-goal:** This tool does not query the UPS directly. Consumer
UPS units (e.g., Prolink PRO1201SFCU) expose no USB/serial telemetry. Power
events are inferred indirectly via `pmset` AC-state transitions on the host
Mac, correlated with network anomalies in the same window. The README must
state this limitation plainly — do not claim direct grid/UPS sensing.

---

## 2. Scope

### In scope
- Cross-run network reliability sampling (RTT, loss, jitter, DNS, public IP)
- macOS power-state sampling via `pmset`
- CPU/memory load sampling (context for interpreting network anomalies)
- Rule-based verdict engine mapped to known proctor failure triggers
- Terminal UI: sticky-header live view + scrolling event log
- Non-TTY fallback (flat logging) for piped/redirected/CI output
- Post-run Markdown + JSON report generation
- Graceful shutdown with valid partial report on SIGINT/SIGTERM

### Out of scope (v0.1.0)
- Direct UPS telemetry (hardware limitation, not a software gap)
- Windows/Linux platform support (macOS-only for v1; `pmset` is Darwin-specific)
- Live upload to any external service — fully local, no telemetry leaves the machine
- Audio/microphone monitoring (irrelevant now that the UPS is in another room)
- Automated remediation (the tool observes and reports; it does not act)

---

## 3. Language & Runtime Decision

**Go 1.22+, single static binary, no CGO.**

Rationale:
- A monitoring daemon needs predictable, low-jitter sampling while the same
  machine is under proctoring-software CPU/GPU load. Go's low, consistent
  runtime overhead is a better fit than Python's interpreter + GC pause profile
  for this specific measurement task.
- Zero-dependency single binary matters on exam morning — no venv, no pip
  resolution, no "works on my machine."
- Matches your existing production stack (SeatGuard, Go-first portfolio),
  keeping the GitHub presentation consistent.

---

## 4. Dependencies

| Purpose | Module | Notes |
|---|---|---|
| ICMP ping | `github.com/prometheus-community/pro-bing` | Unprivileged UDP-mode ping works on macOS without `sudo` — Darwin allows unprivileged `SOCK_DGRAM` ICMP natively, unlike Linux which needs a `sysctl` change. Fallback to TCP dial-timing if ICMP is blocked by network policy. |
| System metrics | `github.com/shirou/gopsutil/v4` (`cpu`, `mem`, `net`) | Use the `/v4` module path explicitly — this is the current major version. |
| Terminal capability detection | `golang.org/x/term` | `term.IsTerminal(fd)` for TTY detection, `term.GetSize(fd)` for width-aware truncation. Deliberately minimal — not a TUI framework. |
| Everything else | Go stdlib | `os/signal`, `context`, `encoding/json`, `time`, `net`, `bufio` |

**Deliberately excluded:** Bubbletea, Lipgloss, tview, termui. A fixed 6-line
sticky header redrawn via raw ANSI escapes needs none of the MVU
(model-view-update) machinery those libraries provide for interactive
keyboard-driven apps. This tool has zero user interaction during a run —
that abstraction would add ~15 transitive dependencies for no benefit, and
work against the goal of an easily auditable `go.sum` for a public repo.

---

## 5. Architecture

```
cmd/examwatch/
    main.go                 — CLI entrypoint, flag parsing, signal handling

internal/probe/
    network.go              — RTT/loss/jitter/DNS/public-IP sampling
    power.go                — pmset parsing, AC-state transition detection
    system.go                — CPU/memory sampling via gopsutil

internal/rules/
    engine.go                — failure-signature evaluation against series data
    signatures.go             — named rule definitions + thresholds as constants

internal/ui/
    header.go                 — sticky-header ANSI renderer + ring-buffer sparklines
    scroller.go                — scrolling event-log writer
    flat.go                    — non-TTY fallback renderer

internal/report/
    markdown.go                — report.md generation
    json.go                     — summary.json generation

internal/store/
    jsonl.go                    — append-only crash-safe writer/reader

testdata/
    sample-run.jsonl             — anonymized example run for tests + README

.github/workflows/ci.yml
Makefile
README.md
LICENSE (MIT)
go.mod / go.sum
```

### Data flow

```
 ticker (1s) ──▶ probe.Network ─┐
 ticker (2s) ──▶ probe.Power   ─┼──▶ store.JSONLWriter (append, flush per line)
 ticker (1s) ──▶ probe.System  ─┘         │
                                          ▼
                              internal/ui (live render, if TTY)
                                          │
                          (post-run) rules.Engine.Evaluate(series)
                                          │
                          report.Markdown + report.JSON
```

All probes run as independent goroutines feeding a single buffered channel
into the JSONL writer, so a stall in one probe (e.g., DNS timeout) never
blocks another (e.g., `pmset` sampling). Writer flushes after every line —
if the Mac itself loses power mid-run, the JSONL file up to that point
remains valid and readable.

---

## 6. Metrics Specification

### 6.1 Network layer (sampled every `--interval`, default 1s)

| Metric | Method | Notes |
|---|---|---|
| RTT to `1.1.1.1` | pro-bing unprivileged ICMP | Primary target |
| RTT to `8.8.8.8` | pro-bing unprivileged ICMP | Secondary target, cross-check |
| TCP handshake time to a third host | `net.DialTimeout("tcp", host:443, 2s)` | Fallback signal if ICMP is filtered |
| Packet loss % | Rolling 10-sample window | `lost / sent * 100` |
| Jitter | stddev of RTT over rolling 10-sample window | Flags feed-freeze risk |
| Public IP | HTTP GET to a plain-text IP echo endpoint, every 15s | Detects Starlink IP churn (documented proctor auto-flag risk) |
| DNS resolution latency | `net.Resolver.LookupHost` against a fixed hostname, every 5s | Proctor apps can hang silently on DNS stalls before ICMP notices anything |

### 6.2 Power layer (sampled every 2s)

Parse `pmset -g batt` stdout. Example output to parse:
```
Now drawing from 'AC Power'
 -InternalBattery-0 (id=...)    98%; charged; 0:00 remaining present: true
```

Extract: battery percentage, AC-connected boolean (`'AC Power'` vs
`'Battery Power'`), charging state, time-remaining string. Emit a discrete
event on every AC-state transition (`AC→Battery` or `Battery→AC`) with a
precise timestamp — this is the anchor point for correlating network
anomalies with an actual power event.

### 6.3 System layer (sampled every 1s)

- CPU utilization: `cpu.Percent(0, false)` (gopsutil)
- Memory: `mem.VirtualMemory()` → `UsedPercent` (gopsutil)

This exists purely as interpretive context — e.g., distinguishing "network
dropped because of a real outage" from "network looked slow because the
Mac was thermal-throttling under proctoring-app load."

---

## 7. Rules Engine — Failure Signatures

Each signature is a pure function over the collected time series, returning
`PASS`, `WARN`, or `FAIL` plus the evidence window (start/end timestamps).

| Signature | Rule | Maps to real-world risk |
|---|---|---|
| `SUSTAINED_LOSS` | ≥5 consecutive seconds at 100% packet loss | Certiverse "Abandoned" session trigger |
| `IP_CHURN` | Public IP changes 1+ times during the run | Session-hijack auto-flag risk on IP-change detection |
| `JITTER_SPIKE` | RTT stddev > 150ms sustained for >10s | Video feed freeze/stutter risk during screen-share |
| `AC_DROP` | AC-connected flips to `false` at least once | Confirms an actual power event occurred — correlate its timestamp against the network signatures above |
| `DNS_STALL` | Resolution latency > 2000ms | Silent proctor-app hang risk before any ICMP-visible symptom |

Final verdict = worst individual signature (`FAIL` > `WARN` > `PASS`).
Thresholds are named constants in `internal/rules/signatures.go`, not
hardcoded inline — this makes them easy to recalibrate as you gather more
runs, and easy for a GitHub contributor to review/adjust without reading
engine internals.

---

## 8. Terminal UI Specification

### 8.1 TTY mode — sticky header + scrolling log

Fixed 6-line header region at the top of the terminal, redrawn in place
each tick using raw ANSI cursor control (`\033[H` + per-line `\033[2K`).
Below it, normal scrolling output for discrete events.

```
┌─ examwatch ── running 00:14:32 / 01:00:00 ──────────────┐
│ RTT (1.1.1.1)   23ms   ▁▂▁▁▃▁▂▁▁▁                        │
│ Loss (10s)      0%     [ok]                              │
│ Jitter          4ms    [ok]                               │
│ Public IP       105.101.x.x   (unchanged)                 │
│ Mac Power       AC connected  98%                          │
│ DNS latency     31ms   [ok]                                │
└─────────────────────────────────────────────────────────┘
[14:32:01] AC_DROP detected — mac switched to battery
[14:32:04] packet loss window: 40% (4/10 pings)
[14:32:11] AC_DROP resolved — mac back on AC
```

- RTT sparkline: 10-sample ring buffer rendered with block characters
  (`▁▂▃▄▅▆▇█`), scaled to the buffer's own min/max. ~20 lines of code, no
  charting dependency.
- Status brackets (`[ok]` / `[warn]` / `[fail]`) colored via raw ANSI SGR
  codes, only when the terminal supports color: check `term.IsTerminal` AND
  absence of a `NO_COLOR` env var.
- Header height fixed at compile time (6 rows) — avoids dynamic layout
  negotiation complexity for a tool that runs once or twice per exam cycle.

### 8.2 Non-TTY fallback

Detected via `!term.IsTerminal(int(os.Stdout.Fd()))`. In this mode:
- No cursor control, no color codes, no sparkline.
- Flat, sequential, timestamped log lines only.
- This matters for: CI pipeline test runs, `examwatch run > out.log`
  redirection, or any consumer piping output into another tool.

---

## 9. CLI Interface

```
examwatch run --duration 60m --out ./run-2026-07-28/ [--interval 1s]
examwatch report ./run-2026-07-28/
```

| Flag | Default | Description |
|---|---|---|
| `--duration` | `60m` | Total run length; matches your target exam length |
| `--out` | `./examwatch-run-<timestamp>/` | Output directory for `log.jsonl`, `report.md`, `summary.json` |
| `--interval` | `1s` | Network/system sampling interval |
| `--no-color` | `false` | Force-disable ANSI color regardless of terminal support |

Graceful shutdown: `SIGINT`/`SIGTERM` triggers an immediate flush of
buffered writes and generation of a valid partial report from whatever data
exists — a rehearsal interrupted early should still produce something useful,
not a corrupt or empty output directory.

---

## 10. Output Artifacts

- **`log.jsonl`** — one JSON object per sample/event, append-only, newline-
  delimited, flushed on every write. Crash-safe by construction.
- **`summary.json`** — machine-readable verdict: per-signature status,
  evidence timestamp ranges, run metadata (start time, duration, interval,
  hostname).
- **`report.md`** — human-readable report: overall verdict banner, ASCII
  sparkline charts for RTT/loss/jitter across the full run (not just the
  live ring buffer), a timeline of discrete events, and the same
  per-signature breakdown as the JSON.

---

## 11. Testing Strategy

Testing tooling is deliberately stdlib-only (`testing`, `testing/iotest`,
`httptest`, `os/exec`) — no `testify` or other assertion library. This keeps
`go.sum` minimal and matches the auditability goal from Section 4; a table
of `if got != want { t.Fatalf(...) }` is not a burden at this project's size.

### 11.1 Unit testing plan

| Package | What is tested | Technique | Coverage target |
|---|---|---|---|
| `internal/rules` | Each of the 5 failure signatures (`SUSTAINED_LOSS`, `IP_CHURN`, `JITTER_SPIKE`, `AC_DROP`, `DNS_STALL`) plus final-verdict aggregation | Table-driven tests with synthetic time series that (a) sit one sample below threshold → `PASS`/no-trigger, (b) sit exactly at threshold, (c) sit one sample above → trigger. Verdict-aggregation tests confirm `FAIL` > `WARN` > `PASS` precedence across combinations. | ≥90% |
| `internal/probe` | Sampling logic in isolation from real network/OS state | Each prober (`network.go`, `power.go`, `system.go`) is defined behind a small interface (`Pinger`, `Resolver`, `BattReader`, `SysSampler`). Unit tests inject fakes that return canned RTT sequences, canned `pmset -g batt` stdout fixtures (including malformed/truncated output), and canned `cpu`/`mem` values. `power.go` parsing is tested against a table of real and edge-case `pmset` stdout strings (both `'AC Power'` and `'Battery Power'`, missing fields, extra whitespace). | ≥85% |
| `internal/ui` | Header rendering, sparkline scaling, flat-mode formatting | Render into a `bytes.Buffer` instead of a real fd; assert on the literal ANSI byte sequence for a fixed input state. Sparkline tests feed known min/max ring-buffer contents and assert the exact block-character string. Color tests toggle `NO_COLOR` and a fake non-TTY writer and assert SGR codes are absent. | ≥80% |
| `internal/report` | `report.md` and `summary.json` generation | Golden-file tests: feed a fixed, small in-memory series + verdict set, generate output, `diff` against a checked-in golden file under `internal/report/testdata/`. A `-update` flag (guarded, not wired into CI) regenerates goldens when the format intentionally changes. | ≥85% |
| `internal/store` | JSONL append/flush/read, crash-safety | Write N lines, kill the writer mid-write (truncate the file), confirm the reader skips the partial trailing line rather than erroring. Confirm every `Write` call results in an immediate `Sync`/flush (verified via a fake `io.Writer` that errors if buffered beyond one line). | ≥85% |
| `cmd/examwatch` | Flag parsing and validation only (not process lifecycle — see E2E) | Direct unit tests against the flag-parsing function, covering defaults, invalid `--duration`/`--interval` values, and out-directory collision handling. | ≥80% |

All packages run under `go test -race ./...` — the multi-goroutine probe
fan-in (Section 5) is exactly the kind of code where a data race is easy to
introduce and easy to miss without `-race`.

### 11.2 End-to-end testing plan

E2E tests build the real `examwatch` binary and exercise it as a subprocess
via `os/exec`, under `cmd/examwatch/e2e_test.go` (guarded by an `e2e` build
tag so `go test ./...` stays fast by default; CI runs both
`go test ./...` and `go test -tags=e2e ./cmd/examwatch/...`).

Because CI has no real network to cut and no battery, every E2E scenario
runs against **fake probes**, selected at process-start via an unexported
env var (`EXAMWATCH_FAKE_PROBES=<fixture-path>`) that is read only in test
builds — never documented as a public flag, so it can't be mistaken for a
supported feature. Each fixture is a small JSON script describing a
deterministic sequence of RTT/loss/DNS/`pmset`/CPU samples for the fake
probes to replay tick-by-tick, letting tests trigger specific signatures on
demand.

| Scenario | Steps | Assertion |
|---|---|---|
| Happy path, all-PASS run | Run `examwatch run --duration 3s --interval 100ms --out <tmp>` against a fixture with no anomalies | Process exits 0; `<tmp>/log.jsonl`, `summary.json`, `report.md` all exist and are non-empty; `summary.json` verdict is `PASS` for every signature |
| Signature trigger: sustained loss | Same, against a fixture with a 100%-loss window ≥5s | `summary.json` shows `SUSTAINED_LOSS: FAIL` with an evidence window matching the fixture's scripted timestamps |
| Signature trigger: AC drop correlated with loss | Fixture scripts an `AC→Battery` transition overlapping a loss window | Both `AC_DROP` and `SUSTAINED_LOSS` present in `summary.json`; `report.md` timeline lists both events in chronological order |
| Graceful shutdown (SIGINT) | Start a 60s run against the happy-path fixture, send `SIGINT` to the subprocess after ~1s, wait for exit | Process exits promptly (bounded wait, e.g. <2s past signal); output directory contains a **valid, parseable** partial `log.jsonl`, `summary.json`, and `report.md` — no empty or truncated-JSON files |
| Graceful shutdown (SIGTERM) | Same as above with `SIGTERM` | Same assertions |
| Non-TTY fallback | Run with stdout piped to a file (never a pty) | Output contains no ANSI escape sequences (`\033[`); lines are flat, timestamped, newline-terminated |
| TTY mode smoke test | Run under a pty (via `github.com/creack/pty`, a **test-only** dependency gated to the `e2e` build tag so it never enters the production `go.sum` closure) for ~1s | Output contains the expected `\033[H` home-cursor sequence and the fixed 6-line header markers at least once |
| `--no-color` flag | Run under a pty with `--no-color` | Output contains cursor-control sequences but no SGR color codes |
| Bad flags | Run with an invalid `--duration` (e.g. `abc`) | Non-zero exit code; usage/error text on stderr; no output directory created |
| `examwatch report` on checked-in fixture | Run `examwatch report testdata/sample-run.jsonl` (no live run) | Generated `report.md`/`summary.json` match the golden files checked in alongside `testdata/sample-run.jsonl` |

E2E tests build the binary once per test run (`TestMain` invokes `go build`
into a temp dir) rather than per-subtest, and share that binary path across
scenarios to keep the suite fast despite spawning multiple subprocesses.

### 11.3 CI enforcement

CI (`golangci-lint`, `go vet`, `go test -race ./...`, `go test -tags=e2e
./cmd/examwatch/...`) runs on Linux runners for speed; macOS-specific
`pmset` parsing is tested via the mocked interface, not a live GitHub
Actions macOS runner (cost/speed tradeoff, and CI doesn't have a battery to
test against anyway). A `go test -race -coverprofile=coverage.out ./...`
step gates merges on the per-package targets in Section 11.1 via a small
`scripts/check-coverage.sh` that parses `coverage.out` — no external
coverage-reporting dependency required.

---

## 12. Repository Layout (GitHub-ready)

```
examwatch/
    cmd/examwatch/main.go
    internal/probe/{network.go,power.go,system.go}
    internal/rules/{engine.go,signatures.go}
    internal/ui/{header.go,scroller.go,flat.go}
    internal/report/{markdown.go,json.go}
    internal/store/jsonl.go
    testdata/sample-run.jsonl
    README.md
    LICENSE
    Makefile
    .github/workflows/ci.yml
    go.mod
    go.sum
```

README must include: the stated hardware limitation (no direct UPS
telemetry), a real sample `report.md` excerpt, install instructions
(`go install` one-liner), and explicit scope boundaries — this is what
separates a credible portfolio tool from an overclaiming one.

---

## 13. Open Decisions (confirm before implementation)

1. **Sparkline count in header**: RTT only, or RTT + loss + jitter stacked?
   Three is more informative on review but cramped on an 80-column terminal
   on a MacBook Air screen.
2. **Public IP echo endpoint**: pick a stable, low-rate-limit plain-text
   endpoint and hardcode it as a fallback-capable list (2-3 candidates) so
   a single endpoint's downtime doesn't break the IP-churn signature.
3. **Retention**: should `examwatch` auto-prune old run directories, or
   leave that entirely to the user? Leaning toward: leave it to the user,
   document it, keep the tool doing one thing.
