// Package state implements deterministic task state persistence and event journaling
// under .sdd/state/<TASK_ID>/ for autonomous agents and external tooling.
package state

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/vikashkumar21/sdd/internal/worktree"
)

var (
	// taskIDRegex enforces the format: one or more letters, hyphen, one or more digits.
	taskIDRegex = regexp.MustCompile(`^[A-Za-z]+-\d+$`)
)

const (
	// StateBaseDir is the relative directory for task states.
	StateBaseDir = ".sdd/state"
	// StateFileName is the filename for task snapshot.
	StateFileName = "state.json"
	// EventsFileName is the filename for append-only event log.
	EventsFileName = "events.jsonl"

	// MaxEventLineBytes is the maximum allowed byte length for a single event line (1MB).
	MaxEventLineBytes = 1024 * 1024
	// MaxStdinEventBytes is the max bytes read from stdin for an event (1MB).
	MaxStdinEventBytes = 1024 * 1024
	// MaxStdinStateBytes is the max bytes read from stdin for state snapshot (5MB).
	MaxStdinStateBytes = 5 * 1024 * 1024
)

// ValidateTaskID validates that taskID matches ^[A-Za-z]+-\d+$, canonicalizes to uppercase,
// and rejects any path traversal characters (/, \, ..).
func ValidateTaskID(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("task ID cannot be empty")
	}
	if strings.ContainsAny(trimmed, "/\\") || strings.Contains(trimmed, "..") {
		return "", fmt.Errorf("invalid task ID %q: path traversal and separators are forbidden", raw)
	}
	if !taskIDRegex.MatchString(trimmed) {
		return "", fmt.Errorf("invalid task ID %q: must match format ^[A-Za-z]+-\\d+$ (e.g. T-003, T-017)", raw)
	}
	return strings.ToUpper(trimmed), nil
}

// TaskStateDir returns the filesystem path to .sdd/state/<TASK_ID>.
func TaskStateDir(root, taskID string) (string, error) {
	normID, err := ValidateTaskID(taskID)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, filepath.FromSlash(StateBaseDir), normID), nil
}

// RelativeStateDir returns the slash-separated relative path to .sdd/state/<TASK_ID>.
func RelativeStateDir(taskID string) (string, error) {
	normID, err := ValidateTaskID(taskID)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s", StateBaseDir, normID), nil
}

// RelativeStateFile returns the slash-separated relative path to .sdd/state/<TASK_ID>/state.json.
func RelativeStateFile(taskID string) (string, error) {
	normID, err := ValidateTaskID(taskID)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s/%s", StateBaseDir, normID, StateFileName), nil
}

// SaveResult describes the JSON response of a save operation.
type SaveResult struct {
	TaskID    string `json:"task_id"`
	Saved     bool   `json:"saved"`
	Path      string `json:"path"`
	UpdatedAt string `json:"updated_at"`
}

// AppendResult describes the JSON response of an append-event operation.
type AppendResult struct {
	TaskID     string `json:"task_id"`
	Appended   bool   `json:"appended"`
	EventCount int    `json:"event_count"`
}

// PurgeResult describes the JSON response of a purge operation.
type PurgeResult struct {
	TaskID string `json:"task_id"`
	Purged bool   `json:"purged"`
	Path   string `json:"path"`
}

// CombinedState describes the complete state snapshot, event log, and worktree status.
type CombinedState struct {
	TaskID         string         `json:"task_id"`
	Found          bool           `json:"found"`
	State          map[string]any `json:"state,omitempty"`
	Events         []any          `json:"events,omitempty"`
	WorktreeActive bool           `json:"worktree_active"`
	WorktreePath   string         `json:"worktree_path,omitempty"`
}

// SaveState atomically writes or updates the task's state.json snapshot.
// The payload is validated as valid JSON, written to a unique temp file in the same
// directory, synced to disk, and renamed into place.
func SaveState(root, taskID string, payload []byte, now time.Time) (SaveResult, error) {
	normID, err := ValidateTaskID(taskID)
	if err != nil {
		return SaveResult{}, err
	}

	payload = bytes.TrimSpace(payload)
	if len(payload) == 0 {
		return SaveResult{}, errors.New("state payload cannot be empty")
	}

	// Validate JSON
	var parsed map[string]any
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return SaveResult{}, fmt.Errorf("invalid state JSON: %w", err)
	}

	dir, err := TaskStateDir(root, normID)
	if err != nil {
		return SaveResult{}, err
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return SaveResult{}, fmt.Errorf("create state directory %s: %w", dir, err)
	}

	// Pretty-print formatted JSON with trailing newline
	formatted, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		formatted = payload
	}
	formatted = append(formatted, '\n')

	targetFile := filepath.Join(dir, StateFileName)
	tempFile := filepath.Join(dir, fmt.Sprintf("%s.tmp.%d.%d", StateFileName, os.Getpid(), time.Now().UnixNano()))

	f, err := os.OpenFile(tempFile, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
	if err != nil {
		return SaveResult{}, fmt.Errorf("create temp state file: %w", err)
	}

	writeErr := func() error {
		if _, err := f.Write(formatted); err != nil {
			return err
		}
		return f.Sync()
	}()

	closeErr := f.Close()
	if writeErr != nil {
		_ = os.Remove(tempFile)
		return SaveResult{}, fmt.Errorf("write temp state file: %w", writeErr)
	}
	if closeErr != nil {
		_ = os.Remove(tempFile)
		return SaveResult{}, fmt.Errorf("close temp state file: %w", closeErr)
	}

	if err := os.Rename(tempFile, targetFile); err != nil {
		_ = os.Remove(tempFile)
		return SaveResult{}, fmt.Errorf("atomic rename to %s: %w", targetFile, err)
	}

	if now.IsZero() {
		now = time.Now()
	}

	relPath, _ := RelativeStateFile(normID)
	return SaveResult{
		TaskID:    normID,
		Saved:     true,
		Path:      relPath,
		UpdatedAt: now.UTC().Format(time.RFC3339),
	}, nil
}

// AppendEvent appends a single immutable lifecycle event to .sdd/state/<TASK_ID>/events.jsonl.
// The event payload is validated as valid JSON, compacted to a single line, and appended
// with crash-resilient file sync.
func AppendEvent(root, taskID string, payload []byte) (AppendResult, error) {
	normID, err := ValidateTaskID(taskID)
	if err != nil {
		return AppendResult{}, err
	}

	payload = bytes.TrimSpace(payload)
	if len(payload) == 0 {
		return AppendResult{}, errors.New("event payload cannot be empty")
	}

	// Validate JSON
	var dummy any
	if err := json.Unmarshal(payload, &dummy); err != nil {
		return AppendResult{}, fmt.Errorf("invalid event JSON: %w", err)
	}

	// Compact JSON onto a single line without CRLF or extraneous whitespace
	var buf bytes.Buffer
	if err := json.Compact(&buf, payload); err != nil {
		return AppendResult{}, fmt.Errorf("compact event JSON: %w", err)
	}
	buf.WriteByte('\n')

	dir, err := TaskStateDir(root, normID)
	if err != nil {
		return AppendResult{}, err
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return AppendResult{}, fmt.Errorf("create state directory %s: %w", dir, err)
	}

	eventsFile := filepath.Join(dir, EventsFileName)
	f, err := os.OpenFile(eventsFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return AppendResult{}, fmt.Errorf("open events file: %w", err)
	}

	writeErr := func() error {
		if _, err := f.Write(buf.Bytes()); err != nil {
			return err
		}
		return f.Sync()
	}()

	closeErr := f.Close()
	if writeErr != nil {
		return AppendResult{}, fmt.Errorf("write event to %s: %w", eventsFile, writeErr)
	}
	if closeErr != nil {
		return AppendResult{}, fmt.Errorf("close events file %s: %w", eventsFile, closeErr)
	}

	count, _ := CountEvents(root, normID)
	return AppendResult{
		TaskID:     normID,
		Appended:   true,
		EventCount: count,
	}, nil
}

// CountEvents returns the number of valid event lines in events.jsonl.
func CountEvents(root, taskID string) (int, error) {
	normID, err := ValidateTaskID(taskID)
	if err != nil {
		return 0, err
	}

	dir, err := TaskStateDir(root, normID)
	if err != nil {
		return 0, err
	}

	eventsFile := filepath.Join(dir, EventsFileName)
	f, err := os.Open(eventsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), MaxEventLineBytes)
	count := 0
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) > 0 {
			count++
		}
	}
	if err := scanner.Err(); err != nil {
		return count, fmt.Errorf("read events file: %w", err)
	}
	return count, nil
}

// ReadEvents reads and parses all event lines from events.jsonl.
func ReadEvents(root, taskID string) ([]any, error) {
	normID, err := ValidateTaskID(taskID)
	if err != nil {
		return nil, err
	}

	dir, err := TaskStateDir(root, normID)
	if err != nil {
		return nil, err
	}

	eventsFile := filepath.Join(dir, EventsFileName)
	f, err := os.Open(eventsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open events file: %w", err)
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), MaxEventLineBytes)
	var events []any
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var evt any
		if err := json.Unmarshal(line, &evt); err != nil {
			return nil, fmt.Errorf("corrupt event line %d in %s: %w", lineNum, eventsFile, err)
		}
		events = append(events, evt)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan events from %s: %w", eventsFile, err)
	}
	return events, nil
}

// ReadState reads and unmarshals state.json for the specified task.
// If state.json does not exist, returns (nil, false, nil).
func ReadState(root, taskID string) (map[string]any, bool, error) {
	normID, err := ValidateTaskID(taskID)
	if err != nil {
		return nil, false, err
	}

	dir, err := TaskStateDir(root, normID)
	if err != nil {
		return nil, false, err
	}

	stateFile := filepath.Join(dir, StateFileName)
	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("read state file: %w", err)
	}

	var state map[string]any
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, true, fmt.Errorf("corrupt state file %s: %w", stateFile, err)
	}
	return state, true, nil
}

// WorktreeStatus holds worktree active status and path.
type WorktreeStatus struct {
	Active bool
	Path   string
}

// WorktreeChecker abstracts worktree detection for testing and runtime.
type WorktreeChecker interface {
	CheckWorktree(taskID string) WorktreeStatus
}

// DefaultWorktreeChecker checks active worktrees using git worktree list and .sdd/worktrees.json.
type DefaultWorktreeChecker struct {
	Root string
}

// CheckWorktree inspects worktrees on disk, in worktree registry, and via git worktree list.
func (c *DefaultWorktreeChecker) CheckWorktree(taskID string) WorktreeStatus {
	normID, err := ValidateTaskID(taskID)
	if err != nil {
		return WorktreeStatus{}
	}

	// 1. Check registry (.sdd/worktrees.json)
	if entries, err := worktree.List(c.Root); err == nil {
		for _, e := range entries {
			if strings.ToUpper(e.TaskID) == normID {
				abs := filepath.Join(c.Root, filepath.FromSlash(e.Path))
				if _, statErr := os.Stat(abs); statErr == nil {
					return WorktreeStatus{Active: true, Path: abs}
				}
			}
		}
	}

	// 2. Check physical directory .sdd/worktrees/<TASK_ID>-*
	baseDir := filepath.Join(c.Root, worktree.WorktreesBaseDir)
	if dirEntries, err := os.ReadDir(baseDir); err == nil {
		for _, e := range dirEntries {
			if e.IsDir() && strings.HasPrefix(strings.ToUpper(e.Name()), normID+"-") {
				abs := filepath.Join(baseDir, e.Name())
				return WorktreeStatus{Active: true, Path: abs}
			}
		}
	}

	// 3. Fallback to git worktree list
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = c.Root
	out, err := cmd.Output()
	if err == nil {
		scanner := bufio.NewScanner(bytes.NewReader(out))
		var currentPath string
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "worktree ") {
				currentPath = strings.TrimPrefix(line, "worktree ")
			} else if strings.HasPrefix(line, "branch ") {
				branch := strings.TrimPrefix(line, "branch ")
				upperBranch := strings.ToUpper(branch)
				if strings.Contains(upperBranch, "/"+normID+"-") || strings.Contains(upperBranch, "/"+normID+"/") || strings.HasSuffix(upperBranch, "/"+normID) {
					return WorktreeStatus{Active: true, Path: currentPath}
				}
			}
		}
	}

	return WorktreeStatus{}
}

// Show reads state snapshot, event log, and cross-checks active git worktrees.
// If checker is nil, DefaultWorktreeChecker is used.
func Show(root, taskID string, checker WorktreeChecker) (CombinedState, error) {
	normID, err := ValidateTaskID(taskID)
	if err != nil {
		return CombinedState{}, err
	}

	state, hasState, err := ReadState(root, normID)
	if err != nil {
		return CombinedState{}, err
	}

	events, err := ReadEvents(root, normID)
	if err != nil {
		return CombinedState{}, err
	}

	if checker == nil {
		checker = &DefaultWorktreeChecker{Root: root}
	}
	wtStatus := checker.CheckWorktree(normID)

	if !hasState && len(events) == 0 {
		return CombinedState{
			TaskID:         normID,
			Found:          false,
			WorktreeActive: wtStatus.Active,
			WorktreePath:   wtStatus.Path,
		}, nil
	}

	return CombinedState{
		TaskID:         normID,
		Found:          true,
		State:          state,
		Events:         events,
		WorktreeActive: wtStatus.Active,
		WorktreePath:   wtStatus.Path,
	}, nil
}

// Purge completely removes the task's state directory .sdd/state/<TASK_ID>.
func Purge(root, taskID string) (PurgeResult, error) {
	normID, err := ValidateTaskID(taskID)
	if err != nil {
		return PurgeResult{}, err
	}

	dir, err := TaskStateDir(root, normID)
	if err != nil {
		return PurgeResult{}, err
	}

	relDir, _ := RelativeStateDir(normID)

	if err := os.RemoveAll(dir); err != nil {
		return PurgeResult{}, fmt.Errorf("purge state directory %s: %w", dir, err)
	}

	return PurgeResult{
		TaskID: normID,
		Purged: true,
		Path:   relDir,
	}, nil
}

// ReadBounded reads at most maxBytes + 1 from r to detect overflow safely.
func ReadBounded(r io.Reader, maxBytes int64) ([]byte, error) {
	lr := io.LimitReader(r, maxBytes+1)
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("input payload exceeds limit of %d bytes", maxBytes)
	}
	return data, nil
}
