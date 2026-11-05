// Package coordinator provides Task Coordinator merge qualification,
// blast-radius regression sweeps, flake guards, and git merge orchestration.
package coordinator

import (
	"bufio"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// BlastResult represents the computed dependency blast radius for modified files.
type BlastResult struct {
	ModulePath          string   `json:"module_path"`
	ModifiedFiles       []string `json:"modified_files"`
	ModifiedPackages    []string `json:"modified_packages"`
	DownstreamPackages  []string `json:"downstream_packages"`
	AllAffectedPackages []string `json:"all_affected_packages"`
}

// ModuleGraph maps Go packages to their direct imports and direct importers.
type ModuleGraph struct {
	ModulePath string
	Packages   map[string][]string // pkg -> imported pkgs (within module)
	Importers  map[string][]string // pkg -> pkgs that import it (within module)
}

// FindModuleRoot finds the nearest directory containing go.mod starting from dir.
func FindModuleRoot(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	curr := abs
	for {
		if _, err := os.Stat(filepath.Join(curr, "go.mod")); err == nil {
			return curr, nil
		}
		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		curr = parent
	}
	return "", fmt.Errorf("no go.mod found in %s or any parent directory", abs)
}

// ReadModulePath parses the module path from go.mod in root.
func ReadModulePath(root string) (string, error) {
	modRoot, err := FindModuleRoot(root)
	if err != nil {
		return "", err
	}
	goModPath := filepath.Join(modRoot, "go.mod")
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return "", fmt.Errorf("reading go.mod: %w", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1], nil
			}
		}
	}
	return "", fmt.Errorf("module declaration not found in %s", goModPath)
}

// BuildModuleGraph inspects all Go source files in root to construct the internal dependency graph.
func BuildModuleGraph(root string) (*ModuleGraph, error) {
	modRoot, err := FindModuleRoot(root)
	if err != nil {
		return nil, err
	}
	root = modRoot

	modPath, err := ReadModulePath(root)
	if err != nil {
		return nil, err
	}

	pkgs := make(map[string]map[string]bool)
	fset := token.NewFileSet()

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules" || name == "dist" || name == "bin" || name == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}

		relDir, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return err
		}
		relDir = filepath.ToSlash(relDir)
		if relDir == "." {
			relDir = "."
		}

		if _, exists := pkgs[relDir]; !exists {
			pkgs[relDir] = make(map[string]bool)
		}

		node, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			// Skip unparseable scratch or template files
			return nil
		}

		for _, imp := range node.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			if strings.HasPrefix(importPath, modPath) {
				// Internal module import
				subPkg := strings.TrimPrefix(importPath, modPath)
				subPkg = strings.TrimPrefix(subPkg, "/")
				if subPkg == "" {
					subPkg = "."
				}
				pkgs[relDir][subPkg] = true
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scanning Go packages in %s: %w", root, err)
	}

	graph := &ModuleGraph{
		ModulePath: modPath,
		Packages:   make(map[string][]string),
		Importers:  make(map[string][]string),
	}

	for p := range pkgs {
		graph.Packages[p] = nil
		graph.Importers[p] = nil
	}

	for p, imps := range pkgs {
		var list []string
		for imp := range imps {
			list = append(list, imp)
			graph.Importers[imp] = append(graph.Importers[imp], p)
		}
		sort.Strings(list)
		graph.Packages[p] = list
	}

	for p := range graph.Importers {
		sort.Strings(graph.Importers[p])
	}

	return graph, nil
}

// ComputeBlastRadius calculates the downstream reverse dependency blast radius for the modified files.
func ComputeBlastRadius(root string, modifiedFiles []string) (*BlastResult, error) {
	graph, err := BuildModuleGraph(root)
	if err != nil {
		return nil, err
	}

	// Normalize modified files
	var normFiles []string
	seenFiles := make(map[string]bool)
	modPkgMap := make(map[string]bool)

	for _, f := range modifiedFiles {
		clean := filepath.ToSlash(filepath.Clean(f))
		clean = strings.TrimPrefix(clean, "./")
		if clean == "" || clean == "." || seenFiles[clean] {
			continue
		}
		seenFiles[clean] = true
		normFiles = append(normFiles, clean)

		// Map to package directory
		pkgDir := filepath.Dir(clean)
		if pkgDir == "" {
			pkgDir = "."
		}
		pkgDir = filepath.ToSlash(pkgDir)

		// Only register if it's a known Go package in the graph
		if _, isPkg := graph.Packages[pkgDir]; isPkg {
			modPkgMap[pkgDir] = true
		}
	}
	sort.Strings(normFiles)

	var modPkgs []string
	for p := range modPkgMap {
		modPkgs = append(modPkgs, p)
	}
	sort.Strings(modPkgs)

	// BFS traversal to find all downstream dependent packages
	downstreamMap := make(map[string]bool)
	queue := append([]string{}, modPkgs...)
	visited := make(map[string]bool)

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		for _, imp := range graph.Importers[curr] {
			if !modPkgMap[imp] && !downstreamMap[imp] {
				downstreamMap[imp] = true
			}
			if !visited[imp] {
				visited[imp] = true
				queue = append(queue, imp)
			}
		}
	}

	var downstreamPkgs []string
	for p := range downstreamMap {
		downstreamPkgs = append(downstreamPkgs, p)
	}
	sort.Strings(downstreamPkgs)

	allMap := make(map[string]bool)
	for _, p := range modPkgs {
		allMap[p] = true
	}
	for _, p := range downstreamPkgs {
		allMap[p] = true
	}

	var allPkgs []string
	for p := range allMap {
		allPkgs = append(allPkgs, p)
	}
	sort.Strings(allPkgs)

	return &BlastResult{
		ModulePath:          graph.ModulePath,
		ModifiedFiles:       normFiles,
		ModifiedPackages:    modPkgs,
		DownstreamPackages:  downstreamPkgs,
		AllAffectedPackages: allPkgs,
	}, nil
}
