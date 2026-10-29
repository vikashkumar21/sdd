package detect

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDetectEmptyIsGreenfield(t *testing.T) {
	res, err := Detect(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if res.SuggestedMode != "greenfield" || len(res.Languages) != 0 || res.SourceFiles != 0 {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestDetectLanguages(t *testing.T) {
	tests := []struct {
		name      string
		files     map[string]string
		wantLangs []string
		wantOther []string
		wantBuild string
		wantTest  string
	}{
		{"go", map[string]string{"go.mod": "module x\n", "main.go": "package main\n"},
			[]string{"go"}, nil, "go build ./...", "go test ./..."},
		{"java maven wrapper", map[string]string{"pom.xml": "<project/>", "mvnw": "#!/bin/sh", "src/A.java": "class A{}"},
			[]string{"java"}, nil, "./mvnw -q package", "./mvnw test"},
		{"java gradle wrapper", map[string]string{"build.gradle": "", "gradlew": "", "src/A.java": "class A{}"},
			[]string{"java"}, nil, "./gradlew build", "./gradlew test"},
		{"python with tests dir", map[string]string{"pyproject.toml": "[project]\n", "tests/test_a.py": "", "a.py": ""},
			[]string{"python"}, nil, "", "pytest"},
		{"typescript by tsconfig", map[string]string{"package.json": `{"scripts":{"build":"tsc","test":"vitest","lint":"eslint ."}}`, "tsconfig.json": "{}", "src/a.ts": ""},
			[]string{"typescript"}, nil, "npm run build", "npm test"},
		{"typescript by dependency", map[string]string{"package.json": `{"devDependencies":{"typescript":"^5"}}`},
			[]string{"typescript"}, nil, "", ""},
		{"plain javascript", map[string]string{"package.json": `{"scripts":{"test":"jest"}}`, "index.js": ""},
			nil, []string{"javascript"}, "", ""},
		{"rust is other", map[string]string{"Cargo.toml": "[package]\n", "src/main.rs": "fn main(){}"},
			nil, []string{"rust"}, "", ""},
		{"mixed go and python", map[string]string{"go.mod": "module x\n", "a.go": "", "scripts/tool.py": "", "requirements.txt": ""},
			[]string{"go", "python"}, nil, "go build ./...", "go test ./..."},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			for rel, content := range tc.files {
				write(t, root, rel, content)
			}
			res, err := Detect(root)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(res.Languages, tc.wantLangs) && !(len(res.Languages) == 0 && len(tc.wantLangs) == 0) {
				t.Errorf("languages = %v, want %v", res.Languages, tc.wantLangs)
			}
			if !reflect.DeepEqual(res.Other, tc.wantOther) && !(len(res.Other) == 0 && len(tc.wantOther) == 0) {
				t.Errorf("other = %v, want %v", res.Other, tc.wantOther)
			}
			if res.Build != tc.wantBuild || res.Test != tc.wantTest {
				t.Errorf("build/test = %q/%q, want %q/%q", res.Build, res.Test, tc.wantBuild, tc.wantTest)
			}
		})
	}
}

func TestDetectModeFromSources(t *testing.T) {
	root := t.TempDir()
	write(t, root, "README.md", "# hi\n")
	res, _ := Detect(root)
	if res.SuggestedMode != "greenfield" {
		t.Fatalf("docs only should be greenfield, got %s", res.SuggestedMode)
	}
	write(t, root, "main.go", "package main\n")
	res, _ = Detect(root)
	if res.SuggestedMode != "brownfield" || res.SourceFiles != 1 {
		t.Fatalf("source present should be brownfield, got %+v", res)
	}
}

func TestDetectIgnoresVendoredDirs(t *testing.T) {
	root := t.TempDir()
	for _, f := range []string{"node_modules/a/x.js", "vendor/y.go", "target/Z.java", ".venv/lib/q.py", "build/o.go"} {
		write(t, root, f, "")
	}
	res, _ := Detect(root)
	if res.SourceFiles != 0 || res.SuggestedMode != "greenfield" {
		t.Fatalf("ignored dirs leaked into the scan: %+v", res)
	}
}

func TestDetectMakeTargetsWinOverDefaults(t *testing.T) {
	root := t.TempDir()
	write(t, root, "go.mod", "module x\n")
	write(t, root, "a.go", "")
	write(t, root, "Makefile", "build:\n\tgo build\n\ntest:\n\tgo test\n\nlint:\n\techo\nfmt:\n\techo\n.PHONY: build\nVAR := x\n")
	res, _ := Detect(root)
	if res.Build != "make build" || res.Test != "make test" || res.Lint != "make lint" || res.Format != "make fmt" {
		t.Fatalf("make targets not preferred: %+v", res)
	}
	want := []string{"build", "fmt", "lint", "test"}
	if !reflect.DeepEqual(res.MakeTargets, want) {
		t.Fatalf("targets = %v, want %v", res.MakeTargets, want)
	}
}

func TestDetectExistingAndCI(t *testing.T) {
	root := t.TempDir()
	write(t, root, "README.md", "x")
	write(t, root, "CLAUDE.md", "x")
	write(t, root, ".github/workflows/ci.yml", "x")
	res, _ := Detect(root)
	if !res.Existing["README.md"] || !res.Existing["CLAUDE.md"] || res.Existing["GEMINI.md"] {
		t.Fatalf("existing map wrong: %v", res.Existing)
	}
	if !reflect.DeepEqual(res.CI, []string{"GitHub Actions"}) {
		t.Fatalf("ci = %v", res.CI)
	}
}

func TestDetectNotADirectory(t *testing.T) {
	root := t.TempDir()
	write(t, root, "f", "x")
	if _, err := Detect(filepath.Join(root, "f")); err == nil {
		t.Fatal("expected error for a file path")
	}
	if _, err := Detect(filepath.Join(root, "missing")); err == nil {
		t.Fatal("expected error for a missing path")
	}
}

func TestDetectIsDeterministic(t *testing.T) {
	root := t.TempDir()
	write(t, root, "go.mod", "module x\n")
	write(t, root, "a/a.go", "")
	write(t, root, "b/b.py", "")
	a, _ := Detect(root)
	b, _ := Detect(root)
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("two scans differ:\n%+v\n%+v", a, b)
	}
}
