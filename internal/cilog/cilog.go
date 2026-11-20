// Package cilog provides high-speed ANSI escape code stripping,
// multi-vector credential and secret sanitization, failure diagnostic extraction,
// and bugfix reproduction document seeding for remote CI failure logs.
package cilog

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Maximum allowed CI log size (10MB, aligns with context.MaxInputBytes).
const MaxLogBytes = 10 * 1024 * 1024

var (
	// ANSI sequence regexes
	ansiCSIRe   = regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z]`)
	ansiOscRe   = regexp.MustCompile(`\x1b\][^\x07\x1b]*(\x07|\x1b\\)`)
	ansiOtherRe = regexp.MustCompile(`\x1b[@-Z\\-_]`)

	// Secret sanitization regexes
	awsKeyRe    = regexp.MustCompile(`\b(AKIA|ASIA|ABIA|ACCA)[0-9A-Z]{16}\b`)
	awsSecretRe = regexp.MustCompile(`(?i)(aws_secret_access_key|aws_secret_key)\s*[:=]\s*["']?([A-Za-z0-9/+=]{40})["']?`)

	ghTokenRe    = regexp.MustCompile(`\b(ghp|gho|ghu|ghs|ghr)_[A-Za-z0-9_]{16,255}\b`)
	ghPatRe      = regexp.MustCompile(`\bgithub_pat_[A-Za-z0-9_]{16,255}\b`)
	bearerAuthRe = regexp.MustCompile(`(?i)(\b(?:authorization:\s*)?bearer\s+)([A-Za-z0-9_\-\.+/=~]{8,})`)
	basicAuthRe  = regexp.MustCompile(`(?i)(\b(?:authorization:\s*)?basic\s+)([A-Za-z0-9+/=]{8,})`)

	privKeyBlockRe = regexp.MustCompile(`-----BEGIN [A-Z ]+ PRIVATE KEY( BLOCK)?-----[\s\S]*?-----END [A-Z ]+ PRIVATE KEY( BLOCK)?-----`)
	genericSecRe   = regexp.MustCompile(`(?i)(password|secret|api_key|apikey|private_key|access_token|client_secret)\s*[:=]\s*["']?([^\s"']{8,})["']?`)
	uriUserInfoRe  = regexp.MustCompile(`(?i)(://[^:\s@]+):([^@\s]+)(@)`)

	// Diagnostics regexes
	goTestFailRe     = regexp.MustCompile(`^--- FAIL:\s*(\S+)`)
	goPkgFailRe      = regexp.MustCompile(`^FAIL\s+(\S+)`)
	pytestFailRe     = regexp.MustCompile(`FAILED\s+(\S+?::\S+)`)
	jestFailRe       = regexp.MustCompile(`(?:●\s+FAIL\s+(\S+)|●\s+(.+?)\s+›\s+(.+))`)
	mvnFailRe        = regexp.MustCompile(`\[ERROR\]\s+(?:Failures:\s*|Tests run:.*Failures:\s*([1-9]\d*))`)
	sddCheckErrorRe  = regexp.MustCompile(`^ERROR\s+(\S+?:\d+)\s+(.*)`)
	stepCommandRunRe = regexp.MustCompile(`(?i)(?:^##\[group\]Run\s+(.*)|^Running\s+(?:command:\s*)?(.*)|\$ (make\s+\S+|go\s+\S+|npm\s+\S+|mvn\s+\S+|pytest\s+\S+|python\s+\S+|sdd\s+\S+))`)
)

// Diagnostics holds extracted metadata from a CI failure log.
type Diagnostics struct {
	Platform        string   `json:"platform"`
	FailedStep      string   `json:"failed_step"`
	PrimaryError    string   `json:"primary_error"`
	FailedTests     []string `json:"failed_tests"`
	FailureSnippet  string   `json:"failure_snippet"`
	RawLength       int      `json:"raw_length"`
	SanitizedLength int      `json:"sanitized_length"`
	RedactionsCount int      `json:"redactions_count"`
}

// StripANSI removes all ANSI escape sequences, color codes, and terminal formatting characters.
func StripANSI(input string) string {
	if input == "" {
		return ""
	}
	s := ansiCSIRe.ReplaceAllString(input, "")
	s = ansiOscRe.ReplaceAllString(s, "")
	s = ansiOtherRe.ReplaceAllString(s, "")
	// Normalize carriage returns (common in CI progress animations)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return s
}

// SanitizeSecrets redacts sensitive credentials, tokens, AWS keys, GitHub tokens, and private keys.
// Returns the sanitized string and the count of redactions performed.
func SanitizeSecrets(input string) (string, int) {
	if input == "" {
		return "", 0
	}

	count := 0

	// 1. Private keys
	if privKeyBlockRe.MatchString(input) {
		matches := privKeyBlockRe.FindAllStringIndex(input, -1)
		count += len(matches)
		input = privKeyBlockRe.ReplaceAllString(input, "[PRIVATE_KEY_REDACTED]")
	}

	// 2. AWS keys
	if awsKeyRe.MatchString(input) {
		matches := awsKeyRe.FindAllStringIndex(input, -1)
		count += len(matches)
		input = awsKeyRe.ReplaceAllString(input, "[AWS_KEY_REDACTED]")
	}

	if awsSecretRe.MatchString(input) {
		matches := awsSecretRe.FindAllStringIndex(input, -1)
		count += len(matches)
		input = awsSecretRe.ReplaceAllString(input, "$1 = [REDACTED]")
	}

	// 3. GitHub tokens
	if ghTokenRe.MatchString(input) {
		matches := ghTokenRe.FindAllStringIndex(input, -1)
		count += len(matches)
		input = ghTokenRe.ReplaceAllString(input, "${1}_[REDACTED]")
	}

	if ghPatRe.MatchString(input) {
		matches := ghPatRe.FindAllStringIndex(input, -1)
		count += len(matches)
		input = ghPatRe.ReplaceAllString(input, "github_pat_[REDACTED]")
	}

	// 4. Authorization headers
	if bearerAuthRe.MatchString(input) {
		matches := bearerAuthRe.FindAllStringIndex(input, -1)
		count += len(matches)
		input = bearerAuthRe.ReplaceAllString(input, "${1}[REDACTED]")
	}

	if basicAuthRe.MatchString(input) {
		matches := basicAuthRe.FindAllStringIndex(input, -1)
		count += len(matches)
		input = basicAuthRe.ReplaceAllString(input, "${1}[REDACTED]")
	}

	// 5. Generic credentials / password assignments
	if genericSecRe.MatchString(input) {
		matches := genericSecRe.FindAllStringIndex(input, -1)
		count += len(matches)
		input = genericSecRe.ReplaceAllString(input, "$1=[REDACTED]")
	}

	// 6. URI connection string credentials
	if uriUserInfoRe.MatchString(input) {
		matches := uriUserInfoRe.FindAllStringIndex(input, -1)
		count += len(matches)
		input = uriUserInfoRe.ReplaceAllString(input, "${1}:[REDACTED]${3}")
	}

	return input, count
}

// ExtractDiagnostics parses the clean, sanitized log to detect platform, failing step, error headline, and snippet.
func ExtractDiagnostics(cleanText string) *Diagnostics {
	d := &Diagnostics{
		Platform: detectPlatform(cleanText),
	}

	normalized := strings.ReplaceAll(cleanText, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	rawLines := strings.Split(normalized, "\n")
	var lines []string
	var failureLineIdxs []int
	var lastStep string

	for lineIdx, line := range rawLines {
		trimmed := strings.TrimSpace(line)
		lines = append(lines, line)

		// Track commands/steps
		if m := stepCommandRunRe.FindStringSubmatch(trimmed); len(m) > 0 {
			for i := 1; i < len(m); i++ {
				if m[i] != "" {
					lastStep = strings.TrimSpace(m[i])
					break
				}
			}
		}

		// Detect Go test failures
		if m := goTestFailRe.FindStringSubmatch(trimmed); len(m) > 1 {
			testName := m[1]
			d.FailedTests = append(d.FailedTests, testName)
			if d.PrimaryError == "" {
				d.PrimaryError = fmt.Sprintf("Go test %s failed", testName)
			}
			failureLineIdxs = append(failureLineIdxs, lineIdx)
		} else if m := goPkgFailRe.FindStringSubmatch(trimmed); len(m) > 1 {
			if d.PrimaryError == "" {
				d.PrimaryError = fmt.Sprintf("Go package %s failed", m[1])
			}
			failureLineIdxs = append(failureLineIdxs, lineIdx)
		}

		// Detect Pytest failures
		if m := pytestFailRe.FindStringSubmatch(trimmed); len(m) > 1 {
			testName := m[1]
			d.FailedTests = append(d.FailedTests, testName)
			if d.PrimaryError == "" {
				d.PrimaryError = fmt.Sprintf("Pytest %s failed", testName)
			}
			failureLineIdxs = append(failureLineIdxs, lineIdx)
		}

		// Detect Jest / Node failures
		if m := jestFailRe.FindStringSubmatch(trimmed); len(m) > 0 {
			testName := ""
			if m[1] != "" {
				testName = m[1]
			} else if len(m) > 3 && m[3] != "" {
				testName = fmt.Sprintf("%s › %s", m[2], m[3])
			}
			if testName != "" {
				d.FailedTests = append(d.FailedTests, testName)
				if d.PrimaryError == "" {
					d.PrimaryError = fmt.Sprintf("Jest test %s failed", testName)
				}
			}
			failureLineIdxs = append(failureLineIdxs, lineIdx)
		}

		// Detect Maven / Java failures
		if mvnFailRe.MatchString(trimmed) {
			if d.PrimaryError == "" {
				d.PrimaryError = "Maven build/test verification failure"
			}
			failureLineIdxs = append(failureLineIdxs, lineIdx)
		}

		// Detect SDD check errors
		if m := sddCheckErrorRe.FindStringSubmatch(trimmed); len(m) > 2 {
			if d.PrimaryError == "" {
				d.PrimaryError = fmt.Sprintf("sdd check failure at %s: %s", m[1], m[2])
			}
			failureLineIdxs = append(failureLineIdxs, lineIdx)
		}

		// Detect panic or fatal
		if strings.HasPrefix(trimmed, "panic:") || strings.Contains(trimmed, "FATAL:") {
			if d.PrimaryError == "" {
				d.PrimaryError = trimmed
			}
			failureLineIdxs = append(failureLineIdxs, lineIdx)
		}
	}

	d.FailedStep = lastStep
	if d.FailedStep == "" {
		d.FailedStep = "test / build step"
	}
	if d.PrimaryError == "" {
		if len(failureLineIdxs) > 0 {
			d.PrimaryError = strings.TrimSpace(lines[failureLineIdxs[0]])
		} else {
			d.PrimaryError = "CI execution failure (non-zero exit code)"
		}
	}

	// Capture failure snippet window (up to 40 lines around failures)
	d.FailureSnippet = buildFailureSnippet(lines, failureLineIdxs)
	return d
}

// Parse processes raw CI log data: strips ANSI, sanitizes credentials, and extracts diagnostics.
func Parse(data []byte) (*Diagnostics, string, error) {
	if len(data) == 0 {
		return nil, "", errors.New("cilog: input is empty")
	}
	if len(data) > MaxLogBytes {
		return nil, "", fmt.Errorf("cilog: input size %d bytes exceeds maximum allowed (%d bytes)", len(data), MaxLogBytes)
	}

	rawLen := len(data)
	clean := StripANSI(string(data))
	sanitized, redactions := SanitizeSecrets(clean)

	diag := ExtractDiagnostics(sanitized)
	diag.RawLength = rawLen
	diag.SanitizedLength = len(sanitized)
	diag.RedactionsCount = redactions

	return diag, sanitized, nil
}

// SeedBugfixReproduction injects the extracted CI diagnostics into the standard reproduction.md template.
func SeedBugfixReproduction(diag *Diagnostics, templateText string) string {
	if diag == nil {
		return templateText
	}

	var observed strings.Builder
	fmt.Fprintf(&observed, "CI job failure on platform %s:\n", diag.Platform)
	fmt.Fprintf(&observed, "- **Primary Error:** `%s`\n", diag.PrimaryError)
	fmt.Fprintf(&observed, "- **Failed Step:** `%s`\n", diag.FailedStep)
	if len(diag.FailedTests) > 0 {
		observed.WriteString("- **Failed Tests:**\n")
		for _, t := range diag.FailedTests {
			fmt.Fprintf(&observed, "  - `%s`\n", t)
		}
	}

	reproCommand := diag.FailedStep
	if reproCommand == "" || reproCommand == "test / build step" {
		reproCommand = "make verify"
	}

	testFunc := "none"
	if len(diag.FailedTests) > 0 {
		testFunc = diag.FailedTests[0]
	}

	snippet := diag.FailureSnippet
	if snippet == "" {
		snippet = diag.PrimaryError
	}

	// Replace sections in template
	res := templateText
	res = strings.Replace(res, "<What actually happens? Include error messages, unexpected return values, or crash logs>", observed.String(), 1)
	res = strings.Replace(res, "<e.g. macOS arm64, Linux amd64, Windows>", fmt.Sprintf("CI Runner (%s container)", diag.Platform), 1)
	res = strings.Replace(res, "<Step 3: Trigger condition>", fmt.Sprintf("Run `%s` in clean working directory", reproCommand), 1)
	res = strings.Replace(res, "<e.g. internal/pkg/feature_test.go:TestReproduceIssue>", testFunc, 1)
	res = strings.Replace(res, "<e.g. go test -v ./internal/pkg -run TestReproduceIssue>", reproCommand, 1)
	res = strings.Replace(res, "<paste real failure output here proving reproduction>", snippet, 1)

	return res
}

func detectPlatform(logText string) string {
	lower := strings.ToLower(logText)
	switch {
	case strings.Contains(lower, "github_action") || strings.Contains(lower, "actions/runner") || strings.Contains(lower, "github.com/actions"):
		return "github"
	case strings.Contains(lower, "gitlab-runner") || strings.Contains(lower, "ci_job_id"):
		return "gitlab"
	case strings.Contains(lower, "bitbucket pipelines") || strings.Contains(lower, "pipelines.bitbucket.org"):
		return "bitbucket"
	case strings.Contains(lower, "agent.jobname") || strings.Contains(lower, "azure pipelines"):
		return "azure"
	case strings.Contains(lower, "cloudbuild") || strings.Contains(lower, "google cloud build") || strings.Contains(lower, "gcr.io"):
		return "gcp"
	case strings.Contains(lower, "aws/codebuild") || strings.Contains(lower, "codebuild"):
		return "aws"
	default:
		return "generic"
	}
}

func buildFailureSnippet(lines []string, failureIdxs []int) string {
	if len(lines) == 0 {
		return ""
	}
	if len(failureIdxs) == 0 {
		// Take last 30 lines as fallback
		start := 0
		if len(lines) > 30 {
			start = len(lines) - 30
		}
		return strings.Join(lines[start:], "\n")
	}

	// Take window around first failure
	first := failureIdxs[0]
	start := first - 15
	if start < 0 {
		start = 0
	}
	end := first + 25
	if len(failureIdxs) > 1 {
		last := failureIdxs[len(failureIdxs)-1]
		if last+10 > end {
			end = last + 10
		}
	}
	if end > len(lines) {
		end = len(lines)
	}

	// Enforce 100 lines cap
	if end-start > 100 {
		end = start + 100
	}

	return strings.Join(lines[start:end], "\n")
}
