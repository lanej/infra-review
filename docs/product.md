# Product Definition

## Purpose

Infra Review is the primary review, diagnosis, and human approval interface for infrastructure changes attached to a GitHub pull request.

GitHub remains the source of change identity and code history. Atlantis remains the planning and execution engine. Infra Review provides the infrastructure-specific model needed to understand what will happen, assess its risk, diagnose failures, collaborate on concerns, and approve or request changes against the exact plan that was reviewed.

The unit of review is a GitHub pull request. A review may contain multiple independently planned infrastructure roots.

## Primary job

> When a pull request proposes infrastructure changes across one or more roots, give me a complete, navigable representation of what will change and its consequences, help me diagnose planning or application problems, and let me make an informed approval decision against the exact plan I reviewed.

## Jobs to be done

### Understand the change

> When infrastructure changes are proposed, help me quickly understand what is changing across all affected roots and why, so I can determine where to spend review attention without reading raw plan output.

The normal review path should summarize consequences rather than reproduce Terraform/OpenTofu syntax. Raw plan output remains available as evidence and an escape hatch.

### Assess risk

> When reviewing a proposed change, identify the operations and relationships most likely to create operational, security, data-loss, availability, or cost risk, so I can investigate the consequential parts of the plan first.

Risk assessment must remain traceable to evidence. Destructive operations, replacements, privilege changes, network exposure, persistence changes, topology changes, and material capacity changes are first-class review concepts.

### Understand impact

> When a resource changes, show me what it depends on and what depends on it across root and module boundaries, so I can understand potential blast radius without reconstructing the architecture mentally.

The graph is a review-oriented browser, not a static architecture diagram. It must support large plans through filtering, hierarchy, neighborhood expansion, upstream/downstream traversal, path finding, and collapsing unchanged subgraphs.

### Investigate evidence

> When a change or finding looks surprising, let me progressively drill from the assessment into the affected resource, property, relationship, plan evidence, source configuration, and GitHub diff until I understand it.

Every derived conclusion should preserve provenance.

### Assess review completeness

> Before I approve a change, tell me whether every expected infrastructure root has successfully produced the required plan and analysis, so I cannot mistake partial analysis for a complete review.

Roots are first-class review objects. Approval must not silently ignore a root that is pending, failed, missing, or stale.

### Collaborate on concerns

> When I find something that needs explanation or correction, let me discuss the specific resource, property change, relationship, finding, or review without losing the infrastructure context.

Discussion should attach to review-domain objects rather than requiring reviewers to refer to opaque plan lines.

### Review other decisions

> When I am reviewing a change, show me who has already approved, requested changes, or raised unresolved concerns, and which exact plan version each decision applies to, so I understand the current human review state.

A newer plan may invalidate an earlier approval. Staleness must be explicit.

### Approve or request changes

> When I have completed my assessment, let me approve or request changes from Infra Review, with my decision bound to the exact commit and plan version I reviewed.

Infra Review is intended to become the primary human approval interface. A future GitHub App may act on behalf of authenticated users while preserving reviewer identity and auditability.

### Diagnose planning failures

> When infrastructure planning fails, show me which root failed, the most relevant error and surrounding execution context, and the underlying logs, so I can distinguish configuration problems from execution or environment failures and fix them quickly.

Atlantis logs are evidence. Structured error extraction and summaries may make them easier to navigate, but must link back to original log lines.

### Diagnose application failures

> When an approved infrastructure change fails during application, show me what succeeded, what failed, where execution stopped, and the relevant logs and resource context, so I can understand the resulting state and determine the next action.

Review does not end at approval. Plan and apply attempts belong to the lifecycle of the same infrastructure change.

### Understand change history

> When a pull request has been replanned or retried, let me inspect prior plans and execution attempts and compare their infrastructure consequences, so I can understand how the proposed change evolved and why prior approvals or findings became stale.

Plans and applies are versioned attempts, not mutable fields on a review.

## Product invariants

1. A reviewer approves an exact plan, not merely a pull request.
2. Plan identity includes the Git commit and the complete set of relevant root plans.
3. A new or changed plan makes affected approvals visibly stale.
4. Partial planning cannot appear equivalent to a complete review.
5. Derived findings and summaries retain links to their evidence.
6. Raw plan and execution logs remain available even when higher-level explanations exist.
7. Human approval belongs in Infra Review.
8. Automatic approval does not. External policy and GitHub automation may determine that human approval is unnecessary.
9. GitHub remains authoritative for source changes and repository identity.
10. Atlantis remains authoritative for Terraform/OpenTofu planning and execution.
11. Infra Review is not a general-purpose infrastructure administration console.

## Core workflow

```text
GitHub PR opened or updated
        |
        v
Affected roots discovered
        |
        v
Atlantis plan attempts
        |
        +---- failure ----> diagnose logs/errors ----> source change ----+
        |                                                               |
        +--------------------------- replan <----------------------------+
        |
        v
All required roots planned
        |
        v
Unified change assessment
        |
        +--> summary / findings
        +--> hierarchical changes
        +--> directed resource graph
        +--> evidence
        +--> reviewer discussion
        |
        v
Human review
        |
        +--> request changes --> new commit/plan --> stale prior review
        |
        v
Approval
        |
        v
Atlantis apply
        |
        +---- failure ----> diagnose partial execution/logs
        |
        v
Applied
```

## Review surfaces

The primary review experience should expose six projections of the same underlying review model:

- **Overview** — completeness, reviewers, approvals, highest-risk changes, findings, and execution state.
- **Changes** — root/module/resource hierarchy with semantic before/after changes.
- **Graph** — robust directed graph browser for relationships, impact, and blast-radius investigation.
- **Findings** — policy, security, reliability, destructive-operation, and cost concerns.
- **Execution** — Atlantis plan/apply attempts, structured failures, and searchable raw logs.
- **History** — plan versions, execution attempts, approvals, invalidations, and infrastructure-level comparisons.

Selecting a resource, finding, graph node, change, or execution error should converge on the same underlying objects and evidence rather than creating isolated experiences.

## Explicit non-goals

Infra Review does not initially:

- edit infrastructure configuration;
- replace GitHub as the source repository or code-review history;
- replace Atlantis as the plan/apply executor;
- autonomously decide to approve a change;
- become a cloud resource administration portal.

## Success criterion

A reviewer should not need to read raw Terraform/OpenTofu plan output or unstructured Atlantis logs to make or debug a routine infrastructure change, while always being able to reach that raw evidence when necessary.
