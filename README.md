# sdd

**A spec-driven development kit generator.** Run `sdd init` in any project, new or existing, and it creates the files that let any AI coding agent take over development under strict, human-approved gates:

- agent contracts: `CLAUDE.md`, `GEMINI.md`, `AGENTS.md` (identical rules)
- living documents: `MISSION.md`, `TECH_STACK.md`, `ROADMAP.md`, `SKILLS.md`
- per-task templates in `features/_templates/` (`requirements.md`, `plan.md`, `validation.md`)
- a `.sdd/manifest.json` recording what was generated

The workflow it installs: the agent drafts requirements and a plan, **the human approves before any code**, the agent implements and verifies, and **the human reviews and commits**. The anthem: *always be deterministic, and take human approval or decisions to accomplish that determinism.*

> Status: **v0.1.0, built and tested on Go 1.22 / Linux.** Not yet verified on macOS or with Go 1.27. See [ROADMAP.md](ROADMAP.md).

## Install

From source (needs Go):

```bash
cd /Users/vikash/projects/sdd
make install        # installs `sdd` into $(go env GOBIN) or $(go env GOPATH)/bin; put that on your PATH
which sdd           # check that no other `sdd` shadows it
sdd version
```

Or use a prebuilt, **unsigned** binary from `dist/` (macOS may quarantine a downloaded binary: `xattr -d com.apple.quarantine sdd_darwin_arm64`).

## Use

```bash
cd your-project
sdd init                 # interactive: detects what it can, asks the rest, shows a plan, asks before writing
sdd init --yes --name acme --lang go,java      # non-interactive
sdd init --dry-run --yes                       # show the plan, write nothing

sdd check                # mechanical consistency check of the kit (exit 1 on errors)
sdd check --strict       # warnings (such as unresolved TODO(human) placeholders) also fail

sdd feature new T-003 upgrade-command         # creates features/T-003-upgrade-command/ from your templates
sdd prompt gemini        # prints a handover prompt for an agent (claude, gemini, agents, or any name)
```

### Greenfield and brownfield

`sdd init` detects which you have (source files present means brownfield) and asks you to confirm.

- **Greenfield** seeds `T-001 Define mission and scope`.
- **Brownfield** seeds `T-001 Discovery`: the agent reads the codebase under the same gates and drafts the facts for your approval. The tool itself only looks at file names and a few manifests, so everything it detects is marked *unverified*.
- It **never overwrites**. Existing files are skipped; existing `CLAUDE.md`/`GEMINI.md`/`AGENTS.md` get a clearly marked managed block appended and your text is kept; `README.md` is never touched. `--force` replaces generated files after backing the originals up in `.sdd/backup/`.
- It refuses to run in `$HOME` or `/`, and refuses a project that already has a hand-made kit (all four core documents) but no manifest, because it would append a second contract to the agent files. Use `sdd check` on such projects; `sdd adopt` is planned (T-004).

## What `sdd check` verifies

Required files exist · agent contracts are identical across agent files (managed block, or everything after the title line for older kits) · every `D-xxx` and `T-xxx` reference resolves · tables and code fences are intact · task statuses are valid and agree with `features/` folders · approval records and validation results are filled once a task is past those gates · `SKILLS.md` has the "Human feedback rules" section · unresolved `TODO(human)` placeholders are counted.

## Improving the markdown later

Templates are embedded in the binary. To change them without rebuilding, point `--templates <dir>` at a directory containing replacement files with the same relative paths (for example `base/MISSION.md.tmpl`, `packs/go/standards.md`, `features/plan.md`); anything you do not override falls back to the embedded version. Template syntax is Go `text/template`. The manifest records the template version and a hash of each generated file so a future `sdd upgrade` can tell untouched files from ones you edited.

## Development

```bash
make verify     # gofmt -s check, go vet, tests (Go toolchain only)
make check      # verify + golangci-lint v2
make dist       # static binaries for darwin/linux (amd64, arm64) + SHA256SUMS
```

Standard library only; no third-party dependencies. This repository is itself developed with its own kit: start with [CLAUDE.md](CLAUDE.md) and [ROADMAP.md](ROADMAP.md).
