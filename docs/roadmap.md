# Statecraft roadmap

The goal is a workbench in which a reviewer can understand and work an infrastructure
change from planning to verified outcome, with each decision grounded in current
evidence. See [product intent](product.md) and [design](../DESIGN.md). This is an
ordered delivery plan with completion criteria, not a date commitment or a claim
that the current mock runtime is production ready.

## Where we are

The [mock workbench](steel-thread.md) is implemented: multi-root inspection, policy
violation acceptance, plan approval, simulated execution/verification, and recovery
scenarios. GitHub and Atlantis adapters have local contract tests and remain outside
the executable. Policy ports exist; an OPA adapter does not. The
[handoff](handoff.md) maps the implementation and its limitations.

## Required capabilities

| Capability | Current state | Required destination / delivery |
| --- | --- | --- |
| Review and root discovery | One synthetic source change; four seeded roots | Repository onboarding and source-change list; explicit complete expected root scope from configured source/execution context. R2/R3 |
| Plan evidence and semantic changes | Display strings and synthetic artifacts | Durable exact artifacts, typed before/after values, unknown/sensitive values, stable resource identity, versioned normalization and provenance. R2 |
| Decision workspace | Five views, resource inspector, filtering, connected-resource links | Consistent evidence navigation, actionable missing context, large-plan investigation, accessible loading/error/recovery behavior. R2/R6 |
| API contract | Protobuf plus handwritten JSON/TS | Generated Connect client/handler boundary with reproducible generation and compatibility checks. R1 |
| Assessment | Two deterministic mock rules | Versioned OPA evaluation with complete expected coverage and explicit indeterminate results. R4 |
| Violation acceptance | Request/grant, fixed personas, one-hour expiry | Authorized request/grant/deny/revoke, policy-specific conditions, scope, evidence, expiry and audit history; hard prohibitions remain hard. R3/R4 |
| Human review | Demo approve/request changes bound to plan/head/assessment | Authenticated reviewers, required reviewers/roles, separation of duties where configured, stale/conflicting decision handling, and source synchronization without invented approvals. R3/R4 |
| Execution | Synchronous simulation plus uncomposed Atlantis command client | Durable attempts/jobs, exact approved artifact enforcement, current gates, lock/concurrency controls, reconciliation and explicit partial/uncertain outcomes. R5 |
| Verification | Simulated state agreement with health explicitly unknown | Actual post-apply state comparison and provenance; drift/incomplete verification distinct from success. Optional operational signals remain separately labeled. R5 |
| History and audit | In-memory events and plan/decision identities | Immutable evidence and assessment history, reasons for invalidation, previous/current plan comparison, durable actor/time/source records and retention. R2–R5 |
| Collaboration | Request-changes comment only | Threads on resources, properties, relationships and findings; resolution must not silently accept risk or authorize apply. R6 |
| Impact graph | Illustrative links and a legacy static graph | Bounded neighborhoods, upstream/downstream traversal, path finding, root/module grouping and unknown-edge provenance at realistic scale. R6 |
| Revert | Product/design intent only | New source proposal with lineage and explicit unrecoverable state/data consequences; normal review and execution controls. R7 |
| Operating the service | Local loopback process, memory store | Tenant/repository access isolation, durable storage/backup, credential and artifact protection, observability, retention, and documented deployment/recovery. R2/R3/R5 |
| Assisted diagnosis/remediation | Future intent only | Evidence-grounded diagnosis and proposed source changes for human review after core evidence and controls exist. R8 |

## Delivery sequence

### R1 — Replace the temporary transport (next)

Keep the mock workflow and its interaction semantics. Generate Go/Connect and
TypeScript bindings from protobuf; map domain values at the transport boundary and
use the generated client in `web/`. Pin generator versions and document a repeatable
regeneration command. Define error mapping for invalid input, denied action,
missing review, stale version, and unavailable service.

Complete when the running browser uses generated RPCs for create/read/action,
provider types remain in adapters, equivalent server-side gates still hold,
contract regeneration is reproducible, and the temporary routes/handwritten TS
contract are removed or have an explicit, bounded migration consumer. Preserve the
original fixture path's intent until its caller is migrated. Verify the workflow
and failure behavior through the new transport. Do not wire production adapters as
part of this slice.

### R2 — Durable proposal and evidence model

Design the ingestion contract and storage port before choosing persistence details.
Capture repository, source change, commit, expected roots, root attempt identity,
exact plan artifacts/digests, structured plan JSON, logs, timestamps, and provenance.
Normalize typed changes and relationships; distinguish unknown, redacted, absent,
and unchanged values. Define canonical PlanSet identity and immutable evidence
references rather than expanding the synthetic digest convention.

Complete when a multi-root sample survives restart, can be replayed from stored
evidence, distinguishes missing/failed/stale roots, and retains full prior versions
for comparison. Duplicate/out-of-order ingestion must not resurrect obsolete
readiness. Address sensitive artifact redaction/access and retention in the design;
raw plans are not automatically safe UI output. The interface should show evidence
freshness and provenance from those records.

### R3 — Authenticated source reviews and authorization

Configure GitHub App installation and user identity behind existing boundaries.
Add repository access and root discovery, authenticated/replay-resistant webhook
intake, source refresh, and reviewer capabilities from trusted backend context.
Persist actor/role evidence and human decisions. Keep external GitHub review history
distinct from authoritative Statecraft plan approval; define reliable publication
of checks and human review results with explicit synchronization failures.

Complete when two real test identities can observe different permitted actions,
untrusted role claims cannot grant acceptance or approval, new source revisions
invalidate old eligibility, and reconnect/replay cannot duplicate human actions.
Document token/secret handling, tenancy, permission requirements and review
attribution. Build on R1/R2; do not introduce live infrastructure mutation yet.

### R4 — OPA assessment and workflow policies

Implement the intended OPA adapter behind `PolicyEvaluator` and `ActionPolicy`.
Choose embedded SDK/evaluation or service deployment through a short documented
decision. Define versioned normalized inputs/output schema, expected policy coverage,
policy/config/reference-data/input digests, time validity and reproducibility.
Policy selection must come from trusted configuration, not the proposed change.

Complete when a real Rego policy can produce satisfied, violated, not-applicable,
and indeterminate outcomes without engine documents entering public contracts.
Exercise a hard prohibition, advisory concern, authorized acceptance, denied
acceptance, revocation, expiry, changed policy/reference facts after approval,
and unavailable/malformed/undefined evaluation. All required incomplete results
must block their dependent decisions. Store immutable assessments and acceptance
audit history; revalidate at dispatch using R2/R3 identity and evidence. Adapter
prototyping against fixtures may proceed earlier without claiming production gates.

### R5 — Enforced execution and verified outcomes

First resolve the [Atlantis exact-artifact limitation](integrations.md#apply-is-not-approval-enforcement).
Document the supported artifact handoff/reconciliation strategy and how other
Atlantis command paths obey the same controls. If this guarantee cannot be met,
keep execution unavailable and show the reason; another confirmation screen does
not solve it.

Then add durable operation intent/receipts, per-root attempts, lock/conflict handling,
identity checks, bounded log access, authenticated notifications, and reconciliation
for timeouts or missing events. Automatic retry must not repeat an uncertain apply.
Partial execution produces a new proposal from the observed state, not a replay of
the old request. Verify actual resulting state separately from execution and health.

Complete when a controlled non-production integration demonstrates that the
approved artifact is the one applied, changed context stops dispatch, duplicate
requests do not repeat execution, partial/unknown outcomes survive a restart, and
verification reports divergence or missing evidence honestly. Requires R2–R4.
Include service observability and operational recovery documentation before a pilot
that can mutate infrastructure.

### R6 — Investigation and collaboration at scale

Grow the existing workbench from normalized evidence rather than inventing a second
model for the graph. Add root/module hierarchy, filters, bounded neighborhoods,
paths and dependency provenance, semantic plan comparison, contextual discussion,
and error-to-log/source navigation. Add review discovery across configured
repositories when the single-review path is dependable.

Complete when a representative large plan can be investigated with keyboard and
pointer without rendering its entire graph, a selection retains evidence context
across views, and comments/resolutions remain separate from policy acceptance and
approval. Define representative size and responsiveness targets with measured data.
This stream can run alongside R3–R5 once R2 supplies reliable evidence; useful
inspection and accessibility work need not wait for live apply.

### R7 — Reviewed revert and richer verification

Prepare a reverse source change with explicit lineage to the applied proposal.
Explain which infrastructure/data effects cannot be automatically restored. Re-plan,
reassess, review, execute, and verify using the normal workflow. Add operational
health/drift signals as separately sourced evidence when required by users.

Complete when a sample revert never bypasses policy/approval, preserves its forward
change lineage, and exposes unresolved recovery consequences. Depends on R2/R5 and
the source-change workflow; this is not an administrative rollback button.

### R8 — Assisted diagnosis (later)

Use retained failures, logs, resource evidence, source history and prior fixes to
explain likely causes and prepare proposed source changes for human review. Do not
add an LLM dependency while implementing the core review/execution foundation.
Any assistance must cite evidence and keep uncertain conclusions visible; it does
not approve changes or gain a separate apply path.

## Design and validation work across milestones

Use [DESIGN.md](../DESIGN.md) for decision prompts, states, and evidence continuity.
Maintain screenshots of real UI changes. Add automated browser coverage for the
critical lifecycle and blocked/stale/concurrent decision paths as interactions grow;
rendering unit tests are not a substitute for those interactions. Migrate the old
`.ui-review` target/selectors before relying on it for the current app. Check
keyboard/screen-reader semantics, focus, contrast, zoom, and narrow layouts against
explicit accessibility criteria rather than treating a screenshot as proof.

## Decisions still to make

| Decision | Resolve before | Constraint |
| --- | --- | --- |
| Evidence/database/object storage and retention | R2 persistence | Preserve exact artifacts, immutable history, access controls, and replay; do not select a store solely to persist the demo JSON shape. |
| Root discovery and ingestion delivery | R2/R3 integration | Establish complete scope and authenticated artifact provenance; the Atlantis API does not supply a historical plan-JSON read endpoint. |
| User identity, role source, required reviewers and tenancy | R3/R4 | Trusted backend context, attributable decisions, clear separation from demo personas and GitHub bot reviews. |
| OPA deployment and policy distribution | R4 | Keep engine choice/configuration inside the adapter; define trusted versioned policy selection and failure behavior. |
| Exact artifact enforcement and alternate execution paths | R5 | A commit match and policy success do not prove the reviewed artifact will be applied. |
| Verification evidence and pilot environment | R5 | Specify what state agreement proves and which operational consequences remain unknown. |
| Graph scale and presentation acceptance criteria | R6 | Choose representative plans and measurable investigation tasks; do not treat the static prototype's row counts as a product requirement. |

Keep priorities and completion criteria current as these decisions are made. Record
material choices with their rationale near the owning architecture/integration
document, and update [the handoff](handoff.md) when a slice changes the runnable path.
