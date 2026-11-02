package cli

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func copyFixtureDir(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copyFixtureDir failed: %v", err)
	}
}

func TestCLIAdoptMissingCoreDocs(t *testing.T) {
	emptyDir := t.TempDir()
	r := exec(t, "", false, "", "adopt", "--dir", emptyDir)
	if r.code != 1 || !strings.Contains(r.err, "missing required core documents") {
		t.Fatalf("expected missing core docs error, got: %+v", r)
	}
}

func TestCLIAdoptDryRun(t *testing.T) {
	srcKit := filepath.Join("..", "check", "testdata", "dbguard-kit")
	target := t.TempDir()
	copyFixtureDir(t, srcKit, target)

	r := exec(t, "", false, "", "adopt", "--dir", target, "--dry-run")
	if r.code != 0 {
		t.Fatalf("dry-run failed: %+v", r)
	}
	if !strings.Contains(r.out, "Dry run: nothing was written") {
		t.Errorf("expected dry run note, got:\n%s", r.out)
	}
	if !strings.Contains(r.out, `"project": "dbguard"`) {
		t.Errorf("expected project 'dbguard' in JSON output, got:\n%s", r.out)
	}
	// Verify no manifest was written
	if exists(target, ".sdd/manifest.json") {
		t.Error(".sdd/manifest.json was created during dry-run")
	}
}

func TestCLIAdoptDbguard(t *testing.T) {
	srcKit := filepath.Join("..", "check", "testdata", "dbguard-kit")
	target := t.TempDir()
	copyFixtureDir(t, srcKit, target)

	r := exec(t, "", false, "", "adopt", "--yes", "--dir", target)
	if r.code != 0 {
		t.Fatalf("adopt failed: %+v", r)
	}
	if !strings.Contains(r.out, "Adopted successfully") {
		t.Errorf("expected success message, got:\n%s", r.out)
	}
	if !exists(target, ".sdd/manifest.json") {
		t.Fatal(".sdd/manifest.json was not created")
	}

	// Now run sdd check on adopted kit
	rCheck := exec(t, "", false, "", "check", "--dir", target)
	if rCheck.code != 0 {
		t.Fatalf("check failed on adopted kit: %+v", rCheck)
	}
}

func TestCLIAdoptAlreadyAdoptedAndForce(t *testing.T) {
	srcKit := filepath.Join("..", "check", "testdata", "dbguard-kit")
	target := t.TempDir()
	copyFixtureDir(t, srcKit, target)

	r1 := exec(t, "", false, "", "adopt", "--yes", "--dir", target)
	if r1.code != 0 {
		t.Fatalf("first adopt failed: %+v", r1)
	}

	// Second adopt without --force fails
	r2 := exec(t, "", false, "", "adopt", "--yes", "--dir", target)
	if r2.code != 1 || !strings.Contains(r2.err, "already has a manifest") {
		t.Fatalf("expected already adopted error, got: %+v", r2)
	}

	// Third adopt with --force succeeds
	r3 := exec(t, "", false, "", "adopt", "--yes", "--force", "--dir", target)
	if r3.code != 0 {
		t.Fatalf("force adopt failed: %+v", r3)
	}
	if !strings.Contains(r3.out, "Backup:") {
		t.Errorf("expected Backup line in output, got:\n%s", r3.out)
	}
}

func TestCLIAdoptInteractiveAbort(t *testing.T) {
	srcKit := filepath.Join("..", "check", "testdata", "dbguard-kit")
	target := t.TempDir()
	copyFixtureDir(t, srcKit, target)

	var out, errb bytes.Buffer
	env := IO{
		In:          strings.NewReader("n\n"),
		Out:         &out,
		Err:         &errb,
		Interactive: true,
		Now:         fixedNow,
	}
	code := Run([]string{"adopt", "--dir", target}, env)
	if code != 1 {
		t.Fatalf("expected exit 1 on abort, got %d", code)
	}
	if !strings.Contains(out.String(), "Aborted") {
		t.Errorf("expected 'Aborted' in output, got:\n%s", out.String())
	}
	if exists(target, ".sdd/manifest.json") {
		t.Error("manifest created despite interactive abort")
	}
}
