package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vikashkumar21/sdd/internal/context"
	"github.com/vikashkumar21/sdd/internal/detect"
	"github.com/vikashkumar21/sdd/internal/roadmap"
	"github.com/vikashkumar21/sdd/internal/role"
)

// DiscoveryPromptContext represents the structured context bundle emitted with --json.
type DiscoveryPromptContext struct {
	TaskID            string   `json:"taskId,omitempty"`
	TaskDir           string   `json:"taskDir,omitempty"`
	Typology          string   `json:"typology,omitempty"`
	ProjectRoot       string   `json:"projectRoot"`
	RoleSkill         string   `json:"roleSkill"`
	DetectedLanguages []string `json:"detectedLanguages"`
	LivingDocs        []string `json:"livingDocs"`
	Checklist         []string `json:"checklist"`
	StagedContext     []string `json:"stagedContext,omitempty"`
	Requirements      string   `json:"requirements,omitempty"`
	Plan              string   `json:"plan,omitempty"`
	Prompt            string   `json:"prompt"`
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
	jsonFlag := fs.Bool("json", false, "output machine-readable discovery context bundle as JSON")

	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return parseExit(err)
	}

	if len(pos) > 1 {
		fprintln(env.Err, "usage: sdd prompt discovery [T-xxx] [--task T-xxx] [--dir <project>] [--json]")
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

	// 3. Resolve codebase-analyst role skill via two-tier storage
	roleSkill, err := role.ResolveSkill(root, "codebase-analyst", nil)
	if err != nil {
		fprintf(env.Err, "sdd prompt discovery: failed to resolve codebase-analyst role skill: %v\n", err)
		return 1
	}

	// 4. Assemble prompt text
	promptText := buildDiscoveryPrompt(
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

	sb.WriteString("## Project Context & Metadata\n")
	fmt.Fprintf(&sb, "- **Project Root:** `%s`\n", filepath.ToSlash(root))
	if len(detectedLangs) > 0 {
		fmt.Fprintf(&sb, "- **Detected Languages:** %s\n", strings.Join(detectedLangs, ", "))
	}
	if len(manifests) > 0 {
		fmt.Fprintf(&sb, "- **Detected Manifests:** %s\n", strings.Join(manifests, ", "))
	}
	if buildCmd != "" {
		fmt.Fprintf(&sb, "- **Suggested Build:** `%s`\n", buildCmd)
	}
	if testCmd != "" {
		fmt.Fprintf(&sb, "- **Suggested Test:** `%s`\n", testCmd)
	}
	if lintCmd != "" {
		fmt.Fprintf(&sb, "- **Suggested Lint:** `%s`\n", lintCmd)
	}
	if len(stagedContext) > 0 {
		sb.WriteString("- **Staged Context Artifacts:**\n")
		for _, s := range stagedContext {
			fmt.Fprintf(&sb, "  - `%s`\n", s)
		}
	}
	sb.WriteString("\n")

	if taskID != "" {
		sb.WriteString("## Discovery Task Details\n")
		fmt.Fprintf(&sb, "- **Task ID:** %s\n", taskID)
		fmt.Fprintf(&sb, "- **Task Directory:** %s\n", taskDir)
		if typology != "" {
			fmt.Fprintf(&sb, "- **Typology:** %s\n", typology)
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

	return sb.String()
}
