package cli

import (
	"encoding/json"
	"path/filepath"

	"github.com/vikashkumar21/sdd/internal/docflow"
)

func runDoc(args []string, env IO) int {
	if len(args) == 0 {
		fprintln(env.Err, "usage: sdd doc <command> [flags]\n\ncommands:\n  init    scaffold docs/architecture/ and docs/business_flows/ with Mermaid diagrams\n  flow    scaffold a new domain flow document in docs/business_flows/<name>.md")
		return 2
	}
	if args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		fprintln(env.Out, "usage: sdd doc <command> [flags]\n\ncommands:\n  init    scaffold docs/architecture/ and docs/business_flows/ with Mermaid diagrams\n  flow    scaffold a new domain flow document in docs/business_flows/<name>.md")
		return 0
	}
	switch args[0] {
	case "init":
		return runDocInit(args[1:], env)
	case "flow":
		return runDocFlow(args[1:], env)
	default:
		fprintf(env.Err, "sdd doc: unknown command %q\n\nusage: sdd doc <init|flow> [flags]\n", args[0])
		return 2
	}
}

func runDocInit(args []string, env IO) int {
	fs := newFlagSet("doc init", env)
	dir := fs.String("dir", ".", "project root")
	force := fs.Bool("force", false, "overwrite existing documentation files (with automatic backups)")
	dryRun := fs.Bool("dry-run", false, "preview planned files without writing")
	jsonOut := fs.Bool("json", false, "output results as JSON")

	if err := fs.Parse(args); err != nil {
		return parseExit(err)
	}

	root, err := filepath.Abs(*dir)
	if err != nil {
		fprintf(env.Err, "sdd doc init: resolve dir: %v\n", err)
		return 1
	}

	opts := docflow.InitOptions{
		Dir:    root,
		Force:  *force,
		DryRun: *dryRun,
		JSON:   *jsonOut,
		Now:    env.now(),
	}

	plan, err := docflow.BuildInitPlan(opts)
	if err != nil {
		fprintf(env.Err, "sdd doc init: %v\n", err)
		return 1
	}

	if *dryRun {
		if *jsonOut {
			data, _ := json.MarshalIndent(plan, "", "  ")
			fprintln(env.Out, string(data))
			return 0
		}
		fprintln(env.Out, "Planned documentation scaffolding (dry-run):")
		for _, p := range plan {
			fprintf(env.Out, "  [%s] %s (%s)\n", p.Action, p.RelPath, p.Reason)
		}
		return 0
	}

	res, err := docflow.Apply(root, plan, opts.Now)
	if err != nil {
		fprintf(env.Err, "sdd doc init: %v\n", err)
		return 1
	}

	if *jsonOut {
		data, _ := json.MarshalIndent(res, "", "  ")
		fprintln(env.Out, string(data))
		return 0
	}

	fprintln(env.Out, "Scaffolded domain flow & architecture documentation:")
	for _, p := range res.Created {
		fprintf(env.Out, "  [created] %s\n", p)
	}
	for _, p := range res.Skipped {
		fprintf(env.Out, "  [skipped] %s (already exists; use --force to overwrite)\n", p)
	}
	for _, p := range res.Backups {
		fprintf(env.Out, "  [backup]  %s\n", p)
	}

	return 0
}

func runDocFlow(args []string, env IO) int {
	fs := newFlagSet("doc flow", env)
	title := fs.String("title", "", "human readable title of the domain flow")
	dir := fs.String("dir", ".", "project root")
	force := fs.Bool("force", false, "overwrite existing flow document (with automatic backup)")
	dryRun := fs.Bool("dry-run", false, "preview planned flow without writing")
	jsonOut := fs.Bool("json", false, "output results as JSON")

	rest, err := parseInterspersed(fs, args)
	if err != nil {
		return parseExit(err)
	}

	if len(rest) == 0 {
		fprintln(env.Err, "sdd doc flow: missing required flow name\n\nusage: sdd doc flow <name> [--title <title>] [flags]")
		return 2
	}
	name := rest[0]

	root, err := filepath.Abs(*dir)
	if err != nil {
		fprintf(env.Err, "sdd doc flow: resolve dir: %v\n", err)
		return 1
	}

	opts := docflow.FlowOptions{
		Dir:    root,
		Name:   name,
		Title:  *title,
		Force:  *force,
		DryRun: *dryRun,
		JSON:   *jsonOut,
		Now:    env.now(),
	}

	plan, err := docflow.BuildFlowPlan(opts)
	if err != nil {
		fprintf(env.Err, "sdd doc flow: %v\n", err)
		return 1
	}

	if *dryRun {
		if *jsonOut {
			data, _ := json.MarshalIndent(plan, "", "  ")
			fprintln(env.Out, string(data))
			return 0
		}
		fprintln(env.Out, "Planned flow documentation (dry-run):")
		for _, p := range plan {
			fprintf(env.Out, "  [%s] %s (%s)\n", p.Action, p.RelPath, p.Reason)
		}
		return 0
	}

	res, err := docflow.Apply(root, plan, opts.Now)
	if err != nil {
		fprintf(env.Err, "sdd doc flow: %v\n", err)
		return 1
	}

	if *jsonOut {
		data, _ := json.MarshalIndent(res, "", "  ")
		fprintln(env.Out, string(data))
		return 0
	}

	for _, p := range res.Created {
		fprintf(env.Out, "  [created] %s\n", p)
	}
	for _, p := range res.Skipped {
		fprintf(env.Out, "  [skipped] %s (already exists; use --force to overwrite)\n", p)
	}
	for _, p := range res.Backups {
		fprintf(env.Out, "  [backup]  %s\n", p)
	}

	return 0
}
