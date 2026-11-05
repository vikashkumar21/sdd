package cli

import (
	"path/filepath"

	"github.com/vikashkumar21/sdd/internal/status"
)

func runStatus(args []string, env IO) int {
	fs := newFlagSet("status", env)
	dir := fs.String("dir", ".", "project root")
	jsonOut := fs.Bool("json", false, "output status as structured JSON")

	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return parseExit(err)
	}
	if len(pos) > 0 {
		fprintf(env.Err, "sdd status: unexpected argument %q\n", pos[0])
		return 2
	}

	root, err := filepath.Abs(*dir)
	if err != nil {
		fprintf(env.Err, "sdd status: %v\n", err)
		return 1
	}

	ps, err := status.Scan(root)
	if err != nil {
		fprintf(env.Err, "sdd status: %v\n", err)
		return 1
	}

	if *jsonOut {
		data, err := status.RenderJSON(ps)
		if err != nil {
			fprintf(env.Err, "sdd status: %v\n", err)
			return 1
		}
		fprintln(env.Out, string(data))
		return 0
	}

	fprint(env.Out, status.RenderText(ps))
	return 0
}
