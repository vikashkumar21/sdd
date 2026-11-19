package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vikashkumar21/sdd/internal/ci"
)

func TestCIInit_Usage(t *testing.T) {
	var out, errOut bytes.Buffer
	env := IO{Out: &out, Err: &errOut}

	code := runCI([]string{}, env)
	if code != 2 {
		t.Fatalf("expected code 2, got %d", code)
	}
	if !strings.Contains(errOut.String(), "usage: sdd ci <command>") {
		t.Errorf("expected usage message, got %q", errOut.String())
	}
}

func TestCIInit_InvalidPlatform(t *testing.T) {
	var out, errOut bytes.Buffer
	env := IO{Out: &out, Err: &errOut}

	code := runCIInit([]string{"--platform", "invalid-platform"}, env)
	if code != 2 {
		t.Fatalf("expected code 2, got %d", code)
	}
	if !strings.Contains(errOut.String(), "unknown platform \"invalid-platform\"") {
		t.Errorf("expected unknown platform error, got %q", errOut.String())
	}
}

func TestCIInit_DryRunAll(t *testing.T) {
	tmpDir := t.TempDir()
	var out, errOut bytes.Buffer
	env := IO{Out: &out, Err: &errOut}

	code := runCIInit([]string{"--platform", "all", "--dir", tmpDir, "--dry-run"}, env)
	if code != 0 {
		t.Fatalf("expected code 0, got %d (err: %s)", code, errOut.String())
	}

	output := out.String()
	if !strings.Contains(output, "Dry-run plan") {
		t.Errorf("expected Dry-run plan in output, got: %s", output)
	}
	if !strings.Contains(output, ".github/workflows/sdd-check.yml (github)") && !strings.Contains(output, ".github\\workflows\\sdd-check.yml (github)") {
		t.Errorf("expected github workflow in dry run, got: %s", output)
	}
	if !strings.Contains(output, "cloudbuild.yaml (gcp)") {
		t.Errorf("expected gcp workflow in dry run, got: %s", output)
	}
	if !strings.Contains(output, "buildspec.yml (aws)") {
		t.Errorf("expected aws workflow in dry run, got: %s", output)
	}

	// Verify no files were actually written
	ghPath := filepath.Join(tmpDir, ".github", "workflows", "sdd-check.yml")
	if _, err := os.Stat(ghPath); err == nil {
		t.Errorf("expected dry-run not to write %s, but file exists", ghPath)
	}
}

func TestCIInit_JSON(t *testing.T) {
	tmpDir := t.TempDir()
	var out, errOut bytes.Buffer
	env := IO{Out: &out, Err: &errOut}

	code := runCIInit([]string{"--platform", "github", "--dir", tmpDir, "--dry-run", "--json"}, env)
	if code != 0 {
		t.Fatalf("expected code 0, got %d (err: %s)", code, errOut.String())
	}

	var results []ci.ResultFile
	if err := json.Unmarshal(out.Bytes(), &results); err != nil {
		t.Fatalf("failed to parse JSON output: %v (raw: %s)", err, out.String())
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Platform != ci.PlatformGitHub {
		t.Errorf("expected platform github, got %s", results[0].Platform)
	}
	if !results[0].Created {
		t.Errorf("expected Created=true")
	}
}

func TestCIInit_CreateAndProtection(t *testing.T) {
	tmpDir := t.TempDir()
	var out, errOut bytes.Buffer
	env := IO{Out: &out, Err: &errOut}

	// 1. Create file
	code := runCIInit([]string{"--platform", "github", "--dir", tmpDir}, env)
	if code != 0 {
		t.Fatalf("expected code 0, got %d (err: %s)", code, errOut.String())
	}

	ghPath := filepath.Join(tmpDir, ".github", "workflows", "sdd-check.yml")
	if _, err := os.Stat(ghPath); err != nil {
		t.Fatalf("expected file to exist at %s: %v", ghPath, err)
	}

	// 2. Run again without force -> must skip
	out.Reset()
	errOut.Reset()
	code = runCIInit([]string{"--platform", "github", "--dir", tmpDir}, env)
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(out.String(), "Skipped") {
		t.Errorf("expected output to mention Skipped, got: %s", out.String())
	}

	// 3. Run with force -> replaces
	out.Reset()
	errOut.Reset()
	code = runCIInit([]string{"--platform", "github", "--dir", tmpDir, "--force"}, env)
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(out.String(), "Replaced") {
		t.Errorf("expected output to mention Replaced, got: %s", out.String())
	}
}
