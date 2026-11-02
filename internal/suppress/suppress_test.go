package suppress

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManifestLoadSave(t *testing.T) {
	tmp := t.TempDir()

	// Initial load when file does not exist
	m, exists, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load on non-existent file failed: %v", err)
	}
	if exists {
		t.Errorf("expected exists=false for new project")
	}
	if m.Schema != CurrentSchema || len(m.Suppressions) != 0 {
		t.Errorf("unexpected empty manifest: %+v", m)
	}

	// Add entry
	e1 := Entry{
		File:   "GEMINI.md",
		Rule:   "contract-parity",
		Reason: "local test divergence",
		Date:   "2026-11-02",
	}
	if err := Add(tmp, e1); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Load back
	m2, exists, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load after Add failed: %v", err)
	}
	if !exists {
		t.Errorf("expected exists=true after Add")
	}
	if len(m2.Suppressions) != 1 {
		t.Fatalf("expected 1 suppression, got %d", len(m2.Suppressions))
	}
	if m2.Suppressions[0] != e1 {
		t.Errorf("expected %+v, got %+v", e1, m2.Suppressions[0])
	}

	// Update existing entry with new reason
	e1Updated := Entry{
		File:   "GEMINI.md",
		Rule:   "contract-parity",
		Reason: "updated reason",
		Date:   "2026-11-03",
	}
	if err := Add(tmp, e1Updated); err != nil {
		t.Fatalf("Add update failed: %v", err)
	}
	m3, _, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load after update failed: %v", err)
	}
	if len(m3.Suppressions) != 1 || m3.Suppressions[0].Reason != "updated reason" {
		t.Errorf("expected updated entry, got %+v", m3.Suppressions)
	}

	// Add second entry and verify deterministic sorting
	e2 := Entry{
		File:   "CLAUDE.md",
		Rule:   "contract-parity",
		Reason: "claude waiver",
		Date:   "2026-11-02",
	}
	if err := Add(tmp, e2); err != nil {
		t.Fatalf("Add e2 failed: %v", err)
	}
	m4, _, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load after e2 failed: %v", err)
	}
	if len(m4.Suppressions) != 2 {
		t.Fatalf("expected 2 suppressions, got %d", len(m4.Suppressions))
	}
	// CLAUDE.md should sort before GEMINI.md
	if m4.Suppressions[0].File != "CLAUDE.md" || m4.Suppressions[1].File != "GEMINI.md" {
		t.Errorf("expected CLAUDE.md first, got: %+v", m4.Suppressions)
	}

	// Remove e2
	if err := RemoveFileRules(tmp, "CLAUDE.md", "contract-parity"); err != nil {
		t.Fatalf("RemoveFileRules failed: %v", err)
	}
	m5, _, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load after remove failed: %v", err)
	}
	if len(m5.Suppressions) != 1 || m5.Suppressions[0].File != "GEMINI.md" {
		t.Errorf("expected only GEMINI.md remaining, got: %+v", m5.Suppressions)
	}
}

func TestMatcherAndStaleDetection(t *testing.T) {
	m := Manifest{
		Schema: CurrentSchema,
		Suppressions: []Entry{
			{File: "GEMINI.md", Rule: "contract-parity", Reason: "waiver 1"},
			{File: "MISSION.md", Rule: "table-format", Reason: "waiver 2"},
			{File: "ROADMAP.md", Rule: "*", Reason: "wildcard waiver"},
		},
	}

	matcher := NewMatcher(m)

	// Match GEMINI.md
	matched, ok := matcher.Match("GEMINI.md", "contract-parity")
	if !ok || matched.Reason != "waiver 1" {
		t.Errorf("expected match for GEMINI.md, got %v (%+v)", ok, matched)
	}

	// Match ROADMAP.md with any rule due to wildcard
	matched2, ok := matcher.Match("ROADMAP.md", "some-rule")
	if !ok || matched2.Reason != "wildcard waiver" {
		t.Errorf("expected match for ROADMAP.md wildcard, got %v (%+v)", ok, matched2)
	}

	// MISSION.md table-format was NOT matched yet
	stale := matcher.Stale()
	if len(stale) != 1 || stale[0].File != "MISSION.md" {
		t.Errorf("expected MISSION.md to be stale, got %+v", stale)
	}

	// Now match MISSION.md
	_, ok = matcher.Match("MISSION.md", "table-format")
	if !ok {
		t.Errorf("expected match for MISSION.md")
	}

	// Now no stale entries
	stale2 := matcher.Stale()
	if len(stale2) != 0 {
		t.Errorf("expected 0 stale entries, got %+v", stale2)
	}
}

func TestRootConfinement(t *testing.T) {
	tmp := t.TempDir()

	// Verify that relative paths are normalized and cannot escape
	e := Entry{
		File:   "../outside.md",
		Rule:   "test",
		Reason: "outside",
	}
	// Normalized path becomes outside.md or cleaned relative path
	norm := NormalizePath(e.File)
	if strings.HasPrefix(norm, "..") {
		t.Errorf("expected path to not have parent traversal, got %s", norm)
	}

	// Check invalid schema version returns error
	invalidSchema := Manifest{
		Schema: 99,
		Suppressions: []Entry{
			{File: "test.md", Rule: "r", Reason: "res"},
		},
	}
	data, _ := json.Marshal(invalidSchema)
	sddDir := filepath.Join(tmp, ".sdd")
	_ = os.MkdirAll(sddDir, 0o755)
	_ = os.WriteFile(filepath.Join(sddDir, "suppressions.json"), data, 0o644)

	_, _, err := Load(tmp)
	if err == nil || !strings.Contains(err.Error(), "unsupported schema version") {
		t.Errorf("expected unsupported schema error, got %v", err)
	}
}
