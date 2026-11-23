package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vikashkumar21/sdd/internal/cli"
)

func TestDocUsageAndErrors(t *testing.T) {
	var out, err bytes.Buffer
	env := cli.IO{Out: &out, Err: &err}

	// 1. Missing subcommand
	code := cli.Run([]string{"doc"}, env)
	if code != 2 {
		t.Fatalf("expected code 2 for missing doc subcommand, got %d", code)
	}
	if !strings.Contains(err.String(), "usage: sdd doc") {
		t.Errorf("expected usage string in err, got %q", err.String())
	}

	// 2. Unknown subcommand
	out.Reset()
	err.Reset()
	code = cli.Run([]string{"doc", "unknown"}, env)
	if code != 2 {
		t.Fatalf("expected code 2 for unknown subcommand, got %d", code)
	}

	// 3. Flow without name
	out.Reset()
	err.Reset()
	code = cli.Run([]string{"doc", "flow"}, env)
	if code != 2 {
		t.Fatalf("expected code 2 for flow without name, got %d", code)
	}
}

func TestDocInit_DryRunAndApply(t *testing.T) {
	tmp := t.TempDir()
	var out, err bytes.Buffer
	env := cli.IO{Out: &out, Err: &err}

	// Dry run
	code := cli.Run([]string{"doc", "init", "--dir", tmp, "--dry-run"}, env)
	if code != 0 {
		t.Fatalf("doc init --dry-run failed with code %d: %s", code, err.String())
	}
	if !strings.Contains(out.String(), "Planned documentation scaffolding (dry-run):") {
		t.Errorf("expected dry-run header, got: %s", out.String())
	}

	// Verify nothing was written in dry run
	if _, serr := os.Stat(filepath.Join(tmp, "docs")); serr == nil {
		t.Errorf("docs directory created during dry run")
	}

	// Apply
	out.Reset()
	err.Reset()
	code = cli.Run([]string{"doc", "init", "--dir", tmp}, env)
	if code != 0 {
		t.Fatalf("doc init failed with code %d: %s", code, err.String())
	}
	if !strings.Contains(out.String(), "[created] docs/architecture/overview.md") {
		t.Errorf("expected created message in out, got: %s", out.String())
	}

	// Verify files exist on disk
	archOverview := filepath.Join(tmp, "docs", "architecture", "overview.md")
	if _, serr := os.Stat(archOverview); serr != nil {
		t.Errorf("file %s not created: %v", archOverview, serr)
	}

	// Second run skips
	out.Reset()
	err.Reset()
	code = cli.Run([]string{"doc", "init", "--dir", tmp}, env)
	if code != 0 {
		t.Fatalf("doc init second run failed with code %d: %s", code, err.String())
	}
	if !strings.Contains(out.String(), "[skipped] docs/architecture/overview.md") {
		t.Errorf("expected skipped message in out, got: %s", out.String())
	}
}

func TestDocFlow_Scaffold(t *testing.T) {
	tmp := t.TempDir()
	var out, err bytes.Buffer
	env := cli.IO{Out: &out, Err: &err}

	code := cli.Run([]string{"doc", "flow", "payment-checkout", "--title", "Payment Checkout Flow", "--dir", tmp}, env)
	if code != 0 {
		t.Fatalf("doc flow failed with code %d: %s", code, err.String())
	}
	if !strings.Contains(out.String(), "[created] docs/business_flows/payment-checkout.md") {
		t.Errorf("expected created message in out, got: %s", out.String())
	}

	flowFile := filepath.Join(tmp, "docs", "business_flows", "payment-checkout.md")
	content, rerr := os.ReadFile(flowFile)
	if rerr != nil {
		t.Fatalf("read flow file failed: %v", rerr)
	}
	if !strings.Contains(string(content), "Payment Checkout Flow") {
		t.Errorf("expected title in flow file, got: %s", string(content))
	}
}

func TestDoc_InjectedClockPropagates(t *testing.T) {
	tmp := t.TempDir()
	var out, err bytes.Buffer
	fixedDate := time.Date(2027, 3, 14, 15, 9, 26, 0, time.UTC)
	env := cli.IO{
		Out: &out,
		Err: &err,
		Now: func() time.Time { return fixedDate },
	}

	code := cli.Run([]string{"doc", "init", "--dir", tmp}, env)
	if code != 0 {
		t.Fatalf("doc init failed: %v", err.String())
	}

	overview := filepath.Join(tmp, "docs", "architecture", "overview.md")
	data, rerr := os.ReadFile(overview)
	if rerr != nil {
		t.Fatalf("read overview failed: %v", rerr)
	}
	if !strings.Contains(string(data), "last_reviewed: 2027-03-14") {
		t.Errorf("expected injected date 2027-03-14 in frontmatter, got:\n%s", string(data))
	}

	out.Reset()
	err.Reset()
	code = cli.Run([]string{"doc", "flow", "auth-login", "--title", "Auth Login Flow", "--dir", tmp}, env)
	if code != 0 {
		t.Fatalf("doc flow failed: %v", err.String())
	}
	flowFile := filepath.Join(tmp, "docs", "business_flows", "auth-login.md")
	data, rerr = os.ReadFile(flowFile)
	if rerr != nil {
		t.Fatalf("read flow failed: %v", rerr)
	}
	if !strings.Contains(string(data), "last_reviewed: 2027-03-14") {
		t.Errorf("expected injected date 2027-03-14 in flow frontmatter, got:\n%s", string(data))
	}
}
