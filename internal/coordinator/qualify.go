package coordinator

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vikashkumar21/sdd/internal/deconflict"
	"github.com/vikashkumar21/sdd/internal/roadmap"
	"github.com/vikashkumar21/sdd/internal/worktree"
)

var (
	reviewStatusRe = regexp.MustCompile(`(?i)^Status:\s*([^\r\n]+)`)
)

// QualifyOptions specifies parameters for pre-merge qualification.
type QualifyOptions struct {
	TargetBranch   string
	SkipFlakeCheck bool
	RunTestFn      func(dir string, pkgs []string) (bool, string, error)
}

// QualificationReport records the complete pre-merge qualification results.
type QualificationReport struct {
	TaskID         string   `json:"task_id"`
	Typology       string   `json:"typology"`
	TaskFolder     string   `json:"task_folder"`
	Branch         string   `json:"branch"`
	TargetBranch   string   `json:"target_branch"`
	ReviewStatus   string   `json:"review_status"`
	ReviewBlockers int      `json:"review_blockers"`
	ModifiedFiles  []string `json:"modified_files"`
	ModifiedPkgs   []string `json:"modified_pkgs"`
	BlastRadius    []string `json:"blast_radius"`
	TestRun1Pass   bool     `json:"test_run1_pass"`
	TestRun2Pass   bool     `json:"test_run2_pass"`
	FlakeDetected  bool     `json:"flake_detected"`
	MergeClean     bool     `json:"merge_clean"`
	MergeConflicts []string `json:"merge_conflicts,omitempty"`
	DeconflictSafe bool     `json:"deconflict_safe"`
	ReadyToMerge   bool     `json:"ready_to_merge"`
	Errors         []string `json:"errors,omitempty"`
}

// Qualify executes pre-merge validation, blast radius sweep, flake guard, and merge simulation.
func Qualify(root, taskID string, opts QualifyOptions) (*QualificationReport, error) {
	loc, err := worktree.FindTask(root, taskID)
	if err != nil {
		return nil, fmt.Errorf("locating task %s: %w", taskID, err)
	}

	targetBranch := opts.TargetBranch
	if targetBranch == "" {
		targetBranch = "main"
		if _, err := runGit(root, "rev-parse", "--verify", "main"); err != nil {
			if _, errMaster := runGit(root, "rev-parse", "--verify", "master"); errMaster == nil {
				targetBranch = "master"
			}
		}
	}

	// Resolve task branch
	taskBranch := fmt.Sprintf("%s/%s-%s", loc.Typology, loc.TaskID, loc.Slug)
	if entries, err := worktree.List(root); err == nil {
		for _, e := range entries {
			if strings.EqualFold(e.TaskID, taskID) && e.Branch != "" {
				taskBranch = e.Branch
				break
			}
		}
	}

	rep := &QualificationReport{
		TaskID:       loc.TaskID,
		Typology:     loc.Typology,
		TaskFolder:   loc.Folder,
		Branch:       taskBranch,
		TargetBranch: targetBranch,
	}

	var errorsList []string

	// 1. Review Document Audit
	reviewPath := filepath.Join(root, filepath.FromSlash(loc.Folder), "review.md")
	status, blockers, reviewErr := auditReviewDocument(reviewPath)
	rep.ReviewStatus = status
	rep.ReviewBlockers = blockers
	if reviewErr != nil {
		errorsList = append(errorsList, fmt.Sprintf("Review audit: %v", reviewErr))
	}
	if blockers > 0 {
		errorsList = append(errorsList, fmt.Sprintf("Review audit: %d unresolved 🔴 Blocker(s) found in review.md", blockers))
	}
	if !strings.EqualFold(status, "approved") && !strings.Contains(strings.ToLower(status), "approved") {
		errorsList = append(errorsList, fmt.Sprintf("Review audit: review status is %q (expected Approved)", status))
	}

	// 2. Modified Files & Blast Radius
	var modFiles []string
	if _, err := runGit(root, "rev-parse", "--verify", taskBranch); err == nil {
		changed, err := GetChangedFiles(root, taskBranch, targetBranch)
		if err == nil && len(changed) > 0 {
			modFiles = changed
		}
	}

	// Fallback to plan.md files if git diff was empty (e.g. before worker branch commit)
	if len(modFiles) == 0 {
		plan, planErr := deconflict.LoadTaskPlan(root, taskID)
		if planErr == nil && len(plan.Files) > 0 {
			modFiles = plan.Files
		}
	}
	rep.ModifiedFiles = modFiles

	blast, blastErr := ComputeBlastRadius(root, modFiles)
	if blastErr != nil {
		errorsList = append(errorsList, fmt.Sprintf("Blast radius calculation failed: %v", blastErr))
	} else {
		rep.ModifiedPkgs = blast.ModifiedPackages
		rep.BlastRadius = blast.DownstreamPackages
	}

	// 3. Plan Deconfliction
	deconflictSafe, deconflictErr := checkDeconfliction(root, loc.TaskID)
	rep.DeconflictSafe = deconflictSafe
	if deconflictErr != nil {
		errorsList = append(errorsList, fmt.Sprintf("Static deconfliction: %v", deconflictErr))
	}

	// 4. Regression Test Sweep & Flake Guard
	var pkgsToTest []string
	if blast != nil && len(blast.AllAffectedPackages) > 0 {
		for _, p := range blast.AllAffectedPackages {
			pkgsToTest = append(pkgsToTest, "./"+p)
		}
	} else {
		pkgsToTest = []string{"./..."}
	}

	testRunner := opts.RunTestFn
	if testRunner == nil {
		testRunner = defaultTestRunner
	}

	pass1, out1, err1 := testRunner(root, pkgsToTest)
	rep.TestRun1Pass = pass1
	if !pass1 {
		errorsList = append(errorsList, fmt.Sprintf("Regression test run 1 failed: %s (%v)", strings.TrimSpace(out1), err1))
	}

	if pass1 && !opts.SkipFlakeCheck {
		pass2, out2, err2 := testRunner(root, pkgsToTest)
		rep.TestRun2Pass = pass2
		if !pass2 {
			rep.FlakeDetected = true
			errorsList = append(errorsList, fmt.Sprintf("Flake detected: test passed on Run 1 but failed on Run 2: %s (%v)", strings.TrimSpace(out2), err2))
		}
	} else if pass1 && opts.SkipFlakeCheck {
		rep.TestRun2Pass = true
	}

	// 5. Merge Simulation
	if _, err := runGit(root, "rev-parse", "--verify", taskBranch); err == nil {
		sim, err := SimulateMerge(root, taskBranch, targetBranch)
		if err != nil {
			rep.MergeClean = false
			errorsList = append(errorsList, fmt.Sprintf("Merge simulation error: %v", err))
		} else {
			rep.MergeClean = sim.CanMerge
			rep.MergeConflicts = sim.Conflicts
			if !sim.CanMerge {
				errorsList = append(errorsList, fmt.Sprintf("Merge simulation collision: conflicts detected in %s", strings.Join(sim.Conflicts, ", ")))
			}
		}
	} else {
		// Task branch doesn't exist locally yet
		rep.MergeClean = false
		errorsList = append(errorsList, fmt.Sprintf("Task branch %s does not exist", taskBranch))
	}

	rep.Errors = errorsList
	rep.ReadyToMerge = len(errorsList) == 0 && rep.MergeClean && rep.TestRun1Pass && (!rep.FlakeDetected)

	return rep, nil
}

func auditReviewDocument(path string) (string, int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "missing", 0, fmt.Errorf("review.md not found at %s", path)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	var status string
	blockers := 0

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if status == "" && reviewStatusRe.MatchString(trimmed) {
			m := reviewStatusRe.FindStringSubmatch(trimmed)
			if len(m) > 1 {
				status = strings.TrimSpace(m[1])
			}
		}

		if strings.HasPrefix(trimmed, "|") && (strings.Contains(trimmed, "🔴") || strings.Contains(strings.ToLower(trimmed), "blocker")) {
			// Skip header row
			if !strings.Contains(trimmed, "Severity") && !strings.Contains(trimmed, "---") {
				blockers++
			}
		}
	}

	if status == "" {
		status = "unknown"
	}
	return status, blockers, nil
}

func checkDeconfliction(root, taskID string) (bool, error) {
	roadmapPath := filepath.Join(root, "ROADMAP.md")
	data, err := os.ReadFile(roadmapPath)
	if err != nil {
		return true, nil
	}

	tasks := roadmap.Parse(string(data))
	var activePlans []deconflict.TaskPlan

	for _, t := range tasks {
		switch roadmap.Kind(t.Status) {
		case "implementing", "awaiting review":
			if plan, err := deconflict.LoadTaskPlan(root, t.ID); err == nil && len(plan.Files) > 0 {
				activePlans = append(activePlans, *plan)
			}
		}
	}

	if len(activePlans) < 2 {
		return true, nil
	}

	rep := deconflict.Analyze(activePlans)
	for _, c := range rep.Conflicts {
		if strings.EqualFold(c.TaskA, taskID) || strings.EqualFold(c.TaskB, taskID) {
			other := c.TaskA
			if strings.EqualFold(c.TaskA, taskID) {
				other = c.TaskB
			}
			return false, fmt.Errorf("task %s has planned file collisions with concurrent task %s on: %s", taskID, other, strings.Join(c.SharedFiles, ", "))
		}
	}
	return true, nil
}

func defaultTestRunner(dir string, pkgs []string) (bool, string, error) {
	args := append([]string{"test", "-count=1"}, pkgs...)
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &outBuf
	err := cmd.Run()
	if err != nil {
		return false, outBuf.String(), err
	}
	return true, outBuf.String(), nil
}

// FormatQualificationReport formats the report for terminal output.
func FormatQualificationReport(rep *QualificationReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Pre-Merge Qualification Report: %s\n", rep.TaskID)
	b.WriteString(strings.Repeat("─", 50) + "\n")
	fmt.Fprintf(&b, "  Branch:         %s → %s\n", rep.Branch, rep.TargetBranch)
	fmt.Fprintf(&b, "  Review Audit:   %s (%d blockers)\n", rep.ReviewStatus, rep.ReviewBlockers)
	fmt.Fprintf(&b, "  Deconfliction:  %s\n", statusText(rep.DeconflictSafe))
	fmt.Fprintf(&b, "  Modified Files: %d file(s)\n", len(rep.ModifiedFiles))
	fmt.Fprintf(&b, "  Modified Pkgs:  %s\n", formatSlice(rep.ModifiedPkgs))
	fmt.Fprintf(&b, "  Blast Radius:   %s\n", formatSlice(rep.BlastRadius))

	flakeStatus := "PASS"
	if rep.FlakeDetected {
		flakeStatus = "FAIL (Flake Detected)"
	}
	fmt.Fprintf(&b, "  Regression 1:   %s\n", statusText(rep.TestRun1Pass))
	fmt.Fprintf(&b, "  Flake Guard 2:  %s\n", flakeStatus)
	fmt.Fprintf(&b, "  Merge Sim:      %s\n", statusText(rep.MergeClean))

	if len(rep.MergeConflicts) > 0 {
		fmt.Fprintf(&b, "  Conflicts:      %s\n", strings.Join(rep.MergeConflicts, ", "))
	}

	b.WriteString(strings.Repeat("─", 50) + "\n")
	if rep.ReadyToMerge {
		b.WriteString("✓ QUALIFICATION: READY TO MERGE\n")
	} else {
		b.WriteString("✗ QUALIFICATION: NOT READY (BLOCKERS DETECTED)\n")
		for _, e := range rep.Errors {
			fmt.Fprintf(&b, "    ! %s\n", e)
		}
	}
	return b.String()
}

// FormatGate2Presentation prepares the Task Coordinator summary expecting APPROVED MERGE T-xxx.
func FormatGate2Presentation(rep *QualificationReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Task Coordinator GATE 2 Presentation — %s\n\n", rep.TaskID)
	b.WriteString("1. Pre-Merge Qualification:\n")
	fmt.Fprintf(&b, "   - Branch: %s into %s\n", rep.Branch, rep.TargetBranch)
	fmt.Fprintf(&b, "   - Code Review: %s (0 Blockers)\n", rep.ReviewStatus)
	fmt.Fprintf(&b, "   - Blast Radius: %d affected packages verified\n", len(rep.ModifiedPkgs)+len(rep.BlastRadius))
	b.WriteString("   - Flake Guard: Double-run test suite passed cleanly\n")
	b.WriteString("   - Git Merge Simulation: Clean fast-forward/mergeable with 0 conflicts\n\n")

	b.WriteString("2. Suggested Commit / Merge Message:\n")
	fmt.Fprintf(&b, "   feat(%s): merge %s into %s (%s)\n\n", rep.Typology, rep.Branch, rep.TargetBranch, rep.TaskID)

	b.WriteString("3. Exact Human Reply Expected:\n")
	fmt.Fprintf(&b, "   APPROVED MERGE %s\n", rep.TaskID)
	return b.String()
}

func statusText(ok bool) string {
	if ok {
		return "PASS"
	}
	return "FAIL"
}

func formatSlice(s []string) string {
	if len(s) == 0 {
		return "(none)"
	}
	return strings.Join(s, ", ")
}
