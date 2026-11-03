package cli

import (
	"path/filepath"

	"github.com/vikashkumar21/sdd/internal/skill"
)

func runSkill(args []string, env IO) int {
	if len(args) == 0 || args[0] != "new" {
		fprintln(env.Err, "usage: sdd skill new <name> [--description \"...\"] [--from-file <path>] [--from-text <string>] [--internal] [--force] [--dir <project>]")
		return 2
	}
	fs := newFlagSet("skill new", env)
	dir := fs.String("dir", ".", "project root")
	desc := fs.String("description", "", "skill description (1-1024 characters)")
	fromFile := fs.String("from-file", "", "path to file containing instructions/notes to ingest into skill body")
	fromText := fs.String("from-text", "", "raw text instructions to ingest into skill body")
	internal := fs.Bool("internal", false, "scaffold under .sdd/skills/<name>/ instead of skills/<name>/")
	force := fs.Bool("force", false, "overwrite existing skill (backs up previous version to .sdd/backup/)")

	pos, err := parseInterspersed(fs, args[1:])
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
