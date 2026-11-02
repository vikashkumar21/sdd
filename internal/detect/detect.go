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
	maxMarkDepth = 4
)

var ignoredDirs = map[string]bool{
	".git": true, ".sdd": true, "node_modules": true, "vendor": true, "target": true,
	"build": true, "dist": true, "bin": true, "obj": true, "out": true, ".venv": true,
	"venv": true, "env": true, "__pycache__": true, ".idea": true, ".vscode": true,
	".gradle": true, ".next": true, "coverage": true, ".mypy_cache": true, ".pytest_cache": true,
	".turbo": true, ".pnpm-store": true, "bower_components": true, ".cache": true, ".terraform": true, ".yarn": true,
}

func isTableSafe(s string) bool {
	return !strings.ContainsAny(s, "|`\r\n")
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
	var packageJSONPaths []string
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
			packageJSONPaths = append(packageJSONPaths, slashRel)
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

	scripts := map[string]string{}
	sort.Slice(packageJSONPaths, func(i, j int) bool {
		depthI := strings.Count(packageJSONPaths[i], "/")
		depthJ := strings.Count(packageJSONPaths[j], "/")
		if depthI != depthJ {
			return depthI < depthJ
		}
		return packageJSONPaths[i] < packageJSONPaths[j]
	})

	for _, pRel := range packageJSONPaths {
		pkg := readPackageJSON(filepath.Join(root, filepath.FromSlash(pRel)))
		if pkg.hasDep("typescript") {
			langs["typescript"] = true
		}
		dir := filepath.ToSlash(filepath.Dir(pRel))
		prefix := "npm "
		if dir != "." {
			prefix = "npm --prefix " + dir + " "
		}

		if val, ok := pkg.Scripts["build"]; ok && scripts["build"] == "" && isTableSafe(val) && isTableSafe(prefix) {
			cmd := prefix + "run build"
			if isTableSafe(cmd) {
				scripts["build"] = cmd
			}
		}
		if val, ok := pkg.Scripts["test"]; ok && scripts["test"] == "" && isTableSafe(val) && isTableSafe(prefix) {
			cmd := prefix + "test"
			if isTableSafe(cmd) {
				scripts["test"] = cmd
			}
		}
		if val, ok := pkg.Scripts["lint"]; ok && scripts["lint"] == "" && isTableSafe(val) && isTableSafe(prefix) {
			cmd := prefix + "run lint"
			if isTableSafe(cmd) {
				scripts["lint"] = cmd
			}
		}
		if val, ok := pkg.Scripts["format"]; ok && scripts["format"] == "" && isTableSafe(val) && isTableSafe(prefix) {
			cmd := prefix + "run format"
			if isTableSafe(cmd) {
				scripts["format"] = cmd
			}
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

type langCommands struct {
	build  string
	test   string
	lint   string
	format string
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

	exists := func(name string) bool {
		_, err := os.Stat(filepath.Join(res.Root, name))
		return err == nil
	}

	langCmds := map[string]langCommands{}
	for _, lang := range res.Languages {
		var lc langCommands
		switch lang {
		case "go":
			lc.build = "go build ./..."
			lc.test = "go test ./..."
			lc.lint = "go vet ./..."
			lc.format = "gofmt -l ."
		case "java":
			switch {
			case exists("gradlew"):
				lc.build = "./gradlew build"
				lc.test = "./gradlew test"
			case exists("mvnw"):
				lc.build = "./mvnw -q package"
				lc.test = "./mvnw test"
			case exists("pom.xml"):
				lc.build = "mvn -q package"
				lc.test = "mvn test"
			case exists("build.gradle") || exists("build.gradle.kts"):
				lc.build = "gradle build"
				lc.test = "gradle test"
			default:
				for _, m := range res.Manifests {
					base := filepath.Base(m)
					dir := filepath.ToSlash(filepath.Dir(m))
					if base == "pom.xml" {
						if dir == "." {
							lc.build = "mvn -q package"
							lc.test = "mvn test"
						} else {
							lc.build = "mvn -f " + dir + "/pom.xml -q package"
							lc.test = "mvn -f " + dir + "/pom.xml test"
						}
						break
					} else if strings.HasPrefix(base, "build.gradle") {
						if dir == "." {
							lc.build = "gradle build"
							lc.test = "gradle test"
						} else {
							lc.build = "gradle -p " + dir + " build"
							lc.test = "gradle -p " + dir + " test"
						}
						break
					}
				}
			}
		case "typescript":
			lc.build = scripts["build"]
			lc.test = scripts["test"]
			lc.lint = scripts["lint"]
			lc.format = scripts["format"]
		case "python":
			hasPytest := exists("tests") || exists("test") ||
				fileMentions(filepath.Join(res.Root, "pyproject.toml"), "pytest") ||
				fileMentions(filepath.Join(res.Root, "requirements.txt"), "pytest")
			if !hasPytest {
				for _, m := range res.Manifests {
					if strings.HasSuffix(m, "pyproject.toml") || strings.HasSuffix(m, "requirements.txt") {
						if fileMentions(filepath.Join(res.Root, filepath.FromSlash(m)), "pytest") {
							hasPytest = true
							break
						}
					}
				}
			}
			if hasPytest {
				lc.test = "pytest"
			}
		}
		langCmds[lang] = lc
	}

	canonicalLangs := []string{"go", "java", "python", "typescript"}
	combine := func(getField func(lc langCommands) string) string {
		var parts []string
		for _, lang := range canonicalLangs {
			if lc, ok := langCmds[lang]; ok {
				val := strings.TrimSpace(getField(lc))
				if val != "" && isTableSafe(val) {
					parts = append(parts, val)
				}
			}
		}
		return strings.Join(parts, " && ")
	}

	if res.Build == "" {
		res.Build = combine(func(lc langCommands) string { return lc.build })
	}
	if res.Test == "" {
		res.Test = combine(func(lc langCommands) string { return lc.test })
	}
	if res.Lint == "" {
		res.Lint = combine(func(lc langCommands) string { return lc.lint })
	}
	if res.Format == "" {
		res.Format = combine(func(lc langCommands) string { return lc.format })
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
