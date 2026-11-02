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
			if !reflect.DeepEqual(res.Languages, tc.wantLangs) && (len(res.Languages) != 0 || len(tc.wantLangs) != 0) {
				t.Errorf("languages = %v, want %v", res.Languages, tc.wantLangs)
			}
			if !reflect.DeepEqual(res.Other, tc.wantOther) && (len(res.Other) != 0 || len(tc.wantOther) != 0) {
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

func TestDetectMonorepoDepth(t *testing.T) {
	root := t.TempDir()
	// Depth 1: services/go.mod
	write(t, root, "services/go.mod", "module s\n")
	// Depth 2: apps/web/package.json
	write(t, root, "apps/web/package.json", `{"devDependencies":{"typescript":"^5"}}`)
	// Depth 3: backend/services/billing/pom.xml
	write(t, root, "backend/services/billing/pom.xml", "<project/>")
	// Depth 4: a/b/c/d/pyproject.toml
	write(t, root, "a/b/c/d/pyproject.toml", "[project]\n")
	// Depth 5: a/b/c/d/e/Cargo.toml (should be skipped because depth > 4)
	write(t, root, "a/b/c/d/e/Cargo.toml", "[package]\n")

	res, err := Detect(root)
	if err != nil {
		t.Fatal(err)
	}

	wantLangs := []string{"go", "java", "python", "typescript"}
	if !reflect.DeepEqual(res.Languages, wantLangs) {
		t.Errorf("languages = %v, want %v", res.Languages, wantLangs)
	}

	// Verify manifests recorded
	hasManifest := func(p string) bool {
		for _, m := range res.Manifests {
			if m == p {
				return true
			}
		}
		return false
	}
	if !hasManifest("services/go.mod") {
		t.Errorf("missing services/go.mod in %v", res.Manifests)
	}
	if !hasManifest("apps/web/package.json") {
		t.Errorf("missing apps/web/package.json in %v", res.Manifests)
	}
	if !hasManifest("backend/services/billing/pom.xml") {
		t.Errorf("missing backend/services/billing/pom.xml in %v", res.Manifests)
	}
	if !hasManifest("a/b/c/d/pyproject.toml") {
		t.Errorf("missing a/b/c/d/pyproject.toml in %v", res.Manifests)
	}
	if hasManifest("a/b/c/d/e/Cargo.toml") {
		t.Errorf("Cargo.toml at depth 5 should NOT be in %v", res.Manifests)
	}
}

func TestDetectNestedPackageJSON(t *testing.T) {
	root := t.TempDir()
	write(t, root, "apps/web/package.json", `{"scripts":{"build":"vite build","test":"vitest","lint":"eslint .","format":"prettier -w ."},"dependencies":{"typescript":"^5.0.0"}}`)

	res, err := Detect(root)
	if err != nil {
		t.Fatal(err)
	}

	wantLangs := []string{"typescript"}
	if !reflect.DeepEqual(res.Languages, wantLangs) {
		t.Fatalf("languages = %v, want %v", res.Languages, wantLangs)
	}

	if res.Build != "npm --prefix apps/web run build" {
		t.Errorf("build = %q, want %q", res.Build, "npm --prefix apps/web run build")
	}
	if res.Test != "npm --prefix apps/web test" {
		t.Errorf("test = %q, want %q", res.Test, "npm --prefix apps/web test")
	}
	if res.Lint != "npm --prefix apps/web run lint" {
		t.Errorf("lint = %q, want %q", res.Lint, "npm --prefix apps/web run lint")
	}
	if res.Format != "npm --prefix apps/web run format" {
		t.Errorf("format = %q, want %q", res.Format, "npm --prefix apps/web run format")
	}
}

func TestDetectPrunesHeavyDirs(t *testing.T) {
	root := t.TempDir()
	pruned := []string{
		".turbo/cache/x.json",
		".pnpm-store/v3/files/00/abc",
		"bower_components/lib/index.js",
		".cache/yarn/v6/file",
		".terraform/providers/aws",
		".yarn/cache/package.zip",
	}
	for _, p := range pruned {
		write(t, root, p, "")
	}
	write(t, root, "main.go", "package main\n")

	res, err := Detect(root)
	if err != nil {
		t.Fatal(err)
	}

	if res.SourceFiles != 1 {
		t.Errorf("source files = %d, want 1", res.SourceFiles)
	}
	if len(res.Other) != 0 {
		t.Errorf("other = %v, want empty", res.Other)
	}
}

func TestDetectPolyglotCommands(t *testing.T) {
	root := t.TempDir()
	write(t, root, "go.mod", "module x\n")
	write(t, root, "main.go", "package main\n")
	write(t, root, "apps/web/package.json", `{"scripts":{"build":"next build","test":"jest","lint":"eslint","format":"prettier"},"devDependencies":{"typescript":"^5"}}`)
	write(t, root, "services/worker/pyproject.toml", "[tool.pytest]\n")
	write(t, root, "services/worker/worker.py", "import pytest\n")

	res, err := Detect(root)
	if err != nil {
		t.Fatal(err)
	}

	wantLangs := []string{"go", "python", "typescript"}
	if !reflect.DeepEqual(res.Languages, wantLangs) {
		t.Fatalf("languages = %v, want %v", res.Languages, wantLangs)
	}

	wantBuild := "go build ./... && npm --prefix apps/web run build"
	if res.Build != wantBuild {
		t.Errorf("build = %q, want %q", res.Build, wantBuild)
	}

	wantTest := "go test ./... && pytest && npm --prefix apps/web test"
	if res.Test != wantTest {
		t.Errorf("test = %q, want %q", res.Test, wantTest)
	}

	wantLint := "go vet ./... && npm --prefix apps/web run lint"
	if res.Lint != wantLint {
		t.Errorf("lint = %q, want %q", res.Lint, wantLint)
	}

	wantFormat := "gofmt -l . && npm --prefix apps/web run format"
	if res.Format != wantFormat {
		t.Errorf("format = %q, want %q", res.Format, wantFormat)
	}
}

func TestDetectMakefilePrecedence(t *testing.T) {
	root := t.TempDir()
	write(t, root, "go.mod", "module x\n")
	write(t, root, "package.json", `{"scripts":{"test":"vitest","build":"tsc"},"devDependencies":{"typescript":"^5"}}`)
	write(t, root, "Makefile", "test:\n\tgo test && npm test\n")

	res, err := Detect(root)
	if err != nil {
		t.Fatal(err)
	}

	// Makefile has test, so test is "make test", but build falls back to combined
	if res.Test != "make test" {
		t.Errorf("test = %q, want %q", res.Test, "make test")
	}
	wantBuild := "go build ./... && npm run build"
	if res.Build != wantBuild {
		t.Errorf("build = %q, want %q", res.Build, wantBuild)
	}
}

func TestDetectCommandTableSafety(t *testing.T) {
	root := t.TempDir()
	// Malformed script names with pipe and backticks
	write(t, root, "package.json", `{"scripts":{"build":"echo 'build' | cat","test":"npm test `+"`bad`"+`"},"devDependencies":{"typescript":"^5"}}`)

	res, err := Detect(root)
	if err != nil {
		t.Fatal(err)
	}

	// Commands with pipe or backticks must be rejected to protect markdown tables
	if res.Build != "" {
		t.Errorf("build = %q, want empty due to unsafe character", res.Build)
	}
	if res.Test != "" {
		t.Errorf("test = %q, want empty due to unsafe character", res.Test)
	}
}
