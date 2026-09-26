# Statecraft UI design contract

## Intent and current surface

Statecraft is a dense analytical workspace for understanding and deciding on
infrastructure change. The TypeScript app in `web/` exercises the review lifecycle
against an isolated Go mock backend. Root-level `index.html`, `app.js`, and
`styles.css` are an older static composition reference, not the current application.

[Product intent](docs/product.md) defines users and invariants. The
[roadmap](docs/roadmap.md) defines feature status and delivery order. This document
owns interaction and presentation requirements; the current screenshots are
examples, not a frozen visual specification.

## Principles

- Lead with the next decision, its scope, and the evidence supporting it.
  Completeness, consequential findings, execution state and human decisions outrank
  decorative summaries.
- Preserve comparison. Resource changes should remain scannable as peers, with
  complete identities and nearby before/after values. Density is useful only while
  labels, grouping, and evidence remain readable.
- Explain consequences before syntax. Keep observed facts, inferred effects, and
  unknowns distinct. Offer the underlying source/plan/log evidence beside summaries.
- Use the same resource and plan identity across overview, changes, policy,
  relationships, execution and history. Selection should converge on one inspector
  context rather than separate disconnected details.
- Use progressive disclosure. Begin with consequential changed resources, then
  expand a relevant neighborhood or evidence trail. Do not require understanding
  an entire infrastructure graph to review one change.
- Make state and uncertainty explicit. Missing evidence, stale evidence, an accepted
  violation, an approved proposal, completed execution and verified state must not
  share a generic green "ready" presentation.

## Decision workflow

| Decision | Show before the action | Explain afterward |
| --- | --- | --- |
| Run plan | Source revision, expected roots, existing failed/stale evidence, and which prior decisions a new plan will invalidate | Per-root progress/failure, coverage and a new proposal identity; no implication that infrastructure was applied |
| Request changes | Exact plan/commit and the concern or missing evidence | Who requested the change, affected proposal, and whether that supersedes a prior approval |
| Request acceptance | Rule/version, violation, consequence, unknowns, acceptance eligibility, required rationale/evidence and authorizer | A pending request is still blocking; requesting is separate from granting |
| Grant acceptance | Exact violation/assessment scope, requester, evidence, conditions, duration and authorizer requirements | Violation remains violated; acceptance has its own status/expiry and does not approve the proposal |
| Approve plan | Exact commit and complete root/plan scope, current assessment, accepted violations, remaining requirements | An attributable approval with its binding; apply remains a separate action |
| Apply plan | Approved artifact scope, target environment/roots, current gates, destructive consequences, and effect of partial execution | What started, changed, failed, or is uncertain; no automatic equivalence to verified health |
| Verify | Applied proposal, observed state/evidence and limits of the verification | State agreement, divergence, missing observations and operational health remain distinct |
| Revert (planned) | Forward-change lineage, proposed reverse change, data/state effects that cannot be restored automatically | A new proposal subject to the same controls, not an undo promise |

Disabled actions need an understandable reason and a useful path forward. Place
relevant evidence and unmet requirements near the action; do not rely solely on a
hover tooltip. Backend policy remains authoritative when displayed state changes.

## Workbench composition

The current desktop layout places root scope, a consequential resource list, and a
persistent inspector beside one another. The inspector supports before/after,
illustrative plan/source output, and related resources. Policies show the purpose
of a rule, observed facts, consequences, unknowns, supporting evidence, and any
acceptance. Execution shows each root and its logs. History retains decision scope
and invalidation context.

Future graph navigation, contextual discussion and plan comparisons should extend
this common evidence model. A broad source editor, policy authoring studio, or
cloud administration console is not required to complete the review workflow.

## States to preserve and extend

| State | Required presentation | Current coverage |
| --- | --- | --- |
| No plan / loading / failed request | Explain what is unavailable, avoid speculative evidence, provide a recovery action and preserve useful drafts | Mock empty/error/pending behavior; automated browser coverage still needed |
| Incomplete plan | All expected roots remain visible; failed/missing roots are not zero-change successes; show available evidence as partial | Mock scenario |
| Stale plan or approval | Display old and current scope and why a fresh decision is needed | Mock new-commit and replan behavior |
| Policy violation | Severity, consequence and enforcement are separate; show acceptance only when the rule permits it | Mock blocking/acceptance and advisory rules |
| Pending / accepted / expired acceptance | Preserve the violation; pending does not clear a gate; expiry can block apply while approval stays recorded | Mock scenarios and workflow |
| Prohibited / denied / revoked acceptance; policy revision | Explain who can resolve it or why no acceptance path exists; invalidate affected eligibility | Required future scenarios, not implemented |
| Conflicting or stale command | Explain changed context, refresh current eligibility and preserve a useful decision draft | Mock version conflict handling |
| Running / partial / uncertain execution | Separate started, applied, failed and unstarted roots; explain reconciliation before retry | Seeded partial failure; durable running/uncertain outcomes still needed |
| Applied / verified / health unknown | Successful execution does not imply resulting-state agreement or operational health | Mock apply/verify distinction |

## Visual and interaction requirements

Statecraft brand assets live in `assets/`. The running app owns presentation in
`web/src/style.css` and view composition in `web/src/review.ts`. Preserve its restrained
forest/neutral palette, semantic risk accents and evidence density unless a design
change has a clear reason. Color must not carry state by itself.

Keyboard focus must remain useful after filtering, changing views, updating state,
opening/closing a decision form and submitting a command. Controls need meaningful
names and status/error announcements without repeatedly reading the entire page.
Use semantic comparisons and readable contrast. Check zoom and long resource names,
not only the short fixture labels. On narrow screens the inspector may stack below
the list, with a clear path between the selected resource and its evidence.

### TASK-001 — Changed-resource comparison

Preserve enough comparable alternatives, full identities, risk/action distinctions,
and before/after evidence to understand the proposal without opening raw output.
Choose representative small and large plans; a fixed synthetic row count is not a
product requirement.

### TASK-002 — Summary hierarchy

Make the current proposal, coverage, consequential concerns and next action easy
to find. Equal-priority summaries should align without overlap. Avoid adding metrics
whose relationship to the decision is unclear.

### TASK-003 — Graph investigation

Provide bounded neighborhoods and paths connected to the resource inspector, with
root/module grouping and relationship provenance. The current relationship list and
legacy graph do not establish large-plan usability or a complete graph browser.

### TASK-004 — Evidence continuity

A user can move from an overview concern to a resource, property change, policy
rule and original evidence while retaining the proposal context. Comparisons and
history must label which version supplied the evidence.

## Verification and PR evidence

Exercise the relevant normal and blocked/recovery states in the running `web/`
app with keyboard and pointer, including a narrow layout. Existing rendered-HTML
tests and screenshots do not replace automated browser coverage or a full
accessibility assessment.

`.ui-review/config.json` and its rules currently target the legacy root-level
prototype and its nine-resource fixture. Their selectors and counts have not been
migrated to `web/`; do not claim they validate this workbench. Recalibrate them
against the requirements above before relying on those checks.

UI pull requests include actual screenshots of the current running result, with
context sufficient to assess the change. Use durable URLs that render inline on
GitHub and verify they load. Include a blocked/recovery state when relevant, and
keep the PR prose focused on the final behavior and why it matters. Documentation-only
updates do not need new screenshots.
