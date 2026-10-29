package scaffold

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vikashkumar21/sdd/internal/block"
	"github.com/vikashkumar21/sdd/internal/detect"
	"github.com/vikashkumar21/sdd/internal/templates"
	"github.com/vikashkumar21/sdd/internal/version"
)

// Kind is what will happen to one file.
type Kind string

// The possible operations.
const (
	KindCreate  Kind = "create"
	KindMerge   Kind = "merge"
	KindReplace Kind = "replace"
	KindSkip    Kind = "skip"
)

// Op is one planned file operation.
type Op struct {
	Path    string // slash-separated, relative to the project root
	Kind    Kind
	Reason  string
	Content string // file content (create/replace) or managed block (merge, block replace)
	Block   bool   // Content is a managed block rather than a whole file
	Managed bool   // the file carries a managed block and Hash covers that block only
	Hash    string // sha256 of the generated content; empty for skips
}

// Plan is the full, ordered list of operations plus the data they were built from.
type Plan struct {
	Root string
	Ops  []Op
	Data Data
}

// Lines renders the plan for humans, one line per operation.
func (p Plan) Lines() []string {
	lines := make([]string, 0, len(p.Ops))
	for _, op := range p.Ops {
		line := fmt.Sprintf("%-8s %s", op.Kind, op.Path)
		if op.Reason != "" {
			line += "  (" + op.Reason + ")"
		}
		lines = append(lines, line)
	}
	return lines
}

func sum(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// BuildPlan decides what to do for every file in the kit. It only reads the
// project; it never writes.
func BuildPlan(root string, a Answers, det detect.Result, set *templates.Set, now time.Time, force bool) (Plan, error) {
	if err := a.Normalize(); err != nil {
		return Plan{}, err
	}
	data := Data{
		Answers:         a,
		Date:            now.UTC().Format("2006-01-02"),
		ToolVersion:     version.Version,
		TemplateVersion: templates.Version,
		Greenfield:      a.Mode == ModeGreenfield,
		Brownfield:      a.Mode == ModeBrownfield,
		Detected:        det,
	}
	var titles []string
	for _, l := range a.Languages {
		pack, err := set.Pack(l)
		if err != nil {
			return Plan{}, err
		}
		data.Packs = append(data.Packs, pack)
		titles = append(titles, pack.Title)
	}
	data.LanguageList = strings.Join(titles, ", ")
	for _, id := range a.Agents {
		data.AgentFiles = append(data.AgentFiles, AgentFile(id))
	}

	p := Plan{Root: root, Data: data}

	body, err := set.Render("base/contract.md.tmpl", data)
	if err != nil {
		return Plan{}, err
	}
	body = strings.Trim(body, "\n")
	blk := block.Begin(templates.Version) + "\n" + body + "\n" + block.End

	for _, id := range a.Agents {
		op, err := agentOp(root, id, a.Project, blk, body, force)
		if err != nil {
			return Plan{}, err
		}
		p.Ops = append(p.Ops, op)
	}

	whole := []struct {
		path, tmpl string
		protect    bool
	}{
		{"MISSION.md", "base/MISSION.md.tmpl", false},
		{"TECH_STACK.md", "base/TECH_STACK.md.tmpl", false},
		{"ROADMAP.md", "base/ROADMAP.md.tmpl", false},
		{"SKILLS.md", "base/SKILLS.md.tmpl", false},
		{"README.md", "base/README.md.tmpl", true},
	}
	for _, w := range whole {
		content, err := set.Render(w.tmpl, data)
		if err != nil {
			return Plan{}, err
		}
		op, err := wholeOp(root, w.path, content, w.protect, force)
		if err != nil {
			return Plan{}, err
		}
		p.Ops = append(p.Ops, op)
	}
	for _, name := range []string{"requirements.md", "plan.md", "validation.md"} {
		content, err := set.Raw("features/" + name)
		if err != nil {
			return Plan{}, err
		}
		op, err := wholeOp(root, "features/_templates/"+name, content, false, force)
		if err != nil {
			return Plan{}, err
		}
		p.Ops = append(p.Ops, op)
	}
	return p, nil
}

func agentTitle(id, project string) string {
	switch id {
	case "claude":
		return fmt.Sprintf("# CLAUDE.md — %s agent contract (Claude)", project)
	case "gemini":
		return fmt.Sprintf("# GEMINI.md — %s agent contract (Gemini)", project)
	default:
		return fmt.Sprintf("# AGENTS.md — %s agent contract (all agents)", project)
	}
}

func agentOp(root, id, project, blk, body string, force bool) (Op, error) {
	path := AgentFile(id)
	abs := filepath.Join(root, path)
	existing, exists, err := readIfExists(abs)
	if err != nil {
		return Op{}, err
	}
	hash := sum(body)
	switch {
	case !exists:
		return Op{Path: path, Kind: KindCreate, Content: agentTitle(id, project) + "\n\n" + blk + "\n", Managed: true, Hash: hash}, nil
	case block.Has(existing):
		if force {
			return Op{Path: path, Kind: KindReplace, Reason: "managed block replaced; text outside it kept; original backed up", Content: blk, Block: true, Managed: true, Hash: hash}, nil
		}
		return Op{Path: path, Kind: KindSkip, Reason: "already contains an sdd block"}, nil
	default:
		return Op{Path: path, Kind: KindMerge, Reason: "existing file kept; managed block appended", Content: blk, Block: true, Managed: true, Hash: hash}, nil
	}
}

func wholeOp(root, path, content string, protect, force bool) (Op, error) {
	abs := filepath.Join(root, filepath.FromSlash(path))
	_, exists, err := readIfExists(abs)
	if err != nil {
		return Op{}, err
	}
	switch {
	case !exists:
		return Op{Path: path, Kind: KindCreate, Content: content, Hash: sum(content)}, nil
	case protect:
		return Op{Path: path, Kind: KindSkip, Reason: "exists; never modified"}, nil
	case force:
		return Op{Path: path, Kind: KindReplace, Reason: "replaced; original backed up", Content: content, Hash: sum(content)}, nil
	default:
		return Op{Path: path, Kind: KindSkip, Reason: "exists; left untouched (--force replaces it, with a backup)"}, nil
	}
}

func readIfExists(path string) (string, bool, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("scaffold: stat %s: %w", path, err)
	}
	if info.IsDir() {
		return "", false, fmt.Errorf("scaffold: %s is a directory, expected a file", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false, fmt.Errorf("scaffold: read %s: %w", path, err)
	}
	return string(data), true, nil
}
