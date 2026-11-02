package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/vikashkumar21/sdd/internal/context"
	"github.com/vikashkumar21/sdd/internal/doc"
	"github.com/vikashkumar21/sdd/internal/jira"
)

type ingestOpts struct {
	dir   string
	text  string
	name  string
	task  string
	jira  bool
	pdf   bool
	files stringList
}

func runIngest(args []string, env IO) int {
	fs := newFlagSet("ingest", env)
	var o ingestOpts
	fs.StringVar(&o.dir, "dir", ".", "project root")
	fs.StringVar(&o.text, "text", "", "raw text string to stage as context")
	fs.StringVar(&o.name, "name", "", "label or slug name for the staged file")
	fs.StringVar(&o.task, "task", "", "task ID to associate context with (e.g. T-001)")
	fs.BoolVar(&o.jira, "jira", false, "validate and stage context as a Jira issue")
	fs.BoolVar(&o.pdf, "pdf", false, "validate and stage context as a PDF document")
	fs.Var(&o.files, "file", "path to file to stage as context (repeatable)")

	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return parseExit(err)
	}
	if len(pos) > 0 {
		fprintf(env.Err, "sdd ingest: unexpected argument %q\n", pos[0])
		return 2
	}

	root, err := filepath.Abs(o.dir)
	if err != nil {
		fprintf(env.Err, "sdd ingest: %v\n", err)
		return 1
	}

	var items []context.Item

	// 1. Files from flags
	for _, f := range o.files {
		data, err := os.ReadFile(f)
		if err != nil {
			fprintf(env.Err, "sdd ingest: read %s: %v\n", f, err)
			return 1
		}
		items = append(items, context.Item{
			Source: "file",
			Origin: filepath.Base(f),
			Name:   o.name,
			Task:   o.task,
			Data:   data,
		})
	}

	// 2. Raw text string
	if o.text != "" {
		items = append(items, context.Item{
			Source: "raw-text",
			Origin: "cli-text",
			Name:   o.name,
			Task:   o.task,
			Data:   []byte(o.text),
		})
	}

	// 3. Stdin if no files and no text
	if len(items) == 0 {
		if env.Interactive {
			fprintln(env.Err, "sdd ingest: nothing to ingest (specify --file, --text, or pipe into stdin)")
			return 2
		}
		data, err := io.ReadAll(io.LimitReader(env.In, context.MaxInputBytes+1))
		if err != nil {
			fprintf(env.Err, "sdd ingest: read stdin: %v\n", err)
			return 1
		}
		if len(strings.TrimSpace(string(data))) == 0 {
			fprintln(env.Err, "sdd ingest: stdin is empty")
			return 2
		}
		items = append(items, context.Item{
			Source: "stdin",
			Origin: "stdin",
			Name:   o.name,
			Task:   o.task,
			Data:   data,
		})
	}

	now := env.now()
	stagedCount := 0
	for _, it := range items {
		if o.jira {
			iss, err := jira.Parse(it.Data)
			if err != nil {
				fprintf(env.Err, "sdd ingest: invalid jira issue: %v\n", err)
				return 1
			}
			it.Format = "jira"
			if it.Name == "" && iss.Key != "" {
				it.Name = iss.Key
			}
		}

		if o.pdf || doc.IsPDF(it.Origin, it.Data) {
			if _, err := doc.ValidatePDF(it.Data); err != nil {
				fprintf(env.Err, "sdd ingest: invalid pdf document: %v\n", err)
				return 1
			}
			it.Format = "pdf"
		}

		entry, err := context.Stage(root, it, now)
		if err != nil {
			fprintf(env.Err, "sdd ingest: %v\n", err)
			return 1
		}
		fprintf(env.Out, "Staged %s (%s, %d bytes) -> %s\n", entry.ID, entry.Format, entry.Bytes, entry.Path)
		stagedCount++
	}

	fprintf(env.Out, "\nSuccessfully staged %d context item(s) in %s\n", stagedCount, context.ManifestPath)
	return 0
}
