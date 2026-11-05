// Package deconflict analyzes planned file target sets across task plans
// to detect collision hazards, classify concurrency safety, and schedule parallel batches.
package deconflict

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/vikashkumar21/sdd/internal/worktree"
)

var (
	filesSectionHeaderRe = regexp.MustCompile(`(?i)^##\s+\d+\.\s*Files(?:\s+to\s+create\s+or\s+modify|\s+to\s+touch|\s+touched)?`)
	sectionHeaderRe      = regexp.MustCompile(`^##\s+`)
)

// TaskPlan represents the extracted file targets from a task's plan.
type TaskPlan struct {
	TaskID   string   `json:"task_id"`
	Typology string   `json:"typology"`
	Slug     string   `json:"slug"`
	Folder   string   `json:"folder"`
	PlanFile string   `json:"plan_file"`
	Files    []string `json:"files"`
}

// Overlap records a shared file collision between two tasks.
type Overlap struct {
	TaskA       string   `json:"task_a"`
	TaskB       string   `json:"task_b"`
	SharedFiles []string `json:"shared_files"`
}

// Report contains the complete results of a deconfliction analysis.
type Report struct {
	Tasks       []TaskPlan  `json:"tasks"`
	Conflicts   []Overlap   `json:"conflicts"`
	SafePairs   [][2]string `json:"safe_pairs"`
	Batches     [][]string  `json:"batches"`
	HasConflict bool        `json:"has_conflict"`
}

// ExtractPlanFiles parses Section 5 of plan.md/patch.md content and returns canonical file paths.
func ExtractPlanFiles(content string) []string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	inFilesSection := false
	var paths []string
	seen := make(map[string]bool)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if filesSectionHeaderRe.MatchString(line) {
			inFilesSection = true
			continue
		}
		if inFilesSection && sectionHeaderRe.MatchString(line) {
			// Left the files section
			break
		}
		if inFilesSection && strings.HasPrefix(line, "|") {
			// Skip markdown table separator and header lines
			if strings.Contains(line, "---") || strings.Contains(strings.ToLower(line), "action") || strings.Contains(strings.ToLower(line), "path") {
				continue
			}
			parts := strings.Split(line, "|")
			if len(parts) >= 2 {
				rawPath := strings.TrimSpace(parts[1])
				rawPath = strings.Trim(rawPath, "` \t\r\n")
				normPath := filepath.ToSlash(filepath.Clean(rawPath))
				normPath = strings.TrimPrefix(normPath, "./")

				if normPath != "" && normPath != "." && normPath != "(none)" && normPath != "none" && !seen[normPath] {
					seen[normPath] = true
					paths = append(paths, normPath)
				}
			}
		}
	}

	sort.Strings(paths)
	return paths
}

// LoadTaskPlan reads and parses plan targets for a given task ID.
func LoadTaskPlan(root, taskID string) (*TaskPlan, error) {
	loc, err := worktree.FindTask(root, taskID)
	if err != nil {
		return nil, err
	}

	folderAbs := filepath.Join(root, filepath.FromSlash(loc.Folder))
	planPath := filepath.Join(folderAbs, "plan.md")
	planFile := "plan.md"
	if _, err := os.Stat(planPath); os.IsNotExist(err) {
		patchPath := filepath.Join(folderAbs, "patch.md")
		if _, patchErr := os.Stat(patchPath); patchErr == nil {
			planPath = patchPath
			planFile = "patch.md"
		} else {
			return &TaskPlan{
				TaskID:   loc.TaskID,
				Typology: loc.Typology,
				Slug:     loc.Slug,
				Folder:   loc.Folder,
				PlanFile: "",
				Files:    []string{},
			}, nil
		}
	}

	data, err := os.ReadFile(planPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", planPath, err)
	}

	files := ExtractPlanFiles(string(data))
	return &TaskPlan{
		TaskID:   loc.TaskID,
		Typology: loc.Typology,
		Slug:     loc.Slug,
		Folder:   loc.Folder,
		PlanFile: planFile,
		Files:    files,
	}, nil
}

// Analyze performs pairwise overlap analysis and schedules parallel batches.
func Analyze(plans []TaskPlan) *Report {
	// Sort plans deterministically by TaskID
	sort.Slice(plans, func(i, j int) bool {
		return plans[i].TaskID < plans[j].TaskID
	})

	var conflicts []Overlap
	var safePairs [][2]string
	conflictGraph := make(map[string]map[string]bool)

	for _, p := range plans {
		conflictGraph[p.TaskID] = make(map[string]bool)
	}

	for i := 0; i < len(plans); i++ {
		for j := i + 1; j < len(plans); j++ {
			p1, p2 := plans[i], plans[j]
			shared := intersect(p1.Files, p2.Files)
			if len(shared) > 0 {
				conflicts = append(conflicts, Overlap{
					TaskA:       p1.TaskID,
					TaskB:       p2.TaskID,
					SharedFiles: shared,
				})
				conflictGraph[p1.TaskID][p2.TaskID] = true
				conflictGraph[p2.TaskID][p1.TaskID] = true
			} else {
				safePairs = append(safePairs, [2]string{p1.TaskID, p2.TaskID})
			}
		}
	}

	// Schedule into parallel batches using greedy disjoint coloring
	var batches [][]string
	for _, p := range plans {
		placed := false
		for bIdx := range batches {
			canAddToBatch := true
			for _, existingID := range batches[bIdx] {
				if conflictGraph[p.TaskID][existingID] {
					canAddToBatch = false
					break
				}
			}
			if canAddToBatch {
				batches[bIdx] = append(batches[bIdx], p.TaskID)
				placed = true
				break
			}
		}
		if !placed {
			batches = append(batches, []string{p.TaskID})
		}
	}

	return &Report{
		Tasks:       plans,
		Conflicts:   conflicts,
		SafePairs:   safePairs,
		Batches:     batches,
		HasConflict: len(conflicts) > 0,
	}
}

// FormatText renders Report for readable terminal output.
func FormatText(rep *Report) string {
	var b strings.Builder
	b.WriteString("Static Plan Deconfliction Report\n")
	var taskIDs []string
	for _, t := range rep.Tasks {
		taskIDs = append(taskIDs, t.TaskID)
	}
	fmt.Fprintf(&b, "Analyzed: %d task(s) (%s)\n\n", len(rep.Tasks), strings.Join(taskIDs, ", "))

	if !rep.HasConflict {
		b.WriteString("✓ Concurrency Status: SAFE (0 file conflicts detected)\n")
		b.WriteString("All analyzed tasks have disjoint planned file sets and can execute in parallel worktrees.\n\n")
	} else {
		fmt.Fprintf(&b, "⚠ Concurrency Status: COLLISION HAZARD (%d conflict(s) detected)\n\n", len(rep.Conflicts))
		b.WriteString("Conflict Matrix:\n")
		for _, c := range rep.Conflicts {
			fmt.Fprintf(&b, "  ! %s ↔ %s share %d file(s):\n", c.TaskA, c.TaskB, len(c.SharedFiles))
			for _, f := range c.SharedFiles {
				fmt.Fprintf(&b, "      - %s\n", f)
			}
		}
		b.WriteString("\n")
	}

	b.WriteString("Recommended Concurrency Schedule:\n")
	for i, batch := range rep.Batches {
		kind := "Parallel Batch"
		if len(batch) == 1 && rep.HasConflict && i > 0 {
			kind = "Sequential Queue"
		}
		fmt.Fprintf(&b, "  Wave %d (%s): %s\n", i+1, kind, strings.Join(batch, ", "))
	}

	return b.String()
}

func intersect(a, b []string) []string {
	set := make(map[string]bool)
	for _, x := range a {
		set[x] = true
	}
	var res []string
	for _, y := range b {
		if set[y] {
			res = append(res, y)
		}
	}
	sort.Strings(res)
	return res
}
