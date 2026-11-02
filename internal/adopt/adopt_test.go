package adopt

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vikashkumar21/sdd/internal/check"
	"github.com/vikashkumar21/sdd/internal/scaffold"
)

var fixedNow = time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)

func copyDir(t *testing.T, src, dst string) {
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
		t.Fatalf("copyDir failed: %v", err)
	}
}

func fileHashes(t *testing.T, root string) map[string]string {
	t.Helper()
	hashes := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, ".sdd/") {
			return nil // exclude manifest and backup directory
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		h := sha256.Sum256(data)
		hashes[rel] = hex.EncodeToString(h[:])
		return nil
	})
	if err != nil {
		t.Fatalf("fileHashes failed: %v", err)
	}
	return hashes
}

func TestAdoptMissingCoreDocs(t *testing.T) {
	emptyDir := t.TempDir()
	_, err := Adopt(emptyDir, Options{})
	if err == nil || !strings.Contains(err.Error(), "missing required core documents") {
		t.Fatalf("expected missing core documents error, got: %v", err)
	}

	// Test hints when alternative file names exist
	withAlt := t.TempDir()
	_ = os.WriteFile(filepath.Join(withAlt, "VISION.md"), []byte("# Vision\n"), 0o644)
	_ = os.WriteFile(filepath.Join(withAlt, "ARCHITECTURE.md"), []byte("# Architecture\n"), 0o644)
	_, errAlt := Adopt(withAlt, Options{})
	if errAlt == nil || !strings.Contains(errAlt.Error(), "found \"VISION.md\"") {
		t.Fatalf("expected hint for VISION.md, got: %v", errAlt)
	}
}

func TestAdoptDbguardKitAndCheck(t *testing.T) {
	srcKit := filepath.Join("..", "check", "testdata", "dbguard-kit")
	target := t.TempDir()
	copyDir(t, srcKit, target)

	// Ensure no manifest exists beforehand
	if _, err := os.Stat(filepath.Join(target, scaffold.ManifestPath)); err == nil {
		t.Fatal("manifest already exists in fixture")
	}

	res, err := Adopt(target, Options{Now: fixedNow})
	if err != nil {
		t.Fatalf("Adopt failed: %v", err)
	}

	if res.Project != "dbguard" {
		t.Errorf("expected project 'dbguard', got %q", res.Project)
	}

	// Verify manifest exists and has valid schema 1
	m, ok, err := scaffold.ReadManifest(target)
	if err != nil || !ok {
		t.Fatalf("ReadManifest failed: %v", err)
	}
	if m.Schema != 1 {
		t.Errorf("expected schema 1, got %d", m.Schema)
	}
	if m.Answers.Project != "dbguard" {
		t.Errorf("manifest project = %q, want 'dbguard'", m.Answers.Project)
	}

	// Verify sdd check passes on the adopted kit with 0 errors
	rep, err := check.Run(target)
	if err != nil {
		t.Fatalf("check.Run failed: %v", err)
	}
	if rep.Count(check.Error) != 0 {
		t.Fatalf("expected 0 errors on adopted kit, got %d: %+v", rep.Count(check.Error), rep.Findings)
	}
}

func TestAdoptZeroMutation(t *testing.T) {
	srcKit := filepath.Join("..", "check", "testdata", "dbguard-kit")
	target := t.TempDir()
	copyDir(t, srcKit, target)

	beforeHashes := fileHashes(t, target)

	_, err := Adopt(target, Options{Now: fixedNow})
	if err != nil {
		t.Fatalf("Adopt failed: %v", err)
	}

	afterHashes := fileHashes(t, target)

	if len(beforeHashes) != len(afterHashes) {
		t.Fatalf("file count mismatch before (%d) and after (%d)", len(beforeHashes), len(afterHashes))
	}
	for path, beforeHash := range beforeHashes {
		afterHash, ok := afterHashes[path]
		if !ok {
			t.Errorf("file disappeared after adopt: %s", path)
			continue
		}
		if beforeHash != afterHash {
			t.Errorf("file mutated during adopt: %s (hash changed from %s to %s)", path, beforeHash, afterHash)
		}
	}
}

func TestAdoptAlreadyAdopted(t *testing.T) {
	srcKit := filepath.Join("..", "check", "testdata", "dbguard-kit")
	target := t.TempDir()
	copyDir(t, srcKit, target)

	_, err := Adopt(target, Options{Now: fixedNow})
	if err != nil {
		t.Fatalf("first Adopt failed: %v", err)
	}

	// Second adopt without --force must fail
	_, errSecond := Adopt(target, Options{Now: fixedNow})
	if errSecond == nil || !strings.Contains(errSecond.Error(), "already has a manifest") {
		t.Fatalf("expected already adopted refusal, got: %v", errSecond)
	}

	// Third adopt with --force must succeed and create backup
	resForce, errForce := Adopt(target, Options{Force: true, Now: fixedNow.Add(time.Hour)})
	if errForce != nil {
		t.Fatalf("Adopt with --force failed: %v", errForce)
	}
	if resForce.BackupPath == "" {
		t.Error("expected BackupPath to be set under --force")
	}
}
