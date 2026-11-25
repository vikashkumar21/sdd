package cli

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/vikashkumar21/sdd/internal/role"
	"github.com/vikashkumar21/sdd/internal/state"
)

// RepairPromptContext represents the structured context bundle for automated repair.
type RepairPromptContext struct {
	TaskID          string             `json:"taskId"`
	TaskDir         string             `json:"taskDir"`
	Typology        string             `json:"typology"`
	Attempt         int                `json:"attempt"`
	MaxLoops        int                `json:"maxLoops"`
	Halted          bool               `json:"halted"`
	HaltReason      string             `json:"haltReason,omitempty"`
	ModelRouting    *role.ModelRouting `json:"model_routing,omitempty"`
	ReproductionDoc string             `json:"reproductionDoc,omitempty"`
	RCADoc          string             `json:"rcaDoc,omitempty"`
	PlanDoc         string             `json:"planDoc,omitempty"`
	ValidationDoc   string             `json:"validationDoc,omitempty"`
	FeedbackRules   string             `json:"feedbackRules"`
	Decisions       []string           `json:"decisions"`
	Prompt          string             `json:"prompt"`
}

func runPromptRepair(args []string, env IO) int {
	fs := newFlagSet("prompt repair", env)
	dir := fs.String("dir", ".", "project root")
	jsonFlag := fs.Bool("json", false, "output machine-readable repair context bundle as JSON")
	maxLoops := fs.Int("max-loops", 3, "maximum automated repair iterations before circuit breaker halt (1-5)")

	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return parseExit(err)
	}

	if len(pos) != 1 {
		fprintln(env.Err, "usage: sdd prompt repair <T-xxx> [--max-loops <N>] [--dir <project>] [--json]")
		return 2
	}

	if *maxLoops < 1 || *maxLoops > 5 {
		fprintf(env.Err, "sdd prompt repair: --max-loops must be between 1 and 5 (got %d)\n", *maxLoops)
		return 2
	}

	rawTaskID := strings.ToUpper(strings.TrimSpace(pos[0]))
	normID, err := state.ValidateTaskID(rawTaskID)
	if err != nil {
		fprintf(env.Err, "sdd prompt repair: invalid task ID: %v\n", err)
		return 1
	}

	root, err := filepath.Abs(*dir)
	if err != nil {
		fprintf(env.Err, "sdd prompt repair: %v\n", err)
		return 1
	}

	taskDirRel, typology, err := findTaskDirectory(root, normID)
	if err != nil {
		fprintf(env.Err, "sdd prompt repair: %v\n", err)
		return 1
	}

	taskDirAbs := filepath.Join(root, filepath.FromSlash(taskDirRel))

	reproDoc := readTaskDoc(taskDirAbs, "reproduction.md")
	if reproDoc == "" {
		reproDoc = readTaskDoc(taskDirAbs, "requirements.md")
	}
	rcaDoc := readTaskDoc(taskDirAbs, "rca.md")
	planDoc := readTaskDoc(taskDirAbs, "plan.md")
	if planDoc == "" {
		planDoc = readTaskDoc(taskDirAbs, "patch.md")
	}
	valDoc := readTaskDoc(taskDirAbs, "validation.md")

	// Read and advance repair attempts in .sdd/state/<T-xxx>/state.json
	now := env.now()
	if now.IsZero() {
		now = time.Now()
	}

	stateMap, hasState, err := state.ReadState(root, normID)
	if err != nil {
		fprintf(env.Err, "sdd prompt repair: read state: %v\n", err)
		return 1
	}
	if !hasState || stateMap == nil {
		stateMap = make(map[string]any)
	}

	currentAttempt := 1
	if rawVal, ok := stateMap["repair_attempts"]; ok {
		switch v := rawVal.(type) {
		case float64:
			currentAttempt = int(v) + 1
		case int:
			currentAttempt = v + 1
		case int64:
			currentAttempt = int(v) + 1
		}
	}

	stateMap["repair_attempts"] = currentAttempt
	stateMap["max_repair_loops"] = *maxLoops
	stateMap["last_repair_prompt_at"] = now.UTC().Format(time.RFC3339)

	stateBytes, err := json.Marshal(stateMap)
	if err != nil {
		fprintf(env.Err, "sdd prompt repair: marshal state: %v\n", err)
		return 1
	}
	if _, err := state.SaveState(root, normID, stateBytes, now); err != nil {
		fprintf(env.Err, "sdd prompt repair: save state: %v\n", err)
		return 1
	}

	// Append event to journal
	eventPayload, _ := json.Marshal(map[string]any{
		"type":      "repair_prompt_generated",
		"task_id":   normID,
		"attempt":   currentAttempt,
		"max_loops": *maxLoops,
		"timestamp": now.UTC().Format(time.RFC3339),
	})
	if _, err := state.AppendEvent(root, normID, eventPayload); err != nil {
		fprintf(env.Err, "sdd prompt repair: warning: append event: %v\n", err)
	}

	halted := currentAttempt >= *maxLoops
	var haltReason string
	if halted {
		haltReason = fmt.Sprintf("circuit breaker ceiling reached: attempt %d of %d", currentAttempt, *maxLoops)
	}

	feedbackRules := extractActiveFeedbackRules(root)
	decisions := extractApprovedDecisions(root)

	roleSkill, _ := role.ResolveSkill(root, "implementer", nil)
	rolesCfg, _ := role.LoadRoles(root)
	repairRouting := role.ResolveRoleRouting(rolesCfg, "implementer")

	promptText := buildRepairPrompt(normID, taskDirRel, typology, currentAttempt, *maxLoops, halted, haltReason, reproDoc, rcaDoc, planDoc, valDoc, feedbackRules, decisions, roleSkill, repairRouting)

	ctx := RepairPromptContext{
		TaskID:          normID,
		TaskDir:         taskDirRel,
		Typology:        typology,
		Attempt:         currentAttempt,
		MaxLoops:        *maxLoops,
		Halted:          halted,
		HaltReason:      haltReason,
		ModelRouting:    &repairRouting,
		ReproductionDoc: reproDoc,
		RCADoc:          rcaDoc,
		PlanDoc:         planDoc,
		ValidationDoc:   valDoc,
		FeedbackRules:   feedbackRules,
		Decisions:       decisions,
		Prompt:          promptText,
	}

	if *jsonFlag {
		enc := json.NewEncoder(env.Out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(ctx); err != nil {
			fprintf(env.Err, "sdd prompt repair: json encode: %v\n", err)
			return 1
		}
		return 0
	}

	fprint(env.Out, promptText)
	return 0
}

func buildRepairPrompt(taskID, taskDir, typology string, attempt, maxLoops int, halted bool, haltReason, reproDoc, rcaDoc, planDoc, valDoc, feedbackRules string, decisions []string, roleSkill string, routing role.ModelRouting) string {
	var prefix strings.Builder

	prefix.WriteString("# Auto-Repair Handover Prompt\n\n")
	prefix.WriteString("You are acting as an **Auto-Repair Implementer** tasked with resolving a verified automated test or CI defect under strict gate isolation.\n")
	prefix.WriteString("Your mandate is to restore a passing verification state with minimum blast radius and zero unapproved dependencies.\n\n")

	prefix.WriteString("## SDD Anthem & Principles\n")
	prefix.WriteString("> **ALWAYS BE DETERMINISTIC. When a decision is needed, STOP and ask the human. Never guess. Never silently choose.**\n\n")

	prefix.WriteString(formatModelRoutingSection(routing, "implementer"))
	prefix.WriteString("\n")

	prefix.WriteString("## Mandatory Step 0: Minimal Failing Test Case\n")
	prefix.WriteString("> **MANDATORY PREREQUISITE:** Before applying any fixes or modifying production code:\n")
	prefix.WriteString("1. Author a minimal, standalone reproduction test case reproducing the reported CI defect.\n")
	prefix.WriteString("2. Execute the test on unpatched code to prove it fails deterministically.\n")
	prefix.WriteString("3. Record the exact execution command and failing output in the task reproduction doc.\n")
	prefix.WriteString("4. NEVER proceed to modify production code without proving failure reproduction.\n\n")

	if strings.TrimSpace(roleSkill) != "" {
		prefix.WriteString("## Implementer Role Skill Reference\n")
		prefix.WriteString("```markdown\n")
		prefix.WriteString(strings.TrimSpace(roleSkill))
		prefix.WriteString("\n```\n\n")
	}

	if len(decisions) > 0 {
		prefix.WriteString("## Approved Architectural Decisions (TECH_STACK.md)\n")
		for _, d := range decisions {
			fmt.Fprintf(&prefix, "- %s\n", d)
		}
		prefix.WriteString("\n")
	}

	if feedbackRules != "" {
		prefix.WriteString("## Active Human Feedback Rules (Binding)\n")
		prefix.WriteString("```markdown\n")
		prefix.WriteString(feedbackRules)
		prefix.WriteString("\n```\n\n")
	}

	prefix.WriteString("## Definition of Done for Auto-Repair\n")
	prefix.WriteString("- [ ] Step 0 minimal reproduction test authored and verified failing on unpatched code\n")
	prefix.WriteString("- [ ] Code repair strictly confined to planned files\n")
	prefix.WriteString("- [ ] Reproduction test passes cleanly on patched code\n")
	prefix.WriteString("- [ ] Full regression suite passes (`make verify`, race detector, determinism `-count=2`)\n")
	prefix.WriteString("- [ ] All findings resolved and recorded in `validation.md`\n")

	var suffix strings.Builder
	suffix.WriteString("## Dynamic Task Context & Circuit Breaker Status\n")
	if halted {
		fmt.Fprintf(&suffix, "> [!CAUTION]\n> **CIRCUIT BREAKER ACTIVATED: ATTEMPT %d OF %d CEILING REACHED**\n", attempt, maxLoops)
		fmt.Fprintf(&suffix, "> The automated repair loop has reached its configured ceiling of %d attempts.\n", maxLoops)
		suffix.WriteString("> In accordance with SDD safety and determinism rules, you MUST:\n")
		suffix.WriteString("> 1. **HALT IMMEDIATELY.** Cease all automated code edits.\n")
		fmt.Fprintf(&suffix, "> 2. Mark the task status in ROADMAP.md as `Blocked (%s)`.\n", haltReason)
		suffix.WriteString("> 3. Escalate the failure diagnostics, root cause findings, and reproduction outputs to the human for guidance.\n\n")
	} else {
		fmt.Fprintf(&suffix, "> [!IMPORTANT]\n> **AUTO-REPAIR LOOP: ATTEMPT %d OF %d**\n", attempt, maxLoops)
		fmt.Fprintf(&suffix, "> You are executing automated repair attempt %d of %d.\n", attempt, maxLoops)
		fmt.Fprintf(&suffix, "> If this attempt fails, the auto-repair loop will terminate at attempt %d.\n\n", maxLoops)
	}

	fmt.Fprintf(&suffix, "- **Task ID:** %s\n", taskID)
	fmt.Fprintf(&suffix, "- **Task Directory:** %s\n", taskDir)
	fmt.Fprintf(&suffix, "- **Typology:** %s\n\n", typology)

	if reproDoc != "" {
		suffix.WriteString("## Failure Diagnostics & Reproduction\n")
		suffix.WriteString("```markdown\n")
		suffix.WriteString(reproDoc)
		suffix.WriteString("\n```\n\n")
	}

	if rcaDoc != "" {
		suffix.WriteString("## Root Cause Analysis (RCA)\n")
		suffix.WriteString("```markdown\n")
		suffix.WriteString(rcaDoc)
		suffix.WriteString("\n```\n\n")
	}

	if planDoc != "" {
		suffix.WriteString("## Approved Plan & Boundaries\n")
		suffix.WriteString("```markdown\n")
		suffix.WriteString(planDoc)
		suffix.WriteString("\n```\n\n")
	}

	if valDoc != "" {
		suffix.WriteString("## Validation State\n")
		suffix.WriteString("```markdown\n")
		suffix.WriteString(valDoc)
		suffix.WriteString("\n```\n\n")
	}

	parts := PromptParts{
		StaticPrefix:  prefix.String(),
		DynamicSuffix: suffix.String(),
	}
	return parts.FullPrompt()
}
