package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPathHygiene_Platforms(t *testing.T) {
	c := &checker{}
	text := `# Setup
Run command on macOS:
` + "```bash" + `
cd /Users/alice/projects/sdd
` + "```" + `
Run command on Linux:
/home/bob/work/sdd
Run command on Windows:
C:\Users\charlie\projects\sdd
Forward-slash Windows:
D:/Users/david/repo
File URI macOS:
file:///Users/eve/projects
File URI Windows:
file:///C:/Users/frank/workspace
Compound username:
/Users/john.doe/repo
`
	c.checkPathHygiene("README.md", text)

	expected := []struct {
		line int
		path string
	}{
		{9, `C:\Users\charlie`},
		{11, "D:/Users/david"},
		{15, "C:/Users/frank"},
		{4, "/Users/alice"},
		{7, "/home/bob"},
		{13, "/Users/eve"},
		{17, "/Users/john.doe"},
	}

	if len(c.findings) != len(expected) {
		t.Fatalf("expected %d findings, got %d: %+v", len(expected), len(c.findings), c.findings)
	}

	for _, exp := range expected {
		found := false
		for _, f := range c.findings {
			if f.Line == exp.line && strings.Contains(f.Message, exp.path) {
				found = true
				if f.Severity != Warn {
					t.Errorf("line %d: expected Warn severity, got %v", exp.line, f.Severity)
				}
				if f.Rule != "path-hygiene" {
					t.Errorf("line %d: expected rule 'path-hygiene', got %s", exp.line, f.Rule)
				}
				break
			}
		}
		if !found {
			t.Errorf("expected finding on line %d for path '%s' not found in %+v", exp.line, exp.path, c.findings)
		}
	}
}

func TestPathHygiene_Exemptions(t *testing.T) {
	c := &checker{}
	text := `# Legitimate Paths
CI Runner: /home/runner/work/sdd
Example username: /Users/username/projects
Generic user: /Users/user/projects
Yourname placeholder: /Users/yourname/app
Example name: /Users/example/tool
Admin user: C:\Users\admin\Desktop
Default profile: C:\Users\default\profile
Windows Public profile: C:\Users\Public\Documents
Ellipsis: /Users/... or /home/... or C:\Users\...
Trailing ellipsis: /Users/.../project
Rule citation: Rule F-002 mandates zero personal paths (/Users/alice/test)
Path hygiene citation: See path-hygiene check for details (/Users/bob/test)
Path hygiene test citation: TestPathHygiene asserts (/Users/charlie/test)
Web URL: https://github.com/Users/alice/repo
`
	c.checkPathHygiene("docs.md", text)

	if len(c.findings) != 0 {
		t.Fatalf("expected 0 findings for exempt paths and rule citations, got %d: %+v", len(c.findings), c.findings)
	}
}

func TestPathHygiene_WarningFormat(t *testing.T) {
	c := &checker{}
	text := `line 1
line 2: /Users/john/repo
`
	c.checkPathHygiene("sub/doc.md", text)

	if len(c.findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(c.findings))
	}

	f := c.findings[0]
	if f.File != "sub/doc.md" {
		t.Errorf("expected file 'sub/doc.md', got '%s'", f.File)
	}
	if f.Line != 2 {
		t.Errorf("expected line 2, got %d", f.Line)
	}
	if f.Severity != Warn {
		t.Errorf("expected Warn severity, got %v", f.Severity)
	}
	if f.Rule != "path-hygiene" {
		t.Errorf("expected rule 'path-hygiene', got '%s'", f.Rule)
	}
	wantMsg := "local developer home path detected (/Users/john); use repository-relative paths or generic clone URLs per Rule F-002"
	if f.Message != wantMsg {
		t.Errorf("expected message:\n  %s\ngot:\n  %s", wantMsg, f.Message)
	}
}

func TestPathHygiene_MarkdownScan(t *testing.T) {
	root := t.TempDir()

	// Seed minimum valid kit structure
	for _, f := range coreDocs {
		if err := os.WriteFile(filepath.Join(root, f), []byte("# "+f+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, a := range []string{"CLAUDE.md", "GEMINI.md"} {
		if err := os.WriteFile(filepath.Join(root, a), []byte("# "+a+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Create root file with personal path
	rootContent := "# Guide\nRun `/Users/dev1/bin/sdd` to execute.\n"
	if err := os.WriteFile(filepath.Join(root, "GUIDE.md"), []byte(rootContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create task folder with personal path
	taskDir := filepath.Join(root, "tasks", "features", "T-001-test")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatal(err)
	}
	taskContent := "# Requirements\nConfig is in `/home/dev2/.config`\n"
	if err := os.WriteFile(filepath.Join(taskDir, "requirements.md"), []byte(taskContent), 0o644); err != nil {
		t.Fatal(err)
	}

	r, err := Run(root)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	var pathFindings []Finding
	for _, f := range r.Findings {
		if f.Rule == "path-hygiene" {
			pathFindings = append(pathFindings, f)
		}
	}

	if len(pathFindings) != 2 {
		t.Fatalf("expected 2 path-hygiene findings across scanned markdown files, got %d: %+v", len(pathFindings), pathFindings)
	}

	if pathFindings[0].File != "GUIDE.md" || pathFindings[0].Line != 2 {
		t.Errorf("unexpected finding 0: %+v", pathFindings[0])
	}
	if pathFindings[1].File != filepath.ToSlash(filepath.Join("tasks", "features", "T-001-test", "requirements.md")) || pathFindings[1].Line != 2 {
		t.Errorf("unexpected finding 1: %+v", pathFindings[1])
	}
}

func TestPathHygiene_Suppression(t *testing.T) {
	root := t.TempDir()

	// Seed minimum valid kit structure
	for _, f := range coreDocs {
		if err := os.WriteFile(filepath.Join(root, f), []byte("# "+f+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, a := range []string{"CLAUDE.md", "GEMINI.md"} {
		if err := os.WriteFile(filepath.Join(root, a), []byte("# "+a+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Add file with personal path
	guide := "# Guide\nRun `/Users/dev1/bin/sdd`\n"
	if err := os.WriteFile(filepath.Join(root, "GUIDE.md"), []byte(guide), 0o644); err != nil {
		t.Fatal(err)
	}

	// Verify unsuppressed finding
	r1, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range r1.Findings {
		if f.Rule == "path-hygiene" && !f.Suppressed {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected unsuppressed path-hygiene finding")
	}

	// Add suppression waiver
	sddDir := filepath.Join(root, ".sdd")
	if err := os.MkdirAll(sddDir, 0o755); err != nil {
		t.Fatal(err)
	}
	suppJSON := `{
  "schema": 1,
  "suppressions": [
    {
      "file": "GUIDE.md",
      "rule": "path-hygiene",
      "reason": "intentional historical reference"
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(sddDir, "suppressions.json"), []byte(suppJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	// Verify suppressed finding
	r2, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}

	suppressedFound := false
	for _, f := range r2.Findings {
		if f.Rule == "path-hygiene" {
			if f.Suppressed && strings.Contains(f.Message, "(suppressed: intentional historical reference)") {
				suppressedFound = true
			} else {
				t.Errorf("expected finding to be suppressed, got: %+v", f)
			}
		}
	}
	if !suppressedFound {
		t.Fatalf("expected suppressed path-hygiene finding in %+v", r2.Findings)
	}
}
