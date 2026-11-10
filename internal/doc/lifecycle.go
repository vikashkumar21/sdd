package doc

import (
	"bufio"
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	dateRegex   = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	validStatus = map[string]bool{
		"active":     true,
		"draft":      true,
		"completed":  true,
		"superseded": true,
		"deprecated": true,
	}
)

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
	scanner := bufio.NewScanner(bytes.NewReader(data))
	lineNo := 0
	foundOpening := false

	for scanner.Scan() {
		lineNo++
		trimmed := strings.TrimSpace(scanner.Text())
		if trimmed == "" {
			continue
		}
		if trimmed == "---" {
			foundOpening = true
			break
		}
		// First non-empty line is not '---', so no frontmatter
		return LifecycleMetadata{}, false, false, data, nil
	}

	if !foundOpening {
		return LifecycleMetadata{}, false, false, data, nil
	}

	hasFrontmatter = true
	var frontmatterLines []string
	foundClosing := false

	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			foundClosing = true
			break
		}
		frontmatterLines = append(frontmatterLines, line)
	}

	if !foundClosing {
		return LifecycleMetadata{}, true, false, data, fmt.Errorf("line %d: unclosed '---' frontmatter delimiter", lineNo)
	}

	// Capture remaining body
	var bodyBuf bytes.Buffer
	for scanner.Scan() {
		bodyBuf.WriteString(scanner.Text())
		bodyBuf.WriteByte('\n')
	}
	body = bodyBuf.Bytes()

	// Parse key-value pairs
	rawMap := make(map[string]string)
	for _, rawLine := range frontmatterLines {
		trimmed := strings.TrimSpace(rawLine)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) == 2 {
			k := strings.ToLower(strings.TrimSpace(parts[0]))
			v := strings.TrimSpace(parts[1])
			// Strip inline comments if preceded by whitespace
			if idx := strings.Index(v, "#"); idx != -1 {
				if idx > 0 && (v[idx-1] == ' ' || v[idx-1] == '\t') {
					v = strings.TrimSpace(v[:idx])
				}
			}
			v = strings.Trim(v, `"'`)
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

	return meta, true, isLifecycle, body, nil
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
		normStatus := strings.ToLower(meta.Status)
		if !validStatus[normStatus] {
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
