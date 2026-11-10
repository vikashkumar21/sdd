# Hotfix Incident Assessment — T-xxx <title>

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Active | Mitigated | Resolved
Roadmap ref: `ROADMAP.md` T-xxx

## 1. Incident Overview
- **Severity Level:** P0 (Outage / Data Loss) | P1 (Critical degradation)
- **Target Release Tag:** `<e.g. v1.3.0>` (Hotfix branches from this release tag)
- **Production Symptom:** <What broke in production? Error rate, crash stack trace, impacted users>
- **Customer / Business Impact:** <Quantify outage or degraded functionality>

## 2. Containment & Immediate Mitigation
- **Immediate Workaround:** <Was a feature flag disabled, proxy rule applied, or traffic rerouted?>
- **Rollback Procedure:** <How to revert if the hotfix fails?>

## 3. Scope Boundary
- Target branch: `hotfix/T-xxx-<slug>` (forked from target release tag)
- Rule: ZERO non-essential edits. Cleanups, renames, and optimizations are STRICTLY FORBIDDEN.
