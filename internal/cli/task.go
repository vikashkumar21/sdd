package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vikashkumar21/sdd/internal/coordinator"
	"github.com/vikashkumar21/sdd/internal/deconflict"
	"github.com/vikashkumar21/sdd/internal/feature"
	"github.com/vikashkumar21/sdd/internal/roadmap"
	"github.com/vikashkumar21/sdd/internal/state"
	"github.com/vikashkumar21/sdd/internal/worktree"
)

func runBugfix(args []string, env IO) int {
	return runTaskNew(args, feature.TypologyBugfix, "bugfix", env)
}

func runHotfix(args []string, env IO) int {
	return runTaskNew(args, feature.TypologyHotfix, "hotfix", env)
}

func runSpike(args []string, env IO) int {
	return runTaskNew(args, feature.TypologySpike, "spike", env)
}

func runTask(args []string, env IO) int {
	if len(args) == 0 {
		fprintln(env.Err, "usage: sdd task <command> [flags]\n\ncommands:\n  new <T-xxx> <slug>     create a task folder (--type feature|bugfix|hotfix|spike)\n  start <T-xxx>          provision an isolated Git worktree for the task\n  clean <T-xxx>          safely remove a task's worktree with backup\n  list                   list active task worktrees\n  deconflict [tasks...]  analyze plan file collisions and schedule parallel batches\n  merge <T-xxx>          qualify pre-merge matrix and merge task into target branch\n  state <command>        manage deterministic task state and events (show|save|append-event|purge)")
		return 2
	}
	switch args[0] {
	case "new":
		return runTaskNewWithFlag(args[1:], env)
	case "start":
		return runTaskStart(args[1:], env)
	case "clean":
		return runTaskClean(args[1:], env)
	case "list":
		return runTaskList(args[1:], env)
	case "deconflict":
		return runTaskDeconflict(args[1:], env)
	case "merge":
		return runTaskMerge(args[1:], env)
	case "state":
		return runTaskState(args[1:], env)
	default:
		fprintf(env.Err, "sdd task: unknown command %q\n\nusage: sdd task <new|start|clean|list|deconflict|merge|state> [flags]\n", args[0])
		return 2
	}
}

func runTaskStart(args []string, env IO) int {
	fs := newFlagSet("task start", env)
	dir := fs.String("dir", ".", "project root")
	base := fs.String("base", "", "base ref or branch (default: HEAD)")
	fromTag := fs.String("from-tag", "", "tag to branch from (primarily for hotfixes)")
	force := fs.Bool("force", false, "recreate worktree if already exists")

	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return parseExit(err)
	}
	if len(pos) != 1 {
		fprintln(env.Err, "usage: sdd task start <T-xxx> [--base <ref>] [--from-tag <tag>] [--force] [--dir <project>]")
		return 2
	}

	root, err := filepath.Abs(*dir)
	if err != nil {
		fprintf(env.Err, "sdd task start: %v\n", err)
		return 1
	}

	opts := worktree.StartOptions{
		Base:    *base,
		FromTag: *fromTag,
		Force:   *force,
		Now:     env.now(),
	}

	entry, err := worktree.Start(root, pos[0], opts)
	if err != nil {
		fprintf(env.Err, "sdd task start: %v\n", err)
		return 1
	}

	fprintf(env.Out, "Provisioned worktree for %s:\n", entry.TaskID)
	fprintf(env.Out, "  Worktree: %s\n", entry.Path)
	fprintf(env.Out, "  Branch:   %s\n", entry.Branch)
	fprintf(env.Out, "  Base:     %s\n", entry.BaseRef)
	fprintf(env.Out, "  Typology: %s\n\n", entry.Typology)
	fprintf(env.Out, "To begin work in the isolated workspace:\n")
	fprintf(env.Out, "  cd %s\n", entry.Path)
	return 0
}

func runTaskClean(args []string, env IO) int {
	fs := newFlagSet("task clean", env)
	dir := fs.String("dir", ".", "project root")
	force := fs.Bool("force", false, "remove even if uncommitted changes exist (backs up to .sdd/backup/)")
	deleteBranch := fs.Bool("delete-branch", false, "also delete the local task git branch")
	purgeState := fs.Bool("purge-state", false, "completely purge task state directory (.sdd/state/<task-id>)")

	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return parseExit(err)
	}
	if len(pos) != 1 {
		fprintln(env.Err, "usage: sdd task clean <T-xxx> [--force] [--delete-branch] [--purge-state] [--dir <project>]")
		return 2
	}

	root, err := filepath.Abs(*dir)
	if err != nil {
		fprintf(env.Err, "sdd task clean: %v\n", err)
		return 1
	}

	opts := worktree.CleanOptions{
		Force:        *force,
		DeleteBranch: *deleteBranch,
		Now:          env.now(),
	}

	if err := worktree.Clean(root, pos[0], opts); err != nil {
		fprintf(env.Err, "sdd task clean: %v\n", err)
		return 1
	}

	fprintf(env.Out, "Cleaned worktree for %s.\n", strings.ToUpper(pos[0]))

	if *purgeState {
		purgeRes, err := state.Purge(root, pos[0])
		if err != nil {
			fprintf(env.Err, "sdd task clean: purge state: %v\n", err)
			return 1
		}
		fprintf(env.Out, "Purged state directory for %s (%s).\n", purgeRes.TaskID, purgeRes.Path)
	}

	return 0
}

func runTaskList(args []string, env IO) int {
	fs := newFlagSet("task list", env)
	dir := fs.String("dir", ".", "project root")
	jsonFlag := fs.Bool("json", false, "output structured JSON")

	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return parseExit(err)
	}
	if len(pos) > 0 {
		fprintln(env.Err, "usage: sdd task list [--json] [--dir <project>]")
		return 2
	}

	root, err := filepath.Abs(*dir)
	if err != nil {
		fprintf(env.Err, "sdd task list: %v\n", err)
		return 1
	}

	entries, err := worktree.List(root)
	if err != nil {
		fprintf(env.Err, "sdd task list: %v\n", err)
		return 1
	}

	if *jsonFlag {
		data, err := json.MarshalIndent(entries, "", "  ")
		if err != nil {
			fprintf(env.Err, "sdd task list: %v\n", err)
			return 1
		}
		fprintln(env.Out, string(data))
		return 0
	}

	if len(entries) == 0 {
		fprintln(env.Out, "No active task worktrees.")
		return 0
	}

	fprintln(env.Out, "Active task worktrees:")
	for _, e := range entries {
		fprintf(env.Out, "  %s (%s): %s\n", e.TaskID, e.Typology, e.Path)
		fprintf(env.Out, "      Branch: %s\n", e.Branch)
		fprintf(env.Out, "      Base:   %s\n", e.BaseRef)
	}
	return 0
}

func runTaskNewWithFlag(args []string, env IO) int {
	fs := newFlagSet("task new", env)
	typeFlag := fs.String("type", "feature", "task typology: feature, bugfix, hotfix, or spike")
	dir := fs.String("dir", ".", "project root")
	title := fs.String("title", "", "task title (default: the title in ROADMAP.md)")
	fromJira := fs.String("from-jira", "", "path or raw text of a Jira issue to import and stage into context")
	fromDoc := fs.String("from-doc", "", "path to reference document or PDF to stage into context")

	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return parseExit(err)
	}
	if len(pos) != 2 {
		fprintln(env.Err, "usage: sdd task new [--type feature|bugfix|hotfix|spike] <T-xxx> <slug> [--title \"...\"] [--from-jira <file|string>] [--from-doc <file>] [--dir <project>]")
		return 2
	}

	typ, err := feature.ParseTypology(*typeFlag)
	if err != nil {
		fprintf(env.Err, "sdd task new: %v\n", err)
		return 2
	}

	return executeTaskScaffold(pos[0], pos[1], typ, *dir, *title, *fromJira, *fromDoc, "task new", env)
}

func runTaskNew(args []string, defaultTypology feature.Typology, cmdName string, env IO) int {
	if len(args) == 0 || args[0] != "new" {
		fprintf(env.Err, "usage: sdd %s new <T-xxx> <slug> [--title \"...\"] [--from-jira <file|string>] [--from-doc <file>] [--dir <project>]\n", cmdName)
		return 2
	}
	fs := newFlagSet(cmdName+" new", env)
	dir := fs.String("dir", ".", "project root")
	title := fs.String("title", "", "task title (default: the title in ROADMAP.md)")
	fromJira := fs.String("from-jira", "", "path or raw text of a Jira issue to import and stage into context")
	fromDoc := fs.String("from-doc", "", "path to reference document or PDF to stage into context")
	pos, err := parseInterspersed(fs, args[1:])
	if err != nil {
		return parseExit(err)
	}
	if len(pos) != 2 {
		fprintf(env.Err, "usage: sdd %s new <T-xxx> <slug> [--title \"...\"] [--from-jira <file|string>] [--from-doc <file>] [--dir <project>]\n", cmdName)
		return 2
	}

	return executeTaskScaffold(pos[0], pos[1], defaultTypology, *dir, *title, *fromJira, *fromDoc, cmdName, env)
}

func executeTaskScaffold(id, slug string, typ feature.Typology, dir, title, fromJira, fromDoc, cmdName string, env IO) int {
	root, err := filepath.Abs(dir)
	if err != nil {
		fprintf(env.Err, "sdd %s: %v\n", cmdName, err)
		return 1
	}

	var jiraData []byte
	var jiraOrigin string
	if fromJira != "" {
		filePath := fromJira
		if !filepath.IsAbs(filePath) {
			if _, err := os.Stat(filePath); err != nil {
				candidate := filepath.Join(root, filePath)
				if _, err2 := os.Stat(candidate); err2 == nil {
					filePath = candidate
				}
			}
		}
		if data, err := os.ReadFile(filePath); err == nil {
			jiraData = data
			jiraOrigin = filepath.Base(filePath)
		} else {
			if strings.Contains(filePath, ".") || strings.Contains(filePath, "/") || strings.Contains(filePath, "\\") {
				fprintf(env.Err, "sdd %s: read jira file %s: %v\n", cmdName, fromJira, err)
				return 1
			}
			jiraData = []byte(fromJira)
			jiraOrigin = "inline"
		}
	}

	var docData []byte
	var docPath string
	if fromDoc != "" {
		filePath := fromDoc
		if !filepath.IsAbs(filePath) {
			if _, err := os.Stat(filePath); err != nil {
				candidate := filepath.Join(root, filePath)
				if _, err2 := os.Stat(candidate); err2 == nil {
					filePath = candidate
				}
			}
		}
		data, err := os.ReadFile(filePath)
		if err != nil {
			fprintf(env.Err, "sdd %s: read doc file %s: %v\n", cmdName, fromDoc, err)
			return 1
		}
		docData = data
		docPath = filePath
	}

	opts := feature.Options{
		Type:         typ,
		Title:        title,
		FromJiraData: jiraData,
		JiraOrigin:   jiraOrigin,
		FromDocData:  docData,
		FromDocPath:  docPath,
		Now:          env.now(),
	}
	res, err := feature.NewWithOptions(root, id, slug, opts)
	if err != nil {
		fprintf(env.Err, "sdd %s: %v\n", cmdName, err)
		return 1
	}

	fprintf(env.Out, "Created %s:\n", res.Dir)
	for _, f := range res.Files {
		fprintf(env.Out, "  %s\n", f)
	}
	if res.StagedContext != "" {
		fprintf(env.Out, "Staged context: %s\n", res.StagedContext)
	}
	if res.StatusUpdated {
		fprintf(env.Out, "ROADMAP.md: %s is now Specifying.\n", id)
	}
	fprintln(env.Out)
	fprintf(env.Out, "Next: draft task documents (no code), then stop for `APPROVED %s`.\n", id)
	return 0
}

func runTaskDeconflict(args []string, env IO) int {
	fs := newFlagSet("task deconflict", env)
	dir := fs.String("dir", ".", "project root")
	jsonFlag := fs.Bool("json", false, "output structured JSON")
	allFlag := fs.Bool("all", false, "analyze all tasks in roadmap that have task folders")

	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return parseExit(err)
	}

	root, err := filepath.Abs(*dir)
	if err != nil {
		fprintf(env.Err, "sdd task deconflict: %v\n", err)
		return 1
	}

	var taskIDs []string
	if len(pos) > 0 {
		for _, p := range pos {
			taskIDs = append(taskIDs, strings.ToUpper(strings.TrimSpace(p)))
		}
	} else {
		// Read ROADMAP.md
		roadmapPath := filepath.Join(root, "ROADMAP.md")
		data, err := os.ReadFile(roadmapPath)
		if err != nil {
			fprintf(env.Err, "sdd task deconflict: read ROADMAP.md: %v\n", err)
			return 1
		}
		tasks := roadmap.Parse(string(data))
		for _, t := range tasks {
			if *allFlag {
				if _, err := worktree.FindTask(root, t.ID); err == nil {
					taskIDs = append(taskIDs, t.ID)
				}
			} else {
				switch roadmap.Kind(t.Status) {
				case "specifying", "awaiting plan approval", "implementing", "awaiting review":
					if _, err := worktree.FindTask(root, t.ID); err == nil {
						taskIDs = append(taskIDs, t.ID)
					}
				}
			}
		}
	}

	if len(taskIDs) < 2 {
		if len(pos) > 0 {
			fprintln(env.Err, "sdd task deconflict: specify at least 2 tasks to analyze for conflicts, or use --all to analyze all roadmap tasks")
			return 2
		}
		if len(taskIDs) == 1 {
			fprintf(env.Out, "Only 1 in-flight task found (%s). No concurrency conflict possible.\n", taskIDs[0])
			return 0
		}
		fprintln(env.Out, "No in-flight tasks found to deconflict. Use 'sdd task deconflict <T-xxx> <T-yyy>' or --all.")
		return 0
	}

	var plans []deconflict.TaskPlan
	for _, id := range taskIDs {
		plan, err := deconflict.LoadTaskPlan(root, id)
		if err != nil {
			fprintf(env.Err, "sdd task deconflict: task %s: %v\n", id, err)
			return 1
		}
		plans = append(plans, *plan)
	}

	rep := deconflict.Analyze(plans)

	if *jsonFlag {
		data, err := json.MarshalIndent(rep, "", "  ")
		if err != nil {
			fprintf(env.Err, "sdd task deconflict: %v\n", err)
			return 1
		}
		fprintln(env.Out, string(data))
		return 0
	}

	fprint(env.Out, deconflict.FormatText(rep))
	return 0
}

func runTaskMerge(args []string, env IO) int {
	fs := newFlagSet("task merge", env)
	dir := fs.String("dir", ".", "project root")
	target := fs.String("target", "", "target merge branch (default: main or master)")
	dryRun := fs.Bool("dry-run", false, "simulate merge and qualify without modifying git state")
	rebase := fs.Bool("rebase", false, "rebase task branch onto target branch before qualification")
	skipFlake := fs.Bool("skip-flake-check", false, "skip second run of test suite")
	jsonFlag := fs.Bool("json", false, "output machine-readable JSON qualification matrix")
	yesFlag := fs.Bool("yes", false, "skip interactive confirmation prompt")

	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return parseExit(err)
	}
	if len(pos) != 1 {
		fprintln(env.Err, "usage: sdd task merge <T-xxx> [--target <branch>] [--dry-run] [--rebase] [--skip-flake-check] [--json] [--yes] [--dir <project>]")
		return 2
	}

	root, err := filepath.Abs(*dir)
	if err != nil {
		fprintf(env.Err, "sdd task merge: %v\n", err)
		return 1
	}

	taskID := strings.ToUpper(strings.TrimSpace(pos[0]))

	targetBranch := *target
	if targetBranch == "" {
		targetBranch = "main"
	}

	// Rebase if requested
	if *rebase {
		loc, err := worktree.FindTask(root, taskID)
		if err != nil {
			fprintf(env.Err, "sdd task merge: locating task %s: %v\n", taskID, err)
			return 1
		}
		branch := fmt.Sprintf("%s/%s-%s", loc.Typology, loc.TaskID, loc.Slug)
		if err := coordinator.RebaseTaskBranch(root, branch, targetBranch); err != nil {
			fprintf(env.Err, "sdd task merge: rebase failed: %v\n", err)
			return 1
		}
		fprintf(env.Out, "Successfully rebased %s onto %s.\n\n", branch, targetBranch)
	}

	opts := coordinator.QualifyOptions{
		TargetBranch:   targetBranch,
		SkipFlakeCheck: *skipFlake,
	}

	rep, err := coordinator.Qualify(root, taskID, opts)
	if err != nil {
		fprintf(env.Err, "sdd task merge: qualification error: %v\n", err)
		return 1
	}

	if *jsonFlag {
		data, err := json.MarshalIndent(rep, "", "  ")
		if err != nil {
			fprintf(env.Err, "sdd task merge: %v\n", err)
			return 1
		}
		fprintln(env.Out, string(data))
		if !rep.ReadyToMerge {
			return 1
		}
		return 0
	}

	fprint(env.Out, coordinator.FormatQualificationReport(rep))

	if *dryRun {
		fprintln(env.Out)
		if rep.ReadyToMerge {
			fprint(env.Out, coordinator.FormatGate2Presentation(rep))
			return 0
		}
		return 1
	}

	if !rep.ReadyToMerge {
		fprintln(env.Err, "\nsdd task merge: task is not ready to merge. Resolve blockers above before merging.")
		return 1
	}

	// Interactive confirmation if not --yes
	if !*yesFlag {
		if !env.Interactive {
			fprintln(env.Err, "sdd task merge: stdin is not interactive; use --yes to proceed non-interactively")
			return 1
		}
		fprintf(env.Out, "\nProceed with merging %s into %s? [y/N]: ", rep.Branch, rep.TargetBranch)
		reader := bufio.NewReader(env.In)
		ans, _ := reader.ReadString('\n')
		ans = strings.TrimSpace(strings.ToLower(ans))
		if ans != "y" && ans != "yes" {
			fprintln(env.Out, "Aborted.")
			return 0
		}
	}

	commitMsg := fmt.Sprintf("feat(%s): merge %s into %s (%s)", rep.Typology, rep.Branch, rep.TargetBranch, rep.TaskID)
	if err := coordinator.ExecuteMerge(root, rep.Branch, rep.TargetBranch, commitMsg); err != nil {
		fprintf(env.Err, "sdd task merge: %v\n", err)
		return 1
	}

	fprintf(env.Out, "\nSuccessfully merged %s into %s.\n", rep.Branch, rep.TargetBranch)
	return 0
}
