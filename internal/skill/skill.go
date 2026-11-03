// Package skill implements the open agentskills.io standard for parsing,
// validating, and discovering modular AI agent skills.
package skill

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	// NameRegex enforces the agentskills.io specification:
	// 1-64 characters, lowercase alphanumeric and single hyphens. Cannot start or end with a hyphen.
	NameRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
)

// Skill represents an agent skill adhering to the agentskills.io specification.
type Skill struct {
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	License       string            `json:"license,omitempty"`
	Compatibility string            `json:"compatibility,omitempty"`
	AllowedTools  string            `json:"allowed_tools,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	Body          string            `json:"body"`
	Path          string            `json:"path,omitempty"` // relative slash-separated path to SKILL.md
}

// ValidationError describes a constraint violation in a skill definition.
type ValidationError struct {
	Field   string
	Message string
	Line    int
}

func (e ValidationError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("line %d: %s: %s", e.Line, e.Field, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Parse extracts YAML frontmatter and markdown body from SKILL.md content
// using only standard library routines.
func Parse(data []byte) (Skill, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))

	// 1. Locate opening delimiter '---'
	lineNo := 0
	foundOpening := false
	for scanner.Scan() {
		lineNo++
		trimmed := strings.TrimSpace(scanner.Text())
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if trimmed == "---" {
			foundOpening = true
			break
		}
		return Skill{}, fmt.Errorf("line %d: missing opening '---' frontmatter delimiter", lineNo)
	}
	if !foundOpening {
		return Skill{}, errors.New("skill: empty or missing opening '---' frontmatter delimiter")
	}

	// 2. Parse frontmatter lines until closing '---'
	var skill Skill
	skill.Metadata = make(map[string]string)
	foundClosing := false

	var currentKey string
	var blockType string // "|" or ">"
	var blockLines []string
	inMetadata := false

	flushBlock := func() {
		if currentKey == "" {
			return
		}
		var val string
		switch blockType {
		case "|":
			val = strings.Join(blockLines, "\n")
		case ">":
			val = strings.Join(blockLines, " ")
		}
		val = strings.TrimSpace(val)
		assignField(&skill, currentKey, val)
		currentKey = ""
		blockType = ""
		blockLines = nil
	}

	for scanner.Scan() {
		lineNo++
		raw := scanner.Text()
		trimmed := strings.TrimSpace(raw)

		if trimmed == "---" {
			flushBlock()
			foundClosing = true
			break
		}

		// Check for block scalar continuation (indented line)
		if blockType != "" {
			if len(raw) > 0 && (raw[0] == ' ' || raw[0] == '\t') {
				blockLines = append(blockLines, strings.TrimSpace(raw))
				continue
			}
			// Unindented line terminates the block
			flushBlock()
		}

		if inMetadata {
			if len(raw) > 0 && (raw[0] == ' ' || raw[0] == '\t') {
				colonIdx := strings.Index(trimmed, ":")
				if colonIdx != -1 {
					mKey := strings.TrimSpace(trimmed[:colonIdx])
					mVal := cleanValue(trimmed[colonIdx+1:])
					skill.Metadata[mKey] = mVal
				}
				continue
			}
			inMetadata = false
		}

		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		colonIdx := strings.Index(trimmed, ":")
		if colonIdx == -1 {
			continue
		}

		key := strings.ToLower(strings.TrimSpace(trimmed[:colonIdx]))
		valPart := strings.TrimSpace(trimmed[colonIdx+1:])

		if key == "metadata" {
			inMetadata = true
			continue
		}

		if valPart == "|" || valPart == ">" {
			currentKey = key
			blockType = valPart
			blockLines = nil
			continue
		}

		assignField(&skill, key, cleanValue(valPart))
	}

	if !foundClosing {
		return Skill{}, errors.New("skill: missing closing '---' frontmatter delimiter")
	}

	// 3. Collect remaining lines as Body
	var bodyLines []string
	for scanner.Scan() {
		bodyLines = append(bodyLines, scanner.Text())
	}
	skill.Body = strings.TrimSpace(strings.Join(bodyLines, "\n"))

	return skill, scanner.Err()
}

func cleanValue(s string) string {
	s = strings.TrimSpace(s)
	// Strip double quotes or single quotes
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			s = s[1 : len(s)-1]
		}
	}
	return strings.TrimSpace(s)
}

func assignField(s *Skill, key, val string) {
	switch key {
	case "name":
		s.Name = val
	case "description":
		s.Description = val
	case "license":
		s.License = val
	case "compatibility":
		s.Compatibility = val
	case "allowed-tools", "allowed_tools":
		s.AllowedTools = val
	}
}

// Validate checks a parsed Skill against the agentskills.io standard.
// If parentDir is non-empty, it also asserts that s.Name matches filepath.Base(parentDir).
func Validate(s Skill, parentDir string) []ValidationError {
	var errs []ValidationError

	// 1. Name validation
	if s.Name == "" {
		errs = append(errs, ValidationError{Field: "name", Message: "name is required"})
	} else {
		if len(s.Name) > 64 {
			errs = append(errs, ValidationError{Field: "name", Message: fmt.Sprintf("name length %d exceeds maximum 64 characters", len(s.Name))})
		}
		if !NameRegex.MatchString(s.Name) {
			errs = append(errs, ValidationError{Field: "name", Message: fmt.Sprintf("name %q must be 1-64 lowercase alphanumeric chars and single hyphens (^[a-z0-9]+(?:-[a-z0-9]+)*$)", s.Name)})
		}
		if parentDir != "" {
			expected := filepath.Base(parentDir)
			if s.Name != expected {
				errs = append(errs, ValidationError{Field: "name", Message: fmt.Sprintf("skill name %q does not match parent directory %q", s.Name, expected)})
			}
		}
	}

	// 2. Description validation
	descTrim := strings.TrimSpace(s.Description)
	if descTrim == "" {
		errs = append(errs, ValidationError{Field: "description", Message: "description is required and cannot be empty"})
	} else if len(descTrim) > 1024 {
		errs = append(errs, ValidationError{Field: "description", Message: fmt.Sprintf("description length %d exceeds maximum 1024 characters", len(descTrim))})
	}

	// 3. Compatibility validation
	if len(strings.TrimSpace(s.Compatibility)) > 500 {
		errs = append(errs, ValidationError{Field: "compatibility", Message: fmt.Sprintf("compatibility length %d exceeds maximum 500 characters", len(strings.TrimSpace(s.Compatibility)))})
	}

	return errs
}

// DiscoveredSkill pairs a parsed Skill with its location on disk and validation errors.
type DiscoveredSkill struct {
	Skill  Skill
	Dir    string // relative directory path
	File   string // relative path to SKILL.md
	Errors []ValidationError
}

// Discover finds and parses all skills located in skills/ and .sdd/skills/ beneath root.
func Discover(root string) ([]DiscoveredSkill, error) {
	candidates := []string{
		"skills",
		filepath.Join(".sdd", "skills"),
	}

	var results []DiscoveredSkill
	seen := make(map[string]bool)

	for _, cand := range candidates {
		baseDir := filepath.Join(root, cand)
		entries, err := os.ReadDir(baseDir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}

		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			skillDirRel := filepath.Join(cand, e.Name())
			skillFileRel := filepath.Join(skillDirRel, "SKILL.md")
			fullPath := filepath.Join(root, skillFileRel)

			data, err := os.ReadFile(fullPath)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					continue
				}
				return nil, err
			}

			normFile := filepath.ToSlash(skillFileRel)
			if seen[normFile] {
				continue
			}
			seen[normFile] = true

			sk, parseErr := Parse(data)
			var valErrs []ValidationError
			if parseErr != nil {
				valErrs = append(valErrs, ValidationError{
					Field:   "frontmatter",
					Message: parseErr.Error(),
				})
			} else {
				valErrs = Validate(sk, e.Name())
			}

			sk.Path = normFile
			results = append(results, DiscoveredSkill{
				Skill:  sk,
				Dir:    filepath.ToSlash(skillDirRel),
				File:   normFile,
				Errors: valErrs,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].File < results[j].File
	})

	return results, nil
}
