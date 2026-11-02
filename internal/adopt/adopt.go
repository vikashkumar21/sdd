// Package adopt inspects pre-existing or hand-crafted spec-driven development kits
// and generates a valid .sdd/manifest.json without modifying any existing files.
package adopt

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/vikashkumar21/sdd/internal/block"
	"github.com/vikashkumar21/sdd/internal/detect"
	"github.com/vikashkumar21/sdd/internal/fsutil"
	"github.com/vikashkumar21/sdd/internal/scaffold"
	"github.com/vikashkumar21/sdd/internal/templates"
	"github.com/vikashkumar21/sdd/internal/version"
)

// CoreDocs lists the documents required for a project to be considered a spec-driven kit.
var CoreDocs = []string{
	"MISSION.md",
	"TECH_STACK.md",
	"ROADMAP.md",
	"SKILLS.md",
}

// Options configures adoption behaviour.
type Options struct {
	Force      bool   // overwrite existing .sdd/manifest.json (with backup)
	Name       string // override project name
	Maintainer string // override maintainer
	Mode       string // override mode (default: brownfield)
	Now        time.Time
}

// Result describes the outcome of sdd adopt.
type Result struct {
	ManifestPath string
	BackupPath   string
	Project      string
	Maintainer   string
	Mode         string
	Agents       []string
	Discovered   []string
}

func sum(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// MissingCoreDocs checks which of the four core documents are absent.
func MissingCoreDocs(root string) []string {
	var missing []string
	for _, doc := range CoreDocs {
		p := filepath.Join(root, filepath.FromSlash(doc))
		if info, err := os.Stat(p); err != nil || info.IsDir() {
			missing = append(missing, doc)
		}
	}
	return missing
}

// DetectAlternateNames checks for common non-standard document names and returns suggestions.
func DetectAlternateNames(root string) []string {
	var hints []string
	candidates := []struct {
		alt, standard string
	}{
		{"VISION.md", "MISSION.md"},
		{"GOALS.md", "MISSION.md"},
		{"ARCHITECTURE.md", "TECH_STACK.md"},
		{"STACK.md", "TECH_STACK.md"},
		{"BACKLOG.md", "ROADMAP.md"},
		{"TODO.md", "ROADMAP.md"},
		{"STANDARDS.md", "SKILLS.md"},
	}
	for _, c := range candidates {
		p := filepath.Join(root, c.alt)
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			hints = append(hints, fmt.Sprintf("found %q (consider renaming to %q)", c.alt, c.standard))
		}
	}
	return hints
}

var (
	missionTitleRe      = regexp.MustCompile(`(?m)^#\s*MISSION\s*[-—:]\s*(.+)`)
	missionMaintainerRe = regexp.MustCompile(`Maintainer:\s*([^·\n]+)`)
	techStackLangRe     = regexp.MustCompile(`Language\(s\):\s*([^\n|]+)`)
)

// BuildManifest inspects the kit files in root and constructs a valid Manifest.
// It reads files only and performs zero mutations.
func BuildManifest(root string, opts Options) (*scaffold.Manifest, []string, error) {
	missing := MissingCoreDocs(root)
	if len(missing) > 0 {
		hints := DetectAlternateNames(root)
		msg := fmt.Sprintf("cannot adopt %s: missing required core documents: %s", root, strings.Join(missing, ", "))
		if len(hints) > 0 {
			msg += " (" + strings.Join(hints, "; ") + ")"
		}
		return nil, nil, errors.New(msg)
	}

	manifestPath := filepath.Join(root, filepath.FromSlash(scaffold.ManifestPath))
	if _, err := os.Stat(manifestPath); err == nil && !opts.Force {
		return nil, nil, errors.New("this project already has a manifest (.sdd/manifest.json); run sdd check, or use --force to overwrite")
	}

	rootHandle, err := os.OpenRoot(root)
	if err != nil {
		return nil, nil, fmt.Errorf("adopt: open root: %w", err)
	}
	defer func() { _ = rootHandle.Close() }()

	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}

	// 1. Detect project name
	project := strings.TrimSpace(opts.Name)
	missionBytes, _ := rootHandle.ReadFile("MISSION.md")
	missionText := strings.ReplaceAll(string(missionBytes), "\r\n", "\n")
	if project == "" {
		if m := missionTitleRe.FindStringSubmatch(missionText); len(m) > 1 {
			project = strings.TrimSpace(m[1])
		}
	}
	if project == "" {
		project = filepath.Base(root)
	}

	// 2. Detect maintainer
	maintainer := strings.TrimSpace(opts.Maintainer)
	if maintainer == "" {
		if m := missionMaintainerRe.FindStringSubmatch(missionText); len(m) > 1 {
			maintainer = strings.TrimSpace(m[1])
		}
	}
	if maintainer == "" {
		maintainer = os.Getenv("USER")
		if maintainer == "" {
			maintainer = os.Getenv("USERNAME")
		}
	}

	// 3. Detect mode
	mode := strings.TrimSpace(opts.Mode)
	if mode == "" {
		mode = scaffold.ModeBrownfield
	}

	// 4. Detect languages and build commands
	det, _ := detect.Detect(root)
	langs := det.Languages
	if len(langs) == 0 {
		techBytes, _ := rootHandle.ReadFile("TECH_STACK.md")
		techText := string(techBytes)
		if m := techStackLangRe.FindStringSubmatch(techText); len(m) > 1 {
			langs = splitLanguages(m[1])
		}
	}
	if len(langs) == 0 {
		langs = []string{"other"}
	}

	// 5. Discover agent files
	var agentIDs []string
	var discovered []string
	var fileEntries []scaffold.FileEntry

	agentChecks := []struct {
		id, file string
	}{
		{"claude", "CLAUDE.md"},
		{"gemini", "GEMINI.md"},
		{"agents", "AGENTS.md"},
	}

	for _, ac := range agentChecks {
		data, err := rootHandle.ReadFile(ac.file)
		if err != nil {
			continue
		}
		discovered = append(discovered, ac.file)
		agentIDs = append(agentIDs, ac.id)

		text := strings.ReplaceAll(string(data), "\r\n", "\n")
		inner, hasBlock := block.Inner(text)
		if hasBlock {
			fileEntries = append(fileEntries, scaffold.FileEntry{
				Path:   ac.file,
				Kind:   "block",
				State:  "existing",
				SHA256: sum(inner),
			})
		} else {
			fileEntries = append(fileEntries, scaffold.FileEntry{
				Path:   ac.file,
				Kind:   "file",
				State:  "existing",
				SHA256: sum(text),
			})
		}
	}

	if len(agentIDs) == 0 {
		agentIDs = []string{"agents"}
	}

	// 6. Discover core living documents
	coreDocs := []string{"MISSION.md", "TECH_STACK.md", "ROADMAP.md", "SKILLS.md", "README.md"}
	for _, doc := range coreDocs {
		data, err := rootHandle.ReadFile(doc)
		if err != nil {
			continue
		}
		discovered = append(discovered, doc)
		text := strings.ReplaceAll(string(data), "\r\n", "\n")
		fileEntries = append(fileEntries, scaffold.FileEntry{
			Path:   doc,
			Kind:   "file",
			State:  "existing",
			SHA256: sum(text),
		})
	}

	// 7. Discover task templates in features/_templates/
	taskTemplates := []string{
		"features/_templates/requirements.md",
		"features/_templates/plan.md",
		"features/_templates/validation.md",
	}
	for _, tmpl := range taskTemplates {
		data, err := rootHandle.ReadFile(filepath.FromSlash(tmpl))
		if err != nil {
			continue
		}
		discovered = append(discovered, tmpl)
		text := strings.ReplaceAll(string(data), "\r\n", "\n")
		fileEntries = append(fileEntries, scaffold.FileEntry{
			Path:   tmpl,
			Kind:   "file",
			State:  "existing",
			SHA256: sum(text),
		})
	}

	manifest := &scaffold.Manifest{
		Schema: 1,
		Tool: scaffold.ToolInfo{
			Name:    "sdd",
			Version: version.Version,
		},
		Templates: scaffold.TemplatesInfo{
			Version: templates.Version,
		},
		Generated: now.UTC().Format("2006-01-02"),
		Answers: scaffold.Answers{
			Project:    project,
			Purpose:    "",
			Mode:       mode,
			Languages:  langs,
			Maintainer: maintainer,
			Build:      det.Build,
			Test:       det.Test,
			Lint:       det.Lint,
			Format:     det.Format,
			Agents:     agentIDs,
		},
		Files: fileEntries,
	}

	return manifest, discovered, nil
}

// Adopt creates the .sdd/manifest.json file for an existing kit without altering any other files.
func Adopt(root string, opts Options) (*Result, error) {
	manifest, discovered, err := BuildManifest(root, opts)
	if err != nil {
		return nil, err
	}

	rootHandle, err := os.OpenRoot(root)
	if err != nil {
		return nil, fmt.Errorf("adopt: open root: %w", err)
	}
	defer func() { _ = rootHandle.Close() }()

	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}

	res := &Result{
		ManifestPath: scaffold.ManifestPath,
		Project:      manifest.Answers.Project,
		Maintainer:   manifest.Answers.Maintainer,
		Mode:         manifest.Answers.Mode,
		Agents:       manifest.Answers.Agents,
		Discovered:   discovered,
	}

	// Backup existing manifest if overwriting with --force
	manifestRel := filepath.FromSlash(scaffold.ManifestPath)
	existingManifest, err := rootHandle.ReadFile(manifestRel)
	if err == nil {
		backupTimestamp := now.UTC().Format("20060102-150405")
		backupDst := filepath.Join(".sdd", "backup", backupTimestamp, manifestRel)
		if err := fsutil.WriteAtomicRoot(rootHandle, backupDst, existingManifest, 0o644); err != nil {
			return nil, fmt.Errorf("adopt: backup existing manifest: %w", err)
		}
		res.BackupPath = filepath.ToSlash(backupDst)
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("adopt: encode manifest: %w", err)
	}

	if err := fsutil.WriteAtomicRoot(rootHandle, manifestRel, append(data, '\n'), 0o644); err != nil {
		return nil, fmt.Errorf("adopt: write manifest: %w", err)
	}

	return res, nil
}

func splitLanguages(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		p := strings.ToLower(strings.TrimSpace(part))
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
