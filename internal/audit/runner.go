package audit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var codeSpanRe = regexp.MustCompile("`([^`]+)`")

// AuditOpts configures the outside verification audit runner.
type AuditOpts struct {
	Verbose bool          `json:"verbose"`
	Timeout time.Duration `json:"timeout"`
}

// CommandAudit records the execution metrics of a single verification command.
type CommandAudit struct {
	Command  string        `json:"command"`
	Passed   bool          `json:"passed"`
	ExitCode int           `json:"exit_code"`
	Duration time.Duration `json:"duration"`
	Output   string        `json:"output,omitempty"`
	Error    string        `json:"error,omitempty"`
}

// AuditReport aggregates the overall outside verification audit outcome.
type AuditReport struct {
	TaskID    string         `json:"task_id"`
	Passed    bool           `json:"passed"`
	Commands  []CommandAudit `json:"commands"`
	TotalTime time.Duration  `json:"total_time"`
}

var permittedPrefixes = []string{
	"go test",
	"go vet",
	"make",
	"sdd check",
	"./bin/sdd check",
	"bin/sdd check",
	"sdd status",
	"./bin/sdd status",
	"bin/sdd status",
	"git grep",
	"git diff",
	"git status",
}

// IsCommandSafe validates whether a command adheres to the verification safety whitelist.
func IsCommandSafe(cmdStr string) (bool, string) {
	trimmed := strings.TrimSpace(cmdStr)
	if trimmed == "" {
		return false, "empty command"
	}

	// Reject shell redirection, backgrounding, pipes, semicolons, and backticks (SEC-002)
	for _, forbidden := range []string{">", "<", "|", ";", "$", "(", ")", "&", "`"} {
		if strings.Contains(trimmed, forbidden) {
			return false, fmt.Sprintf("contains unapproved shell character %q", forbidden)
		}
	}

	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return false, "empty command"
	}

	// Reject dangerous flags known to execute external binaries or write arbitrary paths (SEC-003)
	for _, arg := range parts[1:] {
		lower := strings.ToLower(arg)
		if lower == "-exec" || strings.HasPrefix(lower, "-exec=") ||
			lower == "--output" || strings.HasPrefix(lower, "--output=") ||
			lower == "--ext-diff" || strings.HasPrefix(lower, "--ext-diff=") ||
			lower == "--eval" || strings.HasPrefix(lower, "--eval=") {
			return false, fmt.Sprintf("flag %q is not permitted by verification safety whitelist", arg)
		}

		if parts[0] == "make" {
			// Disallow make flags that change directories or load external Makefiles
			if lower == "-f" || strings.HasPrefix(lower, "-f=") ||
				lower == "-c" || strings.HasPrefix(lower, "-c=") ||
				lower == "--file" || strings.HasPrefix(lower, "--file=") ||
				lower == "--directory" || strings.HasPrefix(lower, "--directory=") {
				return false, fmt.Sprintf("make flag %q is not permitted by verification safety whitelist", arg)
			}
		}
	}

	for _, prefix := range permittedPrefixes {
		if trimmed == prefix || strings.HasPrefix(trimmed, prefix+" ") {
			return true, ""
		}
	}

	return false, fmt.Sprintf("command %q not permitted by verification safety whitelist", trimmed)
}

func splitTableCells(line string) []string {
	var cells []string
	var current strings.Builder
	escaped := false
	inCode := false

	for i := 0; i < len(line); i++ {
		ch := line[i]
		if ch == '\\' && !escaped {
			escaped = true
			continue
		}
		if ch == '`' && !escaped {
			inCode = !inCode
		}
		if ch == '|' && !escaped && !inCode {
			cells = append(cells, current.String())
			current.Reset()
		} else {
			if escaped && ch != '|' {
				current.WriteByte('\\')
			}
			current.WriteByte(ch)
		}
		escaped = false
	}
	cells = append(cells, current.String())
	return cells
}

// ExtractValidationCommands extracts executable verification commands from validation.md.
func ExtractValidationCommands(validationContent string) []string {
	var commands []string
	seen := make(map[string]bool)

	lines := strings.Split(validationContent, "\n")
	currentSection := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## 1.") {
			currentSection = 1
			continue
		} else if strings.HasPrefix(trimmed, "## 2.") {
			currentSection = 2
			continue
		} else if strings.HasPrefix(trimmed, "## ") {
			currentSection = 0
			continue
		}

		if currentSection == 0 || !strings.HasPrefix(trimmed, "|") || strings.Contains(trimmed, "---") {
			continue
		}

		parts := splitTableCells(trimmed)
		var targetCell string

		if currentSection == 1 && len(parts) >= 5 {
			// Section 1: | Req | Method | Command / test | Expected | Actual | Result |
			headerCandidate := strings.ToLower(strings.TrimSpace(parts[3]))
			if strings.Contains(headerCandidate, "command") {
				continue // header row
			}
			targetCell = parts[3]
		} else if currentSection == 2 && len(parts) >= 3 {
			// Section 2: | Command | Result |
			headerCandidate := strings.ToLower(strings.TrimSpace(parts[1]))
			if strings.Contains(headerCandidate, "command") {
				continue // header row
			}
			targetCell = parts[1]
		}

		if targetCell == "" {
			continue
		}

		matches := codeSpanRe.FindAllStringSubmatch(targetCell, -1)
		var rawCmds []string
		if len(matches) > 0 {
			for _, m := range matches {
				if len(m) >= 2 {
					rawCmds = append(rawCmds, m[1])
				}
			}
		} else {
			cleanCell := strings.TrimSpace(targetCell)
			if cleanCell != "" && !strings.EqualFold(cleanCell, "none") && !strings.EqualFold(cleanCell, "n/a") {
				rawCmds = append(rawCmds, cleanCell)
			}
		}

		for _, rawCmd := range rawCmds {
			// Split compound commands chained with &&
			subCmds := strings.Split(rawCmd, "&&")
			for _, sub := range subCmds {
				norm := strings.TrimSpace(sub)
				if norm == "" {
					continue
				}
				if !seen[norm] {
					seen[norm] = true
					commands = append(commands, norm)
				}
			}
		}
	}

	return commands
}

// RunAudit parses validation.md for taskID and executes verification commands mechanically.
func RunAudit(ctx context.Context, root, taskID string, opts AuditOpts) (*AuditReport, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	normTaskID := strings.ToUpper(strings.TrimSpace(taskID))
	if !taskIDPattern.MatchString(normTaskID) {
		return nil, fmt.Errorf("invalid task ID format %q (expected format like T-001)", taskID)
	}

	taskRelDir, _, err := findTaskDirectory(root, normTaskID)
	if err != nil {
		return nil, err
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 2 * time.Minute
	}

	taskDirAbs := filepath.Join(root, filepath.FromSlash(taskRelDir))
	valContent := readDoc(taskDirAbs, "validation.md")
	if strings.TrimSpace(valContent) == "" {
		return nil, fmt.Errorf("validation.md not found or empty in %s", taskRelDir)
	}

	commands := ExtractValidationCommands(valContent)
	if len(commands) == 0 {
		return nil, fmt.Errorf("no executable verification commands found in %s/validation.md", taskRelDir)
	}

	report := &AuditReport{
		TaskID:   normTaskID,
		Passed:   true,
		Commands: make([]CommandAudit, 0, len(commands)),
	}

	startTotal := time.Now()

	for _, cmdStr := range commands {
		cmdAudit := CommandAudit{
			Command: cmdStr,
		}

		safe, reason := IsCommandSafe(cmdStr)
		if !safe {
			cmdAudit.Passed = false
			cmdAudit.ExitCode = 126
			cmdAudit.Error = fmt.Sprintf("safety violation: %s", reason)
			report.Commands = append(report.Commands, cmdAudit)
			report.Passed = false
			break
		}

		parts := strings.Fields(cmdStr)
		if len(parts) == 0 {
			continue
		}

		cmdCtx, cancel := context.WithTimeout(ctx, timeout)
		cmd := exec.CommandContext(cmdCtx, parts[0], parts[1:]...)
		cmd.Dir = root

		var outBuf bytes.Buffer
		cmd.Stdout = &outBuf
		cmd.Stderr = &outBuf

		cmdStart := time.Now()
		runErr := cmd.Run()
		cmdAudit.Duration = time.Since(cmdStart)
		cancel()

		cmdAudit.Output = outBuf.String()
		if runErr != nil {
			cmdAudit.Passed = false
			var exitErr *exec.ExitError
			if errors.As(runErr, &exitErr) {
				cmdAudit.ExitCode = exitErr.ExitCode()
			} else {
				cmdAudit.ExitCode = 1
			}
			cmdAudit.Error = runErr.Error()
			report.Passed = false
		} else {
			cmdAudit.Passed = true
			cmdAudit.ExitCode = 0
		}

		if !opts.Verbose && len(cmdAudit.Output) > 1024 {
			// In non-verbose mode truncate output to keep report clean
			cmdAudit.Output = cmdAudit.Output[:1024] + "\n... (truncated)"
		}

		report.Commands = append(report.Commands, cmdAudit)

		if !cmdAudit.Passed {
			// Short-circuit on failure
			break
		}
	}

	report.TotalTime = time.Since(startTotal)
	return report, nil
}
