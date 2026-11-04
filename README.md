# sdd

**A spec-driven development kit generator.** Run `sdd init` in any project, new or existing, and it creates the files that let any AI coding agent take over development under strict, human-approved gates:

- agent contracts: `CLAUDE.md`, `GEMINI.md`, `AGENTS.md` (identical rules)
- living documents: `MISSION.md`, `TECH_STACK.md`, `ROADMAP.md`, `SKILLS.md`
- per-task templates in `features/_templates/` (`requirements.md`, `plan.md`, `validation.md`)
- a `.sdd/manifest.json` recording what was generated

The workflow it installs: the agent drafts requirements and a plan, **the human approves before any code**, the agent implements and verifies, and **the human reviews and commits**. The anthem: *always be deterministic, and take human approval or decisions to accomplish that determinism.*

> Status: **v1.2.0, built and verified on Linux, macOS, and Windows with Go 1.27.** See [ROADMAP.md](ROADMAP.md).

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
- **Uninstallation:** Run `sdd uninstall --yes` to cleanly remove the binary and restore your shell configuration.

### 3. From source (needs Go)

```bash
cd /Users/vikash/projects/sdd
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

sdd feature new T-003 upgrade-command         # creates features/T-003-upgrade-command/ from your templates
sdd feature new T-003 upgrade-command --from-jira issue.json # seed requirements from Jira and stage into context
sdd feature new T-003 upgrade-command --from-doc spec.pdf    # stage reference PDF/document and link in requirements
sdd prompt gemini        # prints a handover prompt for an agent (claude, gemini, agents, or any name)

sdd skill new code-review --description "Review code for quality" # scaffold a standard agentskills.io skill
sdd skill new deploy-helper --from-file deploy.txt                # import raw notes with agent refinement scaffolding
sdd skill new quick-fix --from-text "run make verify"            # import raw inline text into skill body
sdd skill list                                # list available skills with scope, status, and descriptions
sdd skill list --json                         # output skill catalog as structured JSON
sdd skill export                              # export canonical sdd-protocol skill to skills/sdd-protocol/
sdd skill export --target ./sdk/skills        # export SDD skill for programmatic Agent SDK applications
sdd skill export --global                     # export SDD skill to user's global agent skill directory

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
- **Exporting for IDEs & SDKs:** `sdd skill export` exports the canonical `sdd-protocol` skill to project repositories, user-global directories (`--global`), or programmatic Agent SDK application folders (`--target <dir>`), allowing programmatic agents (Python/TypeScript/Go) and IDE assistants to enforce SDD gates natively.

## What `sdd check` verifies

Required files exist · agent contracts are identical across agent files (managed block, or everything after the title line for older kits) · every `D-xxx` and `T-xxx` reference resolves · tables and code fences are intact · task statuses are valid and agree with `features/` folders · approval records and validation results are filled once a task is past those gates · `SKILLS.md` has the "Human feedback rules" section · unresolved `TODO(human)` placeholders are counted.

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
