package feature

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vikashkumar21/sdd/internal/detect"
	"github.com/vikashkumar21/sdd/internal/roadmap"
	"github.com/vikashkumar21/sdd/internal/scaffold"
	"github.com/vikashkumar21/sdd/internal/templates"
)

func kit(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	set, err := templates.New("")
	if err != nil {
		t.Fatal(err)
	}
	det, _ := detect.Detect(root)
	now := time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)
	p, err := scaffold.BuildPlan(root, scaffold.Answers{Project: "demo", Mode: scaffold.ModeGreenfield, Languages: []string{"go"}}, det, set, now, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := scaffold.Apply(p, now, ""); err != nil {
		t.Fatal(err)
	}
	return root
}

func read(t *testing.T, root, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestNewCreatesFolderFromProjectTemplates(t *testing.T) {
	root := kit(t)
	res, err := New(root, "T-001", "define-scope", "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Dir != "features/T-001-define-scope" || len(res.Files) != 3 || !res.StatusUpdated {
		t.Fatalf("unexpected result: %+v", res)
	}
	req := read(t, root, "features/T-001-define-scope/requirements.md")
	if !strings.HasPrefix(req, "# Requirements — T-001 Define mission and scope (docs only)") {
		t.Errorf("title not taken from the roadmap row: %q", strings.SplitN(req, "\n", 2)[0])
	}
	if strings.Contains(req, "T-xxx") || strings.Contains(read(t, root, "features/T-001-define-scope/plan.md"), "T-xxx") {
		t.Error("T-xxx tokens were not replaced")
	}
	if !strings.Contains(read(t, root, "features/T-001-define-scope/plan.md"), "`APPROVED T-001`") {
		t.Error("approval phrase should name the real task")
	}
	for _, task := range roadmap.Parse(read(t, root, "ROADMAP.md")) {
		if task.ID == "T-001" && roadmap.Kind(task.Status) != "specifying" {
			t.Errorf("T-001 status = %q, want Specifying", task.Status)
		}
	}
}

func TestExplicitTitleWins(t *testing.T) {
	root := kit(t)
	if _, err := New(root, "T-001", "scope", "My custom title"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(read(t, root, "features/T-001-scope/validation.md"), "# Validation — T-001 My custom title") {
		t.Error("explicit title not used")
	}
}

func TestNewRefusesBadInput(t *testing.T) {
	root := kit(t)
	tests := []struct{ name, id, slug string }{
		{"bad id", "T-1", "x"},
		{"bootstrap task", "T-000", "x"},
		{"bad slug upper", "T-001", "Bad Slug"},
		{"bad slug underscore", "T-001", "bad_slug"},
		{"unknown task", "T-042", "x"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := New(root, tc.id, tc.slug, ""); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
	if entries, _ := os.ReadDir(filepath.Join(root, "features")); len(entries) != 1 {
		t.Errorf("failed attempts must not create folders, found %d entries", len(entries))
	}
}

func TestNewRefusesSecondFolderForSameTask(t *testing.T) {
	root := kit(t)
	if _, err := New(root, "T-001", "first", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := New(root, "T-001", "second", ""); err == nil {
		t.Fatal("expected an error for a second folder")
	}
}

func TestStatusOnlyChangesFromNotStarted(t *testing.T) {
	root := kit(t)
	p := filepath.Join(root, "ROADMAP.md")
	data, _ := os.ReadFile(p)
	edited := strings.Replace(string(data), "| Not started |", "| Blocked (waiting on a decision) |", 1)
	if err := os.WriteFile(p, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := New(root, "T-001", "scope", "")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusUpdated {
		t.Error("a Blocked task's status must not be changed")
	}
	if !strings.Contains(read(t, root, "ROADMAP.md"), "Blocked (waiting on a decision)") {
		t.Error("ROADMAP.md was modified")
	}
}

func TestMissingTemplatesIsAnError(t *testing.T) {
	root := kit(t)
	if err := os.Remove(filepath.Join(root, "features", "_templates", "plan.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := New(root, "T-001", "scope", ""); err == nil {
		t.Fatal("expected an error")
	}
	if _, err := os.Stat(filepath.Join(root, "features", "T-001-scope")); err == nil {
		t.Error("nothing may be created when a template is missing")
	}
}

func TestNewConfinedRoot(t *testing.T) {
	nonExistent := filepath.Join(t.TempDir(), "does-not-exist")
	if _, err := New(nonExistent, "T-001", "scope", ""); err == nil {
		t.Fatal("expected error opening non-existent root, got nil")
	}
}
