package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/vikashkumar21/sdd/internal/context"
)

func TestPromptWithoutContext(t *testing.T) {
	root := t.TempDir()
	exec(t, "", false, "", "init", "--yes", "--dir", root, "--name", "nocontext")

	p := exec(t, "", false, "", "prompt", "gemini", "--dir", root)
	if p.code != 0 {
		t.Fatalf("prompt failed: %+v", p)
	}

	if strings.Contains(p.out, "Staged external context") {
		t.Errorf("prompt should omit context section when no context is staged:\n%s", p.out)
	}
	if strings.Contains(p.out, "{{") || strings.Contains(p.out, "<no value>") {
		t.Errorf("prompt has template leftovers")
	}
}

func TestPromptWithStagedContext(t *testing.T) {
	root := t.TempDir()
	exec(t, "", false, "", "init", "--yes", "--dir", root, "--name", "withcontext")

	item := context.Item{
		Source: "raw-text",
		Origin: "meeting-transcript",
		Data:   []byte("Client requirements discussion"),
	}
	if _, err := context.Stage(root, item, time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}

	p := exec(t, "", false, "", "prompt", "claude", "--dir", root)
	if p.code != 0 {
		t.Fatalf("prompt failed: %+v", p)
	}

	for _, want := range []string{
		"Staged external context (.sdd/context/)",
		"meeting-transcript",
		"Extraction and synthesis instructions:",
		"sdd passport",
		"MISSION.md",
		"TECH_STACK.md",
		"ROADMAP.md",
		"GATE 1 IS STRICT",
	} {
		if !strings.Contains(p.out, want) {
			t.Errorf("prompt output missing expected section %q:\n%s", want, p.out)
		}
	}
	if strings.Contains(p.out, "{{") || strings.Contains(p.out, "<no value>") {
		t.Errorf("prompt has template leftovers")
	}
}

func TestPromptWithTaskContext(t *testing.T) {
	root := t.TempDir()
	exec(t, "", false, "", "init", "--yes", "--dir", root, "--name", "taskcontext")

	item := context.Item{
		Source: "file",
		Origin: "jira-101.json",
		Task:   "T-001",
		Data:   []byte(`{"issue": "T-001", "summary": "Initial requirements"}`),
	}
	if _, err := context.Stage(root, item, time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}

	p := exec(t, "", false, "", "prompt", "gemini", "--dir", root, "--task", "T-001")
	if p.code != 0 {
		t.Fatalf("prompt failed: %+v", p)
	}

	for _, want := range []string{
		"Staged external context (.sdd/context/)",
		"Context assigned to focus task (T-001):",
		"jira-101.json",
		"tasks/<typology>/T-xxx-<slug>/requirements.md",
	} {
		if !strings.Contains(p.out, want) {
			t.Errorf("prompt output missing expected section %q:\n%s", want, p.out)
		}
	}
}

func TestPromptWithSkills(t *testing.T) {
	root := t.TempDir()
	exec(t, "", false, "", "init", "--yes", "--dir", root, "--name", "skillprompt")

	// Add a skill
	r := exec(t, "", false, "", "skill", "new", "deploy-checker", "--dir", root, "--description", "Validate production readiness")
	if r.code != 0 {
		t.Fatalf("skill new failed: %s", r.err)
	}

	p := exec(t, "", false, "", "prompt", "gemini", "--dir", root)
	if p.code != 0 {
		t.Fatalf("prompt failed: %+v", p)
	}

	for _, want := range []string{
		"Available Agent Skills (agentskills.io)",
		"`deploy-checker`",
		"Validate production readiness",
		"progressive disclosure: consult full `SKILL.md` when relevant",
	} {
		if !strings.Contains(p.out, want) {
			t.Errorf("prompt output missing expected skill text %q:\n%s", want, p.out)
		}
	}
	if strings.Contains(p.out, "Some skills contain raw ingested notes") {
		t.Errorf("prompt should not have refinement alert when skill has no raw notes")
	}
}

func TestPromptWithRefinementSkills(t *testing.T) {
	root := t.TempDir()
	exec(t, "", false, "", "init", "--yes", "--dir", root, "--name", "refineprompt")

	// Add a raw skill via --from-text
	r := exec(t, "", false, "", "skill", "new", "raw-guide", "--dir", root, "--from-text", "Step A: run tests; Step B: run lint")
	if r.code != 0 {
		t.Fatalf("skill new failed: %s", r.err)
	}

	p := exec(t, "", false, "", "prompt", "claude", "--dir", root)
	if p.code != 0 {
		t.Fatalf("prompt failed: %+v", p)
	}

	for _, want := range []string{
		"Available Agent Skills (agentskills.io)",
		"`raw-guide`",
		"Some skills contain raw ingested notes marked with `<!-- sdd:skill-refine:begin -->`",
		"review and refine them into standard sections",
	} {
		if !strings.Contains(p.out, want) {
			t.Errorf("prompt output missing refinement text %q:\n%s", want, p.out)
		}
	}
}

func TestPromptStaticPrefix(t *testing.T) {
	root := t.TempDir()
	exec(t, "", false, "", "init", "--yes", "--dir", root, "--name", "prefixproj")

	p := exec(t, "", false, "", "prompt", "gemini", "--dir", root)
	if p.code != 0 {
		t.Fatalf("prompt execution failed: %+v", p)
	}

	// Verify order of sections to guarantee KV cache prefix invariance:
	// Static Invariants:
	idxCoordinator := strings.Index(p.out, "acting as the **Task Coordinator**")
	idxAnthem := strings.Index(p.out, "ANTHEM (non-negotiable)")
	idxStep0 := strings.Index(p.out, "## 1. Step 0 - read the project")
	idxStep1 := strings.Index(p.out, "## 2. Step 1 - verify repository state")
	idxStep2 := strings.Index(p.out, "## 3. Step 2 - act on exactly one branch")
	idxHardRules := strings.Index(p.out, "## 4. Hard rules")
	idxGateReport := strings.Index(p.out, "## 5. How you report at every gate")
	// Dynamic Invariants:
	idxDynamicState := strings.Index(p.out, "Dynamic Project State & Context")
	idxDate := strings.Index(p.out, "roadmap snapshot below was taken on")
	idxHumanInputs := strings.Index(p.out, "HUMAN INPUTS")

	for name, idx := range map[string]int{
		"Coordinator":  idxCoordinator,
		"Anthem":       idxAnthem,
		"Step0":        idxStep0,
		"Step1":        idxStep1,
		"Step2":        idxStep2,
		"HardRules":    idxHardRules,
		"GateReport":   idxGateReport,
		"DynamicState": idxDynamicState,
		"Date":         idxDate,
		"HumanInputs":  idxHumanInputs,
	} {
		if idx == -1 {
			t.Fatalf("missing section %s in prompt output:\n%s", name, p.out)
		}
	}

	// Assert strict prefix ordering:
	validOrder := idxCoordinator < idxAnthem &&
		idxAnthem < idxStep0 &&
		idxStep0 < idxStep1 &&
		idxStep1 < idxStep2 &&
		idxStep2 < idxHardRules &&
		idxHardRules < idxGateReport &&
		idxGateReport < idxDynamicState &&
		idxDynamicState <= idxDate &&
		idxDate < idxHumanInputs
	if !validOrder {
		t.Errorf("prompt section ordering violates static-first KV cache prefix pinning:\nCoordinator: %d, Anthem: %d, Step0: %d, Step1: %d, Step2: %d, HardRules: %d, GateReport: %d, DynamicState: %d, Date: %d, HumanInputs: %d",
			idxCoordinator, idxAnthem, idxStep0, idxStep1, idxStep2, idxHardRules, idxGateReport, idxDynamicState, idxDate, idxHumanInputs)
	}
}
