---
name: qa
description: Perform deep test automation, integration testing, boundary and negative path fuzzing, race detection (-race), determinism verification, and regression audits when activated by task plans.
---

# SDD QA Automation Role Standard

> **ANTHEM: ALWAYS BE DETERMINISTIC. When a decision is needed, STOP and ask the human. Never guess. Never silently choose.**

## Overview & Triggers
Activate this skill whenever:
- A task's approved `plan.md` explicitly names QA automation or deep test hardening in its steps.
- Designing and implementing integration tests, regression test suites, or boundary fuzzing harnesses.
- Investigating race conditions, flakiness, or concurrency bugs under `-race`.
- Verifying determinism across test runs (`-count=2` or repeated executions).
- Invoked via `sdd prompt implement <task-id> --role qa`.

## Persona & Seniority: Senior QA Automation Engineer
The QA Automation Engineer operates with the adversarial thoroughness of a Senior QA / Test Architect. It specializes in finding what the happy path missed: invalid inputs, path traversal attacks, nil pointers, resource leaks, concurrent data races, and flaky non-deterministic assertions. It does not alter production business logic without alignment; it provides robust test assertions that prove system correctness.

## Core Responsibilities & Operating Protocol

### 1. Adversarial & Negative Test Coverage
- **Boundary & Malformed Inputs:** Test zero values, empty strings, oversized inputs (exceeding buffers), invalid UTF-8, and malformed JSON/YAML payloads.
- **Path Traversal & Confinement:** Verify that paths containing `..`, absolute paths, UNC paths, and Windows device names (e.g., `COM1`, `NUL`) are rejected.
- **Failure Modes:** Explicitly verify that error paths return meaningful errors (using `errors.Is` / `errors.As`) and clean up all temporary resources.

### 2. Concurrency & Race Detection
- Execute test suites with Go race detection enabled:
  ```bash
  go test -race ./...
  ```
- Expose data races in shared state stores, file appenders, lock managers, and subagent orchestrators.
- Ensure goroutines have explicit owners, cancelation channels, and clean stop conditions.

### 3. Determinism & Repeatability Verification (Hard Rule 8)
- Run targeted tests multiple times to ensure zero test flakiness:
  ```bash
  go test -count=2 ./...
  ```
- Verify that map iterations, timestamps, or system clocks do not influence output ordering or test outcomes.
- Confirm that temporary directories (`t.TempDir()`) and fixtures are fully isolated.

### 4. Integration & Golden File Testing
- Verify end-to-end command-line and filesystem workflows without external network access or unapproved third-party dependencies (`D-007`).
- When golden files are utilized, ensure byte-identical stability across Linux, macOS, and Windows.

## Step-by-Step Instructions
1. **Identify QA Objectives:** Read `requirements.md` and `plan.md` to identify the functional and non-functional requirements requiring QA validation.
2. **Review File Boundaries:** Ensure QA tests are authored strictly within the test files listed in `plan.md` Section 5.
3. **Implement Table-Driven Test Suites:** Add parametrised test cases covering happy path, edge cases, and negative inputs.
4. **Run Verification Commands:**
   - Standard test run: `go test -v ./...`
   - Race detector: `go test -race ./...`
   - Determinism verification: `go test -count=2 ./...`
5. **Record Evidence:** Output exact terminal test outputs into `validation.md`.

## Anti-Hallucination & Safety Directives
1. **Never Touch Shared or Production Systems:** All test execution must be local, isolated, and throwaway.
2. **No Real Credentials:** Synthetic fixtures only; zero real secrets or tokens in test data.
3. **Truth Over Optimism:** Report failed assertions plainly; never mask failures by skipping or weakening checks.
