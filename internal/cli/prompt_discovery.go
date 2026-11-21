package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vikashkumar21/sdd/internal/audit"
	"github.com/vikashkumar21/sdd/internal/context"
	"github.com/vikashkumar21/sdd/internal/detect"
	"github.com/vikashkumar21/sdd/internal/roadmap"
	"github.com/vikashkumar21/sdd/internal/role"
)

// DiscoveryPromptContext represents the structured context bundle emitted with --json.
type DiscoveryPromptContext struct {
	TaskID                  string   `json:"taskId,omitempty"`
	TaskDir                 string   `json:"taskDir,omitempty"`
	Typology                string   `json:"typology,omitempty"`
	ProjectRoot             string   `json:"projectRoot"`
	RoleSkill               string   `json:"roleSkill"`
	Ensemble                bool     `json:"ensemble,omitempty"`
	BundleDir               string   `json:"bundleDir,omitempty"`
	ChallengerRoleSkill     string   `json:"challengerRoleSkill,omitempty"`
	BlindSpotVectors        []string `json:"blindSpotVectors,omitempty"`
	ReconciliationChecklist []string `json:"reconciliationChecklist,omitempty"`
	DetectedLanguages       []string `json:"detectedLanguages"`
	LivingDocs              []string `json:"livingDocs"`
	Checklist               []string `json:"checklist"`
	StagedContext           []string `json:"stagedContext,omitempty"`
	Requirements            string   `json:"requirements,omitempty"`
	Plan                    string   `json:"plan,omitempty"`
	Prompt                  string   `json:"prompt"`
}

func getDiscoveryLivingDocs() []string {
	return []string{
		"TECH_STACK.md (§3 Architecture, §4 Layout, §6 Decisions, §7 Verified Facts, §8 Dependencies)",
		"SKILLS.md (§1 Language Standards & Idioms, §7 Pitfalls Log)",
		"ROADMAP.md (§7 Parking Lot for Discovered Debt, Subsystems & Remediation)",
	}
}

func getDiscoveryChecklist() []string {
	return []string{
		"Pillar 1: AST & Module Map (Directory topology, entry points, package manifests, build scripts)",
		"Pillar 2: Design System & UI Patterns (Font families, CSS frameworks, color themes, component hierarchy)",
		"Pillar 3: Data & State Modeling (Database schemas, ORM models, state containers, APIs)",
		"Pillar 4: Anti-Patterns & Technical Debt (Fragile dependencies, unhandled error patterns, security risks, undocumented quirks)",
	}
}

func getDiscoveryBlindSpotVectors() []string {
	return []string{
		"Vector 1: Hidden Entry Points & Triggers (CLI flags/subcommands, background daemon loops, cron/schedulers, queue workers, webhooks, and event consumers unreferenced in primary entrypoints)",
		"Vector 2: Concurrency, Lifecycle & Shutdown Invariants (unbounded goroutines/threads, missing cancellation propagation, SIGINT/SIGTERM drain hooks, package-level mutable state, race conditions in init)",
		"Vector 3: Data Boundary Invariants & Schema Drift (unhandled database errors, uncommitted/unrolled-back transactions, raw SQL queries bypassing the ORM, schema migration mismatches, in-memory cache desync)",
		"Vector 4: Dynamic Dispatch & Runtime Feature Flags (reflection-based routing, plugin systems, runtime feature flags, environment-conditional code paths invisible to static call graphs)",
		"Vector 5: Orphan & Dead Code (dangling configurations, abandoned routes/endpoints, unexported/unreferenced symbols, unused dependencies)",
		"Vector 6: Implicit External & Ambient Dependencies (undocumented env vars, ambient binaries/system tools assumed to exist on the host, undeclared cloud SDK defaults, unrecorded third-party APIs)",
		"Vector 7: Architectural & Documentation Contradictions (direct discrepancies between existing README/architecture docs and actual runtime code behavior, stale comments, unverified assumptions)",
	}
}

func getDiscoveryReconciliationChecklist() []string {
	return []string{
		"1. Cross-reference primary analyst AST map with challenger-discovered entry points and async event consumers.",
		"2. Reconcile database models and data flows against migration history and raw queries.",
		"3. Ensure all implicit external dependencies, tools, and env vars are registered in TECH_STACK.md §8 and configuration schemas.",
		"4. Sweep codebase for dead code, abandoned endpoints, and dangling config keys, logging candidates into ROADMAP.md §7 Parking Lot.",
		"5. Verify that existing documentation discrepancies are resolved in living documents prior to Gate 1 approval.",
	}
}

func readTaskPlan(taskDirAbs string) string {
	plan := readTaskDoc(taskDirAbs, "plan.md")
	if plan == "" {
		plan = readTaskDoc(taskDirAbs, "patch.md")
	}
	return plan
}

func runPromptDiscovery(args []string, env IO) int {
	fs := newFlagSet("prompt discovery", env)
	dir := fs.String("dir", ".", "project root")
	taskFlag := fs.String("task", "", "focus task, for example T-001 (default: discovery or first open task)")
	ensembleFlag := fs.Bool("ensemble", false, "output multi-agent adversarial discovery ensemble brief")
	jsonFlag := fs.Bool("json", false, "output machine-readable discovery context bundle as JSON")

	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return parseExit(err)
	}

	if len(pos) > 1 {
		fprintln(env.Err, "usage: sdd prompt discovery [T-xxx] [--task T-xxx] [--ensemble] [--dir <project>] [--json]")
		return 2
	}

	root, err := filepath.Abs(*dir)
	if err != nil {
		fprintf(env.Err, "sdd prompt discovery: %v\n", err)
		return 1
	}

	var rawTaskID string
	if len(pos) == 1 && *taskFlag != "" {
		posID := strings.ToUpper(strings.TrimSpace(pos[0]))
		flagID := strings.ToUpper(strings.TrimSpace(*taskFlag))
		if posID != flagID {
			fprintf(env.Err, "sdd prompt discovery: conflicting task IDs specified (%q vs %q)\n", pos[0], *taskFlag)
			return 2
		}
		rawTaskID = posID
	} else if len(pos) == 1 {
		rawTaskID = strings.ToUpper(strings.TrimSpace(pos[0]))
	} else if *taskFlag != "" {
		rawTaskID = strings.ToUpper(strings.TrimSpace(*taskFlag))
	}

	if rawTaskID != "" && !taskIDPattern.MatchString(rawTaskID) {
		fprintf(env.Err, "sdd prompt discovery: invalid task ID format %q (expected format like T-001)\n", rawTaskID)
		return 1
	}

	var taskDirRel, typology, reqs, plan string
	if rawTaskID != "" {
		taskDirRel, typology, err = findTaskDirectory(root, rawTaskID)
		if err != nil {
			fprintf(env.Err, "sdd prompt discovery: %v\n", err)
			return 1
		}
		taskDirAbs := filepath.Join(root, filepath.FromSlash(taskDirRel))
		reqs = readTaskDoc(taskDirAbs, "requirements.md")
		plan = readTaskPlan(taskDirAbs)
	} else {
		// If task ID is omitted, check if ROADMAP.md has an open discovery task or first open task
		roadmapPath := filepath.Join(root, "ROADMAP.md")
		if data, err := os.ReadFile(roadmapPath); err == nil {
			tasks := roadmap.Parse(string(data))
			for _, t := range tasks {
				lowerTitle := strings.ToLower(t.Title)
				if (strings.Contains(lowerTitle, "discovery") || strings.Contains(lowerTitle, "explore") || strings.Contains(lowerTitle, "brownfield")) && roadmap.Kind(t.Status) != "done" {
					if rel, topo, err := findTaskDirectory(root, t.ID); err == nil {
						rawTaskID = t.ID
						taskDirRel = rel
						typology = topo
						taskDirAbs := filepath.Join(root, filepath.FromSlash(taskDirRel))
						reqs = readTaskDoc(taskDirAbs, "requirements.md")
						plan = readTaskPlan(taskDirAbs)
						break
					}
				}
			}
		}
	}

	// 1. Detect project metadata (languages, manifests, commands)
	detResult, _ := detect.Detect(root)
	detectedLangs := append([]string{}, detResult.Languages...)
	if len(detectedLangs) == 0 && len(detResult.Other) > 0 {
		detectedLangs = append(detectedLangs, detResult.Other...)
	}
	sort.Strings(detectedLangs)

	manifests := append([]string{}, detResult.Manifests...)
	sort.Strings(manifests)

	// 2. Read staged context items if present
	var stagedList []string
	if m, exists, err := context.ReadManifest(root); err == nil && exists {
		for _, e := range m.Entries {
			stagedList = append(stagedList, fmt.Sprintf("%s (%s, %d bytes)", e.Origin, e.Format, e.Bytes))
		}
		sort.Strings(stagedList)
	}

	// 3. Resolve role skills via two-tier storage
	roleSkill, err := role.ResolveSkill(root, "codebase-analyst", nil)
	if err != nil {
		fprintf(env.Err, "sdd prompt discovery: failed to resolve codebase-analyst role skill: %v\n", err)
		return 1
	}

	var challengerSkill string
	var bundleDir string
	if *ensembleFlag {
		challengerSkill, err = role.ResolveSkill(root, "discovery-challenger", nil)
		if err != nil {
			fprintf(env.Err, "sdd prompt discovery: failed to resolve discovery-challenger role skill: %v\n", err)
			return 1
		}

		if rawTaskID != "" {
			bundle, stageErr := audit.StageDiscoveryBundle(root, rawTaskID, "", challengerSkill, env.now())
			if stageErr != nil {
				fprintf(env.Err, "sdd prompt discovery: warning: could not stage review bundle: %v\n", stageErr)
			} else if bundle != nil {
				bundleDir = bundle.BundleDir
			}
		}
	}

	// 4. Assemble prompt text
	var promptText string
	if *ensembleFlag {
		promptText = buildDiscoveryEnsemblePrompt(
			rawTaskID,
			taskDirRel,
			typology,
			root,
			detectedLangs,
			manifests,
			detResult.Build,
			detResult.Test,
			detResult.Lint,
			stagedList,
			roleSkill,
			challengerSkill,
			bundleDir,
			reqs,
			plan,
		)
	} else {
		promptText = buildDiscoveryPrompt(
			rawTaskID,
			taskDirRel,
			typology,
			root,
			detectedLangs,
			manifests,
			detResult.Build,
			detResult.Test,
			detResult.Lint,
			stagedList,
			roleSkill,
			reqs,
			plan,
		)
	}

	// 5. Emit JSON or human-readable markdown
	if *jsonFlag {
		ctx := DiscoveryPromptContext{
			TaskID:            rawTaskID,
			TaskDir:           taskDirRel,
			Typology:          typology,
			ProjectRoot:       filepath.ToSlash(root),
			RoleSkill:         roleSkill,
			DetectedLanguages: detectedLangs,
			LivingDocs:        getDiscoveryLivingDocs(),
			Checklist:         getDiscoveryChecklist(),
			StagedContext:     stagedList,
			Requirements:      reqs,
			Plan:              plan,
			Prompt:            promptText,
		}
		if *ensembleFlag {
			ctx.Ensemble = true
			ctx.BundleDir = bundleDir
			ctx.ChallengerRoleSkill = challengerSkill
			ctx.BlindSpotVectors = getDiscoveryBlindSpotVectors()
			ctx.ReconciliationChecklist = getDiscoveryReconciliationChecklist()
		}
		data, err := json.MarshalIndent(ctx, "", "  ")
		if err != nil {
			fprintf(env.Err, "sdd prompt discovery: failed to marshal JSON context: %v\n", err)
			return 1
		}
		fprintln(env.Out, string(data))
		return 0
	}

	fprint(env.Out, promptText)
	return 0
}

func buildDiscoveryPrompt(
	taskID, taskDir, typology, root string,
	detectedLangs []string,
	manifests []string,
	buildCmd, testCmd, lintCmd string,
	stagedContext []string,
	roleSkill, reqs, plan string,
) string {
	var sb strings.Builder

	targetHeader := filepath.Base(root)
	if taskID != "" {
		targetHeader = fmt.Sprintf("Task %s (%s)", taskID, taskDir)
	}

	fmt.Fprintf(&sb, "# Handover: Codebase Discovery & Forensic Extraction for %s\n\n", targetHeader)
	sb.WriteString("You are acting as the **Codebase Analyst & Forensic Technical Auditor** for this project.\n")
	sb.WriteString("Your mandate is a **read-only forensic exploration** of an existing codebase.\n")
	sb.WriteString("You do **not** write application code; you map the codebase into structured specifications.\n\n")

	sb.WriteString("## SDD Anthem & Principles\n")
	sb.WriteString("> **ALWAYS BE DETERMINISTIC. When a decision is needed, STOP and ask the human. Never guess. Never silently choose.**\n\n")

	sb.WriteString("1. **Read-Only Forensic Audit:** Do NOT write or modify application code. You are mapping architecture, design patterns, schemas, and technical debt.\n")
	sb.WriteString("2. **Deterministic Documentation:** Record verified facts with exact dates and inspection paths in living documents.\n")
	sb.WriteString("3. **Standard Library Purity & Dependency Governance:** Flag third-party packages, libraries, and licenses for human decision approval (`D-xxx`).\n\n")

	sb.WriteString("## Canonical Role Skill: Staff Reverse-Engineer & Technical Auditor (codebase-analyst)\n")
	sb.WriteString(strings.TrimSpace(roleSkill))
	sb.WriteString("\n\n")

	sb.WriteString("## 4-Pillar Progressive Extraction Checklist\n")
	sb.WriteString("Execute this progressive extraction checklist systematically:\n\n")
	sb.WriteString("### 1. AST & Module Map (Architecture & Topology)\n")
	sb.WriteString("- [ ] Directory topology and entry points (e.g. `cmd/`, `src/`, `main.go`, `index.ts`, `App.tsx`).\n")
	sb.WriteString("- [ ] Package manifest audit (`go.mod`, `package.json`, `pom.xml`, `requirements.txt`).\n")
	sb.WriteString("- [ ] Build systems, compilation targets, and local developer toolchains (`Makefile`, scripts, Dockerfiles).\n")
	sb.WriteString("- [ ] Subsystem boundaries, internal modules, and external API integrations.\n\n")

	sb.WriteString("### 2. Design System & UI Patterns (Frontend & Layout)\n")
	sb.WriteString("- [ ] UI component hierarchy, layout containers, and view routing.\n")
	sb.WriteString("- [ ] Styling framework: Tailwind, CSS Modules, Styled Components, Sass/SCSS, Vanilla CSS.\n")
	sb.WriteString("- [ ] Design tokens: font families, typography scale, color palettes, spacing units, responsive breakpoints.\n")
	sb.WriteString("- [ ] Asset pipelines, icon libraries, and visual design conventions.\n\n")

	sb.WriteString("### 3. Data & State Modeling (Data Flow & Storage)\n")
	sb.WriteString("- [ ] Persistence engines, database schemas, migration files, and ORM / ODM models.\n")
	sb.WriteString("- [ ] Client/server state management (Redux, Zustand, React Query, Pinia, internal buses).\n")
	sb.WriteString("- [ ] Caching layers (Redis, Memcached, in-memory LRU) and event queues / brokers (Kafka, RabbitMQ).\n")
	sb.WriteString("- [ ] Data validation schemas and serialization contracts (JSON Schema, Protobuf, Zod).\n\n")

	sb.WriteString("### 4. Anti-Patterns & Technical Debt (Hygiene & Pitfalls)\n")
	sb.WriteString("- [ ] Fragile or unmaintained third-party dependencies (outdated versions, deprecated APIs).\n")
	sb.WriteString("- [ ] Unhandled error patterns, swallowed exceptions, empty catch blocks, unchecked panics.\n")
	sb.WriteString("- [ ] Concurrency risks: unowned goroutines, shared mutable state without locks, race conditions.\n")
	sb.WriteString("- [ ] Security hazards: hardcoded secrets/tokens, missing input sanitization, insecure defaults.\n")
	sb.WriteString("- [ ] Undocumented quirks or tribal knowledge to seed the Pitfalls log.\n\n")

	sb.WriteString("## Brownfield Living Document Synthesis Protocol\n")
	sb.WriteString("Synthesize forensic findings directly into the project's living documents:\n")
	sb.WriteString("1. **`TECH_STACK.md`:**\n")
	sb.WriteString("   - Architecture & Directory Layout (§3, §4)\n")
	sb.WriteString("   - Decision Register (`D-xxx`) for discovered frameworks, runtimes, and libraries (§6)\n")
	sb.WriteString("   - Verified-Facts Log with dates and inspection sources (§7)\n")
	sb.WriteString("   - Dependency Register for package and external service contracts (§8)\n")
	sb.WriteString("2. **`SKILLS.md`:**\n")
	sb.WriteString("   - Discovered language standards, naming conventions, and idioms (§1)\n")
	sb.WriteString("   - Discovered quirks, gotchas, and architectural traps -> Pitfalls Log (§7)\n")
	sb.WriteString("3. **`ROADMAP.md`:**\n")
	sb.WriteString("   - Discovered technical debt, unmapped subsystems, and remediation items -> Parking Lot (§7)\n\n")

	appendDiscoveryContextAndTask(&sb, root, detectedLangs, manifests, buildCmd, testCmd, lintCmd, stagedContext, taskID, taskDir, typology, reqs, plan)

	return sb.String()
}

func buildDiscoveryEnsemblePrompt(
	taskID, taskDir, typology, root string,
	detectedLangs []string,
	manifests []string,
	buildCmd, testCmd, lintCmd string,
	stagedContext []string,
	roleSkill, challengerSkill, bundleDir, reqs, plan string,
) string {
	var sb strings.Builder

	targetHeader := filepath.Base(root)
	if taskID != "" {
		targetHeader = fmt.Sprintf("Task %s (%s)", taskID, taskDir)
	}

	fmt.Fprintf(&sb, "# Handover: Adversarial Discovery Ensemble for %s\n\n", targetHeader)
	sb.WriteString("You are orchestrating an **Adversarial Multi-Agent Discovery Ensemble** for brownfield exploration.\n")
	sb.WriteString("To eliminate single-agent blind spots, discovery is partitioned into two specialized roles:\n")
	sb.WriteString("1. **Primary Analyst (`codebase-analyst`):** Conducts 4-pillar forensic extraction and living document drafting.\n")
	sb.WriteString("2. **Adversarial Challenger (`discovery-challenger`):** Operates under stateless reviewer isolation to audit findings across 7 blind-spot vectors.\n\n")

	sb.WriteString("## SDD Anthem & Principles\n")
	sb.WriteString("> **ALWAYS BE DETERMINISTIC. When a decision is needed, STOP and ask the human. Never guess. Never silently choose.**\n\n")

	sb.WriteString("1. **Strictly Read-Only:** Neither agent mutates application source code during Phase Discovery.\n")
	sb.WriteString("2. **Stateless Reviewer Isolation:** The `discovery-challenger` is dispatched in a fresh, isolated context (tabula rasa) without conversational anchoring to the analyst.\n")
	sb.WriteString("3. **Evidence-Based Grounding:** Every discovery claim and adversarial challenge must cite exact repository-relative file paths and line numbers.\n")
	sb.WriteString("4. **Deterministic Reconciliation:** Discrepancies between analyst and challenger findings are reconciled before Gate 1 approval.\n\n")

	if bundleDir != "" {
		sb.WriteString("## Stateless Review Bundle (.sdd/review/)\n")
		sb.WriteString("An isolated, frozen review bundle has been staged for the challenger with zero conversational history:\n")
		fmt.Fprintf(&sb, "- **Bundle Directory:** `%s`\n", filepath.ToSlash(bundleDir))
		fmt.Fprintf(&sb, "- **Staged Findings:** `%s/findings.md`\n", filepath.ToSlash(bundleDir))
		fmt.Fprintf(&sb, "- **Challenger Rubric:** `%s/rubric.md`\n", filepath.ToSlash(bundleDir))
		fmt.Fprintf(&sb, "- **Bundle Manifest:** `%s/manifest.json`\n\n", filepath.ToSlash(bundleDir))
	}

	sb.WriteString("## 1. Primary Analyst Briefing (codebase-analyst)\n")
	sb.WriteString("The primary analyst executes the 4-pillar progressive extraction checklist:\n")
	for _, item := range getDiscoveryChecklist() {
		fmt.Fprintf(&sb, "- %s\n", item)
	}
	sb.WriteString("\n### Canonical Role Skill: Codebase Analyst\n")
	sb.WriteString(strings.TrimSpace(roleSkill))
	sb.WriteString("\n\n")

	sb.WriteString("## 2. Adversarial Challenger Rubric (discovery-challenger)\n")
	sb.WriteString("The discovery challenger interrogates the draft findings against 7 comprehensive blind-spot vectors:\n")
	for _, v := range getDiscoveryBlindSpotVectors() {
		fmt.Fprintf(&sb, "- %s\n", v)
	}
	sb.WriteString("\n### Canonical Role Skill: Discovery Challenger\n")
	sb.WriteString(strings.TrimSpace(challengerSkill))
	sb.WriteString("\n\n")

	sb.WriteString("## 3. Deterministic Reconciliation Protocol (Gate 1 Exit Criteria)\n")
	sb.WriteString("Before presenting the brownfield discovery specifications to the human at Gate 1, the Task Coordinator executes this reconciliation:\n")
	for _, r := range getDiscoveryReconciliationChecklist() {
		fmt.Fprintf(&sb, "- [ ] %s\n", r)
	}
	sb.WriteString("\n")

	appendDiscoveryContextAndTask(&sb, root, detectedLangs, manifests, buildCmd, testCmd, lintCmd, stagedContext, taskID, taskDir, typology, reqs, plan)

	return sb.String()
}

func appendDiscoveryContextAndTask(
	sb *strings.Builder,
	root string,
	detectedLangs []string,
	manifests []string,
	buildCmd, testCmd, lintCmd string,
	stagedContext []string,
	taskID, taskDir, typology, reqs, plan string,
) {
	sb.WriteString("## Project Context & Metadata\n")
	fmt.Fprintf(sb, "- **Project Root:** `%s`\n", filepath.ToSlash(root))
	if len(detectedLangs) > 0 {
		fmt.Fprintf(sb, "- **Detected Languages:** %s\n", strings.Join(detectedLangs, ", "))
	}
	if len(manifests) > 0 {
		fmt.Fprintf(sb, "- **Detected Manifests:** %s\n", strings.Join(manifests, ", "))
	}
	if buildCmd != "" {
		fmt.Fprintf(sb, "- **Suggested Build:** `%s`\n", buildCmd)
	}
	if testCmd != "" {
		fmt.Fprintf(sb, "- **Suggested Test:** `%s`\n", testCmd)
	}
	if lintCmd != "" {
		fmt.Fprintf(sb, "- **Suggested Lint:** `%s`\n", lintCmd)
	}
	if len(stagedContext) > 0 {
		sb.WriteString("- **Staged Context Artifacts:**\n")
		for _, s := range stagedContext {
			fmt.Fprintf(sb, "  - `%s`\n", s)
		}
	}
	sb.WriteString("\n")

	if taskID != "" {
		sb.WriteString("## Discovery Task Details\n")
		fmt.Fprintf(sb, "- **Task ID:** %s\n", taskID)
		fmt.Fprintf(sb, "- **Task Directory:** %s\n", taskDir)
		if typology != "" {
			fmt.Fprintf(sb, "- **Typology:** %s\n", typology)
		}
		if reqs != "" {
			sb.WriteString("\n### Requirements (`requirements.md`)\n")
			sb.WriteString(strings.TrimSpace(reqs))
			sb.WriteString("\n")
		}
		if plan != "" {
			sb.WriteString("\n### Implementation Plan (`plan.md`)\n")
			sb.WriteString(strings.TrimSpace(plan))
			sb.WriteString("\n")
		}
	}
}
