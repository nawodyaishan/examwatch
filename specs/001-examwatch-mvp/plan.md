# Plan 001 — ExamWatch MVP: Technical Implementation Plan

**Status:** Draft — awaiting human architecture approval
**Spec:** `specs/001-examwatch-mvp/spec.md`
**Primary technical source:** `docs/examwatch-tech-spec.md` v0.1.1

---

## Summary

Build `examwatch` as a single-binary Go CLI that samples network, power, and
system state during a candidate-run rehearsal, evaluates the samples against
named failure signatures, and emits a Markdown + JSON report. This plan
covers the application architecture (already detailed in the tech spec) and,
per this update, a full **build, packaging, and release pipeline** modeled
directly on a sibling project, `yt-transcript-md`, which already ships a
working Homebrew-distributed Go CLI with CI/CD. The goal is parity of
process, not novelty: same shape of Makefile-over-scripts, same GoReleaser
config pattern, same two-workflow CI/CD split (PR/push quality gate vs.
tag-triggered release), same Homebrew tap mechanism.

## Inputs Reviewed

- `specs/001-examwatch-mvp/spec.md` (functional requirements, acceptance
  criteria — spec is drafted but **not yet human-approved**; proceeding at
  the user's explicit direction to draft plan/tasks in parallel, per the
  router's allowance for the user to accept phase-order risk explicitly)
- `docs/examwatch-tech-spec.md` (architecture, metrics, rules engine, UI,
  CLI, output artifacts, testing strategy — Sections 1–13)
- `/Users/nawodyaishan/Documents/GitHub/yt-transcript-md/`:
  - `Makefile` — thin wrapper delegating every target to `scripts/*.sh`
  - `scripts/lib/common.sh` — shared logging (`log_info`/`log_warn`/`die`)
    and guard helpers (`require_var`, `require_semver_tag`,
    `require_missing_tag`)
  - `scripts/{build,clean,lint,mod-verify,release,test,test-docker,
    tidy,tidy-check,vet,verify}.sh` — one script per concern, each
    sourcing `common.sh`
  - `.goreleaser.yml` (v2) — cross-compiles linux/darwin × amd64/arm64,
    `CGO_ENABLED=0`, version metadata injected via `ldflags` into an
    `internal/version` package, tar.gz archives, checksums, changelog
    filtering, and a `brews:` block that pushes a formula to a separate
    `homebrew-tap` repo
  - `.github/workflows/ci.yml` — push/PR gate: `mod-verify`, `tidy-check`,
    `vet`, `golangci-lint` (pinned version), `test`, `build`, `govulncheck`
  - `.github/workflows/release.yml` — tag-push (`v*`) only: re-runs
    `make verify`, `govulncheck`, validates the Homebrew tap token can
    reach the tap repo, then runs GoReleaser with `HOMEBREW_TAP_TOKEN`
  - `.golangci.yml` (v2 config, a focused linter set: errcheck, govet,
    ineffassign, staticcheck, unused, gocritic, misspell, prealloc,
    unconvert, whitespace)
  - `internal/version/version.go` — four package vars (`Version`, `Commit`,
    `Date`, `GoVersion`), all `ldflags`-injected, defaulting to
    `dev`/`none`/`unknown`

## Assumptions

- `examwatch` will use a GitHub repo layout parallel to `yt-transcript-md`
  (module path `github.com/nawodyaishan/examwatch`, binary/formula name
  `examwatch`).
- A separate `nawodyaishan/homebrew-tap` repo already exists and is reused
  across the author's tools (the same tap can host multiple formulas) —
  **this plan does not create that repo**; it only adds a new formula
  target to it via GoReleaser. If no such tap exists yet, its creation is a
  human action outside this plan (see Human Architecture Approval Status).
- CI runs on Linux GitHub-hosted runners only, consistent with
  `docs/examwatch-tech-spec.md` §11.3's existing decision to mock
  macOS-only `pmset` behavior rather than use macOS runners.
- The user prefers Makefile-over-scripts (matching `yt-transcript-md`) over
  a Taskfile, since that is the exact, already-proven pattern being
  replicated; introducing a second orchestration format (Taskfile) alongside
  it would add a dependency (the `task` binary) for no behavioral gain over
  `make`, which is preinstalled on macOS/Linux runners and dev machines.

## Architecture Approach

Application architecture (probes → rules engine → UI/report) is unchanged
from `docs/examwatch-tech-spec.md` §5–§10 and is not re-derived here. This
plan adds one new concern: **build, packaging, and release**, structured
as its own layer that touches no application code paths.

```
Makefile                       — thin target wrapper (mirrors yt-transcript-md)
scripts/
  lib/common.sh                — shared log_info/log_warn/die/require_* helpers
  build.sh                      — go build with ldflags-injected version info
  clean.sh                      — rm -rf bin/, coverage.out
  vet.sh                        — go vet ./...
  lint.sh                       — golangci-lint run ./... (skips gracefully if absent locally)
  mod-verify.sh                  — go mod verify
  tidy.sh / tidy-check.sh         — go mod tidy / drift check
  test.sh                        — go test -race -coverprofile=coverage.out ./...
  test-e2e.sh                    — go test -tags=e2e ./cmd/examwatch/...  (examwatch-specific; yt-transcript-md's analog is test-docker.sh)
  verify.sh                      — mod-verify, tidy-check, vet, lint, test, test-e2e, build in sequence
  release.sh                     — semver/clean-tree guards, make verify, git tag + push

internal/version/version.go     — Version/Commit/Date/GoVersion vars, ldflags-injected

.golangci.yml                   — same focused linter set as yt-transcript-md
.goreleaser.yml                 — cross-compile + archive + checksum + changelog + brews:
.github/workflows/ci.yml        — push/PR gate
.github/workflows/release.yml   — tag-triggered release + Homebrew publish
```

### Divergences from `yt-transcript-md` (deliberate, justified)

| Divergence | Reason |
|---|---|
| `goos` limited to `darwin` only (no `linux` build target in `.goreleaser.yml`'s primary build) | `docs/examwatch-tech-spec.md` §2 states Windows/Linux are explicitly out of scope for v1 because `pmset` is Darwin-specific — shipping a Linux binary that cannot power-sample correctly would misrepresent the tool. If a Linux build is wanted later for the network-only subset, that is a future spec, not this one. |
| `test-e2e.sh` replaces `test-docker.sh` | `examwatch`'s E2E strategy (per `docs/examwatch-tech-spec.md` §11.2) already runs as native `go test -tags=e2e`, not inside Docker — there is no Linux-only behavior under test that needs container isolation, unlike `yt-transcript-md`'s presumed use case. |
| Homebrew formula `test:` block | Adapted to `examwatch`'s actual CLI surface: `system "#{bin}/examwatch", "--help"` (no separate `version` subcommand is specified in `docs/examwatch-tech-spec.md` §9 — plan assumes `--version` flag instead; flagged as an open item below). |

## Affected Modules

- New: `Makefile`, `scripts/*.sh`, `scripts/lib/common.sh`, `.golangci.yml`,
  `.goreleaser.yml`, `.github/workflows/ci.yml`,
  `.github/workflows/release.yml`, `internal/version/version.go`
- New (application code, per existing tech spec, unchanged by this update):
  `cmd/examwatch/`, `internal/probe/`, `internal/rules/`, `internal/ui/`,
  `internal/report/`, `internal/store/`
- No existing code is modified — this is a greenfield repo.

## API / Contract Changes

None. This plan adds an internal `--version` flag (see Open Items) as the
only new CLI surface beyond what `docs/examwatch-tech-spec.md` §9 already
specifies (`run`, `report`, `--duration`, `--out`, `--interval`,
`--no-color`).

## Data Model Changes

None. No changes to `log.jsonl`/`summary.json`/`report.md` shapes from
`docs/examwatch-tech-spec.md` §10.

## Dependency Changes

All additions are **build/CI-time tooling**, not runtime `go.mod` module
dependencies (consistent with the tech spec's zero-runtime-dependency
philosophy in §4):

| Dependency | Type | Alternative considered | Approval status |
|---|---|---|---|
| GoReleaser (`~> v2`, via `goreleaser/goreleaser-action`) | CI-only, pinned action | Hand-rolled `go build` matrix + manual formula bump | **Requires approval** — introduces a new external GitHub Action and release automation surface |
| `golangci-lint` (pinned version, matching `yt-transcript-md`'s `v2.12.2` unless a newer stable exists at implementation time) | CI + optional local dev tool | `go vet` alone | **Requires approval** — new CI tool, though same class already vetted in the sibling project |
| `govulncheck` | CI-only, `go install`'d in-workflow | Skip vulnerability scanning | **Requires approval** — network install step inside CI |
| `nawodyaishan/homebrew-tap` GitHub repo + `HOMEBREW_TAP_TOKEN` secret | External infra + secret | Self-hosted formula in the `examwatch` repo itself (`brew install --HEAD` style) | **Requires approval — hard stop.** Per this project's safety boundaries, secrets/credentials and cross-repo infrastructure changes always require explicit human action; this plan cannot create the token or verify tap-repo access on its own. |

No changes to the Go module dependency table in `docs/examwatch-tech-spec.md`
§4 (`pro-bing`, `gopsutil/v4`, `golang.org/x/term`) — those remain as
specified there; the E2E-only `github.com/creack/pty` dependency mentioned in
§11.2 is a `_test.go`/`e2e`-tagged import only, and is a build-time
test dependency, not a shipped-binary dependency, so it does not affect the
release-binary `go.sum` closure in a way GoReleaser's `before.hooks` (`go mod
tidy`) would need special-casing beyond what already exists in the
reference project.

## Security Impact

- Release workflow requires `contents: write` and `id-token: write`
  permissions (matching `yt-transcript-md` exactly) — scoped to the
  `release` GitHub Environment, not the default workflow permission set.
- `HOMEBREW_TAP_TOKEN` is a cross-repo credential; it must be stored as an
  encrypted Actions secret on the `release` environment, never committed,
  and never echoed in workflow logs (the reference project's
  `gh api ... --jq .default_branch` token-validation step avoids printing
  the token itself — this plan replicates that exact validation step
  rather than inventing a new one).
- `govulncheck` runs on every CI push/PR and again before every release,
  matching the reference project's belt-and-suspenders placement.

## Authorization Boundaries

- No change to `examwatch`'s own runtime authorization model (it has none —
  single local user, per spec's Users/Actors section).
- Repository-level authorization (who can push tags, who can approve the
  `release` GitHub Environment) is an org/repo setting outside this plan's
  scope — flagged for human setup, not automated.

## Observability Impact

- CI adds standard GitHub Actions run visibility (pass/fail per job) for
  every push/PR.
- Release workflow logs are the only "production" observability surface,
  since `examwatch` itself has no telemetry (by design, per spec Non-Goals).

## Testing Strategy

No change to `docs/examwatch-tech-spec.md` §11 (unit + E2E plan already
covers application behavior). This plan adds one CI-level verification
concern: **the packaging pipeline itself must be testable without cutting a
real release.**

- `scripts/build.sh` is exercised by `make build` in every CI run (`ci.yml`),
  proving the `ldflags`-injected version build works pre-release.
- `.goreleaser.yml` correctness is checked locally via `goreleaser release
  --snapshot --clean --skip=publish` (a dry run that builds all
  archives/checksums without touching the Homebrew tap or GitHub Releases)
  — this is the task-level verification command for the packaging task, not
  a new automated CI job, to avoid running a full cross-compile matrix on
  every PR.
- `scripts/release.sh`'s guards (`require_semver_tag`, `require_missing_tag`,
  clean-worktree check) are the only packaging logic worth unit-testing in
  Go/Bats-style tests; given their small size and direct 1:1 correspondence
  with the proven `yt-transcript-md` implementation, this plan copies them
  verbatim rather than re-deriving alternative logic, and relies on manual
  verification (`make release V=v0.0.0-test MSG=test` against a disposable
  local tag, never pushed) before the first real tag.

## Failure Modes

| Failure | Detection | Response |
|---|---|---|
| `go mod tidy` drift | `tidy-check.sh` fails CI | Block merge; developer runs `make tidy` |
| Lint/vet regression | CI job fails | Block merge |
| Vulnerable dependency found by `govulncheck` | CI or release job fails | Block merge/release; triage the advisory |
| Homebrew tap push fails (bad token, tap repo unreachable) | Release workflow's token-validation step fails **before** GoReleaser runs | Release job fails fast with a clear error instead of a partial GoReleaser run; GitHub Release itself (via `goreleaser release`) is atomic per GoReleaser's own behavior — a Homebrew-stage failure after GitHub Release publish would leave the release published but the tap formula stale, which must be manually re-run (`goreleaser release --clean` is idempotent for a given tag only if the tag's GitHub Release doesn't already exist — see Rollback) |
| Tag pushed for a version that already has a formula | GoReleaser overwrites the formula file in the tap repo on rebuild | Acceptable — matches upstream GoReleaser behavior; not a new risk introduced here |

## Rollback and Recovery

- CI failures block merge; no rollback needed (nothing shipped).
- A bad release tag: delete the GitHub Release and the tap-repo formula
  commit manually (this plan does not automate release deletion — that is
  a destructive, human-triggered action per this project's safety
  boundaries), then delete and re-push a corrected tag only after
  confirming no consumer has already run `brew install` against the bad
  version (GoReleaser/Homebrew have no automated "yank" — this is a manual,
  rare-path operation, same as in the reference project).
- `scripts/release.sh`'s `require_missing_tag` guard prevents accidentally
  overwriting an existing tag locally, but does not prevent a bad *push* —
  this is an accepted, documented limitation carried over unchanged from
  the reference implementation.

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| Shipping a Homebrew formula implies a stability/support expectation the MVP may not be ready for | README (per tech spec §12) already documents scope boundaries; this plan does not add marketing claims — formula description stays factual ("pre-exam network/power rehearsal tool") |
| Divergent linter/version pins from the reference project drift over time and cause confusing inconsistency across the author's tools | Pin `golangci-lint` to the same version as `yt-transcript-md` at plan-writing time, and note the version explicitly in `.golangci.yml`/`ci.yml` so future upgrades are a deliberate, visible diff |
| `--version` flag is assumed but not yet in the spec/tech-spec CLI table | Flagged below as an open item for spec/tech-spec update before task execution reaches the CLI task |

## Open Items (raised during planning, not yet resolved)

1. `docs/examwatch-tech-spec.md` §9 does not currently list a `--version`
   flag or `version` subcommand, but the Homebrew formula `test:` block
   pattern from `yt-transcript-md` expects one. **Recommendation:** add a
   `--version` flag to the tech spec's CLI table as a small, low-risk
   addition before the corresponding task is implemented. Not blocking
   plan approval — flagged for the tasks phase to route through `spec`
   update via `agentic-sdd-spec` if the user wants it formalized, or
   accepted as an assumption if the user prefers to keep moving.
2. Existence of the `nawodyaishan/homebrew-tap` repository has not been
   verified as part of this plan (no cross-repo read access assumed). Must
   be confirmed before the release task can be executed for real.

## Human Architecture Approval Status

**Approved** by the project owner (2026-07-28), including the proposed
build/CI dependencies (GoReleaser, golangci-lint, govulncheck) and the
Makefile-over-scripts approach. This approval covers plan authoring and
task execution for all Track B tasks **except** the items below, which
remain independently gated per this project's standing safety boundaries
and are not unblocked by this plan approval alone:

- **T-B6** (Homebrew tap repo + `HOMEBREW_TAP_TOKEN` provisioning) — still
  a human-only action; plan approval does not create secrets or external
  repos.
- **T-B9 execution** (actually pushing a release tag that triggers
  `release.yml`) — authoring the workflow file is unblocked now; triggering
  a real release still requires explicit approval at the time of the tag
  push, every time.
