package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vikashkumar21/sdd/internal/state"
)

func setupBugfixTaskForRepair(t *testing.T, taskID, slug string) string {
	t.Helper()
	root := t.TempDir()
	var out, errOut bytes.Buffer
	env := IO{Out: &out, Err: &errOut}

	initCode := Run([]string{"init", "--yes", "--dir", root, "--name", "repair-demo"}, env)
	if initCode != 0 {
		t.Fatalf("init failed: %s", errOut.String())
	}

	ciLog := `=== RUN   TestDatabaseConnection
--- FAIL: TestDatabaseConnection (0.02s)
    db_test.go:34: connection refused to localhost:5432
FAIL
exit status 1
`
	ciFile := filepath.Join(t.TempDir(), "db-failure.log")
	if err := os.WriteFile(ciFile, []byte(ciLog), 0o644); err != nil {
		t.Fatal(err)
	}

	out.Reset()
	errOut.Reset()
	code := Run([]string{"bugfix", "new", taskID, slug, "--from-ci", ciFile, "--dir", root}, env)
	if code != 0 {
		t.Fatalf("bugfix new failed: %s", errOut.String())
	}

	return root
}

func TestPromptRepair_UsageAndValidation(t *testing.T) {
	root := t.TempDir()
	var out, errOut bytes.Buffer
	env := IO{Out: &out, Err: &errOut}

	// 1. Missing task ID
	code := Run([]string{"prompt", "repair"}, env)
	if code != 2 {
		t.Errorf("expected exit code 2 for missing task ID, got %d", code)
	}
	if !strings.Contains(errOut.String(), "usage: sdd prompt repair") {
		t.Errorf("expected usage message, got %s", errOut.String())
	}

	// 2. Invalid max-loops (< 1 or > 5)
	out.Reset()
	errOut.Reset()
	code = Run([]string{"prompt", "repair", "T-001", "--max-loops", "0"}, env)
	if code != 2 {
		t.Errorf("expected exit code 2 for --max-loops 0, got %d", code)
	}
	if !strings.Contains(errOut.String(), "--max-loops must be between 1 and 5") {
		t.Errorf("expected error about --max-loops bounds, got %s", errOut.String())
	}

	out.Reset()
	errOut.Reset()
	code = Run([]string{"prompt", "repair", "T-001", "--max-loops", "6"}, env)
	if code != 2 {
		t.Errorf("expected exit code 2 for --max-loops 6, got %d", code)
	}

	// 3. Invalid task ID format
	out.Reset()
	errOut.Reset()
	code = Run([]string{"prompt", "repair", "INVALID_ID"}, env)
	if code != 1 {
		t.Errorf("expected exit code 1 for invalid task ID, got %d", code)
	}

	// 4. Non-existent task folder
	out.Reset()
	errOut.Reset()
	code = Run([]string{"prompt", "repair", "T-999", "--dir", root}, env)
	if code != 1 {
		t.Errorf("expected exit code 1 for non-existent task, got %d", code)
	}
}

func TestPromptRepair_OutputAndStateLoopTracking(t *testing.T) {
	root := setupBugfixTaskForRepair(t, "T-001", "db-reconnect")

	var out, errOut bytes.Buffer
	env := IO{
		Out: &out,
		Err: &errOut,
		Now: func() time.Time { return time.Date(2026, 11, 19, 14, 0, 0, 0, time.UTC) },
	}

	// Attempt 1: max-loops = 3
	code := Run([]string{"prompt", "repair", "T-001", "--max-loops", "3", "--dir", root}, env)
	if code != 0 {
		t.Fatalf("prompt repair attempt 1 failed with %d: %s", code, errOut.String())
	}

	outStr := out.String()
	if !strings.Contains(outStr, "# Auto-Repair Handover Prompt — T-001") {
		t.Errorf("expected title in prompt, got:\n%s", outStr)
	}
	if !strings.Contains(outStr, "AUTO-REPAIR LOOP: ATTEMPT 1 OF 3") {
		t.Errorf("expected attempt 1 of 3 status, got:\n%s", outStr)
	}
	if !strings.Contains(outStr, "Mandatory Step 0: Minimal Failing Test Case") {
		t.Errorf("expected Step 0 section in prompt, got:\n%s", outStr)
	}
	if !strings.Contains(outStr, "TestDatabaseConnection") {
		t.Errorf("expected failure diagnostics in prompt, got:\n%s", outStr)
	}

	// Check state in .sdd/state/T-001/state.json
	st, ok, err := state.ReadState(root, "T-001")
	if err != nil || !ok {
		t.Fatalf("failed to read task state: ok=%v, err=%v", ok, err)
	}
	if attempts, ok := st["repair_attempts"].(float64); !ok || int(attempts) != 1 {
		t.Errorf("expected repair_attempts=1 in state, got %v", st["repair_attempts"])
	}

	// Attempt 2: max-loops = 3
	out.Reset()
	errOut.Reset()
	code = Run([]string{"prompt", "repair", "T-001", "--max-loops", "3", "--dir", root}, env)
	if code != 0 {
		t.Fatalf("prompt repair attempt 2 failed with %d: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "AUTO-REPAIR LOOP: ATTEMPT 2 OF 3") {
		t.Errorf("expected attempt 2 of 3 status, got:\n%s", out.String())
	}

	// Attempt 3: max-loops = 3 (circuit breaker ceiling reached!)
	out.Reset()
	errOut.Reset()
	code = Run([]string{"prompt", "repair", "T-001", "--max-loops", "3", "--dir", root}, env)
	if code != 0 {
		t.Fatalf("prompt repair attempt 3 failed with %d: %s", code, errOut.String())
	}
	circuitBreakerOut := out.String()
	if !strings.Contains(circuitBreakerOut, "CIRCUIT BREAKER ACTIVATED: ATTEMPT 3 OF 3 CEILING REACHED") {
		t.Errorf("expected circuit breaker ceiling reached alert, got:\n%s", circuitBreakerOut)
	}
	if !strings.Contains(circuitBreakerOut, "HALT IMMEDIATELY") {
		t.Errorf("expected HALT IMMEDIATELY directive in prompt, got:\n%s", circuitBreakerOut)
	}
	if !strings.Contains(circuitBreakerOut, "Blocked (circuit breaker ceiling reached: attempt 3 of 3)") {
		t.Errorf("expected Blocked status instruction in prompt, got:\n%s", circuitBreakerOut)
	}
}

func TestPromptRepair_JSON(t *testing.T) {
	root := setupBugfixTaskForRepair(t, "T-001", "db-reconnect")

	var out, errOut bytes.Buffer
	env := IO{
		Out: &out,
		Err: &errOut,
		Now: func() time.Time { return time.Date(2026, 11, 19, 14, 0, 0, 0, time.UTC) },
	}

	code := Run([]string{"prompt", "repair", "T-001", "--max-loops", "2", "--json", "--dir", root}, env)
	if code != 0 {
		t.Fatalf("prompt repair --json failed with %d: %s", code, errOut.String())
	}

	var ctx RepairPromptContext
	if err := json.Unmarshal(out.Bytes(), &ctx); err != nil {
		t.Fatalf("failed to parse JSON context: %v\nOutput:\n%s", err, out.String())
	}

	if ctx.TaskID != "T-001" {
		t.Errorf("expected TaskID T-001, got %s", ctx.TaskID)
	}
	if ctx.Attempt != 1 {
		t.Errorf("expected Attempt 1, got %d", ctx.Attempt)
	}
	if ctx.MaxLoops != 2 {
		t.Errorf("expected MaxLoops 2, got %d", ctx.MaxLoops)
	}
	if ctx.Halted {
		t.Errorf("expected Halted=false on attempt 1 of 2")
	}
	if !strings.Contains(ctx.ReproductionDoc, "TestDatabaseConnection") {
		t.Errorf("expected ReproductionDoc to contain diagnostics")
	}

	// Next call in JSON mode reaches ceiling (attempt 2 of 2)
	out.Reset()
	errOut.Reset()
	code = Run([]string{"prompt", "repair", "T-001", "--max-loops", "2", "--json", "--dir", root}, env)
	if code != 0 {
		t.Fatalf("second prompt repair --json failed with %d: %s", code, errOut.String())
	}

	var ctx2 RepairPromptContext
	if err := json.Unmarshal(out.Bytes(), &ctx2); err != nil {
		t.Fatalf("failed to parse second JSON context: %v", err)
	}
	if ctx2.Attempt != 2 {
		t.Errorf("expected Attempt 2, got %d", ctx2.Attempt)
	}
	if !ctx2.Halted {
		t.Errorf("expected Halted=true on attempt 2 of 2")
	}
	if !strings.Contains(ctx2.HaltReason, "circuit breaker ceiling reached") {
		t.Errorf("expected HaltReason to mention ceiling, got %q", ctx2.HaltReason)
	}
}
