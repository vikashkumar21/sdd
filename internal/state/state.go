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
	"math"
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

// TokenTelemetry tracks accumulated token consumption metrics for a task.
type TokenTelemetry struct {
	InputTokens     int64 `json:"input_tokens"`
	CacheReadTokens int64 `json:"cache_read_tokens"`
	OutputTokens    int64 `json:"output_tokens"`
	TotalTokens     int64 `json:"total_tokens"`
}

// ExtractTokens extracts TokenTelemetry from a task state map if present.
func ExtractTokens(stateMap map[string]any) (TokenTelemetry, bool) {
	if stateMap == nil {
		return TokenTelemetry{}, false
	}
	raw, ok := stateMap["tokens"]
	if !ok || raw == nil {
		return TokenTelemetry{}, false
	}
	tokenBytes, err := json.Marshal(raw)
	if err != nil {
		return TokenTelemetry{}, false
	}
	var tt TokenTelemetry
	if err := json.Unmarshal(tokenBytes, &tt); err != nil {
		return TokenTelemetry{}, false
	}
	return tt, true
}

// Ingest describes one transcript-derived token recording delta and its source cursor.
type Ingest struct {
	Source   string
	Role     string
	FromStep int
	ToStep   int
	Steps    int
	Delta    TokenTelemetry
}

// CursorFor returns the last ingested step and role recorded for source in stateMap, if any.
func CursorFor(stateMap map[string]any, source string) (lastStep int, role string, ok bool) {
	ingestMap, err := parseIngestMap(stateMap)
	if err != nil || ingestMap == nil {
		return 0, "", false
	}
	entry, exists := ingestMap[source]
	if !exists {
		return 0, "", false
	}
	rawStep, hasStep := entry["last_step"]
	if !hasStep {
		return 0, "", false
	}
	stepFloat, isNum := rawStep.(float64)
	if !isNum {
		return 0, "", false
	}
	r, _ := entry["role"].(string)
	return int(stepFloat), r, true
}

func parseIngestMap(stateMap map[string]any) (map[string]map[string]any, error) {
	if stateMap == nil {
		return map[string]map[string]any{}, nil
	}
	raw, exists := stateMap["ingest"]
	if !exists || raw == nil {
		return map[string]map[string]any{}, nil
	}
	top, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("malformed ingest cursor table in state.json")
	}
	out := make(map[string]map[string]any, len(top))
	for src, v := range top {
		entry, ok := v.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("malformed ingest cursor entry for %q", src)
		}
		rawStep, hasStep := entry["last_step"]
		if !hasStep {
			return nil, fmt.Errorf("ingest cursor entry for %q missing last_step", src)
		}
		if _, isNum := rawStep.(float64); !isNum {
			return nil, fmt.Errorf("ingest cursor entry for %q has non-numeric last_step", src)
		}
		if stepFloat := rawStep.(float64); stepFloat < 0 {
			return nil, fmt.Errorf("ingest cursor entry for %q has negative last_step", src)
		}
		out[src] = entry
	}
	return out, nil
}

// ValidateIngestState verifies that any existing "ingest" cursor table in stateMap is well-formed.
func ValidateIngestState(stateMap map[string]any) error {
	_, err := parseIngestMap(stateMap)
	return err
}

// AccumulateTokens validates non-negative token deltas, guards against int64 overflow,
// and returns the updated TokenTelemetry totals.
func AccumulateTokens(existing TokenTelemetry, input, cache, output int64) (TokenTelemetry, error) {
	if input < 0 || cache < 0 || output < 0 {
		return TokenTelemetry{}, fmt.Errorf("token values cannot be negative (got input=%d, cache=%d, output=%d)", input, cache, output)
	}
	if math.MaxInt64-existing.InputTokens < input ||
		math.MaxInt64-existing.CacheReadTokens < cache ||
		math.MaxInt64-existing.OutputTokens < output {
		return TokenTelemetry{}, fmt.Errorf("token accumulation would overflow int64")
	}
	totals := TokenTelemetry{
		InputTokens:     existing.InputTokens + input,
		CacheReadTokens: existing.CacheReadTokens + cache,
		OutputTokens:    existing.OutputTokens + output,
	}
	if math.MaxInt64-totals.InputTokens < totals.CacheReadTokens ||
		math.MaxInt64-(totals.InputTokens+totals.CacheReadTokens) < totals.OutputTokens {
		return TokenTelemetry{}, fmt.Errorf("total token calculation would overflow int64")
	}
	totals.TotalTokens = totals.InputTokens + totals.CacheReadTokens + totals.OutputTokens
	return totals, nil
}

// RecordTokens validates non-negative token deltas, guards against overflow, accumulates into
// existing state.json under key "tokens", atomically saves the state snapshot, appends a
// "token_recorded" event to events.jsonl, and returns the SaveResult and updated TokenTelemetry.
func RecordTokens(root, taskID string, input, cache, output int64, now time.Time) (SaveResult, TokenTelemetry, error) {
	normID, err := ValidateTaskID(taskID)
	if err != nil {
		return SaveResult{}, TokenTelemetry{}, err
	}

	stateMap, hasState, err := ReadState(root, normID)
	if err != nil {
		return SaveResult{}, TokenTelemetry{}, err
	}
	if !hasState || stateMap == nil {
		stateMap = make(map[string]any)
	}

	existing, _ := ExtractTokens(stateMap)
	totals, err := AccumulateTokens(existing, input, cache, output)
	if err != nil {
		return SaveResult{}, TokenTelemetry{}, err
	}

	stateMap["tokens"] = totals
	payload, err := json.Marshal(stateMap)
	if err != nil {
		return SaveResult{}, TokenTelemetry{}, fmt.Errorf("marshal state with tokens: %w", err)
	}

	if now.IsZero() {
		now = time.Now()
	}

	res, err := SaveState(root, normID, payload, now)
	if err != nil {
		return SaveResult{}, TokenTelemetry{}, err
	}

	eventMap := map[string]any{
		"type":    "token_recorded",
		"task_id": normID,
		"delta": map[string]int64{
			"input":  input,
			"cache":  cache,
			"output": output,
		},
		"totals":      totals,
		"recorded_at": now.UTC().Format(time.RFC3339),
	}
	eventPayload, err := json.Marshal(eventMap)
	if err != nil {
		return res, totals, fmt.Errorf("marshal token event: %w", err)
	}
	if _, err := AppendEvent(root, normID, eventPayload); err != nil {
		return res, totals, fmt.Errorf("append token event: %w", err)
	}

	return res, totals, nil
}

// RecordIngest atomically accumulates transcript-derived token deltas into "tokens",
// updates the per-source cursor under "ingest", and appends a "token_ingested" event.
func RecordIngest(root, taskID string, ing Ingest, now time.Time) (SaveResult, TokenTelemetry, error) {
	normID, err := ValidateTaskID(taskID)
	if err != nil {
		return SaveResult{}, TokenTelemetry{}, err
	}
	if ing.Source == "" {
		return SaveResult{}, TokenTelemetry{}, fmt.Errorf("ingest source must not be empty")
	}
	if ing.Role == "" {
		return SaveResult{}, TokenTelemetry{}, fmt.Errorf("ingest role must not be empty")
	}
	if ing.FromStep < 0 || ing.ToStep < ing.FromStep {
		return SaveResult{}, TokenTelemetry{}, fmt.Errorf("invalid ingest step range %d..%d", ing.FromStep, ing.ToStep)
	}
	if ing.Steps <= 0 {
		return SaveResult{}, TokenTelemetry{}, fmt.Errorf("ingest steps must be positive (got %d)", ing.Steps)
	}

	stateMap, hasState, err := ReadState(root, normID)
	if err != nil {
		return SaveResult{}, TokenTelemetry{}, err
	}
	if !hasState || stateMap == nil {
		stateMap = make(map[string]any)
	}

	ingestMap, err := parseIngestMap(stateMap)
	if err != nil {
		return SaveResult{}, TokenTelemetry{}, err
	}
	if prev, exists := ingestMap[ing.Source]; exists {
		prevStep := int(prev["last_step"].(float64))
		if ing.FromStep <= prevStep {
			return SaveResult{}, TokenTelemetry{}, fmt.Errorf("from-step %d overlaps already-ingested cursor %d for source %s", ing.FromStep, prevStep, ing.Source)
		}
		if prevRole, _ := prev["role"].(string); prevRole != "" && prevRole != ing.Role {
			return SaveResult{}, TokenTelemetry{}, fmt.Errorf("role %q does not match stored role %q for source %s", ing.Role, prevRole, ing.Source)
		}
	}

	existing, _ := ExtractTokens(stateMap)
	totals, err := AccumulateTokens(existing, ing.Delta.InputTokens, ing.Delta.CacheReadTokens, ing.Delta.OutputTokens)
	if err != nil {
		return SaveResult{}, TokenTelemetry{}, err
	}

	ingestMap[ing.Source] = map[string]any{
		"last_step": ing.ToStep,
		"role":      ing.Role,
	}
	stateMap["tokens"] = totals
	stateMap["ingest"] = ingestMap

	payload, err := json.Marshal(stateMap)
	if err != nil {
		return SaveResult{}, TokenTelemetry{}, fmt.Errorf("marshal state with ingest: %w", err)
	}

	if now.IsZero() {
		now = time.Now()
	}

	res, err := SaveState(root, normID, payload, now)
	if err != nil {
		return SaveResult{}, TokenTelemetry{}, err
	}

	eventMap := map[string]any{
		"type":      "token_ingested",
		"task_id":   normID,
		"source":    ing.Source,
		"role":      ing.Role,
		"from_step": ing.FromStep,
		"to_step":   ing.ToStep,
		"steps":     ing.Steps,
		"delta": map[string]int64{
			"input":  ing.Delta.InputTokens,
			"cache":  ing.Delta.CacheReadTokens,
			"output": ing.Delta.OutputTokens,
		},
		"totals":      totals,
		"recorded_at": now.UTC().Format(time.RFC3339),
	}
	eventPayload, err := json.Marshal(eventMap)
	if err != nil {
		return res, totals, fmt.Errorf("marshal token_ingested event: %w", err)
	}
	if _, err := AppendEvent(root, normID, eventPayload); err != nil {
		return res, totals, fmt.Errorf("append token_ingested event: %w", err)
	}

	return res, totals, nil
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
