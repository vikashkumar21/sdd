package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vikashkumar21/sdd/internal/check"
)

func TestCheckJSONFlag(t *testing.T) {
	root := t.TempDir()
	exec(t, "", false, "", "init", "--yes", "--dir", root, "--name", "jsoncheck")

	r := exec(t, "", false, "", "check", "--dir", root, "--json")
	if r.code != 0 {
		t.Fatalf("expected sdd check --json to exit 0, got %d. stderr: %s", r.code, r.err)
	}

	var rep check.JSONReport
	if err := json.Unmarshal([]byte(r.out), &rep); err != nil {
		t.Fatalf("failed to unmarshal JSON output: %v\nOutput was:\n%s", err, r.out)
	}

	if !rep.Valid {
		t.Errorf("expected rep.Valid = true, got false")
	}
	if rep.Summary.Errors != 0 {
		t.Errorf("expected 0 errors, got %d", rep.Summary.Errors)
	}
}

func TestCheckJSONStrictMode(t *testing.T) {
	root := t.TempDir()
	exec(t, "", false, "", "init", "--yes", "--dir", root, "--name", "strictjson")

	// Inject a warning: remove README.md
	_ = os.Remove(filepath.Join(root, "README.md"))

	// Non-strict JSON should still exit 0 (warnings are not failures by default)
	rNonStrict := exec(t, "", false, "", "check", "--dir", root, "--json")
	if rNonStrict.code != 0 {
		t.Errorf("expected non-strict check with warning to exit 0, got %d", rNonStrict.code)
	}
	var repNonStrict check.JSONReport
	if err := json.Unmarshal([]byte(rNonStrict.out), &repNonStrict); err != nil {
		t.Fatal(err)
	}
	if !repNonStrict.Valid {
		t.Errorf("expected repNonStrict.Valid = true without strict mode")
	}
	if repNonStrict.Summary.Warnings == 0 {
		t.Errorf("expected >0 warnings for missing README.md")
	}

	// Strict JSON mode should exit 1 and report valid = false
	rStrict := exec(t, "", false, "", "check", "--dir", root, "--json", "--strict")
	if rStrict.code != 1 {
		t.Errorf("expected strict check with warnings to exit 1, got %d", rStrict.code)
	}
	var repStrict check.JSONReport
	if err := json.Unmarshal([]byte(rStrict.out), &repStrict); err != nil {
		t.Fatal(err)
	}
	if repStrict.Valid {
		t.Errorf("expected repStrict.Valid = false under --strict with warnings")
	}
}

func TestCheckHumanReadableDefault(t *testing.T) {
	root := t.TempDir()
	exec(t, "", false, "", "init", "--yes", "--dir", root, "--name", "plaincheck")

	r := exec(t, "", false, "", "check", "--dir", root)
	if r.code != 0 {
		t.Fatalf("check failed: %+v", r)
	}
	if strings.HasPrefix(strings.TrimSpace(r.out), "{") {
		t.Errorf("plain check output should not start with JSON brace")
	}
	if !strings.Contains(r.out, "0 error(s)") {
		t.Errorf("expected summary count in plain text output")
	}
}
