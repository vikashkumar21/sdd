package cli

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/vikashkumar21/sdd/internal/adopt"
)

type adoptOpts struct {
	dir        string
	dryRun     bool
	yes        bool
	force      bool
	name       string
	maintainer string
	mode       string
}

func runAdopt(args []string, env IO) int {
	fs := newFlagSet("adopt", env)
	var o adoptOpts
	fs.StringVar(&o.dir, "dir", ".", "project root")
	fs.BoolVar(&o.dryRun, "dry-run", false, "preview generated manifest without writing any files")
	fs.BoolVar(&o.yes, "yes", false, "write manifest without interactive confirmation")
	fs.BoolVar(&o.force, "force", false, "overwrite existing .sdd/manifest.json (creates backup)")
	fs.StringVar(&o.name, "name", "", "override project name (default: from MISSION.md or directory name)")
	fs.StringVar(&o.maintainer, "maintainer", "", "override maintainer name (default: from MISSION.md or environment)")
	fs.StringVar(&o.mode, "mode", "", "override mode: greenfield or brownfield (default: brownfield)")

	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return parseExit(err)
	}
	if len(pos) > 0 {
		fprintf(env.Err, "sdd adopt: unexpected argument %q\n", pos[0])
		return 2
	}

	fail := func(format string, a ...any) int {
		fprintf(env.Err, "sdd adopt: "+format+"\n", a...)
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

	opts := adopt.Options{
		Force:      o.force,
		Name:       o.name,
		Maintainer: o.maintainer,
		Mode:       o.mode,
		Now:        env.now(),
	}

	manifest, discovered, err := adopt.BuildManifest(root, opts)
	if err != nil {
		return fail("%v", err)
	}

	fprintf(env.Out, "Adopting kit in %s:\n", root)
	fprintf(env.Out, "  Project:     %s\n", manifest.Answers.Project)
	fprintf(env.Out, "  Maintainer:  %s\n", manifest.Answers.Maintainer)
	fprintf(env.Out, "  Mode:        %s\n", manifest.Answers.Mode)
	fprintf(env.Out, "  Languages:   %s\n", strings.Join(manifest.Answers.Languages, ", "))
	fprintf(env.Out, "  Agents:      %s\n", strings.Join(manifest.Answers.Agents, ", "))
	fprintf(env.Out, "  Discovered:  %d kit files (%s)\n", len(discovered), strings.Join(discovered, ", "))

	if o.dryRun {
		data, err := json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			return fail("encode manifest: %v", err)
		}
		fprintln(env.Out, "\nPlanned .sdd/manifest.json (dry run):")
		fprintln(env.Out, string(data))
		fprintln(env.Out, "\nDry run: nothing was written.")
		return 0
	}

	if env.Interactive && !o.yes {
		pr := newPrompter(env.In, env.Out)
		if !pr.confirm("\nWrite .sdd/manifest.json?") {
			fprintln(env.Out, "Aborted. Nothing was written.")
			return 1
		}
	}

	res, err := adopt.Adopt(root, opts)
	if err != nil {
		return fail("%v", err)
	}

	fprintln(env.Out, "\nAdopted successfully:")
	fprintf(env.Out, "  Manifest:  %s\n", res.ManifestPath)
	if res.BackupPath != "" {
		fprintf(env.Out, "  Backup:    %s\n", res.BackupPath)
	}
	fprintln(env.Out, "No existing documentation or contract files were modified.")
	fprintln(env.Out, "Run `sdd check` to verify kit consistency.")

	return 0
}
