package audit

import (
	"bufio"
	"fmt"
	"path/filepath"
	"strings"
)

// DiffFileHunk represents a single file's diff patch within a unified git diff.
type DiffFileHunk struct {
	OldPath string
	NewPath string
	Header  string
	Body    string
	Raw     string
}

// TargetPath returns the effective path being evaluated (NewPath unless file was deleted).
func (h *DiffFileHunk) TargetPath() string {
	if h.NewPath != "" && h.NewPath != "/dev/null" {
		return h.NewPath
	}
	return h.OldPath
}

// FilterGitDiffByLens filters a raw unified git diff according to the given lens
// (spec, security, pattern, all) and computes token/byte savings metrics.
func FilterGitDiffByLens(rawDiff, lens string) (filteredDiff string, rawBytes, filteredBytes, rawFiles, filteredFiles int, savingsPercent float64) {
	normLens := strings.ToLower(strings.TrimSpace(lens))
	if normLens == "appsec" {
		normLens = "security"
	}
	if normLens == "" {
		normLens = "all"
	}

	rawBytes = len(rawDiff)
	hunks := splitDiffIntoHunks(rawDiff)
	rawFiles = len(hunks)

	if normLens == "all" || rawFiles == 0 {
		return rawDiff, rawBytes, rawBytes, rawFiles, rawFiles, 0.0
	}

	var sb strings.Builder
	for _, h := range hunks {
		path := h.TargetPath()
		switch normLens {
		case "security":
			if isSecurityRelevant(path) {
				sb.WriteString(h.Raw)
				filteredFiles++
			}
		case "pattern":
			if isPatternRelevant(path) {
				sb.WriteString(h.Raw)
				filteredFiles++
			}
		case "spec":
			if isSpecDirectlyRelevant(path) {
				sb.WriteString(h.Raw)
				filteredFiles++
			} else {
				// Retain header for boundary verification, elide bloated diff body
				sb.WriteString(h.Header)
				sb.WriteString("\n# [spec lens: non-code diff body elided for boundary verification]\n\n")
				filteredFiles++
			}
		default:
			// Fallback: retain full hunk for any custom/unrecognized lens
			sb.WriteString(h.Raw)
			filteredFiles++
		}
	}

	if filteredFiles == 0 {
		filteredDiff = fmt.Sprintf("# Note: all changes in this task are outside %s lens focus; no lens-specific diffs detected.\n", normLens)
	} else {
		filteredDiff = sb.String()
	}

	filteredBytes = len(filteredDiff)
	if rawBytes > 0 && rawBytes > filteredBytes {
		savingsPercent = float64(rawBytes-filteredBytes) / float64(rawBytes) * 100.0
	} else {
		savingsPercent = 0.0
	}

	return filteredDiff, rawBytes, filteredBytes, rawFiles, filteredFiles, savingsPercent
}

// isSecurityRelevant determines whether a file has executable logic, configuration,
// dependency declarations, or security surface requiring AppSec scrutiny.
func isSecurityRelevant(path string) bool {
	p := filepath.ToSlash(filepath.Clean(path))
	pLower := strings.ToLower(p)

	// Exclude pure task documentation files and documentation markdown
	if strings.HasPrefix(pLower, "tasks/") && strings.HasSuffix(pLower, ".md") {
		return false
	}
	if strings.HasPrefix(pLower, "docs/") && strings.HasSuffix(pLower, ".md") {
		return false
	}
	if pLower == "readme.md" || pLower == "roadmap.md" || pLower == "mission.md" {
		return false
	}

	// Living governance contracts and security/feedback rules
	if pLower == "claude.md" || pLower == "gemini.md" || pLower == "agents.md" {
		return true
	}
	if strings.HasPrefix(pLower, ".sdd/") {
		// Include roles and manifest, ignore backup or review files
		if strings.HasSuffix(pLower, "roles.json") || strings.HasSuffix(pLower, "manifest.json") {
			return true
		}
	}

	// Security, hook, and permission logic
	if strings.Contains(pLower, "/hook") || strings.Contains(pLower, "/permissions") {
		return true
	}

	// Executable source code
	ext := filepath.Ext(pLower)
	switch ext {
	case ".go", ".py", ".js", ".ts", ".jsx", ".tsx", ".java", ".c", ".cpp", ".h", ".rs", ".rb", ".php", ".cs":
		return true
	case ".sh", ".bash", ".zsh", ".ps1":
		return true
	case ".json", ".yaml", ".yml", ".toml", ".xml", ".ini":
		return true
	case ".bin", ".exe", ".so", ".dll", ".dylib", ".dat":
		return true
	}

	// Build scripts, containers, CI workflows, and dependencies
	base := filepath.Base(pLower)
	if base == "makefile" || strings.HasPrefix(base, "dockerfile") || base == "containerfile" {
		return true
	}
	if base == "go.mod" || base == "go.sum" || base == "package.json" || base == "pom.xml" || strings.HasPrefix(base, "build.gradle") {
		return true
	}
	if strings.HasPrefix(pLower, ".github/") || base == ".gitlab-ci.yml" || base == "cloudbuild.yaml" || base == "buildspec.yml" {
		return true
	}

	return false
}

// isPatternRelevant determines whether a file affects architecture decisions,
// coding standards, idioms, or determinism.
func isPatternRelevant(path string) bool {
	p := filepath.ToSlash(filepath.Clean(path))
	pLower := strings.ToLower(p)

	// Exclude task validation logs, task reviews, and roadmap changelogs
	if strings.HasPrefix(pLower, "tasks/") && (strings.HasSuffix(pLower, "validation.md") || strings.HasSuffix(pLower, "review.md")) {
		return false
	}
	if pLower == "roadmap.md" || strings.HasPrefix(pLower, "docs/") || pLower == "readme.md" || pLower == "mission.md" {
		return false
	}

	// Architecture & Standards living docs and contracts
	if pLower == "tech_stack.md" || pLower == "skills.md" || pLower == "claude.md" || pLower == "gemini.md" || pLower == "agents.md" {
		return true
	}
	if strings.HasPrefix(pLower, ".sdd/roles.json") {
		return true
	}

	// Source code, templates, and tests
	if strings.HasPrefix(pLower, "internal/templates/") {
		return true
	}
	ext := filepath.Ext(pLower)
	switch ext {
	case ".go", ".py", ".js", ".ts", ".java", ".rs", ".rb", ".cs":
		return true
	case ".json", ".yaml", ".yml", ".toml":
		return true
	case ".bin", ".dat":
		return true
	}

	return false
}

// isSpecDirectlyRelevant determines whether a file contains core functional logic,
// unit/regression tests, or task specifications.
func isSpecDirectlyRelevant(path string) bool {
	p := filepath.ToSlash(filepath.Clean(path))
	pLower := strings.ToLower(p)

	// Task requirements and implementation plans are directly relevant to spec audit
	if strings.HasPrefix(pLower, "tasks/") && (strings.HasSuffix(pLower, "requirements.md") || strings.HasSuffix(pLower, "plan.md")) {
		return true
	}

	// Executable source code and unit tests
	ext := filepath.Ext(pLower)
	switch ext {
	case ".go", ".py", ".js", ".ts", ".jsx", ".tsx", ".java", ".c", ".cpp", ".rs", ".rb", ".cs":
		return true
	case ".bin", ".dat":
		return true
	}

	return false
}

// splitDiffIntoHunks parses a unified git diff into itemized file hunks.
func splitDiffIntoHunks(rawDiff string) []DiffFileHunk {
	var hunks []DiffFileHunk
	if strings.TrimSpace(rawDiff) == "" {
		return hunks
	}

	scanner := bufio.NewScanner(strings.NewReader(rawDiff))
	var currentHunk *DiffFileHunk
	var currentRaw strings.Builder
	var currentHeader strings.Builder
	var inHeader bool

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "diff --git ") {
			if currentHunk != nil {
				currentHunk.Raw = currentRaw.String()
				currentHunk.Header = strings.TrimRight(currentHeader.String(), "\n")
				hunks = append(hunks, *currentHunk)
				currentRaw.Reset()
				currentHeader.Reset()
			}

			oldPath, newPath := parseDiffGitPaths(line)
			currentHunk = &DiffFileHunk{
				OldPath: oldPath,
				NewPath: newPath,
			}
			inHeader = true
		}

		if currentHunk != nil {
			currentRaw.WriteString(line + "\n")
			if inHeader {
				currentHeader.WriteString(line + "\n")
				if strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "Binary files ") {
					inHeader = false
				}
			}
		}
	}

	if currentHunk != nil {
		currentHunk.Raw = currentRaw.String()
		currentHunk.Header = strings.TrimRight(currentHeader.String(), "\n")
		hunks = append(hunks, *currentHunk)
	}

	return hunks
}

// parseDiffGitPaths extracts old and new paths from a "diff --git a/<old> b/<new>" header.
func parseDiffGitPaths(line string) (oldPath, newPath string) {
	trimmed := strings.TrimPrefix(line, "diff --git ")
	parts := strings.Split(trimmed, " ")
	if len(parts) >= 2 {
		oldPath = strings.TrimPrefix(parts[0], "a/")
		newPath = strings.TrimPrefix(parts[1], "b/")
		oldPath = strings.Trim(oldPath, "\"")
		newPath = strings.Trim(newPath, "\"")
	}
	return oldPath, newPath
}

// FilterFindingsForLens filters multi-loop remediation findings for a specific lens domain.
func FilterFindingsForLens(findings []FindingItem, lens string) []FindingItem {
	normLens := strings.ToLower(strings.TrimSpace(lens))
	if normLens == "appsec" {
		normLens = "security"
	}
	if normLens == "" || normLens == "all" || len(findings) == 0 {
		return findings
	}

	var filtered []FindingItem
	for _, f := range findings {
		if isFindingRelevantToLens(f, normLens) {
			filtered = append(filtered, f)
		}
	}

	return filtered
}

// isFindingRelevantToLens determines whether a finding should be audited by the target lens.
func isFindingRelevantToLens(f FindingItem, lens string) bool {
	idUpper := strings.ToUpper(f.ID)
	catLower := strings.ToLower(f.Category)

	// Explicit ID prefix matching
	hasSpecPrefix := strings.HasPrefix(idUpper, "SPEC-")
	hasSecPrefix := strings.HasPrefix(idUpper, "SEC-")
	hasPatPrefix := strings.HasPrefix(idUpper, "PAT-") || strings.HasPrefix(idUpper, "PQ-")

	switch lens {
	case "security":
		if hasSecPrefix {
			return true
		}
		if hasSpecPrefix || hasPatPrefix {
			return false
		}
		return strings.Contains(catLower, "sec") ||
			strings.Contains(catLower, "threat") ||
			strings.Contains(catLower, "safety") ||
			strings.Contains(catLower, "vulnerab") ||
			strings.Contains(catLower, "confin") ||
			strings.Contains(catLower, "secret") ||
			strings.Contains(catLower, "traversal") ||
			strings.Contains(catLower, "command") ||
			strings.Contains(catLower, "auth") ||
			isGeneralUnassigned(catLower)
	case "pattern":
		if hasPatPrefix {
			return true
		}
		if hasSpecPrefix || hasSecPrefix {
			return false
		}
		return strings.Contains(catLower, "pattern") ||
			strings.Contains(catLower, "qualit") ||
			strings.Contains(catLower, "idiom") ||
			strings.Contains(catLower, "decis") ||
			strings.Contains(catLower, "determin") ||
			strings.Contains(catLower, "lint") ||
			strings.Contains(catLower, "style") ||
			strings.Contains(catLower, "arch") ||
			isGeneralUnassigned(catLower)
	case "spec":
		if hasSpecPrefix {
			return true
		}
		if hasSecPrefix || hasPatPrefix {
			return false
		}
		return strings.Contains(catLower, "spec") ||
			strings.Contains(catLower, "traceab") ||
			strings.Contains(catLower, "boundar") ||
			strings.Contains(catLower, "omission") ||
			strings.Contains(catLower, "coverage") ||
			strings.Contains(catLower, "require") ||
			strings.Contains(catLower, "test") ||
			isGeneralUnassigned(catLower)
	default:
		return true
	}
}

// isGeneralUnassigned checks if a category is unclassified or generic (e.g. F-001, General).
func isGeneralUnassigned(cat string) bool {
	c := strings.TrimSpace(cat)
	if c == "" || c == "n/a" || c == "general" || c == "other" || c == "f-001" || c == "correctness" {
		return true
	}
	return false
}
