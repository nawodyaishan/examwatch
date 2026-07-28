# 🛡️ examwatch

[![Go Report Card](https://goreportcard.com/badge/github.com/nawodyaishan/examwatch)](https://goreportcard.com/report/github.com/nawodyaishan/examwatch)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)](https://golang.org)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://github.com/nawodyaishan/examwatch/pulls)

**`examwatch`** is a specialized macOS terminal application designed for one highly critical task: **verifying your internet and power backup readiness before taking a high-stakes, online proctored certification exam.**

Home-proctored certification exams (like Pearson VUE, PSI Bridge, and Certiverse) enforce strict zero-tolerance policies on network drops, camera freezing, and power losses. If you live in a grid-unstable region, taking an exam from home is incredibly stressful. 

`examwatch` lets you run a **simulated exam rehearsal** during a grid outage (like a UPS test or a scheduled cut). It actively monitors your network and system telemetry at a 1-second resolution, analyzing everything against the strict failure signatures of popular proctoring platforms to give you a definitive **PASS**, **WARN**, or **FAIL** verdict.

Don't gamble your exam fees. Test your grid stability *before* you are financially and professionally committed.

---

## ⚡ Why examwatch?

- **Interactive UI Wizard:** Simply run `examwatch run` to be greeted with a gorgeous terminal UI that guides you through the setup—no need to memorize CLI flags!
- **Deterministic Failure Analysis:** We measure your connection against strict rules like `SUSTAINED_LOSS`, `JITTER_SPIKE`, `IP_CHURN`, and `DNS_STALL`.
- **Beautiful TTY Dashboard:** A live, scrolling terminal dashboard that plots ASCII sparklines for Ping, Jitter, Packet Loss, Mac Battery state, and real-time CPU/Mem usage.
- **Deep System Telemetry:** Actively monitors your CPU, Memory, and Disk usage to alert you of sudden spikes that could cause exam software to crash.
- **Detailed Forensic Reports:** Generates a full markdown timeline (`report.md`) of exactly when your network dropped or when your power failed, down to the second.
- **Privacy-First:** Fully offline and local. **No telemetry or data leaves your machine.**

---

## ⚠️ Hardware Limitations & Scope

**`examwatch` does NOT query your UPS directly.** 

Consumer UPS units often expose no accessible USB/serial telemetry on macOS without proprietary drivers. Instead, power events are inferred indirectly via Darwin `pmset` AC-state transitions on your Mac battery.
- **What this means for you:** If your Mac switches from "AC Power" to "Battery Power" (`AC_DROP`), `examwatch` assumes the grid has dropped and your UPS has taken over.
- **Supported Platforms:** macOS only (v1). Relies on Darwin-specific `pmset` behavior.

---

## 📦 Installation

Install `examwatch` using Homebrew for the easiest setup:

```bash
brew install nawodyaishan/tap/examwatch
```

Or, if you have a Go 1.22+ environment, you can install directly via `go install`:

```bash
go install github.com/nawodyaishan/examwatch/cmd/examwatch@latest
```

---

## 🚀 Usage

Run a rehearsal for the expected duration of your exam. We recommend simulating an outage by unplugging your modem's UPS from the wall while this is running.

```bash
# Launch the interactive wizard
examwatch run
```

You can also bypass the wizard by providing the flags directly:

```bash
# Start a 60-minute simulated exam test
examwatch run --duration 60m --out ./my-run-results/

# Example with a custom sampling interval
examwatch run --duration 120m --interval 2s --out ./my-run-results/
```

If you abort the run early (via `Ctrl+C`), `examwatch` gracefully shuts down and generates a valid partial report based on the data collected up to that point.

---

## 📄 Output Artifacts

At the end of a run, `examwatch` produces forensic artifacts in your `--out` directory:

1. **`log.jsonl`** — One JSON object per sample/event, append-only, flushed on every write. Crash-safe by construction.
2. **`summary.json`** — Machine-readable verdict mapping to proctor-app failure signatures.
3. **`report.md`** — A human-readable Markdown report detailing the timeline of events.

<details>
<summary><b>Click to view a sample <code>report.md</code> excerpt</b></summary>

```markdown
# examwatch run report

## Verdict: FAIL [FAIL]

| Field | Value |
|---|---|
| Start time | 2026-07-28 14:00:00 UTC |
| Duration | 1h0m0s |

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
</details>

---

## 🤝 Contributing

We warmly welcome collaborators to `examwatch`! Whether you want to add Windows/Linux support, implement direct UPS querying protocols, or add new network heuristics, your PRs are appreciated.

### Getting Started

1. **Fork the repo** and clone it locally.
2. **Install Go 1.25+**.
3. **Run the interactive wizard:**
   ```bash
   make run
   ```
4. Run the development verification suite before pushing:
   ```bash
   make verify
   ```
5. **Submit your Pull Request!**

*If you're unsure where to start, check the Issues tab for `good first issue` tags.*

---

## 📝 License

Distributed under the MIT License. See `LICENSE` for more information.
