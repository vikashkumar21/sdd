---
status: active
last_reviewed: 2026-11-22
---
# Domain Flow: Sample Business Flow

> **ALWAYS BE DETERMINISTIC. Human approval resolves every decision.**

- **Flow Identifier:** `sample_flow`
- **Typology:** Core Business Transaction
- **Primary Actors:** Client, Application Coordinator, Storage Service

## 1. Flow Preconditions & Invariants

1. Precondition: Initial state is verified and caller is authorized.
2. Invariant: All mutations are idempotent or transactional; failure triggers immediate rollback.

## 2. Sequence Diagram

```mermaid
sequenceDiagram
    autonumber
    actor Client
    participant Coordinator as Application Coordinator
    participant Engine as Domain Engine
    participant Store as State Store

    Client->>Coordinator: Request Sample Business Flow
    activate Coordinator
    Coordinator->>Engine: Validate Parameters & State
    activate Engine
    alt Validation Failed
        Engine-->>Coordinator: ValidationError (reason)
        Coordinator-->>Client: 400 Bad Request
    else Validation Succeeded
        Engine->>Store: Prepare Transaction
        activate Store
        Store-->>Engine: TransactionReady
        Engine->>Store: Commit State Changes
        Store-->>Engine: StateCommitted
        deactivate Store
        Engine-->>Coordinator: OperationSucceeded
        deactivate Engine
        Coordinator-->>Client: 200 OK (Success Payload)
    end
    deactivate Coordinator
```

## 3. Entity State Transitions

```mermaid
stateDiagram-v2
    [*] --> Pending
    Pending --> Validating: validate()
    Validating --> Processing: checkPassed
    Validating --> Failed: checkFailed
    Processing --> Committed: commit()
    Processing --> RolledBack: error()
    Committed --> [*]
    Failed --> [*]
    RolledBack --> [*]
```

## 4. Failure Modes & Compensation

| Failure Scenario | Detection Point | Automated Recovery / Compensation | User Action Required |
|---|---|---|---|
| Invalid Input | Coordinator validation step | Immediate fail-safe refusal; error message | Correct parameters and retry |
| Storage Collision | State store commit | Transaction rollback; lock release | Retry with backoff |
