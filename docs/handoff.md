# Agent handoff

This describes the review-workbench implementation introduced in
[PR #7](https://github.com/lanej/statecraft/pull/7), following the adapter foundation
and policy design. Check the current checkout and PR state before using that link
as a statement about `main`; this document describes code, not merge status.

## Read and decide

Start with [AGENTS.md](../AGENTS.md). The [product definition](product.md) owns
intent and invariants; [DESIGN.md](../DESIGN.md) owns interaction rules;
[architecture](architecture.md), [integrations](integrations.md), and
[policies](policies.md) own boundary semantics. The [roadmap](roadmap.md) owns
priority, feature gaps, and completion criteria. This file owns the current code
map and practical continuation notes.

The recommended next implementation is **R1: generated Connect transport**. Keep
it bounded to replacing the handwritten runtime boundary without changing the
review behavior or enabling external execution. User feedback on the workbench can
be addressed independently; do not treat the current composition as a frozen design.

## Current baseline

| Area | What exists | What it does not establish |
| --- | --- | --- |
| Review workspace | Overview, root/resource comparison, inspector, before/after, illustrative source/plan evidence, relationships, policy, execution, history | A scalable graph, real plan parsing, source editing, collaboration, or historical evidence comparison |
| Workflow | Plan, request changes, request/grant acceptance, approve, apply, verify | Authentication, real role separation, persistent jobs, or production execution |
| Policy | `PolicyEvaluator` and `ActionPolicy` ports with deterministic sample rules | OPA/Rego evaluation, trusted policy distribution, reference-data/input digests, revocation, or organization policy administration |
| Store | Isolated in-memory sessions; transactional expected-version mutation; detached reads | Durable storage, restart recovery, retention, tenant isolation, or a production audit store |
| Integration adapters | GitHub reads/reviews/checks; Atlantis command mapping and notification decoding; local HTTP contract tests | Runtime credentials, configured installations, authenticated webhook ingress, reconciliation, or exact-artifact apply |
| API | Versioned protobuf definition plus runnable JSON routes and handwritten TS contract | Generated Connect handler/client composition |

GitHub/Atlantis/OPA types do not define the frontend or domain. Initial sample
models deliberately cover less than the target architecture; do not read every
future entity in `architecture.md` as an implemented feature.

## Code map

| Change you need to make | Start here |
| --- | --- |
| Runtime composition and listen address | `cmd/statecraft/main.go` |
| Demo HTTP routes, decoding, error mapping | `internal/adapters/httpapi/demo.go` and its test |
| Workflow transitions and command-time revalidation | `internal/service/demo_workflow.go` and its test |
| Review, evidence, decision, and acceptance shapes | `internal/domain/review.go`, `internal/domain/workflow.go` |
| Source and execution contracts | `internal/domain/source_control.go`, `internal/domain/execution.go`, `internal/ports/` |
| Seed scenarios and recovery planning | `internal/adapters/mock/scenarios.go` |
| Sample assessment and action gates | `internal/adapters/mock/policies.go` |
| Transactional demo store | `internal/adapters/mock/store.go` |
| Source metadata refresh and invalidation | `internal/service/source_reviews.go`, `internal/domain/source_control.go` |
| External mappings | `internal/adapters/github/`, `internal/adapters/atlantis/` |
| Public API schema and generation | `proto/statecraft/v1/review.proto`, `buf.yaml`, `buf.gen.yaml` |
| Frontend contract and request/focus/session behavior | `web/src/contracts.ts`, `web/src/main.ts` |
| View composition and presentation logic | `web/src/review.ts`, `web/src/style.css` |
| Frontend rendering regressions | `web/test/review.test.mjs` |
| Actual running-UI examples | `docs/screenshots/` |
| Legacy static composition checks | `.ui-review/config.json`, `.ui-review/rules.json` — target root-level prototype only |

The frontend is plain TypeScript with HTML render functions, not a component
framework. Refactor when a use case warrants it; a framework migration is not a
prerequisite for the next feature. The raw source/plan strings are illustrative
fixtures, not parsed or executable infrastructure configuration.

## Run and exercise

Use the commands in [AGENTS.md](../AGENTS.md) and the HTTP routes/port options in
[steel-thread.md](steel-thread.md). The current app creates a session through the
backend rather than importing frontend fixture JSON. Session identity lives in
per-tab browser storage; reset/scenario selection creates another session. The
API caps the store at 512 sessions and a restart clears them.

| Selector option | Expected observation / next action |
| --- | --- |
| Policy review | Database replacement blocks approval/apply. Request acceptance with rationale and recovery evidence; simulate the database owner's grant; separately approve, apply, then verify. |
| Ready to plan | No plan evidence is shown. Run a plan to collect it. |
| Routine change | Advisory connectivity concern remains visible. Approval is separate from apply. Requesting changes supersedes that reviewer's approval. |
| Incomplete planning | Network failed and supplies no current resource changes. Replanning restores the root's evidence and complete assessment. |
| Stale approval | New head differs from the retained plan. Replanning creates new evidence; old decisions remain historical. |
| Expired acceptance | Approval remains recorded, the violation remains violated, and expiry blocks apply. Request renewed acceptance. |
| Partial apply failure | API already changed, network failed, observability did not start. Replanning excludes applied changes and requires a decision on remaining work. |

The owner and reviewer are fixed demo personas. Simulating owner acceptance is not
an authorization mechanism. Background refresh updates displayed eligibility;
every command still checks current state and expiry on the server. A stale version
returns a conflict instead of silently replaying a decision.

## Limits that matter for the next agent

- `PlanSnapshot` uses review-local IDs and synthetic digests. History keeps plan
  identities/change IDs, human decisions, and simulated attempts; it does not keep
  full immutable old plans, evaluations, relationships, and evidence for replay.
- `PlanPolicyInput` is currently a review plus an explicit time. `ActionPolicyInput`
  adds the action and violation ID. Trusted actor/role and versioned reference-data
  context from the policy design are not implemented.
- Changes use display strings for properties. Typed values, unknowns, sensitive
  values, stable resource identity across plans, and evidence references need a
  normalized plan-ingestion design before processing real artifacts.
- Demo transitions complete synchronously. There is no durable queue, operation
  receipt, uncertain-result recovery, external lock model, or execution cancel path.
- The JSON API and TS contract can drift from protobuf until R1. Remote Buf plugins
  are not pinned yet; generation and reproducibility are part of that task.
- Current tests exercise domain/service behavior, provider mapping, HTTP validation,
  and rendered HTML. Browser walkthroughs/screenshots exist, but there is no
  automated end-to-end browser suite or comprehensive accessibility verification.
- The `.ui-review` selectors and comparison counts describe the older nine-resource
  static prototype, not the four-change workbench. They must be migrated before
  claiming that tool validates this UI.
- The Atlantis command API's replan-before-apply behavior is a production blocker,
  not a frontend confirmation problem. [Read the boundary](integrations.md#apply-is-not-approval-enforcement)
  before proposing live execution.

## Keep the handoff useful

When finishing a slice, update current status and the next bounded task, link any
new source-of-truth design or adapter contract, and record unresolved decisions.
Do not preserve chat chronology, machine-specific setup, or passing test counts as
permanent product documentation. Evidence, remaining limitations, and decisions
another agent would otherwise have to rediscover are the useful handoff.
