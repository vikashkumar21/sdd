package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vikashkumar21/sdd/internal/check"
	"github.com/vikashkumar21/sdd/internal/feature"
	"github.com/vikashkumar21/sdd/internal/roadmap"
	"github.com/vikashkumar21/sdd/internal/scaffold"
	"github.com/vikashkumar21/sdd/internal/templates"
)

func runCheck(args []string, env IO) int {
	fs := newFlagSet("check", env)
	dir := fs.String("dir", ".", "project root")
	strict := fs.Bool("strict", false, "treat warnings (including unresolved TODO(human) placeholders) as failures")
	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return parseExit(err)
	}
	if len(pos) > 0 {
		fmt.Fprintf(env.Err, "sdd check: unexpected argument %q\n", pos[0])
		return 2
	}
	root, err := filepath.Abs(*dir)
	if err != nil {
		fmt.Fprintf(env.Err, "sdd check: %v\n", err)
		return 1
	}
	rep, err := check.Run(root)
	if err != nil {
		fmt.Fprintf(env.Err, "sdd check: %v\n", err)
		return 1
	}
	for _, f := range rep.Findings {
		loc := f.File
		if loc == "" {
			loc = "(project)"
		} else if f.Line > 0 {
			loc = fmt.Sprintf("%s:%d", f.File, f.Line)
		}
		fmt.Fprintf(env.Out, "%-5s %s  %s\n", f.Severity, loc, f.Message)
	}
	fmt.Fprintf(env.Out, "\n%d error(s), %d warning(s), %d note(s)\n", rep.Count(check.Error), rep.Count(check.Warn), rep.Count(check.Info))
	if rep.Failed(*strict) {
		return 1
	}
	return 0
}

func runFeature(args []string, env IO) int {
	if len(args) == 0 || args[0] != "new" {
		fmt.Fprintln(env.Err, "usage: sdd feature new <T-xxx> <slug> [--title \"...\"] [--dir <project>]")
		return 2
	}
	fs := newFlagSet("feature new", env)
	dir := fs.String("dir", ".", "project root")
	title := fs.String("title", "", "task title (default: the title in ROADMAP.md)")
	pos, err := parseInterspersed(fs, args[1:])
	if err != nil {
		return parseExit(err)
	}
	if len(pos) != 2 {
		fmt.Fprintln(env.Err, "usage: sdd feature new <T-xxx> <slug> [--title \"...\"] [--dir <project>]")
		return 2
	}
	root, err := filepath.Abs(*dir)
	if err != nil {
		fmt.Fprintf(env.Err, "sdd feature: %v\n", err)
		return 1
	}
	res, err := feature.New(root, pos[0], pos[1], *title)
	if err != nil {
		fmt.Fprintf(env.Err, "sdd feature: %v\n", err)
		return 1
	}
	fmt.Fprintf(env.Out, "Created %s:\n", res.Dir)
	for _, f := range res.Files {
		fmt.Fprintf(env.Out, "  %s\n", f)
	}
	if res.StatusUpdated {
		fmt.Fprintf(env.Out, "ROADMAP.md: %s is now Specifying.\n", pos[0])
	}
	fmt.Fprintf(env.Out, "\nNext: draft requirements.md, plan.md and validation.md (no code), then stop for `APPROVED %s`.\n", pos[0])
	return 0
}

type taskView struct{ ID, Title, Status string }

type promptData struct {
	Project, Root, AgentName, AgentFile, Date string
	AgentFiles                                []string
	OpenTasks                                 []taskView
	FocusTask                                 string
}

func agentFor(name string) (display, file string) {
	switch strings.ToLower(name) {
	case "claude":
		return "Claude", "CLAUDE.md"
	case "gemini":
		return "Gemini", "GEMINI.md"
	case "agents":
		return "an AI coding agent", "AGENTS.md"
	default:
		return name, "AGENTS.md"
	}
}

func runPrompt(args []string, env IO) int {
	fs := newFlagSet("prompt", env)
	dir := fs.String("dir", ".", "project root")
	task := fs.String("task", "", "focus task, for example T-001 (default: the first open task)")
	tpl := fs.String("templates", "", "directory whose files override the embedded templates")
	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return parseExit(err)
	}
	if len(pos) != 1 {
		fmt.Fprintln(env.Err, "usage: sdd prompt <agent> [--task T-xxx] [--dir <project>]   (agent: claude, gemini, agents, or any name)")
		return 2
	}
	fail := func(format string, a ...any) int {
		fmt.Fprintf(env.Err, "sdd prompt: "+format+"\n", a...)
		return 1
	}
	root, err := filepath.Abs(*dir)
	if err != nil {
		return fail("%v", err)
	}
	display, agentFile := agentFor(pos[0])
	if _, err := os.Stat(filepath.Join(root, agentFile)); err != nil {
		return fail("%s not found in %s; run `sdd init` first", agentFile, root)
	}

	project := filepath.Base(root)
	var agentFiles []string
	if m, ok, err := scaffold.ReadManifest(root); err != nil {
		return fail("%v", err)
	} else if ok {
		project = m.Answers.Project
		for _, id := range m.Answers.Agents {
			agentFiles = append(agentFiles, scaffold.AgentFile(id))
		}
	}
	if len(agentFiles) == 0 {
		for _, f := range []string{"CLAUDE.md", "GEMINI.md", "AGENTS.md"} {
			if _, err := os.Stat(filepath.Join(root, f)); err == nil {
				agentFiles = append(agentFiles, f)
			}
		}
	}

	data := promptData{Project: project, Root: root, AgentName: display, AgentFile: agentFile, AgentFiles: agentFiles, Date: env.now().UTC().Format("2006-01-02")}
	text, err := os.ReadFile(filepath.Join(root, "ROADMAP.md"))
	if err != nil {
		return fail("read ROADMAP.md: %v", err)
	}
	var all []roadmap.Task
	for _, t := range roadmap.Parse(strings.ReplaceAll(string(text), "\r\n", "\n")) {
		all = append(all, t)
		if roadmap.Kind(t.Status) != "done" {
			data.OpenTasks = append(data.OpenTasks, taskView{ID: t.ID, Title: t.Title, Status: roadmap.Normalize(t.Status)})
		}
	}
	focus, ferr := pickFocus(all, *task)
	if ferr != nil {
		return fail("%v", ferr)
	}
	data.FocusTask = focus

	set, err := templates.New(*tpl)
	if err != nil {
		return fail("%v", err)
	}
	out, err := set.Render("base/prompt.md.tmpl", data)
	if err != nil {
		return fail("%v", err)
	}
	fmt.Fprint(env.Out, out)
	return 0
}

func pickFocus(all []roadmap.Task, wanted string) (string, error) {
	describe := func(t roadmap.Task) string {
		return fmt.Sprintf("%s — %s (%s)", t.ID, t.Title, roadmap.Normalize(t.Status))
	}
	if wanted != "" {
		for _, t := range all {
			if t.ID == wanted {
				return describe(t), nil
			}
		}
		return "", fmt.Errorf("%s is not a task in ROADMAP.md", wanted)
	}
	for _, t := range all {
		switch roadmap.Kind(t.Status) {
		case "not started", "specifying", "awaiting plan approval", "implementing", "awaiting review":
			return describe(t), nil
		}
	}
	return "", nil
}
