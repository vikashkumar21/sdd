# Requirements — T-003 sdd upgrade: refresh generated files from newer templates without overwriting human edits

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-02)
Roadmap ref: `ROADMAP.md` T-003

## 1. Goal
Refresh generated spec-driven development kit files from newer embedded or override templates using manifest SHA256 hashes, updating untouched files atomically with backups while preserving human-edited files with side-by-side `.sdd-new` proposals and unified diff previews.

## 2. Context and references
- Mission / principles: Principle 1 (Deterministic), Principle 3 (Human in the loop), Principle 5 (Never overwrite human work).
- Decisions this relies on: D-017 (Manifest schema 1, managed blocks, and proposal format).
- Depends on tasks: T-002, T-007 (os.Root confinement), T-012 (safe repair & diff engine).

## 3. Scope
**In scope**
- `sdd upgrade` CLI command accepting `--dir`, `--dry-run`, `--yes`, `--force`, `--templates`.
- Verification of `.sdd/manifest.json` (schema 1) with safe refusal when missing or corrupted.
- Detection of template version changes and template override changes.
- Content hash comparison for full files and managed blocks against manifest hashes.
- Atomic updates for untouched files with backups in `.sdd/backup/<timestamp>/`.
- Safe handling of human-edited files: preserve original, write side-by-side `<file>.sdd-new` proposal, and print unified diff preview.
- Support for `--force` flag to overwrite edited files (always backing up original first).
- Manifest update recording new template version, tool version, and updated file hashes.
- Path-confined operations via `os.Root` and stdlib atomic writes.

**Out of scope**
- Kits lacking `.sdd/manifest.json` (handled by `sdd adopt`, T-004).
- Automatic semantic 3-way merging of arbitrary markdown text.
- Network access or remote template fetching.

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL refuse `sdd upgrade` if `.sdd/manifest.json` does not exist or has an unsupported schema. | MUST | test |
| R-002 | The system SHALL report that the kit is already up-to-date if template versions and contents match and `--force` is not set. | MUST | test |
| R-003 | The system SHALL atomically update untouched files (files matching manifest SHA256) to the new template content and back up the originals in `.sdd/backup/`. | MUST | test |
| R-004 | The system SHALL preserve human-edited files (files differing from manifest SHA256), write a side-by-side proposal file (`<path>.sdd-new`), and output a unified diff preview. | MUST | test |
| R-005 | The system SHALL update managed blocks in agent contract files when the managed block is untouched, preserving any human rules outside the managed block. | MUST | test |
| R-006 | The system SHALL support `--dry-run` to preview all planned updates, diffs, and proposals without modifying the filesystem. | MUST | test |
| R-007 | The system SHALL support `--force` to overwrite human-edited files after backing up original versions to `.sdd/backup/`. | MUST | test |
| R-008 | The system SHALL update `.sdd/manifest.json` with the new template version, tool version, and updated file hashes upon successful upgrade. | MUST | test |

## 5. Non-functional requirements
- **Determinism:** Given the same initial files, manifest, and templates, `sdd upgrade` SHALL produce byte-identical files, proposals, and manifest entries.
- **Fail-safe behaviour:** In case of I/O error or malformed block, abort before corrupting user files.
- **Confined I/O:** All file operations SHALL remain confined to the project root via `os.Root`.
- **Dependencies:** Standard library only (`D-007`).

## 6. Constraints and assumptions
- The project has been initialized with `sdd init` or possesses a valid `.sdd/manifest.json`.
- Template versions follow semantic versioning strings (e.g. `1.0.0`, `1.1.0`).

## 7. Open questions
(None. Resolved via Decision D-017.)

## 8. Acceptance criteria
- Given a project with untouched generated files and an upgraded template set, when `sdd upgrade` runs, then all untouched files are updated to new template content, originals are backed up, and manifest is updated.
- Given a project where the human edited `MISSION.md`, when `sdd upgrade` runs, then `MISSION.md` remains unchanged, `MISSION.md.sdd-new` is written, a unified diff is displayed, and the command exits 0.
- Given a project without `.sdd/manifest.json`, when `sdd upgrade` runs, then it refuses with an actionable error directing to `sdd init` or `sdd adopt`.
