package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vikashkumar21/sdd/internal/roadmap"
)

func TestCheckPassport_FreshKit(t *testing.T) {
	root := generate(t)

	r := run(t, root)
	for _, f := range r.Findings {
		if strings.HasPrefix(f.Rule, "passport-") {
			t.Errorf("unexpected passport finding on fresh kit: %+v", f)
		}
	}
}

func TestCheckPassport_MissingRoadmap(t *testing.T) {
	emptyDir := t.TempDir()
	rEmpty, err := Run(emptyDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range rEmpty.Findings {
		if strings.HasPrefix(f.Rule, "passport-") {
			t.Errorf("expected passport check to skip cleanly when ROADMAP.md is missing, got: %+v", f)
		}
	}
}

func TestCheckPassport_ActiveTaskSync(t *testing.T) {
	root := generate(t)

	// Set an active task in ROADMAP.md
	roadmapPath := filepath.Join(root, "ROADMAP.md")
	content, err := os.ReadFile(roadmapPath)
	if err != nil {
		t.Fatal(err)
	}
	newContent := strings.Replace(string(content), "| Not started |", "| Implementing |", 1)
	if err := os.WriteFile(roadmapPath, []byte(newContent), 0644); err != nil {
		t.Fatal(err)
	}

	c := &checker{root: root, texts: make(map[string]string)}
	tasks := c.checkRoadmap()
	c.checkPassportConsistency(tasks)

	for _, f := range c.findings {
		if strings.HasPrefix(f.Rule, "passport-") {
			t.Errorf("unexpected passport finding with synced active task: %+v", f)
		}
	}
}

func TestCheckPassport_ActiveTaskMismatch(t *testing.T) {
	root := generate(t)

	c := &checker{root: root, texts: make(map[string]string)}
	// Pass mismatched synthetic active task list
	mismatchedTasks := []roadmap.Task{
		{ID: "T-999", Title: "Fake task", Status: "Implementing"},
	}
	c.checkPassportConsistency(mismatchedTasks)

	var found bool
	for _, f := range c.findings {
		if f.Rule == "passport-tasks" && f.Severity == Error {
			found = true
			if !strings.Contains(f.Message, "does not match") && !strings.Contains(f.Message, "out of sync") {
				t.Errorf("unexpected message: %s", f.Message)
			}
		}
	}
	if !found {
		t.Errorf("expected passport-tasks Error finding when active tasks mismatch")
	}
}

func TestCheckPassport_GenerateError(t *testing.T) {
	root := t.TempDir()
	// Create ROADMAP.md as a directory so passport.Generate fails
	roadmapPath := filepath.Join(root, "ROADMAP.md")
	if err := os.Mkdir(roadmapPath, 0755); err != nil {
		t.Fatal(err)
	}

	c := &checker{root: root, texts: make(map[string]string)}
	tasks := []roadmap.Task{
		{ID: "T-001", Status: "Implementing"},
	}
	c.checkPassportConsistency(tasks)

	var found bool
	for _, f := range c.findings {
		if f.Rule == "passport-generate" && f.Severity == Error {
			found = true
		}
	}
	if !found {
		t.Errorf("expected passport-generate Error when ROADMAP.md cannot be read")
	}
}
