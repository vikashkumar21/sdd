package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vikashkumar21/sdd/internal/context"
)

func TestInitWithContext(t *testing.T) {
	root := t.TempDir()
	srcFile := filepath.Join(t.TempDir(), "transcript.txt")
	if err := os.WriteFile(srcFile, []byte("Client kickoff discussion transcript..."), 0o644); err != nil {
		t.Fatal(err)
	}

	var out, errBuf bytes.Buffer
	env := IO{
		Out: &out,
		Err: &errBuf,
		Now: func() time.Time { return time.Date(2026, 10, 30, 10, 0, 0, 0, time.UTC) },
	}

	args := []string{
		"init",
		"--yes",
		"--dir", root,
		"--name", "context-project",
		"--context-file", srcFile,
		"--context-text", "Initial architecture notes and scope constraints",
	}

	code := Run(args, env)
	if code != 0 {
		t.Fatalf("runInit failed with %d: %s", code, errBuf.String())
	}

	m, ok, err := context.ReadManifest(root)
	if err != nil || !ok {
		t.Fatalf("ReadManifest failed: ok=%v, err=%v", ok, err)
	}
	if len(m.Entries) != 2 {
		t.Fatalf("expected 2 context entries, got %d", len(m.Entries))
	}
}
