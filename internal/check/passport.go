package check

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vikashkumar21/sdd/internal/passport"
	"github.com/vikashkumar21/sdd/internal/roadmap"
)

func (c *checker) checkPassportConsistency(tasks []roadmap.Task) {
	roadmapPath := filepath.Join(c.root, "ROADMAP.md")
	if _, err := os.Stat(roadmapPath); err != nil {
		// Skip cleanly if ROADMAP.md is missing (e.g. greenfield / partial dirs)
		return
	}
	if len(tasks) == 0 {
		return
	}

	p, err := passport.Generate(c.root, time.Unix(0, 0).UTC())
	if err != nil {
		c.add(Error, "ROADMAP.md", 0, "passport-generate", "%v", err)
		return
	}

	// Budget validation
	if p.EstimatedTokens > passport.MaxMarkdownTokens {
		c.add(Warn, "ROADMAP.md", 0, "passport-budget",
			"passport estimated tokens %d exceeds budget limit %d", p.EstimatedTokens, passport.MaxMarkdownTokens)
	}
	if p.ByteSize > passport.MaxMarkdownBytes {
		c.add(Warn, "ROADMAP.md", 0, "passport-budget",
			"passport byte size %d exceeds budget limit %d", p.ByteSize, passport.MaxMarkdownBytes)
	}

	// Confinement and link validation
	rootHandle, rerr := os.OpenRoot(c.root)
	if rerr != nil {
		c.add(Error, "ROADMAP.md", 0, "passport-root", "cannot open project root: %v", rerr)
		return
	}
	defer func() { _ = rootHandle.Close() }()

	for _, doc := range p.LivingDocs {
		clean := filepath.Clean(filepath.FromSlash(strings.TrimSpace(doc.Path)))
		if !filepath.IsLocal(clean) {
			c.add(Error, doc.Path, 0, "passport-link", "living document path %q escapes project root", doc.Path)
			continue
		}
		if _, err := rootHandle.Stat(clean); err != nil {
			c.add(Warn, doc.Path, 0, "passport-link", "living document %s linked in passport does not exist", doc.Path)
		}
	}

	// Task synchronization validation
	var activeTaskIDs []string
	for _, t := range tasks {
		kind := roadmap.Kind(t.Status)
		switch kind {
		case "specifying", "awaiting plan approval", "implementing", "awaiting review", "blocked":
			activeTaskIDs = append(activeTaskIDs, t.ID)
		}
	}

	if len(activeTaskIDs) != len(p.ActiveTasks) {
		c.add(Error, "ROADMAP.md", 0, "passport-tasks",
			"passport active tasks count (%d) out of sync with ROADMAP.md (%d)", len(p.ActiveTasks), len(activeTaskIDs))
	} else {
		for i, t := range p.ActiveTasks {
			if i < len(activeTaskIDs) && t.ID != activeTaskIDs[i] {
				c.add(Error, "ROADMAP.md", 0, "passport-tasks",
					"passport active task %s does not match ROADMAP.md task %s", t.ID, activeTaskIDs[i])
				break
			}
		}
	}
}
