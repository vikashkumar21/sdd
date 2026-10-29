package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vikashkumar21/sdd/internal/detect"
	"github.com/vikashkumar21/sdd/internal/scaffold"
	"github.com/vikashkumar21/sdd/internal/templates"
)

var now = time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)

func generate(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	set, err := templates.New("")
	if err != nil {
		t.Fatal(err)
	}
	det, _ := detect.Detect(root)
	a := scaffold.Answers{Project: "demo", Mode: scaffold.ModeGreenfield, Languages: []string{"go"}}
	p, err := scaffold.BuildPlan(root, a, det, set, now, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := scaffold.Apply(p, now, ""); err != nil {
		t.Fatal(err)
	}
	return root
}

func edit(t *testing.T, root, rel string, fn func(string) string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(fn(string(data))), 0o644); err != nil {
		t.Fatal(err)
	}
}

func put(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func run(t *testing.T, root string) Report {
	t.Helper()
	r, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func has(r Report, sev Severity, substr string) bool {
	for _, f := range r.Findings {
		if f.Severity == sev && strings.Contains(f.Message+" "+f.File, substr) {
			return true
		}
	}
	return false
}

func dump(r Report) string {
	var sb strings.Builder
	for _, f := range r.Findings {
		sb.WriteString(string(f.Severity) + " " + f.File + " " + f.Message + "\n")
	}
	return sb.String()
}

func TestFreshKitHasNoErrors(t *testing.T) {
	r := run(t, generate(t))
	if r.Count(Error) != 0 {
		t.Fatalf("a freshly generated kit must have no errors:\n%s", dump(r))
	}
	if !has(r, Warn, "TODO(human)") {
		t.Errorf("expected a warning about TODO placeholders:\n%s", dump(r))
	}
	if has(r, Info, "changed since sdd generated") {
		t.Errorf("a freshly generated kit must show no drift:\n%s", dump(r))
	}
	if r.Failed(false) {
		t.Error("warnings alone must not fail a non-strict check")
	}
	if !r.Failed(true) {
		t.Error("strict mode must fail while TODO placeholders remain")
	}
}

func TestCheckIsDeterministic(t *testing.T) {
	root := generate(t)
	if dump(run(t, root)) != dump(run(t, root)) {
		t.Fatal("two runs gave different reports")
	}
}

// The dbguard kit was written by hand before this checker existed; it must pass.
func TestRealWorldDbguardKitPasses(t *testing.T) {
	r := run(t, filepath.Join("testdata", "dbguard-kit"))
	if r.Count(Error) != 0 {
		t.Fatalf("the dbguard kit must have no errors:\n%s", dump(r))
	}
}

func TestDetectsProblems(t *testing.T) {
	tests := []struct {
		name string
		mut  func(t *testing.T, root string)
		sev  Severity
		want string
	}{
		{"missing required file", func(t *testing.T, root string) {
			if err := os.Remove(filepath.Join(root, "MISSION.md")); err != nil {
				t.Fatal(err)
			}
		}, Error, "MISSION.md"},
		{"agent contract drift", func(t *testing.T, root string) {
			edit(t, root, "GEMINI.md", func(s string) string { return strings.Replace(s, "Never guess.", "Guess freely.", 1) })
		}, Error, "agent contract differs"},
		{"dangling decision reference", func(t *testing.T, root string) {
			edit(t, root, "ROADMAP.md", func(s string) string { return s + "\nSee D-099.\n" })
		}, Error, "D-099"},
		{"dangling task reference", func(t *testing.T, root string) {
			edit(t, root, "MISSION.md", func(s string) string { return s + "\nBlocked by T-042.\n" })
		}, Error, "T-042"},
		{"broken table", func(t *testing.T, root string) {
			edit(t, root, "SKILLS.md", func(s string) string {
				return strings.Replace(s, "| Date | Pitfall | Source / task |", "| Date | Pitfall | Source | task |", 1)
			})
		}, Error, "pipes"},
		{"unbalanced fence", func(t *testing.T, root string) {
			edit(t, root, "MISSION.md", func(s string) string { return s + "\n```\nunclosed\n" })
		}, Error, "unbalanced code fence"},
		{"missing feedback section", func(t *testing.T, root string) {
			edit(t, root, "SKILLS.md", func(s string) string {
				return strings.Replace(s, "## 8. Human feedback rules", "## 8. Something else", 1)
			})
		}, Error, "Human feedback rules"},
		{"bad status", func(t *testing.T, root string) {
			edit(t, root, "ROADMAP.md", func(s string) string { return strings.Replace(s, "| Not started |", "| Almost there |", 1) })
		}, Warn, "unrecognised status"},
		{"in-flight task without feature folder", func(t *testing.T, root string) {
			edit(t, root, "ROADMAP.md", func(s string) string { return strings.Replace(s, "| Not started |", "| Implementing |", 1) })
		}, Error, "has no features/T-001"},
		{"feature folder without task row", func(t *testing.T, root string) {
			for _, f := range []string{"requirements.md", "plan.md", "validation.md"} {
				put(t, root, "features/T-077-ghost/"+f, "# x\n")
			}
		}, Error, "T-077"},
		{"feature folder missing a file", func(t *testing.T, root string) {
			put(t, root, "features/T-001-a/requirements.md", "# x\n")
		}, Error, "missing plan.md"},
		{"bad folder name", func(t *testing.T, root string) {
			put(t, root, "features/Task_One/requirements.md", "# x\n")
		}, Error, "short-slug"},
		{"blank approval record past approval", func(t *testing.T, root string) {
			for _, f := range []string{"requirements.md", "plan.md", "validation.md"} {
				put(t, root, "features/T-001-a/"+f, "# x\nApproved by: ______\nNOT RUN\n")
			}
			edit(t, root, "ROADMAP.md", func(s string) string { return strings.Replace(s, "| Not started |", "| Awaiting review |", 1) })
		}, Warn, "approval record is still blank"},
		{"duplicate task id", func(t *testing.T, root string) {
			edit(t, root, "ROADMAP.md", func(s string) string {
				return strings.Replace(s, "\n| T-001 |", "\n| T-000 |", 1)
			})
		}, Error, "duplicate task ID"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := generate(t)
			tc.mut(t, root)
			r := run(t, root)
			if !has(r, tc.sev, tc.want) {
				t.Fatalf("expected %s containing %q, got:\n%s", tc.sev, tc.want, dump(r))
			}
		})
	}
}

func TestManagedBlockParityIgnoresTextOutsideTheBlock(t *testing.T) {
	root := generate(t)
	edit(t, root, "CLAUDE.md", func(s string) string { return "# My own header\n\nTeam notes.\n\n" + s + "\nMore notes.\n" })
	if r := run(t, root); r.Count(Error) != 0 {
		t.Fatalf("text outside the managed block must not affect parity:\n%s", dump(r))
	}
}

func TestEditedGeneratedFilesAreReportedAsInfoOnly(t *testing.T) {
	root := generate(t)
	edit(t, root, "MISSION.md", func(s string) string { return s + "\nMy edit.\n" })
	r := run(t, root)
	if !has(r, Info, "changed since sdd generated") || r.Count(Error) != 0 {
		t.Fatalf("expected an INFO note and no errors:\n%s", dump(r))
	}
}

func TestEmptyDirectoryReportsMissingFiles(t *testing.T) {
	r := run(t, t.TempDir())
	if r.Count(Error) < 5 {
		t.Fatalf("expected many missing-file errors:\n%s", dump(r))
	}
}

func TestNotADirectory(t *testing.T) {
	if _, err := Run(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Fatal("expected an error")
	}
}
