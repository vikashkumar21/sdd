// Package detect inspects a project directory and reports what it can find
// from file names and a few manifest files. Every result is a suggestion, not
// a fact: callers must present it to the human as unverified.
package detect

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Result is what Detect found.
type Result struct {
	Root          string
	HasGit        bool
	Languages     []string        // known language packs found: go, java, python, typescript
	Other         []string        // other stacks found (rust, ruby, javascript, ...)
	SourceFiles   int             // number of recognised source files seen
	Capped        bool            // true when the scan stopped at its file limit
	SuggestedMode string          // "greenfield" or "brownfield"
	Existing      map[string]bool // sdd-relevant names that already exist in the root
	Build         string          // suggested commands; empty when unknown
	Test          string
	Lint          string
	Format        string
	TopDirs       []string
	CI            []string
	Manifests     []string
	MakeTargets   []string
}

const (
	maxVisited   = 20000
	maxMarkDepth = 2
)

var ignoredDirs = map[string]bool{
	".git": true, ".sdd": true, "node_modules": true, "vendor": true, "target": true,
	"build": true, "dist": true, "bin": true, "obj": true, "out": true, ".venv": true,
	"venv": true, "env": true, "__pycache__": true, ".idea": true, ".vscode": true,
	".gradle": true, ".next": true, "coverage": true, ".mypy_cache": true, ".pytest_cache": true,
}

var extLang = map[string]string{".go": "go", ".java": "java", ".py": "python", ".ts": "typescript", ".tsx": "typescript"}

var otherExt = map[string]string{
	".rs": "rust", ".rb": "ruby", ".php": "php", ".cs": "dotnet", ".kt": "kotlin",
	".swift": "swift", ".scala": "scala", ".js": "javascript", ".jsx": "javascript",
	".c": "c", ".cc": "c++", ".cpp": "c++",
}

var existingNames = []string{
	"README.md", "README", "CLAUDE.md", "GEMINI.md", "AGENTS.md", "MISSION.md",
	"TECH_STACK.md", "ROADMAP.md", "SKILLS.md", "features", "docs", "Makefile", "Dockerfile",
}

var makeTargetRe = regexp.MustCompile(`(?m)^([A-Za-z0-9_-][A-Za-z0-9_.-]*)\s*:(?:[^=]|$)`)

// Detect scans root. It never writes and never runs other programs.
func Detect(root string) (Result, error) {
	info, err := os.Stat(root)
	if err != nil {
		return Result{}, fmt.Errorf("detect: %w", err)
	}
	if !info.IsDir() {
		return Result{}, fmt.Errorf("detect: %s is not a directory", root)
	}
	res := Result{Root: root, Existing: map[string]bool{}}
	if _, err := os.Lstat(filepath.Join(root, ".git")); err == nil {
		res.HasGit = true
	}
	for _, name := range existingNames {
		if _, err := os.Stat(filepath.Join(root, name)); err == nil {
			res.Existing[name] = true
		}
	}

	langs := map[string]bool{}
	other := map[string]bool{}
	langExt := map[string]int{}
	otherExtCount := map[string]int{}
	manifests := map[string]bool{}
	rootPackage := false
	nestedPackage := false

	visited := 0
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil // unreadable entries are skipped; detection is best-effort
		}
		rel, err := filepath.Rel(root, path)
		if err != nil || rel == "." {
			return nil
		}
		depth := strings.Count(rel, string(filepath.Separator))
		if d.IsDir() {
			if ignoredDirs[d.Name()] {
				return filepath.SkipDir
			}
			if depth == 0 {
				res.TopDirs = append(res.TopDirs, d.Name())
			}
			return nil
		}
		visited++
		if visited > maxVisited {
			res.Capped = true
			return fs.SkipAll
		}
		name := d.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if lang, ok := extLang[ext]; ok {
			langExt[lang]++
			res.SourceFiles++
		} else if lang, ok := otherExt[ext]; ok {
			otherExtCount[lang]++
			res.SourceFiles++
		}
		if depth > maxMarkDepth {
			return nil
		}
		slashRel := filepath.ToSlash(rel)
		switch name {
		case "go.mod":
			langs["go"] = true
			manifests[slashRel] = true
		case "go.work":
			langs["go"] = true
		case "pom.xml", "build.gradle", "build.gradle.kts", "settings.gradle", "settings.gradle.kts":
			langs["java"] = true
			if name == "pom.xml" || strings.HasPrefix(name, "build.gradle") {
				manifests[slashRel] = true
			}
		case "pyproject.toml", "requirements.txt", "setup.py", "Pipfile":
			langs["python"] = true
			manifests[slashRel] = true
		case "tsconfig.json":
			langs["typescript"] = true
		case "package.json":
			manifests[slashRel] = true
			if depth == 0 {
				rootPackage = true
			} else {
				nestedPackage = true
			}
		case "Cargo.toml":
			other["rust"] = true
			manifests[slashRel] = true
		case "Gemfile":
			other["ruby"] = true
			manifests[slashRel] = true
		case "composer.json":
			other["php"] = true
			manifests[slashRel] = true
		}
		if strings.HasSuffix(name, ".csproj") || strings.HasSuffix(name, ".sln") {
			other["dotnet"] = true
		}
		return nil
	})

	var scripts map[string]string
	if rootPackage {
		pkg := readPackageJSON(filepath.Join(root, "package.json"))
		scripts = pkg.Scripts
		if pkg.hasDep("typescript") {
			langs["typescript"] = true
		}
	}
	if (rootPackage || nestedPackage) && !langs["typescript"] {
		other["javascript"] = true
	}
	for lang, n := range langExt {
		if n >= 3 {
			langs[lang] = true
		}
	}
	for lang, n := range otherExtCount {
		if n >= 3 {
			other[lang] = true
		}
	}
	if langs["typescript"] {
		delete(other, "javascript")
	}

	res.Languages = sortedKeys(langs)
	res.Other = sortedKeys(other)
	res.Manifests = sortedKeys(manifests)
	sort.Strings(res.TopDirs)
	res.CI = detectCI(root)
	res.MakeTargets = makeTargets(root)
	res.SuggestedMode = "greenfield"
	if res.SourceFiles > 0 {
		res.SuggestedMode = "brownfield"
	}
	suggest(&res, scripts)
	return res, nil
}

type packageJSON struct {
	Scripts         map[string]string `json:"scripts"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

func (p packageJSON) hasDep(name string) bool {
	_, a := p.Dependencies[name]
	_, b := p.DevDependencies[name]
	return a || b
}

func readPackageJSON(path string) packageJSON {
	var p packageJSON
	data, err := os.ReadFile(path)
	if err != nil {
		return p
	}
	_ = json.Unmarshal(data, &p) // malformed JSON simply yields no information
	return p
}

func detectCI(root string) []string {
	var ci []string
	checks := []struct{ path, name string }{
		{".github/workflows", "GitHub Actions"},
		{".gitlab-ci.yml", "GitLab CI"},
		{"Jenkinsfile", "Jenkins"},
		{"azure-pipelines.yml", "Azure Pipelines"},
		{".circleci", "CircleCI"},
	}
	for _, c := range checks {
		if _, err := os.Stat(filepath.Join(root, c.path)); err == nil {
			ci = append(ci, c.name)
		}
	}
	return ci
}

func makeTargets(root string) []string {
	data, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	for _, m := range makeTargetRe.FindAllStringSubmatch(string(data), -1) {
		if !strings.HasPrefix(m[1], ".") {
			seen[m[1]] = true
		}
	}
	return sortedKeys(seen)
}

func suggest(res *Result, scripts map[string]string) {
	has := func(t string) bool {
		for _, x := range res.MakeTargets {
			if x == t {
				return true
			}
		}
		return false
	}
	if has("build") {
		res.Build = "make build"
	}
	if has("test") {
		res.Test = "make test"
	}
	if has("lint") {
		res.Lint = "make lint"
	}
	switch {
	case has("fmt"):
		res.Format = "make fmt"
	case has("format"):
		res.Format = "make format"
	}
	primary := ""
	for _, l := range []string{"go", "java", "typescript", "python"} {
		for _, have := range res.Languages {
			if have == l && primary == "" {
				primary = l
			}
		}
	}
	fill := func(dst *string, v string) {
		if *dst == "" {
			*dst = v
		}
	}
	exists := func(name string) bool {
		_, err := os.Stat(filepath.Join(res.Root, name))
		return err == nil
	}
	switch primary {
	case "go":
		fill(&res.Build, "go build ./...")
		fill(&res.Test, "go test ./...")
		fill(&res.Lint, "go vet ./...")
		fill(&res.Format, "gofmt -l .")
	case "java":
		switch {
		case exists("gradlew"):
			fill(&res.Build, "./gradlew build")
			fill(&res.Test, "./gradlew test")
		case exists("mvnw"):
			fill(&res.Build, "./mvnw -q package")
			fill(&res.Test, "./mvnw test")
		case exists("pom.xml"):
			fill(&res.Build, "mvn -q package")
			fill(&res.Test, "mvn test")
		case exists("build.gradle") || exists("build.gradle.kts"):
			fill(&res.Build, "gradle build")
			fill(&res.Test, "gradle test")
		}
	case "typescript":
		if _, ok := scripts["build"]; ok {
			fill(&res.Build, "npm run build")
		}
		if _, ok := scripts["test"]; ok {
			fill(&res.Test, "npm test")
		}
		if _, ok := scripts["lint"]; ok {
			fill(&res.Lint, "npm run lint")
		}
		if _, ok := scripts["format"]; ok {
			fill(&res.Format, "npm run format")
		}
	case "python":
		if exists("tests") || exists("test") || fileMentions(filepath.Join(res.Root, "pyproject.toml"), "pytest") ||
			fileMentions(filepath.Join(res.Root, "requirements.txt"), "pytest") {
			fill(&res.Test, "pytest")
		}
	}
}

func fileMentions(path, needle string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(data)), needle)
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
