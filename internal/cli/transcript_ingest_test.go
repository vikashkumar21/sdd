package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vikashkumar21/sdd/internal/state"
)

func writeSynthTranscript(t *testing.T, path string, lines []string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestTranscriptIngest_WindowAndIdempotency(t *testing.T) {
	proj := t.TempDir()
	home := t.TempDir()
	convID := "11111111-2222-3333-4444-555555555555"
	tPath := filepath.Join(home, ".gemini", "antigravity", "brain", convID, ".system_generated", "logs", "transcript.jsonl")

	lines := []string{
		`{"step_index":9,"source":"MODEL","type":"PLANNER_RESPONSE","input_tokens":999,"cache_read_tokens":999,"output_tokens":999}`,
		`{"step_index":10,"source":"MODEL","type":"PLANNER_RESPONSE","input_tokens":100,"cache_read_tokens":1000,"output_tokens":10}`,
		`{"step_index":11,"source":"SYSTEM","type":"CHECKPOINT","input_tokens":5000,"cache_read_tokens":5000,"output_tokens":5000}`,
		`{"step_index":12,"source":"MODEL","type":"PLANNER_RESPONSE","input_tokens":200,"cache_read_tokens":2000,"output_tokens":20}`,
	}
	writeSynthTranscript(t, tPath, lines)

	// First run without --from-step fails (exit 1) and writes nothing
	rFail := exec(t, "", false, home, "record-tokens", "T-064", "--dir", proj, "--conversation", convID)
	if rFail.code != 1 || !strings.Contains(rFail.err, "--from-step is required") {
		t.Fatalf("expected exit 1 requiring --from-step, got %+v", rFail)
	}

	// First ingestion from step 10
	r1 := exec(t, "", false, home, "record-tokens", "T-064", "--dir", proj, "--conversation", convID, "--from-step", "10", "--json")
	if r1.code != 0 {
		t.Fatalf("first ingest failed: %+v", r1)
	}
	var out1 TranscriptIngestResult
	if err := json.Unmarshal([]byte(r1.out), &out1); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, r1.out)
	}
	if out1.Delta.InputTokens != 300 || out1.Delta.CacheReadTokens != 3000 || out1.Delta.OutputTokens != 30 ||
		out1.FromStep != 10 || out1.ToStep != 12 || out1.Steps != 2 || !out1.Saved {
		t.Fatalf("unexpected out1: %+v", out1)
	}

	// Replay with no --from-step defaults to cursor+1 (13) -> nothing new, exit 0, totals unchanged
	r2 := exec(t, "", false, home, "record-tokens", "T-064", "--dir", proj, "--conversation", convID, "--json")
	if r2.code != 0 {
		t.Fatalf("replay failed: %+v", r2)
	}
	var out2 TranscriptIngestResult
	if err := json.Unmarshal([]byte(r2.out), &out2); err != nil {
		t.Fatalf("decode replay JSON: %v", err)
	}
	if !out2.NothingNew || out2.Saved || out2.Totals.TotalTokens != 3330 {
		t.Fatalf("unexpected replay out2: %+v", out2)
	}

	// Overlapping explicit --from-step rejected
	rOverlap := exec(t, "", false, home, "record-tokens", "T-064", "--dir", proj, "--conversation", convID, "--from-step", "12")
	if rOverlap.code != 1 {
		t.Fatalf("expected exit 1 on overlap, got %+v", rOverlap)
	}

	// Role mismatch for same source rejected
	rBadRole := exec(t, "", false, home, "record-tokens", "T-064", "--dir", proj, "--conversation", convID, "--role", "spec-auditor")
	if rBadRole.code != 1 {
		t.Fatalf("expected exit 1 on role mismatch, got %+v", rBadRole)
	}

	// Append new step 13 and ingest again (auto-resumes from 13)
	writeSynthTranscript(t, tPath, append(lines,
		`{"step_index":13,"source":"MODEL","type":"PLANNER_RESPONSE","input_tokens":50,"cache_read_tokens":500,"output_tokens":5}`,
	))
	r3 := exec(t, "", false, home, "record-tokens", "T-064", "--dir", proj, "--conversation", convID, "--json")
	if r3.code != 0 {
		t.Fatalf("second slice failed: %+v", r3)
	}
	var out3 TranscriptIngestResult
	_ = json.Unmarshal([]byte(r3.out), &out3)
	if out3.FromStep != 13 || out3.ToStep != 13 || out3.Totals.InputTokens != 350 || out3.Totals.TotalTokens != 3885 {
		t.Fatalf("unexpected out3: %+v", out3)
	}

	// Verify state.json and events.jsonl never contain home or file path
	stateBytes, _ := os.ReadFile(filepath.Join(proj, ".sdd", "state", "T-064", "state.json"))
	evBytes, _ := os.ReadFile(filepath.Join(proj, ".sdd", "state", "T-064", "events.jsonl"))
	for _, s := range []string{string(stateBytes), string(evBytes), r1.out, r3.out} {
		if strings.Contains(s, home) {
			t.Errorf("absolute home path %q leaked into %s", home, s)
		}
	}
}

func TestTranscriptIngest_DryRunAndExplicitFile(t *testing.T) {
	proj := t.TempDir()
	extDir := t.TempDir()
	tFile := filepath.Join(extDir, "headless.jsonl")
	sentinel := "SECRET_SENTINEL_IN_PROMPT_7711"
	writeSynthTranscript(t, tFile, []string{
		`{"step_index":0,"source":"MODEL","type":"PLANNER_RESPONSE","input_tokens":40,"cache_read_tokens":400,"output_tokens":4,"content":"` + sentinel + `"}`,
	})

	// Seed initial manual tokens so we verify projected dry-run totals and byte-identical preservation
	seed := exec(t, "", false, "", "record-tokens", "T-064", "--dir", proj, "--input", "10", "--cache", "100", "--output", "1")
	if seed.code != 0 {
		t.Fatalf("seed failed: %+v", seed)
	}
	statePath := filepath.Join(proj, ".sdd", "state", "T-064", "state.json")
	evPath := filepath.Join(proj, ".sdd", "state", "T-064", "events.jsonl")
	beforeState, _ := os.ReadFile(statePath)
	beforeEv, _ := os.ReadFile(evPath)

	rDry := exec(t, "", false, "", "record-tokens", "T-064", "--dir", proj, "--transcript", tFile, "--from-step", "0", "--role", "spec-auditor", "--dry-run", "--json")
	if rDry.code != 0 {
		t.Fatalf("dry-run failed: %+v", rDry)
	}
	var outDry TranscriptIngestResult
	if err := json.Unmarshal([]byte(rDry.out), &outDry); err != nil {
		t.Fatalf("decode dry JSON: %v", err)
	}
	if !outDry.DryRun || outDry.Saved || !strings.HasPrefix(outDry.Source, "file:") || outDry.Totals.InputTokens != 50 || outDry.Totals.TotalTokens != 555 {
		t.Fatalf("unexpected dry-run result: %+v", outDry)
	}

	afterState, _ := os.ReadFile(statePath)
	afterEv, _ := os.ReadFile(evPath)
	if string(beforeState) != string(afterState) || string(beforeEv) != string(afterEv) {
		t.Errorf("dry-run mutated state or events")
	}

	// Ensure neither extDir nor sentinel leaked
	if strings.Contains(rDry.out, extDir) || strings.Contains(rDry.out, sentinel) {
		t.Errorf("dry-run output leaked path or sentinel: %s", rDry.out)
	}

	// Real commit via task state alias works too
	rReal := exec(t, "", false, "", "task", "state", "record-tokens", "T-064", "--dir", proj, "--transcript", tFile, "--from-step", "0", "--role", "spec-auditor")
	if rReal.code != 0 {
		t.Fatalf("real ingest failed: %+v", rReal)
	}
	tt, ok := func() (state.TokenTelemetry, bool) {
		m, _, _ := state.ReadState(proj, "T-064")
		return state.ExtractTokens(m)
	}()
	if !ok || tt.TotalTokens != 555 {
		t.Errorf("unexpected stored totals: %+v, ok=%v", tt, ok)
	}
}

func TestTranscriptIngest_FlagsAndExitCodes(t *testing.T) {
	proj := t.TempDir()
	cases := []struct {
		name string
		args []string
		code int
	}{
		{"both sources", []string{"record-tokens", "T-064", "--dir", proj, "--conversation", "11111111-2222-3333-4444-555555555555", "--transcript", "x.jsonl"}, 2},
		{"mixed with input", []string{"record-tokens", "T-064", "--dir", proj, "--conversation", "11111111-2222-3333-4444-555555555555", "--input", "10"}, 2},
		{"transcript-root without conversation", []string{"record-tokens", "T-064", "--dir", proj, "--transcript", "x.jsonl", "--transcript-root", "/tmp"}, 2},
		{"from-step without source", []string{"record-tokens", "T-064", "--dir", proj, "--from-step", "0"}, 2},
		{"negative from-step", []string{"record-tokens", "T-064", "--dir", proj, "--transcript", "x.jsonl", "--from-step", "-1"}, 2},
		{"negative to-step", []string{"record-tokens", "T-064", "--dir", proj, "--transcript", "x.jsonl", "--from-step", "0", "--to-step", "-1"}, 2},
		{"bad role", []string{"record-tokens", "T-064", "--dir", proj, "--transcript", "x.jsonl", "--from-step", "0", "--role", "BAD ROLE!"}, 2},
		{"traversal id", []string{"record-tokens", "T-064", "--dir", proj, "--conversation", "../evil", "--from-step", "0"}, 1},
		{"empty home", []string{"record-tokens", "T-064", "--dir", proj, "--conversation", "11111111-2222-3333-4444-555555555555", "--from-step", "0"}, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			if tc.name == "empty home" {
				home = ""
			}
			r := exec(t, "", false, home, tc.args...)
			if r.code != tc.code {
				t.Errorf("got exit %d, want %d; err=%s", r.code, tc.code, r.err)
			}
		})
	}
}

func TestTranscriptIngest_BoundedToStepRootAndOverflow(t *testing.T) {
	proj := t.TempDir()
	customRoot := t.TempDir()
	convID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	tPath := filepath.Join(customRoot, convID, ".system_generated", "logs", "transcript.jsonl")
	writeSynthTranscript(t, tPath, []string{
		`{"step_index":0,"source":"MODEL","type":"PLANNER_RESPONSE","input_tokens":10,"cache_read_tokens":100,"output_tokens":1}`,
		`{"step_index":1,"source":"MODEL","type":"PLANNER_RESPONSE","input_tokens":20,"cache_read_tokens":200,"output_tokens":2}`,
		`{"step_index":2,"source":"MODEL","type":"PLANNER_RESPONSE","input_tokens":30,"cache_read_tokens":300,"output_tokens":3}`,
	})

	// --from-step > --to-step fails (exit 1)
	rBadRange := exec(t, "", false, "", "record-tokens", "T-064", "--dir", proj, "--conversation", convID, "--transcript-root", customRoot, "--from-step", "2", "--to-step", "1")
	if rBadRange.code != 1 {
		t.Fatalf("expected exit 1 on from > to, got %+v", rBadRange)
	}

	// Text-mode --dry-run over bounded 0..1 (< LastStepSeen=2)
	rDryText := exec(t, "", false, "", "record-tokens", "T-064", "--dir", proj, "--conversation", convID, "--transcript-root", customRoot, "--from-step", "0", "--to-step", "1", "--dry-run")
	if rDryText.code != 0 || !strings.Contains(rDryText.out, "[dry-run] Would ingest") {
		t.Fatalf("expected dry-run text output, got %+v", rDryText)
	}

	// Commit bounded 0..1 -> cursor stops at 1 (min(requestedTo=1, LastStepSeen=2))
	r1 := exec(t, "", false, "", "record-tokens", "T-064", "--dir", proj, "--conversation", convID, "--transcript-root", customRoot, "--from-step", "0", "--to-step", "1", "--json")
	if r1.code != 0 {
		t.Fatalf("bounded ingest failed: %+v", r1)
	}
	var out1 TranscriptIngestResult
	_ = json.Unmarshal([]byte(r1.out), &out1)
	if out1.ToStep != 1 || out1.LastStepSeen != 2 || out1.Delta.TotalTokens != 333 {
		t.Fatalf("unexpected bounded out1: %+v", out1)
	}

	// Continuation with no --from-step picks up step 2
	r2 := exec(t, "", false, "", "record-tokens", "T-064", "--dir", proj, "--conversation", convID, "--transcript-root", customRoot, "--json")
	if r2.code != 0 {
		t.Fatalf("continuation failed: %+v", r2)
	}
	var out2 TranscriptIngestResult
	_ = json.Unmarshal([]byte(r2.out), &out2)
	if out2.FromStep != 2 || out2.ToStep != 2 || out2.Totals.TotalTokens != 666 {
		t.Fatalf("unexpected continuation out2: %+v", out2)
	}

	// Replay text-mode prints "Nothing new"
	rNothing := exec(t, "", false, "", "record-tokens", "T-064", "--dir", proj, "--conversation", convID, "--transcript-root", customRoot)
	if rNothing.code != 0 || !strings.Contains(rNothing.out, "Nothing new") {
		t.Fatalf("expected Nothing new text output, got %+v", rNothing)
	}

	// --dry-run int64 overflow rejected cleanly
	ovFile := filepath.Join(t.TempDir(), "overflow.jsonl")
	writeSynthTranscript(t, ovFile, []string{
		`{"step_index":0,"source":"MODEL","type":"PLANNER_RESPONSE","input_tokens":9223372036854775807,"cache_read_tokens":1,"output_tokens":0}`,
	})
	rOv := exec(t, "", false, "", "record-tokens", "T-064", "--dir", proj, "--transcript", ovFile, "--from-step", "0", "--dry-run")
	if rOv.code != 1 || !strings.Contains(rOv.err, "overflow") {
		t.Fatalf("expected dry-run overflow error, got %+v", rOv)
	}
}
