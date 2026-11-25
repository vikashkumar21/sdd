package audit

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/vikashkumar21/sdd/internal/fsutil"
	"github.com/vikashkumar21/sdd/internal/role"
)

const maxUntrackedFileSize = 1024 * 1024 // 1MB limit for untracked files in diffs
const maxDocSize = 1024 * 1024           // 1MB limit for task documents

var taskIDPattern = regexp.MustCompile(`^T-\d{3,}$`)
var reqIDPattern = regexp.MustCompile(`^R-\d{3,}$`)

// RequirementItem represents a functional or non-functional requirement extracted from requirements.md.
type RequirementItem struct {
	ID           string `json:"id"`
	Statement    string `json:"statement"`
	Priority     string `json:"priority,omitempty"`
	Verification string `json:"verification,omitempty"`
}

// ExtractRequirements parses markdown content of requirements.md and extracts all R-xxx items.
func ExtractRequirements(content string) ([]RequirementItem, error) {
	if strings.TrimSpace(content) == "" {
		return nil, nil
	}

	var reqs []RequirementItem
	seen := make(map[string]bool)

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "|") {
			continue
		}
		parts := strings.Split(trimmed, "|")
		if len(parts) < 3 {
			continue
		}
		id := strings.TrimSpace(parts[1])
		if !reqIDPattern.MatchString(id) {
			continue
		}
		if seen[id] {
			continue
		}
		seen[id] = true

		statement := strings.TrimSpace(parts[2])
		priority := ""
		if len(parts) > 3 {
			priority = strings.TrimSpace(parts[3])
		}
		verification := ""
		if len(parts) > 4 {
			verification = strings.TrimSpace(parts[4])
		}

		reqs = append(reqs, RequirementItem{
			ID:           id,
			Statement:    statement,
			Priority:     priority,
			Verification: verification,
		})
	}

	sort.Slice(reqs, func(i, j int) bool {
		var numA, numB int
		_, errA := fmt.Sscanf(reqs[i].ID, "R-%d", &numA)
		_, errB := fmt.Sscanf(reqs[j].ID, "R-%d", &numB)
		if errA == nil && errB == nil && numA != numB {
			return numA < numB
		}
		return reqs[i].ID < reqs[j].ID
	})

	return reqs, nil
}

// FormatTraceabilityLedger formats requirements into an unanchored markdown ledger template.
func FormatTraceabilityLedger(reqs []RequirementItem) string {
	var sb strings.Builder
	sb.WriteString("# Requirements Traceability Ledger\n\n")
	sb.WriteString("> **Omission-First Review Protocol:** Complete this ledger by inspecting candidate code and test symbols BEFORE reading `diff.patch`. Any unmapped requirement or missing test is an immediate 🔴 Blocker.\n\n")
	sb.WriteString("| Req ID | Requirement Statement | Code Symbol & File | Positive Test Evidence | Negative / Boundary Test Evidence | Status |\n")
	sb.WriteString("|---|---|---|---|---|---|\n")

	for _, req := range reqs {
		statement := strings.ReplaceAll(req.Statement, "\n", " ")
		statement = strings.ReplaceAll(statement, "|", `\|`)
		fmt.Fprintf(&sb, "| %s | %s | | | | Open |\n", req.ID, statement)
	}

	return sb.String()
}

// FindingItem represents an itemized review finding needing verification.
type FindingItem struct {
	ID       string `json:"id"`
	Category string `json:"category,omitempty"`
	Severity string `json:"severity"`
	Finding  string `json:"finding"`
	Action   string `json:"action,omitempty"`
	Status   string `json:"status"`
}

// ExtractFindingsToVerify parses review.md markdown and extracts findings requiring verification.
func ExtractFindingsToVerify(content string) ([]FindingItem, error) {
	if strings.TrimSpace(content) == "" {
		return nil, nil
	}

	var findings []FindingItem
	scanner := bufio.NewScanner(strings.NewReader(content))
	var currentSection string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") {
			currentSection = strings.ToLower(line)
			continue
		}

		if strings.HasPrefix(line, "|") && !strings.Contains(line, "---") {
			// Only parse tables within or under Risk & Findings section
			if strings.Contains(currentSection, "traceability") {
				continue
			}
			parts := strings.Split(line, "|")
			if len(parts) >= 8 {
				id := strings.TrimSpace(parts[1])
				category := strings.TrimSpace(parts[2])
				severity := strings.TrimSpace(parts[3])
				finding := strings.TrimSpace(parts[4])
				action := strings.TrimSpace(parts[5])
				status := strings.TrimSpace(parts[6])

				// Skip header or placeholder row
				if strings.EqualFold(id, "id") || strings.EqualFold(id, "req id") || strings.EqualFold(id, "(none)") || id == "" {
					continue
				}
				if strings.EqualFold(severity, "severity") || strings.Contains(severity, "---") {
					continue
				}

				findings = append(findings, FindingItem{
					ID:       id,
					Category: category,
					Severity: severity,
					Finding:  finding,
					Action:   action,
					Status:   status,
				})
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return findings, nil
}

// FormatFindingsToVerify formats findings into a clean, objective checklist markdown table.
func FormatFindingsToVerify(findings []FindingItem, loop ...int) string {
	l := 2
	if len(loop) > 0 && loop[0] > 1 {
		l = loop[0]
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "# Previous Findings to Verify (Remediation Loop %d)\n\n", l)
	sb.WriteString("> **Zero-Negotiation Isolation:** The candidate code below must be evaluated objectively against requirements. Verify whether previous defect conditions are resolved without conversational bias.\n\n")
	sb.WriteString("| Finding ID | Category | Severity | Defect Description | Suggested Action | Verification Status |\n")
	sb.WriteString("|---|---|---|---|---|---|\n")

	for _, f := range findings {
		desc := strings.ReplaceAll(f.Finding, "\n", " ")
		desc = strings.ReplaceAll(desc, "|", `\|`)
		act := strings.ReplaceAll(f.Action, "\n", " ")
		act = strings.ReplaceAll(act, "|", `\|`)
		fmt.Fprintf(&sb, "| %s | %s | %s | %s | %s | Pending Verification |\n", f.ID, f.Category, f.Severity, desc, act)
	}

	return sb.String()
}

// PlanBundle contains staged metadata for an isolated plan review.
type PlanBundle struct {
	TaskID    string    `json:"task_id"`
	TaskDir   string    `json:"task_dir"`
	BundleDir string    `json:"bundle_dir"`
	CreatedAt time.Time `json:"created_at"`
	Files     []string  `json:"files"`
}

// DiscoveryBundle contains staged metadata for an isolated brownfield discovery review.
type DiscoveryBundle struct {
	TaskID    string    `json:"task_id"`
	TaskDir   string    `json:"task_dir"`
	BundleDir string    `json:"bundle_dir"`
	CreatedAt time.Time `json:"created_at"`
	Files     []string  `json:"files"`
}

// ReviewBundle contains staged metadata for an isolated code review triad.
type ReviewBundle struct {
	TaskID             string    `json:"task_id"`
	Lens               string    `json:"lens"`
	Loop               int       `json:"loop,omitempty"`
	TaskDir            string    `json:"task_dir"`
	BundleDir          string    `json:"bundle_dir"`
	CreatedAt          time.Time `json:"created_at"`
	Files              []string  `json:"files"`
	RawDiffBytes       int       `json:"raw_diff_bytes"`
	FilteredDiffBytes  int       `json:"filtered_diff_bytes"`
	DiffSavingsPercent float64   `json:"diff_savings_percent"`
	RawFilesCount      int       `json:"raw_files_count"`
	FilteredFilesCount int       `json:"filtered_files_count"`
}

// EnsureGitignoreReview ensures .sdd/review/ is ignored in the root .gitignore.
func EnsureGitignoreReview(root string) error {
	rootHandle, err := os.OpenRoot(root)
	if err != nil {
		return fmt.Errorf("opening root: %w", err)
	}
	defer func() { _ = rootHandle.Close() }()

	content, err := rootHandle.ReadFile(".gitignore")
	if err != nil && !os.IsNotExist(err) && !errors.Is(err, fs.ErrNotExist) {
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

	return fsutil.WriteAtomicRoot(rootHandle, ".gitignore", buf.Bytes(), 0o644)
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

	rootHandle, err := os.OpenRoot(root)
	if err != nil {
		return nil, fmt.Errorf("opening root: %w", err)
	}
	defer func() { _ = rootHandle.Close() }()

	bundleDirRel := filepath.Join(".sdd", "review", normTaskID, "plan")

	fi, statErr := rootHandle.Lstat(filepath.FromSlash(taskRelDir))
	if statErr != nil || !fi.IsDir() || (fi.Mode()&os.ModeSymlink != 0) {
		return nil, fmt.Errorf("task directory is invalid or symlink: %s", taskRelDir)
	}
	reqContent := readDoc(taskRelDir, "requirements.md", rootHandle)
	planContent := readDoc(taskRelDir, "plan.md", rootHandle)
	if planContent == "" {
		planContent = readDoc(taskRelDir, "patch.md", rootHandle)
	}
	valContent := readDoc(taskRelDir, "validation.md", rootHandle)

	// 1. spec.md: combines requirements, plan, and validation matrix
	var specBuf bytes.Buffer
	fmt.Fprintf(&specBuf, "# Specification Bundle: %s (Plan Review)\n\n", normTaskID)
	specBuf.WriteString("## Requirements (requirements.md)\n\n")
	specBuf.WriteString(strings.TrimSpace(reqContent))
	specBuf.WriteString("\n\n---\n\n## Implementation Plan (plan.md)\n\n")
	specBuf.WriteString(strings.TrimSpace(planContent))
	if strings.TrimSpace(valContent) != "" {
		specBuf.WriteString("\n\n---\n\n## Validation Strategy (validation.md)\n\n")
		specBuf.WriteString(strings.TrimSpace(valContent))
	}
	specBuf.WriteString("\n")

	specRel := filepath.Join(bundleDirRel, "spec.md")
	if err := fsutil.WriteAtomicRoot(rootHandle, filepath.FromSlash(specRel), specBuf.Bytes(), 0o644); err != nil {
		return nil, fmt.Errorf("writing spec.md: %w", err)
	}

	// 2. rubric.md: plan-reviewer role skill
	roleSkill, err := role.ResolveSkill(root, "plan-reviewer", nil)
	if err != nil {
		return nil, fmt.Errorf("resolving plan-reviewer skill: %w", err)
	}
	rubricRel := filepath.Join(bundleDirRel, "rubric.md")
	if err := fsutil.WriteAtomicRoot(rootHandle, filepath.FromSlash(rubricRel), []byte(strings.TrimSpace(roleSkill)+"\n"), 0o644); err != nil {
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
	manifestRel := filepath.Join(bundleDirRel, "manifest.json")
	if err := fsutil.WriteAtomicRoot(rootHandle, filepath.FromSlash(manifestRel), manifestData, 0o644); err != nil {
		return nil, fmt.Errorf("writing manifest.json: %w", err)
	}
	bundle.Files = append(bundle.Files, filepath.ToSlash(filepath.Join(bundleDirRel, "manifest.json")))

	return bundle, nil
}

// StageDiscoveryBundle stages an isolated, frozen review bundle for Phase Discovery challenger audit.
func StageDiscoveryBundle(root, taskID, findingsContent, rubricContent string, now ...time.Time) (*DiscoveryBundle, error) {
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

	rootHandle, err := os.OpenRoot(root)
	if err != nil {
		return nil, fmt.Errorf("opening root: %w", err)
	}
	defer func() { _ = rootHandle.Close() }()

	bundleDirRel := filepath.Join(".sdd", "review", normTaskID, "discovery")

	fi, statErr := rootHandle.Lstat(filepath.FromSlash(taskRelDir))
	if statErr != nil || !fi.IsDir() || (fi.Mode()&os.ModeSymlink != 0) {
		return nil, fmt.Errorf("task directory is invalid or symlink: %s", taskRelDir)
	}
	reqContent := readDoc(taskRelDir, "requirements.md", rootHandle)
	planContent := readDoc(taskRelDir, "plan.md", rootHandle)
	if planContent == "" {
		planContent = readDoc(taskRelDir, "patch.md", rootHandle)
	}

	// 1. findings.md: combines findingsContent, requirements, and plan
	var findingsBuf bytes.Buffer
	fmt.Fprintf(&findingsBuf, "# Discovery Findings Bundle: %s\n\n", normTaskID)
	if strings.TrimSpace(findingsContent) != "" {
		findingsBuf.WriteString("## Discovery Report & Synthesis\n\n")
		findingsBuf.WriteString(strings.TrimSpace(findingsContent))
		findingsBuf.WriteString("\n\n---\n\n")
	}
	if strings.TrimSpace(reqContent) != "" {
		findingsBuf.WriteString("## Requirements (requirements.md)\n\n")
		findingsBuf.WriteString(strings.TrimSpace(reqContent))
		findingsBuf.WriteString("\n\n---\n\n")
	}
	if strings.TrimSpace(planContent) != "" {
		findingsBuf.WriteString("## Implementation Plan (plan.md)\n\n")
		findingsBuf.WriteString(strings.TrimSpace(planContent))
		findingsBuf.WriteString("\n")
	}

	findingsRel := filepath.Join(bundleDirRel, "findings.md")
	if err := fsutil.WriteAtomicRoot(rootHandle, filepath.FromSlash(findingsRel), findingsBuf.Bytes(), 0o644); err != nil {
		return nil, fmt.Errorf("writing findings.md: %w", err)
	}

	// 2. rubric.md: discovery-challenger role skill
	resolvedRubric := strings.TrimSpace(rubricContent)
	if resolvedRubric == "" {
		skill, err := role.ResolveSkill(root, "discovery-challenger", nil)
		if err != nil {
			return nil, fmt.Errorf("resolving discovery-challenger skill: %w", err)
		}
		resolvedRubric = skill
	}
	rubricRel := filepath.Join(bundleDirRel, "rubric.md")
	if err := fsutil.WriteAtomicRoot(rootHandle, filepath.FromSlash(rubricRel), []byte(strings.TrimSpace(resolvedRubric)+"\n"), 0o644); err != nil {
		return nil, fmt.Errorf("writing rubric.md: %w", err)
	}

	stagedFiles := []string{
		filepath.ToSlash(filepath.Join(bundleDirRel, "findings.md")),
		filepath.ToSlash(filepath.Join(bundleDirRel, "rubric.md")),
	}

	// Stage external reference context from .sdd/context/ if present
	contextDirRel := filepath.ToSlash(filepath.Join(".sdd", "context"))
	if _, err := fs.Stat(rootHandle.FS(), contextDirRel); err == nil {
		var ctxFiles []string
		_ = fs.WalkDir(rootHandle.FS(), contextDirRel, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d == nil || d.IsDir() {
				return nil
			}
			ctxFiles = append(ctxFiles, path)
			return nil
		})
		sort.Strings(ctxFiles)
		for _, srcPath := range ctxFiles {
			relInContext := strings.TrimPrefix(srcPath, contextDirRel)
			relInContext = strings.TrimPrefix(relInContext, "/")
			if relInContext == "" {
				continue
			}
			destRel := filepath.Join(bundleDirRel, "context", filepath.FromSlash(relInContext))
			content, err := rootHandle.ReadFile(filepath.FromSlash(srcPath))
			if err != nil {
				continue
			}
			if err := fsutil.WriteAtomicRoot(rootHandle, filepath.FromSlash(destRel), content, 0o644); err != nil {
				return nil, fmt.Errorf("writing discovery context file %s: %w", destRel, err)
			}
			stagedFiles = append(stagedFiles, filepath.ToSlash(destRel))
		}
	}

	createdAt := time.Now().UTC()
	if len(now) > 0 && !now[0].IsZero() {
		createdAt = now[0].UTC()
	}

	// 3. manifest.json
	bundle := &DiscoveryBundle{
		TaskID:    normTaskID,
		TaskDir:   taskRelDir,
		BundleDir: filepath.ToSlash(bundleDirRel),
		CreatedAt: createdAt,
		Files:     stagedFiles,
	}

	manifestData, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling discovery manifest: %w", err)
	}
	manifestRel := filepath.Join(bundleDirRel, "manifest.json")
	if err := fsutil.WriteAtomicRoot(rootHandle, filepath.FromSlash(manifestRel), manifestData, 0o644); err != nil {
		return nil, fmt.Errorf("writing manifest.json: %w", err)
	}
	bundle.Files = append(bundle.Files, filepath.ToSlash(filepath.Join(bundleDirRel, "manifest.json")))

	return bundle, nil
}

// StageReviewBundle stages an isolated, frozen review bundle for Phase C.2 code review triad.
func StageReviewBundle(root, taskID, lens string, now ...time.Time) (*ReviewBundle, error) {
	return StageReviewBundleWithLoop(root, taskID, lens, 1, now...)
}

// StageReviewBundleWithLoop stages an isolated, frozen review bundle for a specific remediation loop.
func StageReviewBundleWithLoop(root, taskID, lens string, loop int, now ...time.Time) (*ReviewBundle, error) {
	if loop < 1 || loop > 3 {
		return nil, fmt.Errorf("invalid loop %d (expected 1, 2, or 3 per 3-loop circuit breaker)", loop)
	}

	normTaskID := strings.ToUpper(strings.TrimSpace(taskID))
	if !taskIDPattern.MatchString(normTaskID) {
		return nil, fmt.Errorf("invalid task ID format %q (expected format like T-001)", taskID)
	}

	normLens := NormalizeLens(lens)

	taskRelDir, _, err := findTaskDirectory(root, normTaskID)
	if err != nil {
		return nil, err
	}

	if err := EnsureGitignoreReview(root); err != nil {
		return nil, fmt.Errorf("ensuring .gitignore review entry: %w", err)
	}

	rootHandle, err := os.OpenRoot(root)
	if err != nil {
		return nil, fmt.Errorf("opening root: %w", err)
	}
	defer func() { _ = rootHandle.Close() }()

	bundleSubdir := normLens
	if normLens == "all" {
		bundleSubdir = "code"
	}
	bundleDirRel := filepath.Join(".sdd", "review", normTaskID, bundleSubdir)

	fi, statErr := rootHandle.Lstat(filepath.FromSlash(taskRelDir))
	if statErr != nil || !fi.IsDir() || (fi.Mode()&os.ModeSymlink != 0) {
		return nil, fmt.Errorf("task directory is invalid or symlink: %s", taskRelDir)
	}
	reqContent := readDoc(taskRelDir, "requirements.md", rootHandle)
	planContent := readDoc(taskRelDir, "plan.md", rootHandle)
	if planContent == "" {
		planContent = readDoc(taskRelDir, "patch.md", rootHandle)
	}

	// 1. spec.md
	var specBuf bytes.Buffer
	fmt.Fprintf(&specBuf, "# Specification Bundle: %s (Code Review Triad)\n\n", normTaskID)
	specBuf.WriteString("## Requirements (requirements.md)\n\n")
	specBuf.WriteString(strings.TrimSpace(reqContent))
	specBuf.WriteString("\n\n---\n\n## Implementation Plan (plan.md)\n\n")
	specBuf.WriteString(strings.TrimSpace(planContent))
	specBuf.WriteString("\n")

	specRel := filepath.Join(bundleDirRel, "spec.md")
	if err := fsutil.WriteAtomicRoot(rootHandle, filepath.FromSlash(specRel), specBuf.Bytes(), 0o644); err != nil {
		return nil, fmt.Errorf("writing spec.md: %w", err)
	}

	// 2. ledger.md: unanchored requirements traceability ledger
	reqItems, _ := ExtractRequirements(reqContent)
	ledgerText := FormatTraceabilityLedger(reqItems)
	ledgerRel := filepath.Join(bundleDirRel, "ledger.md")
	if err := fsutil.WriteAtomicRoot(rootHandle, filepath.FromSlash(ledgerRel), []byte(ledgerText), 0o644); err != nil {
		return nil, fmt.Errorf("writing ledger.md: %w", err)
	}

	// 3. diff.patch: git diff filtered by lens
	rawDiffText, err := extractGitDiff(root)
	if err != nil {
		rawDiffText = fmt.Sprintf("# Note: could not extract git diff: %v\n", err)
	}
	diffText, rawBytes, filteredBytes, rawFiles, filteredFiles, savingsPercent := FilterGitDiffByLens(rawDiffText, normLens)
	diffRel := filepath.Join(bundleDirRel, "diff.patch")
	if err := fsutil.WriteAtomicRoot(rootHandle, filepath.FromSlash(diffRel), []byte(diffText), 0o644); err != nil {
		return nil, fmt.Errorf("writing diff.patch: %w", err)
	}

	// 4. rubric.md: role skill for lens
	roleSkill := resolveReviewRubric(root, normLens)
	rubricRel := filepath.Join(bundleDirRel, "rubric.md")
	if err := fsutil.WriteAtomicRoot(rootHandle, filepath.FromSlash(rubricRel), []byte(strings.TrimSpace(roleSkill)+"\n"), 0o644); err != nil {
		return nil, fmt.Errorf("writing rubric.md: %w", err)
	}

	stagedFiles := []string{
		filepath.ToSlash(filepath.Join(bundleDirRel, "spec.md")),
		filepath.ToSlash(filepath.Join(bundleDirRel, "ledger.md")),
		filepath.ToSlash(filepath.Join(bundleDirRel, "diff.patch")),
		filepath.ToSlash(filepath.Join(bundleDirRel, "rubric.md")),
	}

	// 5. findings_to_verify.md: staged for remediation loops (loop > 1)
	if loop > 1 {
		reviewContent := readDoc(taskRelDir, "review.md", rootHandle)
		findings, _ := ExtractFindingsToVerify(reviewContent)
		filteredFindings := FilterFindingsForLens(findings, normLens)
		if len(filteredFindings) > 0 {
			findingsText := FormatFindingsToVerify(filteredFindings, loop)
			findingsRel := filepath.Join(bundleDirRel, "findings_to_verify.md")
			if err := fsutil.WriteAtomicRoot(rootHandle, filepath.FromSlash(findingsRel), []byte(findingsText), 0o644); err != nil {
				return nil, fmt.Errorf("writing findings_to_verify.md: %w", err)
			}
			stagedFiles = append(stagedFiles, filepath.ToSlash(filepath.Join(bundleDirRel, "findings_to_verify.md")))
		}
	}

	createdAt := time.Now().UTC()
	if len(now) > 0 && !now[0].IsZero() {
		createdAt = now[0].UTC()
	}

	// 6. manifest.json
	bundle := &ReviewBundle{
		TaskID:             normTaskID,
		Lens:               normLens,
		Loop:               loop,
		TaskDir:            taskRelDir,
		BundleDir:          filepath.ToSlash(bundleDirRel),
		CreatedAt:          createdAt,
		Files:              stagedFiles,
		RawDiffBytes:       rawBytes,
		FilteredDiffBytes:  filteredBytes,
		DiffSavingsPercent: savingsPercent,
		RawFilesCount:      rawFiles,
		FilteredFilesCount: filteredFiles,
	}

	manifestData, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling review manifest: %w", err)
	}
	manifestRel := filepath.Join(bundleDirRel, "manifest.json")
	if err := fsutil.WriteAtomicRoot(rootHandle, filepath.FromSlash(manifestRel), manifestData, 0o644); err != nil {
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

	if bundle.Loop > 1 {
		fmt.Fprintf(&sb, "# Handover: Stateless Remediation Review (Loop %d) for %s (%s)\n\n", bundle.Loop, bundle.TaskID, lensTitle)
		fmt.Fprintf(&sb, "You are acting as an **Independent Subagent Reviewer (%s)** for task `%s` on remediation **Loop %d**.\n", lensTitle, bundle.TaskID, bundle.Loop)
		sb.WriteString("Your mandate is to impartially audit the remediated candidate implementation before GATE 2 human review.\n")
		sb.WriteString("**You are an independent, tabula-rasa reviewer evaluating this task in fresh, isolated context. Rely strictly on the staged artifacts in your review bundle without conversational bias or assumptions.**\n\n")
	} else {
		fmt.Fprintf(&sb, "# Handover: Stateless Subagent Code Review for %s (%s)\n\n", bundle.TaskID, lensTitle)
		fmt.Fprintf(&sb, "You are acting as an **Independent Subagent Reviewer (%s)** for task `%s`.\n", lensTitle, bundle.TaskID)
		sb.WriteString("Your mandate is to impartially audit the implementation before GATE 2 human review.\n")
		sb.WriteString("**You are an independent reviewer evaluating this task in fresh, isolated context. Rely strictly on the staged artifacts in your review bundle without conversational bias or assumptions.**\n\n")
	}

	sb.WriteString("## SDD Anthem & Principles\n")
	sb.WriteString("> **ALWAYS BE DETERMINISTIC. When a decision is needed, STOP and ask the human. Never guess. Never silently choose.**\n\n")

	sb.WriteString("## Staged Review Bundle Artifacts\n")
	fmt.Fprintf(&sb, "- **Bundle Directory:** `%s/`\n", bundle.BundleDir)
	fmt.Fprintf(&sb, "- **Frozen Specification:** `%s/spec.md` (requirements and plan)\n", bundle.BundleDir)
	fmt.Fprintf(&sb, "- **Traceability Ledger:** `%s/ledger.md` (unanchored requirements ledger to complete before reading diff.patch)\n", bundle.BundleDir)
	if bundle.Loop > 1 {
		fmt.Fprintf(&sb, "- **Findings to Verify:** `%s/findings_to_verify.md` (objective checklist of previous defects to verify)\n", bundle.BundleDir)
	}
	fmt.Fprintf(&sb, "- **Git Diff Patch:** `%s/diff.patch` (working tree changes and untracked files)\n", bundle.BundleDir)
	fmt.Fprintf(&sb, "- **Review Rubric:** `%s/rubric.md` (canonical role skill instructions for %s)\n", bundle.BundleDir, bundle.Lens)
	fmt.Fprintf(&sb, "- **Bundle Manifest:** `%s/manifest.json`\n\n", bundle.BundleDir)

	sb.WriteString("## Review Rubric (3-Tier Severity)\n")
	sb.WriteString("- 🔴 **Blocker (Developer Attention Needed):** Spec violation (`requirements.md` not met), plan drift (files touched outside `plan.md`), security hazard (secrets/unvalidated inputs), test gaps, unapproved dependency. MUST BE RESOLVED before commit.\n")
	sb.WriteString("- 🟡 **Warning (Attention Suggested):** Sub-optimal pattern, edge case risk, missing doc or comment.\n")
	sb.WriteString("- 🟢 **Minor (Safe / Informational):** Style, minor naming or non-critical improvement.\n\n")

	if bundle.Loop > 1 {
		sb.WriteString("## Review Instructions (Remediation Loop)\n")
		fmt.Fprintf(&sb, "1. **Unanchored Traceability Scan (Step 1):** Open `%s/ledger.md` and verify each requirement against the codebase symbols and test suites before inspecting `%s/diff.patch`. Fill in Code Symbol, Positive Test, and Negative / Boundary Test Evidence. Any unmapped requirement or missing test is an immediate 🔴 Blocker.\n", bundle.BundleDir, bundle.BundleDir)
		fmt.Fprintf(&sb, "2. **Defect Remediation Verification (Step 2):** Open `%s/findings_to_verify.md` and inspect `%s/diff.patch` to verify whether each previous defect has been genuinely resolved in code. Do not accept conversational developer explanations or unverified assertions.\n", bundle.BundleDir, bundle.BundleDir)
		fmt.Fprintf(&sb, "3. **Diff Quality Audit (Step 3):** Inspect `%s/diff.patch` against `%s/spec.md` using the rubric in `%s/rubric.md` to ensure no new regressions, plan drift, or security flaws were introduced during remediation.\n", bundle.BundleDir, bundle.BundleDir, bundle.BundleDir)
		fmt.Fprintf(&sb, "4. Output all findings, the completed Requirements Traceability Ledger, and sign-off in your review report (the primary worker agent will batch all reviewer findings into `%s/review.md`).\n", bundle.TaskDir)
		sb.WriteString("   - If findings include any 🔴 Blocker, set status to `Changes requested` with specific remediation steps.\n")
		sb.WriteString("   - Per Rule F-001, all findings (🔴 Blockers, 🟡 Warnings, 🟢 Minors) must be cataloged clearly; the worker agent is required to resolve all findings or provide explicit justification before GATE 2.\n")
		sb.WriteString("   - If 0 Blockers remain, set status to `Approved` with your reviewer attribution (including your lens name).\n")
	} else {
		sb.WriteString("## Review Instructions\n")
		fmt.Fprintf(&sb, "1. **Unanchored Traceability Scan (Step 1):** Open `%s/ledger.md` and verify each requirement against the codebase symbols and test suites before inspecting `%s/diff.patch`. Fill in Code Symbol, Positive Test, and Negative / Boundary Test Evidence. Any unmapped requirement or missing test is an immediate 🔴 Blocker.\n", bundle.BundleDir, bundle.BundleDir)
		fmt.Fprintf(&sb, "2. **Diff Quality Audit (Step 2):** Inspect `%s/diff.patch` against `%s/spec.md` using the rubric in `%s/rubric.md`.\n", bundle.BundleDir, bundle.BundleDir, bundle.BundleDir)
		fmt.Fprintf(&sb, "3. Output all findings, the completed Requirements Traceability Ledger, and sign-off in your review report (the primary worker agent will batch all reviewer findings into `%s/review.md`).\n", bundle.TaskDir)
		sb.WriteString("   - If findings include any 🔴 Blocker, set status to `Changes requested` with specific remediation steps.\n")
		sb.WriteString("   - Per Rule F-001, all findings (🔴 Blockers, 🟡 Warnings, 🟢 Minors) must be cataloged clearly; the worker agent is required to resolve all findings or provide explicit justification before GATE 2.\n")
		sb.WriteString("   - If 0 Blockers remain, set status to `Approved` with your reviewer attribution (including your lens name).\n")
	}

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

	// If working tree diff against HEAD is empty, check branch diff against main
	if len(bytes.TrimSpace(out)) == 0 {
		cmdBranch := exec.CommandContext(ctx, "git", "diff", "main...HEAD")
		cmdBranch.Dir = root
		if bOut, bErr := cmdBranch.Output(); bErr == nil && len(bytes.TrimSpace(bOut)) > 0 {
			out = bOut
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

func readDoc(dir, name string, rootHandle ...*os.Root) string {
	if len(rootHandle) > 0 && rootHandle[0] != nil {
		relPath := filepath.Join(filepath.FromSlash(dir), name)
		fi, err := rootHandle[0].Lstat(relPath)
		if err != nil || fi.IsDir() || (fi.Mode()&os.ModeSymlink != 0) || fi.Size() > maxDocSize {
			return ""
		}
		b, err := rootHandle[0].ReadFile(relPath)
		if err != nil {
			return ""
		}
		return string(b)
	}
	target := filepath.Join(dir, name)
	fi, err := os.Lstat(target)
	if err != nil || fi.IsDir() || (fi.Mode()&os.ModeSymlink != 0) || fi.Size() > maxDocSize {
		return ""
	}
	b, err := os.ReadFile(target)
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
			if (e.IsDir() || e.Type()&fs.ModeSymlink != 0) && (e.Name() == taskID || strings.HasPrefix(e.Name(), prefix)) {
				return filepath.ToSlash(filepath.Join(t.dir, e.Name())), t.name, nil
			}
		}
	}

	return "", "", fmt.Errorf("task %s not found in any tasks/ or legacy typology directory", taskID)
}
