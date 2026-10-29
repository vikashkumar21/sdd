package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vikashkumar21/sdd/internal/detect"
	"github.com/vikashkumar21/sdd/internal/scaffold"
	"github.com/vikashkumar21/sdd/internal/templates"
)

type initOpts struct {
	dir                                          string
	yes, dryRun, force                           bool
	name, purpose, mode, lang, maintainer        string
	build, test, lint, format, agents, templates string
	rules                                        stringList
}

func runInit(args []string, env IO) int {
	fs := newFlagSet("init", env)
	var o initOpts
	fs.StringVar(&o.dir, "dir", ".", "project root")
	fs.BoolVar(&o.yes, "yes", false, "accept detected values and defaults without prompting")
	fs.BoolVar(&o.dryRun, "dry-run", false, "show the plan and write nothing")
	fs.BoolVar(&o.force, "force", false, "replace existing generated files (originals are backed up; README.md is never touched)")
	fs.StringVar(&o.name, "name", "", "project name (default: directory name)")
	fs.StringVar(&o.purpose, "purpose", "", "one-line purpose")
	fs.StringVar(&o.mode, "mode", "", "greenfield or brownfield (default: detected)")
	fs.StringVar(&o.lang, "lang", "", "comma-separated languages: "+strings.Join(templates.LanguageIDs(), ", "))
	fs.StringVar(&o.maintainer, "maintainer", "", "the human who approves and commits")
	fs.StringVar(&o.build, "build", "", "build command")
	fs.StringVar(&o.test, "test", "", "test command")
	fs.StringVar(&o.lint, "lint", "", "lint command")
	fs.StringVar(&o.format, "format", "", "format command")
	fs.StringVar(&o.agents, "agents", "", "comma-separated agent files: "+strings.Join(scaffold.AgentIDs(), ", ")+" (default: all)")
	fs.StringVar(&o.templates, "templates", "", "directory whose files override the embedded templates")
	fs.Var(&o.rules, "rule", "extra hard rule for the agent contract (repeatable)")
	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return parseExit(err)
	}
	if len(pos) > 0 {
		fprintf(env.Err, "sdd init: unexpected argument %q\n", pos[0])
		return 2
	}
	given := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { given[f.Name] = true })
	fail := func(format string, a ...any) int {
		fprintf(env.Err, "sdd init: "+format+"\n", a...)
		return 1
	}

	root, err := filepath.Abs(o.dir)
	if err != nil {
		return fail("%v", err)
	}
	if err := guardRoot(root, env.home()); err != nil {
		return fail("%v", err)
	}
	if !env.Interactive && !o.yes {
		return fail("no terminal to ask questions on; re-run with --yes (and flags such as --name and --lang) to run non-interactively")
	}
	if m, ok, err := scaffold.ReadManifest(root); err != nil {
		return fail("%v", err)
	} else if ok && !o.force {
		return fail("this project is already initialised (sdd %s, %s); run `sdd check`, or use --force to regenerate with backups", m.Tool.Version, scaffold.ManifestPath)
	}

	if !o.force && handMadeKit(root) {
		return fail("this project already has a spec-driven kit (MISSION.md, TECH_STACK.md, ROADMAP.md, and SKILLS.md all exist) but no %s, so it was made by hand or by an older tool; running init would append a second contract to its agent files. Run `sdd check` instead (`sdd adopt` is planned as T-004), or use --force if you really want to proceed", scaffold.ManifestPath)
	}

	det, err := detect.Detect(root)
	if err != nil {
		return fail("%v", err)
	}
	printDetected(env, det)

	var pr *prompter
	if env.Interactive && !o.yes {
		pr = newPrompter(env.In, env.Out)
		fprintln(env.Out, "\nAnswer each question, or press Enter to accept the value in brackets.")
	}
	ans, err := collectAnswers(env, pr, o, given, det, root)
	if err != nil {
		return fail("%v", err)
	}
	set, err := templates.New(o.templates)
	if err != nil {
		return fail("%v", err)
	}
	plan, err := scaffold.BuildPlan(root, ans, det, set, env.now(), o.force)
	if err != nil {
		return fail("%v", err)
	}

	fprintf(env.Out, "\nPlan for %s (%s):\n", root, plan.Data.Mode)
	for _, l := range plan.Lines() {
		fprintf(env.Out, "  %s\n", l)
	}
	fprintf(env.Out, "  %-8s %s\n", "create", scaffold.ManifestPath)
	if o.dryRun {
		fprintln(env.Out, "\nDry run: nothing was written.")
		return 0
	}
	if pr != nil && !pr.confirm("\nWrite these files?") {
		fprintln(env.Out, "Aborted. Nothing was written.")
		return 1
	}

	rep, err := scaffold.Apply(plan, env.now(), o.templates)
	if err != nil {
		return fail("%v", err)
	}
	counts := map[scaffold.Kind]int{}
	for _, e := range rep.Entries {
		counts[e.Kind]++
	}
	fprintf(env.Out, "\nDone: %d created, %d merged, %d replaced, %d skipped.\n",
		counts[scaffold.KindCreate], counts[scaffold.KindMerge], counts[scaffold.KindReplace], counts[scaffold.KindSkip])
	if rep.BackupDir != "" {
		fprintf(env.Out, "Originals backed up in %s\n", rep.BackupDir)
	}
	fprintf(env.Out, "\nNext steps:\n")
	fprintf(env.Out, "  1. Review the generated files. Nothing is committed; you commit (agents never do).\n")
	fprintf(env.Out, "  2. Fill the %d %s placeholder(s), or let the agent propose them in T-001 under the gates.\n", rep.Todos, scaffold.Todo)
	fprintf(env.Out, "  3. Run `sdd check`.\n")
	fprintf(env.Out, "  4. Hand over to an agent: `sdd prompt gemini` (or claude, agents).\n")
	return 0
}

func printDetected(env IO, det detect.Result) {
	fprintln(env.Out, "Detected (unverified; you confirm or correct each value):")
	langs := strings.Join(det.Languages, ", ")
	if langs == "" {
		langs = "none"
	}
	if len(det.Other) > 0 {
		langs += "  (other: " + strings.Join(det.Other, ", ") + ")"
	}
	fprintf(env.Out, "  languages      %s\n", langs)
	fprintf(env.Out, "  source files   %d%s\n", det.SourceFiles, map[bool]string{true: " (scan capped)", false: ""}[det.Capped])
	fprintf(env.Out, "  git repository %s\n", map[bool]string{true: "yes", false: "no"}[det.HasGit])
	fprintf(env.Out, "  suggested mode %s\n", det.SuggestedMode)
	if det.Build+det.Test+det.Lint+det.Format != "" {
		fprintf(env.Out, "  commands       build=%q test=%q lint=%q format=%q\n", det.Build, det.Test, det.Lint, det.Format)
	}
	var existing []string
	for _, n := range []string{"README.md", "CLAUDE.md", "GEMINI.md", "AGENTS.md", "MISSION.md", "TECH_STACK.md", "ROADMAP.md", "SKILLS.md"} {
		if det.Existing[n] {
			existing = append(existing, n)
		}
	}
	if len(existing) > 0 {
		fprintf(env.Out, "  existing docs  %s\n", strings.Join(existing, ", "))
	}
}

func collectAnswers(env IO, pr *prompter, o initOpts, given map[string]bool, det detect.Result, root string) (scaffold.Answers, error) {
	var a scaffold.Answers
	str := func(flagName, flagVal, label, def string) string {
		switch {
		case given[flagName]:
			return flagVal
		case pr != nil:
			return pr.ask(label, def)
		default:
			return def
		}
	}
	choose := func(flagName, flagVal, label, def string, validate func(string) error) (string, error) {
		switch {
		case given[flagName]:
			return flagVal, nil
		case pr != nil:
			return pr.askValid(label, def, validate)
		default:
			return def, nil
		}
	}

	mode, err := choose("mode", o.mode, "Project mode (greenfield or brownfield)", det.SuggestedMode, func(v string) error {
		v = strings.ToLower(strings.TrimSpace(v))
		if v != scaffold.ModeGreenfield && v != scaffold.ModeBrownfield {
			return errors.New("answer greenfield or brownfield")
		}
		return nil
	})
	if err != nil {
		return a, err
	}
	a.Mode = strings.ToLower(strings.TrimSpace(mode))
	a.Project = str("name", o.name, "Project name", filepath.Base(root))
	a.Purpose = str("purpose", o.purpose, "One-line purpose (blank to fill in later)", "")

	langDef := strings.Join(det.Languages, ",")
	if langDef == "" {
		langDef = "other"
	}
	langs, err := choose("lang", o.lang, "Languages ("+strings.Join(templates.LanguageIDs(), ", ")+")", langDef, func(v string) error {
		_, err := scaffold.ParseLanguages(splitList(v))
		return err
	})
	if err != nil {
		return a, err
	}
	a.Languages = splitList(langs)

	a.Maintainer = str("maintainer", o.maintainer, "Maintainer (the human who approves and commits)", env.getenv("USER"))
	a.Build = str("build", o.build, "Build command (blank if unknown)", det.Build)
	a.Test = str("test", o.test, "Test command (blank if unknown)", det.Test)
	a.Lint = str("lint", o.lint, "Lint command (blank if unknown)", det.Lint)
	a.Format = str("format", o.format, "Format command (blank if unknown)", det.Format)

	agents, err := choose("agents", o.agents, "Agent files ("+strings.Join(scaffold.AgentIDs(), ", ")+")", strings.Join(scaffold.AgentIDs(), ","), func(v string) error {
		_, err := scaffold.ParseAgents(splitList(v))
		return err
	})
	if err != nil {
		return a, err
	}
	a.Agents = splitList(agents)

	switch {
	case len(o.rules) > 0:
		a.ExtraRules = o.rules
	case pr != nil:
		a.ExtraRules = pr.askLines("Extra hard rules for agents, if any")
	}
	return a, nil
}

// guardRoot refuses to initialise a directory that is clearly not a project.
func guardRoot(root, home string) error {
	if root == filepath.Dir(root) {
		return errors.New("refusing to run in the filesystem root")
	}
	if home != "" && samePath(root, home) {
		return errors.New("refusing to run in your home directory; cd into a project directory first")
	}
	info, err := os.Stat(root)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", root)
	}
	return nil
}

func samePath(a, b string) bool {
	ra, err1 := filepath.EvalSymlinks(a)
	rb, err2 := filepath.EvalSymlinks(b)
	if err1 == nil && err2 == nil {
		return ra == rb
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

// handMadeKit reports whether all four core documents already exist, which
// means a kit was written without sdd (there is no manifest, or init would
// have refused earlier).
func handMadeKit(root string) bool {
	for _, name := range []string{"MISSION.md", "TECH_STACK.md", "ROADMAP.md", "SKILLS.md"} {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			return false
		}
	}
	return true
}
