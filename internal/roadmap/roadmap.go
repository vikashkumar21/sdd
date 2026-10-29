// Package roadmap parses and edits the task tables in ROADMAP.md.
//
// A task row starts with "| T-nnn |" (optionally "T-nnn.m") and ends with its
// status as the last cell. Cells must not contain a literal pipe.
package roadmap

import (
	"regexp"
	"strings"
)

// Task is one row of a roadmap task table.
type Task struct {
	ID     string
	Title  string
	Status string // as written, including any markdown emphasis
	Line   int    // 1-based line number in the file
}

var rowRe = regexp.MustCompile(`^\|\s*(T-\d{3}(?:\.\d+)?)\s*\|`)

var validStatuses = []string{
	"not started", "specifying", "awaiting plan approval",
	"implementing", "awaiting review", "done", "blocked",
}

// Parse returns every task row found in the text, in file order.
func Parse(text string) []Task {
	var tasks []Task
	for i, line := range strings.Split(text, "\n") {
		m := rowRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		cells := splitRow(line)
		if len(cells) < 3 {
			continue
		}
		tasks = append(tasks, Task{ID: m[1], Title: cells[1], Status: cells[len(cells)-1], Line: i + 1})
	}
	return tasks
}

func splitRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	parts := strings.Split(line, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// Normalize removes markdown emphasis and code ticks from a status cell.
func Normalize(status string) string {
	status = strings.ReplaceAll(status, "**", "")
	status = strings.ReplaceAll(status, "`", "")
	return strings.TrimSpace(status)
}

// Kind returns the canonical lower-case status ("done", "blocked", ...) that
// the cell starts with, or "" when it is not a recognised status.
func Kind(status string) string {
	lower := strings.ToLower(Normalize(status))
	for _, s := range validStatuses {
		if strings.HasPrefix(lower, s) {
			return s
		}
	}
	return ""
}

// ValidStatus reports whether the status starts with one of the allowed
// status values (for example "Blocked (reason)" or "Done (COMMITTED T-000)").
func ValidStatus(status string) bool {
	return Kind(status) != ""
}

// SetStatus replaces the status of the row with the given ID, but only when
// its current normalised status equals from (case-insensitive). It returns the
// new text and whether a change was made.
func SetStatus(text, id, from, to string) (string, bool) {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		m := rowRe.FindStringSubmatch(line)
		if m == nil || m[1] != id {
			continue
		}
		cells := splitRow(line)
		if !strings.EqualFold(Normalize(cells[len(cells)-1]), from) {
			return text, false
		}
		trimmed := strings.TrimRight(line, " \t\r")
		if !strings.HasSuffix(trimmed, "|") {
			return text, false
		}
		body := trimmed[:len(trimmed)-1]
		idx := strings.LastIndex(body, "|")
		if idx < 0 {
			return text, false
		}
		lines[i] = body[:idx+1] + " " + to + " |"
		return strings.Join(lines, "\n"), true
	}
	return text, false
}
