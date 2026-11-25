package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/vikashkumar21/sdd/internal/audit"
	"github.com/vikashkumar21/sdd/internal/deconflict"
	"github.com/vikashkumar21/sdd/internal/role"
)

var taskIDPattern = regexp.MustCompile(`^[A-Za-z]+-\d+$`)

// ReviewPromptContext represents the structured context bundle for subagent reviews.
type ReviewPromptContext struct {
	TaskID             string                       `json:"taskId"`
	TaskDir            string                       `json:"taskDir"`
	Typology           string                       `json:"typology"`
	Lens               string                       `json:"lens"`
	Loop               int                          `json:"loop,omitempty"`
	LensFocus          []string                     `json:"lensFocus,omitempty"`
	RoleSkill          string                       `json:"roleSkill,omitempty"`
	ModelRouting       *role.ModelRouting           `json:"model_routing,omitempty"`
	TriadRouting       map[string]role.ModelRouting `json:"triad_model_routing,omitempty"`
	Requirements       string                       `json:"requirements,omitempty"`
	Plan               string                       `json:"plan,omitempty"`
	Validation         string                       `json:"validation,omitempty"`
	Review             string                       `json:"review,omitempty"`
	ChangedFiles       []string                     `json:"changedFiles"`
	FeedbackRules      string                       `json:"feedbackRules"`
	BundleDir          string                       `json:"bundleDir,omitempty"`
	BundleFiles        []string                     `json:"bundleFiles,omitempty"`
	RawDiffBytes       int                          `json:"rawDiffBytes"`
	FilteredDiffBytes  int                          `json:"filteredDiffBytes"`
	DiffSavingsPercent float64                      `json:"diffSavingsPercent"`
	RawFilesCount      int                          `json:"rawFilesCount"`
	FilteredFilesCount int                          `json:"filteredFilesCount"`
	Prompt             string                       `json:"prompt"`
}

// PlanReviewPromptContext represents the structured context bundle for pre-GATE 1 plan reviews.
type PlanReviewPromptContext struct {
	TaskID        string             `json:"taskId"`
	TaskDir       string             `json:"taskDir"`
	Typology      string             `json:"typology"`
	RoleSkill     string             `json:"roleSkill,omitempty"`
	ModelRouting  *role.ModelRouting `json:"model_routing,omitempty"`
	Requirements  string             `json:"requirements,omitempty"`
	Plan          string             `json:"plan,omitempty"`
	Validation    string             `json:"validation,omitempty"`
	PlannedFiles  []string           `json:"plannedFiles"`
	FeedbackRules string             `json:"feedbackRules"`
	Decisions     []string           `json:"decisions"`
	Rubric        []string           `json:"rubric"`
	BundleDir     string             `json:"bundleDir,omitempty"`
	BundleFiles   []string           `json:"bundleFiles,omitempty"`
	Prompt        string             `json:"prompt"`
}

// ImplementPromptContext represents the structured context bundle for Phase B implementation.
type ImplementPromptContext struct {
	TaskID          string             `json:"taskId"`
	TaskDir         string             `json:"taskDir"`
	Typology        string             `json:"typology"`
	Role            string             `json:"role"`
	RoleSkill       string             `json:"roleSkill,omitempty"`
	ModelRouting    *role.ModelRouting `json:"model_routing,omitempty"`
	ActivatedSkills []string           `json:"activatedSkills,omitempty"`
	Requirements    string             `json:"requirements,omitempty"`
	Plan            string             `json:"plan,omitempty"`
	Validation      string             `json:"validation,omitempty"`
	PlannedFiles    []string           `json:"plannedFiles"`
	FeedbackRules   string             `json:"feedbackRules"`
	Decisions       []string           `json:"decisions"`
	Prompt          string             `json:"prompt"`
}

var (
	qaKeywordPattern     = regexp.MustCompile(`(?i)(?:\b(qa|quality\s+assurance|fuzzing|fuzz|race\s+detector|benchmark)\b|-race\b)`)
	devopsKeywordPattern = regexp.MustCompile(`(?i)\b(devops|sre|ci/cd|docker|container|packaging|release\s+pipeline|cross-compil)\b`)
)

func runPromptPlanReview(args []string, env IO) int {
	fs := newFlagSet("prompt plan-review", env)
	dir := fs.String("dir", ".", "project root")
	jsonFlag := fs.Bool("json", false, "output machine-readable plan review context bundle as JSON")

	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return parseExit(err)
	}

	if len(pos) != 1 {
		fprintln(env.Err, "usage: sdd prompt plan-review <T-xxx> [--dir <project>] [--json]")
		return 2
	}

	rawTaskID := strings.ToUpper(strings.TrimSpace(pos[0]))
	if !taskIDPattern.MatchString(rawTaskID) {
		fprintf(env.Err, "sdd prompt plan-review: invalid task ID format %q (expected format like T-001)\n", pos[0])
		return 1
	}

	root, err := filepath.Abs(*dir)
	if err != nil {
		fprintf(env.Err, "sdd prompt plan-review: %v\n", err)
		return 1
	}

	taskDirRel, typology, err := findTaskDirectory(root, rawTaskID)
	if err != nil {
		fprintf(env.Err, "sdd prompt plan-review: %v\n", err)
		return 1
	}

	taskDirAbs := filepath.Join(root, filepath.FromSlash(taskDirRel))

	requirements := readTaskDoc(taskDirAbs, "requirements.md")
	plan := readTaskDoc(taskDirAbs, "plan.md")
	if plan == "" {
		plan = readTaskDoc(taskDirAbs, "patch.md")
	}
	validation := readTaskDoc(taskDirAbs, "validation.md")

	plannedFiles := deconflict.ExtractPlanFiles(plan)
	if len(plannedFiles) == 0 {
		fprintf(env.Err, "sdd prompt implement: warning: no planned files resolved from %s/plan.md Section 5\n", taskDirRel)
	}
	feedbackRules := extractActiveFeedbackRules(root)
	decisions := extractApprovedDecisions(root)

	roleSkill, _ := role.ResolveSkill(root, "plan-reviewer", nil)
	rolesCfg, _ := role.LoadRoles(root)
	routing := role.ResolveRoleRouting(rolesCfg, "plan-reviewer")

	// Stage isolated, frozen review bundle by default (D-044)
	bundle, bErr := audit.StagePlanBundle(root, rawTaskID, env.now())
	if bErr != nil {
		fprintf(env.Err, "sdd prompt plan-review: warning: could not stage review bundle: %v\n", bErr)
	}

	rubric := getPlanReviewRubric()
	promptText := buildPlanReviewPrompt(rawTaskID, taskDirRel, typology, plannedFiles, feedbackRules, decisions, rubric, roleSkill, bundle, routing)

	var bundleDir string
	var bundleFiles []string
	if bundle != nil {
		bundleDir = bundle.BundleDir
		bundleFiles = bundle.Files
	}

	ctx := PlanReviewPromptContext{
		TaskID:        rawTaskID,
		TaskDir:       taskDirRel,
		Typology:      typology,
		RoleSkill:     roleSkill,
		ModelRouting:  &routing,
		Requirements:  requirements,
		Plan:          plan,
		Validation:    validation,
		PlannedFiles:  plannedFiles,
		FeedbackRules: feedbackRules,
		Decisions:     decisions,
		Rubric:        rubric,
		BundleDir:     bundleDir,
		BundleFiles:   bundleFiles,
		Prompt:        promptText,
	}

	if *jsonFlag {
		enc := json.NewEncoder(env.Out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(ctx); err != nil {
			fprintf(env.Err, "sdd prompt plan-review: json encode: %v\n", err)
			return 1
		}
		return 0
	}

	fprint(env.Out, promptText)
	return 0
}

func getPlanReviewRubric() []string {
	return []string{
		"Spec Completeness: atomic R-xxx requirements with clear verification methods; explicit non-goals; empty 'Open questions' section.",
		"Plan Soundness: steps small enough to review in one sitting; each step names exact files and proving check.",
		"Boundary Precision: Section 5 files list is exhaustive; zero unapproved file modifications or dependencies (D-007).",
		"Risk & Rollback: failure modes, concurrency constraints, and clean rollback paths identified.",
		"Testability: validation matrix maps every requirement to concrete test commands and expected outcomes.",
	}
}

func extractApprovedDecisions(root string) []string {
	content, err := os.ReadFile(filepath.Join(root, "TECH_STACK.md"))
	if err != nil {
		return nil
	}
	var decisions []string
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "| D-") && strings.Contains(line, "Approved") {
			parts := strings.Split(line, "|")
			if len(parts) >= 3 {
				id := strings.TrimSpace(parts[1])
				title := strings.TrimSpace(parts[2])
				decisions = append(decisions, fmt.Sprintf("%s: %s", id, title))
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil
	}
	return decisions
}

func resolveSubagentModelParam(tier string) string {
	switch strings.ToLower(strings.TrimSpace(tier)) {
	case "standard":
		return "flash"
	case "fast":
		return "flash_lite"
	case "frontier", "advanced":
		return "inherit"
	default:
		return "inherit"
	}
}

func roleTitleFor(roleName string) string {
	switch strings.ToLower(strings.TrimSpace(roleName)) {
	case "spec-auditor":
		return "Independent Spec Auditor"
	case "appsec-specialist":
		return "Independent AppSec Specialist"
	case "pattern-auditor":
		return "Independent Pattern & Quality Auditor"
	case "plan-reviewer":
		return "Independent Plan Reviewer"
	case "implementer":
		return "Senior Software Engineer (Implementer)"
	case "qa":
		return "Senior QA Automation Engineer (QA)"
	case "devops":
		return "Senior DevOps / SRE Engineer (DevOps)"
	case "codebase-analyst":
		return "Codebase Discovery Analyst"
	case "discovery-challenger":
		return "Adversarial Discovery Challenger"
	default:
		if strings.TrimSpace(roleName) == "" {
			return "Subagent"
		}
		return roleName
	}
}

func formatModelRoutingSection(r role.ModelRouting, roleName string) string {
	var sb strings.Builder
	sb.WriteString("## Model Routing Guidance\n")
	if roleName != "" {
		fmt.Fprintf(&sb, "- **Role:** %s\n", roleName)
	}
	fmt.Fprintf(&sb, "- **Model Tier:** %s\n", r.Tier)
	fmt.Fprintf(&sb, "- **Reasoning Effort:** %s\n", r.ReasoningEffort)
	fmt.Fprintf(&sb, "- **Recommended Family:** %s\n", r.RecommendedFamily)
	fmt.Fprintf(&sb, "- **Subagent Model Parameter:** `Model=%q`\n\n", resolveSubagentModelParam(r.Tier))
	sb.WriteString("### Subagent Invocation Snippet\n")
	sb.WriteString("```text\n")
	fmt.Fprintf(&sb, "invoke_subagent(\n  Role=%q,\n  Model=%q,\n  Prompt=\"...\"\n)\n", roleTitleFor(roleName), resolveSubagentModelParam(r.Tier))
	sb.WriteString("```\n")
	return sb.String()
}

func formatTriadModelRoutingSection(spec, sec, pat role.ModelRouting) string {
	var sb strings.Builder
	sb.WriteString("## Model Routing Guidance\n")
	sb.WriteString("Independent Subagent Review Triad dispatch recommendations:\n")
	sb.WriteString("- **Spec Auditor (spec-auditor):**\n")
	fmt.Fprintf(&sb, "  - Model Tier: %s\n", spec.Tier)
	fmt.Fprintf(&sb, "  - Reasoning Effort: %s\n", spec.ReasoningEffort)
	fmt.Fprintf(&sb, "  - Recommended Family: %s\n", spec.RecommendedFamily)
	fmt.Fprintf(&sb, "  - Subagent Model Parameter: `Model=%q`\n", resolveSubagentModelParam(spec.Tier))
	sb.WriteString("- **AppSec Specialist (appsec-specialist):**\n")
	fmt.Fprintf(&sb, "  - Model Tier: %s\n", sec.Tier)
	fmt.Fprintf(&sb, "  - Reasoning Effort: %s\n", sec.ReasoningEffort)
	fmt.Fprintf(&sb, "  - Recommended Family: %s\n", sec.RecommendedFamily)
	fmt.Fprintf(&sb, "  - Subagent Model Parameter: `Model=%q`\n", resolveSubagentModelParam(sec.Tier))
	sb.WriteString("- **Pattern Auditor (pattern-auditor):**\n")
	fmt.Fprintf(&sb, "  - Model Tier: %s\n", pat.Tier)
	fmt.Fprintf(&sb, "  - Reasoning Effort: %s\n", pat.ReasoningEffort)
	fmt.Fprintf(&sb, "  - Recommended Family: %s\n", pat.RecommendedFamily)
	fmt.Fprintf(&sb, "  - Subagent Model Parameter: `Model=%q`\n\n", resolveSubagentModelParam(pat.Tier))
	sb.WriteString("### Subagent Invocation Snippets (Review Triad)\n")
	sb.WriteString("```text\n")
	fmt.Fprintf(&sb, "// 1. Spec Auditor (Standard Tier)\ninvoke_subagent(\n  Role=%q,\n  Model=%q,\n  Prompt=\"...\"\n)\n\n", roleTitleFor("spec-auditor"), resolveSubagentModelParam(spec.Tier))
	fmt.Fprintf(&sb, "// 2. AppSec Specialist (Frontier Tier)\ninvoke_subagent(\n  Role=%q,\n  Model=%q,\n  Prompt=\"...\"\n)\n\n", roleTitleFor("appsec-specialist"), resolveSubagentModelParam(sec.Tier))
	fmt.Fprintf(&sb, "// 3. Pattern Auditor (Standard Tier)\ninvoke_subagent(\n  Role=%q,\n  Model=%q,\n  Prompt=\"...\"\n)\n", roleTitleFor("pattern-auditor"), resolveSubagentModelParam(pat.Tier))
	sb.WriteString("```\n")
	return sb.String()
}

func buildPlanReviewPrompt(taskID, taskDir, typology string, plannedFiles []string, feedbackRules string, decisions, rubric []string, roleSkill string, bundle *audit.PlanBundle, routing role.ModelRouting) string {
	var prefix strings.Builder

	prefix.WriteString("# Handover: Independent Pre-GATE 1 Plan Review\n\n")
	prefix.WriteString("You are acting as an **Independent Plan Reviewer**.\n")
	prefix.WriteString("Your mandate is to impartially audit the Phase A specification and implementation plan **before** GATE 1 presentation to the human.\n")
	prefix.WriteString("**You are strictly independent; the author agent cannot review or approve its own plan.**\n\n")

	prefix.WriteString("## SDD Anthem & Principles\n")
	prefix.WriteString("> **ALWAYS BE DETERMINISTIC. When a decision is needed, STOP and ask the human. Never guess. Never silently choose.**\n\n")

	if strings.TrimSpace(roleSkill) != "" {
		prefix.WriteString("## Canonical Role Skill: Plan Reviewer (plan-reviewer)\n")
		prefix.WriteString(strings.TrimSpace(roleSkill))
		prefix.WriteString("\n\n")
	}

	prefix.WriteString("## Plan Review Rubric & Audit Criteria\n")
	for _, item := range rubric {
		fmt.Fprintf(&prefix, "- [ ] %s\n", item)
	}
	prefix.WriteString("\n")

	prefix.WriteString(formatModelRoutingSection(routing, "plan-reviewer"))
	prefix.WriteString("\n")

	if len(decisions) > 0 {
		prefix.WriteString("## Approved Architectural Decisions (TECH_STACK.md)\n")
		for _, d := range decisions {
			fmt.Fprintf(&prefix, "- %s\n", d)
		}
		prefix.WriteString("\n")
	}

	prefix.WriteString("## Active Feedback Rules (SKILLS.md §8)\n")
	prefix.WriteString(feedbackRules)
	prefix.WriteString("\n\n")

	prefix.WriteString("## Plan Review Evaluation Standards\n")
	prefix.WriteString("1. Audit the specification and plan against the **Plan Review Rubric** above.\n")
	prefix.WriteString("2. Report any 🔴 Blocker (missing step, boundary hole, untestable assertion, unapproved dependency) or 🟡 Warning.\n")
	prefix.WriteString("3. Verify that the `Open questions` section in `requirements.md` is strictly empty.\n")
	prefix.WriteString("4. Provide an explicit sign-off recommendation: `[Approved / Changes Requested]` with actionable remediation notes.\n")

	var suffix strings.Builder
	suffix.WriteString("## Dynamic Task Context & Scope\n")
	fmt.Fprintf(&suffix, "- **Task ID:** %s\n", taskID)
	fmt.Fprintf(&suffix, "- **Task Directory:** %s\n", taskDir)
	fmt.Fprintf(&suffix, "- **Typology:** %s\n", typology)
	suffix.WriteString("- **Planned Files to Touch:**\n")
	if len(plannedFiles) == 0 {
		suffix.WriteString("  - (none resolved from plan)\n")
	} else {
		for _, f := range plannedFiles {
			fmt.Fprintf(&suffix, "  - `%s`\n", f)
		}
	}
	suffix.WriteString("\n")

	if bundle != nil {
		suffix.WriteString("## Staged Stateless Review Bundle (.sdd/review/)\n")
		suffix.WriteString("An isolated, frozen review bundle has been staged for tabula-rasa evaluation with zero conversational history:\n")
		fmt.Fprintf(&suffix, "- **Bundle Directory:** `%s/`\n", bundle.BundleDir)
		fmt.Fprintf(&suffix, "- **Frozen Specification:** `%s/spec.md` (frozen requirements and plan)\n", bundle.BundleDir)
		fmt.Fprintf(&suffix, "- **Review Rubric:** `%s/rubric.md` (canonical plan-reviewer role skill)\n", bundle.BundleDir)
		fmt.Fprintf(&suffix, "- **Bundle Manifest:** `%s/manifest.json`\n\n", bundle.BundleDir)
		suffix.WriteString("**You are an independent reviewer evaluating this task in fresh, isolated context. Rely strictly on the staged artifacts in your review bundle without conversational bias or assumptions.**\n\n")
	}

	suffix.WriteString("## Step-by-Step Plan Review Instructions\n")
	if bundle != nil {
		fmt.Fprintf(&suffix, "1. Read `%s/spec.md` (or `%s/requirements.md` and `%s/plan.md`) and audit against the **Plan Review Rubric** above.\n", bundle.BundleDir, taskDir, taskDir)
	} else {
		fmt.Fprintf(&suffix, "1. Read `%s/requirements.md`, `%s/plan.md`, and `%s/validation.md`.\n", taskDir, taskDir, taskDir)
	}
	suffix.WriteString("2. Record your findings and sign-off recommendation.\n")

	parts := PromptParts{
		StaticPrefix:  prefix.String(),
		DynamicSuffix: suffix.String(),
	}
	return parts.FullPrompt()
}

func runPromptImplement(args []string, env IO) int {
	fs := newFlagSet("prompt implement", env)
	dir := fs.String("dir", ".", "project root")
	roleFlag := fs.String("role", "implementer", "role persona to activate: implementer, qa, devops")
	jsonFlag := fs.Bool("json", false, "output machine-readable implementation context bundle as JSON")

	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return parseExit(err)
	}

	if len(pos) != 1 {
		fprintln(env.Err, "usage: sdd prompt implement <T-xxx> [--role <implementer|qa|devops>] [--dir <project>] [--json]")
		return 2
	}

	rawTaskID := strings.ToUpper(strings.TrimSpace(pos[0]))
	if !taskIDPattern.MatchString(rawTaskID) {
		fprintf(env.Err, "sdd prompt implement: invalid task ID format %q (expected format like T-001)\n", pos[0])
		return 1
	}

	targetRole := strings.ToLower(strings.TrimSpace(*roleFlag))
	switch targetRole {
	case "implementer", "qa", "devops":
		// valid
	default:
		fprintf(env.Err, "sdd prompt implement: invalid role %q (available: implementer, qa, devops)\n", *roleFlag)
		return 1
	}

	root, err := filepath.Abs(*dir)
	if err != nil {
		fprintf(env.Err, "sdd prompt implement: %v\n", err)
		return 1
	}

	taskDirRel, typology, err := findTaskDirectory(root, rawTaskID)
	if err != nil {
		fprintf(env.Err, "sdd prompt implement: %v\n", err)
		return 1
	}

	taskDirAbs := filepath.Join(root, filepath.FromSlash(taskDirRel))

	requirements := readTaskDoc(taskDirAbs, "requirements.md")
	plan := readTaskDoc(taskDirAbs, "plan.md")
	if plan == "" {
		plan = readTaskDoc(taskDirAbs, "patch.md")
	}
	validation := readTaskDoc(taskDirAbs, "validation.md")

	plannedFiles := deconflict.ExtractPlanFiles(plan)
	if len(plannedFiles) == 0 {
		fprintf(env.Err, "sdd prompt implement: warning: no planned files resolved from %s/plan.md Section 5\n", taskDirRel)
	}
	feedbackRules := extractActiveFeedbackRules(root)
	decisions := extractApprovedDecisions(root)

	roleSkill, _ := role.ResolveSkill(root, targetRole, nil)
	rolesCfg, _ := role.LoadRoles(root)
	implementRouting := role.ResolveRoleRouting(rolesCfg, targetRole)
	activatedSkills := detectOperationalSkills(plan)

	promptText := buildImplementPrompt(rawTaskID, taskDirRel, typology, targetRole, roleSkill, activatedSkills, plannedFiles, feedbackRules, decisions, implementRouting)

	ctx := ImplementPromptContext{
		TaskID:          rawTaskID,
		TaskDir:         taskDirRel,
		Typology:        typology,
		Role:            targetRole,
		RoleSkill:       roleSkill,
		ModelRouting:    &implementRouting,
		ActivatedSkills: activatedSkills,
		Requirements:    requirements,
		Plan:            plan,
		Validation:      validation,
		PlannedFiles:    plannedFiles,
		FeedbackRules:   feedbackRules,
		Decisions:       decisions,
		Prompt:          promptText,
	}

	if *jsonFlag {
		enc := json.NewEncoder(env.Out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(ctx); err != nil {
			fprintf(env.Err, "sdd prompt implement: json encode: %v\n", err)
			return 1
		}
		return 0
	}

	fprint(env.Out, promptText)
	return 0
}

func detectOperationalSkills(planContent string) []string {
	if planContent == "" {
		return nil
	}
	scanTarget := planContent
	if idx := strings.Index(planContent, "## 4. Implementation steps"); idx != -1 {
		scanTarget = planContent[idx:]
		if nextSec := strings.Index(scanTarget[len("## 4. Implementation steps"):], "## "); nextSec != -1 {
			scanTarget = scanTarget[:len("## 4. Implementation steps")+nextSec]
		}
	}

	var activated []string
	if qaKeywordPattern.MatchString(scanTarget) {
		activated = append(activated, "qa")
	}
	if devopsKeywordPattern.MatchString(scanTarget) {
		activated = append(activated, "devops")
	}
	sort.Strings(activated)
	return activated
}

func buildImplementPrompt(taskID, taskDir, typology, targetRole, roleSkill string, activatedSkills, plannedFiles []string, feedbackRules string, decisions []string, routing role.ModelRouting) string {
	roleTitle := roleTitleFor(targetRole)

	var prefix strings.Builder
	fmt.Fprintf(&prefix, "# Handover: Phase B Implementation & Engineering (%s)\n\n", roleTitle)
	fmt.Fprintf(&prefix, "You are acting as the **%s** (%s).\n", roleTitle, targetRole)
	prefix.WriteString("Your mandate is to execute the approved specification and plan step-by-step within the isolated task worktree branch.\n\n")

	prefix.WriteString("## SDD Anthem & Principles\n")
	prefix.WriteString("> **ALWAYS BE DETERMINISTIC. When a decision is needed, STOP and ask the human. Never guess. Never silently choose.**\n\n")

	if strings.TrimSpace(roleSkill) != "" {
		fmt.Fprintf(&prefix, "## Canonical Role Skill: %s (%s)\n", roleTitle, targetRole)
		prefix.WriteString(strings.TrimSpace(roleSkill))
		prefix.WriteString("\n\n")
	}

	prefix.WriteString(formatModelRoutingSection(routing, targetRole))
	prefix.WriteString("\n")

	prefix.WriteString("## Standard Engineering Policies & Language Standards\n")
	prefix.WriteString("1. **Strict Boundary Precision (Hard Rule 11):** Modify ONLY files listed under Planned Files Boundary.\n")
	prefix.WriteString("2. **Standard Library Purity (Decision D-007):** Zero third-party dependencies; standard library only.\n")
	prefix.WriteString("3. **Determinism by Construction (Hard Rule 8):** Inject clocks and ID generators; table-driven tests; race detection (`-race`); determinism verification (`-count=2`).\n")
	prefix.WriteString("4. **Git Governance:** Operate strictly within the isolated task worktree branch. Never modify `main`. The human reviews and commits at Gate 2.\n\n")

	if len(decisions) > 0 {
		prefix.WriteString("## Approved Architectural Decisions (TECH_STACK.md)\n")
		for _, d := range decisions {
			fmt.Fprintf(&prefix, "- %s\n", d)
		}
		prefix.WriteString("\n")
	}

	prefix.WriteString("## Active Feedback Rules (SKILLS.md §8)\n")
	prefix.WriteString(feedbackRules)
	prefix.WriteString("\n\n")

	var suffix strings.Builder
	suffix.WriteString("## Dynamic Task Context & Boundaries\n")
	fmt.Fprintf(&suffix, "- **Task ID:** %s\n", taskID)
	fmt.Fprintf(&suffix, "- **Task Directory:** %s\n", taskDir)
	fmt.Fprintf(&suffix, "- **Typology:** %s\n", typology)
	suffix.WriteString("- **Active Role:** " + targetRole + "\n")
	suffix.WriteString("- **Planned Files Boundary (Section 5):**\n")
	if len(plannedFiles) == 0 {
		suffix.WriteString("  - (none resolved from plan)\n")
	} else {
		for _, f := range plannedFiles {
			fmt.Fprintf(&suffix, "  - `%s`\n", f)
		}
	}
	suffix.WriteString("\n")

	if len(activatedSkills) > 0 {
		suffix.WriteString("## Conditional Operational Skills Activated by Plan\n")
		suffix.WriteString("The implementation plan contains steps activating the following specialized operational skills:\n")
		for _, s := range activatedSkills {
			fmt.Fprintf(&suffix, "- **%s**: Operational steps detected in plan.md\n", s)
		}
		suffix.WriteString("\n")
	}

	suffix.WriteString("## Step-by-Step Execution Instructions\n")
	fmt.Fprintf(&suffix, "1. Read `%s/requirements.md` and `%s/plan.md` carefully.\n", taskDir, taskDir)
	suffix.WriteString("2. Execute `plan.md` Section 4 step-by-step, running proving checks after each step.\n")
	suffix.WriteString("3. Enforce strict boundary precision (Hard Rule 11): modify ONLY files listed under Planned Files Boundary above.\n")
	suffix.WriteString("4. Standard library only (Decision D-007): introduce zero third-party dependencies.\n")
	suffix.WriteString("5. Determinism: inject clocks and ID generators; table-driven tests; verify tests pass with identical outputs.\n")
	fmt.Fprintf(&suffix, "6. After implementation, run all verification commands and record real outcomes in `%s/validation.md`.\n", taskDir)

	parts := PromptParts{
		StaticPrefix:  prefix.String(),
		DynamicSuffix: suffix.String(),
	}
	return parts.FullPrompt()
}

func runPromptReview(args []string, env IO) int {
	fs := newFlagSet("prompt review", env)
	dir := fs.String("dir", ".", "project root")
	jsonFlag := fs.Bool("json", false, "output machine-readable review context bundle as JSON")
	lensFlag := fs.String("lens", "all", "review lens: spec, security (or appsec), pattern, or all")
	loopFlag := fs.Int("loop", 1, "remediation loop number (1..3 per 3-loop circuit breaker)")

	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return parseExit(err)
	}

	if len(pos) != 1 {
		fprintln(env.Err, "usage: sdd prompt review <T-xxx> [--lens <spec|security|pattern|all>] [--loop <1..3>] [--dir <project>] [--json]")
		return 2
	}

	if *loopFlag < 1 || *loopFlag > 3 {
		fprintf(env.Err, "sdd prompt review: invalid --loop %d (expected 1, 2, or 3 per 3-loop circuit breaker)\n", *loopFlag)
		return 2
	}

	normLens := audit.NormalizeLens(*lensFlag)
	switch normLens {
	case "spec", "security", "pattern", "all":
		// valid
	default:
		fprintf(env.Err, "sdd prompt review: invalid --lens %q (expected: spec, security, pattern, or all)\n", *lensFlag)
		return 2
	}

	rawTaskID := strings.ToUpper(strings.TrimSpace(pos[0]))
	if !taskIDPattern.MatchString(rawTaskID) {
		fprintf(env.Err, "sdd prompt review: invalid task ID format %q (expected format like T-001)\n", pos[0])
		return 1
	}

	root, err := filepath.Abs(*dir)
	if err != nil {
		fprintf(env.Err, "sdd prompt review: %v\n", err)
		return 1
	}

	taskDirRel, typology, err := findTaskDirectory(root, rawTaskID)
	if err != nil {
		fprintf(env.Err, "sdd prompt review: %v\n", err)
		return 1
	}

	taskDirAbs := filepath.Join(root, filepath.FromSlash(taskDirRel))

	requirements := readTaskDoc(taskDirAbs, "requirements.md")
	plan := readTaskDoc(taskDirAbs, "plan.md")
	if plan == "" {
		plan = readTaskDoc(taskDirAbs, "patch.md") // hotfix typology
	}
	validation := readTaskDoc(taskDirAbs, "validation.md")
	review := readTaskDoc(taskDirAbs, "review.md")

	changedFiles := resolveChangedFiles(root, plan)
	feedbackRules := extractActiveFeedbackRules(root)

	var roleSkill string
	switch normLens {
	case "spec":
		roleSkill, _ = role.ResolveSkill(root, "spec-auditor", nil)
	case "security":
		roleSkill, _ = role.ResolveSkill(root, "appsec-specialist", nil)
	case "pattern":
		roleSkill, _ = role.ResolveSkill(root, "pattern-auditor", nil)
	case "all":
		s1, _ := role.ResolveSkill(root, "spec-auditor", nil)
		s2, _ := role.ResolveSkill(root, "appsec-specialist", nil)
		s3, _ := role.ResolveSkill(root, "pattern-auditor", nil)
		var parts []string
		if s1 != "" {
			parts = append(parts, s1)
		}
		if s2 != "" {
			parts = append(parts, s2)
		}
		if s3 != "" {
			parts = append(parts, s3)
		}
		roleSkill = strings.Join(parts, "\n\n---\n\n")
	}

	// Stage isolated, frozen review bundle by default (D-044)
	bundle, bErr := audit.StageReviewBundleWithLoop(root, rawTaskID, normLens, *loopFlag, env.now())
	if bErr != nil {
		fprintf(env.Err, "sdd prompt review: warning: could not stage review bundle: %v\n", bErr)
	}

	rolesCfg, _ := role.LoadRoles(root)
	var activeRouting role.ModelRouting
	var triadRouting map[string]role.ModelRouting
	switch normLens {
	case "spec":
		activeRouting = role.ResolveRoleRouting(rolesCfg, "spec-auditor")
	case "security":
		activeRouting = role.ResolveRoleRouting(rolesCfg, "appsec-specialist")
	case "pattern":
		activeRouting = role.ResolveRoleRouting(rolesCfg, "pattern-auditor")
	default: // "all"
		specR := role.ResolveRoleRouting(rolesCfg, "spec-auditor")
		secR := role.ResolveRoleRouting(rolesCfg, "appsec-specialist")
		patR := role.ResolveRoleRouting(rolesCfg, "pattern-auditor")
		activeRouting = secR // Frontier tier across triad
		triadRouting = map[string]role.ModelRouting{
			"spec-auditor":      specR,
			"appsec-specialist": secR,
			"pattern-auditor":   patR,
		}
	}

	lensFocus := getLensFocusItems(normLens)
	promptText := buildReviewPromptWithLens(rawTaskID, taskDirRel, normLens, changedFiles, feedbackRules, lensFocus, roleSkill, bundle, activeRouting, triadRouting)

	var bundleDir string
	var bundleFiles []string
	var rawBytes, filteredBytes, rawFiles, filteredFiles int
	var savingsPercent float64
	if bundle != nil {
		bundleDir = bundle.BundleDir
		bundleFiles = bundle.Files
		rawBytes = bundle.RawDiffBytes
		filteredBytes = bundle.FilteredDiffBytes
		savingsPercent = bundle.DiffSavingsPercent
		rawFiles = bundle.RawFilesCount
		filteredFiles = bundle.FilteredFilesCount
	}

	ctx := ReviewPromptContext{
		TaskID:             rawTaskID,
		TaskDir:            taskDirRel,
		Typology:           typology,
		Lens:               normLens,
		Loop:               *loopFlag,
		LensFocus:          lensFocus,
		RoleSkill:          roleSkill,
		ModelRouting:       &activeRouting,
		TriadRouting:       triadRouting,
		Requirements:       requirements,
		Plan:               plan,
		Validation:         validation,
		Review:             review,
		ChangedFiles:       changedFiles,
		FeedbackRules:      feedbackRules,
		BundleDir:          bundleDir,
		BundleFiles:        bundleFiles,
		RawDiffBytes:       rawBytes,
		FilteredDiffBytes:  filteredBytes,
		DiffSavingsPercent: savingsPercent,
		RawFilesCount:      rawFiles,
		FilteredFilesCount: filteredFiles,
		Prompt:             promptText,
	}

	if *jsonFlag {
		enc := json.NewEncoder(env.Out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(ctx); err != nil {
			fprintf(env.Err, "sdd prompt review: json encode: %v\n", err)
			return 1
		}
		return 0
	}

	fprint(env.Out, promptText)
	return 0
}

func getLensFocusItems(lens string) []string {
	specItems := []string{
		"Traceability: every requirement (R-xxx) maps to verified code and concrete test evidence in validation.md.",
		"Plan boundaries: touch ONLY files listed in plan.md Section 5 (zero scope creep).",
		"Negative and failure cases are explicitly tested and verified.",
		"Standard library purity: zero unapproved third-party dependencies (D-007).",
	}
	secItems := []string{
		"Input validation & bound sizes on all external inputs; fail safe on ambiguity.",
		"Filesystem confinement: reject directory traversal, enforce path bounds via os.Root or clean canonical paths.",
		"Secrets & privacy hygiene: zero credentials, tokens, connection strings, or personal data; synthetic test fixtures only.",
		"Command execution safety: never concatenate shell strings; use separate arguments.",
	}
	patItems := []string{
		"Architectural decision conformance: strictly respect approved decisions in TECH_STACK.md (D-xxx).",
		"Determinism: inject clocks, random sources, and ID generators; no map iteration order dependency.",
		"Code idioms: follow Effective Go, Go Code Review Comments, context.Context first, %w error wrapping.",
		"Continuous learning: verify compliance with active feedback rules in SKILLS.md §8 (F-xxx).",
		"Concurrency safety: verify goroutine lifecycles, race detector compliance, and no data races.",
	}

	switch lens {
	case "spec":
		return specItems
	case "security":
		return secItems
	case "pattern":
		return patItems
	default:
		var all []string
		all = append(all, specItems...)
		all = append(all, secItems...)
		all = append(all, patItems...)
		return all
	}
}

func findTaskDirectory(root, taskID string) (relDir, typology string, err error) {
	typologies := []struct {
		dir  string
		name string
	}{
		{filepath.Join("tasks", "features"), "features"},
		{filepath.Join("tasks", "bugfixes"), "bugfixes"},
		{filepath.Join("tasks", "hotfixes"), "hotfixes"},
		{filepath.Join("tasks", "spikes"), "spikes"},
		{"features", "features"},
		{"bugfixes", "bugfixes"},
		{"hotfixes", "hotfixes"},
		{"spikes", "spikes"},
	}
	prefix := taskID + "-"

	for _, topo := range typologies {
		base := filepath.Join(root, topo.dir)
		entries, err := os.ReadDir(base)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() || strings.HasPrefix(e.Name(), "_") {
				continue
			}
			if e.Name() == taskID || strings.HasPrefix(e.Name(), prefix) {
				return filepath.ToSlash(filepath.Join(topo.dir, e.Name())), topo.name, nil
			}
		}
	}

	return "", "", fmt.Errorf("task %s folder not found across tasks/ (or legacy features/, bugfixes/, hotfixes/, spikes/)", taskID)
}

func readTaskDoc(taskDir, filename string) string {
	b, err := os.ReadFile(filepath.Join(taskDir, filename))
	if err != nil {
		return ""
	}
	return strings.ReplaceAll(string(b), "\r\n", "\n")
}

func resolveChangedFiles(root, planContent string) []string {
	fileMap := make(map[string]bool)

	// 1. Attempt git diff against main or HEAD
	if isGitRepo(root) {
		cmd := osexec.Command("git", "diff", "--name-only", "main...HEAD")
		cmd.Dir = root
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err == nil && out.Len() > 0 {
			scanner := bufio.NewScanner(&out)
			for scanner.Scan() {
				f := strings.TrimSpace(scanner.Text())
				if f != "" {
					fileMap[filepath.ToSlash(f)] = true
				}
			}
		}

		// Also check unstaged/staged working tree changes
		statusCmd := osexec.Command("git", "status", "--porcelain")
		statusCmd.Dir = root
		var statusOut bytes.Buffer
		statusCmd.Stdout = &statusOut
		if err := statusCmd.Run(); err == nil && statusOut.Len() > 0 {
			scanner := bufio.NewScanner(&statusOut)
			for scanner.Scan() {
				line := scanner.Text()
				if len(line) >= 4 {
					f := strings.TrimSpace(line[3:])
					// Handle rename "old -> new"
					if idx := strings.Index(f, "->"); idx != -1 {
						f = strings.TrimSpace(f[idx+2:])
					}
					if f != "" {
						fileMap[filepath.ToSlash(f)] = true
					}
				}
			}
		}
	}

	// 2. If no git files found or as fallback, extract from plan
	if len(fileMap) == 0 && planContent != "" {
		for _, f := range deconflict.ExtractPlanFiles(planContent) {
			fileMap[f] = true
		}
	}

	var files []string
	for f := range fileMap {
		files = append(files, f)
	}
	sort.Strings(files)
	return files
}

func isGitRepo(root string) bool {
	gitDir := filepath.Join(root, ".git")
	fi, err := os.Stat(gitDir)
	return err == nil && (fi.IsDir() || !fi.IsDir()) // .git can be a file in worktrees
}

func extractActiveFeedbackRules(root string) string {
	content, err := os.ReadFile(filepath.Join(root, "SKILLS.md"))
	if err != nil {
		return "(none active)"
	}

	scanner := bufio.NewScanner(bytes.NewReader(content))
	inSection8 := false
	inRegister := false
	var ruleLines []string

	section8Header := regexp.MustCompile(`^##\s+8\.\s+Human feedback rules`)
	nextSectionHeader := regexp.MustCompile(`^##\s+\d+\.`)

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if section8Header.MatchString(trimmed) {
			inSection8 = true
			continue
		}

		if inSection8 && nextSectionHeader.MatchString(trimmed) {
			break
		}

		if inSection8 {
			if strings.HasPrefix(trimmed, "### Register") {
				inRegister = true
				continue
			}
			if inRegister && strings.HasPrefix(trimmed, "|") {
				if strings.Contains(trimmed, "---") || strings.Contains(strings.ToLower(trimmed), "applies to") {
					continue
				}
				if strings.Contains(trimmed, "(none yet)") {
					continue
				}
				parts := strings.Split(trimmed, "|")
				if len(parts) >= 4 {
					id := strings.TrimSpace(parts[1])
					rule := strings.TrimSpace(parts[3])
					if id != "" && rule != "" {
						ruleLines = append(ruleLines, fmt.Sprintf("- **%s:** %s", id, rule))
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "(none active)"
	}

	if len(ruleLines) == 0 {
		return "(none active)"
	}
	return strings.Join(ruleLines, "\n")
}

func buildReviewPromptWithLens(taskID, taskDir, lens string, files []string, feedbackRules string, focusItems []string, roleSkill string, bundle *audit.ReviewBundle, activeRouting role.ModelRouting, triadRouting map[string]role.ModelRouting) string {
	lensTitle := "Ensemble Review Triad"
	roleDesc := "across all three pillars (Spec Auditor, AppSec Specialist, Pattern Auditor)"
	switch lens {
	case "spec":
		lensTitle = "Spec Auditor Lens"
		roleDesc = "with a dedicated focus on requirement traceability, negative test cases, and plan boundaries"
	case "security":
		lensTitle = "AppSec Specialist Lens"
		roleDesc = "with a dedicated focus on threat modeling, input validation, path confinement, and secret hygiene"
	case "pattern":
		lensTitle = "Pattern & Quality Auditor Lens"
		roleDesc = "with a dedicated focus on architectural decision conformance, determinism, language idioms, and feedback rules"
	}

	var prefix strings.Builder
	fmt.Fprintf(&prefix, "# Handover: Independent Subagent Review Triad (%s)\n\n", lensTitle)
	fmt.Fprintf(&prefix, "You are acting as an **Independent Subagent Reviewer (%s)**.\n", lensTitle)
	fmt.Fprintf(&prefix, "Your mandate is to impartially audit candidate implementations %s before GATE 2 human review.\n", roleDesc)
	prefix.WriteString("**You are an independent, tabula-rasa reviewer evaluating this task in fresh, isolated context. Rely strictly on the staged artifacts in your review bundle without conversational bias or assumptions.**\n\n")

	prefix.WriteString("## SDD Anthem & Principles\n")
	prefix.WriteString("> **ALWAYS BE DETERMINISTIC. When a decision is needed, STOP and ask the human. Never guess. Never silently choose.**\n\n")

	prefix.WriteString("## Review Rubric (3-Tier Severity)\n")
	prefix.WriteString("- 🔴 **Blocker (Developer Attention Needed):** Spec violation (`requirements.md` not met), plan drift (files touched outside `plan.md`), security hazard (secrets/unvalidated inputs), test gaps, unapproved dependency. MUST BE RESOLVED before commit.\n")
	prefix.WriteString("- 🟡 **Warning (Attention Suggested):** Sub-optimal pattern, edge case risk, missing doc or comment.\n")
	prefix.WriteString("- 🟢 **Minor (Safe / Informational):** Style, minor naming or non-critical improvement.\n\n")

	if strings.TrimSpace(roleSkill) != "" {
		prefix.WriteString("## Canonical Role Skill Instructions\n")
		prefix.WriteString(strings.TrimSpace(roleSkill))
		prefix.WriteString("\n\n")
	}

	if lens == "all" && triadRouting != nil {
		prefix.WriteString(formatTriadModelRoutingSection(triadRouting["spec-auditor"], triadRouting["appsec-specialist"], triadRouting["pattern-auditor"]))
		prefix.WriteString("\n")
	} else {
		targetRole := "spec-auditor"
		switch lens {
		case "security":
			targetRole = "appsec-specialist"
		case "pattern":
			targetRole = "pattern-auditor"
		}
		prefix.WriteString(formatModelRoutingSection(activeRouting, targetRole))
		prefix.WriteString("\n")
	}

	prefix.WriteString("## Lens Focus & Verification Questions\n")
	for _, item := range focusItems {
		fmt.Fprintf(&prefix, "- [ ] %s\n", item)
	}
	prefix.WriteString("\n")

	if lens != "all" {
		prefix.WriteString("## Parallel Review Triad Protocol\n")
		fmt.Fprintf(&prefix, "You are one of three parallel subagents in the **3-Subagent Parallel Review Triad** (Spec Auditor, AppSec Specialist, Pattern Auditor).\n")
		fmt.Fprintf(&prefix, "Your evaluation focus is EXCLUSIVELY the %s. Record your findings under the `%s` section in `review.md`.\n", lensTitle, lensTitle)
		prefix.WriteString("If multiple subagents report findings, the primary agent will remediate all findings together in a single batched fix commit (consuming 1 loop in the 3-loop circuit breaker).\n\n")
	}

	prefix.WriteString("## Active Feedback Rules (SKILLS.md §8)\n")
	prefix.WriteString(feedbackRules)
	prefix.WriteString("\n\n")

	prefix.WriteString("## General Review Standards & Traceability Protocol\n")
	prefix.WriteString("1. **Step 1 (Unanchored Traceability Scan):** Open `ledger.md` and verify each requirement against codebase implementation and tests before opening `diff.patch`. Ensure every requirement has concrete code symbols, positive test evidence, and negative/boundary test evidence. Any missing requirement is an automatic 🔴 Blocker.\n")
	prefix.WriteString("2. **Step 2 (Defect Remediation Verification):** Open `findings_to_verify.md` (when present on remediation loops) and verify whether each previous defect has been genuinely resolved in `diff.patch`. Do not accept conversational developer explanations or unverified assertions.\n")
	prefix.WriteString("3. **Step 3 (Diff Quality & Lens Audit):** Inspect `diff.patch` against `spec.md` using the rubric in `rubric.md`.\n")
	prefix.WriteString("4. Per Rule F-001, all findings (🔴 Blockers, 🟡 Warnings, 🟢 Minors) must be cataloged clearly; the worker agent is required to resolve all findings or provide explicit justification before GATE 2.\n")

	var suffix strings.Builder
	suffix.WriteString("## Dynamic Task Context & Scope\n")
	fmt.Fprintf(&suffix, "- **Task ID:** %s\n", taskID)
	fmt.Fprintf(&suffix, "- **Task Directory:** %s\n", taskDir)
	fmt.Fprintf(&suffix, "- **Review Lens:** %s\n", lens)
	if bundle != nil && bundle.Loop > 1 {
		fmt.Fprintf(&suffix, "- **Remediation Loop:** Remediation Review (Loop %d)\n", bundle.Loop)
	} else {
		suffix.WriteString("- **Remediation Loop:** Initial Review (Loop 1)\n")
	}
	suffix.WriteString("- **Changed Files:**\n")
	if len(files) == 0 {
		suffix.WriteString("  - (none resolved from git diff or plan)\n")
	} else {
		for _, f := range files {
			fmt.Fprintf(&suffix, "  - `%s`\n", f)
		}
	}
	suffix.WriteString("\n")

	if bundle != nil {
		suffix.WriteString("## Staged Stateless Review Bundle (.sdd/review/)\n")
		suffix.WriteString("An isolated, frozen review bundle has been staged for tabula-rasa evaluation with zero conversational history:\n")
		fmt.Fprintf(&suffix, "- **Bundle Directory:** `%s/`\n", bundle.BundleDir)
		fmt.Fprintf(&suffix, "- **Frozen Specification:** `%s/spec.md` (frozen requirements and plan)\n", bundle.BundleDir)
		fmt.Fprintf(&suffix, "- **Traceability Ledger:** `%s/ledger.md` (unanchored requirements ledger to complete before reading diff.patch)\n", bundle.BundleDir)
		if bundle.Loop > 1 {
			fmt.Fprintf(&suffix, "- **Findings to Verify:** `%s/findings_to_verify.md` (objective checklist of previous defects to verify without developer prose)\n", bundle.BundleDir)
		}
		fmt.Fprintf(&suffix, "- **Git Diff Patch:** `%s/diff.patch` (working tree changes and untracked files)\n", bundle.BundleDir)
		fmt.Fprintf(&suffix, "- **Review Rubric:** `%s/rubric.md` (canonical role skill instructions for %s)\n", bundle.BundleDir, bundle.Lens)
		fmt.Fprintf(&suffix, "- **Bundle Manifest:** `%s/manifest.json`\n", bundle.BundleDir)
		if bundle.DiffSavingsPercent > 0.0 {
			fmt.Fprintf(&suffix, "- **Diff Optimization:** %d bytes filtered from %d raw bytes (%.1f%% token savings across %d files)\n", bundle.FilteredDiffBytes, bundle.RawDiffBytes, bundle.DiffSavingsPercent, bundle.FilteredFilesCount)
		}
		suffix.WriteString("\n")
		suffix.WriteString("**You are an independent reviewer evaluating this task in fresh, isolated context. Rely strictly on the staged artifacts in your review bundle without conversational bias or assumptions.**\n\n")
	}

	suffix.WriteString("## Step-by-Step Task Review Instructions\n")
	if bundle != nil && bundle.Loop > 1 {
		fmt.Fprintf(&suffix, "1. **Step 1 (Unanchored Traceability Scan):** Open `%s/ledger.md` and verify each requirement against codebase implementation and tests before opening `%s/diff.patch`. Ensure every requirement has concrete code symbols, positive test evidence, and negative/boundary test evidence. Any missing requirement is an automatic 🔴 Blocker.\n", bundle.BundleDir, bundle.BundleDir)
		fmt.Fprintf(&suffix, "2. **Step 2 (Defect Remediation Verification):** Open `%s/findings_to_verify.md` and verify whether each previous defect has been genuinely resolved in `%s/diff.patch`. Do not accept conversational developer explanations or unverified assertions.\n", bundle.BundleDir, bundle.BundleDir)
		fmt.Fprintf(&suffix, "3. **Step 3 (Diff Quality & Lens Audit):** Inspect `%s/diff.patch` against `%s/spec.md` using the rubric in `%s/rubric.md` to ensure no new regressions, plan drift, or security flaws were introduced during remediation.\n", bundle.BundleDir, bundle.BundleDir, bundle.BundleDir)
	} else if bundle != nil {
		fmt.Fprintf(&suffix, "1. **Step 1 (Unanchored Traceability Scan):** Open `%s/ledger.md` and verify each requirement against codebase implementation and tests before opening `%s/diff.patch`. Ensure every requirement has concrete code symbols, positive test evidence, and negative/boundary test evidence. Any missing requirement is an automatic 🔴 Blocker.\n", bundle.BundleDir, bundle.BundleDir)
		fmt.Fprintf(&suffix, "2. **Step 2 (Diff Quality & Lens Audit):** Inspect `%s/diff.patch` against `%s/spec.md` using the rubric in `%s/rubric.md`.\n", bundle.BundleDir, bundle.BundleDir, bundle.BundleDir)
	} else {
		fmt.Fprintf(&suffix, "1. **Step 1 (Unanchored Traceability Scan):** Inspect candidate implementation and tests against `%s/requirements.md` before reading git diff. Ensure every requirement has concrete code symbols, positive test evidence, and negative/boundary test evidence.\n", taskDir)
		fmt.Fprintf(&suffix, "2. **Step 2 (Diff Quality & Lens Audit):** Inspect the git diff and changed files against `%s/requirements.md` and `%s/plan.md`.\n", taskDir, taskDir)
	}
	suffix.WriteString("3. Verify each point listed under **Lens Focus & Verification Questions**.\n")
	fmt.Fprintf(&suffix, "4. Record all findings, the completed Requirements Traceability Ledger, and sign-off in `%s/review.md`.\n", taskDir)
	suffix.WriteString("   - If findings include any 🔴 Blocker, set status to `Changes requested` with specific remediation steps.\n")
	suffix.WriteString("   - Per Rule F-001, all findings (🔴 Blockers, 🟡 Warnings, 🟢 Minors) must be cataloged clearly; the worker agent is required to resolve all findings or provide explicit justification before GATE 2.\n")
	suffix.WriteString("   - If 0 Blockers remain, set status to `Approved` with your reviewer attribution (including your lens name).\n")

	parts := PromptParts{
		StaticPrefix:  prefix.String(),
		DynamicSuffix: suffix.String(),
	}
	return parts.FullPrompt()
}
