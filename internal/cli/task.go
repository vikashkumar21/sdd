package cli

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/vikashkumar21/sdd/internal/feature"
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
		fprintln(env.Err, "usage: sdd task <command> [flags]\n\ncommands:\n  new <T-xxx> <slug>   create a task folder (--type feature|bugfix|hotfix|spike)")
		return 2
	}
	switch args[0] {
	case "new":
		return runTaskNewWithFlag(args[1:], env)
	default:
		fprintf(env.Err, "sdd task: unknown command %q\n\nusage: sdd task <new> [flags]\n", args[0])
		return 2
	}
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
