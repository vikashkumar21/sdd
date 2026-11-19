package cli

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/vikashkumar21/sdd/internal/ci"
)

func runCI(args []string, env IO) int {
	if len(args) == 0 {
		fprintln(env.Err, "usage: sdd ci <command> [flags]\n\ncommands:\n  init    generate CI workflow files for GitHub, GitLab, Bitbucket, Azure, GCP, AWS")
		return 2
	}
	if args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		fprintln(env.Out, "usage: sdd ci <command> [flags]\n\ncommands:\n  init    generate CI workflow files for GitHub, GitLab, Bitbucket, Azure, GCP, AWS")
		return 0
	}
	switch args[0] {
	case "init":
		return runCIInit(args[1:], env)
	default:
		fprintf(env.Err, "sdd ci: unknown command %q\n\nusage: sdd ci <init> [flags]\n", args[0])
		return 2
	}
}

func runCIInit(args []string, env IO) int {
	fs := newFlagSet("ci init", env)
	platformStr := fs.String("platform", "github", "CI platform (github, gitlab, bitbucket, azure, gcp, aws, all)")
	dir := fs.String("dir", ".", "project root")
	dryRun := fs.Bool("dry-run", false, "preview generated workflow files without writing")
	force := fs.Bool("force", false, "overwrite existing CI workflow files")
	jsonOut := fs.Bool("json", false, "output results as JSON")

	if err := fs.Parse(args); err != nil {
		return parseExit(err)
	}

	platform, err := ci.ParsePlatform(*platformStr)
	if err != nil {
		fprintln(env.Err, err.Error())
		return 2
	}

	root, err := filepath.Abs(*dir)
	if err != nil {
		fprintf(env.Err, "sdd ci init: resolve dir: %v\n", err)
		return 1
	}

	cfg := ci.Config{
		Platform: platform,
		DryRun:   *dryRun,
		Force:    *force,
	}

	files, err := ci.GenerateWorkflows(root, cfg)
	if err != nil {
		fprintf(env.Err, "sdd ci init: generate workflows: %v\n", err)
		return 1
	}

	if !*dryRun {
		if err := ci.ApplyWorkflows(root, files, *force); err != nil {
			fprintf(env.Err, "sdd ci init: apply workflows: %v\n", err)
			return 1
		}
	}

	if *jsonOut {
		data, err := json.MarshalIndent(files, "", "  ")
		if err != nil {
			fprintf(env.Err, "sdd ci init: marshal json: %v\n", err)
			return 1
		}
		fprintln(env.Out, string(data))
		return 0
	}

	if *dryRun {
		fprintln(env.Out, "Dry-run plan for CI workflow initialization:")
		for _, f := range files {
			status := "create"
			if f.Skipped {
				status = fmt.Sprintf("skip (%s)", f.Reason)
			} else if f.Modified {
				status = fmt.Sprintf("replace (%s)", f.Reason)
			}
			fprintf(env.Out, "  [%s] %s (%s)\n", status, f.Path, f.Platform)
		}
		return 0
	}

	createdCount := 0
	skippedCount := 0
	modifiedCount := 0
	for _, f := range files {
		if f.Created {
			createdCount++
			fprintf(env.Out, "Created %s (%s)\n", f.Path, f.Platform)
		} else if f.Modified {
			modifiedCount++
			fprintf(env.Out, "Replaced %s (%s)\n", f.Path, f.Platform)
		} else if f.Skipped {
			skippedCount++
			fprintf(env.Out, "Skipped %s (%s: %s)\n", f.Path, f.Platform, f.Reason)
		}
	}

	if skippedCount > 0 && !*force {
		fprintf(env.Out, "\nNote: %d file(s) skipped because they already exist. Use --force to overwrite.\n", skippedCount)
	}
	return 0
}
