package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestPassportCLI(t *testing.T) {
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("resolving repo root: %v", err)
	}

	t.Run("default markdown output", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		env := IO{Out: &stdout, Err: &stderr}
		code := runPassport([]string{"--dir", repoRoot}, env)
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d, stderr: %s", code, stderr.String())
		}
		out := stdout.String()
		if !strings.Contains(out, "# sdd — Project Passport") {
			t.Errorf("expected passport header in output, got:\n%s", out)
		}
		if !strings.Contains(out, "Living Documents (On-Demand)") {
			t.Errorf("expected living docs section in output, got:\n%s", out)
		}
	})

	t.Run("json output", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		env := IO{Out: &stdout, Err: &stderr}
		code := runPassport([]string{"--dir", repoRoot, "--json"}, env)
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d, stderr: %s", code, stderr.String())
		}
		out := stdout.String()
		if !strings.Contains(out, `"project_name": "sdd"`) {
			t.Errorf("expected json project_name in output, got:\n%s", out)
		}
	})

	t.Run("unexpected argument exits 2", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		env := IO{Out: &stdout, Err: &stderr}
		code := runPassport([]string{"unexpected_arg"}, env)
		if code != 2 {
			t.Errorf("expected exit code 2, got %d", code)
		}
		if !strings.Contains(stderr.String(), "unexpected argument") {
			t.Errorf("expected error message in stderr, got: %s", stderr.String())
		}
	})

	t.Run("missing directory exits 1", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		env := IO{Out: &stdout, Err: &stderr}
		code := runPassport([]string{"--dir", filepath.Join(repoRoot, "nonexistent-dir-12345")}, env)
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
	})
}

func TestStatusCompactCLI(t *testing.T) {
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("resolving repo root: %v", err)
	}

	t.Run("status --compact renders markdown passport", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		env := IO{Out: &stdout, Err: &stderr}
		code := runStatus([]string{"--dir", repoRoot, "--compact"}, env)
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d, stderr: %s", code, stderr.String())
		}
		out := stdout.String()
		if !strings.Contains(out, "# sdd — Project Passport") {
			t.Errorf("expected passport header, got:\n%s", out)
		}
	})

	t.Run("status --compact --json renders json passport", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		env := IO{Out: &stdout, Err: &stderr}
		code := runStatus([]string{"--dir", repoRoot, "--compact", "--json"}, env)
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d, stderr: %s", code, stderr.String())
		}
		out := stdout.String()
		if !strings.Contains(out, `"project_name": "sdd"`) {
			t.Errorf("expected json project_name, got:\n%s", out)
		}
	})
}
