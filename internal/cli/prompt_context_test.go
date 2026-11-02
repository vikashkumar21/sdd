package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/vikashkumar21/sdd/internal/context"
)

func TestPromptWithoutContext(t *testing.T) {
	root := t.TempDir()
	exec(t, "", false, "", "init", "--yes", "--dir", root, "--name", "nocontext")

	p := exec(t, "", false, "", "prompt", "gemini", "--dir", root)
	if p.code != 0 {
		t.Fatalf("prompt failed: %+v", p)
	}

	if strings.Contains(p.out, "Staged external context") {
		t.Errorf("prompt should omit context section when no context is staged:\n%s", p.out)
	}
	if strings.Contains(p.out, "{{") || strings.Contains(p.out, "<no value>") {
		t.Errorf("prompt has template leftovers")
	}
}

func TestPromptWithStagedContext(t *testing.T) {
	root := t.TempDir()
	exec(t, "", false, "", "init", "--yes", "--dir", root, "--name", "withcontext")

	item := context.Item{
		Source: "raw-text",
		Origin: "meeting-transcript",
		Data:   []byte("Client requirements discussion"),
	}
	if _, err := context.Stage(root, item, time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}

	p := exec(t, "", false, "", "prompt", "claude", "--dir", root)
	if p.code != 0 {
		t.Fatalf("prompt failed: %+v", p)
	}

	for _, want := range []string{
		"Staged external context (.sdd/context/)",
		"meeting-transcript",
		"Extraction and synthesis instructions:",
		"MISSION.md",
		"TECH_STACK.md",
		"ROADMAP.md",
		"GATE 1 IS STRICT",
	} {
		if !strings.Contains(p.out, want) {
			t.Errorf("prompt output missing expected section %q:\n%s", want, p.out)
		}
	}
	if strings.Contains(p.out, "{{") || strings.Contains(p.out, "<no value>") {
		t.Errorf("prompt has template leftovers")
	}
}

func TestPromptWithTaskContext(t *testing.T) {
	root := t.TempDir()
	exec(t, "", false, "", "init", "--yes", "--dir", root, "--name", "taskcontext")

	item := context.Item{
		Source: "file",
		Origin: "jira-101.json",
		Task:   "T-001",
		Data:   []byte(`{"issue": "T-001", "summary": "Initial requirements"}`),
	}
	if _, err := context.Stage(root, item, time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}

	p := exec(t, "", false, "", "prompt", "gemini", "--dir", root, "--task", "T-001")
	if p.code != 0 {
		t.Fatalf("prompt failed: %+v", p)
	}

	for _, want := range []string{
		"Staged external context (.sdd/context/)",
		"Context assigned to focus task (T-001):",
		"jira-101.json",
		"features/T-xxx-<slug>/requirements.md",
	} {
		if !strings.Contains(p.out, want) {
			t.Errorf("prompt output missing expected section %q:\n%s", want, p.out)
		}
	}
}
