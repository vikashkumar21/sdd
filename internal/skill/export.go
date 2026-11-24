package skill

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vikashkumar21/sdd/internal/fsutil"
)

// CanonicalSDDProtocolSkill is the reference agentskills.io definition of the SDD protocol.
const CanonicalSDDProtocolSkill = `---
name: sdd-protocol
description: Spec-Driven Development (SDD) contract, deterministic gating protocol (GATE 1, GATE 2), decision protocols, and task execution rules for AI coding agents.
---

# Spec-Driven Development (SDD) Protocol

> **ANTHEM: ALWAYS BE DETERMINISTIC. When a decision is needed, STOP and ask the human. Never guess. Never silently choose.**

## Overview & Triggers
Activate this skill whenever:
- Taking over an existing repository or bootstrapping a greenfield project.
- Specifying, planning, implementing, or reviewing any task across work typologies (features, bugfixes, hotfixes, spikes).
- A human requires strict gating, deterministic verification, and non-negotiable review checkpoints before modifying code or living documents.

## The SDD Anthem & Hard Rules
1. **No Code Before GATE 1:** Only task documentation in tasks/<typology>/T-xxx-<slug>/ (requirements.md, plan.md, validation.md, review.md) may be written during Phase A.
2. **Never Commit or Push:** The human always reviews the diff and runs git commit. Agents only suggest commit messages.
3. **Fail Safe:** When input, state, or intent is unknown or ambiguous, choose the most restrictive outcome or stop and ask.
4. **Deterministic by Construction:** Same input gives same output. No hidden clocks, randomness, or map-iteration order in decision logic.
5. **Standard Library Only:** Zero external third-party dependencies without an approved human decision.
6. **No Scope Creep:** Unplanned findings go to ROADMAP.md Parking lot, never into the current task.
7. **Rule F-001 (Resolve All Findings):** Worker agents must resolve all findings (Blockers, Warnings, Minors) before presenting to the human at GATE 2.

## The 4-Phase Workflow
Every task follows this sequence without skipping or reordering:

| Phase | What the agent does | Gate |
|---|---|---|
| A. Specify | Create tasks/<typology>/T-xxx-<slug>/ and draft requirements.md, plan.md, validation.md, review.md. Invoke independent Plan Review Subagent (sdd prompt plan-review). Remediate findings until 0 Blockers. | GATE 1 (APPROVED T-xxx) |
| B. Implement | Execute plan.md step by step. Touch only planned files. | none |
| C.1 Verify | Run all checks in validation.md and record real commands/results. | none |
| C.2 Review Triad | Spawn 3 parallel subagents: Spec Auditor, AppSec Specialist, Pattern Auditor (sdd prompt review --lens <lens>). Remediate findings in unified fix pass (max 3 loops). | 0 Blockers across all lenses |
| D. Close | Update living docs, set status to Awaiting review, present diff and commit message. | GATE 2 (COMMITTED T-xxx) |

## Decision Protocol (D-xxx)
When a choice is not settled in TECH_STACK.md or an approved plan, STOP and emit:
` + "```" + `
DECISION NEEDED: D-xxx — <title>
Context:        <why this must be decided now>
Options:        A) …   B) …   C) …
Recommendation: <option + reason>
Blocks:         <task or step>
Reply with:     DECIDE D-xxx: <A|B|C|other>
` + "```" + `

## Progressive Disclosure & SDK Agent Ingestion
- Programmatic SDK agents (Antigravity Python SDK, Anthropic Agent SDK, LangGraph) can load this skill directory or inject this contract into their system prompts.
- All actions taken by SDK agents must respect GATE 1 (waiting for human approval string) before executing code-modifying tools.
`

// CanonicalCodeReviewSkill is the reference agentskills.io definition of the SDD code review standard.
const CanonicalCodeReviewSkill = `---
name: code-review
description: Deterministic code review rubric, progressive context ingestion, 3-tier risk grading (Blocker, Warning, Minor), and 3-subagent parallel review triad standards (Spec Auditor, AppSec Specialist, Pattern Auditor) for AI coding agents.
---

# SDD Code Review Standard

> **ANTHEM: ALWAYS BE DETERMINISTIC. When a decision is needed, STOP and ask the human. Never guess. Never silently choose.**

## Overview & Triggers
Activate this skill whenever:
- Performing code review on a feature branch, staged change set, or pull request.
- Verifying code against approved specifications before Phase D / GATE 2 commit.
- Inspecting code diffs for spec alignment, architectural drift, security hazards, or unhandled errors.

## The Ensemble Review Triad (3-Subagent Parallel Review Triad)
Every code review in Phase C.2 deploys three independent subagents in parallel, each auditing through exactly one specialized lens:

1. **🔎 Spec Auditor Lens (Scope & Correctness):**
   - **Traceability:** Every requirement (R-xxx) in requirements.md maps to concrete implementation and test evidence in validation.md.
   - **Boundary Enforcement:** Touch ONLY files listed in plan.md Section 5. Zero unapproved file modifications or scope creep.
   - **Negative Coverage:** Negative, boundary, and failure cases are explicitly tested and verified.
   - **Standard Library Purity:** Zero unapproved third-party dependencies (D-007).

2. **🛡️ AppSec Specialist Lens (Security & Safety):**
   - **Threat Surface & Input Validation:** Bound sizes on all external inputs; parse strictly; treat untrusted input as hostile.
   - **Confinement & Traversal:** Verify filesystem confinement via os.Root or clean paths; reject path traversal and escape sequences.
   - **Secrets & Privacy Hygiene:** Zero credentials, tokens, connection strings, or personal data in logs, tests, docs, or code. Synthetic fixtures only.
   - **Command Safety:** Never construct shell commands via string concatenation; maintain separate arguments.
   - **Fail Safe:** Ambiguous or unknown state fails to the most restrictive outcome.

3. **📐 Pattern & Quality Auditor Lens (Architecture & Standards):**
   - **Decision Conformance:** Respects all approved decisions in TECH_STACK.md (D-xxx).
   - **Determinism:** Inject clocks, random sources, and ID generators; avoid map iteration order dependencies.
   - **Code Quality:** Idiomatic Go (Effective Go, Go Code Review Comments); context.Context first for blocking/IO; %w error wrapping; clean linting (make lint).
   - **Concurrency Safety:** Verify goroutine lifecycles, race detector compliance, and zero data races.
   - **Continuous Learning:** Active feedback rules in SKILLS.md §8 (F-xxx) complied with; recurring patterns captured.

## Ephemeral Tabula-Rasa Reviewer Lifecycle & Zero-Negotiation Standard
To eliminate Repair-Loop Justification Bleed where developers sway reviewers with conversational rationales or excuses, SDD mandates strict reviewer session ephemerality and zero-negotiation loop isolation (D-049):
- Ephemeral Subagent Spawning: Reviewer subagent conversations MUST NOT be reused across remediation loops. Reusing conversations via send_message or carrying conversational transcripts into subsequent loops is strictly forbidden. For every review iteration, dispatch fresh, independent reviewer subagents (invoke_subagent).
- Zero-Negotiation Loop Isolation: When re-staging review bundles for Loop 2 and 3 via sdd prompt review <TASK_ID> --loop <N>, the bundle contains ONLY candidate diff.patch, spec.md, ledger.md, and objective findings_to_verify.md. Developer prose, chat transcripts, excuses, or negotiation dialogues are strictly scrubbed.

## Batch Remediation & Conflict Resolution
- **Single Remediator:** Review subagents are strictly read-only auditors; only the primary worker agent modifies code.
- **Batched Remediation:** When multiple lenses report changes, the primary agent consolidates all findings into a single fix pass, consuming 1 loop in the 3-loop circuit breaker.
- **Conflict Resolution:** Security, safety, and determinism take precedence over convenience. If recommendations fundamentally conflict, the agent halts and issues DECISION NEEDED: D-xxx to the human.

## 3-Tier Risk & Finding Rubric

| Severity | Definition | Action Required |
|---|---|---|
| 🔴 **Blocker** | Spec violation, security hazard, test gap, plan deviation, unapproved dependency. | MUST be resolved before GATE 2 commit. |
| 🟡 **Warning** | Sub-optimal pattern, edge case risk, missing doc or comment. | MUST be resolved or explicitly justified before GATE 2 (Rule F-001). |
| 🟢 **Minor** | Style, naming refinement, or non-critical micro-optimization. | MUST be resolved or explicitly justified before GATE 2 (Rule F-001). |

## Rule F-001 Finding Resolution Mandate
- Worker agents MUST resolve all findings (🔴 Blockers, 🟡 Warnings, and 🟢 Minors) reported during review before presenting to the human at GATE 2.
- Any finding not remediated must have an explicit written justification with human approval recorded in review.md.
- review.md can only achieve Approved sign-off when 0 Blockers, 0 unresolved Warnings, and 0 unresolved Minors remain.

## 3-Loop Circuit Breaker
- Maximum 3 review-fix iterations (1 initial audit + up to 2 fix attempts) between worker agent and reviewer triad.
- If Blockers persist after attempt 3, halt immediately, mark the task Blocked, and escalate to the human.

## Review Output Format
Record all findings in tasks/<typology>/T-xxx-<slug>/review.md following the standard template. If any 🔴 Blocker remains, or if any 🟡 Warning or 🟢 Minor remains unresolved without explicit justification, the task CANNOT proceed to GATE 2.
`

// CanonicalTaskCoordinatorSkill is the reference agentskills.io definition of the SDD Task Coordinator skill.
const CanonicalTaskCoordinatorSkill = `---
name: task-coordinator
description: Orchestrate multi-agent task execution, concurrency deconfliction, independent subagent reviews, and pre-merge qualification in SDD.
---

# SDD Task Coordinator Standard

> **ANTHEM: ALWAYS BE DETERMINISTIC. When a decision is needed, STOP and ask the human. Never guess. Never silently choose.**

## Overview & Triggers
Activate this skill whenever:
- Operating as an orchestrator or coordinator agent supervising one or more worker agents across SDD phases.
- Brainstorming initiatives, assessing viability, and decomposing large epics into small, reviewable task slices.
- Provisioning isolated git worktrees, tracking task state in .sdd/state/, and analyzing static plan overlap.
- Handing off completed implementation to the Review Triad before GATE 2.
- Running pre-merge qualification, blast-radius regression sweeps, or presenting merges for human review.

## Persona & Seniority: Staff / Principal Orchestrator
The Task Coordinator acts as a Staff Orchestrator and Technical Project Lead. It does NOT implement feature code directly. Instead, it maintains architectural alignment, enforces strict boundaries, supervises specialized worker agents, and ensures human oversight across all irreversible operations.

## Core Responsibilities & Operating Protocol

### 1. Task Brainstorming, Viability & Small-Slice Decomposition
- **Feasibility & Usability:** Evaluate task scope against project principles in MISSION.md and capacity in ROADMAP.md (12 h/week capacity, S/M/L sizing).
- **PR Size Guardrail:** Decompose large initiatives so that each resulting task touches a bounded number of files (typically <= 10–15 files). Never allow an agent to bundle massive, unreviewable diffs into a single task.
- **Dependency Sequencing:** Ensure prerequisites are completed before downstream tasks are started.

### 2. State Persistence, Event Journaling & Crash Recovery
- **Deterministic State Store:** Every task lifecycle transition is journaled using sdd task state:
  - On task start: record phase transition event (sdd task state append-event <task-id> --event '{"kind":"PHASE_TRANSITION","to_phase":"Specify"}').
  - On state updates: save snapshot atomically (sdd task state save <task-id> --state '<json>').
  - On crash or restart: read snapshot (sdd task state show <task-id> --json) to resume from the latest verified checkpoint without re-running completed work.
- **Clean Teardown:** After human Gate 2 commit, purge task state if requested (sdd task clean <task-id> --purge-state).

### 3. Worktree Concurrency & Static Plan Deconfliction
- **Isolation:** Provision isolated git worktrees for workers via sdd task start <task-id>.
- **Collision Detection:** Before dispatching concurrent workers, run sdd task deconflict to analyze target file collisions in plan.md Section 5 across in-flight tasks. If collisions are detected, sequence the tasks sequentially rather than in parallel.

### 4. Subagent Supervision & Two-Tier Agent Roles (Hierarchy)
- **Worker Confinement:** Workers operate strictly inside their isolated git worktree / task branch (task/<T-xxx>). Autonomous commits are permitted on the isolated task branch; workers are strictly forbidden from writing to, rebasing, or merging into main. Workers cannot self-approve their own review.md.
- **Role Dispatch:** Dispatch specialized subagents with their dedicated role skill:
  - Phase A: Architect / Planner (architect) drafts; Pre-GATE 1 Plan Review Subagent (plan-reviewer) audits.
  - Phase B: Implementer (implementer) executes in isolated branch.
  - Phase C.2: 3-Subagent Parallel Review Triad (spec-auditor, appsec-specialist, pattern-auditor) audits in parallel.
- **Ephemeral Tabula-Rasa Reviewer Lifecycle & Zero-Negotiation Protocol:**
  - Single-Loop Reviewer Ephemerality: Reviewer subagent sessions MUST NOT be reused across remediation loops. Reusing conversations via send_message across loops is strictly forbidden (D-049).
  - Fresh Dispatch Per Loop: Every remediation loop (Loop 2 and Loop 3) requires dispatching brand new, ephemeral reviewer subagents (invoke_subagent) operating in isolated tabula-rasa contexts with zero conversational history.
  - Zero-Negotiation Loop Isolation: Remediation prompts and review bundles (.sdd/review/<TASK_ID>/code/) MUST NOT contain developer prose, excuses, rationales, apologies, or negotiation transcripts. Stage candidate diffs alongside pure, objective finding checklists (findings_to_verify.md) generated via sdd prompt review <TASK_ID> --loop <N>.
- **Rule F-001 Compliance:** The Task Coordinator ensures all findings (Blockers, Warnings, Minors) are resolved in a unified batch remediation pass before presenting to the human.
- **3-Loop Circuit Breaker:** At most 3 review-fix iterations. If Blockers persist after attempt 3, halt, mark the task Blocked, and escalate to the human.

### 5. Pre-Merge Qualification & Gate 2 Presentation
Before any merge into main:
1. Verify task status is Awaiting review.
2. Verify independent review sign-off: review.md must have status Approved with 0 Blockers, and all findings resolved or justified per Rule F-001.
3. Simulate git merge in an isolated throwaway worktree: sdd task merge --simulate <T-xxx> (or sdd task merge <task-id> --dry-run).
4. Run full blast-radius regression test suite on the simulated merge commit.
5. Present merge qualification summary to the human for approval (APPROVED MERGE T-xxx / COMMITTED T-xxx).
`

// ExportOptions holds parameters for exporting canonical SDD skills.
type ExportOptions struct {
	ProjectRoot string
	TargetDir   string
	Global      bool
	Force       bool
	Home        func() string
	Clock       func() time.Time
}

// ExportResult contains the outcome of skill export.
type ExportResult struct {
	Name      string // Skill name (e.g. sdd-protocol, code-review)
	Path      string // Slash-separated path to created SKILL.md
	BackupDir string // Relative backup dir if overwritten
}

// ExportProtocolSkill exports the canonical sdd-protocol skill to the target destination.
func ExportProtocolSkill(opts ExportOptions) (ExportResult, error) {
	return ExportSkill("sdd-protocol", opts)
}

// ExportSkill exports a named canonical skill (sdd-protocol or code-review).
func ExportSkill(skillName string, opts ExportOptions) (ExportResult, error) {
	var res ExportResult
	res.Name = skillName

	var content string
	switch skillName {
	case "sdd-protocol":
		content = CanonicalSDDProtocolSkill
	case "code-review":
		content = CanonicalCodeReviewSkill
	case "task-coordinator":
		content = CanonicalTaskCoordinatorSkill
	default:
		return res, fmt.Errorf("unknown canonical skill %q (expected sdd-protocol, code-review, task-coordinator, or all)", skillName)
	}

	var targetBase string
	if opts.Global {
		home := ""
		if opts.Home != nil {
			home = opts.Home()
		} else {
			home, _ = os.UserHomeDir()
		}
		if strings.TrimSpace(home) == "" {
			return res, errors.New("cannot export globally: home directory not determined")
		}

		geminiDir := filepath.Join(home, ".gemini", "antigravity")
		claudeDir := filepath.Join(home, ".claude")
		if stat, err := os.Stat(geminiDir); err == nil && stat.IsDir() {
			targetBase = filepath.Join(geminiDir, "skills")
		} else if stat, err := os.Stat(claudeDir); err == nil && stat.IsDir() {
			targetBase = filepath.Join(claudeDir, "skills")
		} else {
			targetBase = filepath.Join(home, ".sdd", "skills")
		}
	} else if opts.TargetDir != "" {
		targetBase = opts.TargetDir
		if !filepath.IsAbs(targetBase) && opts.ProjectRoot != "" {
			targetBase = filepath.Join(opts.ProjectRoot, targetBase)
		}
	} else {
		if opts.ProjectRoot == "" {
			return res, errors.New("project root required when target directory is not specified")
		}
		targetBase = filepath.Join(opts.ProjectRoot, "skills")
	}

	destDir := filepath.Join(targetBase, skillName)
	destFile := filepath.Join(destDir, "SKILL.md")

	if existingBytes, readErr := os.ReadFile(destFile); readErr == nil {
		if !opts.Force {
			return res, fmt.Errorf("skill %q already exists at %s (use --force to overwrite)", skillName, filepath.ToSlash(destFile))
		}
		now := time.Now().UTC()
		if opts.Clock != nil {
			now = opts.Clock().UTC()
		}
		backupTimestamp := now.Format("20060102-150405")

		var backupPath string
		if opts.ProjectRoot != "" && strings.HasPrefix(destFile, opts.ProjectRoot) {
			backupRel := filepath.ToSlash(filepath.Join(".sdd", "backup", backupTimestamp))
			relDest, _ := filepath.Rel(opts.ProjectRoot, destFile)
			backupPath = filepath.Join(opts.ProjectRoot, filepath.FromSlash(backupRel), relDest)
			res.BackupDir = backupRel
		} else {
			backupPath = filepath.Join(targetBase, ".backup", backupTimestamp, skillName, "SKILL.md")
			res.BackupDir = filepath.ToSlash(filepath.Join(targetBase, ".backup", backupTimestamp))
		}

		if err := fsutil.WriteAtomic(backupPath, existingBytes, 0o644); err != nil {
			return res, fmt.Errorf("backup existing skill: %w", err)
		}
	}

	parsed, parseErr := Parse([]byte(content))
	if parseErr != nil {
		return res, fmt.Errorf("canonical skill frontmatter invalid: %w", parseErr)
	}
	validationErrors := Validate(parsed, destDir)
	if len(validationErrors) > 0 {
		return res, fmt.Errorf("canonical skill validation failed: %s", validationErrors[0].Message)
	}

	if err := fsutil.WriteAtomic(destFile, []byte(content), 0o644); err != nil {
		return res, fmt.Errorf("write skill file: %w", err)
	}

	res.Path = filepath.ToSlash(destFile)
	return res, nil
}

// ExportSkills exports one or all canonical skills according to skillName ("all", "sdd-protocol", "code-review").
func ExportSkills(skillName string, opts ExportOptions) ([]ExportResult, error) {
	target := strings.TrimSpace(skillName)
	if target == "" || target == "all" {
		names := []string{"sdd-protocol", "code-review", "task-coordinator"}
		var results []ExportResult
		for _, name := range names {
			res, err := ExportSkill(name, opts)
			if err != nil {
				return results, err
			}
			results = append(results, res)
		}
		return results, nil
	}

	res, err := ExportSkill(target, opts)
	if err != nil {
		return nil, err
	}
	return []ExportResult{res}, nil
}
