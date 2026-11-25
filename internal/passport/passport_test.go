package passport

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var fixedTestTime = time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)

func TestGenerate(t *testing.T) {
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("resolving repo root: %v", err)
	}

	p, err := Generate(repoRoot, fixedTestTime)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if p.ProjectName != "sdd" {
		t.Errorf("expected project name 'sdd', got %q", p.ProjectName)
	}
	if p.Mission == "" {
		t.Error("expected non-empty mission")
	}
	if p.Stack == "" {
		t.Error("expected non-empty stack")
	}
	if len(p.LivingDocs) != 5 {
		t.Errorf("expected 5 living doc pointers, got %d", len(p.LivingDocs))
	}
	if p.ByteSize > MaxMarkdownBytes {
		t.Errorf("byte size %d exceeds %d", p.ByteSize, MaxMarkdownBytes)
	}
	if p.EstimatedTokens > MaxMarkdownTokens {
		t.Errorf("estimated tokens %d exceeds %d", p.EstimatedTokens, MaxMarkdownTokens)
	}

	warnings := p.Validate()
	if len(warnings) > 0 {
		t.Errorf("unexpected validation warnings: %v", warnings)
	}

	md := p.RenderMarkdown()
	if !strings.Contains(md, "# sdd — Project Passport") {
		t.Errorf("rendered markdown missing header, got:\n%s", md)
	}
	if !strings.Contains(md, "Living Documents (On-Demand)") {
		t.Errorf("rendered markdown missing Living Documents, got:\n%s", md)
	}

	jsonStr, err := p.RenderJSON()
	if err != nil {
		t.Fatalf("RenderJSON failed: %v", err)
	}
	if !strings.Contains(jsonStr, `"project_name": "sdd"`) {
		t.Errorf("json missing project_name, got:\n%s", jsonStr)
	}
}

func TestBudgetCeiling(t *testing.T) {
	tmpDir := t.TempDir()

	// Create minimal ROADMAP.md
	roadmapContent := `# ROADMAP — testproject

## 3. Milestones
| Milestone | Goal | Tasks | Est. hours | Exit criteria |
|---|---|---|---|---|
| M1 | MVP | T-001 | 3 | Done |

## 4. Tasks
### M1 — MVP
| ID | Title | Size | Dep | Description | Decision | Status |
|---|---|---|---|---|---|---|
| T-001 | Init | S | - | Initial task | - | Done |
| T-002 | Long task 2 | S | - | Long task description | - | Implementing |
| T-003 | Long task 3 | S | - | Long task description | - | Specifying |
| T-004 | Long task 4 | S | - | Long task description | - | Awaiting review |
| T-005 | Long task 5 | S | - | Long task description | - | Blocked |
`
	if err := os.WriteFile(filepath.Join(tmpDir, "ROADMAP.md"), []byte(roadmapContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create SKILLS.md with many rules including a superseded rule
	var sb strings.Builder
	sb.WriteString("# SKILLS — testproject\n\n## 8. Human feedback rules\n\n### Register\n\n")
	sb.WriteString("| ID | Date | Rule | Applies to | Source | Verified by | Status |\n")
	sb.WriteString("|---|---|---|---|---|---|---|\n")
	sb.WriteString("| F-000 | 2026-10-01 | Old superseded rule | All | Human | Check | Superseded by F-001 |\n")
	for i := 1; i <= 15; i++ {
		sb.WriteString("| F-00" + string(rune('0'+i)) + " | 2026-11-01 | Very long human feedback rule description designed to test budget bounds | All | Source | Check | Active |\n")
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "SKILLS.md"), []byte(sb.String()), 0644); err != nil {
		t.Fatal(err)
	}

	// Create MISSION.md
	missionContent := "# MISSION — testproject\n\n## 1. One-liner\n\nA comprehensive test platform for progressive disclosure.\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "MISSION.md"), []byte(missionContent), 0644); err != nil {
		t.Fatal(err)
	}

	p, err := Generate(tmpDir, fixedTestTime)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if p.ByteSize > MaxMarkdownBytes {
		t.Errorf("byte size %d exceeds %d", p.ByteSize, MaxMarkdownBytes)
	}
	if p.EstimatedTokens > MaxMarkdownTokens {
		t.Errorf("estimated tokens %d exceeds %d", p.EstimatedTokens, MaxMarkdownTokens)
	}

	// Verify superseded rule F-000 was filtered out
	for _, r := range p.ActiveRules {
		if r.ID == "F-000" {
			t.Errorf("expected F-000 to be filtered out as superseded, but was present")
		}
	}

	md := p.RenderMarkdown()
	if !strings.Contains(md, "+1 more active tasks") {
		t.Errorf("expected active tasks compaction indicator, got:\n%s", md)
	}
	if !strings.Contains(md, "+10 more rules in SKILLS.md §8") {
		t.Errorf("expected rules compaction indicator, got:\n%s", md)
	}
}

func TestDeterministicOutput(t *testing.T) {
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("resolving repo root: %v", err)
	}

	p1, err := Generate(repoRoot, fixedTestTime)
	if err != nil {
		t.Fatal(err)
	}
	p2, err := Generate(repoRoot, fixedTestTime)
	if err != nil {
		t.Fatal(err)
	}

	md1 := p1.RenderMarkdown()
	md2 := p2.RenderMarkdown()
	if md1 != md2 {
		t.Error("RenderMarkdown is not deterministic")
	}

	json1, _ := p1.RenderJSON()
	json2, _ := p2.RenderJSON()
	if json1 != json2 {
		t.Error("RenderJSON is not deterministic")
	}
}

func TestZeroClockNormalization(t *testing.T) {
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("resolving repo root: %v", err)
	}

	p, err := Generate(repoRoot, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	expected := time.Unix(0, 0).UTC()
	if !p.GeneratedAt.Equal(expected) {
		t.Errorf("expected zero time to normalize to %v, got %v", expected, p.GeneratedAt)
	}
}

func TestMissingLivingDocs(t *testing.T) {
	tmpDir := t.TempDir()
	roadmapContent := `# ROADMAP — minimal

## 3. Milestones
| Milestone | Goal | Tasks | Est. hours | Exit criteria |
|---|---|---|---|---|
| M1 | MVP | T-001 | 3 | Done |

## 4. Tasks
### M1 — MVP
| ID | Title | Size | Dep | Description | Decision | Status |
|---|---|---|---|---|---|---|
| T-001 | Init | S | - | Initial task | - | Not started |
`
	if err := os.WriteFile(filepath.Join(tmpDir, "ROADMAP.md"), []byte(roadmapContent), 0644); err != nil {
		t.Fatal(err)
	}

	// MISSION.md, TECH_STACK.md, and SKILLS.md are missing
	p, err := Generate(tmpDir, fixedTestTime)
	if err != nil {
		t.Fatalf("expected successful generation without optional docs, got: %v", err)
	}

	if p.Mission != "" {
		t.Errorf("expected empty mission, got %q", p.Mission)
	}
	if p.Stack != "" {
		t.Errorf("expected empty stack, got %q", p.Stack)
	}
	if len(p.ActiveRules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(p.ActiveRules))
	}
	if p.NextTask == nil || p.NextTask.ID != "T-001" {
		t.Errorf("expected NextTask T-001, got %+v", p.NextTask)
	}

	// LivingDocs should only contain ROADMAP.md (the only doc on disk)
	if len(p.LivingDocs) != 1 || p.LivingDocs[0].Name != "ROADMAP.md" {
		t.Errorf("expected only ROADMAP.md in LivingDocs, got %+v", p.LivingDocs)
	}

	md := p.RenderMarkdown()
	if !strings.Contains(md, "Next Task") {
		t.Errorf("expected Next Task section in markdown, got:\n%s", md)
	}
}

func TestMissingRoadmap(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := Generate(tmpDir, fixedTestTime)
	if err == nil {
		t.Fatal("expected error when ROADMAP.md is missing, got nil")
	}
}

func TestPolyglotStackAndCRLF(t *testing.T) {
	tmpDir := t.TempDir()

	roadmapContent := "# ROADMAP — polyglot\r\n\r\n## 4. Tasks\r\n### M1 — Poly\r\n| ID | Title | Size | Dep | Description | Decision | Status |\r\n|---|---|---|---|---|---|---|\r\n| T-001 | Init | S | - | First | - | Done |\r\n"
	_ = os.WriteFile(filepath.Join(tmpDir, "ROADMAP.md"), []byte(roadmapContent), 0644)

	techContent := "# TECH_STACK — polyglot\r\n\r\n## 1. Stack at a glance\r\n\r\n| Layer | Choice | Status | Decision |\r\n|---|---|---|---|\r\n| Language | Go | Approved | D-001 |\r\n| Language | TypeScript | Approved | D-002 |\r\n| Build | make build | Approved | D-003 |\r\n"
	_ = os.WriteFile(filepath.Join(tmpDir, "TECH_STACK.md"), []byte(techContent), 0644)

	p, err := Generate(tmpDir, fixedTestTime)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(p.Stack, "Go; TypeScript") {
		t.Errorf("expected combined polyglot languages, got: %s", p.Stack)
	}
}

func TestValidateWarnings(t *testing.T) {
	p := &Passport{
		EstimatedTokens: MaxMarkdownTokens + 10,
		ByteSize:        MaxMarkdownBytes + 50,
	}
	warnings := p.Validate()
	if len(warnings) != 2 {
		t.Errorf("expected 2 warnings for oversized passport, got %d: %v", len(warnings), warnings)
	}
}

func BenchmarkGenerate(b *testing.B) {
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		b.Fatalf("resolving repo root: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Generate(repoRoot, fixedTestTime)
		if err != nil {
			b.Fatal(err)
		}
	}
}
