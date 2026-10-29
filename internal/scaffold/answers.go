// Package scaffold plans and applies the spec-driven kit: it decides, file by
// file, what to create, merge, or skip, and records what it did in a manifest.
package scaffold

import (
	"errors"
	"fmt"
	"strings"

	"github.com/vikashkumar21/sdd/internal/detect"
	"github.com/vikashkumar21/sdd/internal/templates"
)

// Project modes.
const (
	ModeGreenfield = "greenfield"
	ModeBrownfield = "brownfield"
)

// Todo is the marker for content the human still has to supply.
const Todo = "TODO(human)"

// AgentIDs lists the supported agent files in canonical order.
func AgentIDs() []string { return []string{"claude", "gemini", "agents"} }

// AgentFile returns the file name for an agent ID.
func AgentFile(id string) string {
	switch id {
	case "claude":
		return "CLAUDE.md"
	case "gemini":
		return "GEMINI.md"
	default:
		return "AGENTS.md"
	}
}

// Answers are the human's inputs to `sdd init`.
type Answers struct {
	Project    string   `json:"project"`
	Purpose    string   `json:"purpose"`
	Mode       string   `json:"mode"`
	Maintainer string   `json:"maintainer"`
	Languages  []string `json:"languages"`
	Build      string   `json:"build,omitempty"`
	Test       string   `json:"test,omitempty"`
	Lint       string   `json:"lint,omitempty"`
	Format     string   `json:"format,omitempty"`
	Agents     []string `json:"agents"`
	ExtraRules []string `json:"extraRules,omitempty"`
}

// Normalize validates the answers, applies defaults, and puts list values into
// canonical order so the generated output is deterministic.
func (a *Answers) Normalize() error {
	for _, f := range []struct{ name, val string }{
		{"project name", a.Project}, {"maintainer", a.Maintainer},
		{"build command", a.Build}, {"test command", a.Test},
		{"lint command", a.Lint}, {"format command", a.Format},
	} {
		if strings.ContainsAny(f.val, "\r\n") {
			return fmt.Errorf("%s must be a single line", f.name)
		}
		if strings.ContainsAny(f.val, "|`") {
			return fmt.Errorf("%s must not contain '|' or a backtick (they break markdown tables)", f.name)
		}
	}
	a.Project = strings.TrimSpace(a.Project)
	if a.Project == "" {
		return errors.New("project name must not be empty")
	}
	switch a.Mode {
	case ModeGreenfield, ModeBrownfield:
	default:
		return fmt.Errorf("mode must be %q or %q, got %q", ModeGreenfield, ModeBrownfield, a.Mode)
	}
	a.Purpose = strings.TrimSpace(strings.ReplaceAll(a.Purpose, "\n", " "))
	if a.Purpose == "" {
		a.Purpose = Todo + ": one-line purpose"
	}
	a.Maintainer = strings.TrimSpace(a.Maintainer)
	if a.Maintainer == "" {
		a.Maintainer = Todo
	}
	a.Build, a.Test = strings.TrimSpace(a.Build), strings.TrimSpace(a.Test)
	a.Lint, a.Format = strings.TrimSpace(a.Lint), strings.TrimSpace(a.Format)

	langs, err := canonical(a.Languages, templates.LanguageIDs(), "language")
	if err != nil {
		return err
	}
	if len(langs) == 0 {
		langs = []string{"other"}
	}
	a.Languages = langs
	agents, err := canonical(a.Agents, AgentIDs(), "agent")
	if err != nil {
		return err
	}
	if len(agents) == 0 {
		agents = AgentIDs()
	}
	a.Agents = agents

	var rules []string
	for _, r := range a.ExtraRules {
		r = strings.TrimSpace(strings.ReplaceAll(r, "\n", " "))
		if r != "" {
			rules = append(rules, r)
		}
	}
	a.ExtraRules = rules
	return nil
}

// ParseLanguages validates language IDs and returns them in canonical order.
func ParseLanguages(values []string) ([]string, error) {
	return canonical(values, templates.LanguageIDs(), "language")
}

// ParseAgents validates agent IDs and returns them in canonical order.
func ParseAgents(values []string) ([]string, error) {
	return canonical(values, AgentIDs(), "agent")
}

// canonical returns the subset of allowed that appears in values, in the
// canonical order of allowed, and rejects unknown values.
func canonical(values, allowed []string, what string) ([]string, error) {
	want := map[string]bool{}
	for _, v := range values {
		v = strings.ToLower(strings.TrimSpace(v))
		if v == "" {
			continue
		}
		ok := false
		for _, a := range allowed {
			if a == v {
				ok = true
			}
		}
		if !ok {
			return nil, fmt.Errorf("unknown %s %q (allowed: %s)", what, v, strings.Join(allowed, ", "))
		}
		want[v] = true
	}
	var out []string
	for _, a := range allowed {
		if want[a] {
			out = append(out, a)
		}
	}
	return out, nil
}

// Data is everything the templates may reference.
type Data struct {
	Answers
	Date            string
	ToolVersion     string
	TemplateVersion string
	Greenfield      bool
	Brownfield      bool
	LanguageList    string
	Packs           []templates.Pack
	AgentFiles      []string
	Detected        detect.Result
}
