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

// ExportOptions holds parameters for exporting the canonical SDD skill.
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
	Path      string // Slash-separated path to created SKILL.md
	BackupDir string // Relative backup dir if overwritten
}

// ExportProtocolSkill exports the canonical sdd-protocol skill to the target destination.
func ExportProtocolSkill(opts ExportOptions) (ExportResult, error) {
	var res ExportResult

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

	destDir := filepath.Join(targetBase, "sdd-protocol")
	destFile := filepath.Join(destDir, "SKILL.md")

	if existingBytes, readErr := os.ReadFile(destFile); readErr == nil {
		if !opts.Force {
			return res, fmt.Errorf("skill \"sdd-protocol\" already exists at %s (use --force to overwrite)", filepath.ToSlash(destFile))
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
			backupPath = filepath.Join(targetBase, ".backup", backupTimestamp, "sdd-protocol", "SKILL.md")
			res.BackupDir = filepath.ToSlash(filepath.Join(targetBase, ".backup", backupTimestamp))
		}

		if err := fsutil.WriteAtomic(backupPath, existingBytes, 0o644); err != nil {
			return res, fmt.Errorf("backup existing skill: %w", err)
		}
	}

	parsed, parseErr := Parse([]byte(CanonicalSDDProtocolSkill))
	if parseErr != nil {
		return res, fmt.Errorf("canonical skill frontmatter invalid: %w", parseErr)
	}
	validationErrors := Validate(parsed, destDir)
	if len(validationErrors) > 0 {
		return res, fmt.Errorf("canonical skill validation failed: %s", validationErrors[0].Message)
	}

	if err := fsutil.WriteAtomic(destFile, []byte(CanonicalSDDProtocolSkill), 0o644); err != nil {
		return res, fmt.Errorf("write skill file: %w", err)
	}

	res.Path = filepath.ToSlash(destFile)
	return res, nil
}
