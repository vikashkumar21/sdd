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
- Specifying, planning, or implementing any feature, bug fix, refactor, or architectural change.
- A human requires strict gating, deterministic verification, and non-negotiable review checkpoints before modifying code or living documents.

## The SDD Anthem & Hard Rules
1. **No Code Before GATE 1:** Only task documentation in features/T-xxx-<slug>/ may be written during Phase A.
2. **Never Commit or Push:** The human always reviews the diff and runs git commit. Agents only suggest commit messages.
3. **Fail Safe:** When input, state, or intent is unknown or ambiguous, choose the most restrictive outcome or stop and ask.
4. **Deterministic by Construction:** Same input gives same output. No hidden clocks, randomness, or map-iteration order in decision logic.
5. **Standard Library Only:** Zero external third-party dependencies without an approved human decision.
6. **No Scope Creep:** Unplanned findings go to ROADMAP.md Parking lot, never into the current task.

## The 4-Phase Workflow
Every task follows this sequence without skipping or reordering:

| Phase | What the agent does | Gate |
|---|---|---|
| A. Specify | Create features/T-xxx-<slug>/ and draft requirements.md, plan.md, validation.md. | GATE 1 (APPROVED T-xxx) |
| B. Implement | Execute plan.md step by step. Touch only planned files. | none |
| C. Verify | Run all checks in validation.md and record real commands/results. | none |
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
description: Deterministic code review rubric, progressive context ingestion, 3-tier risk grading (Blocker, Warning, Minor), and task review artifact guidelines for AI coding agents.
---

# SDD Code Review Standard

> **ANTHEM: ALWAYS BE DETERMINISTIC. When a decision is needed, STOP and ask the human. Never guess. Never silently choose.**

## Overview & Triggers
Activate this skill whenever:
- Performing code review on a feature branch, staged change set, or pull request.
- Verifying code against approved specifications before Phase D / GATE 2 commit.
- Inspecting code diffs for spec alignment, architectural drift, security hazards, or unhandled errors.

## The 4 Review Pillars
Every code review evaluates the changes against four mandatory pillars:

1. **Correctness & Spec Alignment:**
   - Every requirement (R-xxx) in requirements.md maps to concrete implementation and test evidence in validation.md.
   - Touch ONLY files listed in plan.md. Zero scope creep.
   - Negative and failure cases are explicitly tested.

2. **Architectural & Decision Conformance:**
   - Respects all approved decisions in TECH_STACK.md (D-xxx).
   - Zero unapproved third-party dependencies (D-007 standard library purity).
   - Complies with active human feedback rules in SKILLS.md §8 (F-xxx).
   - Deterministic by construction: inject clocks, random sources, and avoid map iteration order dependencies.

3. **Security & Quality Hygiene:**
   - No credentials, tokens, connection strings, or personal data in logs, tests, or code.
   - Bound sizes on all external inputs; fail safe on ambiguity.
   - Clean linting (make lint) and formatting (gofmt -s).

4. **Continuous Learning & Feedback Synthesis:**
   - If a finding reveals a recurring or general pattern, formulate an F-xxx rule for SKILLS.md §8 or scaffold a new skill via sdd skill new.

## 3-Tier Risk & Finding Rubric

| Severity | Definition | Action Required |
|---|---|---|
| 🔴 **Blocker** | Spec violation, security hazard, test gap, plan deviation, unapproved dependency. | MUST be resolved before GATE 2 commit. |
| 🟡 **Warning** | Sub-optimal pattern, edge case risk, missing doc or comment. | Developer attention suggested; address or explicitly accept. |
| 🟢 **Minor** | Style, naming refinement, or non-critical micro-optimization. | Informational suggestion; non-blocking. |

## Review Output Format
Record all findings in features/T-xxx-<slug>/review.md following the standard template. If any 🔴 Blocker remains, the review outcome is "Changes requested" and the task CANNOT proceed to GATE 2.
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
- Operating as an orchestrator or coordinator agent supervising one or more worker agents.
- Managing concurrent tasks across isolated git worktrees or branches.
- Handing off completed implementation to an independent subagent reviewer before GATE 2.
- Running pre-merge qualification, blast-radius regression checks, or presenting merges for human review.

## Two-Tier Agent Roles & Boundaries
1. **Worker Agents:**
   - Operate strictly inside their isolated git worktree / task branch (task/<T-xxx>).
   - Autonomous commits are permitted on the isolated task branch.
   - Strictly forbidden from writing to, rebasing, or merging into main.
   - Cannot self-approve their own review.md.
2. **Task Coordinator Agent:**
   - Supervises task provisioning, scheduling, and deconfliction (sdd task start, sdd task deconflict).
   - Dispatches independent subagent reviewers using turnkey review contexts (sdd prompt review).
   - Executes pre-merge qualification simulations and regression sweeps (sdd task merge).
   - Presents qualified merges and suggested commit messages to the human for GATE 2 sign-off (APPROVED MERGE T-xxx / COMMITTED T-xxx).

## Phase C.2 Independent Subagent Audit Protocol
- When a worker agent completes Phase C.1 validation checks, an independent subagent reviewer MUST be spawned.
- The handover prompt is compiled using sdd prompt review <T-xxx> (or --json for programmatic agent harnesses).
- Reviewer audits:
  1. Correctness & spec traceability (requirements.md to validation.md).
  2. Plan boundary compliance (no file touched outside plan.md).
  3. Security & quality hygiene (no leaked secrets, stdlib purity, lint clean).
  4. Active feedback rules in SKILLS.md §8.
- **3-Loop Circuit Breaker:**
  - Maximum 3 iterations (1 initial review + up to 2 fix attempts).
  - If Blockers persist after attempt 3, the Task Coordinator halts, marks the task Blocked, and escalates directly to the human.

## Pre-Merge Qualification Workflow
Before any merge into main:
1. Check task status: must be Awaiting review.
2. Check independent review sign-off: review.md must have status Approved with 0 Blockers.
3. Simulate git merge in an isolated throwaway worktree: sdd task merge --simulate <T-xxx>.
4. Run full blast-radius regression test suite on the simulated merge commit.
5. Present merge qualification summary to the human for approval.
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
