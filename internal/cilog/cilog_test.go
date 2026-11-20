package cilog

import (
	"strings"
	"testing"
)

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain text",
			input: "hello world\nthis is clean",
			want:  "hello world\nthis is clean",
		},
		{
			name:  "standard color codes",
			input: "\x1b[31mFAIL:\x1b[0m \x1b[1;32mTestFoo\x1b[0m (0.05s)",
			want:  "FAIL: TestFoo (0.05s)",
		},
		{
			name:  "cursor and clear sequences",
			input: "Progress: 50%\x1b[2K\rProgress: 100%\nDone",
			want:  "Progress: 50%\nProgress: 100%\nDone",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripANSI(tt.input)
			if got != tt.want {
				t.Errorf("StripANSI() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSanitizeSecrets(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantContains string
		wantRedacted bool
		wantCount    int
	}{
		{
			name:         "AWS Access Key",
			input:        "Connecting using AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE to us-east-1",
			wantContains: "AWS_ACCESS_KEY_ID=[AWS_KEY_REDACTED]",
			wantRedacted: true,
			wantCount:    1,
		},
		{
			name:         "GitHub Classic Token",
			input:        "git remote set-url origin https://ghp_abcdefghijklmnopqrstuvwxyz012345@github.com/org/repo",
			wantContains: "https://ghp_[REDACTED]@github.com/org/repo",
			wantRedacted: true,
			wantCount:    1,
		},
		{
			name:         "GitHub Fine-Grained PAT",
			input:        "export GITHUB_TOKEN=github_pat_11AAAAAAA0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ012345678901",
			wantContains: "export GITHUB_TOKEN=github_pat_[REDACTED]",
			wantRedacted: true,
			wantCount:    1,
		},
		{
			name:         "Bearer Authorization Header",
			input:        "curl -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.e30.xyz' https://api.example.com",
			wantContains: "Authorization: Bearer [REDACTED]",
			wantRedacted: true,
			wantCount:    1,
		},
		{
			name: "Private Key Block",
			input: `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEA0Y1+abcdefghijklmnopqrstuvwxyz
-----END RSA PRIVATE KEY-----`,
			wantContains: "[PRIVATE_KEY_REDACTED]",
			wantRedacted: true,
			wantCount:    1,
		},
		{
			name:         "AWS STS Temporary Key",
			input:        "AWS_SESSION_TOKEN=ASIAPROP1234567890AB",
			wantContains: "[AWS_KEY_REDACTED]",
			wantRedacted: true,
			wantCount:    1,
		},
		{
			name:         "GitHub Runner Registration Token",
			input:        "RUNNER_TOKEN=ghr_1234567890abcdef1234567890abcdef",
			wantContains: "ghr_[REDACTED]",
			wantRedacted: true,
			wantCount:    1,
		},
		{
			name:         "Bearer Authorization Header with Base64 Padding",
			input:        "curl -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.e30.xyz==' https://api.example.com",
			wantContains: "Authorization: Bearer [REDACTED]",
			wantRedacted: true,
			wantCount:    1,
		},
		{
			name: "PGP Private Key Block",
			input: `-----BEGIN PGP PRIVATE KEY BLOCK-----
Version: GnuPG v2
mQENBF...
-----END PGP PRIVATE KEY BLOCK-----`,
			wantContains: "[PRIVATE_KEY_REDACTED]",
			wantRedacted: true,
			wantCount:    1,
		},
		{
			name:         "Database URI Userinfo Password",
			input:        "DATABASE_URL=postgres://user:superSecretPassword123@db.internal:5432/prod",
			wantContains: "postgres://user:[REDACTED]@",
			wantRedacted: true,
			wantCount:    1,
		},
		{
			name:         "Key-Value Secret Assignment",
			input:        "export DB_PASSWORD=\"SuperSecretPassword1234\"\nexport API_KEY='abcdef1234567890'",
			wantContains: "[REDACTED]",
			wantRedacted: true,
			wantCount:    2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, count := SanitizeSecrets(tt.input)
			if tt.wantRedacted && !strings.Contains(got, tt.wantContains) {
				t.Errorf("SanitizeSecrets() result %q does not contain %q", got, tt.wantContains)
			}
			if tt.wantRedacted && count != tt.wantCount {
				t.Errorf("SanitizeSecrets() redactions = %d, want %d", count, tt.wantCount)
			}
		})
	}
}

func TestExtractDiagnostics_GoTest(t *testing.T) {
	logContent := `
GITHUB_ACTION=run-tests
$ go test -v ./...
=== RUN   TestLogin
--- PASS: TestLogin (0.01s)
=== RUN   TestProcessOrder
    order_test.go:42: unexpected order status: want 200, got 500
--- FAIL: TestProcessOrder (0.02s)
FAIL
FAIL	github.com/example/orders	0.035s
FAIL
`
	diag, clean, err := Parse([]byte(logContent))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if diag.Platform != "github" {
		t.Errorf("Platform = %q, want github", diag.Platform)
	}
	if len(diag.FailedTests) != 1 || diag.FailedTests[0] != "TestProcessOrder" {
		t.Errorf("FailedTests = %v, want [TestProcessOrder]", diag.FailedTests)
	}
	if !strings.Contains(diag.PrimaryError, "TestProcessOrder") {
		t.Errorf("PrimaryError = %q, want containing TestProcessOrder", diag.PrimaryError)
	}
	if !strings.Contains(diag.FailureSnippet, "unexpected order status") {
		t.Errorf("FailureSnippet missing failure line, got:\n%s", diag.FailureSnippet)
	}
	if !strings.Contains(clean, "FAIL: TestProcessOrder") {
		t.Errorf("clean output missing test result")
	}
}

func TestExtractDiagnostics_Pytest(t *testing.T) {
	logContent := `
Running with gitlab-runner 16.0.0
$ pytest tests/
============================= test session starts ==============================
tests/test_api.py::test_auth PASSED
tests/test_api.py::test_payment FAILED
=================================== FAILURES ===================================
_________________________________ test_payment _________________________________
    def test_payment():
>       assert response.status_code == 200
E       assert 500 == 200
tests/test_api.py:55: AssertionError
=========================== short test summary info ============================
FAILED tests/test_api.py::test_payment - assert 500 == 200
`
	diag, _, err := Parse([]byte(logContent))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if diag.Platform != "gitlab" {
		t.Errorf("Platform = %q, want gitlab", diag.Platform)
	}
	if len(diag.FailedTests) == 0 || !strings.Contains(diag.FailedTests[0], "test_payment") {
		t.Errorf("FailedTests = %v, want containing test_payment", diag.FailedTests)
	}
	if !strings.Contains(diag.FailureSnippet, "AssertionError") {
		t.Errorf("FailureSnippet missing assertion error:\n%s", diag.FailureSnippet)
	}
}

func TestExtractDiagnostics_Jest(t *testing.T) {
	logContent := `
Bitbucket Pipelines
$ npm test
FAIL src/auth.test.ts
  ● Authentication › should reject expired token
    expect(received).toBe(expected) // Object.is equality
    Expected: 401
    Received: 200
`
	diag, _, err := Parse([]byte(logContent))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if diag.Platform != "bitbucket" {
		t.Errorf("Platform = %q, want bitbucket", diag.Platform)
	}
	if len(diag.FailedTests) == 0 {
		t.Fatalf("expected failed tests detected for Jest, got %v", diag.FailedTests)
	}
}

func TestExtractDiagnostics_SDDCheck(t *testing.T) {
	logContent := `
$ sdd check --strict
ERROR ROADMAP.md:42  T-042 has invalid status "Pending"
ERROR internal/ci/ci.go:12  unapproved dependency found
sdd check: 2 error(s), 0 warning(s)
`
	diag, _, err := Parse([]byte(logContent))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !strings.Contains(diag.PrimaryError, "sdd check failure") {
		t.Errorf("PrimaryError = %q, want sdd check failure", diag.PrimaryError)
	}
}

func TestExtractDiagnostics_Maven(t *testing.T) {
	logContent := `
[INFO] Scanning for projects...
[INFO] Running com.example.OrderServiceTest
[ERROR] Failures: 
[ERROR]   OrderServiceTest.testPaymentFailure:56 expected:<200> but was:<500>
[INFO] 
[ERROR] Tests run: 8, Failures: 1, Errors: 0, Skipped: 0
[INFO] BUILD FAILURE
`
	diag, _, err := Parse([]byte(logContent))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !strings.Contains(diag.PrimaryError, "Maven build/test verification failure") {
		t.Errorf("PrimaryError = %q, want Maven verification failure", diag.PrimaryError)
	}
	if !strings.Contains(diag.FailureSnippet, "OrderServiceTest.testPaymentFailure") {
		t.Errorf("FailureSnippet missing test failure details:\n%s", diag.FailureSnippet)
	}
}

func TestParse_EmptyLog(t *testing.T) {
	_, _, err := Parse([]byte(""))
	if err == nil || !strings.Contains(err.Error(), "cilog: input is empty") {
		t.Errorf("expected 'cilog: input is empty' for empty bytes, got: %v", err)
	}

	_, _, err = Parse(nil)
	if err == nil || !strings.Contains(err.Error(), "cilog: input is empty") {
		t.Errorf("expected 'cilog: input is empty' for nil bytes, got: %v", err)
	}
}

func TestParse_OversizedLog(t *testing.T) {
	oversized := make([]byte, MaxLogBytes+10)
	_, _, err := Parse(oversized)
	if err == nil {
		t.Errorf("expected error for log exceeding %d bytes, got nil", MaxLogBytes)
	}
}

func TestSeedBugfixReproduction(t *testing.T) {
	template := `# Bug Reproduction & Diagnosis — T-xxx <title>

## 1. Problem Summary
- **Observed Behavior:** <What actually happens? Include error messages, unexpected return values, or crash logs>
- **Expected Behavior:** <What should happen according to specifications?>
- **Severity / Impact:** <Blocker / High / Medium / Low — user impact>

## 2. Environment & Preconditions
- OS & Architecture: <e.g. macOS arm64, Linux amd64, Windows>

## 3. Reproduction Steps
1. <Step 1>
2. <Step 2>
3. <Step 3: Trigger condition>

## 4. Mandatory Step 0: Minimal Failing Test Case
- **Test File & Function:** <e.g. internal/pkg/feature_test.go:TestReproduceIssue>
- **Command to Execute:** <e.g. go test -v ./internal/pkg -run TestReproduceIssue>
- **Observed Failure Output (unpatched code):**
<paste real failure output here proving reproduction>`

	diag := &Diagnostics{
		Platform:       "github",
		FailedStep:     "go test -v ./...",
		PrimaryError:   "Go test TestProcessOrder failed",
		FailedTests:    []string{"TestProcessOrder"},
		FailureSnippet: "order_test.go:42: unexpected order status: want 200, got 500",
	}

	seeded := SeedBugfixReproduction(diag, template)

	if !strings.Contains(seeded, "Go test TestProcessOrder failed") {
		t.Errorf("seeded missing primary error:\n%s", seeded)
	}
	if !strings.Contains(seeded, "CI Runner (github container)") {
		t.Errorf("seeded missing runner environment:\n%s", seeded)
	}
	if !strings.Contains(seeded, "TestProcessOrder") {
		t.Errorf("seeded missing test function name")
	}
	if !strings.Contains(seeded, "unexpected order status: want 200, got 500") {
		t.Errorf("seeded missing failure snippet")
	}
}
