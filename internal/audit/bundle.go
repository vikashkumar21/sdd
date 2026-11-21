package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/vikashkumar21/sdd/internal/role"
)

const maxUntrackedFileSize = 1024 * 1024 // 1MB limit for untracked files in diffs

var taskIDPattern = regexp.MustCompile(`^T-\d{3,}$`)

// PlanBundle contains staged metadata for an isolated plan review.
type PlanBundle struct {
	TaskID    string    `json:"task_id"`
	TaskDir   string    `json:"task_dir"`
	BundleDir string    `json:"bundle_dir"`
	CreatedAt time.Time `json:"created_at"`
	Files     []string  `json:"files"`
}

// ReviewBundle contains staged metadata for an isolated code review triad.
type ReviewBundle struct {
	TaskID    string    `json:"task_id"`
	Lens      string    `json:"lens"`
	TaskDir   string    `json:"task_dir"`
	BundleDir string    `json:"bundle_dir"`
	CreatedAt time.Time `json:"created_at"`
	Files     []string  `json:"files"`
}

// EnsureGitignoreReview ensures .sdd/review/ is ignored in the root .gitignore.
func EnsureGitignoreReview(root string) error {
	gitIgnorePath := filepath.Join(root, ".gitignore")
	content, err := os.ReadFile(gitIgnorePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	entry := ".sdd/review/"
	if strings.Contains(string(content), entry) {
		return nil
	}

	var buf bytes.Buffer
	buf.Write(content)
	if len(content) > 0 && !bytes.HasSuffix(content, []byte("\n")) {
		buf.WriteString("\n")
	}
	buf.WriteString("\n# SDD stateless review bundles\n.sdd/review/\n")

	return os.WriteFile(gitIgnorePath, buf.Bytes(), 0o644)
}

// StagePlanBundle stages an isolated, frozen review bundle for Phase A plan review.
func StagePlanBundle(root, taskID string, now ...time.Time) (*PlanBundle, error) {
	normTaskID := strings.ToUpper(strings.TrimSpace(taskID))
	if !taskIDPattern.MatchString(normTaskID) {
		return nil, fmt.Errorf("invalid task ID format %q (expected format like T-001)", taskID)
	}

	taskRelDir, _, err := findTaskDirectory(root, normTaskID)
	if err != nil {
		return nil, err
	}

	if err := EnsureGitignoreReview(root); err != nil {
		return nil, fmt.Errorf("ensuring .gitignore review entry: %w", err)
	}

	bundleDirRel := filepath.Join(".sdd", "review", normTaskID, "plan")
	bundleDirAbs := filepath.Join(root, bundleDirRel)
	if err := os.MkdirAll(bundleDirAbs, 0o755); err != nil {
		return nil, fmt.Errorf("creating plan review bundle dir: %w", err)
	}

	taskDirAbs := filepath.Join(root, filepath.FromSlash(taskRelDir))
	reqContent := readDoc(taskDirAbs, "requirements.md")
	planContent := readDoc(taskDirAbs, "plan.md")
	if planContent == "" {
		planContent = readDoc(taskDirAbs, "patch.md")
	}

	// 1. spec.md: combines requirements and plan
	var specBuf bytes.Buffer
	fmt.Fprintf(&specBuf, "# Specification Bundle: %s (Plan Review)\n\n", normTaskID)
	specBuf.WriteString("## Requirements (requirements.md)\n\n")
	specBuf.WriteString(strings.TrimSpace(reqContent))
	specBuf.WriteString("\n\n---\n\n## Implementation Plan (plan.md)\n\n")
	specBuf.WriteString(strings.TrimSpace(planContent))
	specBuf.WriteString("\n")

	specPath := filepath.Join(bundleDirAbs, "spec.md")
	if err := os.WriteFile(specPath, specBuf.Bytes(), 0o644); err != nil {
		return nil, fmt.Errorf("writing spec.md: %w", err)
	}

	// 2. rubric.md: plan-reviewer role skill
	roleSkill, _ := role.ResolveSkill(root, "plan-reviewer", nil)
	rubricPath := filepath.Join(bundleDirAbs, "rubric.md")
	if err := os.WriteFile(rubricPath, []byte(strings.TrimSpace(roleSkill)+"\n"), 0o644); err != nil {
		return nil, fmt.Errorf("writing rubric.md: %w", err)
	}

	stagedFiles := []string{
		filepath.ToSlash(filepath.Join(bundleDirRel, "spec.md")),
		filepath.ToSlash(filepath.Join(bundleDirRel, "rubric.md")),
	}

	createdAt := time.Now().UTC()
	if len(now) > 0 && !now[0].IsZero() {
		createdAt = now[0].UTC()
	}

	// 3. manifest.json
	bundle := &PlanBundle{
		TaskID:    normTaskID,
		TaskDir:   taskRelDir,
		BundleDir: filepath.ToSlash(bundleDirRel),
		CreatedAt: createdAt,
		Files:     stagedFiles,
	}

	manifestData, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling plan manifest: %w", err)
	}
	manifestPath := filepath.Join(bundleDirAbs, "manifest.json")
	if err := os.WriteFile(manifestPath, manifestData, 0o644); err != nil {
		return nil, fmt.Errorf("writing manifest.json: %w", err)
	}
	bundle.Files = append(bundle.Files, filepath.ToSlash(filepath.Join(bundleDirRel, "manifest.json")))

	return bundle, nil
}

// StageReviewBundle stages an isolated, frozen review bundle for Phase C.2 code review triad.
func StageReviewBundle(root, taskID, lens string, now ...time.Time) (*ReviewBundle, error) {
	normTaskID := strings.ToUpper(strings.TrimSpace(taskID))
	if !taskIDPattern.MatchString(normTaskID) {
		return nil, fmt.Errorf("invalid task ID format %q (expected format like T-001)", taskID)
	}

	normLens := strings.ToLower(strings.TrimSpace(lens))
	if normLens == "" {
		normLens = "all"
	}

	taskRelDir, _, err := findTaskDirectory(root, normTaskID)
	if err != nil {
		return nil, err
	}

	if err := EnsureGitignoreReview(root); err != nil {
		return nil, fmt.Errorf("ensuring .gitignore review entry: %w", err)
	}

	bundleDirRel := filepath.Join(".sdd", "review", normTaskID, "code")
	bundleDirAbs := filepath.Join(root, bundleDirRel)
	if err := os.MkdirAll(bundleDirAbs, 0o755); err != nil {
		return nil, fmt.Errorf("creating code review bundle dir: %w", err)
	}

	taskDirAbs := filepath.Join(root, filepath.FromSlash(taskRelDir))
	reqContent := readDoc(taskDirAbs, "requirements.md")
	planContent := readDoc(taskDirAbs, "plan.md")
	if planContent == "" {
		planContent = readDoc(taskDirAbs, "patch.md")
	}

	// 1. spec.md
	var specBuf bytes.Buffer
	fmt.Fprintf(&specBuf, "# Specification Bundle: %s (Code Review Triad)\n\n", normTaskID)
	specBuf.WriteString("## Requirements (requirements.md)\n\n")
	specBuf.WriteString(strings.TrimSpace(reqContent))
	specBuf.WriteString("\n\n---\n\n## Implementation Plan (plan.md)\n\n")
	specBuf.WriteString(strings.TrimSpace(planContent))
	specBuf.WriteString("\n")

	specPath := filepath.Join(bundleDirAbs, "spec.md")
	if err := os.WriteFile(specPath, specBuf.Bytes(), 0o644); err != nil {
		return nil, fmt.Errorf("writing spec.md: %w", err)
	}

	// 2. diff.patch: git diff including untracked new files
	diffText, err := extractGitDiff(root)
	if err != nil {
		diffText = fmt.Sprintf("# Note: could not extract git diff: %v\n", err)
	}
	diffPath := filepath.Join(bundleDirAbs, "diff.patch")
	if err := os.WriteFile(diffPath, []byte(diffText), 0o644); err != nil {
		return nil, fmt.Errorf("writing diff.patch: %w", err)
	}

	// 3. rubric.md: role skill for lens
	roleSkill := resolveReviewRubric(root, normLens)
	rubricPath := filepath.Join(bundleDirAbs, "rubric.md")
	if err := os.WriteFile(rubricPath, []byte(strings.TrimSpace(roleSkill)+"\n"), 0o644); err != nil {
		return nil, fmt.Errorf("writing rubric.md: %w", err)
	}

	stagedFiles := []string{
		filepath.ToSlash(filepath.Join(bundleDirRel, "spec.md")),
		filepath.ToSlash(filepath.Join(bundleDirRel, "diff.patch")),
		filepath.ToSlash(filepath.Join(bundleDirRel, "rubric.md")),
	}

	createdAt := time.Now().UTC()
	if len(now) > 0 && !now[0].IsZero() {
		createdAt = now[0].UTC()
	}

	// 4. manifest.json
	bundle := &ReviewBundle{
		TaskID:    normTaskID,
		Lens:      normLens,
		TaskDir:   taskRelDir,
		BundleDir: filepath.ToSlash(bundleDirRel),
		CreatedAt: createdAt,
		Files:     stagedFiles,
	}

	manifestData, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling review manifest: %w", err)
	}
	manifestPath := filepath.Join(bundleDirAbs, "manifest.json")
	if err := os.WriteFile(manifestPath, manifestData, 0o644); err != nil {
		return nil, fmt.Errorf("writing manifest.json: %w", err)
	}
	bundle.Files = append(bundle.Files, filepath.ToSlash(filepath.Join(bundleDirRel, "manifest.json")))

	return bundle, nil
}

// FormatPlanReviewPrompt generates a tabula-rasa, isolated handover prompt for Phase A plan review.
func FormatPlanReviewPrompt(bundle *PlanBundle, root string) (string, error) {
	if bundle == nil {
		return "", fmt.Errorf("nil plan bundle")
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "# Handover: Stateless Pre-GATE 1 Plan Review for %s\n\n", bundle.TaskID)
	fmt.Fprintf(&sb, "You are acting as an **Independent Plan Reviewer** for task `%s`.\n", bundle.TaskID)
	sb.WriteString("Your mandate is to impartially audit the Phase A specification and implementation plan **before** GATE 1 presentation to the human.\n")
	sb.WriteString("**You are an independent reviewer evaluating this task in fresh, isolated context. Rely strictly on the staged artifacts in your review bundle without conversational bias or assumptions.**\n\n")

	sb.WriteString("## SDD Anthem & Principles\n")
	sb.WriteString("> **ALWAYS BE DETERMINISTIC. When a decision is needed, STOP and ask the human. Never guess. Never silently choose.**\n\n")

	sb.WriteString("## Staged Review Bundle Artifacts\n")
	fmt.Fprintf(&sb, "- **Bundle Directory:** `%s/`\n", bundle.BundleDir)
	fmt.Fprintf(&sb, "- **Frozen Specification:** `%s/spec.md` (requirements and plan)\n", bundle.BundleDir)
	fmt.Fprintf(&sb, "- **Review Rubric:** `%s/rubric.md` (canonical plan-reviewer role skill)\n", bundle.BundleDir)
	fmt.Fprintf(&sb, "- **Bundle Manifest:** `%s/manifest.json`\n\n", bundle.BundleDir)

	sb.WriteString("## Plan Review Instructions\n")
	fmt.Fprintf(&sb, "1. Read `%s/spec.md` and audit against `%s/rubric.md`.\n", bundle.BundleDir, bundle.BundleDir)
	sb.WriteString("2. Check for spec completeness (atomic R-xxx), plan soundness, boundary precision, risk/rollback, and testability.\n")
	sb.WriteString("3. Report any 🔴 Blocker (missing step, boundary hole, untestable assertion, unapproved dependency) or 🟡 Warning.\n")
	fmt.Fprintf(&sb, "4. Verify that the 'Open questions' section in `%s/requirements.md` is empty.\n", bundle.TaskDir)
	sb.WriteString("5. Provide an explicit sign-off recommendation: `[Approved / Changes Requested]` with actionable remediation notes.\n")

	return sb.String(), nil
}

// FormatReviewPrompt generates a tabula-rasa, isolated handover prompt for Phase C.2 code review triad.
func FormatReviewPrompt(bundle *ReviewBundle, root string) (string, error) {
	if bundle == nil {
		return "", fmt.Errorf("nil review bundle")
	}

	var sb strings.Builder
	lensTitle := "Ensemble Review Triad"
	if bundle.Lens != "" && bundle.Lens != "all" {
		switch bundle.Lens {
		case "spec":
			lensTitle = "Spec Auditor Lens"
		case "security":
			lensTitle = "AppSec Specialist Lens"
		case "pattern":
			lensTitle = "Pattern & Quality Auditor Lens"
		default:
			lensTitle = fmt.Sprintf("%s Lens", bundle.Lens)
		}
	}

	fmt.Fprintf(&sb, "# Handover: Stateless Subagent Code Review for %s (%s)\n\n", bundle.TaskID, lensTitle)
	fmt.Fprintf(&sb, "You are acting as an **Independent Subagent Reviewer (%s)** for task `%s`.\n", lensTitle, bundle.TaskID)
	sb.WriteString("Your mandate is to impartially audit the implementation before GATE 2 human review.\n")
	sb.WriteString("**You are an independent reviewer evaluating this task in fresh, isolated context. Rely strictly on the staged artifacts in your review bundle without conversational bias or assumptions.**\n\n")

	sb.WriteString("## SDD Anthem & Principles\n")
	sb.WriteString("> **ALWAYS BE DETERMINISTIC. When a decision is needed, STOP and ask the human. Never guess. Never silently choose.**\n\n")

	sb.WriteString("## Staged Review Bundle Artifacts\n")
	fmt.Fprintf(&sb, "- **Bundle Directory:** `%s/`\n", bundle.BundleDir)
	fmt.Fprintf(&sb, "- **Frozen Specification:** `%s/spec.md` (requirements and plan)\n", bundle.BundleDir)
	fmt.Fprintf(&sb, "- **Git Diff Patch:** `%s/diff.patch` (working tree changes and untracked files)\n", bundle.BundleDir)
	fmt.Fprintf(&sb, "- **Review Rubric:** `%s/rubric.md` (canonical role skill instructions for %s)\n", bundle.BundleDir, bundle.Lens)
	fmt.Fprintf(&sb, "- **Bundle Manifest:** `%s/manifest.json`\n\n", bundle.BundleDir)

	sb.WriteString("## Review Rubric (3-Tier Severity)\n")
	sb.WriteString("- 🔴 **Blocker (Developer Attention Needed):** Spec violation (`requirements.md` not met), plan drift (files touched outside `plan.md`), security hazard (secrets/unvalidated inputs), test gaps, unapproved dependency. MUST BE RESOLVED before commit.\n")
	sb.WriteString("- 🟡 **Warning (Attention Suggested):** Sub-optimal pattern, edge case risk, missing doc or comment.\n")
	sb.WriteString("- 🟢 **Minor (Safe / Informational):** Style, minor naming or non-critical improvement.\n\n")

	sb.WriteString("## Review Instructions\n")
	fmt.Fprintf(&sb, "1. Inspect `%s/diff.patch` against `%s/spec.md` using the rubric in `%s/rubric.md`.\n", bundle.BundleDir, bundle.BundleDir, bundle.BundleDir)
	fmt.Fprintf(&sb, "2. Record all findings and sign-off in `%s/review.md`.\n", bundle.TaskDir)
	sb.WriteString("   - If findings include any 🔴 Blocker, set status to `Changes requested` with specific remediation steps.\n")
	sb.WriteString("   - Per Rule F-001, all findings (🔴 Blockers, 🟡 Warnings, 🟢 Minors) must be cataloged clearly; the worker agent is required to resolve all findings or provide explicit justification before GATE 2.\n")
	sb.WriteString("   - If 0 Blockers remain, set status to `Approved` with your reviewer attribution (including your lens name).\n")

	return sb.String(), nil
}

func resolveReviewRubric(root, lens string) string {
	switch lens {
	case "spec":
		s, _ := role.ResolveSkill(root, "spec-auditor", nil)
		return s
	case "security":
		s, _ := role.ResolveSkill(root, "appsec-specialist", nil)
		return s
	case "pattern":
		s, _ := role.ResolveSkill(root, "pattern-auditor", nil)
		return s
	default:
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
		return strings.Join(parts, "\n\n---\n\n")
	}
}

// extractGitDiff extracts tracked git diff and synthesizes new file diffs for untracked files.
func extractGitDiff(root string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "diff", "HEAD")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		// Fallback to git diff without HEAD if initial repository commit
		cmd2 := exec.CommandContext(ctx, "git", "diff")
		cmd2.Dir = root
		out, err = cmd2.Output()
		if err != nil {
			return "", err
		}
	}

	var sb strings.Builder
	sb.Write(out)

	// Check untracked files via git status --porcelain
	statusCmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	statusCmd.Dir = root
	statusOut, sErr := statusCmd.Output()
	if sErr == nil {
		lines := strings.Split(string(statusOut), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "?? ") {
				relPath := strings.TrimSpace(strings.TrimPrefix(line, "?? "))
				relPath = strings.Trim(relPath, "\"")
				// Ignore internal or transient files
				if strings.HasPrefix(relPath, ".sdd/") ||
					strings.HasPrefix(relPath, ".git/") ||
					strings.HasPrefix(relPath, "bin/") ||
					strings.HasPrefix(relPath, "dist/") {
					continue
				}

				absPath := filepath.Join(root, filepath.FromSlash(relPath))
				fi, statErr := os.Lstat(absPath)
				// Reject directories and symlinks to prevent traversal outside workspace (SEC-001)
				if statErr != nil || fi.IsDir() || (fi.Mode()&os.ModeSymlink != 0) {
					continue
				}

				// Bound file size on untracked files (SEC-001)
				if fi.Size() > maxUntrackedFileSize {
					fmt.Fprintf(&sb, "\n# (untracked file %s exceeds 1MB limit; skipped)\n", relPath)
					continue
				}

				data, readErr := os.ReadFile(absPath)
				if readErr != nil {
					continue
				}

				// Check for binary content (SEC-001)
				if bytes.IndexByte(data, 0) != -1 {
					fmt.Fprintf(&sb, "\ndiff --git a/%s b/%s\nnew file mode 100644\nBinary files /dev/null and b/%s differ\n", relPath, relPath, relPath)
					continue
				}

				// Synthesize standard unified diff for untracked new file
				fileLines := strings.Split(string(data), "\n")
				numLines := len(fileLines)
				if numLines == 0 {
					numLines = 1
				}

				fmt.Fprintf(&sb, "\ndiff --git a/%s b/%s\n", relPath, relPath)
				sb.WriteString("new file mode 100644\n")
				sb.WriteString("--- /dev/null\n")
				fmt.Fprintf(&sb, "+++ b/%s\n", relPath)
				fmt.Fprintf(&sb, "@@ -0,0 +1,%d @@\n", numLines)
				for _, fl := range fileLines {
					sb.WriteString("+" + fl + "\n")
				}
			}
		}
	}

	res := sb.String()
	if strings.TrimSpace(res) == "" {
		return "# (no git diff or untracked changes detected)\n", nil
	}
	return res, nil
}

func readDoc(dir, name string) string {
	b, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return ""
	}
	return string(b)
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
	for _, t := range typologies {
		base := filepath.Join(root, t.dir)
		entries, rErr := os.ReadDir(base)
		if rErr != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() && (e.Name() == taskID || strings.HasPrefix(e.Name(), prefix)) {
				return filepath.ToSlash(filepath.Join(t.dir, e.Name())), t.name, nil
			}
		}
	}

	return "", "", fmt.Errorf("task %s not found in any tasks/ or legacy typology directory", taskID)
}
