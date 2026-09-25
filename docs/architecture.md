# Architecture

This document establishes the initial boundaries for implementing the product defined in [product.md](./product.md).

## Architectural style

Use a hexagonal (ports-and-adapters) architecture.

The domain must understand concepts such as repositories, reviews, roots, plans, execution attempts, approvals, logs, and source changes, but it must not encode GitHub or Atlantis objects as those concepts.

GitHub and Atlantis are required initial integrations, not architectural boundaries.

Core application logic should depend on ports such as:

```text
SourceControl
  GetChange
  GetCommit
  GetDiff
  GetReviewers
  PublishDecision
  PublishStatus

Planner
  DiscoverRoots
  GetPlanAttempts
  GetPlanArtifact
  GetExecutionLogs

Executor
  Apply
  GetApplyAttempts
  GetExecutionLogs

PolicyEvaluator
  GetFindings

IdentityProvider
  ResolveActor
  Authorize
```

Initial adapters implement these ports using GitHub and Atlantis. Provider-specific identifiers and raw payloads may be retained as evidence and integration metadata, but they should not leak into core domain behavior or frontend contracts.

This boundary should make it possible to replace or add source-control, planning, execution, policy, and identity integrations without redesigning the review model.

## System boundaries

```text
GitHub                           Atlantis
  |                                |
  | PRs, commits, identity         | plans, applies, logs
  | reviews, checks                |
  +---------------+----------------+
                  |
                  v
        +-------------------+
        | Statecraft API  |
        | Go + Connect      |
        +---------+---------+
                  |
          normalized domain
                  |
                  v
        +-------------------+
        | Review frontend   |
        | TypeScript        |
        +-------------------+
```

The frontend does not consume raw OpenTofu/Terraform or Atlantis representations as its application model. The backend normalizes external inputs into review-domain objects.

## Initial domain model

### Repository

A GitHub repository installed/configured for Statecraft.

### Review

An infrastructure review corresponding to a GitHub pull request and current head commit.

Owns:

- repository and pull-request identity;
- head commit;
- expected roots;
- current plan set;
- review completeness;
- reviewers and decisions;
- findings;
- graph;
- execution history;\n- verification state;\n- revert lineage.

### Root

An independently plannable/applicable infrastructure unit.

A root is a product concept. Atlantis projects may map onto roots, but the frontend should not depend on Atlantis terminology.

### PlanSet

The complete collection of root plans constituting one reviewable infrastructure proposal.

Its identity must be deterministic and immutable, for example from:

- repository;
- pull request;
- commit SHA;
- ordered root identities;
- root plan digests.

Human approvals bind to a PlanSet.

### PlanAttempt

One attempt to produce a plan for a root.

Includes status, timestamps, artifact identity, logs, errors, and normalized changes.

### ApplyAttempt

One attempt to apply an approved root/plan set.

Includes status, timestamps, logs, errors, and any known partial-execution information.

### Resource

A normalized infrastructure object. Provider-specific data may be retained as evidence but should not define the frontend API.

### Change

A semantic difference for a Resource within a PlanSet.

At minimum:

- create;
- modify;
- replace;
- delete.

Changes contain property-level before/after evidence.

### Relationship

A directed edge between resources.

Relationships include:

- source;
- target;
- kind;
- evidence;
- confidence/provenance.

Examples include REFERENCES, DEPENDS_ON, ROUTES_TO, READS_FROM, WRITES_TO, AUTHORIZED_BY, and RUNS_ON.

### Finding

A review concern derived from policy or analysis.

Includes category, severity, blocking state, affected objects, explanation, status, and evidence.

### ReviewDecision

A human decision bound to a PlanSet.

Includes reviewer identity, decision, timestamp, optional message, and synchronization state with GitHub.

### Discussion

A thread attached to a Review, Resource, Change, Relationship, Finding, or execution event.

### ExecutionEvent

Structured information extracted from Atlantis execution and logs. Raw logs remain authoritative evidence.

## Frontend

Target: TypeScript.

The frontend owns interaction state and presentation:

- review dashboard;
- hierarchical change browser;
- directed graph browser;
- resource inspector;
- finding investigation;
- approval/request-changes interactions;
- execution/log exploration;
- plan history and comparison;\n- verification state;\n- revert preparation and lineage.

The graph must be designed for hundreds or thousands of resources. It should not assume that rendering the entire graph is useful.

Server-supported graph operations should include:

- changed-only graph;
- neighborhood by depth;
- upstream traversal;
- downstream traversal;
- path between resources;
- grouping/collapse by root and module;
- filtering by change/risk/finding;
- cross-root relationships.

## Backend

Target: Go with Connect.

The backend owns:

- GitHub integration and webhook ingestion;
- Atlantis plan/apply/log ingestion;
- root discovery and lifecycle;
- OpenTofu/Terraform normalization;
- plan-set identity and history;
- resource graph construction/query;
- findings and policy results;
- review/approval state;
- audit history;
- API authorization;\n- durable evidence/history suitable for future assisted diagnostics and remediation.

The backend should preserve raw artifacts separately from normalized projections so normalization can evolve without losing evidence.

## API shape

Initial services should follow domain boundaries rather than external systems.

```text
ReviewService
  ListReviews
  GetReview
  ApproveReview
  RequestChanges

ChangeService
  ListChanges
  GetChange
  ComparePlanSets

GraphService
  GetGraph
  GetNeighborhood
  GetUpstream
  GetDownstream
  FindPaths

FindingService
  ListFindings
  GetFinding

ExecutionService
  ListAttempts
  GetAttempt
  GetLogs

DiscussionService
  ListThreads
  AddComment
  ResolveThread
```

This is directional, not yet a protobuf contract.

## Integration principle

GitHub and Atlantis are adapters around the domain model, not the domain model itself. Integration code should live at the hexagonal boundary behind explicit ports rather than being called directly from domain services.

That keeps the product capable of evolving independently while preserving the current operating contract:

- GitHub owns code and repository identity.
- Atlantis executes plans and applies.
- Statecraft owns infrastructure-specific human review, diagnosis, and approval.
