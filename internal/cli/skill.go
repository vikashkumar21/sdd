package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vikashkumar21/sdd/internal/fsutil"
	"github.com/vikashkumar21/sdd/internal/role"
	"github.com/vikashkumar21/sdd/internal/skill"
)

func runSkill(args []string, env IO) int {
	if len(args) == 0 {
		fprintln(env.Err, "usage: sdd skill <command> [flags]\n\ncommands:\n  new <name>   scaffold or import a new agentskills.io skill\n  list         list available skills in the project\n  export       export SDD protocol as a native agent skill")
		return 2
	}
	switch args[0] {
	case "new":
		return runSkillNew(args[1:], env)
	case "list":
		return runSkillList(args[1:], env)
	case "export":
		return runSkillExport(args[1:], env)
	default:
		fprintf(env.Err, "sdd skill: unknown command %q\n\nusage: sdd skill <new|list|export> [flags]\n", args[0])
		return 2
	}
}

func runSkillNew(args []string, env IO) int {
	fs := newFlagSet("skill new", env)
	dir := fs.String("dir", ".", "project root")
	desc := fs.String("description", "", "skill description (1-1024 characters)")
	fromFile := fs.String("from-file", "", "path to file containing instructions/notes to ingest into skill body")
	fromText := fs.String("from-text", "", "raw text instructions to ingest into skill body")
	internal := fs.Bool("internal", false, "scaffold under .sdd/skills/<name>/ instead of skills/<name>/")
	force := fs.Bool("force", false, "overwrite existing skill (backs up previous version to .sdd/backup/)")

	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return parseExit(err)
	}
	if len(pos) != 1 {
		fprintln(env.Err, "usage: sdd skill new <name> [--description \"...\"] [--from-file <path>] [--from-text <string>] [--internal] [--force] [--dir <project>]")
		return 2
	}

	name := pos[0]
	root, err := filepath.Abs(*dir)
	if err != nil {
		fprintf(env.Err, "sdd skill new: %v\n", err)
		return 1
	}

	opts := skill.CreateOptions{
		Name:        name,
		Description: *desc,
		FromFile:    *fromFile,
		FromText:    *fromText,
		Internal:    *internal,
		Force:       *force,
		Clock:       env.Now,
	}

	res, err := skill.CreateSkill(root, opts)
	if err != nil {
		fprintf(env.Err, "sdd skill new: %v\n", err)
		return 1
	}

	fprintf(env.Out, "Created skill %s at %s\n", name, res.Path)
	if res.BackupDir != "" {
		fprintf(env.Out, "Previous version backed up to %s\n", res.BackupDir)
	}
	return 0
}

type skillJSONItem struct {
	Name        string   `json:"name"`
	Scope       string   `json:"scope"`
	Status      string   `json:"status"`
	Path        string   `json:"path"`
	Description string   `json:"description,omitempty"`
	Errors      []string `json:"errors,omitempty"`
}

func runSkillList(args []string, env IO) int {
	fs := newFlagSet("skill list", env)
	dir := fs.String("dir", ".", "project root")
	jsonOut := fs.Bool("json", false, "output skills as structured JSON")

	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return parseExit(err)
	}
	if len(pos) > 0 {
		fprintf(env.Err, "sdd skill list: unexpected argument %q\n", pos[0])
		return 2
	}

	root, err := filepath.Abs(*dir)
	if err != nil {
		fprintf(env.Err, "sdd skill list: %v\n", err)
		return 1
	}

	discovered, err := skill.Discover(root)
	if err != nil {
		fprintf(env.Err, "sdd skill list: %v\n", err)
		return 1
	}

	var items []skillJSONItem
	for _, sk := range discovered {
		name := sk.Skill.Name
		if name == "" {
			name = filepath.Base(sk.Dir)
		}
		scope := "project"
		if strings.HasPrefix(filepath.ToSlash(sk.File), ".sdd/skills/") {
			scope = "internal"
		}
		status := "valid"
		var errMsgs []string
		if len(sk.Errors) > 0 {
			status = "invalid"
			for _, ve := range sk.Errors {
				errMsgs = append(errMsgs, ve.Error())
			}
		}
		items = append(items, skillJSONItem{
			Name:        name,
			Scope:       scope,
			Status:      status,
			Path:        sk.File,
			Description: sk.Skill.Description,
			Errors:      errMsgs,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})

	if *jsonOut {
		data, jerr := json.MarshalIndent(items, "", "  ")
		if jerr != nil {
			fprintf(env.Err, "sdd skill list: marshal json: %v\n", jerr)
			return 1
		}
		fprintln(env.Out, string(data))
		return 0
	}

	if len(items) == 0 {
		fprintln(env.Out, "No skills found.")
		return 0
	}

	fprintf(env.Out, "%-20s  %-10s  %-10s  %s\n", "NAME", "SCOPE", "STATUS", "DESCRIPTION")
	for _, item := range items {
		fprintf(env.Out, "%-20s  %-10s  %-10s  %s\n", item.Name, item.Scope, item.Status, item.Description)
	}
	return 0
}

func runSkillExport(args []string, env IO) int {
	fs := newFlagSet("skill export", env)
	dir := fs.String("dir", ".", "project root")
	target := fs.String("target", "", "custom destination directory for exported skill")
	skillName := fs.String("skill", "all", "canonical skill to export: sdd-protocol, code-review, task-coordinator, or all")
	rolesFlag := fs.Bool("roles", false, "export all canonical role skills (appsec-specialist, architect, codebase-analyst, coordinator, devops, discovery-challenger, implementer, pattern-auditor, plan-reviewer, qa, spec-auditor)")
	roleFlag := fs.String("role", "", "export a specific canonical role skill by name")
	global := fs.Bool("global", false, "export to user's global agent skill directory")
	force := fs.Bool("force", false, "overwrite existing skill (backs up previous version to .sdd/backup/)")

	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return parseExit(err)
	}
	if len(pos) > 0 {
		fprintf(env.Err, "sdd skill export: unexpected argument %q\n", pos[0])
		return 2
	}

	root, err := filepath.Abs(*dir)
	if err != nil {
		fprintf(env.Err, "sdd skill export: %v\n", err)
		return 1
	}

	opts := skill.ExportOptions{
		ProjectRoot: root,
		TargetDir:   *target,
		Global:      *global,
		Force:       *force,
		Home:        env.Home,
		Clock:       env.Now,
	}

	var results []skill.ExportResult
	if *roleFlag != "" {
		res, err := exportRoleSkill(*roleFlag, opts)
		if err != nil {
			fprintf(env.Err, "sdd skill export: %v\n", err)
			return 1
		}
		results = []skill.ExportResult{res}
	} else if *rolesFlag {
		for _, name := range role.CanonicalRoleNames() {
			res, err := exportRoleSkill(name, opts)
			if err != nil {
				fprintf(env.Err, "sdd skill export: %v\n", err)
				return 1
			}
			results = append(results, res)
		}
	} else {
		var err error
		results, err = skill.ExportSkills(*skillName, opts)
		if err != nil {
			fprintf(env.Err, "sdd skill export: %v\n", err)
			return 1
		}
	}

	for _, res := range results {
		fprintf(env.Out, "Exported skill %s to %s\n", res.Name, res.Path)
		if res.BackupDir != "" {
			fprintf(env.Out, "Previous version backed up to %s\n", res.BackupDir)
		}
	}
	return 0
}

func exportRoleSkill(roleName string, opts skill.ExportOptions) (skill.ExportResult, error) {
	canonical := false
	for _, c := range role.CanonicalRoleNames() {
		if c == roleName {
			canonical = true
			break
		}
	}
	if !canonical {
		return skill.ExportResult{}, fmt.Errorf("unknown canonical role %q (available roles: %s)", roleName, strings.Join(role.CanonicalRoleNames(), ", "))
	}

	content, err := role.ResolveSkill(opts.ProjectRoot, roleName, nil)
	if err != nil {
		return skill.ExportResult{}, fmt.Errorf("resolve role skill %s: %w", roleName, err)
	}

	var targetBase string
	if opts.Global {
		home := ""
		if opts.Home != nil {
			home = opts.Home()
		} else {
			home, _ = os.UserHomeDir()
		}
		if strings.TrimSpace(home) == "" {
			return skill.ExportResult{}, errors.New("cannot export globally: home directory not determined")
		}
		geminiDir := filepath.Join(home, ".gemini", "antigravity")
		claudeDir := filepath.Join(home, ".claude")
		if stat, err := os.Stat(geminiDir); err == nil && stat.IsDir() {
			targetBase = filepath.Join(geminiDir, "skills")
		} else if stat, err := os.Stat(claudeDir); err == nil && stat.IsDir() {
			targetBase = filepath.Join(claudeDir, "skills")
		} else {
			targetBase = filepath.Join(home, ".sdd", "skills")
		}
	} else if opts.TargetDir != "" {
		targetBase = opts.TargetDir
		if !filepath.IsAbs(targetBase) && opts.ProjectRoot != "" {
			targetBase = filepath.Join(opts.ProjectRoot, targetBase)
		}
	} else {
		if opts.ProjectRoot == "" {
			return skill.ExportResult{}, errors.New("project root required when target directory is not specified")
		}
		targetBase = filepath.Join(opts.ProjectRoot, "skills")
	}

	destDir := filepath.Join(targetBase, roleName)
	destFile := filepath.Join(destDir, "SKILL.md")

	var backupDir string
	if _, err := os.Stat(destFile); err == nil {
		if !opts.Force {
			return skill.ExportResult{}, fmt.Errorf("skill %q already exists at %s (use --force to overwrite)", roleName, filepath.ToSlash(destFile))
		}
		now := time.Now().UTC()
		if opts.Clock != nil {
			now = opts.Clock().UTC()
		}
		relBackup := filepath.Join(".sdd", "backup", fmt.Sprintf("backup-%s", now.Format("20060102-150405")))
		backupDir = filepath.Join(opts.ProjectRoot, relBackup)
		if err := os.MkdirAll(backupDir, 0755); err != nil {
			return skill.ExportResult{}, fmt.Errorf("create backup directory: %w", err)
		}
		data, _ := os.ReadFile(destFile)
		_ = os.WriteFile(filepath.Join(backupDir, fmt.Sprintf("%s-SKILL.md", roleName)), data, 0644)
	}

	parsed, err := skill.Parse([]byte(content))
	if err != nil {
		return skill.ExportResult{}, fmt.Errorf("role skill frontmatter invalid: %w", err)
	}
	if vErrs := skill.Validate(parsed, destDir); len(vErrs) > 0 {
		return skill.ExportResult{}, fmt.Errorf("role skill validation failed: %s", vErrs[0].Message)
	}

	if err := fsutil.WriteAtomic(destFile, []byte(content), 0644); err != nil {
		return skill.ExportResult{}, fmt.Errorf("write skill file: %w", err)
	}

	return skill.ExportResult{
		Name:      roleName,
		Path:      filepath.ToSlash(destFile),
		BackupDir: backupDir,
	}, nil
}
