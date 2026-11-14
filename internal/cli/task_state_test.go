package cli

import (
	"encoding/json"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vikashkumar21/sdd/internal/state"
)

func TestTaskState_Usage(t *testing.T) {
	r := exec(t, "", false, "", "task", "state")
	if r.code != 2 {
		t.Fatalf("expected exit code 2, got %d", r.code)
	}
	if !strings.Contains(r.err, "usage: sdd task state <command>") {
		t.Errorf("expected usage message in stderr, got:\n%s", r.err)
	}

	r2 := exec(t, "", false, "", "task", "state", "bogus")
	if r2.code != 2 {
		t.Fatalf("expected exit code 2 for unknown command, got %d", r2.code)
	}
	if !strings.Contains(r2.err, "unknown command \"bogus\"") {
		t.Errorf("expected unknown command message, got:\n%s", r2.err)
	}
}

func TestTaskState_ShowEmpty(t *testing.T) {
	dir := t.TempDir()

	// 1. Text mode on non-existent task state
	r1 := exec(t, "", false, "", "task", "state", "show", "T-017", "--dir", dir)
	if r1.code != 1 {
		t.Fatalf("expected exit code 1 for missing state in text mode, got %d", r1.code)
	}
	if !strings.Contains(r1.err, "no state or events found for T-017") {
		t.Errorf("expected missing state error message, got:\n%s", r1.err)
	}

	// 2. JSON mode on non-existent task state
	r2 := exec(t, "", false, "", "task", "state", "show", "T-017", "--dir", dir, "--json")
	if r2.code != 0 {
		t.Fatalf("expected exit code 0 for missing state in JSON mode, got %d (err: %s)", r2.code, r2.err)
	}
	var res map[string]any
	if err := json.Unmarshal([]byte(r2.out), &res); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v\nOutput: %s", err, r2.out)
	}
	if res["found"] != false || res["task_id"] != "T-017" {
		t.Errorf("unexpected empty JSON result: %+v", res)
	}

	// 3. Path traversal in task ID
	r3 := exec(t, "", false, "", "task", "state", "show", "../T-017", "--dir", dir)
	if r3.code != 1 {
		t.Fatalf("expected exit code 1 for path traversal task ID, got %d", r3.code)
	}
	if !strings.Contains(r3.err, "path traversal") {
		t.Errorf("expected path traversal error message, got:\n%s", r3.err)
	}
}

func TestTaskState_SaveAndAppend(t *testing.T) {
	dir := t.TempDir()
	taskID := "T-017"

	// 1. Missing payload flags
	rMissing := exec(t, "", false, "", "task", "state", "save", taskID, "--dir", dir)
	if rMissing.code != 2 {
		t.Fatalf("expected exit code 2 when neither --state nor --stdin is passed, got %d", rMissing.code)
	}

	// 2. Save via --state with --json
	statePayload := `{"phase":"Implement","planned_files":["internal/state/state.go"]}`
	rSave := exec(t, "", false, "", "task", "state", "save", taskID, "--dir", dir, "--state", statePayload, "--json")
	if rSave.code != 0 {
		t.Fatalf("expected exit code 0 for save, got %d (err: %s)", rSave.code, rSave.err)
	}
	var saveRes state.SaveResult
	if err := json.Unmarshal([]byte(rSave.out), &saveRes); err != nil {
		t.Fatalf("failed to unmarshal save JSON: %v\nOutput: %s", err, rSave.out)
	}
	if !saveRes.Saved || saveRes.TaskID != "T-017" || saveRes.Path != ".sdd/state/T-017/state.json" {
		t.Errorf("unexpected SaveResult: %+v", saveRes)
	}

	// 3. Save via --stdin in text mode
	rSaveStdin := exec(t, `{"phase":"Verify","planned_files":["internal/state/state.go"]}`, false, "", "task", "state", "save", taskID, "--dir", dir, "--stdin")
	if rSaveStdin.code != 0 {
		t.Fatalf("expected exit code 0 for save --stdin, got %d (err: %s)", rSaveStdin.code, rSaveStdin.err)
	}
	if !strings.Contains(rSaveStdin.out, "Saved task state for T-017") {
		t.Errorf("expected save confirmation, got:\n%s", rSaveStdin.out)
	}

	// 4. Append event via --event with --json
	evtPayload1 := `{"kind":"GATE_EVALUATED","details":"GATE 1 Approved"}`
	rEvt1 := exec(t, "", false, "", "task", "state", "append-event", taskID, "--dir", dir, "--event", evtPayload1, "--json")
	if rEvt1.code != 0 {
		t.Fatalf("expected exit code 0 for append-event, got %d (err: %s)", rEvt1.code, rEvt1.err)
	}
	var evtRes1 state.AppendResult
	if err := json.Unmarshal([]byte(rEvt1.out), &evtRes1); err != nil {
		t.Fatalf("failed to unmarshal event JSON: %v\nOutput: %s", err, rEvt1.out)
	}
	if !evtRes1.Appended || evtRes1.EventCount != 1 {
		t.Errorf("unexpected AppendResult 1: %+v", evtRes1)
	}

	// 5. Append event via --stdin in text mode
	rEvtStdin := exec(t, `{"kind":"IMPLEMENT_STEP","details":"Step 1 completed"}`, false, "", "task", "state", "append-event", taskID, "--dir", dir, "--stdin")
	if rEvtStdin.code != 0 {
		t.Fatalf("expected exit code 0 for append-event --stdin, got %d (err: %s)", rEvtStdin.code, rEvtStdin.err)
	}
	if !strings.Contains(rEvtStdin.out, "Appended event to .sdd/state/T-017/events.jsonl (total events: 2)") {
		t.Errorf("expected append confirmation, got:\n%s", rEvtStdin.out)
	}

	// 6. Show state in JSON mode
	rShowJSON := exec(t, "", false, "", "task", "state", "show", taskID, "--dir", dir, "--json")
	if rShowJSON.code != 0 {
		t.Fatalf("expected exit code 0 for show --json, got %d (err: %s)", rShowJSON.code, rShowJSON.err)
	}
	var showRes state.CombinedState
	if err := json.Unmarshal([]byte(rShowJSON.out), &showRes); err != nil {
		t.Fatalf("failed to unmarshal show JSON: %v\nOutput: %s", err, rShowJSON.out)
	}
	if !showRes.Found || showRes.TaskID != "T-017" {
		t.Errorf("unexpected CombinedState: %+v", showRes)
	}
	if showRes.State["phase"] != "Verify" {
		t.Errorf("expected phase Verify, got %v", showRes.State["phase"])
	}
	if len(showRes.Events) != 2 {
		t.Errorf("expected 2 events, got %d", len(showRes.Events))
	}

	// 7. Show state in text mode
	rShowText := exec(t, "", false, "", "task", "state", "show", taskID, "--dir", dir)
	if rShowText.code != 0 {
		t.Fatalf("expected exit code 0 for show text mode, got %d (err: %s)", rShowText.code, rShowText.err)
	}
	if !strings.Contains(rShowText.out, "Task State: T-017") || !strings.Contains(rShowText.out, "phase: Verify") || !strings.Contains(rShowText.out, "Events (2):") {
		t.Errorf("unexpected text show output:\n%s", rShowText.out)
	}
}

func TestTaskState_Purge(t *testing.T) {
	dir := t.TempDir()
	taskID := "T-017"

	// Create state
	_ = exec(t, "", false, "", "task", "state", "save", taskID, "--dir", dir, "--state", `{"phase":"Done"}`)

	// 1. Non-interactive purge without --force should fail with exit code 1
	rNoForce := exec(t, "", false, "", "task", "state", "purge", taskID, "--dir", dir)
	if rNoForce.code != 1 {
		t.Fatalf("expected exit code 1 without --force in non-interactive mode, got %d", rNoForce.code)
	}
	if !strings.Contains(rNoForce.err, "--force required") {
		t.Errorf("expected --force error message, got:\n%s", rNoForce.err)
	}

	// 2. Interactive purge with confirmation "y"
	rInteractiveYes := exec(t, "y\n", true, "", "task", "state", "purge", taskID, "--dir", dir)
	if rInteractiveYes.code != 0 {
		t.Fatalf("expected exit code 0 for interactive purge yes, got %d (err: %s)", rInteractiveYes.code, rInteractiveYes.err)
	}
	if !strings.Contains(rInteractiveYes.out, "Purged state directory for T-017") {
		t.Errorf("expected purge confirmation, got:\n%s", rInteractiveYes.out)
	}

	// Recreate state for next test
	_ = exec(t, "", false, "", "task", "state", "save", taskID, "--dir", dir, "--state", `{"phase":"Done"}`)

	// 3. Interactive purge with cancellation "n"
	rInteractiveNo := exec(t, "n\n", true, "", "task", "state", "purge", taskID, "--dir", dir)
	if rInteractiveNo.code != 0 {
		t.Fatalf("expected exit code 0 for interactive purge cancellation, got %d", rInteractiveNo.code)
	}
	if !strings.Contains(rInteractiveNo.out, "Purge cancelled.") {
		t.Errorf("expected cancellation message, got:\n%s", rInteractiveNo.out)
	}

	// 4. Purge with --force --json
	rForceJSON := exec(t, "", false, "", "task", "state", "purge", taskID, "--dir", dir, "--force", "--json")
	if rForceJSON.code != 0 {
		t.Fatalf("expected exit code 0 for purge --force --json, got %d (err: %s)", rForceJSON.code, rForceJSON.err)
	}
	var purgeRes state.PurgeResult
	if err := json.Unmarshal([]byte(rForceJSON.out), &purgeRes); err != nil {
		t.Fatalf("failed to parse purge JSON: %v\nOutput: %s", err, rForceJSON.out)
	}
	if !purgeRes.Purged || purgeRes.TaskID != "T-017" || purgeRes.Path != ".sdd/state/T-017" {
		t.Errorf("unexpected PurgeResult: %+v", purgeRes)
	}

	// Verify state directory is gone
	stateDir := filepath.Join(dir, ".sdd", "state", "T-017")
	if _, err := os.Stat(stateDir); !os.IsNotExist(err) {
		t.Errorf("expected state dir to be deleted, stat error = %v", err)
	}
}

func TestTaskClean_PurgeState(t *testing.T) {
	dir := t.TempDir()

	runGit := func(args ...string) {
		t.Helper()
		cmd := osexec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Vikash",
			"GIT_AUTHOR_EMAIL=vikash@example.com",
			"GIT_COMMITTER_NAME=Vikash",
			"GIT_COMMITTER_EMAIL=vikash@example.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	runGit("init")
	runGit("config", "user.name", "Vikash")
	runGit("config", "user.email", "vikash@example.com")
	_ = os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test Repo\n"), 0o644)
	runGit("add", "README.md")
	runGit("commit", "-m", "init")

	// Create task folder
	taskFolder := filepath.Join(dir, "tasks", "features", "T-001-worktree-cli")
	if err := os.MkdirAll(taskFolder, 0o755); err != nil {
		t.Fatal(err)
	}

	// 1. Start task worktree
	rStart := exec(t, "", false, "", "task", "start", "T-001", "--dir", dir)
	if rStart.code != 0 {
		t.Fatalf("expected exit 0 for task start, got %d: %s (err: %s)", rStart.code, rStart.out, rStart.err)
	}

	// 2. Save task state
	rSave := exec(t, "", false, "", "task", "state", "save", "T-001", "--dir", dir, "--state", `{"phase":"Implement"}`)
	if rSave.code != 0 {
		t.Fatalf("expected exit 0 for task state save, got %d: %s", rSave.code, rSave.err)
	}

	stateDir := filepath.Join(dir, ".sdd", "state", "T-001")
	if _, err := os.Stat(stateDir); err != nil {
		t.Fatalf("expected state dir to exist before clean: %v", err)
	}

	// 3. Run sdd task clean T-001 --purge-state
	rClean := exec(t, "", false, "", "task", "clean", "T-001", "--dir", dir, "--delete-branch", "--purge-state")
	if rClean.code != 0 {
		t.Fatalf("expected exit 0 for task clean --purge-state, got %d: %s (err: %s)", rClean.code, rClean.out, rClean.err)
	}

	if !strings.Contains(rClean.out, "Cleaned worktree for T-001") {
		t.Errorf("expected clean worktree confirmation, got:\n%s", rClean.out)
	}
	if !strings.Contains(rClean.out, "Purged state directory for T-001") {
		t.Errorf("expected purge state confirmation, got:\n%s", rClean.out)
	}

	// Verify state directory is removed
	if _, err := os.Stat(stateDir); !os.IsNotExist(err) {
		t.Errorf("expected state dir to be deleted after --purge-state, stat err = %v", err)
	}
}
