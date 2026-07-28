# Spec 001 — ExamWatch Pre-Exam Rehearsal Tool (MVP)

**Status:** Draft — awaiting human approval
**Derived from:** `docs/examwatch-tech-spec.md` v0.1.1
**Spec type:** Feature spec

---

## Problem Statement

Home-proctored certification exams (e.g. Certiverse, PSI Bridge, Pearson VUE
OnVUE) enforce zero-tolerance policies on network drops, audio anomalies, and
webcam disruptions during a live exam session. A candidate whose internet
connection or power backup fails mid-exam can be flagged as "abandoned" or
suspected of misconduct, with serious financial and professional
consequences — and there is currently no way to find this out except by
having it happen during a real, paid attempt.

Candidates in grid-unstable regions have no tool to rehearse their setup
under realistic failure conditions (a scheduled power cut, a UPS test, an
unannounced outage) and get a clear, evidence-backed answer to "would my
setup have survived a real exam window?"

## Goals

- Let a candidate run a timed rehearsal that mirrors their planned exam
  window and get a deterministic **PASS / WARN / FAIL** verdict per named
  failure signature, not a vague "looks fine."
- Ground every signature in a documented, real proctoring-platform failure
  trigger, so the verdict maps to actual exam risk rather than arbitrary
  thresholds.
- Produce a durable, shareable record of the rehearsal (human-readable and
  machine-readable) that the candidate can review, re-check, or keep as
  evidence of due diligence.
- Work reliably even when the rehearsal is interrupted early or the run is
  piped/redirected instead of viewed live.

## Non-Goals

- Directly sensing the UPS or grid state. This tool infers power events only
  from the host Mac's own AC/battery state; it does not and cannot query the
  UPS hardware directly. This limitation must be visible to the user, not
  hidden behind confident-sounding output.
- Monitoring audio/microphone conditions.
- Automated remediation of detected problems (the tool observes and reports;
  it never takes corrective action).
- Uploading any telemetry off the candidate's machine — this is a fully
  local rehearsal tool.
- Supporting platforms other than macOS in this version.
- Guaranteeing a match to any specific proctoring vendor's live detection
  logic — signatures are best-effort approximations of documented behavior,
  not a certified compatibility claim.

## Users / Actors

- **Candidate (primary actor):** a person preparing for a home-proctored
  exam who runs a rehearsal on their own Mac ahead of exam day, during a
  self-scheduled outage test (e.g., turning off the UPS, or during a known
  maintenance window).
- **No secondary/admin actor.** This is a single-user, local-only tool; there
  is no multi-user account model, no shared dashboard, and no remote party
  who views results.

## User Journeys

### Journey 1 — Rehearsal during a planned outage test
1. Candidate schedules a UPS/power test for a duration matching their real
   exam window.
2. Candidate starts a rehearsal run for that same duration just before
   cutting power.
3. Candidate watches (or ignores) the live view while the outage occurs.
4. Power returns; the rehearsal continues to completion or the candidate
   stops it once satisfied.
5. Candidate reviews the generated report to see whether any failure
   signature would have put their real exam at risk, and when.

### Journey 2 — Rehearsal cut short
1. Candidate starts a rehearsal.
2. Something forces an early stop (they need the machine, or a real
   emergency occurs).
3. Candidate interrupts the run.
4. Candidate still gets a valid, readable report for the partial window
   that was captured — not an error or an empty folder.

### Journey 3 — Reviewing a past rehearsal
1. Candidate has a previous run's raw data on disk.
2. Candidate regenerates or re-reviews the human-readable report from that
   raw data without needing to re-run the rehearsal.

### Journey 4 — Unattended / piped run
1. Candidate (or a script) starts a rehearsal with output redirected to a
   file or another process, rather than watched live in a terminal.
2. The tool still produces complete, correctly formatted output — without
   assuming a live, interactive terminal is attached.

## Functional Requirements

1. The candidate can start a timed rehearsal specifying a duration matching
   their intended exam length.
2. During the rehearsal, the tool continuously observes: network
   reachability/quality, host power-source state, and host system load, at
   a regular, short sampling cadence.
3. The tool evaluates the captured rehearsal data against a fixed set of
   **named failure signatures**, each representing a documented real-world
   proctoring-platform risk (sustained connection loss, IP address change
   mid-session, video-feed-risk jitter, an actual power-source transition,
   and silent name-resolution stalls).
4. Each failure signature resolves to one of exactly three states: PASS,
   WARN, or FAIL, with the evidence (time window) backing that state.
5. The rehearsal produces one overall verdict, which is the worst of all
   individual signature states.
6. The candidate can view rehearsal progress live if running in an
   interactive terminal, with enough information to know the run is healthy
   without needing to read raw data.
7. If the rehearsal is not running in an interactive terminal (e.g. output
   is redirected or piped), the tool produces plain, sequential status
   output with no assumption of cursor control.
8. If the candidate interrupts the rehearsal before its scheduled duration
   elapses, the tool stops promptly and still produces a complete,
   consistent report reflecting only the data captured up to that point.
9. Every completed or interrupted rehearsal produces:
   - a raw, complete record of every sample and event captured (for
     re-processing or independent inspection),
   - a machine-readable verdict summary,
   - a human-readable report explaining the verdict, its evidence, and a
     timeline of relevant events.
10. The candidate can regenerate the human-readable and machine-readable
    reports from a previously captured raw record, without re-running the
    rehearsal.
11. The tool must never claim to sense the UPS or grid directly; any
    power-related finding must be presented as inferred from the host
    machine's own power state.
12. The tool must not transmit any captured data off the candidate's
    machine.

## Acceptance Criteria

- **AC-1:** Given a rehearsal is started with a specified duration, when the
  duration elapses without interruption, then the tool exits cleanly and all
  three output artifacts (raw record, machine-readable summary,
  human-readable report) exist and are internally consistent with each
  other (same verdict, same evidence windows).
- **AC-2:** Given a rehearsal captures a sustained total loss of network
  reachability for at least the documented threshold duration, when the
  rehearsal completes, then the corresponding failure signature is reported
  as FAIL with an evidence window matching the actual loss window.
- **AC-3:** Given a rehearsal captures no anomalies at all, when the
  rehearsal completes, then every signature is PASS and the overall verdict
  is PASS.
- **AC-4:** Given a rehearsal is interrupted by the candidate before its
  scheduled duration, when the interruption occurs, then the tool exits
  within a bounded, short time and produces a valid (non-empty,
  non-corrupt) report reflecting only the captured partial window.
- **AC-5:** Given the tool's output is redirected to a file rather than a
  live terminal, when the rehearsal runs, then the output contains no
  terminal control sequences and is fully readable as flat text.
- **AC-6:** Given a host power-source transition occurs during a rehearsal
  (AC to battery, or battery to AC), when the rehearsal completes, then the
  report shows the transition's precise timestamp and correlates it against
  any network anomalies in the same window, and the report text does not
  claim direct UPS/grid sensing.
- **AC-7:** Given a previously captured raw record, when the candidate
  requests a report from it directly, then the same human-readable and
  machine-readable outputs are produced without needing a new rehearsal
  run.
- **AC-8:** Given the candidate provides an invalid rehearsal configuration
  (e.g., a nonsensical duration), when they attempt to start a rehearsal,
  then the tool refuses to start, explains why, and creates no partial
  output.

## Success Criteria

- A candidate can complete one full rehearsal cycle (start → outage →
  review report) without needing to read source code or raw data to
  understand the result.
- The verdict and evidence in the human-readable report are sufficient, on
  their own, for the candidate to decide whether their setup is
  exam-ready — measured by: every FAIL/WARN signature names a concrete time
  window and a plain-language reason, with no signature left unexplained.
- Interrupting a rehearsal never produces an unusable (empty, corrupt, or
  contradictory) output — measured by: 100% of interrupted-rehearsal test
  scenarios yield a report that parses successfully and matches the
  captured data.
- The tool never overstates its own sensing capability — measured by: the
  UPS/grid limitation is stated in the tool's own output/documentation, not
  only in developer-facing docs.

## Edge Cases

- Rehearsal started while already on battery power (no AC→battery
  transition occurs, but the whole run is on battery).
- Rehearsal where the network never recovers before the run ends (loss
  persists through to completion).
- Rehearsal duration of a length that doesn't cleanly divide into the
  sampling cadence.
- Rehearsal interrupted within the first sampling interval (near-zero data
  captured) — must still produce a valid, clearly "incomplete" report
  rather than a crash.
- Public IP address changes during the run for a reason unrelated to any
  outage (e.g., ISP-side reassignment) — reported the same as any other
  churn event, since the tool cannot distinguish cause, only occurrence.
- Requesting a report from a raw record that is itself truncated (e.g., from
  a prior hard crash rather than a graceful interruption) — must degrade to
  "report on available data" rather than fail outright.
- Two rehearsals started with overlapping/colliding output locations.

## Data Sensitivity & Compliance Notes

- Captured data includes the candidate's public IP address history and
  rough network-performance characteristics. This is sensitive to the
  extent any IP address is sensitive, but is standard networking
  information, not a special compliance category (no health, payment, or
  government-ID data is involved).
- All captured data stays local to the candidate's machine by design (see
  Non-Goals); there is no transmission, account system, or third-party
  storage in scope, which avoids most data-protection obligations that
  would otherwise apply to a networked service.
- No regulatory certification (e.g., accessibility, exam-security
  standards) is claimed or targeted by this spec.

## API / Integration Expectations

- No network service, API, or webhook integration is in scope — this is a
  local command-line rehearsal tool, invoked directly by the candidate.
- The tool depends on outbound reachability to a small number of
  well-known public internet endpoints (for reachability/latency probing
  and public-IP detection) and on reading the host machine's own
  power-source state. No inbound network exposure is created.
- Raw and summary output formats are treated as a stable local contract:
  something a candidate (or a future script) can reasonably expect to
  parse or re-process later, even though no external consumer is
  specified in this version.

## Assumptions

- The candidate has administrative ability to run the tool on their own Mac
  and to trigger their own outage test (UPS, breaker, or scheduled
  maintenance) — the tool does not orchestrate the outage itself.
- A rehearsal is a single, self-contained session per invocation; there is
  no requirement to compare across multiple past rehearsals in this
  version.
- "Exam window" duration is candidate-supplied and trusted as accurate;
  the tool does not attempt to detect or infer the candidate's actual exam
  schedule.
- macOS is an acceptable platform restriction for v1, consistent with the
  candidate's own primary machine.

## Open Questions

1. **Live-view density:** should the live view surface one health
   indicator (connection RTT) or several stacked indicators (RTT, loss,
   jitter) at once? *Default assumption if undecided:* show all three,
   favoring completeness over minimalism, since the candidate is watching
   voluntarily and can ignore extra detail.
2. **Public-IP check reliability:** the IP-churn signature depends on an
   external endpoint that could itself be unreachable or rate-limiting.
   *Default assumption if undecided:* treat repeated failures to check
   public IP as a neutral/unknown state for that signature, not a false
   FAIL.
3. **Retention of past rehearsal output:** should old rehearsal folders be
   auto-cleaned, or left entirely to the candidate? *Default assumption if
   undecided:* leave retention to the candidate; the tool does not delete
   its own past output.

None of the above block acceptance-criteria testability or change data
sensitivity/compatibility; they are deferred to `agentic-sdd-plan` with the
stated defaults as assumptions, per the low-risk clarification rule.

## Human Approval Status

**Approved** by the project owner (2026-07-28). Cleared to proceed to
`agentic-sdd-tasks` execution and implementation.
