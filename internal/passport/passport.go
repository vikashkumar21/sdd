// Package passport generates deterministic, high-density project passport micro-indexes
// for progressive disclosure, orienting AI agents with minimal context tokens.
package passport

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/vikashkumar21/sdd/internal/status"
)

// Budget ceilings for the rendered Markdown passport micro-index.
const (
	MaxMarkdownBytes  = 2000
	MaxMarkdownTokens = 500
)

// Passport synthesizes the core orientation metadata of an SDD project.
type Passport struct {
	ProjectName     string                `json:"project_name"`
	Mission         string                `json:"mission"`
	Stack           string                `json:"stack"`
	ActiveMilestone *status.MilestoneInfo `json:"active_milestone,omitempty"`
	ActiveTasks     []status.TaskInfo     `json:"active_tasks,omitempty"`
	NextTask        *status.TaskInfo      `json:"next_task,omitempty"`
	ActiveRules     []RuleSummary         `json:"active_rules,omitempty"`
	LivingDocs      []DocPointer          `json:"living_docs"`
	GeneratedAt     time.Time             `json:"generated_at"`
	EstimatedTokens int                   `json:"estimated_tokens"`
	ByteSize        int                   `json:"byte_size"`
}

// RuleSummary captures a human-feedback rule ID and concise summary.
type RuleSummary struct {
	ID      string `json:"id"`
	Summary string `json:"summary"`
}

// DocPointer references a living document and its purpose.
type DocPointer struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Description string `json:"description"`
}

var (
	titleRe         = regexp.MustCompile(`(?i)^#\s+(?:ROADMAP|MISSION)\s*—\s*(.+)`)
	oneLinerHdrRe   = regexp.MustCompile(`(?i)^##\s+1\.\s+One-liner`)
	stackHdrRe      = regexp.MustCompile(`(?i)^##\s+1\.\s+Stack at a glance`)
	feedbackRulesRe = regexp.MustCompile(`(?i)^##\s+.*Human feedback rules`)
	rulesHdrRe      = regexp.MustCompile(`(?i)^###\s+Register`)
	sectionHdrRe    = regexp.MustCompile(`(?i)^##\s+`)
	subHdrRe        = regexp.MustCompile(`(?i)^###\s+`)
)

func readBoundedFile(path string, maxBytes int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return io.ReadAll(io.LimitReader(f, maxBytes))
}

// EstimateTokens calculates an approximate token count from UTF-8 string content
// using deterministic character heuristics (~3.7 chars per token for markdown/code).
func EstimateTokens(s string) int {
	if s == "" {
		return 0
	}
	runes := utf8.RuneCountInString(s)
	tokens := (runes * 10) / 37
	if tokens == 0 && runes > 0 {
		return 1
	}
	return tokens
}

// Generate synthesizes a deterministic project passport micro-index.
// The provided clock 'now' is injected for byte-identical determinism;
// if now.IsZero(), it normalizes to time.Unix(0, 0).UTC().
func Generate(root string, now time.Time) (*Passport, error) {
	if now.IsZero() {
		now = time.Unix(0, 0).UTC()
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolving root path: %w", err)
	}

	// 1. Scan project status from ROADMAP.md and workspace tasks
	ps, err := status.Scan(absRoot)
	if err != nil {
		return nil, fmt.Errorf("scanning status: %w", err)
	}

	// 2. Extract ProjectName
	projectName := extractProjectName(absRoot)

	// 3. Extract Mission from MISSION.md
	mission := extractMission(absRoot)

	// 4. Extract Stack from TECH_STACK.md (polyglot-aware)
	stack := extractStack(absRoot)

	// 5. Extract ActiveRules from SKILLS.md Register table
	activeRules := extractActiveRules(absRoot)

	// 6. Assemble living document pointers (only include files that exist on disk)
	candidateDocs := []DocPointer{
		{
			Name:        "MISSION.md",
			Path:        "MISSION.md",
			Description: "Project mission, scope, principles, non-goals",
		},
		{
			Name:        "TECH_STACK.md",
			Path:        "TECH_STACK.md",
			Description: "Stack, architecture, Decision register §6, verified facts §7",
		},
		{
			Name:        "ROADMAP.md",
			Path:        "ROADMAP.md",
			Description: "Milestones, tasks, statuses, parking lot",
		},
		{
			Name:        "SKILLS.md",
			Path:        "SKILLS.md",
			Description: "Language standards §1, playbooks, feedback rules §8",
		},
		{
			Name:        "README.md",
			Path:        "README.md",
			Description: "Current public-facing overview and quickstart",
		},
	}

	var livingDocs []DocPointer
	for _, doc := range candidateDocs {
		if _, err := os.Stat(filepath.Join(absRoot, doc.Path)); err == nil {
			livingDocs = append(livingDocs, doc)
		}
	}

	p := &Passport{
		ProjectName:     projectName,
		Mission:         mission,
		Stack:           stack,
		ActiveMilestone: ps.ActiveMilestone,
		ActiveTasks:     ps.ActiveTasks,
		NextTask:        ps.NextTask,
		ActiveRules:     activeRules,
		LivingDocs:      livingDocs,
		GeneratedAt:     now,
	}

	rendered := p.RenderMarkdown()
	p.ByteSize = len(rendered)
	p.EstimatedTokens = EstimateTokens(rendered)

	return p, nil
}

func extractProjectName(root string) string {
	// Try manifest.json first (bounded to 16KB)
	manifestPath := filepath.Join(root, ".sdd", "manifest.json")
	if data, err := readBoundedFile(manifestPath, 16*1024); err == nil {
		var manifest struct {
			Answers struct {
				Project string `json:"project"`
			} `json:"answers"`
		}
		if json.Unmarshal(data, &manifest) == nil && strings.TrimSpace(manifest.Answers.Project) != "" {
			return strings.TrimSpace(manifest.Answers.Project)
		}
	}

	// Try ROADMAP.md header (bounded to 32KB)
	roadmapPath := filepath.Join(root, "ROADMAP.md")
	if data, err := readBoundedFile(roadmapPath, 32*1024); err == nil {
		scanner := bufio.NewScanner(strings.NewReader(normalizeCRLF(string(data))))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if m := titleRe.FindStringSubmatch(line); len(m) > 1 {
				return strings.TrimSpace(m[1])
			}
			if strings.HasPrefix(line, "#") {
				break
			}
		}
	}

	// Try MISSION.md header (bounded to 16KB)
	missionPath := filepath.Join(root, "MISSION.md")
	if data, err := readBoundedFile(missionPath, 16*1024); err == nil {
		scanner := bufio.NewScanner(strings.NewReader(normalizeCRLF(string(data))))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if m := titleRe.FindStringSubmatch(line); len(m) > 1 {
				return strings.TrimSpace(m[1])
			}
			if strings.HasPrefix(line, "#") {
				break
			}
		}
	}

	return filepath.Base(root)
}

func extractMission(root string) string {
	missionPath := filepath.Join(root, "MISSION.md")
	data, err := readBoundedFile(missionPath, 16*1024)
	if err != nil {
		// Fallback to manifest if available
		manifestPath := filepath.Join(root, ".sdd", "manifest.json")
		if mdata, merr := readBoundedFile(manifestPath, 16*1024); merr == nil {
			var manifest struct {
				Answers struct {
					Purpose string `json:"purpose"`
				} `json:"answers"`
			}
			if json.Unmarshal(mdata, &manifest) == nil && strings.TrimSpace(manifest.Answers.Purpose) != "" {
				res := strings.TrimSpace(manifest.Answers.Purpose)
				if utf8.RuneCountInString(res) > 200 {
					res = string([]rune(res)[:197]) + "..."
				}
				return res
			}
		}
		return ""
	}

	content := normalizeCRLF(string(data))
	scanner := bufio.NewScanner(strings.NewReader(content))
	inOneLiner := false
	var para []string

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if oneLinerHdrRe.MatchString(trimmed) {
			inOneLiner = true
			continue
		}

		if inOneLiner {
			if strings.HasPrefix(trimmed, "## ") {
				break
			}
			// Skip blockquotes or status markers if placed directly under one-liner header
			if strings.HasPrefix(trimmed, ">") || strings.HasPrefix(trimmed, "Status:") {
				continue
			}
			if trimmed == "" {
				if len(para) > 0 {
					break
				}
				continue
			}
			para = append(para, trimmed)
		}
	}

	result := strings.Join(para, " ")
	if utf8.RuneCountInString(result) > 200 {
		result = string([]rune(result)[:197]) + "..."
	}
	return result
}

func extractStack(root string) string {
	techPath := filepath.Join(root, "TECH_STACK.md")
	data, err := readBoundedFile(techPath, 32*1024)
	if err != nil {
		// Fallback to manifest if available
		manifestPath := filepath.Join(root, ".sdd", "manifest.json")
		if mdata, merr := readBoundedFile(manifestPath, 16*1024); merr == nil {
			var manifest struct {
				Answers struct {
					Languages []string `json:"languages"`
					Build     string   `json:"build"`
				} `json:"answers"`
			}
			if json.Unmarshal(mdata, &manifest) == nil {
				var parts []string
				if len(manifest.Answers.Languages) > 0 {
					parts = append(parts, "Languages: "+strings.Join(manifest.Answers.Languages, ", "))
				}
				if manifest.Answers.Build != "" {
					parts = append(parts, "Build: "+manifest.Answers.Build)
				}
				return strings.Join(parts, " | ")
			}
		}
		return ""
	}

	content := normalizeCRLF(string(data))
	scanner := bufio.NewScanner(strings.NewReader(content))
	inTable := false
	var languages []string
	var buildTool string
	var testTool string
	var lintTool string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if stackHdrRe.MatchString(line) {
			inTable = true
			continue
		}
		if inTable {
			if strings.HasPrefix(line, "## ") {
				break
			}
			if !strings.HasPrefix(line, "|") {
				continue
			}
			parts := splitTableCells(line)
			if len(parts) < 2 {
				continue
			}
			layer := strings.ToLower(strings.TrimSpace(parts[0]))
			choice := strings.TrimSpace(parts[1])

			if strings.Contains(layer, "language") && !strings.Contains(layer, "version") {
				if choice != "" && choice != "Choice" {
					languages = append(languages, choice)
				}
			} else if layer == "build" && buildTool == "" {
				if choice != "" && choice != "Choice" {
					buildTool = choice
				}
			} else if layer == "test" && testTool == "" {
				if choice != "" && choice != "Choice" {
					testTool = choice
				}
			} else if layer == "lint" && lintTool == "" {
				if choice != "" && choice != "Choice" {
					lintTool = choice
				}
			}
		}
	}

	var summaryParts []string
	if len(languages) > 0 {
		summaryParts = append(summaryParts, strings.Join(languages, "; "))
	}
	if buildTool != "" {
		summaryParts = append(summaryParts, buildTool)
	}
	if testTool != "" {
		summaryParts = append(summaryParts, testTool)
	}
	if lintTool != "" {
		summaryParts = append(summaryParts, lintTool)
	}

	return strings.Join(summaryParts, " | ")
}

func extractActiveRules(root string) []RuleSummary {
	skillsPath := filepath.Join(root, "SKILLS.md")
	data, err := readBoundedFile(skillsPath, 64*1024)
	if err != nil {
		return nil
	}

	content := normalizeCRLF(string(data))
	scanner := bufio.NewScanner(strings.NewReader(content))
	inHumanFeedback := false
	inRegister := false
	var rules []RuleSummary

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if feedbackRulesRe.MatchString(line) {
			inHumanFeedback = true
			continue
		}

		if inHumanFeedback && rulesHdrRe.MatchString(line) {
			inRegister = true
			continue
		}

		if inRegister {
			if sectionHdrRe.MatchString(line) || subHdrRe.MatchString(line) {
				break
			}
			if !strings.HasPrefix(line, "|") {
				continue
			}
			cells := splitTableCells(line)
			if len(cells) < 7 {
				continue
			}

			id := strings.TrimSpace(cells[0])
			rule := strings.TrimSpace(cells[2])
			statusCol := strings.TrimSpace(cells[6])

			// Skip header, divider, empty, or placeholder rows
			if id == "ID" || strings.HasPrefix(id, "-") || id == "" || strings.Contains(id, "(none yet)") {
				continue
			}

			// Skip superseded rules
			if strings.HasPrefix(strings.ToLower(statusCol), "superseded") {
				continue
			}

			// Bound summary in struct to <= 100 runes
			if utf8.RuneCountInString(rule) > 100 {
				rule = string([]rune(rule)[:97]) + "..."
			}

			rules = append(rules, RuleSummary{
				ID:      id,
				Summary: rule,
			})
		}
	}

	return rules
}

// RenderMarkdown formats the passport into a compact markdown micro-index,
// applying deterministic compaction to strictly satisfy byte and token ceilings.
func (p *Passport) RenderMarkdown() string {
	// Try standard rendering (Level 0)
	out := p.renderMarkdownLevel(0)
	if len(out) <= MaxMarkdownBytes && EstimateTokens(out) <= MaxMarkdownTokens {
		return out
	}

	// Compaction Level 1: truncate rule summaries to 40 characters
	out = p.renderMarkdownLevel(1)
	if len(out) <= MaxMarkdownBytes && EstimateTokens(out) <= MaxMarkdownTokens {
		return out
	}

	// Compaction Level 2: truncate stack and summarize rules by ID only
	out = p.renderMarkdownLevel(2)
	if len(out) <= MaxMarkdownBytes && EstimateTokens(out) <= MaxMarkdownTokens {
		return out
	}

	// Compaction Level 3: maximum compression
	out = p.renderMarkdownLevel(3)
	if len(out) <= MaxMarkdownBytes && EstimateTokens(out) <= MaxMarkdownTokens {
		return out
	}

	// Step (c) Hard ceiling: truncate strictly on UTF-8 rune boundaries until both bounds are met
	runes := []rune(out)
	for len(runes) > 0 && (len(string(runes)) > MaxMarkdownBytes || EstimateTokens(string(runes)) > MaxMarkdownTokens) {
		runes = runes[:len(runes)-1]
	}
	return string(runes)
}

func (p *Passport) renderMarkdownLevel(level int) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "# %s — Project Passport\n\n", p.ProjectName)

	if p.Mission != "" {
		mission := p.Mission
		if level >= 2 && len([]rune(mission)) > 120 {
			mission = string([]rune(mission)[:117]) + "..."
		}
		fmt.Fprintf(&sb, "**Mission:** %s\n", mission)
	}

	if p.Stack != "" {
		stack := p.Stack
		if level >= 2 {
			// Compact stack to primary language
			parts := strings.Split(stack, "|")
			if len(parts) > 0 {
				stack = strings.TrimSpace(parts[0])
			}
		}
		fmt.Fprintf(&sb, "**Stack:** %s\n", stack)
	}

	if p.ActiveMilestone != nil {
		fmt.Fprintf(&sb, "**Active Milestone:** %s — %s (%d%% complete)\n",
			p.ActiveMilestone.ID, p.ActiveMilestone.Title, p.ActiveMilestone.Percent)
	}

	sb.WriteString("\n")

	// Active Tasks or Next Task
	if len(p.ActiveTasks) > 0 {
		sb.WriteString("## Active Tasks\n")
		taskLimit := 3
		for i, t := range p.ActiveTasks {
			if i >= taskLimit {
				remaining := len(p.ActiveTasks) - taskLimit
				fmt.Fprintf(&sb, "- ... (+%d more active tasks)\n", remaining)
				break
			}
			title := t.Title
			if level >= 1 && len([]rune(title)) > 50 {
				title = string([]rune(title)[:47]) + "..."
			}
			fmt.Fprintf(&sb, "- [%s] %s (%s)\n", t.ID, title, t.Status)
		}
	} else if p.NextTask != nil {
		sb.WriteString("## Next Task\n")
		title := p.NextTask.Title
		if level >= 1 && len([]rune(title)) > 50 {
			title = string([]rune(title)[:47]) + "..."
		}
		fmt.Fprintf(&sb, "- [%s] %s (Not started)\n", p.NextTask.ID, title)
	} else {
		sb.WriteString("## Active Tasks\n- (none active; all roadmap tasks complete)\n")
	}

	// Active Feedback Rules
	if len(p.ActiveRules) > 0 {
		sb.WriteString("\n## Active Feedback Rules (SKILLS.md §8)\n")
		ruleLimit := 5
		for i, r := range p.ActiveRules {
			if i >= ruleLimit {
				remaining := len(p.ActiveRules) - ruleLimit
				fmt.Fprintf(&sb, "- ... (+%d more rules in SKILLS.md §8)\n", remaining)
				break
			}

			summary := r.Summary
			maxChars := 60
			if level == 1 {
				maxChars = 40
			} else if level >= 2 {
				// Level 2+ rule IDs only or very short
				fmt.Fprintf(&sb, "- **%s**\n", r.ID)
				continue
			}

			if len([]rune(summary)) > maxChars {
				summary = string([]rune(summary)[:maxChars-3]) + "..."
			}
			fmt.Fprintf(&sb, "- **%s**: %s\n", r.ID, summary)
		}
	}

	// Living Documents
	if len(p.LivingDocs) > 0 {
		sb.WriteString("\n## Living Documents (On-Demand)\n")
		for _, doc := range p.LivingDocs {
			if level >= 3 {
				fmt.Fprintf(&sb, "- `%s`\n", doc.Name)
			} else {
				fmt.Fprintf(&sb, "- `%s` — %s\n", doc.Name, doc.Description)
			}
		}
	}

	return strings.TrimRight(sb.String(), "\n") + "\n"
}

// RenderJSON serializes the passport into indented JSON.
func (p *Passport) RenderJSON() (string, error) {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshalling passport JSON: %w", err)
	}
	return string(data), nil
}

// Validate inspects the passport against the token and byte budget limits.
func (p *Passport) Validate() []string {
	var warnings []string
	if p.EstimatedTokens > MaxMarkdownTokens {
		warnings = append(warnings, fmt.Sprintf("passport estimated tokens %d exceeds budget limit %d", p.EstimatedTokens, MaxMarkdownTokens))
	}
	if p.ByteSize > MaxMarkdownBytes {
		warnings = append(warnings, fmt.Sprintf("passport byte size %d exceeds budget limit %d", p.ByteSize, MaxMarkdownBytes))
	}
	return warnings
}

func normalizeCRLF(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

func splitTableCells(line string) []string {
	line = strings.Trim(line, "|")
	raw := strings.Split(line, "|")
	var out []string
	for _, r := range raw {
		out = append(out, strings.TrimSpace(r))
	}
	return out
}
