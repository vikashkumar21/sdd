package cli

import (
	"path/filepath"

	"github.com/vikashkumar21/sdd/internal/passport"
)

func runPassport(args []string, env IO) int {
	fs := newFlagSet("passport", env)
	dir := fs.String("dir", ".", "project root")
	jsonOut := fs.Bool("json", false, "output passport as structured JSON")

	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return parseExit(err)
	}
	if len(pos) > 0 {
		fprintf(env.Err, "sdd passport: unexpected argument %q\n", pos[0])
		return 2
	}

	root, err := filepath.Abs(*dir)
	if err != nil {
		fprintf(env.Err, "sdd passport: %v\n", err)
		return 1
	}

	p, err := passport.Generate(root, env.now().UTC())
	if err != nil {
		fprintf(env.Err, "sdd passport: %v\n", err)
		return 1
	}

	if *jsonOut {
		data, err := p.RenderJSON()
		if err != nil {
			fprintf(env.Err, "sdd passport: %v\n", err)
			return 1
		}
		fprintln(env.Out, data)
		return 0
	}

	fprint(env.Out, p.RenderMarkdown())
	return 0
}
