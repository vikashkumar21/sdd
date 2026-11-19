package ci

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParsePlatform(t *testing.T) {
	tests := []struct {
		input   string
		want    Platform
		wantErr bool
	}{
		{"github", PlatformGitHub, false},
		{"GH", PlatformGitHub, false},
		{"actions", PlatformGitHub, false},
		{"gitlab", PlatformGitLab, false},
		{"GL", PlatformGitLab, false},
		{"bitbucket", PlatformBitbucket, false},
		{"azure", PlatformAzure, false},
		{"azure-devops", PlatformAzure, false},
		{"gcp", PlatformGCP, false},
		{"cloudbuild", PlatformGCP, false},
		{"aws", PlatformAWS, false},
		{"codebuild", PlatformAWS, false},
		{"all", PlatformAll, false},
		{"unknown-platform", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		got, err := ParsePlatform(tt.input)
		if tt.wantErr && err == nil {
			t.Errorf("ParsePlatform(%q) expected error, got nil", tt.input)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("ParsePlatform(%q) unexpected error: %v", tt.input, err)
		}
		if got != tt.want {
			t.Errorf("ParsePlatform(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestGenerateWorkflows_AllPlatforms(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		Platform: PlatformAll,
		Project:  "test-app",
		Language: "go",
		BuildCmd: "go build -v ./...",
		LintCmd:  "make lint",
		TestCmd:  "go test -v ./...",
	}

	files, err := GenerateWorkflows(tmpDir, cfg)
	if err != nil {
		t.Fatalf("GenerateWorkflows failed: %v", err)
	}

	if len(files) != 6 {
		t.Fatalf("expected 6 files, got %d", len(files))
	}

	expectedPaths := map[Platform]string{
		PlatformGitHub:    ".github/workflows/sdd-check.yml",
		PlatformGitLab:    ".gitlab-ci.yml",
		PlatformBitbucket: "bitbucket-pipelines.yml",
		PlatformAzure:     "azure-pipelines.yml",
		PlatformGCP:       "cloudbuild.yaml",
		PlatformAWS:       "buildspec.yml",
	}

	for _, f := range files {
		wantPath := expectedPaths[f.Platform]
		if f.Path != wantPath {
			t.Errorf("platform %s path = %q, want %q", f.Platform, f.Path, wantPath)
		}
		if !f.Created {
			t.Errorf("platform %s expected Created=true", f.Platform)
		}

		// Verify key invariants in content
		if !strings.Contains(f.Content, "sdd check --strict") {
			t.Errorf("platform %s missing 'sdd check --strict'", f.Platform)
		}
		if !strings.Contains(f.Content, "go build -v ./...") {
			t.Errorf("platform %s missing compile command", f.Platform)
		}
		if !strings.Contains(f.Content, "Publish") {
			t.Errorf("platform %s missing publish placeholder", f.Platform)
		}
	}
}

func TestGenerateWorkflows_Polyglot(t *testing.T) {
	tmpDir := t.TempDir()

	languages := []struct {
		lang       string
		wantBuild  string
		wantSetup  string
		wantDocker string
	}{
		{"python", "python -m compileall", "python-version: '3.12'", "Publish Docker"},
		{"typescript", "npm run build", "node-version: '20'", "Publish Docker"},
		{"java", "./mvnw compile -B", "distribution: 'temurin'", "Publish Docker"},
		{"go", "go build -v ./...", "go-version: '1.27'", "Publish Docker"},
	}

	for _, tc := range languages {
		t.Run(tc.lang, func(t *testing.T) {
			cfg := Config{
				Platform: PlatformGitHub,
				Project:  "poly-app",
				Language: tc.lang,
			}
			files, err := GenerateWorkflows(tmpDir, cfg)
			if err != nil {
				t.Fatalf("failed to generate for %s: %v", tc.lang, err)
			}
			if len(files) != 1 {
				t.Fatalf("expected 1 file, got %d", len(files))
			}
			content := files[0].Content
			if !strings.Contains(content, tc.wantBuild) {
				t.Errorf("%s missing build %q", tc.lang, tc.wantBuild)
			}
			if !strings.Contains(content, tc.wantSetup) {
				t.Errorf("%s missing setup %q", tc.lang, tc.wantSetup)
			}
			if !strings.Contains(content, tc.wantDocker) {
				t.Errorf("%s missing docker placeholder", tc.lang)
			}
		})
	}
}

func TestApplyWorkflows_Protection(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		Platform: PlatformGitHub,
		Project:  "guard-app",
		Language: "go",
	}

	// 1. Initial generation and apply
	files, err := GenerateWorkflows(tmpDir, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyWorkflows(tmpDir, files, false); err != nil {
		t.Fatal(err)
	}

	targetPath := filepath.Join(tmpDir, filepath.FromSlash(files[0].Path))
	originalData, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}

	// 2. Modify existing file to simulate human customization
	customizedData := string(originalData) + "\n# Custom human step\n"
	if err := os.WriteFile(targetPath, []byte(customizedData), 0o644); err != nil {
		t.Fatal(err)
	}

	// 3. Second generation without force -> must report Skipped
	files2, err := GenerateWorkflows(tmpDir, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !files2[0].Skipped {
		t.Errorf("expected Skipped=true on existing file without force")
	}

	if err := ApplyWorkflows(tmpDir, files2, false); err != nil {
		t.Fatal(err)
	}
	// Verify content remains customized
	afterData, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(afterData) != customizedData {
		t.Errorf("expected file to be preserved without force, but was overwritten")
	}

	// 4. Third generation WITH force -> must report Modified and overwrite
	cfg.Force = true
	files3, err := GenerateWorkflows(tmpDir, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !files3[0].Modified {
		t.Errorf("expected Modified=true with force")
	}
	if err := ApplyWorkflows(tmpDir, files3, true); err != nil {
		t.Fatal(err)
	}
	forcedData, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(forcedData), "# Custom human step") {
		t.Errorf("expected force overwrite to replace customized data")
	}
}
