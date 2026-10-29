// Package cli implements the sdd command line: dispatch, flag parsing,
// interactive prompts, and the init, check, feature, and prompt commands.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/vikashkumar21/sdd/internal/templates"
	"github.com/vikashkumar21/sdd/internal/version"
)

// IO is everything the commands need from their environment, so tests can
// run them without touching the real terminal, clock, or home directory.
type IO struct {
	In          io.Reader
	Out, Err    io.Writer
	Interactive bool // stdin is a terminal a human is typing at
	Now         func() time.Time
	Getenv      func(string) string
	Home        func() string
}

func (e IO) now() time.Time {
	if e.Now != nil {
		return e.Now()
	}
	return time.Now()
}

func (e IO) getenv(key string) string {
	if e.Getenv == nil {
		return ""
	}
	return e.Getenv(key)
}

func (e IO) home() string {
	if e.Home == nil {
		return ""
	}
	return e.Home()
}

const usageText = `sdd - spec-driven development kit

usage: sdd <command> [flags]

commands:
  init                       generate the spec-driven kit in this project (new or existing)
  check                      verify the kit is consistent: files, agent contracts, references, tables, statuses
  feature new <T-xxx> <slug> create features/T-xxx-<slug>/ from the project's templates
  prompt <agent>             print a handover prompt for an agent (claude, gemini, agents, or any name)
  version                    print the sdd version

Run "sdd <command> -h" for the flags of a command.
Exit codes: 0 success, 1 failure or findings, 2 usage error.
`

// Run executes one sdd command and returns the process exit code.
func Run(args []string, env IO) int {
	if len(args) == 0 {
		fmt.Fprint(env.Err, usageText)
		return 2
	}
	switch args[0] {
	case "init":
		return runInit(args[1:], env)
	case "check":
		return runCheck(args[1:], env)
	case "feature":
		return runFeature(args[1:], env)
	case "prompt":
		return runPrompt(args[1:], env)
	case "version", "--version", "-v":
		fmt.Fprintf(env.Out, "sdd %s (templates %s)\n", version.Version, templates.Version)
		return 0
	case "help", "-h", "--help":
		fmt.Fprint(env.Out, usageText)
		return 0
	default:
		fmt.Fprintf(env.Err, "sdd: unknown command %q\n\n%s", args[0], usageText)
		return 2
	}
}

func newFlagSet(name string, env IO) *flag.FlagSet {
	fs := flag.NewFlagSet("sdd "+name, flag.ContinueOnError)
	fs.SetOutput(env.Err)
	return fs
}

// parseInterspersed parses flags that may appear before, between, or after
// positional arguments and returns the positional arguments.
func parseInterspersed(fs *flag.FlagSet, args []string) ([]string, error) {
	var pos []string
	for len(args) > 0 {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		args = fs.Args()
		if len(args) == 0 {
			break
		}
		pos = append(pos, args[0])
		args = args[1:]
	}
	return pos, nil
}

// parseExit maps a flag-parsing error to an exit code (help is success).
func parseExit(err error) int {
	if errors.Is(err, flag.ErrHelp) {
		return 0
	}
	return 2
}

// stringList is a repeatable string flag.
type stringList []string

func (s *stringList) String() string     { return strings.Join(*s, "; ") }
func (s *stringList) Set(v string) error { *s = append(*s, v); return nil }

func splitList(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ' ' || r == ';' })
}
