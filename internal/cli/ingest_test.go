package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vikashkumar21/sdd/internal/context"
)

func TestIngestText(t *testing.T) {
	root := t.TempDir()
	var out, errBuf bytes.Buffer
	env := IO{
		Out:    &out,
		Err:    &errBuf,
		Now:    func() time.Time { return time.Date(2026, 10, 30, 10, 0, 0, 0, time.UTC) },
		Getenv: func(string) string { return "" },
	}

	code := Run([]string{"ingest", "--dir", root, "--text", "meeting transcript content", "--name", "kickoff"}, env)
	if code != 0 {
		t.Fatalf("runIngest failed with %d: %s", code, errBuf.String())
	}

	m, ok, err := context.ReadManifest(root)
	if err != nil || !ok {
		t.Fatalf("ReadManifest failed: ok=%v, err=%v", ok, err)
	}
	if len(m.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(m.Entries))
	}
	if !strings.Contains(m.Entries[0].Path, "kickoff") {
		t.Errorf("expected path to contain kickoff, got %s", m.Entries[0].Path)
	}
}

func TestIngestFiles(t *testing.T) {
	root := t.TempDir()
	srcFile := filepath.Join(t.TempDir(), "spec.md")
	if err := os.WriteFile(srcFile, []byte("# Project Spec\nDetails here"), 0o644); err != nil {
		t.Fatal(err)
	}

	var out, errBuf bytes.Buffer
	env := IO{
		Out: &out,
		Err: &errBuf,
		Now: func() time.Time { return time.Date(2026, 10, 30, 10, 0, 0, 0, time.UTC) },
	}

	code := Run([]string{"ingest", "--dir", root, "--file", srcFile}, env)
	if code != 0 {
		t.Fatalf("runIngest failed: %s", errBuf.String())
	}

	m, ok, err := context.ReadManifest(root)
	if err != nil || !ok {
		t.Fatal(err)
	}
	if len(m.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(m.Entries))
	}
	if m.Entries[0].Format != "markdown" {
		t.Errorf("expected markdown format, got %s", m.Entries[0].Format)
	}
}

func TestIngestStdin(t *testing.T) {
	root := t.TempDir()
	var out, errBuf bytes.Buffer
	stdin := strings.NewReader("piped input context")
	env := IO{
		In:          stdin,
		Out:         &out,
		Err:         &errBuf,
		Interactive: false,
		Now:         func() time.Time { return time.Date(2026, 10, 30, 10, 0, 0, 0, time.UTC) },
	}

	code := Run([]string{"ingest", "--dir", root, "--name", "stdin-notes"}, env)
	if code != 0 {
		t.Fatalf("runIngest stdin failed: %s", errBuf.String())
	}

	m, ok, err := context.ReadManifest(root)
	if err != nil || !ok {
		t.Fatal(err)
	}
	if len(m.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(m.Entries))
	}
	if m.Entries[0].Source != "stdin" {
		t.Errorf("expected source stdin, got %s", m.Entries[0].Source)
	}
}

func TestIngestTaskAssociation(t *testing.T) {
	root := t.TempDir()
	var out, errBuf bytes.Buffer
	env := IO{
		Out: &out,
		Err: &errBuf,
		Now: func() time.Time { return time.Date(2026, 10, 30, 10, 0, 0, 0, time.UTC) },
	}

	code := Run([]string{"ingest", "--dir", root, "--text", "task notes", "--task", "T-002"}, env)
	if code != 0 {
		t.Fatalf("runIngest task failed: %s", errBuf.String())
	}

	m, ok, err := context.ReadManifest(root)
	if err != nil || !ok {
		t.Fatal(err)
	}
	if len(m.Entries) != 1 || m.Entries[0].Task != "T-002" {
		t.Fatalf("expected entry with task T-002: %+v", m.Entries)
	}
	if !strings.Contains(m.Entries[0].Path, ".sdd/context/T-002/") {
		t.Errorf("expected path inside .sdd/context/T-002/: %s", m.Entries[0].Path)
	}
}
