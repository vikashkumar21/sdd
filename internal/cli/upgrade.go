package cli

import (
	"path/filepath"
	"strings"

	"github.com/vikashkumar21/sdd/internal/upgrade"
)

type upgradeOpts struct {
	dir          string
	templatesDir string
	dryRun       bool
	yes          bool
	force        bool
}

func runUpgrade(args []string, env IO) int {
	fs := newFlagSet("upgrade", env)
	var o upgradeOpts
	fs.StringVar(&o.dir, "dir", ".", "project root")
	fs.StringVar(&o.templatesDir, "templates", "", "directory whose files override the embedded templates")
	fs.BoolVar(&o.dryRun, "dry-run", false, "show planned updates and diffs without writing any files")
	fs.BoolVar(&o.yes, "yes", false, "apply updates and write proposals without interactive confirmation")
	fs.BoolVar(&o.force, "force", false, "overwrite edited files instead of creating side-by-side proposals (creates backups)")

	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return parseExit(err)
	}
	if len(pos) > 0 {
		fprintf(env.Err, "sdd upgrade: unexpected argument %q\n", pos[0])
		return 2
	}

	fail := func(format string, a ...any) int {
		fprintf(env.Err, "sdd upgrade: "+format+"\n", a...)
		return 1
	}

	if isDriveRoot(o.dir) {
		return fail("refusing to run in the filesystem root")
	}

	root, err := filepath.Abs(o.dir)
	if err != nil {
		return fail("%v", err)
	}
	if err := guardRoot(root, env.home()); err != nil {
		return fail("%v", err)
	}

	plan, err := upgrade.BuildPlan(root, upgrade.Options{
		TemplatesDir: o.templatesDir,
		Force:        o.force,
		Now:          env.now(),
	})
	if err != nil {
		return fail("%v", err)
	}

	if plan.UpToDate {
		fprintf(env.Out, "Kit is already up-to-date (templates %s).\n", plan.NewTemplatesVer)
		return 0
	}

	fprintf(env.Out, "Upgrade plan for %s (templates %s -> %s):\n", root, plan.OldTemplatesVer, plan.NewTemplatesVer)
	for _, mig := range plan.Migrations {
		fprintf(env.Out, "  %-10s %s -> %s\n", "migrate", mig.From, mig.To)
	}
	for _, it := range plan.Items {
		switch it.Action {
		case upgrade.ActionUnchanged:
			fprintf(env.Out, "  %-10s %s\n", "unchanged", it.Path)
		case upgrade.ActionUpdate:
			fprintf(env.Out, "  %-10s %s  (untouched; will update in place)\n", "update", it.Path)
		case upgrade.ActionProposal:
			fprintf(env.Out, "  %-10s %s  (edited; will write %s.sdd-new)\n", "proposal", it.Path, it.Path)
		case upgrade.ActionForce:
			fprintf(env.Out, "  %-10s %s  (edited; will replace under --force)\n", "replace", it.Path)
		case upgrade.ActionCreate:
			fprintf(env.Out, "  %-10s %s  (new file)\n", "create", it.Path)
		}
	}

	// Print unified diff previews for proposals or updates
	for _, it := range plan.Items {
		if it.Diff != "" && (it.Action == upgrade.ActionProposal || it.Action == upgrade.ActionForce || o.dryRun) {
			fprintln(env.Out, "")
			fprintln(env.Out, it.Diff)
		}
	}

	if o.dryRun {
		fprintln(env.Out, "\nDry run: nothing was written.")
		return 0
	}

	if env.Interactive && !o.yes {
		pr := newPrompter(env.In, env.Out)
		if !pr.confirm("\nApply these upgrades?") {
			fprintln(env.Out, "Aborted. Nothing was written.")
			return 1
		}
	}

	rep, err := upgrade.Apply(plan, env.now())
	if err != nil {
		return fail("%v", err)
	}

	fprintln(env.Out, "\nUpgrade applied successfully:")
	if len(rep.Migrated) > 0 {
		fprintf(env.Out, "  Migrated:  %s\n", strings.Join(rep.Migrated, ", "))
	}
	if len(rep.Updated) > 0 {
		fprintf(env.Out, "  Updated:   %s\n", strings.Join(rep.Updated, ", "))
	}
	if len(rep.Created) > 0 {
		fprintf(env.Out, "  Created:   %s\n", strings.Join(rep.Created, ", "))
	}
	if len(rep.Proposals) > 0 {
		fprintf(env.Out, "  Proposals: %s (review and merge into original files)\n", strings.Join(rep.Proposals, ", "))
	}
	if rep.BackupDir != "" {
		fprintf(env.Out, "  Backup:    %s\n", rep.BackupDir)
	}

	return 0
}
