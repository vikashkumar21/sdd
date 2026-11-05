package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vikashkumar21/sdd/internal/check"
	"github.com/vikashkumar21/sdd/internal/context"
	"github.com/vikashkumar21/sdd/internal/feature"
	"github.com/vikashkumar21/sdd/internal/repair"
	"github.com/vikashkumar21/sdd/internal/roadmap"
	"github.com/vikashkumar21/sdd/internal/scaffold"
	"github.com/vikashkumar21/sdd/internal/skill"
	"github.com/vikashkumar21/sdd/internal/suppress"
	"github.com/vikashkumar21/sdd/internal/templates"
)

func runCheck(args []string, env IO) int {
	fs := newFlagSet("check", env)
	dir := fs.String("dir", ".", "project root")
	strict := fs.Bool("strict", false, "treat warnings (including unresolved TODO(human) placeholders) as failures")
	jsonOut := fs.Bool("json", false, "output findings as structured JSON")
	fix := fs.Bool("fix", false, "safely repair mechanical issues (contract parity, table alignment)")
	dryRun := fs.Bool("dry-run", false, "preview repairs without applying changes")
	yes := fs.Bool("yes", false, "apply repairs without interactive confirmation")
	fileFlag := fs.String("file", "", "restrict repairs to specified comma-separated file(s)")
	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return parseExit(err)
	}
	if len(pos) > 0 {
		fprintf(env.Err, "sdd check: unexpected argument %q\n", pos[0])
		return 2
	}
	root, err := filepath.Abs(*dir)
	if err != nil {
		fprintf(env.Err, "sdd check: %v\n", err)
		return 1
	}

	if *fix {
		var filterFiles []string
		if *fileFlag != "" {
			for _, part := range strings.Split(*fileFlag, ",") {
				p := strings.TrimSpace(part)
				if p != "" {
					filterFiles = append(filterFiles, p)
				}
			}
		}
		plan, perr := repair.PlanRepairs(root, filterFiles)
		if perr != nil {
			fprintf(env.Err, "sdd check --fix: %v\n", perr)
			return 1
		}

		checkRep, _ := check.Run(root)

		if len(plan.Patches) == 0 && len(checkRep.StaleWaivers) == 0 {
			fprintln(env.Out, "No mechanical repairs needed.")
			return 0
		}

		p := newPrompter(env.In, env.Out)

		if len(plan.Patches) > 0 {
			fprintln(env.Out, "Proposed repairs:")
			for _, patch := range plan.Patches {
				if patch.Suppressed {
					fprintf(env.Out, "\n%s (%s) [SUPPRESSED: %s]:\n", patch.Path, patch.Description, patch.SuppressReason)
				} else {
					fprintf(env.Out, "\n%s (%s):\n", patch.Path, patch.Description)
				}
				fprintln(env.Out, strings.TrimRight(patch.Diff, "\n"))
			}

			if *dryRun {
				fprintf(env.Out, "\nDry run: %d file(s) would be repaired. No changes written.\n", len(plan.Patches))
				return 0
			}

			approved := map[string]bool{}
			if *yes {
				for _, patch := range plan.Patches {
					approved[patch.Path] = true
				}
			} else {
				if len(plan.Patches) == 1 {
					patch := plan.Patches[0]
					ans := p.ask(fmt.Sprintf("\nApply repair to %s? [y]es / [n]o / [s]uppress", patch.Path), "n")
					switch strings.ToLower(strings.TrimSpace(ans)) {
					case "y", "yes":
						approved[patch.Path] = true
					case "s", "suppress":
						reason := p.ask("Waiver reason", "")
						if reason == "" {
							reason = "waived via sdd check --fix"
						}
						_ = suppress.Add(root, suppress.Entry{
							File:   patch.Path,
							Rule:   patch.Rule,
							Reason: reason,
							Date:   env.now().Format("2006-01-02"),
						})
						fprintf(env.Out, "Recorded waiver for %s (%s).\n", patch.Path, patch.Rule)
					}
				} else {
					answer := p.ask("\nApply repairs? [y]es to all / [n]o / [p]er-file", "n")
					switch strings.ToLower(strings.TrimSpace(answer)) {
					case "y", "yes":
						for _, patch := range plan.Patches {
							approved[patch.Path] = true
						}
					case "p", "per-file":
						for _, patch := range plan.Patches {
							ans := p.ask(fmt.Sprintf("Apply repair to %s? [y]es / [n]o / [s]uppress", patch.Path), "n")
							switch strings.ToLower(strings.TrimSpace(ans)) {
							case "y", "yes":
								approved[patch.Path] = true
							case "s", "suppress":
								reason := p.ask("Waiver reason", "")
								if reason == "" {
									reason = "waived via sdd check --fix"
								}
								_ = suppress.Add(root, suppress.Entry{
									File:   patch.Path,
									Rule:   patch.Rule,
									Reason: reason,
									Date:   env.now().Format("2006-01-02"),
								})
								fprintf(env.Out, "Recorded waiver for %s (%s).\n", patch.Path, patch.Rule)
							}
						}
					default:
						fprintln(env.Out, "Repairs aborted. No changes written.")
					}
				}
			}

			if len(approved) > 0 {
				res, aerr := repair.Apply(root, plan, approved, env.now())
				if aerr != nil {
					fprintf(env.Err, "sdd check --fix: apply: %v\n", aerr)
					return 1
				}
				fprintf(env.Out, "\nRepaired %d file(s). Original(s) backed up in %s/\n", len(res.Repaired), res.BackupDir)
			}
		}

		if len(checkRep.StaleWaivers) > 0 && !*dryRun {
			fprintln(env.Out, "\nStale waiver(s) found in .sdd/suppressions.json (no longer triggered):")
			for _, sw := range checkRep.StaleWaivers {
				fprintf(env.Out, "  - %s (%s): %s\n", sw.File, sw.Rule, sw.Reason)
			}
			if *yes || p.confirm("Remove stale waiver(s) from .sdd/suppressions.json?") {
				if err := suppress.RemoveEntries(root, checkRep.StaleWaivers); err == nil {
					fprintf(env.Out, "Pruned %d stale waiver(s) from .sdd/suppressions.json.\n", len(checkRep.StaleWaivers))
				}
			}
		}

		return 0
	}

	rep, err := check.Run(root)
	if err != nil {
		fprintf(env.Err, "sdd check: %v\n", err)
		return 1
	}

	if *jsonOut {
		data, jerr := rep.JSON(*strict)
		if jerr != nil {
			fprintf(env.Err, "sdd check: marshal json: %v\n", jerr)
			return 1
		}
		fprintln(env.Out, string(data))
		if rep.Failed(*strict) {
			return 1
		}
		return 0
	}

	for _, f := range rep.Findings {
		loc := f.File
		if loc == "" {
			loc = "(project)"
		} else if f.Line > 0 {
			loc = fmt.Sprintf("%s:%d", f.File, f.Line)
		}
		fprintf(env.Out, "%-5s %s  %s\n", f.Severity, loc, f.Message)
	}
	fprintf(env.Out, "\n%d error(s), %d warning(s), %d note(s)\n", rep.Count(check.Error), rep.Count(check.Warn), rep.Count(check.Info))
	if rep.Failed(*strict) {
		return 1
	}
	return 0
}

func runFeature(args []string, env IO) int {
	return runTaskNew(args, feature.TypologyFeature, "feature", env)
}

type taskView struct{ ID, Title, Status string }

type contextView struct {
	ID     string
	Source string
	Origin string
	Path   string
	Task   string
	Format string
	Bytes  int64
}

type skillView struct {
	Name        string
	Scope       string
	Description string
	Path        string
	Valid       bool
}

type promptData struct {
	Project, Root, AgentName, AgentFile, Date string
	AgentFiles                                []string
	OpenTasks                                 []taskView
	FocusTask                                 string
	FocusTaskID                               string
	StagedContext                             []contextView
	TaskContext                               []contextView
	Skills                                    []skillView
	HasRefineSkills                           bool
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
		fprintln(env.Err, "usage: sdd prompt <agent> [--task T-xxx] [--dir <project>]   (agent: claude, gemini, agents, or any name)")
		return 2
	}
	fail := func(format string, a ...any) int {
		fprintf(env.Err, "sdd prompt: "+format+"\n", a...)
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
	focus, focusID, ferr := pickFocus(all, *task)
	if ferr != nil {
		return fail("%v", ferr)
	}
	data.FocusTask = focus
	data.FocusTaskID = focusID

	if cm, ok, err := context.ReadManifest(root); err == nil && ok {
		for _, e := range cm.Entries {
			formatDisplay := e.Format
			if strings.EqualFold(e.Format, "pdf") {
				formatDisplay = "pdf (multimodal inspection)"
			}
			cv := contextView{
				ID:     e.ID,
				Source: e.Source,
				Origin: e.Origin,
				Path:   e.Path,
				Task:   e.Task,
				Format: formatDisplay,
				Bytes:  e.Bytes,
			}
			data.StagedContext = append(data.StagedContext, cv)
			if focusID != "" && e.Task == focusID {
				data.TaskContext = append(data.TaskContext, cv)
			}
		}
	}

	if discovered, err := skill.Discover(root); err == nil && len(discovered) > 0 {
		var svList []skillView
		hasRefine := false
		for _, sk := range discovered {
			name := sk.Skill.Name
			if name == "" {
				name = filepath.Base(sk.Dir)
			}
			scope := "project"
			if strings.HasPrefix(filepath.ToSlash(sk.File), ".sdd/skills/") {
				scope = "internal"
			}
			valid := len(sk.Errors) == 0
			svList = append(svList, skillView{
				Name:        name,
				Scope:       scope,
				Description: sk.Skill.Description,
				Path:        sk.File,
				Valid:       valid,
			})
			if strings.Contains(sk.Skill.Body, "sdd:skill-refine:begin") {
				hasRefine = true
			}
		}
		sort.Slice(svList, func(i, j int) bool {
			return svList[i].Name < svList[j].Name
		})
		data.Skills = svList
		data.HasRefineSkills = hasRefine
	}

	set, err := templates.New(*tpl)
	if err != nil {
		return fail("%v", err)
	}
	out, err := set.Render("base/prompt.md.tmpl", data)
	if err != nil {
		return fail("%v", err)
	}
	fprint(env.Out, out)
	return 0
}

func pickFocus(all []roadmap.Task, wanted string) (string, string, error) {
	describe := func(t roadmap.Task) string {
		return fmt.Sprintf("%s — %s (%s)", t.ID, t.Title, roadmap.Normalize(t.Status))
	}
	if wanted != "" {
		for _, t := range all {
			if t.ID == wanted {
				return describe(t), t.ID, nil
			}
		}
		return "", "", fmt.Errorf("%s is not a task in ROADMAP.md", wanted)
	}
	for _, t := range all {
		switch roadmap.Kind(t.Status) {
		case "not started", "specifying", "awaiting plan approval", "implementing", "awaiting review":
			return describe(t), t.ID, nil
		}
	}
	return "", "", nil
}
