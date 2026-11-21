package cli

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"time"

	"github.com/vikashkumar21/sdd/internal/audit"
)

func runTaskAudit(args []string, env IO) int {
	fs := newFlagSet("task audit", env)
	dir := fs.String("dir", ".", "project root")
	jsonFlag := fs.Bool("json", false, "output machine-readable audit report as JSON")
	verbose := fs.Bool("verbose", false, "include full command stdout and stderr outputs")
	timeout := fs.Duration("timeout", 2*time.Minute, "maximum execution duration per command")

	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return parseExit(err)
	}

	if len(pos) != 1 {
		fprintln(env.Err, "usage: sdd task audit <T-xxx> [--dir <project>] [--json] [--verbose] [--timeout <duration>]")
		return 2
	}

	taskID := strings.ToUpper(strings.TrimSpace(pos[0]))
	if !taskIDPattern.MatchString(taskID) {
		fprintf(env.Err, "sdd task audit: invalid task ID format %q (expected format like T-001)\n", pos[0])
		return 1
	}

	root, err := filepath.Abs(*dir)
	if err != nil {
		fprintf(env.Err, "sdd task audit: %v\n", err)
		return 1
	}

	opts := audit.AuditOpts{
		Verbose: *verbose,
		Timeout: *timeout,
	}

	report, err := audit.RunAudit(context.Background(), root, taskID, opts)
	if err != nil {
		if *jsonFlag {
			errReport := map[string]any{
				"task_id": taskID,
				"passed":  false,
				"error":   err.Error(),
			}
			enc := json.NewEncoder(env.Out)
			enc.SetIndent("", "  ")
			_ = enc.Encode(errReport)
		} else {
			fprintf(env.Err, "sdd task audit: %v\n", err)
		}
		return 1
	}

	if *jsonFlag {
		enc := json.NewEncoder(env.Out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fprintf(env.Err, "sdd task audit: json encode: %v\n", err)
			return 1
		}
	} else {
		printAuditConsoleReport(env, report, *verbose)
	}

	if !report.Passed {
		return 1
	}
	return 0
}

func printAuditConsoleReport(env IO, report *audit.AuditReport, verbose bool) {
	fprintf(env.Out, "\nOutside Validation Audit for %s\n", report.TaskID)
	fprintf(env.Out, "%s\n", strings.Repeat("=", 60))

	passedCount := 0
	failedCount := 0

	for _, cmd := range report.Commands {
		if cmd.Passed {
			passedCount++
			fprintf(env.Out, "  [PASS] %s (%s)\n", cmd.Command, cmd.Duration.Round(time.Millisecond))
		} else {
			failedCount++
			if cmd.Error != "" {
				fprintf(env.Out, "  [FAIL] %s (%s)\n", cmd.Command, cmd.Error)
			} else {
				fprintf(env.Out, "  [FAIL] %s (exit code %d, %s)\n", cmd.Command, cmd.ExitCode, cmd.Duration.Round(time.Millisecond))
			}
		}

		if verbose && cmd.Output != "" {
			fprintf(env.Out, "         Output:\n")
			for _, line := range strings.Split(strings.TrimSpace(cmd.Output), "\n") {
				fprintf(env.Out, "           %s\n", line)
			}
		}
	}

	fprintf(env.Out, "%s\n", strings.Repeat("=", 60))
	if report.Passed {
		fprintf(env.Out, "Result: PASS (%d/%d verification commands passed in %s)\n\n",
			passedCount, len(report.Commands), report.TotalTime.Round(time.Millisecond))
	} else {
		fprintf(env.Out, "Result: FAIL (%d passed, %d failed out of %d commands in %s)\n\n",
			passedCount, failedCount, len(report.Commands), report.TotalTime.Round(time.Millisecond))
	}
}
