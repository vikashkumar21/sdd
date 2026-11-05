// Package status provides fast scanning, progress analysis, gate resolution,
// and formatted summaries of an SDD project's roadmap and feature tasks.
package status

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/vikashkumar21/sdd/internal/roadmap"
)

// MilestoneInfo records the progress and state of one roadmap milestone.
type MilestoneInfo struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	TasksTotal int    `json:"tasks_total"`
	TasksDone  int    `json:"tasks_done"`
	Percent    int    `json:"percent"`
	Status     string `json:"status"` // "Done", "In progress", "Not started"
}

// GateInfo describes the gate status and expected human action for a task.
type GateInfo struct {
	TaskID string `json:"task_id"`
	Gate   string `json:"gate"`   // "GATE 1", "GATE 2", or ""
	Name   string `json:"name"`   // "Approve requirements and plan", "Review and commit"
	Status string `json:"status"` // "Pending approval", "Awaiting review", "In progress"
	Action string `json:"action"` // e.g. "Reply 'APPROVED T-027'"
}

// TaskInfo represents a task with its SDD phase and gate details.
type TaskInfo struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	Milestone      string    `json:"milestone"`
	Status         string    `json:"status"`
	Phase          string    `json:"phase"`
	Folder         string    `json:"folder,omitempty"`
	Typology       string    `json:"typology,omitempty"`
	Gate           *GateInfo `json:"gate,omitempty"`
	ReviewBlockers int       `json:"review_blockers,omitempty"`
}

// TaskSummary aggregates counts across all roadmap tasks.
type TaskSummary struct {
	Total      int `json:"total"`
	Done       int `json:"done"`
	InProgress int `json:"in_progress"`
	Blocked    int `json:"blocked"`
	NotStarted int `json:"not_started"`
}

// ProjectStatus is the complete status of an SDD repository.
type ProjectStatus struct {
	Root            string          `json:"root"`
	ActiveMilestone *MilestoneInfo  `json:"active_milestone,omitempty"`
	Milestones      []MilestoneInfo `json:"milestones"`
	TaskSummary     TaskSummary     `json:"task_summary"`
	ActiveTasks     []TaskInfo      `json:"active_tasks"`
	NextTask        *TaskInfo       `json:"next_task,omitempty"`
	NextGate        *GateInfo       `json:"next_gate,omitempty"`
	Warnings        []string        `json:"warnings,omitempty"`
}

var (
	milestoneRowRe = regexp.MustCompile(`^\|\s*(M\d+)\s*\|\s*([^|]+)\|`)
	sectionHdrRe   = regexp.MustCompile(`(?i)^##\s+\d+\.\s*(.+)`)
	milestoneHdrRe = regexp.MustCompile(`(?i)^###\s*(M\d+)\s*(?:[—\-:]\s*(.+))?`)
	factHdrRe      = regexp.MustCompile(`(?i)^\|\s*Date\s*\|\s*Fact\s*\|\s*Source\s*\|\s*Re-verify at\s*\|`)
	taskRefRe      = regexp.MustCompile(`\bT-\d{3}(?:\.\d+)?\b`)
	milestoneRefRe = regexp.MustCompile(`\bM\d+\b`)
)

// Scan reads and inspects the SDD project at root.
func Scan(root string) (*ProjectStatus, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolving root path: %w", err)
	}

	roadmapPath := filepath.Join(absRoot, "ROADMAP.md")
	roadmapBytes, err := os.ReadFile(roadmapPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("ROADMAP.md not found in %s", absRoot)
		}
		return nil, fmt.Errorf("reading ROADMAP.md: %w", err)
	}
	roadmapContent := string(roadmapBytes)

	// 1. Parse tasks from ROADMAP.md
	tasks := roadmap.Parse(roadmapContent)

	// 2. Parse milestone titles from Section 3 table and Section 4 headers
	milestoneTitles := parseMilestoneTitles(roadmapContent)

	// 3. Discover feature folders
	featureFolders := discoverFeatureFolders(absRoot)

	// 4. Group tasks by milestone and calculate task summary
	summary := TaskSummary{Total: len(tasks)}
	tasksByMilestone := make(map[string][]roadmap.Task)
	var activeTasks []TaskInfo
	var nextNotStarted *TaskInfo
	completedMilestones := make(map[string]bool)

	for _, t := range tasks {
		normStatus := roadmap.Normalize(t.Status)
		kind := roadmap.Kind(t.Status)
		mID := strings.ToUpper(t.Milestone)
		if mID == "" {
			mID = "OTHER"
		}
		tasksByMilestone[mID] = append(tasksByMilestone[mID], t)

		switch kind {
		case "done":
			summary.Done++
		case "blocked":
			summary.Blocked++
			tInfo := buildTaskInfo(absRoot, t, normStatus, kind, featureFolders)
			activeTasks = append(activeTasks, tInfo)
		case "specifying", "awaiting plan approval", "implementing", "awaiting review":
			summary.InProgress++
			tInfo := buildTaskInfo(absRoot, t, normStatus, kind, featureFolders)
			activeTasks = append(activeTasks, tInfo)
		case "not started":
			summary.NotStarted++
			if nextNotStarted == nil {
				nextNotStarted = &TaskInfo{
					ID:        t.ID,
					Title:     t.Title,
					Milestone: t.Milestone,
					Status:    normStatus,
					Phase:     "Not started",
				}
			}
		default:
			summary.NotStarted++
		}
	}

	// 5. Build Milestones list
	milestoneIDs := orderedMilestoneIDs(milestoneTitles, tasksByMilestone)
	var milestones []MilestoneInfo
	var activeMilestone *MilestoneInfo

	for _, mID := range milestoneIDs {
		mTasks := tasksByMilestone[mID]
		total := len(mTasks)
		done := 0
		hasInProgress := false

		for _, t := range mTasks {
			kind := roadmap.Kind(t.Status)
			if kind == "done" {
				done++
			} else if kind != "not started" {
				hasInProgress = true
			}
		}

		percent := 0
		if total > 0 {
			percent = (done * 100) / total
		}

		mStatus := "Not started"
		if total > 0 && done == total {
			mStatus = "Done"
			completedMilestones[mID] = true
		} else if done > 0 || hasInProgress {
			mStatus = "In progress"
		}

		title := milestoneTitles[mID]
		if title == "" {
			title = mID
		}

		mInfo := MilestoneInfo{
			ID:         mID,
			Title:      title,
			TasksTotal: total,
			TasksDone:  done,
			Percent:    percent,
			Status:     mStatus,
		}
		milestones = append(milestones, mInfo)

		if activeMilestone == nil && mStatus == "In progress" {
			activeCopy := mInfo
			activeMilestone = &activeCopy
		}
	}

	// If no milestone is in progress, pick first not started; if all done, pick last
	if activeMilestone == nil && len(milestones) > 0 {
		for _, m := range milestones {
			if m.Status == "Not started" {
				activeCopy := m
				activeMilestone = &activeCopy
				break
			}
		}
		if activeMilestone == nil {
			activeCopy := milestones[len(milestones)-1]
			activeMilestone = &activeCopy
		}
	}

	// 6. Resolve NextGate
	var nextGate *GateInfo
	if len(activeTasks) > 0 {
		nextGate = activeTasks[0].Gate
	}

	// 7. Check warnings
	var warnings []string
	if len(activeTasks) > 1 {
		var ids []string
		for _, at := range activeTasks {
			ids = append(ids, at.ID)
		}
		warnings = append(warnings, fmt.Sprintf("Multiple tasks are active simultaneously (%s) — SDD Rule 3 requires one task at a time", strings.Join(ids, ", ")))
	}

	for _, at := range activeTasks {
		if at.ReviewBlockers > 0 {
			warnings = append(warnings, fmt.Sprintf("Task %s has %d unresolved 🔴 Blocker(s) in %s/review.md", at.ID, at.ReviewBlockers, at.Folder))
		}
	}

	// Check verified facts expiry in TECH_STACK.md
	factWarnings := checkFactExpiry(absRoot, completedMilestones, tasks)
	warnings = append(warnings, factWarnings...)

	sort.Strings(warnings)

	return &ProjectStatus{
		Root:            absRoot,
		ActiveMilestone: activeMilestone,
		Milestones:      milestones,
		TaskSummary:     summary,
		ActiveTasks:     activeTasks,
		NextTask:        nextNotStarted,
		NextGate:        nextGate,
		Warnings:        warnings,
	}, nil
}

func buildTaskInfo(root string, t roadmap.Task, normStatus, kind string, featureFolders map[string]string) TaskInfo {
	folder := featureFolders[t.ID]
	typology := ""
	if folder != "" {
		slashFolder := filepath.ToSlash(folder)
		switch {
		case strings.HasPrefix(slashFolder, "features/"):
			typology = "feature"
		case strings.HasPrefix(slashFolder, "bugfixes/"):
			typology = "bugfix"
		case strings.HasPrefix(slashFolder, "hotfixes/"):
			typology = "hotfix"
		case strings.HasPrefix(slashFolder, "spikes/"):
			typology = "spike"
		}
	}

	var phase string
	var gate *GateInfo

	switch kind {
	case "specifying", "awaiting plan approval":
		phase = "Phase A (Specify)"
		reqFile := "requirements.md"
		switch typology {
		case "hotfix":
			reqFile = "incident.md"
		case "bugfix":
			reqFile = "reproduction.md"
		case "spike":
			reqFile = "hypotheses.md"
		}
		gate = &GateInfo{
			TaskID: t.ID,
			Gate:   "GATE 1",
			Name:   "Approve requirements and plan",
			Status: "Pending plan approval",
			Action: fmt.Sprintf("Review %s/%s and plan.md; reply 'APPROVED %s'", folder, reqFile, t.ID),
		}
	case "implementing":
		phase = "Phase B (Implement)"
		planFile := "plan.md"
		if typology == "hotfix" {
			planFile = "patch.md"
		}
		gate = &GateInfo{
			TaskID: t.ID,
			Gate:   "GATE 2",
			Name:   "Implementation in progress",
			Status: "Implementing",
			Action: fmt.Sprintf("Execute %s/%s step by step; gate approval pending verification", folder, planFile),
		}
	case "awaiting review":
		phase = "Phase D (Close)"
		gate = &GateInfo{
			TaskID: t.ID,
			Gate:   "GATE 2",
			Name:   "Review and commit",
			Status: "Awaiting review",
			Action: fmt.Sprintf("Review diff & %s/validation.md; run git commit; reply 'COMMITTED %s'", folder, t.ID),
		}
	case "blocked":
		phase = "Blocked"
		gate = &GateInfo{
			TaskID: t.ID,
			Gate:   "",
			Name:   "Blocked",
			Status: normStatus,
			Action: "Resolve blocking dependency before proceeding",
		}
	default:
		phase = normStatus
	}

	blockers := 0
	if folder != "" {
		blockers = countReviewBlockers(filepath.Join(root, folder, "review.md"))
	}

	return TaskInfo{
		ID:             t.ID,
		Title:          t.Title,
		Milestone:      t.Milestone,
		Status:         normStatus,
		Phase:          phase,
		Folder:         folder,
		Typology:       typology,
		Gate:           gate,
		ReviewBlockers: blockers,
	}
}

func parseMilestoneTitles(content string) map[string]string {
	titles := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(content))
	inMilestoneTable := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if sm := sectionHdrRe.FindStringSubmatch(line); sm != nil {
			inMilestoneTable = strings.Contains(strings.ToLower(sm[1]), "milestone")
			continue
		}

		if inMilestoneTable && strings.HasPrefix(line, "|") {
			if strings.Contains(line, "---") || strings.Contains(strings.ToLower(line), "est. hours") {
				continue
			}
			m := milestoneRowRe.FindStringSubmatch(line)
			if len(m) >= 3 {
				id := strings.ToUpper(strings.TrimSpace(m[1]))
				title := strings.TrimSpace(m[2])
				titles[id] = title
			}
			continue
		}

		if m := milestoneHdrRe.FindStringSubmatch(line); m != nil {
			id := strings.ToUpper(m[1])
			if len(m) >= 3 && m[2] != "" {
				title := strings.TrimSpace(m[2])
				if titles[id] == "" {
					titles[id] = title
				}
			}
		}
	}
	return titles
}

func discoverFeatureFolders(root string) map[string]string {
	folders := make(map[string]string)
	typologies := []string{"features", "bugfixes", "hotfixes", "spikes"}
	for _, topDir := range typologies {
		dir := filepath.Join(root, topDir)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name := entry.Name()
			if strings.HasPrefix(name, "_") {
				continue
			}
			parts := strings.SplitN(name, "-", 3)
			if len(parts) >= 2 && strings.HasPrefix(name, "T-") {
				taskID := parts[0] + "-" + parts[1]
				folders[taskID] = filepath.Join(topDir, name)
			}
		}
	}
	return folders
}

func countReviewBlockers(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	count := 0
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "|") || strings.Contains(line, "---") || strings.Contains(line, "Category") {
			continue
		}
		if strings.Contains(line, "🔴") || strings.Contains(strings.ToLower(line), "blocker") {
			parts := strings.Split(line, "|")
			if len(parts) >= 7 {
				statusCol := strings.ToLower(strings.TrimSpace(parts[6]))
				if statusCol == "open" || statusCol == "pending" || statusCol == "" {
					count++
				}
			}
		}
	}
	return count
}

func checkFactExpiry(root string, completedMilestones map[string]bool, tasks []roadmap.Task) []string {
	techStackPath := filepath.Join(root, "TECH_STACK.md")
	data, err := os.ReadFile(techStackPath)
	if err != nil {
		return nil
	}

	taskMap := make(map[string]roadmap.Task)
	for _, t := range tasks {
		taskMap[t.ID] = t
	}

	var warnings []string
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	inFactsTable := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if factHdrRe.MatchString(line) {
			inFactsTable = true
			continue
		}
		if !inFactsTable {
			continue
		}
		if !strings.HasPrefix(line, "|") {
			inFactsTable = false
			continue
		}
		if strings.Contains(line, "---") {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 5 {
			continue
		}
		fact := strings.TrimSpace(parts[2])
		reVerifyAt := strings.TrimSpace(parts[4])
		if fact == "" || strings.HasPrefix(fact, "(") {
			continue
		}

		for _, tID := range taskRefRe.FindAllString(reVerifyAt, -1) {
			if t, exists := taskMap[tID]; exists && roadmap.Kind(t.Status) == "done" {
				warnings = append(warnings, fmt.Sprintf("Verified fact %q in TECH_STACK.md is past due for re-verification (%s is Done)", truncate(fact, 40), tID))
			}
		}

		for _, mID := range milestoneRefRe.FindAllString(reVerifyAt, -1) {
			if completedMilestones[strings.ToUpper(mID)] {
				warnings = append(warnings, fmt.Sprintf("Verified fact %q in TECH_STACK.md is past due for re-verification (milestone %s is completed)", truncate(fact, 40), mID))
			}
		}
	}
	return warnings
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func orderedMilestoneIDs(titles map[string]string, tasks map[string][]roadmap.Task) []string {
	seen := make(map[string]bool)
	var ordered []string

	// Milestone regex for numerical sorting (e.g. M0, M1, ..., M10)
	milestoneNumRe := regexp.MustCompile(`^M(\d+)$`)

	var knownNumeric []int
	var nonNumeric []string

	for id := range titles {
		if !seen[id] {
			seen[id] = true
			if m := milestoneNumRe.FindStringSubmatch(id); m != nil {
				if n, err := strconv.Atoi(m[1]); err == nil {
					knownNumeric = append(knownNumeric, n)
				} else {
					nonNumeric = append(nonNumeric, id)
				}
			} else {
				nonNumeric = append(nonNumeric, id)
			}
		}
	}

	for id := range tasks {
		if !seen[id] {
			seen[id] = true
			if m := milestoneNumRe.FindStringSubmatch(id); m != nil {
				if n, err := strconv.Atoi(m[1]); err == nil {
					knownNumeric = append(knownNumeric, n)
				} else {
					nonNumeric = append(nonNumeric, id)
				}
			} else {
				nonNumeric = append(nonNumeric, id)
			}
		}
	}

	sort.Ints(knownNumeric)
	sort.Strings(nonNumeric)

	for _, n := range knownNumeric {
		ordered = append(ordered, fmt.Sprintf("M%d", n))
	}
	ordered = append(ordered, nonNumeric...)
	return ordered
}

// RenderText formats ProjectStatus for clean terminal display.
func RenderText(ps *ProjectStatus) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Project: %s\n", filepath.Base(ps.Root))

	// Active Milestone summary
	if ps.ActiveMilestone != nil {
		fmt.Fprintf(&b, "Milestone: %s — %s (%d/%d tasks, %d%%)\n",
			ps.ActiveMilestone.ID, ps.ActiveMilestone.Title,
			ps.ActiveMilestone.TasksDone, ps.ActiveMilestone.TasksTotal, ps.ActiveMilestone.Percent)
	}

	// Active Tasks or Idle
	b.WriteString("\nActive Task:\n")
	if len(ps.ActiveTasks) == 0 {
		b.WriteString("  (None — project is idle)\n")
		if ps.NextTask != nil {
			fmt.Fprintf(&b, "  Next: %s — %s (Run 'sdd feature new %s <slug>' to begin)\n",
				ps.NextTask.ID, ps.NextTask.Title, ps.NextTask.ID)
		}
	} else {
		for _, at := range ps.ActiveTasks {
			marker := "[>]"
			if at.Phase == "Blocked" {
				marker = "[!]"
			}
			fmt.Fprintf(&b, "  %s %s: %s\n", marker, at.ID, at.Title)
			fmt.Fprintf(&b, "      Milestone: %s\n", at.Milestone)
			fmt.Fprintf(&b, "      Phase:     %s\n", at.Phase)
			fmt.Fprintf(&b, "      Status:    %s\n", at.Status)
			if at.Typology != "" {
				fmt.Fprintf(&b, "      Typology:  %s\n", at.Typology)
			}
			if at.Folder != "" {
				fmt.Fprintf(&b, "      Folder:    %s\n", at.Folder)
			}
			if at.Gate != nil && at.Gate.Gate != "" {
				fmt.Fprintf(&b, "      Pending:   %s (%s)\n", at.Gate.Gate, at.Gate.Name)
				fmt.Fprintf(&b, "      Action:    %s\n", at.Gate.Action)
			} else if at.Gate != nil && at.Gate.Action != "" {
				fmt.Fprintf(&b, "      Action:    %s\n", at.Gate.Action)
			}
			if at.ReviewBlockers > 0 {
				fmt.Fprintf(&b, "      Review:    %d unresolved 🔴 Blocker(s)\n", at.ReviewBlockers)
			}
		}
	}

	// Milestones Breakdown
	b.WriteString("\nRoadmap Milestones:\n")
	for _, m := range ps.Milestones {
		marker := "[ ]"
		switch m.Status {
		case "Done":
			marker = "[x]"
		case "In progress":
			marker = "[>]"
		}
		fmt.Fprintf(&b, "  %s %-4s %-42s (%d/%d tasks, %d%%)\n",
			marker, m.ID+":", m.Title, m.TasksDone, m.TasksTotal, m.Percent)
	}

	// Tasks Overview
	fmt.Fprintf(&b, "\nTasks Overview:\n  Total: %d | Done: %d | In Progress: %d | Blocked: %d | Not Started: %d\n",
		ps.TaskSummary.Total, ps.TaskSummary.Done, ps.TaskSummary.InProgress, ps.TaskSummary.Blocked, ps.TaskSummary.NotStarted)

	// Warnings
	if len(ps.Warnings) > 0 {
		b.WriteString("\nWarnings:\n")
		for _, w := range ps.Warnings {
			fmt.Fprintf(&b, "  ! %s\n", w)
		}
	}

	return b.String()
}

// RenderJSON formats ProjectStatus as 2-space indented JSON bytes.
func RenderJSON(ps *ProjectStatus) ([]byte, error) {
	return json.MarshalIndent(ps, "", "  ")
}
