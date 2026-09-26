# Statecraft UI design contract

## Purpose and task

Statecraft is a dense analytical workspace for understanding, diagnosing, assessing, discussing, approving, applying, verifying, and reverting infrastructure changes. The TypeScript workbench in `web/` now exercises the review lifecycle against an isolated mock backend. The original root-level prototype remains a separate composition reference.

## Principles and tradeoffs

- Lead with the review decision: completeness, consequential findings, execution state, and approvals outrank decorative chrome.
- Preserve comparison: changed resources remain scannable as peers; do not trade evidence for empty space on larger displays.
- Keep graph and inspector coupled: selecting evidence in the graph or table must converge on the same resource context.
- Use progressive disclosure for complicated plans rather than rendering the entire infrastructure graph by default.
- Density is intentional. Readability, alignment, and coherent grouping constrain density; whitespace is not a goal by itself.
- Raw plan and execution evidence remain reachable even when Statecraft provides semantic summaries.

## Visual system and ownership

Statecraft brand assets live under `assets/`. The running workbench owns its presentation in `web/src/style.css` and its view functions in `web/src/review.ts`. Viewrule measures the rendered contract rather than prescribing components.

## Behavior and resilience

The workbench preserves a resource inspector alongside root filtering and changed-resource comparison. Policy details show facts, consequences, unknowns, and evidence; acceptance and plan approval remain separate. The scenario selector exercises incomplete plans, stale approvals, expired acceptance, and partial execution recovery. Decision forms show exact plan/commit scope before committing a simulated action. Keyboard focus survives view changes and filtering. The responsive layout stacks the inspector below the change list on small screens.

Large graph navigation, source editing, policy administration, and production actor/role enforcement remain future work.

## Requirements and verification

### TASK-001 — Changed-resource comparison
Mandatory. The changed-resource table must preserve enough alternatives and complete labels to compare the plan without opening raw output. Viewrule checks declared comparison count, context, labels, and row spacing.

### TASK-002 — Summary hierarchy
Mandatory. Equal-priority summary metrics must align and remain non-overlapping. Viewrule checks geometry; human review determines whether the chosen metrics deserve prominence.

### TASK-003 — Graph investigation
Mandatory product requirement, partially assessed in the prototype. The graph must support focused review of changed resources and relationships. Current Viewrule rules do not establish graph comprehensibility or large-plan scalability; those require interaction tests and human review.

### TASK-004 — Evidence continuity
Mandatory product requirement, currently human-reviewed. A selected change should preserve a path from summary to resource details and underlying evidence.

## Evidence and unresolved decisions

Product scope and assumed workflows are defined in `docs/product.md`; architecture is in `docs/architecture.md`. The current mock plan/state are synthetic. This baseline is not an approved visual reference. Composition rules should be recalibrated when the TypeScript frontend replaces the prototype.

## UI change evidence

UI pull requests include actual screenshots of the current running result. Use
stable image URLs that render inline in the PR, and verify the URLs load before
handoff. Keep the description focused on the final change and why it matters;
automated checks report their own results.
