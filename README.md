# examwatch

`examwatch` is a macOS pre-exam rehearsal instrument. It runs during a simulated grid outage (e.g., a UPS test, scheduled maintenance, or a live unannounced cut), collects network and system telemetry at 1-second resolution, and produces a deterministic **PASS / WARN / FAIL** verdict against named failure signatures derived from documented proctoring-platform behavior (e.g., Pearson VUE, Certiverse, PSI Bridge). 

Home-proctored certification exams enforce zero-tolerance policies on network drops and webcam disruptions. If you live in a grid-unstable region, `examwatch` gives you the concrete evidence you need to know whether your internet + power backup setup will actually survive a full exam window *before* you are financially and professionally committed to an attempt.

---

## ⚠️ Important Scope & Hardware Limitations

**`examwatch` does NOT query your UPS directly.** 

Consumer UPS units often expose no USB/serial telemetry. Power events are inferred indirectly via `pmset` AC-state transitions on the host Mac, correlated with network anomalies in the same time window. 
- **What this means for you:** This tool observes the power state of your Mac battery. It assumes that if your Mac switches to battery power (`AC_DROP`), the grid has dropped. 
- **Supported Platforms:** macOS only (v1). Relies on Darwin-specific `pmset` behavior.
- **Privacy:** Fully local. No telemetry or data leaves your machine.

---

## Installation

You can install `examwatch` directly using `go install` (requires Go 1.22+):

```bash
go install github.com/nawodyaishan/examwatch/cmd/examwatch@latest
```

Alternatively, you can tap it via Homebrew:

```bash
brew install nawodyaishan/tap/examwatch
```

---

## Usage

Run a rehearsal for the expected duration of your exam (e.g., `60m` or `120m`).

```bash
# Start a 60-minute simulated exam test
examwatch run --duration 60m --out ./my-run-results/

# Example with a custom sampling interval
examwatch run --duration 120m --interval 2s --out ./my-run-results/
```

During the run, `examwatch` will display a live, sticky TTY terminal dashboard with sparklines tracking your RTT, Jitter, Packet Loss, and macOS Power states. 

If you abort the run early (via `Ctrl+C`), `examwatch` performs a graceful shutdown and generates a valid partial report based on the data collected up to that point.

---

## Output Artifacts

At the end of a run (or upon graceful shutdown), `examwatch` produces the following files in your `--out` directory:

1. **`log.jsonl`** — One JSON object per sample/event, append-only, and flushed on every write. Crash-safe by construction.
2. **`summary.json`** — Machine-readable verdict mapping to proctor-app failure signatures (`SUSTAINED_LOSS`, `IP_CHURN`, `JITTER_SPIKE`, `AC_DROP`, `DNS_STALL`).
3. **`report.md`** — A human-readable Markdown report detailing the timeline of events.

### Sample `report.md` Excerpt

Below is a generated snippet of a run that experienced a grid outage, causing the Mac to drop to battery power (`AC_DROP`), followed by a complete router failure resulting in `SUSTAINED_LOSS`:

```markdown
# examwatch run report

## Verdict: FAIL [FAIL]

| Field | Value |
|---|---|
| Start time | 2026-07-28 14:00:00 UTC |
| Duration | 1h0m0s |
| Interval | 1s |

## Timeline

- `2026-07-28 14:05:00 UTC` AC_DROP (WARN) started: AC power disconnected during the run
- `2026-07-28 14:10:00 UTC` SUSTAINED_LOSS (FAIL) started: 100% packet loss sustained for consecutive samples
- `2026-07-28 14:11:00 UTC` SUSTAINED_LOSS (FAIL) ended
- `2026-07-28 14:15:00 UTC` AC_DROP (WARN) ended

## Signatures

| Signature | Status | Evidence | Detail |
|---|---|---|---|
| SUSTAINED_LOSS | FAIL | 14:10:00 UTC → 14:11:00 UTC | 100% packet loss sustained for consecutive samples |
| AC_DROP | WARN | 14:05:00 UTC → 14:15:00 UTC | AC power disconnected during the run |
| IP_CHURN | PASS | — | — |
```

---

## License

MIT License
