package cli

import (
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vikashkumar21/sdd/internal/skill"
)

func runSkill(args []string, env IO) int {
	if len(args) == 0 {
		fprintln(env.Err, "usage: sdd skill <command> [flags]\n\ncommands:\n  new <name>   scaffold or import a new agentskills.io skill\n  list         list available skills in the project")
		return 2
	}
	switch args[0] {
	case "new":
		return runSkillNew(args[1:], env)
	case "list":
		return runSkillList(args[1:], env)
	default:
		fprintf(env.Err, "sdd skill: unknown command %q\n\nusage: sdd skill <new|list> [flags]\n", args[0])
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
