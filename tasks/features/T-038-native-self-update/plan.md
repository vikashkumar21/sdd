---
status: draft
task_id: T-038
---
# Plan — T-038 Native sdd self-update CLI binary updater

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Draft

## 1. Approach summary
Implement a self-contained in-place binary self-update engine in `internal/update` using pure Go standard library (`net/http`, `crypto/sha256`, `os`, `path/filepath`, `runtime`), wire the CLI command `sdd self-update` (with alias `sdd update`) into `internal/cli`, provide flags (`--check`, `--dry-run`, `--version`, `--force`, `--yes`, `--json`), and verify with 100% offline unit and integration tests using `httptest.Server`.

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| D-034 | Native binary self-update architecture and verification | Option A (In-place self-update via sdd self-update with GitHub Releases API, SHA256 verification, atomic swap, and rollback) / Option B (Package managers only) | Option A | Approved 2026-11-10 |

## 3. Design

### Core Updater Package (`internal/update/update.go`)
- Types:
  ```go
  type Release struct {
      TagName     string  `json:"tag_name"`
      HTMLURL     string  `json:"html_url"`
      Assets      []Asset `json:"assets"`
      PublishedAt string  `json:"published_at"`
  }

  type Asset struct {
      Name               string `json:"name"`
      BrowserDownloadURL string `json:"browser_download_url"`
      Size               int64  `json:"size"`
  }

  type CheckResult struct {
      CurrentVersion  string `json:"current_version"`
      LatestVersion   string `json:"latest_version"`
      UpdateAvailable bool   `json:"update_available"`
      AssetURL        string `json:"asset_url,omitempty"`
      ChecksumURL     string `json:"checksum_url,omitempty"`
      AssetName       string `json:"asset_name,omitempty"`
      TargetTag       string `json:"target_tag,omitempty"`
  }

  type Options struct {
      BaseURL        string       // custom API base URL (defaults to GitHub Releases API)
      Client         *http.Client // custom HTTP client
      CurrentVersion string       // current binary version
      TargetVersion  string       // specific version tag requested (--version)
      GOOS           string       // defaults to runtime.GOOS
      GOARCH         string       // defaults to runtime.GOARCH
      Force          bool         // reinstall or overwrite same version
      CheckOnly      bool         // check only
      DryRun         bool         // preview download & checksum without replacing
  }
  ```
- Workflow:
  1. `Check(ctx context.Context, opts Options) (CheckResult, error)`:
     - Formats endpoint: if `opts.TargetVersion != ""`, uses `.../releases/tags/<tag>`, else `.../releases/latest`.
     - Sets HTTP `User-Agent: sdd/<opts.CurrentVersion>` (required by GitHub API).
     - Compares semver of release tag against `opts.CurrentVersion`.
     - Finds platform asset matching `sdd_<goos>_<goarch>[.exe]` and `SHA256SUMS`.
     - Returns `CheckResult`.
  2. `DownloadAndVerify(ctx context.Context, res CheckResult, opts Options) ([]byte, error)`:
     - Downloads `SHA256SUMS` and binary bytes using bounded readers (`io.LimitReader`: 1MB limit for checksums, 100MB limit for binary asset) to prevent DoS/memory exhaustion.
     - Computes SHA256 digest of downloaded binary bytes.
     - Parses `SHA256SUMS` lines (`<hash>  <filename>`), tolerating variable whitespace and trimming optional `*` binary mode indicator prefix from filenames.
     - Verifies exact match of computed SHA256 against published checksum for `res.AssetName`.
     - Returns verified bytes or checksum mismatch error.
  3. `ReplaceExecutable(execPath string, newBytes []byte) error`:
     - Resolves symlinks using `filepath.EvalSymlinks(execPath)` so the actual executable binary is updated rather than clobbering a symlink.
     - Staged write to temporary file `targetPath + ".tmp"`.
     - Sets permissions `0755` on Unix.
     - Cross-platform atomic swap:
       - On Unix: `os.Rename(tempPath, targetPath)`.
       - On Windows: attempts to clean up any pre-existing `targetPath + ".old"`, renames current running binary to `targetPath + ".old"`, then renames `tempPath` to `targetPath`. If the second step fails, restores `.old` back to `targetPath`. Removes `.old` if possible or leaves for next run cleanup.

### CLI Command (`internal/cli/self_update.go`)
- Dispatch: `sdd self-update` and `sdd update`.
- Flags:
  - `--check`: Query and print update status without modifying files.
  - `--dry-run`: Download and verify checksum without writing to executable.
  - `--version <tag>`: Pin or install specific version.
  - `--force`: Allow reinstalling or replacing matching version.
  - `--yes`: Skip interactive confirmation prompt.
  - `--json`: Machine-readable output.

## 4. Implementation steps
Each step is small enough to review in one sitting and names its proving check.

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Implement core self-update engine (`Check`, `DownloadAndVerify`, `ReplaceExecutable`) in `internal/update/update.go` with table-driven tests and `httptest.Server`. | `internal/update/update.go`, `internal/update/update_test.go` | `go test -v ./internal/update` |
| 2 | Wire `self-update` and `update` commands into `internal/cli/self_update.go`, `internal/cli/cli.go`, and add CLI tests. | `internal/cli/self_update.go`, `internal/cli/cli.go`, `internal/cli/cli_test.go` | `go test -v ./internal/cli -run TestSelfUpdate` |
| 3 | Update living documents (`README.md`, `ROADMAP.md`, `TECH_STACK.md`). | `README.md`, `ROADMAP.md`, `TECH_STACK.md` | `sdd check --strict`, `make build` |
| 4 | Run verification suite (`make verify`, `make check`), invoke 3-subagent parallel review triad on T-038, remediate findings, and present GATE 2. | `tasks/features/T-038-native-self-update/` | `make check`, `review.md` sign-off |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/update/update.go` | create | Core binary self-update engine (release discovery, checksum, atomic replacement) |
| `internal/update/update_test.go` | create | Unit and integration tests with offline httptest.Server |
| `internal/cli/self_update.go` | create | CLI subcommand handler for sdd self-update and sdd update |
| `internal/cli/cli.go` | modify | Dispatch self-update and update commands |
| `internal/cli/cli_test.go` | modify | Test CLI command routing and flags |
| `README.md` | modify | Document sdd self-update command usage and options |
| `ROADMAP.md` | modify | Track T-038 status and decisions |
| `TECH_STACK.md` | modify | Record Decision D-034 approval and verified fact |
| `tasks/features/T-038-native-self-update/requirements.md` | modify | Requirements specification |
| `tasks/features/T-038-native-self-update/plan.md` | modify | Implementation plan |
| `tasks/features/T-038-native-self-update/validation.md` | modify | Validation traceability matrix |
| `tasks/features/T-038-native-self-update/review.md` | modify | Review triad audit artifact |

## 6. Dependencies
Standard library only (`D-007`). Zero new external dependencies.

## 7. Living docs to update

| File | Change |
|---|---|
| `ROADMAP.md` | Status to `Implementing`, then `Awaiting review`, then `Done` |
| `TECH_STACK.md` | Record Decision D-034 Approved and verified fact for T-038 in Section 7 |
| `README.md` | Document `sdd self-update` usage |

## 8. Risks, mitigations, rollback
- **Risk:** Network errors or rate limits on GitHub API.
  - **Mitigation:** Clear, user-friendly error messages advising user to check network connectivity or try again. In offline tests, custom `BaseURL` is injected via `httptest.Server`.
- **Risk:** Corrupted download or tampered binary.
  - **Mitigation:** Strict SHA256 checksum verification against `SHA256SUMS` before replacing binary. If checksum fails, abort immediately without touching the executable.
- **Risk:** Running executable locked by OS during swap (especially Windows).
  - **Mitigation:** Staged temporary write; on Windows rename running binary to `.old` before atomic replacement, with automatic rollback if replacement fails.
- **Rollback:** `git restore` on planned files if needed.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
- `F-001`: Active (binding). All review findings (Blockers, Warnings, Minors) will be resolved before GATE 2 close.

## 10. Approval record
Approved by: Vikash (human)   Date: 2026-11-10   Phrase received: APPROVED T-038
