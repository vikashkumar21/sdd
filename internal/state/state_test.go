package state

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type mockWorktreeChecker struct {
	status WorktreeStatus
}

func (m *mockWorktreeChecker) CheckWorktree(taskID string) WorktreeStatus {
	return m.status
}

func TestValidateTaskID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "standard id", input: "T-001", want: "T-001", wantErr: false},
		{name: "lowercase id", input: "t-017", want: "T-017", wantErr: false},
		{name: "custom prefix", input: "FEATURE-42", want: "FEATURE-42", wantErr: false},
		{name: "trimmed whitespace", input: "  t-999  ", want: "T-999", wantErr: false},
		{name: "empty input", input: "", wantErr: true},
		{name: "only whitespace", input: "   ", wantErr: true},
		{name: "path traversal dotdot", input: "../T-001", wantErr: true},
		{name: "path traversal slash", input: "foo/T-001", wantErr: true},
		{name: "path traversal backslash", input: `foo\T-001`, wantErr: true},
		{name: "middle dotdot", input: "T-..001", wantErr: true},
		{name: "missing hyphen", input: "T001", wantErr: true},
		{name: "underscore", input: "T_001", wantErr: true},
		{name: "missing digits", input: "T-", wantErr: true},
		{name: "missing prefix", input: "-123", wantErr: true},
		{name: "non-digit suffix", input: "T-abc", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateTaskID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateTaskID(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ValidateTaskID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestStoreLayout(t *testing.T) {
	root := "/test/root"
	taskID := "t-017"

	dir, err := TaskStateDir(root, taskID)
	if err != nil {
		t.Fatalf("TaskStateDir error: %v", err)
	}
	expectedDir := filepath.Join(root, ".sdd", "state", "T-017")
	if dir != expectedDir {
		t.Errorf("TaskStateDir = %q, want %q", dir, expectedDir)
	}

	relDir, err := RelativeStateDir(taskID)
	if err != nil {
		t.Fatalf("RelativeStateDir error: %v", err)
	}
	if relDir != ".sdd/state/T-017" {
		t.Errorf("RelativeStateDir = %q, want .sdd/state/T-017", relDir)
	}

	relFile, err := RelativeStateFile(taskID)
	if err != nil {
		t.Fatalf("RelativeStateFile error: %v", err)
	}
	if relFile != ".sdd/state/T-017/state.json" {
		t.Errorf("RelativeStateFile = %q, want .sdd/state/T-017/state.json", relFile)
	}
}

func TestSaveAtomic(t *testing.T) {
	tempDir := t.TempDir()
	taskID := "T-017"

	fixedTime := time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)
	payload := []byte(`{"phase":"Implement","planned_files":["internal/state/state.go"]}`)

	res, err := SaveState(tempDir, taskID, payload, fixedTime)
	if err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	if !res.Saved || res.TaskID != "T-017" {
		t.Errorf("unexpected SaveResult: %+v", res)
	}
	if res.Path != ".sdd/state/T-017/state.json" {
		t.Errorf("expected path .sdd/state/T-017/state.json, got %q", res.Path)
	}
	if res.UpdatedAt != "2026-11-14T10:00:00Z" {
		t.Errorf("expected UpdatedAt 2026-11-14T10:00:00Z, got %q", res.UpdatedAt)
	}

	// Verify file on disk
	stateFile := filepath.Join(tempDir, ".sdd", "state", "T-017", "state.json")
	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("read saved state.json: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal state.json: %v", err)
	}
	if parsed["phase"] != "Implement" {
		t.Errorf("expected phase Implement, got %v", parsed["phase"])
	}

	// Ensure no temp files left behind
	entries, err := os.ReadDir(filepath.Dir(stateFile))
	if err != nil {
		t.Fatalf("read state dir: %v", err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp.") {
			t.Errorf("found leftover temp file: %s", e.Name())
		}
	}

	// Update existing state
	updatedPayload := []byte(`{"phase":"Verify","planned_files":["internal/state/state.go"]}`)
	res2, err := SaveState(tempDir, taskID, updatedPayload, fixedTime.Add(time.Hour))
	if err != nil {
		t.Fatalf("SaveState update failed: %v", err)
	}
	if !res2.Saved {
		t.Errorf("expected res2.Saved == true")
	}

	data2, _ := os.ReadFile(stateFile)
	var parsed2 map[string]any
	_ = json.Unmarshal(data2, &parsed2)
	if parsed2["phase"] != "Verify" {
		t.Errorf("expected phase Verify, got %v", parsed2["phase"])
	}
}

func TestSaveInvalidJSON(t *testing.T) {
	tempDir := t.TempDir()

	invalidPayloads := [][]byte{
		nil,
		[]byte(""),
		[]byte("   "),
		[]byte("{invalid-json"),
		[]byte("not a json"),
		[]byte(`{"unclosed": "string}`),
	}

	for _, payload := range invalidPayloads {
		_, err := SaveState(tempDir, "T-001", payload, time.Now())
		if err == nil {
			t.Errorf("expected error for payload %q, got nil", string(payload))
		}
	}
}

func TestAppendEvent(t *testing.T) {
	tempDir := t.TempDir()
	taskID := "T-017"

	// 1. First event
	evt1 := []byte(`{"kind":"PHASE_TRANSITION","from_phase":"","to_phase":"Specify","details":"Task initialized"}`)
	res1, err := AppendEvent(tempDir, taskID, evt1)
	if err != nil {
		t.Fatalf("AppendEvent 1 failed: %v", err)
	}
	if !res1.Appended || res1.EventCount != 1 {
		t.Errorf("unexpected AppendResult 1: %+v", res1)
	}

	// 2. Second event with formatted/multiline json
	evt2 := []byte(`{
		"kind": "GATE_EVALUATED",
		"details": "GATE 1 Approved"
	}`)
	res2, err := AppendEvent(tempDir, taskID, evt2)
	if err != nil {
		t.Fatalf("AppendEvent 2 failed: %v", err)
	}
	if !res2.Appended || res2.EventCount != 2 {
		t.Errorf("unexpected AppendResult 2: %+v", res2)
	}

	// 3. Inspect raw events.jsonl
	eventsFile := filepath.Join(tempDir, ".sdd", "state", "T-017", "events.jsonl")
	data, err := os.ReadFile(eventsFile)
	if err != nil {
		t.Fatalf("read events.jsonl: %v", err)
	}

	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines in events.jsonl, got %d:\n%s", len(lines), string(data))
	}

	// Each line must be valid JSON and compact
	for i, line := range lines {
		var dummy map[string]any
		if err := json.Unmarshal([]byte(line), &dummy); err != nil {
			t.Errorf("line %d is not valid JSON: %v", i+1, err)
		}
	}

	// 4. Invalid event payload rejected
	_, err = AppendEvent(tempDir, taskID, []byte("{invalid"))
	if err == nil {
		t.Errorf("expected error appending invalid JSON, got nil")
	}

	// Count remains 2
	count, err := CountEvents(tempDir, taskID)
	if err != nil {
		t.Fatalf("CountEvents failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}
}

func TestEventScannerBuffer(t *testing.T) {
	tempDir := t.TempDir()
	taskID := "T-099"

	// Create events file with a corrupt line
	stateDir := filepath.Join(tempDir, ".sdd", "state", "T-099")
	_ = os.MkdirAll(stateDir, 0o755)
	eventsFile := filepath.Join(stateDir, "events.jsonl")

	content := `{"kind":"EVT_1"}
{CORRUPT_JSON_LINE
{"kind":"EVT_2"}
`
	_ = os.WriteFile(eventsFile, []byte(content), 0o644)

	_, err := ReadEvents(tempDir, taskID)
	if err == nil {
		t.Fatalf("expected ReadEvents to fail on corrupt line, got nil")
	}
	if !strings.Contains(err.Error(), "corrupt event line 2") {
		t.Errorf("expected error message to mention line 2, got %v", err)
	}
}

func TestShow(t *testing.T) {
	tempDir := t.TempDir()
	taskID := "T-017"

	mockChecker := &mockWorktreeChecker{
		status: WorktreeStatus{Active: true, Path: "/path/to/worktree/T-017"},
	}

	// 1. Show when no state or events exist
	showEmpty, err := Show(tempDir, taskID, mockChecker)
	if err != nil {
		t.Fatalf("Show empty failed: %v", err)
	}
	if showEmpty.Found {
		t.Errorf("expected Found == false, got true")
	}
	if !showEmpty.WorktreeActive || showEmpty.WorktreePath != "/path/to/worktree/T-017" {
		t.Errorf("unexpected worktree info: %+v", showEmpty)
	}

	// 2. Save state and append event
	_ = os.MkdirAll(filepath.Join(tempDir, ".sdd", "state", "T-017"), 0o755)
	_, _ = SaveState(tempDir, taskID, []byte(`{"phase":"Implement","branch":"feature/T-017"}`), time.Now())
	_, _ = AppendEvent(tempDir, taskID, []byte(`{"kind":"PHASE_TRANSITION","to_phase":"Implement"}`))

	showPopulated, err := Show(tempDir, taskID, mockChecker)
	if err != nil {
		t.Fatalf("Show populated failed: %v", err)
	}
	if !showPopulated.Found {
		t.Fatalf("expected Found == true")
	}
	if showPopulated.State["phase"] != "Implement" {
		t.Errorf("expected state phase Implement, got %v", showPopulated.State["phase"])
	}
	if len(showPopulated.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(showPopulated.Events))
	}
	if !showPopulated.WorktreeActive {
		t.Errorf("expected worktree_active == true")
	}
}

func TestPurge(t *testing.T) {
	tempDir := t.TempDir()
	taskID := "T-017"

	// Create state and events
	_, _ = SaveState(tempDir, taskID, []byte(`{"phase":"Done"}`), time.Now())
	_, _ = AppendEvent(tempDir, taskID, []byte(`{"kind":"COMPLETED"}`))

	dir := filepath.Join(tempDir, ".sdd", "state", "T-017")
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("expected dir to exist before purge: %v", err)
	}

	res, err := Purge(tempDir, taskID)
	if err != nil {
		t.Fatalf("Purge failed: %v", err)
	}
	if !res.Purged || res.TaskID != "T-017" {
		t.Errorf("unexpected PurgeResult: %+v", res)
	}
	if res.Path != ".sdd/state/T-017" {
		t.Errorf("expected path .sdd/state/T-017, got %q", res.Path)
	}

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("expected dir to be removed after purge, stat err = %v", err)
	}

	// Purging again is a no-op that succeeds
	res2, err := Purge(tempDir, taskID)
	if err != nil {
		t.Fatalf("Purge non-existent failed: %v", err)
	}
	if !res2.Purged {
		t.Errorf("expected res2.Purged == true")
	}
}

func TestReadBounded(t *testing.T) {
	data := []byte("hello world")

	// 1. Within bound
	read1, err := ReadBounded(bytes.NewReader(data), 20)
	if err != nil {
		t.Fatalf("ReadBounded failed: %v", err)
	}
	if string(read1) != "hello world" {
		t.Errorf("expected 'hello world', got %q", string(read1))
	}

	// 2. Exactly at bound
	read2, err := ReadBounded(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("ReadBounded exact failed: %v", err)
	}
	if string(read2) != "hello world" {
		t.Errorf("expected 'hello world', got %q", string(read2))
	}

	// 3. Exceeding bound
	_, err = ReadBounded(bytes.NewReader(data), 5)
	if err == nil {
		t.Fatalf("expected error when reading data exceeding bound, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds limit") {
		t.Errorf("expected error message to mention exceeds limit, got %v", err)
	}
}
