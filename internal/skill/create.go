package skill

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vikashkumar21/sdd/internal/fsutil"
)

// CreateOptions holds parameters for creating a new agent skill.
type CreateOptions struct {
	Name        string
	Description string
	FromFile    string
	FromText    string
	Internal    bool
	Force       bool
	Clock       func() time.Time
}

// CreateResult contains the outcome of skill creation.
type CreateResult struct {
	Path      string // Relative path to created SKILL.md
	BackupDir string // Relative path to backup directory if overwritten
}

// CreateSkill scaffolds a standard agentskills.io skill at skills/<name>/SKILL.md
// (or .sdd/skills/<name>/SKILL.md if opts.Internal is true).
func CreateSkill(root string, opts CreateOptions) (CreateResult, error) {
	var res CreateResult

	if strings.TrimSpace(opts.Name) == "" {
		return res, errors.New("skill name cannot be empty")
	}
	if err := ValidateName(opts.Name); err != nil {
		return res, fmt.Errorf("invalid skill name %q: %w", opts.Name, err)
	}

	if opts.FromFile != "" && opts.FromText != "" {
		return res, errors.New("cannot specify both --from-file and --from-text")
	}

	desc := strings.TrimSpace(opts.Description)
	if desc == "" {
		desc = fmt.Sprintf("Agent skill for %s", opts.Name)
	}
	if err := ValidateDescription(desc); err != nil {
		return res, fmt.Errorf("invalid skill description: %w", err)
	}

	baseDir := "skills"
	if opts.Internal {
		baseDir = filepath.Join(".sdd", "skills")
	}
	relFile := filepath.Join(baseDir, opts.Name, "SKILL.md")

	var rawContent string
	if opts.FromFile != "" {
		filePath := opts.FromFile
		if !filepath.IsAbs(filePath) {
			filePath = filepath.Join(root, filePath)
		}
		data, err := os.ReadFile(filePath)
		if err != nil {
			return res, fmt.Errorf("read from-file %s: %w", opts.FromFile, err)
		}
		rawContent = strings.TrimSpace(string(data))
	} else if opts.FromText != "" {
		rawContent = strings.TrimSpace(opts.FromText)
	}

	rootHandle, err := os.OpenRoot(root)
	if err != nil {
		return res, fmt.Errorf("open root %s: %w", root, err)
	}
	defer func() { _ = rootHandle.Close() }()

	cleanRel := filepath.Clean(relFile)
	fullPath := filepath.Join(root, cleanRel)

	if existingBytes, readErr := os.ReadFile(fullPath); readErr == nil {
		if !opts.Force {
			return res, fmt.Errorf("skill %q already exists at %s (use --force to overwrite)", opts.Name, filepath.ToSlash(cleanRel))
		}
		now := time.Now().UTC()
		if opts.Clock != nil {
			now = opts.Clock().UTC()
		}
		backupTimestamp := now.Format("20060102-150405")
		backupRel := filepath.ToSlash(filepath.Join(".sdd", "backup", backupTimestamp))
		backupDst := filepath.Join(backupRel, filepath.FromSlash(cleanRel))

		if err := fsutil.WriteAtomicRoot(rootHandle, backupDst, existingBytes, 0o644); err != nil {
			return res, fmt.Errorf("backup existing skill: %w", err)
		}
		res.BackupDir = backupRel
	}

	var bodyBuilder strings.Builder
	fmt.Fprintf(&bodyBuilder, "---\nname: %s\ndescription: %s\n---\n\n", opts.Name, desc)
	fmt.Fprintf(&bodyBuilder, "# %s\n\n", opts.Name)

	if rawContent != "" {
		bodyBuilder.WriteString("<!-- sdd:skill-refine:begin -->\n")
		bodyBuilder.WriteString("> [!NOTE]\n")
		bodyBuilder.WriteString("> The notes below were ingested from raw context. When an agent takes over,\n")
		bodyBuilder.WriteString("> refine these into the standard sections below.\n\n")
		bodyBuilder.WriteString("## Ingested Context\n")
		bodyBuilder.WriteString(rawContent)
		bodyBuilder.WriteString("\n<!-- sdd:skill-refine:end -->\n\n")
	}

	bodyBuilder.WriteString("## Overview & Triggers\n\n")
	bodyBuilder.WriteString("## Step-by-Step Instructions\n\n")
	bodyBuilder.WriteString("## Verification\n\n")
	bodyBuilder.WriteString("## Troubleshooting\n")

	renderedContent := bodyBuilder.String()

	// Pre-validate that rendered output parses cleanly
	parsed, parseErr := Parse([]byte(renderedContent))
	if parseErr != nil {
		return res, fmt.Errorf("generated skill frontmatter invalid: %w", parseErr)
	}
	validationErrors := Validate(parsed, filepath.Dir(fullPath))
	if len(validationErrors) > 0 {
		return res, fmt.Errorf("generated skill validation failed: %s", validationErrors[0].Message)
	}

	if err := fsutil.WriteAtomicRoot(rootHandle, cleanRel, []byte(renderedContent), 0o644); err != nil {
		return res, fmt.Errorf("write skill file %s: %w", cleanRel, err)
	}

	res.Path = filepath.ToSlash(cleanRel)
	return res, nil
}
