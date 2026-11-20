---
status: draft
---
# Bug Reproduction & Diagnosis — T-xxx <title>

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Draft | Approved (YYYY-MM-DD)
Roadmap ref: `ROADMAP.md` T-xxx

## 1. Problem Summary
- **Observed Behavior:** <What actually happens? Include error messages, unexpected return values, or crash logs>
- **Expected Behavior:** <What should happen according to specifications?>
- **Severity / Impact:** <Blocker / High / Medium / Low — user impact>

## 2. Environment & Preconditions
- OS & Architecture: <e.g. macOS arm64, Linux amd64, Windows>
- Runtime / Language Version: <e.g. Go 1.27, Node 20, Python 3.12>
- Configuration / Flags / Preconditions: <specific setup needed to trigger bug>

## 3. Reproduction Steps
1. <Step 1>
2. <Step 2>
3. <Step 3: Trigger condition>

## 4. Mandatory Step 0: Minimal Failing Test Case
> **MANDATORY PREREQUISITE:** Before Phase B implementation or Gate 1 approval, you MUST author a minimal test case that reproduces the defect. The test MUST fail on current unpatched code.

- **Test File & Function:** `<e.g. internal/pkg/feature_test.go:TestReproduceIssue>`
- **Command to Execute:** `<e.g. go test -v ./internal/pkg -run TestReproduceIssue>`
- **Observed Failure Output (unpatched code):**
```text
<paste real failure output here proving reproduction>
```
- **Failing Test Verified:** [ ] Yes (GATE 1 cannot be approved without this verified failure)

## 5. Open Questions
<Must be EMPTY before GATE 1. Each unresolved item becomes a DECISION NEEDED block.>
