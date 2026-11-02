# Requirements — T-013 P-10: Agent permission manifest generation

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved
Roadmap ref: `ROADMAP.md` T-013

## 1. Goal
Provide native agent permission manifest generation (`sdd permissions` and `sdd init --permissions`) that generates IDE and agent configuration files (Claude Code, Cursor, Gemini/Antigravity) technically denying execution of git write/mutation commands to enforce Hard Rule 2 ("Never commit or push").

## 2. Context and references
- Mission / principles touched: Principles 1 (Deterministic), 2 (Fail safe), 3 (Human in the loop for the irreversible), Hard Rule 2 ("Never commit or push. Suggest; the human commits.")
- Decisions this relies on: `D-003` (Go), `D-004` (Toolchain), `D-007` (Standard library only), `D-010` (Never overwrite human work), `D-022` (Agent permission manifest generation schema and targets)
- Depends on tasks: `T-002` (Done)

## 3. Scope
**In scope**
- Standalone CLI command: `sdd permissions [flags]`.
- Optional integration flag: `sdd init --permissions`.
- Target platforms:
  - Claude Code: `.claude/settings.json` denying bash commands: `git add`, `git commit`, `git push`, `git tag`, `git reset`, `git rebase`, `git stash`, `git checkout --`, `git merge`, `git cherry-pick`.
  - Cursor: `.cursor/rules/sdd-git-protection.mdc` denying git write operations.
  - Gemini / Antigravity: `.gemini/rules/git-protection.md` denying git write operations.
- Non-destructive merging: if `.claude/settings.json` already exists, merges denied commands into `permissions.deny` without overwriting existing human settings.
- Dry run support (`--dry-run`) to preview planned permission files.
- Filesystem safety: confinement to project root via `os.Root` and atomic writes via `fsutil.WriteAtomicRoot`.
- Consistency check: `sdd check` verifies syntax of generated permission manifests if present.

**Out of scope**
- Shell-level OS sandboxing or kernel-level filesystem locks.
- Network proxy interception.

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL provide a `permissions` command (`sdd permissions`) that generates agent permission configuration files denying git state mutation commands. | MUST | test |
| R-002 | The system SHALL generate `.claude/settings.json` with a `permissions.deny` list containing all git write commands (`git add`, `git commit`, `git push`, `git tag`, `git reset`, `git rebase`, `git stash`, `git checkout --`, `git merge`, `git cherry-pick`). | MUST | test |
| R-003 | The system SHALL generate `.cursor/rules/sdd-git-protection.mdc` specifying strict prohibition of git state modifications. | MUST | test |
| R-004 | The system SHALL generate `.gemini/rules/git-protection.md` specifying strict prohibition of git state modifications. | MUST | test |
| R-005 | The system SHALL safely merge new deny rules into existing JSON settings files without destroying pre-existing user configurations. | MUST | test |
| R-006 | The system SHALL support `--dry-run` to preview all generated permission manifests without writing to disk. | MUST | test |
| R-007 | The system SHALL support `--permissions` in `sdd init` to generate permission manifests during project initialization. | MUST | test |
| R-008 | The system SHALL confine all filesystem operations within the project root using `os.Root` and atomic writes. | MUST | test |

## 5. Non-functional requirements
- **Determinism:** File generation, JSON formatting, and rule arrays SHALL be deterministically sorted and byte-identical across identical runs.
- **Fail-safe behaviour:** Existing files SHALL never be silently overwritten; replacements require explicit flags and leave timestamped backups.
- **Dependencies:** Standard library only (`D-007`).

## 6. Constraints and assumptions
- Agent IDEs read project-relative configuration files (`.claude/`, `.cursor/`, `.gemini/`).
- Markdown rules complement technical JSON permission blocks.

## 7. Open questions
(None - settled in Decision D-022)

## 8. Acceptance criteria
- Given a project root, when `sdd permissions` is run, then `.claude/settings.json`, `.cursor/rules/sdd-git-protection.mdc`, and `.gemini/rules/git-protection.md` are created with git write restrictions.
- Given a project with an existing `.claude/settings.json`, when `sdd permissions` is run, then git deny rules are added while preserving existing settings.
- Given `sdd permissions --dry-run`, when run, then the plan is printed and no files are written.
