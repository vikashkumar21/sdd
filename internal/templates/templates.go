// Package templates loads and renders the markdown kit that sdd generates.
//
// Templates are embedded in the binary. An optional override directory can
// replace any individual template file without rebuilding sdd; files missing
// from the override directory fall back to the embedded ones.
package templates

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"text/template"
)

// Version identifies the embedded template set. Bump it whenever any template
// changes so that generated projects record which templates they came from.
const Version = "1.0.0"

//go:embed all:data
var embedded embed.FS

// Set is a loaded template set.
type Set struct {
	fsys     fs.FS
	override string
}

// Pack is the language-specific content for one language.
type Pack struct {
	ID        string
	Title     string
	Standards string // markdown bullets for SKILLS.md
	Contract  string // short markdown bullets for the agent contract
	SkillMap  string // markdown table rows for the human skill map
}

var packTitles = map[string]string{
	"go": "Go", "java": "Java", "python": "Python", "typescript": "TypeScript", "generic": "Other / unspecified",
}

// LanguageIDs lists the selectable languages in canonical order. "other"
// selects the generic pack.
func LanguageIDs() []string {
	return []string{"go", "java", "python", "typescript", "other"}
}

// New returns the embedded template set, layered over overrideDir when it is
// not empty.
func New(overrideDir string) (*Set, error) {
	base, err := fs.Sub(embedded, "data")
	if err != nil {
		return nil, fmt.Errorf("templates: embedded data: %w", err)
	}
	if overrideDir == "" {
		return &Set{fsys: base}, nil
	}
	info, err := os.Stat(overrideDir)
	if err != nil {
		return nil, fmt.Errorf("templates: override directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("templates: override %s is not a directory", overrideDir)
	}
	return &Set{fsys: overlay{top: os.DirFS(overrideDir), base: base}, override: overrideDir}, nil
}

// Override returns the override directory in use, or "".
func (s *Set) Override() string { return s.override }

type overlay struct{ top, base fs.FS }

func (o overlay) Open(name string) (fs.File, error) {
	f, err := o.top.Open(name)
	if err == nil {
		return f, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	return o.base.Open(name)
}

// Raw returns a template file exactly as stored, without rendering it.
func (s *Set) Raw(name string) (string, error) {
	data, err := fs.ReadFile(s.fsys, name)
	if err != nil {
		return "", fmt.Errorf("templates: read %s: %w", name, err)
	}
	return string(data), nil
}

// Render parses the named template and executes it with data.
func (s *Set) Render(name string, data any) (string, error) {
	text, err := s.Raw(name)
	if err != nil {
		return "", err
	}
	tpl, err := template.New(name).Funcs(template.FuncMap{
		"join": strings.Join,
		"add":  func(a, b int) int { return a + b },
	}).Option("missingkey=error").Parse(text)
	if err != nil {
		return "", fmt.Errorf("templates: parse %s: %w", name, err)
	}
	var sb strings.Builder
	if err := tpl.Execute(&sb, data); err != nil {
		return "", fmt.Errorf("templates: render %s: %w", name, err)
	}
	return sb.String(), nil
}

// Pack loads the language pack with the given ID ("other" maps to "generic").
func (s *Set) Pack(id string) (Pack, error) {
	if id == "other" {
		id = "generic"
	}
	title, ok := packTitles[id]
	if !ok {
		return Pack{}, fmt.Errorf("templates: unknown language pack %q", id)
	}
	read := func(file string) (string, error) {
		text, err := s.Raw("packs/" + id + "/" + file)
		return strings.Trim(text, "\n"), err
	}
	p := Pack{ID: id, Title: title}
	var err error
	if p.Standards, err = read("standards.md"); err != nil {
		return Pack{}, err
	}
	if p.Contract, err = read("contract.md"); err != nil {
		return Pack{}, err
	}
	if p.SkillMap, err = read("skillmap.md"); err != nil {
		return Pack{}, err
	}
	return p, nil
}
