# Statecraft — Product Definition

## Purpose

Statecraft is the primary workbench for infrastructure changes attached to a GitHub pull request.

It is where infrastructure changes are **planned, diagnosed, understood, assessed, discussed, approved, applied, verified, and—when necessary—reverted**.

GitHub remains the source of change identity and code history. Atlantis remains the planning and execution engine. Statecraft provides the infrastructure-specific model and interface needed to work a change through its lifecycle.

The unit of work is a GitHub pull request. A review may contain multiple independently planned infrastructure roots.

This document describes the intended product, not a list of shipped capabilities.
The [roadmap](roadmap.md) owns feature status and delivery order; the
[handoff](handoff.md) describes the current mock implementation.

## People and decisions

| Person / responsibility | Decision the workbench should support |
| --- | --- |
| Change author | Is the complete proposal available, what failed, and what source correction or evidence is needed next? |
| Infrastructure reviewer | What will change, who or what may be affected, what remains unknown, and is this exact proposal acceptable? |
| Policy or resource owner | Does this rule permit accepting this violation, and are the rationale, recovery evidence, scope and duration sufficient? |
| Operator / incident responder | What actually executed, what state now exists, what remains uncertain, and what is the next justified recovery action? |

These are responsibilities, not hard-coded production roles. Authorization and any
required separation of duties come from trusted organization policy. The mock
personas demonstrate a workflow without defining that production policy.

The interface should answer five questions before a consequential decision:
**What am I deciding? What evidence covers it? What could happen? What is still
unknown or blocking? What happens after I act?** Display confidence and provenance
with the conclusion; an attractive summary is not evidence by itself.

## Change lifecycle

```text
Plan -> Diagnose -> Understand -> Assess -> Discuss -> Approve -> Apply -> Verify
  ^         |            |          |          |                    |       |
  |         +---- fix ---+----------+----------+--------------------+       |
  |                                                                      |
  +--------------------------- Revert <-----------------------------------+
                                  |
                                  +----> Plan -> Assess -> Approve -> Apply -> Verify
```

The lifecycle is intentionally iterative. Planning and application failures return the user to diagnosis. Requested changes create new plans. A revert is itself an infrastructure change and goes through the same plan, assessment, approval, apply, and verification controls rather than acting as a privileged undo operation.

## Primary job

> When a pull request proposes infrastructure changes across one or more roots, give me a complete, navigable workbench to understand the change and its consequences, diagnose planning or application problems, assess risk, collaborate with other reviewers, approve the exact plan I reviewed, verify its application, and safely prepare a revert when necessary.

## Jobs to be done

### Plan the change

> When a pull request changes infrastructure, show me the planning state of every affected root and assemble successful root plans into one coherent reviewable change, so I know whether the proposal is complete and ready for assessment.

### Diagnose planning failures

> When infrastructure planning fails, show me which root failed, the most relevant error and surrounding execution context, and the underlying logs, so I can distinguish configuration problems from execution or environment failures and fix them quickly.

Atlantis logs are evidence. Structured error extraction and summaries may make them easier to navigate, but must link back to original log lines.

### Understand the change

> When infrastructure changes are proposed, help me quickly understand what is changing across all affected roots and why, so I can determine where to spend review attention without reading raw plan output.

The normal review path should summarize consequences rather than reproduce Terraform/OpenTofu syntax. Raw plan output remains available as evidence and an escape hatch.

### Assess risk

> When reviewing a proposed change, identify the operations and relationships most likely to create operational, security, data-loss, availability, or cost risk, so I can investigate the consequential parts of the plan first.

Risk assessment must remain traceable to evidence. Destructive operations, replacements, privilege changes, network exposure, persistence changes, topology changes, and material capacity changes are first-class concepts.

### Assess policies

Policy is a first-class part of assessment. The workbench presents versioned
policies, their violations, evaluation coverage, and the requirements governing
review and execution in Statecraft terms. OPA/Rego is an implementation behind an
adapter. See [policies.md](./policies.md) for the policy and acceptance design.

### Resolve or accept violations

> When a proposed change violates policy, explain the requirement, its consequences,
> and the allowed resolution paths, so I can correct the change or request an
> authorized, scoped acceptance with the necessary rationale and evidence.

Accepting a violation preserves it as a recorded concern. Acceptance has scope,
conditions, expiry, and an audit trail; it neither establishes policy compliance
nor replaces approval of the exact PlanSet. Some policies prohibit acceptance.
Missing or failed evaluation must remain visibly different from a satisfied policy.

### Understand impact

> When a resource changes, show me what it depends on and what depends on it across root and module boundaries, so I can understand potential blast radius without reconstructing the architecture mentally.

The graph is a workbench for investigation, not a static architecture diagram. It must support large plans through filtering, hierarchy, neighborhood expansion, upstream/downstream traversal, path finding, and collapsing unchanged subgraphs.

### Investigate evidence

> When a change or finding looks surprising, let me progressively drill from the assessment into the affected resource, property, relationship, plan evidence, source configuration, and GitHub diff until I understand it.

Every derived conclusion should preserve provenance.

### Assess completeness

> Before I approve a change, tell me whether every expected infrastructure root has successfully produced the required plan and analysis, so I cannot mistake partial analysis for a complete review.

Roots are first-class workbench objects. Approval must not silently ignore a root that is pending, failed, missing, or stale.

### Discuss concerns

> When I find something that needs explanation or correction, let me discuss the specific resource, property change, relationship, finding, or review without losing the infrastructure context.

Discussion should attach to domain objects rather than requiring reviewers to refer to opaque plan lines.

### Review other decisions

> When I am reviewing a change, show me who has already approved, requested changes, or raised unresolved concerns, and which exact plan version each decision applies to, so I understand the current human review state.

A newer plan may invalidate an earlier approval. Staleness must be explicit.

### Approve or request changes

> When I have completed my assessment, let me approve or request changes from Statecraft, with my decision bound to the exact commit and plan version I reviewed.

Statecraft is intended to become the primary human approval interface. A future GitHub App may act on behalf of authenticated users while preserving reviewer identity and auditability.

### Apply the change

> When a change has satisfied its approval requirements, show me the application state across its roots and make the transition from approved plan to Atlantis execution explicit, so I can follow what is happening rather than losing context at approval time.

### Diagnose application failures

> When an approved infrastructure change fails during application, show me what succeeded, what failed, where execution stopped, and the relevant logs and resource context, so I can understand the resulting state and determine the next action.

The workbench does not end at approval. Plan and apply attempts belong to the lifecycle of the same infrastructure change.

### Verify the result

> After application, show me whether the intended infrastructure change was actually realized and surface any known divergence or incomplete execution, so a successful command is not mistaken for a verified outcome.

Initially verification may be limited to Atlantis/OpenTofu execution and resulting state. The model should allow richer health, drift, or operational signals later without making them prerequisites for the first implementation.

### Understand change history

> When a pull request has been replanned or retried, let me inspect prior plans and execution attempts and compare their infrastructure consequences, so I can understand how the proposed change evolved and why prior approvals or findings became stale.

Plans and applies are versioned attempts, not mutable fields on a review.

### Revert safely

> When an applied infrastructure change needs to be undone, help me understand the desired prior state, prepare the reverse infrastructure change, assess its consequences, and send it through the normal approval and execution lifecycle, so reverting does not bypass the controls used for forward changes.

A revert is not a magical rollback. It creates a new proposed change with its own plan, findings, approvals, application, and verification.

## Product invariants

1. A reviewer approves an exact plan, not merely a pull request.
2. Plan identity includes the Git commit and the complete set of relevant root plans.
3. A new or changed plan makes affected approvals visibly stale.
4. Partial planning cannot appear equivalent to a complete review.
5. Derived findings and summaries retain links to their evidence.
6. Raw plan and execution logs remain available even when higher-level explanations exist.
7. Human approval belongs in Statecraft.
8. Automatic approval does not. External policy and GitHub automation may determine that human approval is unnecessary.
9. GitHub remains authoritative for source changes and repository identity.
10. Atlantis remains authoritative for Terraform/OpenTofu planning and execution.
11. A successful apply is distinct from a verified outcome.
12. Reverts use the same planning, assessment, approval, execution, and verification controls as forward changes.
13. Statecraft is not a general-purpose infrastructure administration console.
14. Policy-engine types and rule languages stay behind adapters; policies,
    evaluations, violations, acceptance, and action eligibility are domain concepts.
15. Accepting a violation preserves the violation and is distinct from approving a
    PlanSet. Expired, revoked, or out-of-scope acceptance cannot satisfy a gate.
16. Required policy evaluation must establish complete, current coverage. Missing
    or undefined results cannot silently count as compliance or permission.

## Workbench surfaces

The primary experience should expose projections of the same underlying change model:

- **Overview** — lifecycle state, completeness, reviewers, approvals, highest-risk changes, findings, and execution state.
- **Changes** — root/module/resource hierarchy with semantic before/after changes.
- **Graph** — robust directed graph browser for relationships, impact, and blast-radius investigation.
- **Findings** — policy, security, reliability, destructive-operation, and cost concerns.
- **Execution** — Atlantis plan/apply attempts, structured failures, searchable raw logs, and verification state.
- **History** — plan versions, execution attempts, approvals, invalidations, infrastructure-level comparisons, and reverts.

Selecting a resource, finding, graph node, change, or execution error should converge on the same underlying objects and evidence rather than creating isolated experiences.

## Future direction: assisted diagnosis and remediation

Statecraft should eventually be able to help move failed or risky changes toward a better state, not merely explain them.

A future assisted-diagnostics/remediation capability could combine:

- structured plan and apply failures;
- Atlantis execution logs;
- normalized resource and relationship context;
- policy findings;
- repository source and GitHub history;
- prior failures and successful fixes;
- organization-specific instructions and skill files;
- LLM reasoning over that evidence.

The initial goal would be **assistance rather than autonomous mutation**: diagnose likely causes, surface relevant precedent, suggest concrete fixes, and prepare a proposed source change for human review. Repeated failure/fix history should become useful product data rather than disappearing into old PRs and logs.

This capability is explicitly out of scope for the initial foundation. The current architecture should preserve the evidence and history required to support it later without introducing an LLM or remediation subsystem now.

## Explicit non-goals

Statecraft does not initially:

- provide a general-purpose editor for infrastructure configuration;
- replace GitHub as the source repository or code history;
- replace Atlantis as the plan/apply executor;
- autonomously decide to approve a change;
- automatically diagnose, edit, or remediate infrastructure changes in the initial implementation;
- provide a privileged rollback mechanism that bypasses review;
- become a cloud resource administration portal.

## Success criterion

A user should be able to work a routine infrastructure change from planning through verification—or diagnose and safely revert it—without needing to reconstruct the change from raw Terraform/OpenTofu output, Atlantis comments, and unstructured logs, while always being able to reach that raw evidence when necessary.
