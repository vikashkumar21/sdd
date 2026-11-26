package transcript

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSource_ValidateConversationID(t *testing.T) {
	valid := "b1711312-e6e1-4b0a-b515-891d9f6747ac"
	if err := ValidateConversationID(valid); err != nil {
		t.Errorf("valid UUID rejected: %v", err)
	}
	for _, bad := range []string{
		"",
		"../escape",
		"B1711312-E6E1-4B0A-B515-891D9F6747AC", // uppercase rejected
		"b1711312-e6e1-4b0a-b515-891d9f6747ag", // non-hex
		"b1711312e6e14b0ab515891d9f6747ac",     // missing dashes
		"b1711312-e6e1-4b0a-b515-891d9f6747ac/../../etc/passwd",
	} {
		if err := ValidateConversationID(bad); err == nil {
			t.Errorf("expected error for invalid ID %q", bad)
		}
	}
}

func TestSource_DefaultRoot(t *testing.T) {
	if _, err := DefaultRoot(""); err == nil {
		t.Errorf("expected error for empty home")
	}
	if _, err := DefaultRoot("   "); err == nil {
		t.Errorf("expected error for whitespace home")
	}
	got, err := DefaultRoot("/tmp/fakehome")
	if err != nil {
		t.Fatalf("DefaultRoot: %v", err)
	}
	want := filepath.Join("/tmp/fakehome", ".gemini", "antigravity", "brain")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSource_OpenConversation(t *testing.T) {
	root := t.TempDir()
	id := "11111111-2222-3333-4444-555555555555"
	logDir := filepath.Join(root, id, ".system_generated", "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatal(err)
	}
	tPath := filepath.Join(logDir, "transcript.jsonl")
	if err := os.WriteFile(tPath, []byte(`{"step_index":0}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	rc, err := OpenConversation(root, id)
	if err != nil {
		t.Fatalf("OpenConversation: %v", err)
	}
	data, _ := io.ReadAll(rc)
	_ = rc.Close()
	if !strings.Contains(string(data), "step_index") {
		t.Errorf("unexpected content: %s", data)
	}

	// Missing conversation
	if _, err := OpenConversation(root, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"); err == nil {
		t.Errorf("expected error for missing conversation")
	}

	// Symlink escape inside root rejected by os.OpenRoot
	outside := filepath.Join(t.TempDir(), "secret.jsonl")
	_ = os.WriteFile(outside, []byte(`{"step_index":0}`+"\n"), 0o644)
	symID := "22222222-2222-3333-4444-555555555555"
	symLogDir := filepath.Join(root, symID, ".system_generated", "logs")
	_ = os.MkdirAll(symLogDir, 0o755)
	if err := os.Symlink(outside, filepath.Join(symLogDir, "transcript.jsonl")); err == nil {
		if _, err := OpenConversation(root, symID); err == nil {
			t.Errorf("expected symlink escape under OpenRoot to be rejected")
		}
	}
}

func TestSource_OpenFileAndKey(t *testing.T) {
	dir := t.TempDir()
	fPath := filepath.Join(dir, "session.jsonl")
	if err := os.WriteFile(fPath, []byte(`{"step_index":0}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	rc, err := OpenFile(fPath)
	if err != nil {
		t.Fatalf("OpenFile: %v", err)
	}
	_ = rc.Close()

	// Directory rejected
	if _, err := OpenFile(dir); err == nil {
		t.Errorf("expected directory to be rejected")
	}

	// Symlink rejected
	symPath := filepath.Join(dir, "link.jsonl")
	if err := os.Symlink(fPath, symPath); err == nil {
		if _, err := OpenFile(symPath); err == nil {
			t.Errorf("expected symlink to be rejected")
		}
	}

	// Deterministic path-free key
	k1 := SourceKeyForPath(fPath)
	k2 := SourceKeyForPath(filepath.Join(dir, ".", "session.jsonl"))
	if k1 != k2 || !strings.HasPrefix(k1, "file:") || len(k1) != len("file:")+16 {
		t.Errorf("unexpected source keys: %q vs %q", k1, k2)
	}
	if strings.Contains(k1, dir) {
		t.Errorf("source key leaked directory path: %q", k1)
	}
}
