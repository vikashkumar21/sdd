package cli

import (
	"errors"
	"flag"
	"path/filepath"
	"strings"

	"github.com/vikashkumar21/sdd/internal/permissions"
)

func runPermissions(args []string, env IO) int {
	fs := newFlagSet("permissions", env)
	dir := fs.String("dir", ".", "project root directory")
	agents := fs.String("agents", "all", "comma-separated agent targets (claude, cursor, gemini, all)")
	dryRun := fs.Bool("dry-run", false, "preview generated permission files without writing")
	force := fs.Bool("force", false, "overwrite existing rules (backs up to .sdd/backup/)")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		fprintf(env.Err, "sdd permissions: %v\n", err)
		return 2
	}

	root, err := filepath.Abs(*dir)
	if err != nil {
		fprintf(env.Err, "sdd permissions: %v\n", err)
		return 1
	}

	var rawTargets []string
	for _, part := range strings.Split(*agents, ",") {
		t := strings.TrimSpace(part)
		if t != "" {
			rawTargets = append(rawTargets, t)
		}
	}

	plan, err := permissions.BuildPlan(root, rawTargets, *force)
	if err != nil {
		fprintf(env.Err, "sdd permissions: plan: %v\n", err)
		return 1
	}

	if len(plan.Files) == 0 {
		fprintln(env.Out, "No permission manifests to generate.")
		return 0
	}

	fprintln(env.Out, "Proposed agent permission manifests:")
	for _, fp := range plan.Files {
		fprintf(env.Out, "  [%s] %s (%s)\n", fp.Action, fp.Path, fp.Description)
	}

	if *dryRun {
		fprintf(env.Out, "\nDry run: %d manifest(s) planned. No files written.\n", len(plan.Files))
		return 0
	}

	if err := permissions.Apply(root, plan, *force, env.now()); err != nil {
		fprintf(env.Err, "sdd permissions: apply: %v\n", err)
		return 1
	}

	written := 0
	for _, fp := range plan.Files {
		if fp.Action != permissions.ActionSkip {
			written++
		}
	}
	fprintf(env.Out, "\nGenerated/updated %d agent permission manifest(s).\n", written)
	return 0
}
