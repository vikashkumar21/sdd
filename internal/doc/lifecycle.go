package doc

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	dateRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

func isValidStatus(status string) bool {
	switch strings.ToLower(status) {
	case "active", "draft", "completed", "superseded", "deprecated":
		return true
	default:
		return false
	}
}

// LifecycleMetadata represents the standard YAML frontmatter lifecycle schema.
type LifecycleMetadata struct {
	Status       string `json:"status"`
	TaskID       string `json:"task_id,omitempty"`
	LastReviewed string `json:"last_reviewed,omitempty"`
	SupersededBy string `json:"superseded_by,omitempty"`
}

// LifecycleValidationError describes a validation error in document lifecycle metadata.
type LifecycleValidationError struct {
	Rule    string
	Field   string
	Message string
}

func (e LifecycleValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ParseFrontmatter extracts YAML frontmatter from Markdown data.
// It returns the parsed metadata, whether frontmatter was present,
// whether it represents lifecycle metadata, the remaining markdown body, and any error.
func ParseFrontmatter(data []byte) (meta LifecycleMetadata, hasFrontmatter bool, isLifecycle bool, body []byte, err error) {
	lines := bytes.Split(data, []byte("\n"))
	startIdx := -1
	endIdx := -1
	byteOffset := 0

	for i, rawLine := range lines {
		lineStr := string(bytes.TrimRight(rawLine, "\r"))
		trimmed := strings.TrimSpace(lineStr)

		if startIdx == -1 {
			if trimmed == "" {
				byteOffset += len(rawLine) + 1
				continue
			}
			if trimmed == "---" {
				startIdx = i
				byteOffset += len(rawLine) + 1
				continue
			}
			// First non-empty line is not '---', so no frontmatter
			return LifecycleMetadata{}, false, false, data, nil
		}

		// Inside frontmatter: looking for closing ---
		if trimmed == "---" {
			endIdx = i
			byteOffset += len(rawLine) + 1
			break
		}
		byteOffset += len(rawLine) + 1
	}

	if startIdx == -1 {
		return LifecycleMetadata{}, false, false, data, nil
	}

	if endIdx == -1 {
		return LifecycleMetadata{}, true, false, data, fmt.Errorf("line %d: unclosed '---' frontmatter delimiter", len(lines))
	}

	hasFrontmatter = true
	if byteOffset <= len(data) {
		body = data[byteOffset:]
	} else {
		body = nil
	}

	// Parse key-value pairs between startIdx and endIdx
	rawMap := make(map[string]string)
	for i := startIdx + 1; i < endIdx; i++ {
		lineStr := string(bytes.TrimRight(lines[i], "\r"))
		trimmed := strings.TrimSpace(lineStr)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) == 2 {
			k := strings.ToLower(strings.TrimSpace(parts[0]))
			v := parseFrontmatterValue(parts[1])
			rawMap[k] = v
		}
	}

	meta.Status = rawMap["status"]
	meta.TaskID = rawMap["task_id"]
	meta.LastReviewed = rawMap["last_reviewed"]
	meta.SupersededBy = rawMap["superseded_by"]

	// Check if this frontmatter has any lifecycle fields
	if meta.Status != "" || meta.TaskID != "" || meta.LastReviewed != "" || meta.SupersededBy != "" {
		isLifecycle = true
	}

	return meta, hasFrontmatter, isLifecycle, body, nil
}

func parseFrontmatterValue(v string) string {
	v = strings.TrimSpace(v)
	if len(v) == 0 {
		return ""
	}
	quote := v[0]
	if quote == '"' || quote == '\'' {
		endQuote := strings.IndexByte(v[1:], quote)
		if endQuote != -1 {
			return v[1 : 1+endQuote]
		}
		return strings.Trim(v, `"'`)
	}
	if idx := strings.Index(v, "#"); idx != -1 {
		if idx > 0 && (v[idx-1] == ' ' || v[idx-1] == '\t') {
			v = strings.TrimSpace(v[:idx])
		}
	}
	return strings.Trim(v, `"'`)
}

// ValidateLifecycle checks a LifecycleMetadata struct for consistency against SDD standards.
// fileExists checks if a relative file path exists in the repository.
func ValidateLifecycle(meta LifecycleMetadata, docPath string, fileExists func(string) bool) []LifecycleValidationError {
	var errs []LifecycleValidationError

	// 1. Status validation
	if meta.Status == "" {
		errs = append(errs, LifecycleValidationError{
			Rule:    "lifecycle-status",
			Field:   "status",
			Message: "missing required 'status' field (expected active, draft, completed, superseded, or deprecated)",
		})
	} else {
		if !isValidStatus(meta.Status) {
			errs = append(errs, LifecycleValidationError{
				Rule:    "lifecycle-status",
				Field:   "status",
				Message: fmt.Sprintf("invalid status %q (expected active, draft, completed, superseded, or deprecated)", meta.Status),
			})
		}
	}

	// 2. Date validation (last_reviewed)
	if meta.LastReviewed != "" {
		if !dateRegex.MatchString(meta.LastReviewed) {
			errs = append(errs, LifecycleValidationError{
				Rule:    "lifecycle-date",
				Field:   "last_reviewed",
				Message: fmt.Sprintf("invalid date format %q in 'last_reviewed' (expected YYYY-MM-DD)", meta.LastReviewed),
			})
		}
	}

	// 3. Superseded validation
	if strings.ToLower(meta.Status) == "superseded" {
		if strings.TrimSpace(meta.SupersededBy) == "" {
			errs = append(errs, LifecycleValidationError{
				Rule:    "lifecycle-superseded",
				Field:   "superseded_by",
				Message: "status is 'superseded' but 'superseded_by' path is empty",
			})
		} else if fileExists != nil {
			target := strings.TrimSpace(meta.SupersededBy)
			// Check relative to root first, then relative to doc directory
			exists := fileExists(target)
			if !exists && docPath != "" {
				docDir := filepath.Dir(docPath)
				combined := filepath.Clean(filepath.Join(docDir, target))
				exists = fileExists(combined)
			}
			if !exists {
				errs = append(errs, LifecycleValidationError{
					Rule:    "lifecycle-superseded",
					Field:   "superseded_by",
					Message: fmt.Sprintf("superseded target file %q does not exist", target),
				})
			}
		}
	}

	return errs
}
