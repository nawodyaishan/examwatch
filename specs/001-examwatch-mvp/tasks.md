# Tasks 001 — ExamWatch MVP

**Status:** Spec and plan **approved** by the project owner (2026-07-28).
Cleared for implementation. Tasks marked "Approval needed: yes — hard stop"
(T-B6) and the release-execution half of T-B9 remain independently gated
per standing project safety boundaries — see plan.md's Human Architecture
Approval Status for the exact scope of what this approval does and does not
unblock.

**Source artifacts:** `specs/001-examwatch-mvp/spec.md`,
`specs/001-examwatch-mvp/plan.md`, `docs/examwatch-tech-spec.md`

---

## Track Summary

Two tracks, independently sequenced:

- **Track A — Application**: probes → rules engine → UI → report → store →
  CLI wiring, per `docs/examwatch-tech-spec.md` §5–§10. Summarized here
  (already detailed in the tech spec); not the focus of this update.
- **Track B — Build, Packaging & Release**: Makefile-over-scripts, GoReleaser,
  Homebrew tap, two-workflow CI/CD — modeled on
  `/Users/nawodyaishan/Documents/GitHub/yt-transcript-md/`. This is the
  focus of this update and is broken out in full detail below.

Track B does not depend on Track A being complete — the packaging
scaffolding (Makefile, scripts, CI quality gate) is valuable from the first
commit and should land early so every subsequent Track A task is verified
the same way (`make verify`) from day one. The release-publishing half of
Track B (GoReleaser, Homebrew, release workflow) does depend on Track A
producing a working `cmd/examwatch` binary, since there is nothing to
release before then.

## Prerequisites

- Go 1.22+ toolchain available locally and in CI (`go.mod` not yet created —
  first Track A task).
- `nawodyaishan/homebrew-tap` repository confirmed to exist and be writable
  by the release token holder (human-verified; see T-B6, Approval needed).
- `HOMEBREW_TAP_TOKEN` secret provisioned on a `release` GitHub Environment
  in the `examwatch` repo (human-provisioned; see T-B6, Approval needed).

---

## Track A — Application (summary; see tech spec for full detail)

| ID | Objective | Verification |
|---|---|---|
| T-A1 | Scaffold `go.mod`, `cmd/examwatch/main.go`, package skeletons per tech spec §5 | `go build ./...` |
| T-A2 | Implement `internal/probe/{network,power,system}.go` behind small interfaces (per plan.md testing strategy) | `go test ./internal/probe/...` |
| T-A3 | Implement `internal/rules/{engine,signatures}.go` | `go test ./internal/rules/...` |
| T-A4 | Implement `internal/store/jsonl.go` | `go test ./internal/store/...` |
| T-A5 | Implement `internal/ui/{header,scroller,flat}.go` | `go test ./internal/ui/...` |
| T-A6 | Implement `internal/report/{markdown,json}.go` + golden fixtures | `go test ./internal/report/...` |
| T-A7 | Wire `cmd/examwatch` CLI (`run`, `report`, flags incl. `--version` — see plan.md Open Item 1) | `go build ./... && ./bin/examwatch --help` |
| T-A8 | E2E test suite (`cmd/examwatch/e2e_test.go`, `e2e` build tag, fake-probe fixtures) | `go test -tags=e2e ./cmd/examwatch/...` |

These are intentionally left at summary depth — `docs/examwatch-tech-spec.md`
§5–§11 already specifies them at implementation detail; expand into full
task-card format via `agentic-sdd-tasks` once Track B scaffolding lands, so
each Track A task can declare `make test`/`make verify` as its verification
command.

---

## Track B — Build, Packaging & Release (full detail)

### T-B1 — Scaffold `internal/version` package

- **Objective:** Add the `ldflags`-injectable version package that both
  `scripts/build.sh` and `.goreleaser.yml` depend on.
- **Source artifacts:** plan.md (Architecture Approach); reference:
  `yt-transcript-md/internal/version/version.go`
- **Allowed files:** `internal/version/version.go`
- **Forbidden files:** everything else
- **Acceptance criteria:** package exports `Version`, `Commit`, `Date`,
  `GoVersion` string vars, each defaulting to `"dev"`, `"none"`,
  `"unknown"`, `"unknown"` respectively.
- **Verification:** `go build ./internal/version/...`
- **Dependencies:** T-A1 (needs `go.mod` to exist)
- **Risk:** low
- **Approval needed:** no
- **Status:** pending

### T-B2 — Add `scripts/lib/common.sh` and core scripts

- **Objective:** Port the shared logging/guard library and the
  non-release scripts verbatim in spirit from `yt-transcript-md`.
- **Source artifacts:** plan.md Architecture Approach diagram;
  `yt-transcript-md/scripts/lib/common.sh`,
  `yt-transcript-md/scripts/{build,clean,vet,lint,mod-verify,tidy,tidy-check,test}.sh`
- **Allowed files:** `scripts/lib/common.sh`, `scripts/build.sh`,
  `scripts/clean.sh`, `scripts/vet.sh`, `scripts/lint.sh`,
  `scripts/mod-verify.sh`, `scripts/tidy.sh`, `scripts/tidy-check.sh`,
  `scripts/test.sh`
- **Forbidden files:** `scripts/release.sh`, `scripts/test-e2e.sh`,
  `scripts/verify.sh` (separate tasks below, since they depend on
  decisions/guards worth reviewing on their own)
- **Acceptance criteria:**
  - `scripts/test.sh` runs `go test -race -coverprofile=coverage.out ./...`
    (adds `-race` beyond the reference project, per
    `docs/examwatch-tech-spec.md` §11.1's explicit race-detection
    requirement for the multi-goroutine probe fan-in)
  - `scripts/build.sh` injects `Version`/`Commit`/`Date`/`GoVersion` into
    `internal/version` via `-ldflags`, output to `bin/examwatch`
  - `scripts/lint.sh` skips gracefully (warns, exit 0) if `golangci-lint`
    is not installed locally, matching the reference project's
    local-dev-friendliness
  - all scripts are executable (`chmod +x`) and start with
    `source "$(dirname "$0")/lib/common.sh"`
- **Verification:** `bash scripts/build.sh && ./bin/examwatch --help`
- **Dependencies:** T-B1
- **Risk:** low
- **Approval needed:** no
- **Status:** pending

### T-B3 — Add `scripts/test-e2e.sh` (examwatch-specific divergence)

- **Objective:** Native `go test -tags=e2e` runner, replacing the reference
  project's Docker-based `test-docker.sh` — no Linux-only behavior needs
  container isolation here (per plan.md's Divergences table).
- **Source artifacts:** plan.md Divergences table; `docs/examwatch-tech-spec.md` §11.2
- **Allowed files:** `scripts/test-e2e.sh`
- **Acceptance criteria:** runs `go test -tags=e2e -v -timeout 180s ./cmd/examwatch/...`
- **Verification:** `bash scripts/test-e2e.sh` (passes once T-A8 lands; may
  be a no-op/skip until then)
- **Dependencies:** T-B2
- **Risk:** low
- **Approval needed:** no
- **Status:** pending

### T-B4 — Add `scripts/verify.sh` and `Makefile`

- **Objective:** Single entrypoint (`make verify`) chaining all quality
  gates, and the thin `Makefile` wrapper exposing every script as a target.
- **Source artifacts:** `yt-transcript-md/Makefile`,
  `yt-transcript-md/scripts/verify.sh`
- **Allowed files:** `Makefile`, `scripts/verify.sh`
- **Acceptance criteria:**
  - `scripts/verify.sh` runs, in order: `mod-verify.sh`, `tidy-check.sh`,
    `vet.sh`, `lint.sh`, `test.sh`, `test-e2e.sh`, `build.sh`
  - `Makefile` exposes: `help`, `tidy`, `tidy-check`, `mod-verify`, `vet`,
    `lint`, `test`, `test-e2e`, `build`, `run`, `verify`, `clean`, `tag`,
    `release` — same target names as the reference project plus
    `test-e2e` in place of `test-docker`
  - `make help` lists every target with a one-line description
- **Verification:** `make verify` exits 0 on a clean checkout with Track A
  stubbed enough to build
- **Dependencies:** T-B2, T-B3
- **Risk:** low
- **Approval needed:** no
- **Status:** pending

### T-B5 — Add `.golangci.yml`

- **Objective:** Same focused linter set as `yt-transcript-md`, pinned to
  the same config version.
- **Source artifacts:** `yt-transcript-md/.golangci.yml`
- **Allowed files:** `.golangci.yml`
- **Acceptance criteria:** `version: "2"`; linters enabled: errcheck, govet,
  ineffassign, staticcheck, unused, gocritic, misspell, prealloc,
  unconvert, whitespace; `run.timeout: 5m`; `issues.max-issues-per-linter: 0`,
  `max-same-issues: 0`
- **Verification:** `golangci-lint run ./...` (locally, if installed) or
  `make lint`
- **Dependencies:** T-B2
- **Risk:** low
- **Approval needed:** no
- **Status:** pending

### T-B6 — Provision Homebrew tap access (human action)

- **Objective:** Confirm or create `nawodyaishan/homebrew-tap` and
  provision `HOMEBREW_TAP_TOKEN` as an encrypted secret on a `release`
  GitHub Environment in the `examwatch` repo.
- **Source artifacts:** plan.md Dependency Changes table (hard-stop item);
  plan.md Human Architecture Approval Status
- **Allowed files:** none — this is a GitHub repo/org settings + external
  repo action, not a code change.
- **Acceptance criteria:** `gh api repos/nawodyaishan/homebrew-tap --jq
  .default_branch` succeeds using the provisioned token, run manually by a
  human before T-B9 is executed.
- **Verification:** manual `gh api` call above; no automated check possible
  from inside this repo.
- **Dependencies:** none (can happen in parallel with all other T-B tasks)
- **Risk:** high (secrets, cross-repo infrastructure)
- **Approval needed:** **yes — hard stop, human-only task.** No agent should
  attempt to create tokens, secrets, or the tap repository.
- **Status:** blocked on human action

### T-B7 — Add `.goreleaser.yml`

- **Objective:** Cross-compile config for macOS-only targets (per plan.md's
  Divergences table — no `linux` goos, unlike the reference project),
  with the `brews:` Homebrew publish block.
- **Source artifacts:** `yt-transcript-md/.goreleaser.yml`; plan.md
  Divergences table; `docs/examwatch-tech-spec.md` §2 (macOS-only scope),
  §9 (CLI surface)
- **Allowed files:** `.goreleaser.yml`
- **Acceptance criteria:**
  - `version: 2`, `project_name: examwatch`
  - `builds[0].main: ./cmd/examwatch`, `binary: examwatch`,
    `env: [CGO_ENABLED=0]`, `goos: [darwin]` only, `goarch: [amd64, arm64]`
  - `ldflags` inject into `github.com/nawodyaishan/examwatch/internal/version`
    (module path to be confirmed against actual `go.mod` from T-A1)
  - `archives`, `checksum`, `changelog` sections match the reference
    project's shape
  - `brews[0].repository` targets `nawodyaishan/homebrew-tap`,
    `directory: Formula`, `license: MIT`
  - `brews[0].test` block: `system "#{bin}/examwatch", "--help"` and, if
    T-A7 adds `--version` (plan.md Open Item 1), a second line exercising it
- **Verification:** `goreleaser release --snapshot --clean --skip=publish`
  succeeds locally (per plan.md Testing Strategy — dry run, no publish)
- **Dependencies:** T-A1, T-A7, T-B1
- **Risk:** medium (external tool config; no secrets touched by this task
  itself — the snapshot verification command never publishes)
- **Approval needed:** review recommended before first real use, but the
  file itself can be authored and dry-run-verified without touching
  secrets or the tap repo
- **Status:** pending

### T-B8 — Add `.github/workflows/ci.yml`

- **Objective:** Push/PR quality gate, mirroring the reference project's
  job structure.
- **Source artifacts:** `yt-transcript-md/.github/workflows/ci.yml`
- **Allowed files:** `.github/workflows/ci.yml`
- **Acceptance criteria:**
  - Triggers: `push` to `main`, all `pull_request`
  - Steps: checkout, setup-go (from `go.mod`), `make mod-verify`,
    `make tidy-check`, `make vet`, `golangci-lint-action` (pinned version
    matching T-B5), `make test`, `make test-e2e`, `make build`, install +
    run `govulncheck`
  - No step requires secrets (this workflow never touches
    `HOMEBREW_TAP_TOKEN`)
- **Verification:** workflow passes on a PR against a scaffolded Track A
  stub
- **Dependencies:** T-B4, T-B5
- **Risk:** low — no secrets, no external publish
- **Approval needed:** no (adding CI config is safe; flagged only because
  it introduces the pinned `golangci-lint-action` and `govulncheck` install
  step as new CI-time tooling per plan.md's Dependency Changes table — low
  risk, but noting for completeness)
- **Status:** pending

### T-B9 — Add `.github/workflows/release.yml`

- **Objective:** Tag-triggered release: verify, `govulncheck`, validate
  Homebrew tap token, run GoReleaser.
- **Source artifacts:** `yt-transcript-md/.github/workflows/release.yml`
- **Allowed files:** `.github/workflows/release.yml`
- **Acceptance criteria:**
  - Trigger: `push` tags matching `v*`
  - Permissions: `contents: write`, `id-token: write`; `environment: release`
  - Steps: checkout (`fetch-depth: 0`), setup-go, install `govulncheck`,
    `make verify`, run `govulncheck`, resolve `GOVERSION` env var, validate
    tap token via `gh api repos/nawodyaishan/homebrew-tap --jq
    .default_branch`, run `goreleaser-action` (`~> v2`) with
    `GITHUB_TOKEN`/`HOMEBREW_TAP_TOKEN`/`GOVERSION` env
- **Verification:** cannot be fully verified without a real tag push and a
  live `HOMEBREW_TAP_TOKEN` — this task's real-world verification is
  necessarily deferred to the first actual release, gated by T-B6.
- **Dependencies:** T-B7, T-B8, **T-B6 (must be resolved before this
  workflow is ever triggered for real, though the YAML itself can be
  authored and code-reviewed beforehand)**
- **Risk:** high (this workflow, once triggered, publishes externally and
  consumes a secret)
- **Approval needed:** **yes.** Authoring the workflow file is low-risk and
  can proceed; actually pushing a tag that triggers it requires explicit
  human approval each time, consistent with this project's release-action
  safety boundary.
- **Status:** pending (file authoring only; execution blocked)

### T-B10 — Add `--version` flag to CLI and tech spec (resolves plan.md Open Item 1)

- **Objective:** Small CLI addition so the Homebrew formula's `test:` block
  has something to exercise, and so `internal/version` data is
  user-visible.
- **Source artifacts:** plan.md Open Items §1
- **Allowed files:** `cmd/examwatch/main.go` (or wherever flags are parsed
  per T-A7), `docs/examwatch-tech-spec.md` (§9 CLI Interface table — add
  the row)
- **Acceptance criteria:** `examwatch --version` prints
  `examwatch <Version> (<Commit>, built <Date>)` and exits 0; tech spec §9
  table updated to list it
- **Verification:** `./bin/examwatch --version`
- **Dependencies:** T-A7, T-B1
- **Risk:** low
- **Approval needed:** no
- **Status:** pending

---

## Dependency Order

```
T-A1 → T-B1 → T-B2 → T-B3 → T-B4 (make verify usable)
                  └──────────────→ T-B5 → T-B8 (ci.yml)
T-A7 → T-B10 (--version)
T-A1, T-A7, T-B1 → T-B7 (.goreleaser.yml)
T-B7, T-B8 → T-B9 (release.yml)  [execution gated by T-B6]
T-B6 (human, parallel-safe, no code dependency) → gates real use of T-B9
```

## Parallel-Safe Groups

- **Group 1** (disjoint files, no shared state): T-B1, T-B5, T-B6 can start
  immediately and in parallel — none touch the same files or depend on each
  other.
- **Group 2** (after T-B1): T-B2 and T-B3 touch disjoint script files and
  can proceed in parallel.
- **Group 3**: T-A2, T-A3, T-A4, T-A5 (Track A probes/rules/store/ui) are
  disjoint packages and parallel-safe once T-A1 lands — not detailed as
  full task cards here since they're summarized at tech-spec depth.

## Verification Matrix

| Task | Command |
|---|---|
| T-B1 | `go build ./internal/version/...` |
| T-B2 | `bash scripts/build.sh && ./bin/examwatch --help` |
| T-B3 | `bash scripts/test-e2e.sh` |
| T-B4 | `make verify` |
| T-B5 | `make lint` |
| T-B6 | `gh api repos/nawodyaishan/homebrew-tap --jq .default_branch` (human, manual) |
| T-B7 | `goreleaser release --snapshot --clean --skip=publish` |
| T-B8 | GitHub Actions run on a test PR |
| T-B9 | file review only until a real tag is pushed (human-gated) |
| T-B10 | `./bin/examwatch --version` |

## Blocked or Approval-Required Work

- **T-B6** — hard stop, human-only: Homebrew tap repo + secret provisioning.
- **T-B9** — file authoring is unblocked; triggering it for a real release
  requires (a) T-B6 resolved and (b) explicit human approval to push a
  release tag, every time (this is a standing project rule, not a one-time
  gate).
- **Whole plan** — per plan.md's Human Architecture Approval Status, none
  of the Track B "Approval needed: yes/reviewed" tasks should be executed
  against the real repository/CI until the plan itself is approved; task
  *authoring* (writing the files) may proceed now since the user has
  explicitly asked for it, but treat `git push`, tag creation, and secret
  configuration as separate, individually-gated actions regardless of this
  document's existence.
