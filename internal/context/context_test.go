package context

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var fixedTime = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)

func TestContextStageText(t *testing.T) {
	root := t.TempDir()
	item := Item{
		Source: "raw-text",
		Origin: "cli-input",
		Name:   "meeting-transcript",
		Data:   []byte("# Meeting Notes\nRequirements discussion..."),
	}

	entry, err := Stage(root, item, fixedTime)
	if err != nil {
		t.Fatalf("Stage failed: %v", err)
	}

	if entry.ID != "ctx-001" {
		t.Errorf("ID = %q, want ctx-001", entry.ID)
	}
	if entry.Source != "raw-text" {
		t.Errorf("Source = %q, want raw-text", entry.Source)
	}
	if !strings.HasPrefix(entry.Path, ".sdd/context/20261030-120000-meeting-transcript") {
		t.Errorf("unexpected path: %q", entry.Path)
	}

	// Read content back
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(entry.Path)))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !bytes.Equal(content, item.Data) {
		t.Fatalf("content mismatch")
	}

	// Verify manifest
	m, ok, err := ReadManifest(root)
	if err != nil || !ok {
		t.Fatalf("ReadManifest failed: ok=%v, err=%v", ok, err)
	}
	if len(m.Entries) != 1 {
		t.Fatalf("expected 1 entry in manifest, got %d", len(m.Entries))
	}
	if m.Entries[0].SHA256 != entry.SHA256 {
		t.Errorf("manifest hash mismatch")
	}
}

func TestContextStageTaskAssociation(t *testing.T) {
	root := t.TempDir()
	item := Item{
		Source: "file",
		Origin: "spec.json",
		Task:   "T-001",
		Data:   []byte(`{"key": "value"}`),
	}

	entry, err := Stage(root, item, fixedTime)
	if err != nil {
		t.Fatalf("Stage failed: %v", err)
	}

	if entry.Task != "T-001" {
		t.Errorf("Task = %q, want T-001", entry.Task)
	}
	if !strings.HasPrefix(entry.Path, ".sdd/context/T-001/") {
		t.Errorf("expected path to start with .sdd/context/T-001/, got %q", entry.Path)
	}
}

func TestInputSizeLimit(t *testing.T) {
	root := t.TempDir()
	oversized := make([]byte, MaxInputBytes+1)
	item := Item{
		Source: "raw-text",
		Data:   oversized,
	}

	if _, err := Stage(root, item, fixedTime); err == nil {
		t.Fatal("expected error for oversized input, got nil")
	}
}

func TestInvalidTaskID(t *testing.T) {
	root := t.TempDir()
	item := Item{
		Source: "raw-text",
		Task:   "TASK-123",
		Data:   []byte("test"),
	}

	if _, err := Stage(root, item, fixedTime); err == nil {
		t.Fatal("expected error for invalid task ID, got nil")
	}
}

func TestStagingMultipleItems(t *testing.T) {
	root := t.TempDir()

	item1 := Item{
		Source: "raw-text",
		Name:   "first",
		Data:   []byte("first context"),
	}
	entry1, err := Stage(root, item1, fixedTime)
	if err != nil {
		t.Fatal(err)
	}
	if entry1.ID != "ctx-001" {
		t.Errorf("entry1.ID = %q, want ctx-001", entry1.ID)
	}

	item2 := Item{
		Source: "file",
		Origin: "second.txt",
		Data:   []byte("second context"),
	}
	entry2, err := Stage(root, item2, fixedTime.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if entry2.ID != "ctx-002" {
		t.Errorf("entry2.ID = %q, want ctx-002", entry2.ID)
	}

	m, ok, err := ReadManifest(root)
	if err != nil || !ok {
		t.Fatal(err)
	}
	if len(m.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(m.Entries))
	}
}
