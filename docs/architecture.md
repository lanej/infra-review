# Architecture

This document establishes the implementation boundaries for the product defined in
[product.md](./product.md). The [handoff](handoff.md) identifies the implemented
subset; the [roadmap](roadmap.md) orders the remaining work. Domain vocabulary and
system diagrams below describe the target unless explicitly marked current.

## Architectural style

Statecraft uses a hexagonal (ports-and-adapters) architecture.

The domain understands repositories, reviews, roots, plan sets, execution attempts,
approvals, findings, policies, violations, acceptances, resources, relationships,
evidence, and source changes. GitHub, Atlantis, and OPA types stay in adapters.

GitHub and Atlantis are required initial integrations. OPA/Rego is the intended
initial policy integration. These implementations do not define the domain model.

### Current outbound ports

```text
SourceControl
  GetChange
  ListChangedFiles
  ListReviewDecisions
  PublishDecision
  PublishStatus

Planner
  Plan

Executor
  Apply

ReviewStore
  GetReview

WorkflowStore (isolated demo sessions)
  Create
  Update (expected version, atomic mutation)

DemoPlanner
  ChangesForPlan (synthetic evidence only)

PolicyEvaluator
  EvaluatePlan

ActionPolicy
  EvaluateAction
```

Ports will expand only when a domain use case requires them. In particular,
historical plan/log retrieval should be backed by Statecraft's evidence store rather
than pretending Atlantis exposes a stable historical read API.

Provider-specific identifiers and raw payloads may be retained as evidence and
integration metadata, but they must not leak into frontend contracts or core domain
behavior.

See [integrations.md](./integrations.md) for the concrete GitHub and Atlantis mapping.
See [policies.md](./policies.md) for the policy domain, assessment and
action-policy ports, OPA boundary, and violation-acceptance workflow. The mock runtime
now exercises these ports through `DemoWorkflow`; it has no access to the live
`Planner`, `Executor`, or `SourceControl` ports. `DemoPlanner` is intentionally a
separate simulation capability, not an alternative production execution path.

## Target system boundaries

```text
GitHub                                  Atlantis
  |                                        |
  | PRs, commits, reviews, checks          | native HTTP API
  |                                        | plan/apply; later locks/drift
  |                                        | workflow evidence/webhooks
  +------------------+---------------------+
                     |
                     v
           +-------------------+
           | Statecraft API    |
           | Go + Connect      |
           +---------+---------+
                     |
              normalized domain
                     |
              durable evidence
                     |
                     v
           +-------------------+
           | TypeScript UI     |
           +-------------------+
```

The frontend never consumes GitHub, Atlantis, OPA decision documents, or raw
OpenTofu/Terraform representations as its application model. OPA evaluates policies
through outbound domain ports; Statecraft services enforce the resulting decisions.

Atlantis is an execution-system adapter, not a GitHub-comment parser. Statecraft uses
Atlantis's native HTTP API for command integration and keeps that alpha wire contract
inside the adapter. GitHub remains authoritative for source-change and review context;
Atlantis remains authoritative for command/execution context. Future lock or drift
capabilities should extend the same adapter behind domain ports rather than introducing
a comment-scraping control path.

## Domain model

The runnable mock uses `Review`, `PlanSnapshot`, basic policy/acceptance records,
and simulated attempts. Canonical PlanSets, full immutable historical assessments,
durable evidence and several entities below still need implementation. In
particular, a synthetic `PlanSnapshot.Digest` is not the artifact identity described
by the target PlanSet contract.

### Repository

A source repository installed/configured for Statecraft.

### Review

An infrastructure review corresponding to a source change, initially a GitHub pull
request.

A Review owns or references:

- repository and source-change identity;
- head commit;
- expected roots;
- current plan set;
- review completeness;
- reviewers and decisions;
- findings;
- resource graph;
- execution history;
- verification state;
- revert lineage.

### Root

An independently plannable/applicable infrastructure unit.

Atlantis projects may map onto roots, but a root is a Statecraft concept. A named
Atlantis project or directory/workspace tuple is external identity used by the
Atlantis adapter.

### PlanSet

The complete collection of root plans constituting one reviewable infrastructure
proposal.

Its identity must be deterministic and immutable from at least:

- repository;
- source-change identity;
- commit SHA;
- ordered root identities;
- exact root plan digests.

Human approvals bind to a PlanSet.

### PlanAttempt

One attempt to produce a plan for a root.

Includes status, timestamps, artifact identity, logs/errors, and normalized changes.

### ApplyAttempt

One attempt to apply an approved root/plan set.

Includes status, timestamps, logs/errors, and known partial-execution information.

### Resource and Change

A Resource is a normalized infrastructure object. A Change is a semantic difference
for that resource within a PlanSet: create, modify, replace, or delete, with
property-level before/after evidence.

### Relationship

A directed edge between resources with a relationship kind, evidence, and
confidence/provenance.

Examples include REFERENCES, DEPENDS_ON, ROUTES_TO, READS_FROM, WRITES_TO,
AUTHORIZED_BY, and RUNS_ON.

### Finding

A review concern derived from policy or analysis, including severity, blocking
state, affected objects, explanation, status, and evidence.

### Policy, PolicyEvaluation, and PolicyViolation

A Policy is a versioned requirement. A PolicyEvaluation records assessment of an
exact PlanSet and normalized input against immutable policy and reference-data
versions, with coverage, outcomes, errors, and PolicyViolations. A policy-derived
Finding references its violation and evidence. Severity, evaluation completeness,
compliance, and action eligibility are separate concepts.

### ViolationAcceptance and ActionDecision

A ViolationAcceptance is an authorized, scoped, time-bounded decision to tolerate
specified violations when policy permits. It preserves the violations and does not
approve the PlanSet. An ActionDecision records whether a particular actor may
perform a particular action, with requirements, reasons, and evidence. Statecraft
enforces that decision against current context before recording or dispatching an
action. Undefined or incomplete required evaluation cannot authorize approval or
apply. See [policies.md](./policies.md) for identity and invalidation rules.

### ReviewDecision

A human decision bound to an exact PlanSet and commit, with reviewer identity,
timestamp, optional message, and external synchronization metadata.

Source-control reviews are separate `ExternalReviewDecision` history in
`Review.SourceDecisions`. Refreshing that history preserves Statecraft's own
decisions. A source review's commit or editable body cannot establish an
authoritative PlanSet approval; an authenticated persisted decision is required.
The temporary JSON bridge exposes only Statecraft decisions, matching the current
protobuf/frontend approval contract.

A change to an existing review's head marks the review and roots stale while
retaining earlier evidence and decisions as history. Metadata refresh does not
restore readiness; new planning evidence must establish that separately.

### Discussion

A thread attached to a Review, Resource, Change, Relationship, Finding, or execution
event.

### Evidence

Raw external artifacts are durable evidence, separate from normalized projections.
Normalization can evolve without destroying the source material used to derive a
decision.

## Frontend

Target: TypeScript.

The frontend owns interaction and presentation for:

- review overview;
- hierarchical change browser;
- directed graph browser;
- resource inspector;
- finding investigation;
- approval/request-changes interactions;
- execution/log exploration;
- plan history and comparison;
- verification state;
- revert preparation and lineage.

The graph must support hundreds or thousands of resources without assuming that
rendering the complete graph is useful.

Server-supported graph operations should eventually include changed-only views,
bounded neighborhoods, upstream/downstream traversal, path finding, grouping by
root/module, filtering by change/risk/finding, and cross-root relationships.

## Backend

Target: Go with Connect.

The backend owns:

- source-control integration and webhook ingestion;
- planning/execution adapters;
- plan/apply evidence ingestion;
- root discovery and lifecycle;
- OpenTofu/Terraform normalization;
- plan-set identity and history;
- resource graph construction/query;
- policy assessment, violations, acceptance, and action eligibility;
- review/approval state;
- durable execution evidence;
- audit history;
- authorization;
- durable history suitable for future assisted diagnostics/remediation.

## API shape

Initial public services should follow Statecraft domain boundaries rather than
external systems:

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

This is directional, not yet the final protobuf contract.

## Integration principle

GitHub, Atlantis, and OPA integrate through adapters around the domain model.

- GitHub owns source changes and repository identity.
- Atlantis executes plan/apply workflows.
- OPA evaluates infrastructure and workflow policies; Rego and engine types remain
  inside the integration boundary.
- Statecraft owns the durable infrastructure-specific review, diagnosis, evidence,
  violation acceptance, enforcement, and human approval model.
