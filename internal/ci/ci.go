// Package ci generates and applies universal CI workflow configurations
// across GitHub Actions, GitLab CI, Bitbucket Pipelines, Azure DevOps,
// Google Cloud Build, and AWS CodeBuild.
package ci

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vikashkumar21/sdd/internal/detect"
	"github.com/vikashkumar21/sdd/internal/fsutil"
	"github.com/vikashkumar21/sdd/internal/scaffold"
	"github.com/vikashkumar21/sdd/internal/templates"
)

// Platform represents a supported CI/CD platform.
type Platform string

const (
	PlatformGitHub    Platform = "github"
	PlatformGitLab    Platform = "gitlab"
	PlatformBitbucket Platform = "bitbucket"
	PlatformAzure     Platform = "azure"
	PlatformGCP       Platform = "gcp"
	PlatformAWS       Platform = "aws"
	PlatformAll       Platform = "all"
)

// AllPlatforms lists all concrete platforms supported by sdd.
var AllPlatforms = []Platform{
	PlatformGitHub,
	PlatformGitLab,
	PlatformBitbucket,
	PlatformAzure,
	PlatformGCP,
	PlatformAWS,
}

// platformPaths maps platforms to their standard repository-relative file paths using forward slashes.
var platformPaths = map[Platform]string{
	PlatformGitHub:    ".github/workflows/sdd-check.yml",
	PlatformGitLab:    ".gitlab-ci.yml",
	PlatformBitbucket: "bitbucket-pipelines.yml",
	PlatformAzure:     "azure-pipelines.yml",
	PlatformGCP:       "cloudbuild.yaml",
	PlatformAWS:       "buildspec.yml",
}

// platformTemplates maps platforms to their embedded template names.
var platformTemplates = map[Platform]string{
	PlatformGitHub:    "ci/github.yml.tmpl",
	PlatformGitLab:    "ci/gitlab.yml.tmpl",
	PlatformBitbucket: "ci/bitbucket.yml.tmpl",
	PlatformAzure:     "ci/azure.yml.tmpl",
	PlatformGCP:       "ci/gcp.yml.tmpl",
	PlatformAWS:       "ci/aws.yml.tmpl",
}

// ParsePlatform parses a platform string into a validated Platform constant.
func ParsePlatform(name string) (Platform, error) {
	lower := strings.ToLower(strings.TrimSpace(name))
	switch lower {
	case "github", "gh", "actions":
		return PlatformGitHub, nil
	case "gitlab", "gl":
		return PlatformGitLab, nil
	case "bitbucket", "bb":
		return PlatformBitbucket, nil
	case "azure", "ado", "azure-devops":
		return PlatformAzure, nil
	case "gcp", "cloudbuild", "google":
		return PlatformGCP, nil
	case "aws", "codebuild", "amazon":
		return PlatformAWS, nil
	case "all":
		return PlatformAll, nil
	default:
		return "", fmt.Errorf("unknown platform %q; allowed platforms are github, gitlab, bitbucket, azure, gcp, aws, all", name)
	}
}

// Config controls workflow generation.
type Config struct {
	Platform       Platform
	DryRun         bool
	Force          bool
	Project        string
	Language       string
	BuildCmd       string
	LintCmd        string
	TestCmd        string
	DeterminismCmd string
}

// ResultFile describes a generated or planned CI workflow file.
type ResultFile struct {
	Platform Platform `json:"platform"`
	Path     string   `json:"path"`
	Content  string   `json:"content,omitempty"`
	Created  bool     `json:"created"`
	Skipped  bool     `json:"skipped"`
	Modified bool     `json:"modified"`
	Reason   string   `json:"reason,omitempty"`
}

// TemplateData holds values passed to the CI workflow templates.
type TemplateData struct {
	Project        string
	Language       string
	IsGo           bool
	IsPython       bool
	IsTypeScript   bool
	IsJava         bool
	IsGeneric      bool
	BuildCmd       string
	LintCmd        string
	TestCmd        string
	DeterminismCmd string
}

// BuildTemplateData infers project configuration from manifest or detection and defaults.
func BuildTemplateData(root string, cfg Config) TemplateData {
	data := TemplateData{
		Project:        cfg.Project,
		Language:       cfg.Language,
		BuildCmd:       cfg.BuildCmd,
		LintCmd:        cfg.LintCmd,
		TestCmd:        cfg.TestCmd,
		DeterminismCmd: cfg.DeterminismCmd,
	}

	// Try reading manifest first
	if manifest, ok, err := scaffold.ReadManifest(root); err == nil && ok {
		if data.Project == "" {
			data.Project = manifest.Answers.Project
		}
		if data.Language == "" && len(manifest.Answers.Languages) > 0 {
			data.Language = manifest.Answers.Languages[0]
		}
		if data.BuildCmd == "" && manifest.Answers.Build != "" {
			data.BuildCmd = manifest.Answers.Build
		}
		if data.LintCmd == "" && manifest.Answers.Lint != "" {
			data.LintCmd = manifest.Answers.Lint
		}
		if data.TestCmd == "" && manifest.Answers.Test != "" {
			data.TestCmd = manifest.Answers.Test
		}
	}

	// Fall back to detection if still missing values
	if data.Language == "" || data.BuildCmd == "" || data.TestCmd == "" {
		if det, err := detect.Detect(root); err == nil {
			if data.Project == "" {
				data.Project = filepath.Base(root)
			}
			if data.Language == "" && len(det.Languages) > 0 {
				data.Language = det.Languages[0]
			}
			if data.BuildCmd == "" && det.Build != "" {
				data.BuildCmd = det.Build
			}
			if data.LintCmd == "" && det.Lint != "" {
				data.LintCmd = det.Lint
			}
			if data.TestCmd == "" && det.Test != "" {
				data.TestCmd = det.Test
			}
		}
	}

	if data.Project == "" {
		data.Project = "project"
	}
	if data.Language == "" {
		data.Language = "other"
	}

	// Set language booleans
	switch strings.ToLower(data.Language) {
	case "go", "golang":
		data.IsGo = true
	case "python", "py":
		data.IsPython = true
	case "typescript", "ts", "javascript", "js", "node":
		data.IsTypeScript = true
	case "java":
		data.IsJava = true
	default:
		data.IsGeneric = true
	}

	// Supply defaults for empty commands based on language
	if data.BuildCmd == "" {
		switch {
		case data.IsGo:
			data.BuildCmd = "go build -v ./..."
		case data.IsPython:
			data.BuildCmd = "python -m compileall ."
		case data.IsTypeScript:
			data.BuildCmd = "npm run build"
		case data.IsJava:
			data.BuildCmd = "./mvnw compile -B"
		default:
			data.BuildCmd = "echo \"Compile/build step\""
		}
	}

	if data.LintCmd == "" {
		switch {
		case data.IsGo:
			data.LintCmd = "go vet ./..."
		case data.IsPython:
			data.LintCmd = "ruff check . || flake8 ."
		case data.IsTypeScript:
			data.LintCmd = "npm run lint"
		case data.IsJava:
			data.LintCmd = "./mvnw checkstyle:check"
		default:
			data.LintCmd = "echo \"Lint check step\""
		}
	}

	if data.TestCmd == "" {
		switch {
		case data.IsGo:
			data.TestCmd = "go test -v -race ./..."
		case data.IsPython:
			data.TestCmd = "pytest -v"
		case data.IsTypeScript:
			data.TestCmd = "npm test"
		case data.IsJava:
			data.TestCmd = "./mvnw test -B"
		default:
			data.TestCmd = "echo \"Test execution step\""
		}
	}

	if data.DeterminismCmd == "" {
		switch {
		case data.IsGo:
			data.DeterminismCmd = "go test ./... -count=2"
		case data.IsPython:
			data.DeterminismCmd = "pytest -v"
		case data.IsTypeScript:
			data.DeterminismCmd = "npm test"
		case data.IsJava:
			data.DeterminismCmd = "./mvnw test -B"
		default:
			data.DeterminismCmd = "echo \"Determinism verification step\""
		}
	}

	return data
}

// GenerateWorkflows plans the CI workflow files for the requested platform(s).
func GenerateWorkflows(root string, cfg Config) ([]ResultFile, error) {
	platforms := []Platform{}
	if cfg.Platform == PlatformAll {
		platforms = AllPlatforms
	} else {
		platforms = append(platforms, cfg.Platform)
	}

	data := BuildTemplateData(root, cfg)
	tmplSet, err := templates.New("")
	if err != nil {
		return nil, fmt.Errorf("ci: load templates: %w", err)
	}

	var results []ResultFile
	for _, p := range platforms {
		tmplName, ok := platformTemplates[p]
		if !ok {
			return nil, fmt.Errorf("ci: no template for platform %q", p)
		}

		rendered, err := tmplSet.Render(tmplName, data)
		if err != nil {
			return nil, fmt.Errorf("ci: render template %s: %w", tmplName, err)
		}

		relPath := platformPaths[p]
		fullPath := filepath.Join(root, filepath.FromSlash(relPath))
		res := ResultFile{
			Platform: p,
			Path:     relPath,
			Content:  rendered,
		}

		if _, err := os.Stat(fullPath); err == nil {
			if cfg.Force {
				res.Modified = true
				res.Reason = "existing file replaced (--force)"
			} else {
				res.Skipped = true
				res.Reason = "exists; never overwritten without --force"
			}
		} else {
			res.Created = true
			res.Reason = "new workflow file"
		}

		results = append(results, res)
	}

	return results, nil
}

// ApplyWorkflows writes generated workflow files to disk, adhering to overwrite guards.
func ApplyWorkflows(root string, files []ResultFile, force bool) error {
	for _, file := range files {
		if file.Skipped && !force {
			continue
		}
		fullPath := filepath.Join(root, filepath.FromSlash(file.Path))
		if err := fsutil.WriteAtomic(fullPath, []byte(file.Content), 0o644); err != nil {
			return fmt.Errorf("ci: apply %s: %w", file.Path, err)
		}
	}
	return nil
}
