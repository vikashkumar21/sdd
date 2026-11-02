# Plan — T-016 Handover prompt enrichment for staged context

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved

## 1. Approach summary
Enhance `sdd prompt` handover generation to discover staged context artifacts from `.sdd/context/manifest.json` and render clear synthesis guidance for AI agents:
1. In `internal/cli/commands.go`:
   - Extend `promptData` with `StagedContext []contextView` and `TaskContext []contextView`.
   - In `runPrompt`, read `.sdd/context/manifest.json` via `context.ReadManifest(root)`.
   - If present, populate `StagedContext` and isolate any `TaskContext` matching the focus task ID.
2. In `internal/templates/data/base/prompt.md.tmpl`:
   - Add conditional section `{{if .StagedContext}}...{{end}}`.
   - List staged artifacts with ID, source, origin, path, format, size.
   - Provide concrete extraction instructions guiding the agent on how to map findings into `MISSION.md`, `TECH_STACK.md`, `ROADMAP.md`, `SKILLS.md`, or task requirements under Gate 1.
3. In `internal/cli/commands_test.go`:
   - Add test cases verifying prompt rendering when context is present, when task-specific context exists, and when context is absent.
4. Verify:
   - `make check`, `go test -race ./...`, `GOOS=windows go vet ./...`.

## 2. Decisions (new or affected)
- D-007 (Standard library only)
- D-015 (Approved: Go 1.27 toolchain)
- D-018 (Approved: Ingestion architecture; agent extracts under Gate 1)
- D-020 (Approved: Text & file context staged for agent reading)

## 3. Design
- **`internal/cli/commands.go`**:
  ```go
  type contextView struct {
      ID     string
      Source string
      Origin string
      Path   string
      Task   string
      Format string
      Bytes  int64
  }

  type promptData struct {
      Project, Root, AgentName, AgentFile, Date string
      AgentFiles                                []string
      OpenTasks                                 []taskView
      FocusTask                                 string
      StagedContext                             []contextView
      TaskContext                               []contextView
  }
  ```
- **`internal/templates/data/base/prompt.md.tmpl`**:
  ```markdown
  {{if .StagedContext}}
  ## Staged external context (.sdd/context/)

  The following external materials were staged for this project:
  | ID | Format | Origin / Source | Staged Path | Task |
  |---|---|---|---|---|
  {{range .StagedContext}}| {{.ID}} | {{.Format}} | {{.Origin}} ({{.Source}}) | {{.Path}} | {{if .Task}}{{.Task}}{{else}}(general){{end}} |
  {{end}}

  **Extraction and synthesis instructions:**
  1. Read the staged context files above using read-only tools.
  2. Map the extracted findings into the appropriate living documents or task specifications:
     - Vision, user personas, in/out-of-scope constraints -> propose in `MISSION.md`
     - Architecture, technology choices, database/cloud constraints -> propose in `TECH_STACK.md` (as `D-xxx` decisions or verified facts)
     - Deliverables, work breakdown, milestones -> propose in `ROADMAP.md` (`T-xxx` rows)
     - Team conventions, dos/don'ts, human feedback -> propose in `SKILLS.md`
     - Task-specific acceptance criteria -> propose in `features/T-xxx-<slug>/requirements.md`
  3. **GATE 1 IS STRICT:** Propose all changes as draft documents first. The human must review and approve before any code is written or living docs are modified.
  {{end}}
  ```

## 4. Implementation steps

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Extend `promptData` and update `runPrompt` to read context manifest | `internal/cli/commands.go` | Unit test in `commands_test.go` |
| 2 | Update `prompt.md.tmpl` with staged context section and synthesis guidance | `internal/templates/data/base/prompt.md.tmpl` | Template render test in `commands_test.go` |
| 3 | Add CLI integration tests for `sdd prompt` with and without context | `internal/cli/commands_test.go` | `go test -v -run TestPrompt ./internal/cli` |
| 4 | Run linter, race detector, Windows cross-vet, and consistency check | Project root | `make check`, `go test -race ./...`, `go run ./cmd/sdd check` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/cli/commands.go` | modify | Load context manifest into `promptData` |
| `internal/templates/data/base/prompt.md.tmpl` | modify | Add staged context section and semantic extraction instructions |
| `internal/cli/commands_test.go` | modify | Add test cases for prompt context enrichment |
| `features/T-016-prompt-context-enrichment/requirements.md` | modify/review | Requirements specification |
| `features/T-016-prompt-context-enrichment/plan.md` | modify/review | Implementation plan |
| `features/T-016-prompt-context-enrichment/validation.md` | modify/review | Validation evidence |

## 6. Dependencies
None. Standard library only (D-007).

## 7. Living docs to update

| File | Change |
|---|---|
| `ROADMAP.md` | Mark T-016 `Awaiting review` then `Done`; update notes |
| `TECH_STACK.md` | Note prompt context enrichment in architecture section |
| `README.md` | Note prompt context synthesis in prompt command documentation |

## 8. Risks, mitigations, rollback
- **Risk:** Existing custom prompt templates break if fields are missing.
  - *Mitigation:* `StagedContext` is an optional slice; missing fields in older templates are no-ops.
- **Risk:** Large numbers of staged files clutter the prompt.
  - *Mitigation:* Tabular summary displays one compact line per artifact; agent reads the actual files on disk via path references.
- **Rollback:** `git restore .` reverts to previous prompt template and command.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
None active yet.

## 10. Approval record
Approved by: Vikash (human)   Date: 2026-11-02   Phrase received: `APPROVED T-016`
