# sdd

**A spec-driven development kit generator.** Run `sdd init` in any project, new or existing, and it creates the files that let any AI coding agent take over development under strict, human-approved gates:

- agent contracts: `CLAUDE.md`, `GEMINI.md`, `AGENTS.md` (identical rules)
- living documents: `MISSION.md`, `TECH_STACK.md`, `ROADMAP.md`, `SKILLS.md`
- per-task templates for all work typologies in `tasks/features/_templates/`, `tasks/bugfixes/_templates/`, `tasks/hotfixes/_templates/`, `tasks/spikes/_templates/` with language-pack-aware review rubrics (Go, Java, Python, TypeScript, generic)
- a root `.ragignore` excluding ephemeral and build artifacts from RAG/vector indexing
- safe brownfield proposal files (`README.md.sdd-new`) preserving existing project documentation untouched
- a `.sdd/manifest.json` recording what was generated
- heterogeneous agent role routing and tier-calibrated dispatch in `.sdd/roles.json` with copy-pasteable `invoke_subagent` snippets in `sdd prompt` commands (`Model: "flash"` for routine verification, reducing token usage by $\ge 60\%$)

The workflow it installs: the agent drafts requirements and a plan, **the human approves before any code**, the agent implements and verifies, and **the human reviews and commits**. The anthem: *always be deterministic, and take human approval or decisions to accomplish that determinism.*

> Status: **v1.10.0, built and verified on Linux, macOS, and Windows with Go 1.27.** See [ROADMAP.md](ROADMAP.md).

## Install

### 1. Universal One-Line Installer (Recommended — macOS, Linux, Windows)

No Go toolchain or administrator/root permissions required:

**macOS & Linux (bash, zsh, sh):**
```bash
curl -fsSL https://raw.githubusercontent.com/vikashkumar21/sdd/main/scripts/install.sh | sh
```

**Windows (PowerShell):**
```powershell
irm https://raw.githubusercontent.com/vikashkumar21/sdd/main/scripts/install.ps1 | iex
```

This automatically detects your OS and architecture, downloads the verified prebuilt static binary from GitHub Releases, installs it to `~/.local/bin/sdd` (or `%LOCALAPPDATA%\Programs\sdd\sdd.exe`), and safely configures your shell PATH.

### 2. Double-Click Setup & Manual Binary

Prebuilt static binaries for macOS, Linux, and Windows are available in `dist/` or release downloads:

- **Double-click setup:** Double-click the downloaded binary in macOS Finder or Windows Explorer. An interactive terminal will prompt you to install `sdd` directly to your user path (`~/.local/bin/sdd` on macOS/Linux, `%LOCALAPPDATA%\Programs\sdd\sdd.exe` on Windows) and configure your shell `PATH` (`~/.zshrc`, `~/.bashrc`, or user registry). It pauses with `Press Enter to exit...` so the confirmation remains visible before closing.
- **CLI self-install:** Run `./sdd install` from any downloaded binary directory.
- **In-place self-update:** Run `sdd self-update` (or `sdd update`) to automatically check, download, verify SHA256 integrity, and upgrade your binary to the latest GitHub release.
- **Uninstallation:** Run `sdd uninstall --yes` to cleanly remove the binary and restore your shell configuration.

### 3. From source (needs Go)

```bash
git clone https://github.com/vikashkumar21/sdd.git
cd sdd
make install        # installs `sdd` into $(go env GOBIN) or $(go env GOPATH)/bin; put that on your PATH
which sdd           # check that no other `sdd` shadows it
sdd version
```

Or run `./dist/sdd_darwin_arm64 install` (on macOS clear quarantine if needed: `xattr -d com.apple.quarantine dist/sdd_darwin_arm64`).

## Use

```bash
cd your-project
sdd init                 # interactive: detects what it can, asks the rest, shows a plan, asks before writing
sdd init --yes --name acme --lang go,java      # non-interactive
sdd init --dry-run --yes                       # show the plan, write nothing

sdd check                # mechanical consistency check of the kit (exit 1 on errors)
sdd check --strict       # warnings (such as unresolved TODO(human) placeholders) also fail
sdd check --json         # machine-readable JSON output for CI pipelines and scripts
sdd check --fix          # interactive mechanical repair (contract parity, table alignment)
sdd check --fix --dry-run # preview repairs with unified diffs without modifying files
sdd check --fix --yes    # apply repairs without confirmation (backs up to .sdd/backup/)

sdd status               # summarise roadmap milestones, active tasks, and pending gates
sdd status --json        # machine-readable JSON summary for agent orchestration and CI

sdd task new --type feature T-003 upgrade-command # unified task scaffolding across all 4 typologies under tasks/
sdd feature new T-003 upgrade-command         # creates tasks/features/T-003-upgrade-command/ from your templates
sdd bugfix new T-003 upgrade-command          # creates tasks/bugfixes/T-003-upgrade-command/ (reproduction & RCA)
sdd hotfix new T-003 upgrade-command          # creates tasks/hotfixes/T-003-upgrade-command/ (incident & patch)
sdd spike new T-003 upgrade-command           # creates tasks/spikes/T-003-upgrade-command/ (hypotheses & findings)
sdd task start T-003                          # provision isolated git worktree in .sdd/worktrees/T-003-...
sdd task list                                 # list active task worktrees (--json supported)
sdd task clean T-003                          # safely teardown worktree (creates backup if dirty)
sdd task clean T-003 --purge-state            # clean worktree and completely purge .sdd/state/T-003
sdd task state show T-003 --json              # read state snapshot, events journal, and worktree status
sdd task state save T-003 --state '<json>'    # atomically save task state snapshot (--stdin, --json supported)
sdd task state append-event T-003 --event '<json>' # append lifecycle event to .sdd/state/T-003/events.jsonl
sdd task state record-tokens T-003 --input 1200 --cache 4000 --output 300 # record task token consumption delta (--stdin, --json)
sdd record-tokens T-003 --input 1200 --cache 4000 --output 300 # turnkey top-level shorthand for task token recording
sdd task state purge T-003 --force            # remove task state store from .sdd/state/T-003 (--json supported)
sdd task deconflict                           # analyze plan file collisions across in-flight tasks
sdd task deconflict T-003 T-004               # analyze collisions between specific tasks (--json, --all)
sdd task merge T-003 --dry-run                # qualify pre-merge matrix, blast radius, flake guard, and simulate merge
sdd task merge T-003                          # merge task branch into main after 0 blockers & regression sweeps (passes SDD_HUMAN_APPROVED=1 token)
sdd hook install                              # provision branch-aware Git pre-commit & pre-merge-commit hooks (Hard Rule 2 mechanical lock)
sdd hook install --force                      # overwrite existing SDD hooks (preserves foreign hooks in *.sdd-orig)
sdd hook status                               # inspect hook installation state, target paths, and chaining (--json supported)
sdd hook uninstall --yes                      # safely remove SDD hooks and restore foreign chained hooks
sdd task audit T-003                          # mechanically execute verification commands outside agent narrative (--json, --verbose)
sdd review audit T-003                        # backwards-compatible alias for sdd task audit
sdd feature new T-003 upgrade-command --from-jira issue.json # seed requirements from Jira and stage into context
sdd feature new T-003 upgrade-command --from-doc spec.pdf    # stage reference PDF/document and link in requirements
sdd bugfix new T-003 auth-failure --from-ci failure.log      # seed bugfix reproduction.md from raw or staged CI failure log
sdd ingest --ci-log --file failure.log                       # scrub ANSI/secrets and stage sanitized CI failure log into context
sdd ingest --ci-log --seed-bugfix --task T-003 --file log    # stage sanitized CI log and seed bugfix task folder
sdd prompt gemini        # prints a handover prompt for an agent (claude, gemini, agents, or any name)
sdd prompt gemini --compact # generate high-density micro-context prompt (< 400 tokens)
sdd prompt discovery                        # generate turnkey brownfield exploration prompt (codebase-analyst role, Coverage Ledger, --json)
sdd prompt discovery --ensemble             # generate adversarial discovery brief (unanchored challenger inventory, Coverage Ledger diff, context cross-check)
sdd prompt discovery T-003                  # generate discovery prompt linked to specific task requirements and plan
sdd prompt discovery T-003 --ensemble       # stage stateless review bundle (.sdd/review/T-003/discovery/ with context/) and output ensemble brief
sdd prompt plan-review T-003                  # generate turnkey pre-GATE 1 plan review prompt (--json supported)
sdd prompt implement T-003                    # generate turnkey Phase B implementation prompt (implementer role, --json supported)
sdd prompt implement T-003 --role qa          # generate implementation prompt enriched with QA automation skill
sdd prompt implement T-003 --role devops      # generate implementation prompt enriched with DevOps / SRE skill
sdd prompt review T-003                       # generate turnkey subagent review prompt for Phase C.2 independent audit (Rule F-001)
sdd prompt review T-003 --lens spec          # review focused on requirements traceability & plan boundaries
sdd prompt review T-003 --lens security      # review focused on AppSec, threat surface, input validation & secrets
sdd prompt review T-003 --lens pattern       # review focused on architecture decisions, determinism & language idioms
sdd prompt review T-003 --lens all           # ensemble review covering all three lenses (default)
sdd prompt review T-003 --loop 2             # generate tabula-rasa remediation prompt with isolated finding checklist (loops 1-3)
sdd prompt review T-003 --json               # structured review context bundle with lens metadata for Agent SDKs
sdd prompt repair T-003                       # generate turnkey auto-repair prompt for bugfix reproduction & resolution
sdd prompt repair T-003 --max-loops 3         # configure auto-repair loop ceiling before circuit breaker halt (1-5)
sdd prompt repair T-003 --json                # structured repair context bundle with loop tracking for Agent SDKs

sdd ci init                               # generate CI workflow for GitHub Actions (.github/workflows/sdd-check.yml)
sdd ci init --platform all                # generate workflows for GitHub, GitLab, Bitbucket, Azure, GCP, and AWS
sdd ci init --platform gcp                # generate Google Cloud Build workflow (cloudbuild.yaml)
sdd ci init --platform aws                # generate AWS CodeBuild specification (buildspec.yml)
sdd ci init --dry-run                     # preview planned CI workflows without writing to disk
sdd ci init --force                       # overwrite existing CI workflow files

sdd doc init                              # scaffold docs/architecture/ and docs/business_flows/ with Mermaid diagrams
sdd doc init --dry-run                    # preview planned architecture and flow docs without writing
sdd doc init --force                      # overwrite existing doc files (creates backups in .sdd/backup/docs/)
sdd doc flow user-login                   # scaffold docs/business_flows/user-login.md with sequence & state diagrams
sdd doc flow user-login --title "User Login" # specify human-readable flow title

sdd task pr T-003                         # synthesize task quartet into a Pull Request title and description
sdd task pr T-003 --dry-run               # preview PR title and body on stdout
sdd task pr T-003 --output pr.md          # write generated PR description to a file
sdd task pr T-003 --json                  # output machine-readable PRBundle for GitHub/GitLab CLI or Agent SDKs
sdd task pr T-003 --allow-unreviewed      # package PR even if review is not yet approved

sdd skill new code-review --description "Review code for quality" # scaffold a standard agentskills.io skill
sdd skill new deploy-helper --from-file deploy.txt                # import raw notes with agent refinement scaffolding
sdd skill new quick-fix --from-text "run make verify"            # import raw inline text into skill body
sdd skill list                                # list available skills with scope, status, and descriptions
sdd skill list --json                         # output skill catalog as structured JSON
sdd skill export                                  # export canonical skills (sdd-protocol, code-review, task-coordinator) to skills/
sdd skill export --roles                          # export all 11 canonical role skills (governance, execution, operations, discovery)
sdd skill export --role codebase-analyst          # export specific canonical discovery role skill
sdd skill export --role implementer               # export specific canonical execution role skill (e.g. implementer, qa, devops)
sdd skill export --role appsec-specialist         # export specific canonical governance role skill
sdd skill export --skill task-coordinator         # export specific canonical skill
sdd skill export --skill code-review          # export specific canonical skill
sdd skill export --target ./sdk/skills        # export SDD skills for programmatic Agent SDK applications
sdd skill export --global                     # export SDD skills to user's global agent skill directory

sdd ingest --file notes.txt --name meeting    # stage external files into .sdd/context/
sdd ingest --text "meeting notes..."          # stage raw text strings as context
sdd ingest --jira --file issue.json           # validate and stage Jira issue export into .sdd/context/
sdd ingest --pdf --file architecture.pdf      # validate and stage PDF document into .sdd/context/
cat transcript.txt | sdd ingest               # stage piped stdin

sdd permissions                               # generate agent permission configs denying git writes
sdd permissions --agents claude,cursor        # generate for specific agents
sdd permissions --dry-run                     # preview planned manifests without writing

sdd upgrade                               # refresh kit files from newer templates without overwriting human edits
sdd upgrade --dry-run                     # preview planned updates and diffs without writing files
sdd upgrade --yes                         # apply updates and write proposals without prompting
sdd upgrade --force                       # overwrite edited files (always creates backups in .sdd/backup/)

sdd adopt                                 # adopt a hand-crafted kit by generating .sdd/manifest.json with zero file mutations
sdd adopt --dry-run                       # preview planned manifest JSON without writing
sdd adopt --yes                           # adopt without interactive confirmation
sdd adopt --force                         # re-adopt if a manifest already exists (backs up existing manifest)

sdd install                               # install sdd to ~/.local/bin/sdd and configure shell PATH
sdd install --dry-run                     # preview binary and shell profile changes without writing
sdd uninstall                             # cleanly remove binary and strip PATH additions from shell profile
sdd uninstall --yes                       # uninstall without interactive confirmation

sdd self-update                           # update sdd binary in place to the latest GitHub release (alias: sdd update)
sdd self-update --check                   # check if an update is available without downloading or modifying files
sdd self-update --check --json            # machine-readable JSON update check
sdd self-update --dry-run                 # download and verify binary without replacing executable
sdd self-update --version 1.5.0           # install a specific release version (upgrade or downgrade)
sdd self-update --yes                     # proceed without interactive confirmation
```

### Greenfield and brownfield

`sdd init` detects which you have (source files present means brownfield) and asks you to confirm.

- **Greenfield** seeds `T-001 Define mission and scope`.
- **Brownfield** seeds `T-001 Discovery`: the agent reads the codebase under the same gates and drafts the facts for your approval. The tool itself only looks at file names and a few manifests, so everything it detects is marked *unverified*.
- It **never overwrites**. Existing files are skipped; existing `CLAUDE.md`/`GEMINI.md`/`AGENTS.md` get a clearly marked managed block appended and your text is kept; `README.md` is never touched. `--force` replaces generated files after backing the originals up in `.sdd/backup/`.
- It refuses to run in `$HOME` or `/`, and refuses a project that already has a hand-made kit (all four core documents) but no manifest, because it would append a second contract to the agent files. Run `sdd adopt` on such projects to generate `.sdd/manifest.json` with zero changes to existing files.

### Agent skills and progressive disclosure

`sdd` implements the open Linux Foundation / Anthropic `agentskills.io` standard (`skills/<name>/SKILL.md`):
- **Scaffolding & Ingestion:** `sdd skill new <name>` creates standard skills from scratch, or wraps raw files (`--from-file`) and text (`--from-text`) within an agent refinement block (`<!-- sdd:skill-refine:begin -->`).
- **Discovery:** `sdd skill list` scans `skills/` (project-facing) and `.sdd/skills/` (internal/infrastructure), validating naming and bounds (`--json` supported).
- **Progressive Disclosure Handover:** `sdd prompt <agent>` dynamically compiles a concise table of available skills (name and trigger description, ~50 tokens total). The AI agent knows exactly what skills exist and when to activate them, without bloating prompt context with full markdown bodies. If unrefined raw notes are detected, the agent is directed to synthesize them into standard sections per `SKILLS.md`.
- **Exporting for IDEs & SDKs:** `sdd skill export` exports canonical skills to project repositories, user-global directories (`--global`), or programmatic Agent SDK application folders (`--target <dir>`), allowing programmatic agents (Python/TypeScript/Go) and IDE assistants to enforce SDD gates natively.
- **Role-Based Governance Skills & Heterogeneous Model Routing:** `.sdd/roles.json` defines deterministic phase-to-role assignments and model capability routing (`model_tier`: frontier, advanced, standard, fast; `reasoning_effort`: high, medium, low, none; `recommended_family`). Role skills resolve via a two-tier hierarchy: project overrides in `skills/<role>/SKILL.md` take precedence over infrastructure baselines in `.sdd/skills/<role>/SKILL.md` and embedded templates. Turnkey prompts (`sdd prompt plan-review`, `sdd prompt implement`, `sdd prompt review --lens <lens>`, `sdd prompt discovery`, `sdd prompt repair`) automatically enrich markdown context with `## Model Routing Guidance` and export structured `model_routing` objects in `--json` mode for programmatic Agent SDK orchestrators.

### Universal CI workflows and automated PR packaging

`sdd` bridges local agent development and remote team review with universal CI configurations and automated pull request generation:
- **Universal Multi-Platform CI:** `sdd ci init` generates canonical CI workflows across **GitHub Actions** (`.github/workflows/sdd-check.yml`), **GitLab CI** (`.gitlab-ci.yml`), **Bitbucket Pipelines** (`bitbucket-pipelines.yml`), **Azure DevOps** (`azure-pipelines.yml`), **Google Cloud Build** (`cloudbuild.yaml`), and **AWS CodeBuild** (`buildspec.yml`).
- **Polyglot & Dependency-Free:** Workflows automatically configure language toolchains for Go, Python, TypeScript/Node, Java, and Generic stacks, running `sdd check --strict` via the static binary installer without requiring a Go compiler on Python/Node/Java CI runners.
- **Explicit Compile & Release Stages:** Pipelines enforce linting, explicit compile/build steps before test execution, test suites, determinism flake guards, and include placeholder release stages for publishing application binaries and Docker container images.
- **Automated PR Packaging:** `sdd task pr <T-xxx>` parses task quartet documents across all work typologies (features, bugfixes, hotfixes, spikes), extracts requirements, planned changes, validation matrices, and Review Triad certifications, and synthesizes a standardized PR title (`feat(T-xxx): ...`) and Markdown body.
- **Pre-PR Qualification Gate:** Validates that tasks have 0 unresolved 🔴 Blockers in `review.md` before packaging (or exits code 1 unless `--allow-unreviewed` is specified). Supports `--json` exporting a machine-readable `PRBundle` for GitHub CLI (`gh pr create`), GitLab CLI (`glab mr create`), or Agent SDKs.

### Stateless review context isolation and outside validation

SDD eliminates author agent self-approval bias and conversational anchoring through native tabula-rasa reviewer isolation and mechanical validation auditing:
- **Stateless Review Isolation by Default:**
  - `sdd prompt plan-review <T-xxx>` freezes requirements and implementation plan into `.sdd/review/<T-xxx>/plan/spec.md`, stages canonical plan-reviewer instructions into `rubric.md`, and outputs an isolated, tabula-rasa prompt stripped of historical conversational baggage.
  - `sdd prompt review <T-xxx> [--lens <lens>]` extracts working tree diffs and newly created untracked files into `.sdd/review/<T-xxx>/code/diff.patch`, stages frozen specifications in `spec.md`, pre-seeds the unanchored `ledger.md`, and sets up isolated bundles for each Review Triad auditor (Spec Auditor, AppSec Specialist, Pattern Auditor).
  - Transient review bundles are stored in `.sdd/review/` and automatically excluded from version control and RAG indexing.
- **Unanchored Requirements Traceability Ledger & Omission-First Protocol:**
  - `sdd prompt review <T-xxx>` parses task requirements and auto-generates `.sdd/review/<T-xxx>/code/ledger.md` pre-populated with all `R-xxx` requirement statements.
  - Reviewers follow the Omission-First Protocol, mapping every requirement to concrete code symbols and positive/negative test execution evidence *before* inspecting `diff.patch`, eliminating the "Diff Framing" anchoring bias and omission blindness.
  - `sdd check` mechanically enforces that approved reviews include `## Requirements Traceability Ledger`, cover all declared `R-xxx` requirements without placeholders, verify negative tests, and achieve `Verified` status.
- **Ephemeral Tabula-Rasa Reviewer Lifecycle & Zero-Negotiation Loop Isolation:**
  - Forbids reusing reviewer conversations across remediation loops (`send_message` across loops is strictly prohibited).
  - Every loop dispatches fresh ephemeral subagents (`invoke_subagent`) into isolated tabula-rasa contexts with zero conversational baggage.
  - Eliminates "Repair-Loop Justification Bleed" by stripping developer narrative, apologies, or negotiation transcripts from remediation prompts and bundles.
  - Multi-loop review bundle staging (`sdd prompt review <T-xxx> --loop <2|3>`) generates `.sdd/review/<T-xxx>/code/findings_to_verify.md`, presenting candidate diffs alongside pure, objective defect checklists.
- **Outside Validation Audit Runner:**
  - `sdd task audit <T-xxx>` (and alias `sdd review audit <T-xxx>`) mechanically parses verification commands from `validation.md`, executes them outside the author agent's narrative, and measures true exit codes, durations, and output streams.
  - **Strict Security Whitelisting:** Permits only recognized build, test, lint, and verification binaries (`go test`, `go vet`, `make`, `sdd check`, `sdd status`, `git grep`, `git diff`, `git status`) while strictly rejecting shell redirection, pipes, semicolons, backgrounding, and dangerous flags (`-exec`, `--output`, `make -f`, `make -C`).
  - Supports `--json` output for CI pipeline gate verification and Task Coordinator orchestration.

## What `sdd check` verifies

Required files exist · root `.ragignore` exists · `.sdd/roles.json` schema, role skills integrity, and model tier/reasoning constraints (`roles-tier`, `roles-reasoning`, `roles-family`) · document lifecycle YAML frontmatter metadata (status, date, superseded references) · agent contracts are identical across agent files (managed block, or everything after the title line for older kits) · every `D-xxx` and `T-xxx` reference resolves · TECH_STACK.md §8 cross-project and service dependencies (decision resolution, direction and status enums) · tables and code fences are intact · task statuses are valid and agree with task folders across all typologies (`tasks/features/`, `tasks/bugfixes/`, `tasks/hotfixes/`, `tasks/spikes/`) · worktree branch naming standards · static plan target collisions across concurrent in-flight tasks (`task-overlap`) · 64KB specification context budget guard · bugfix reproduction step verified · requirements traceability ledger completeness and omission absence (`traceability-ledger`) · review remediation hygiene and negotiation absence (`remediation-hygiene`) · approval records and validation results are filled once a task is past those gates · `SKILLS.md` has the "Human feedback rules" section · unresolved `TODO(human)` placeholders are counted.

Finding waivers registered in `.sdd/suppressions.json` are downgraded to informational notes so `--strict` does not exit with failure. In `sdd check --fix`, suppressed repairs are highlighted with `[SUPPRESSED]` badges and can be kept waived or repaired (which clears the waiver). You can also interactively record waivers during `sdd check --fix` by answering `s`. If an issue is resolved, stale waivers are reported in `sdd check` and can be interactively pruned in `sdd check --fix`.

## Improving the markdown later

Templates are embedded in the binary. To change them without rebuilding, point `--templates <dir>` at a directory containing replacement files with the same relative paths (for example `base/MISSION.md.tmpl`, `packs/go/standards.md`, `features/plan.md`); anything you do not override falls back to the embedded version. Template syntax is Go `text/template`. The manifest records the template version and a hash of each generated file so `sdd upgrade` can tell untouched files from ones you edited. Untouched files are updated in place with backups in `.sdd/backup/`, while edited files receive side-by-side `.sdd-new` proposals and unified diff previews so human work is never silently overwritten.

## Development

```bash
make verify     # gofmt -s check, go vet, tests (Go toolchain only)
make check      # verify + golangci-lint v2
make dist       # static binaries for darwin/linux/windows (amd64, arm64) + SHA256SUMS
```

Standard library only; no third-party dependencies. This repository is itself developed with its own kit: start with [CLAUDE.md](CLAUDE.md) and [ROADMAP.md](ROADMAP.md).
