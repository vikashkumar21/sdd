package cli

import (
	"bufio"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vikashkumar21/sdd/internal/install"
)

func runInstall(args []string, env IO) int {
	fs := newFlagSet("install", env)
	dryRun := fs.Bool("dry-run", false, "preview installation without writing files")
	force := fs.Bool("force", false, "force reinstall even if already at target location")
	yes := fs.Bool("yes", false, "skip confirmation prompt")
	fs.BoolVar(yes, "y", false, "alias for --yes")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		fprintf(env.Err, "sdd install: %v\n", err)
		return 2
	}

	execPath := env.getenv("SDD_EXEC_PATH")
	opts := install.InstallOptions{
		ExecutablePath: execPath,
		Home:           env.Home,
		Getenv:         env.Getenv,
		Force:          *force,
		DryRun:         *dryRun,
	}

	res, err := install.Install(opts)
	if err != nil {
		fprintf(env.Err, "sdd install: %v\n", err)
		return 1
	}

	if res.AlreadyInstalled {
		fprintf(env.Out, "sdd is already installed at %s\n", res.BinaryPath)
		return 0
	}

	if *dryRun {
		fprintf(env.Out, "Dry run: would install sdd to %s\n", res.BinaryPath)
		if res.PathUpdated {
			if len(res.ProfilePaths) > 0 {
				fprintf(env.Out, "Dry run: would add %s to PATH in:\n", res.TargetDir)
				for _, p := range res.ProfilePaths {
					fprintf(env.Out, "  %s\n", p)
				}
			} else {
				fprintf(env.Out, "Dry run: would add %s to user environment PATH\n", res.TargetDir)
			}
		}
		return 0
	}

	fprintf(env.Out, "✓ Installed sdd to %s\n", res.BinaryPath)
	if res.PathUpdated {
		if len(res.ProfilePaths) > 0 {
			fprintf(env.Out, "✓ Added %s to PATH in:\n", res.TargetDir)
			for _, p := range res.ProfilePaths {
				fprintf(env.Out, "  %s\n", p)
			}
			fprintf(env.Out, "\nTo use sdd now, restart your terminal or run:\n  source %s\n", res.ProfilePaths[0])
		} else {
			fprintf(env.Out, "✓ Added %s to user environment PATH\n", res.TargetDir)
		}
	} else {
		fprintf(env.Out, "✓ %s is already on your PATH\n", res.TargetDir)
	}

	return 0
}

func runUninstall(args []string, env IO) int {
	fs := newFlagSet("uninstall", env)
	dryRun := fs.Bool("dry-run", false, "preview uninstallation without removing files")
	yes := fs.Bool("yes", false, "skip confirmation prompt")
	fs.BoolVar(yes, "y", false, "alias for --yes")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		fprintf(env.Err, "sdd uninstall: %v\n", err)
		return 2
	}

	if !*yes && env.Interactive {
		p := newPrompter(env.In, env.Out)
		if !p.confirm("Are you sure you want to uninstall sdd?") {
			fprintln(env.Out, "Uninstall cancelled.")
			return 0
		}
	}

	opts := install.UninstallOptions{
		Home:   env.Home,
		Getenv: env.Getenv,
		DryRun: *dryRun,
	}

	res, err := install.Uninstall(opts)
	if err != nil {
		fprintf(env.Err, "sdd uninstall: %v\n", err)
		return 1
	}

	if *dryRun {
		fprintf(env.Out, "Dry run: would remove %s\n", res.BinaryPath)
		if len(res.CleanedFiles) > 0 {
			fprintf(env.Out, "Dry run: would clean PATH configuration from:\n")
			for _, p := range res.CleanedFiles {
				fprintf(env.Out, "  %s\n", p)
			}
		}
		return 0
	}

	if res.Removed {
		fprintf(env.Out, "✓ Removed %s\n", res.BinaryPath)
	} else {
		fprintf(env.Out, "sdd binary was not found at %s\n", res.BinaryPath)
	}

	if len(res.CleanedFiles) > 0 {
		fprintf(env.Out, "✓ Cleaned PATH configuration from:\n")
		for _, p := range res.CleanedFiles {
			fprintf(env.Out, "  %s\n", p)
		}
	}

	return 0
}

func maybeInteractiveInstall(env IO) (handled bool, exitCode int) {
	if !env.Interactive {
		return false, 0
	}

	execPath := env.getenv("SDD_EXEC_PATH")
	if execPath == "" {
		ep, err := os.Executable()
		if err != nil {
			return false, 0
		}
		execPath = ep
	}
	execPath = filepath.Clean(execPath)

	goos := runtime.GOOS
	targetBin, targetDir, err := install.ResolveTarget(goos, env.home(), env.Getenv)
	if err != nil {
		return false, 0
	}
	targetBin = filepath.Clean(targetBin)

	// If already running the installed binary, do not prompt
	if execPath == targetBin {
		return false, 0
	}

	fprintf(env.Out, "sdd is not installed on your PATH.\n\n")
	fprintf(env.Out, "Current location: %s\n", execPath)
	fprintf(env.Out, "Target location:  %s\n\n", targetBin)

	p := newPrompter(env.In, env.Out)
	answer := p.ask("Would you like to install sdd now? (y/n)", "y")
	if strings.ToLower(strings.TrimSpace(answer)) != "y" && strings.ToLower(strings.TrimSpace(answer)) != "yes" {
		return false, 0
	}

	opts := install.InstallOptions{
		ExecutablePath: execPath,
		Home:           env.Home,
		Getenv:         env.Getenv,
		GOOS:           goos,
	}

	res, err := install.Install(opts)
	if err != nil {
		fprintf(env.Err, "Installation failed: %v\n", err)
		waitForEnter(env.In, env.Out)
		return true, 1
	}

	fprintf(env.Out, "\n✓ Successfully installed sdd to %s\n", res.BinaryPath)
	if res.PathUpdated && len(res.ProfilePaths) > 0 {
		fprintf(env.Out, "✓ Added %s to PATH in:\n", targetDir)
		for _, prof := range res.ProfilePaths {
			fprintf(env.Out, "  %s\n", prof)
		}
		fprintf(env.Out, "\nTo start using sdd, restart your terminal or run:\n  source %s\n", res.ProfilePaths[0])
	} else if res.PathUpdated {
		fprintf(env.Out, "✓ Added %s to user environment PATH\n", targetDir)
	}

	fprintf(env.Out, "\n")
	waitForEnter(env.In, env.Out)
	return true, 0
}

func waitForEnter(r io.Reader, w io.Writer) {
	if r == nil || w == nil {
		return
	}
	fprintf(w, "Press Enter to exit...")
	scanner := bufio.NewScanner(r)
	if scanner.Scan() {
		return
	}
}
