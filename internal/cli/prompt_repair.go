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
	var b strings.Builder

	fmt.Fprintf(&b, "# Auto-Repair Handover Prompt — %s (%s)\n\n", taskID, taskDir)

	if halted {
		fmt.Fprintf(&b, "> [!CAUTION]\n> **CIRCUIT BREAKER ACTIVATED: ATTEMPT %d OF %d CEILING REACHED**\n", attempt, maxLoops)
		fmt.Fprintf(&b, "> The automated repair loop has reached its configured ceiling of %d attempts.\n", maxLoops)
		b.WriteString("> In accordance with SDD safety and determinism rules, you MUST:\n")
		b.WriteString("> 1. **HALT IMMEDIATELY.** Cease all automated code edits.\n")
		fmt.Fprintf(&b, "> 2. Mark the task status in ROADMAP.md as `Blocked (%s)`.\n", haltReason)
		b.WriteString("> 3. Escalate the failure diagnostics, root cause findings, and reproduction outputs to the human for guidance.\n\n")
	} else {
		fmt.Fprintf(&b, "> [!IMPORTANT]\n> **AUTO-REPAIR LOOP: ATTEMPT %d OF %d**\n", attempt, maxLoops)
		fmt.Fprintf(&b, "> You are executing automated repair attempt %d of %d.\n", attempt, maxLoops)
		fmt.Fprintf(&b, "> If this attempt fails, the auto-repair loop will terminate at attempt %d.\n\n", maxLoops)
	}

	b.WriteString("## Model Routing Guidance\n")
	b.WriteString("- **Role:** implementer\n")
	fmt.Fprintf(&b, "- **Model Tier:** %s\n", routing.Tier)
	fmt.Fprintf(&b, "- **Reasoning Effort:** %s\n", routing.ReasoningEffort)
	fmt.Fprintf(&b, "- **Recommended Family:** %s\n\n", routing.RecommendedFamily)

	b.WriteString("## 1. Mandatory Step 0: Minimal Failing Test Case\n")
	b.WriteString("> **MANDATORY PREREQUISITE:** Before applying any fixes or modifying production code:\n")
	b.WriteString("1. Author a minimal, standalone reproduction test case reproducing the reported CI defect.\n")
	b.WriteString("2. Execute the test on unpatched code to prove it fails deterministically.\n")
	b.WriteString("3. Record the exact execution command and failing output in the task reproduction doc.\n")
	b.WriteString("4. NEVER proceed to modify production code without proving failure reproduction.\n\n")

	if reproDoc != "" {
		b.WriteString("## 2. Failure Diagnostics & Reproduction\n")
		b.WriteString("```markdown\n")
		b.WriteString(reproDoc)
		b.WriteString("\n```\n\n")
	}

	if rcaDoc != "" {
		b.WriteString("## 3. Root Cause Analysis (RCA)\n")
		b.WriteString("```markdown\n")
		b.WriteString(rcaDoc)
		b.WriteString("\n```\n\n")
	}

	if planDoc != "" {
		b.WriteString("## 4. Approved Plan & Boundaries\n")
		b.WriteString("```markdown\n")
		b.WriteString(planDoc)
		b.WriteString("\n```\n\n")
	}

	if feedbackRules != "" {
		b.WriteString("## 5. Active Human Feedback Rules (Binding)\n")
		b.WriteString("```markdown\n")
		b.WriteString(feedbackRules)
		b.WriteString("\n```\n\n")
	}

	if len(decisions) > 0 {
		b.WriteString("## 6. Approved Decisions\n")
		for _, d := range decisions {
			fmt.Fprintf(&b, "- %s\n", d)
		}
		b.WriteString("\n")
	}

	if roleSkill != "" {
		b.WriteString("## 7. Implementer Role Skill Reference\n")
		b.WriteString("```markdown\n")
		b.WriteString(roleSkill)
		b.WriteString("\n```\n\n")
	}

	b.WriteString("## 8. Definition of Done for Auto-Repair\n")
	b.WriteString("- [ ] Step 0 minimal reproduction test authored and verified failing on unpatched code\n")
	b.WriteString("- [ ] Code repair strictly confined to planned files\n")
	b.WriteString("- [ ] Reproduction test passes cleanly on patched code\n")
	b.WriteString("- [ ] Full regression suite passes (`make verify`, race detector, determinism `-count=2`)\n")
	b.WriteString("- [ ] All findings resolved and recorded in `validation.md`\n")

	return b.String()
}
