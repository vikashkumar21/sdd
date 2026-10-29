// Command sdd generates and checks a spec-driven development kit for a
// project: agent contracts, living documents, and per-task templates.
package main

import (
	"os"
	"time"

	"github.com/vikashkumar21/sdd/internal/cli"
)

func main() {
	env := cli.IO{
		In:          os.Stdin,
		Out:         os.Stdout,
		Err:         os.Stderr,
		Interactive: isTerminal(os.Stdin),
		Now:         time.Now,
		Getenv:      os.Getenv,
		Home: func() string {
			h, err := os.UserHomeDir()
			if err != nil {
				return ""
			}
			return h
		},
	}
	os.Exit(cli.Run(os.Args[1:], env))
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
