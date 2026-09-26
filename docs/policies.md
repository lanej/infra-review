# Policies, violations, and acceptance

This is the design contract for the next policy integration. The current runtime
still uses mock findings; the policy ports, OPA adapter, persistence, and acceptance
actions described here are not implemented yet.

Statecraft owns the meaning of a policy assessment and the decisions made from it.
OPA evaluates Rego behind an adapter, just as GitHub and Atlantis supply their
capabilities behind adapters. Rego packages, queries, bundles, SDK values, and raw
decision documents are not domain or frontend models.

## Domain vocabulary

| Concept | Meaning |
| --- | --- |
| Policy | An identified, versioned requirement with a human-readable purpose, applicability, and evidence requirements. Severity and enforcement are separate: a serious concern can be advisory, and a low-severity requirement can block an action. |
| PolicySet | An immutable selection of policy versions and configuration, identified by a digest. This identifies the applicable rules independently of the engine's deployment or bundle format. |
| PolicyEvaluation | An immutable assessment of an exact PlanSet and normalized input snapshot against an exact PolicySet and reference-data snapshot. It records scope, coverage, timestamps, per-policy outcomes, violations, and evaluation errors. |
| PolicyViolation | A specific unmet requirement, referencing its policy version, evaluation, affected roots/resources/properties, explanation, and evidence. It can be presented as a policy-derived Finding; other analyses can produce Findings without producing policy violations. |
| ViolationAcceptance | An authorized decision to tolerate specified violations under the applicable policy. It records the requester and authorizer, rationale, scope, conditions, evidence, expiry, and revocation history. |
| ActionDecision | Whether an actor may perform a particular action on an exact target now, with the requirements, reasons, and evidence behind that result. |

Acceptance does not modify the original violation or make the policy compliant.
The UI can show **violated · accepted until 18:00**, while the action gate records
that a permitted acceptance satisfied one requirement. Acknowledging a finding or
resolving a discussion is not acceptance. Accepting one violation is not approving
the PlanSet, and approving a PlanSet does not implicitly accept its violations.

The initial acceptance scope should be a specific set of violation IDs, evaluation,
PlanSet, and policy versions. A correlation fingerprint can help compare violations
between evaluations; it cannot transfer acceptance to a new plan. Standing waivers
are a separate future use case, not an implicit extension of this scope.

## Two policy responsibilities

Infrastructure assessment asks: **What requirements does this proposal violate?**
Workflow authorization asks: **May this actor perform this action, given the current
evidence and requirements?** The same OPA adapter may supply both, but their inputs
and outputs should have separate typed contracts.

Proposed outbound ports, to be introduced alongside their application use cases:

```text
PolicyEvaluator
  EvaluatePlan(PlanPolicyInput) -> PolicyEvaluation

ActionPolicy
  EvaluateAction(ActionPolicyInput) -> ActionDecision
```

`PlanPolicyInput` contains a schema version, PlanSet identity, expected root scope,
normalized resource changes and relationships, evidence coverage, PolicySet
identity, and versioned reference facts. Unknown relationships or redacted values
are explicit; the evaluator must not infer completeness from an empty collection.

`ActionPolicyInput` contains the intended action (run plan, approve plan, accept a
violation, apply), exact target/scope, authenticated actor, current evaluation,
current approval and acceptance records, and authoritative external requirements.
The application assembles these inputs from trusted services and stores. It does
not trust role claims, acceptance status, or policy selection supplied by the UI.

Required context depends on the action: running a plan targets a source revision
and expected roots before a PlanSet exists; approval and apply target an exact
PlanSet; acceptance targets specified violations. Planning must not depend on an
assessment that only planning can produce.

An ActionDecision has an outcome of `permitted`, `denied`, or `indeterminate`, plus
typed requirements and reasons. A requirement identifies its source, scope,
fulfillment evidence, and who can resolve it. No generic query string or arbitrary
engine JSON is exposed through either port. Published contracts use Statecraft
types; raw engine results are retained separately as evidence.

Statecraft application services enforce decisions, persist evaluations and human
actions, and dispatch permitted commands. The evaluator neither records human
approval nor starts execution. Policy can report that human review is not required
under an externally configured rule; Statecraft must not fabricate an approval
record to represent that result.

## Completion, compliance, and eligibility

These are independent dimensions:

- **Execution:** pending, running, completed, or failed evaluation.
- **Coverage:** complete or incomplete, with expected and actually evaluated scope.
- **Per-policy outcome:** satisfied, violated, not applicable, or indeterminate.
- **Acceptance:** requested, granted, denied, expired, revoked, or invalidated.
- **Action eligibility:** permitted, denied, or indeterminate for the requested
  action and actor.

Evaluation attempts retain their lifecycle events; completed assessments are
immutable. The current mock Finding's `Blocking` boolean is only a presentation
projection, not a substitute for these per-action requirements or acceptance state.

A successful engine call is not proof that all expected policies ran. An empty
violation list is not sufficient evidence of compliance. Missing output, unsupported
schema, unavailable required facts, timeout, and malformed results remain errors or
indeterminate assessments. Required incomplete or indeterminate assessments cannot
authorize approval or apply. A valid, explicitly complete result with no violations
can establish compliance for its declared scope.

A policy can prohibit acceptance or require a particular role, independent
authorization, supporting evidence, conditions, and a maximum duration. Those
requirements are themselves evaluated before granting acceptance. A requested
acceptance never clears a gate. If evidence required to accept a violation is
missing, that acceptance remains unavailable.

## Identity, freshness, and enforcement

Persist the policy versions, configuration digest, reference-data digest, input
schema and digest, evaluation time, and engine provenance used for each assessment.
Plan identity alone cannot reproduce an assessment when policy or external facts
have changed. Time-dependent rules use an explicit evaluation time and validity
boundary rather than an unrecorded clock dependency.

A new plan, applicable policy revision, changed material reference facts, or expired
assessment requires reevaluation. Old assessments, acceptances, and approvals remain
in history but cannot establish current eligibility by themselves. Changed
requirements may require renewed human approval even when the infrastructure plan
is unchanged. Acceptance expiry or revocation also reopens the relevant gate.

Before approval, acceptance, or execution, the backend checks current eligibility
and verifies that the target and assessment context still match. If anything changes
between the displayed decision and dispatch, stop and present the changed context.
Disabling a frontend button is presentation, not enforcement.

GitHub rules, Statecraft policy decisions, and Atlantis execution requirements keep
their own provenance. Passing one does not bypass another. Atlantis policy results
remain external execution-gate evidence unless their identity, scope, and versions
can be reconciled with a Statecraft assessment. If an upstream result combines
policy success with approved exceptions, retain that distinction or mark the
missing detail unknown; do not invent a clean compliance result.

The [existing apply limitation](./integrations.md#apply-is-not-approval-enforcement)
still applies: authorizing execution cannot turn an API that replans into a guarantee
that the approved artifact will be applied. Enforcement must extend through the
execution boundary, including other enabled Atlantis command paths. Until that
boundary is established, policy readiness must not be presented as production apply
authorization.

## OPA adapter

OPA is the first intended policy engine. Prefer its official
[`github.com/open-policy-agent/opa/v1/sdk`](https://www.openpolicyagent.org/docs/integration#integrating-with-the-go-sdk)
when embedding OPA with policy distribution and management. Its official
`v1/rego` package supports evaluation-only embedding. A separately operated OPA
service can instead use its [REST decision API](https://www.openpolicyagent.org/docs/rest-api).
Deployment choice remains an adapter/composition decision, without changing the
domain ports.

The adapter owns:

- mapping versioned Statecraft inputs into policy documents;
- configured decision paths, Rego packages, bundle loading, and engine lifecycle;
- validating a versioned decision schema before mapping it into domain results;
- checking expected policy coverage and actual policy/data revisions;
- translating errors and preserving diagnostic evidence without leaking SDK types;
- enforcing execution limits and the configured capabilities available to policies.

The decision schema is a Statecraft integration contract implemented by our policy
packages, not a standard OPA violation format. It must report evaluated policy
identities and outcomes, including explicit not-applicable results. Evaluation
selection must come from trusted policy configuration so a proposal cannot remove
its own blocking rules. Policy-distribution administration is distinct from
accepting a violation on a review.

OPA can return HTTP 200 with no `result` for an undefined named decision. Treat that
as indeterminate, never as a pass. Rego built-in runtime errors can also evaluate
to undefined by default; use strict built-in error handling where supported, schema
validation, and policy tests. Strict errors do not make an otherwise undefined
decision complete. See the official [integration guide](https://www.openpolicyagent.org/docs/integration)
and [Rego error behavior](https://www.openpolicyagent.org/docs/policy-language#errors).

## Review workflow and presentation

1. **Assess the proposal.** Evaluate the complete PlanSet and expose missing roots,
   policies, and evidence before asking for a decision.
2. **Explain the violation.** Show the rule's purpose, the observed facts, inferred
   consequences, remaining unknowns, and the exact evidence. Distinguish severity
   from its effect on each action.
3. **Resolve or request acceptance.** Offer a source correction, the policy's
   permitted acceptance path, or a request for the required reviewer. Show why
   acceptance is unavailable when the policy prohibits it.
4. **Authorize acceptance.** Show the exact violation scope, PlanSet, rationale,
   evidence, conditions, duration, and authorizer requirements. Preserve the
   violation after a grant and show when its acceptance expires.
5. **Approve the proposal.** Present remaining requirements, accepted violations,
   and current evidence beside the PlanSet approval. Keep acceptance and approval
   as separately auditable decisions.
6. **Check execution eligibility.** Revalidate freshness, acceptance validity,
   approval requirements, and external gates against the exact proposed operation.
7. **Track the outcome.** Keep violations, acceptances, policy revisions, execution,
   and verification connected in history.

Example: a production database replacement violates a data-protection requirement.
Policy may permit acceptance only after a database owner supplies recovery evidence
and an authorized reviewer grants a limited exception. The UI shows the violation,
the evidence, and the active acceptance together. A separate PlanSet approval is
still required if the workflow policy calls for one; an expired acceptance blocks
apply even if that approval remains in history.

The next interactive design should exercise a hard prohibition, an advisory
violation, a permitted acceptance, an unauthorized acceptance request, a policy
revision after approval, and an expired acceptance immediately before apply.
