package cli

import (
	"encoding/json"
	"fmt"
	osexec "os/exec"
	"path/filepath"
	"strings"

	"github.com/vikashkumar21/sdd/internal/hook"
)

func runHook(args []string, env IO) int {
	if len(args) == 0 {
		fprintln(env.Err, "usage: sdd hook <command> [flags]\n\ncommands:\n  install      install pre-commit and pre-merge-commit hooks into .git/hooks/\n  status       check status of git commit hooks (--json for structured output)\n  uninstall    safely remove sdd git hooks and restore original hooks\n  check-gate1  verify Gate 1 plan approval discipline for staged files")
		return 2
	}
	if args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		fprintln(env.Out, "usage: sdd hook <command> [flags]\n\ncommands:\n  install      install pre-commit and pre-merge-commit hooks into .git/hooks/\n  status       check status of git commit hooks (--json for structured output)\n  uninstall    safely remove sdd git hooks and restore original hooks\n  check-gate1  verify Gate 1 plan approval discipline for staged files")
		return 0
	}
	switch args[0] {
	case "install":
		return runHookInstall(args[1:], env)
	case "status":
		return runHookStatus(args[1:], env)
	case "uninstall":
		return runHookUninstall(args[1:], env)
	case "check-gate1":
		return runHookCheckGate1(args[1:], env)
	default:
		fprintf(env.Err, "sdd hook: unknown command %q\n\nusage: sdd hook <install|status|uninstall|check-gate1> [flags]\n", args[0])
		return 2
	}
}

func runHookInstall(args []string, env IO) int {
	fs := newFlagSet("hook install", env)
	dir := fs.String("dir", ".", "project root")
	force := fs.Bool("force", false, "force overwrite of hooks")

	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return parseExit(err)
	}
	if len(pos) > 0 {
		fprintf(env.Err, "sdd hook install: unexpected argument %q\n", pos[0])
		return 2
	}

	root, err := filepath.Abs(*dir)
	if err != nil {
		fprintf(env.Err, "sdd hook install: resolve dir: %v\n", err)
		return 1
	}

	rep, err := hook.Install(root, hook.InstallOptions{Force: *force})
	if err != nil {
		fprintf(env.Err, "sdd hook install: %v\n", err)
		return 1
	}

	fprintln(env.Out, "Installed sdd git hooks in "+rep.HooksDir+":")
	fprintln(env.Out, "  pre-commit       (blocks unauthorized commits to main, checks gate 1 discipline)")
	fprintln(env.Out, "  pre-merge-commit (blocks unauthorized direct merges to main)")
	if rep.Chained {
		for _, orig := range rep.OriginalHooks {
			fprintln(env.Out, "  chained original hook: "+orig)
		}
	}
	return 0
}

func runHookStatus(args []string, env IO) int {
	fs := newFlagSet("hook status", env)
	dir := fs.String("dir", ".", "project root")
	jsonOut := fs.Bool("json", false, "output status as structured JSON")

	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return parseExit(err)
	}
	if len(pos) > 0 {
		fprintf(env.Err, "sdd hook status: unexpected argument %q\n", pos[0])
		return 2
	}

	root, err := filepath.Abs(*dir)
	if err != nil {
		fprintf(env.Err, "sdd hook status: resolve dir: %v\n", err)
		return 1
	}

	rep, err := hook.Status(root)
	if err != nil {
		fprintf(env.Err, "sdd hook status: %v\n", err)
		return 1
	}

	if *jsonOut {
		enc := json.NewEncoder(env.Out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(rep); err != nil {
			fprintf(env.Err, "sdd hook status: encode json: %v\n", err)
			return 1
		}
		return 0
	}

	fprintln(env.Out, "Git Hooks Status:")
	fprintln(env.Out, "  Hooks Directory:  "+rep.HooksDir)
	if rep.Installed {
		fprintln(env.Out, "  Installed:        Yes (sdd managed)")
		fprintln(env.Out, "  Pre-commit:       Present (executable: "+fmt.Sprint(rep.Executable)+")")
		fprintln(env.Out, "  Pre-merge-commit: Present")
	} else {
		fprintln(env.Out, "  Installed:        No (run 'sdd hook install' to enable)")
	}
	if rep.Chained {
		fprintln(env.Out, "  Chained Hooks:    Yes")
		for _, o := range rep.OriginalHooks {
			fprintln(env.Out, "    - "+o)
		}
	}
	return 0
}

func runHookUninstall(args []string, env IO) int {
	fs := newFlagSet("hook uninstall", env)
	dir := fs.String("dir", ".", "project root")
	yes := fs.Bool("yes", false, "skip interactive confirmation")
	force := fs.Bool("force", false, "force removal of hooks even if modified or foreign")

	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return parseExit(err)
	}
	if len(pos) > 0 {
		fprintf(env.Err, "sdd hook uninstall: unexpected argument %q\n", pos[0])
		return 2
	}

	root, err := filepath.Abs(*dir)
	if err != nil {
		fprintf(env.Err, "sdd hook uninstall: resolve dir: %v\n", err)
		return 1
	}

	if !*yes && env.Interactive {
		fprintf(env.Out, "Remove sdd git hooks from %s? (y/N): ", root)
		var resp string
		if _, err := fmt.Fscanln(env.In, &resp); err != nil || (strings.ToLower(resp) != "y" && strings.ToLower(resp) != "yes") {
			fprintln(env.Out, "Uninstallation aborted.")
			return 0
		}
	}

	rep, err := hook.Uninstall(root, hook.UninstallOptions{Force: *force})
	if err != nil {
		fprintf(env.Err, "sdd hook uninstall: %v\n", err)
		return 1
	}

	fprintln(env.Out, "Removed sdd git hooks:")
	for _, r := range rep.RemovedHooks {
		fprintln(env.Out, "  removed: "+r)
	}
	for _, res := range rep.RestoredHooks {
		fprintln(env.Out, "  restored original: "+res)
	}
	return 0
}

func runHookCheckGate1(args []string, env IO) int {
	fs := newFlagSet("hook check-gate1", env)
	dir := fs.String("dir", ".", "project root")
	branch := fs.String("branch", "", "current git branch")

	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return parseExit(err)
	}
	if len(pos) > 0 {
		fprintf(env.Err, "sdd hook check-gate1: unexpected argument %q\n", pos[0])
		return 2
	}

	root, err := filepath.Abs(*dir)
	if err != nil {
		fprintf(env.Err, "sdd hook check-gate1: resolve dir: %v\n", err)
		return 1
	}

	currentBranch := *branch
	if currentBranch == "" {
		// Resolve branch from git
		cmd := osexec.Command("git", "-C", root, "symbolic-ref", "--short", "HEAD")
		out, err := cmd.Output()
		if err == nil {
			currentBranch = strings.TrimSpace(string(out))
		}
	}

	rep, err := hook.CheckGate1(root, currentBranch)
	if err != nil {
		fprintf(env.Err, "sdd hook check-gate1: %v\n", err)
		return 1
	}

	if !rep.Compliant {
		fprintln(env.Err, rep.Reason)
		return 1
	}

	return 0
}
