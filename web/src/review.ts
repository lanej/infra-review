import type { Action, Change, Decision, Review, Violation } from "./contracts";
export type { Review } from "./contracts";
export type View =
  | "overview"
  | "changes"
  | "policies"
  | "execution"
  | "history";
export type UIState = {
  view: View;
  selected: string;
  root: string;
  query: string;
  detail: string;
  form: null | {
    action: Action;
    violationId: string;
    reason: string;
    evidence: string;
  };
  busy: boolean;
  error: string;
  notice: string;
};
export const initialUI = (): UIState => ({
  view: "changes",
  selected: "c3",
  root: "all",
  query: "",
  detail: "impact",
  form: null,
  busy: false,
  error: "",
  notice: "",
});
export const scenarios = [
  ["review", "Policy review"],
  ["unplanned", "Ready to plan"],
  ["ready", "Routine change"],
  ["incomplete", "Incomplete planning"],
  ["stale", "Stale approval"],
  ["expired", "Expired acceptance"],
  ["partial", "Partial apply failure"],
];
export const escapeHTML = (value: unknown): string =>
  String(value ?? "").replace(
    /[&<>"']/g,
    (c) =>
      ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[
        c
      ]!,
  );
const e = escapeHTML;
const labels: Record<string, string> = {
  ready_for_review: "Ready for review",
  unplanned: "Awaiting plan",
  incomplete: "Plan incomplete",
  stale: "Evidence stale",
  approved: "Approved",
  changes_requested: "Changes requested",
  partial: "Partially applied",
  applied: "Applied",
  verified: "State verified",
  not_started: "Not started",
  no_changes: "No changes",
  state_matches: "State matches",
  modify: "Update",
  create: "Create",
  replace: "Replace",
  delete: "Delete",
  planned: "Planned",
  failed: "Failed",
  granted: "Accepted",
  requested: "Pending owner",
  expired: "Expired",
  not_applicable: "Not applicable",
  satisfied: "Satisfied",
  violated: "Violated",
};
export const label = (value: string) =>
  Object.hasOwn(labels, value) ? labels[value] : value.replaceAll("_", " ");
const date = (value: string) => {
  const d = new Date(value);
  return Number.isNaN(d.getTime())
    ? "Not recorded"
    : d.toLocaleString(undefined, {
        month: "short",
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
      });
};
const tone = (value: string) =>
  [
    "critical",
    "failed",
    "partial",
    "changes_requested",
    "violated",
    "replace",
  ].includes(value)
    ? "danger"
    : ["high", "expired", "stale", "incomplete", "requested"].includes(value)
      ? "warning"
      : [
            "approved",
            "applied",
            "verified",
            "planned",
            "satisfied",
            "granted",
          ].includes(value)
        ? "good"
        : "neutral";
const badge = (text: string, state = "neutral") =>
  `<span class="badge badge--${tone(state)}">${e(text)}</span>`;
const btn = (text: string, action: string, attrs = "", style = "") =>
  `<button type="button" class="button ${style}" data-ui="${action}" ${attrs}>${e(text)}</button>`;
const gate = (r: Review, a: Action) => r.actions.find((x) => x.action === a);
const allowed = (r: Review, a: Action) => gate(r, a)?.outcome === "permitted";
export function decisionContext(r: Review, d: Decision): string {
  if (r.state === "stale" || (r.plan && r.plan.commitSha !== r.headSha))
    return "Stale evidence";
  if (!d.commitSha || !r.headSha) return "Commit not recorded";
  if (d.commitSha !== r.headSha) return "Previous commit";
  if (!d.planSetId) return "Plan not recorded";
  if (!r.plan || d.planSetId !== r.plan.id) return "Previous plan";
  if (!r.policy || !d.evaluationId || d.evaluationId !== r.policy.id)
    return "Previous assessment";
  return "";
}
export function currentApproval(r: Review): boolean {
  const latest = new Map<string, string>();
  for (const d of r.decisions)
    if (!decisionContext(r, d)) latest.set(d.actor, d.decision);
  return (
    ![...latest.values()].includes("changes_requested") &&
    [...latest.values()].includes("approved")
  );
}
export const currentAcceptance = (r: Review, v: Violation) =>
  [...r.acceptances]
    .reverse()
    .find(
      (a) =>
        a.violationId === v.id &&
        a.planSetId === r.plan?.id &&
        a.evaluationId === r.policy?.id &&
        a.policyVersion === v.policyVersion,
    );
const currentEvidence = (r: Review) =>
  !!r.plan &&
  r.plan.commitSha === r.headSha &&
  r.policy?.planSetId === r.plan.id &&
  r.policy?.coverage === "complete" &&
  r.policy.status === "completed" &&
  !["stale", "partial", "applied", "verified"].includes(r.state);
function workflowButton(
  r: Review,
  s: UIState,
  action: Action,
  text: string,
  primary = false,
) {
  return btn(
    text,
    "command",
    `data-action="${action}" ${s.busy || !allowed(r, action) ? "disabled" : ""} aria-describedby="reason-${action}"`,
    primary ? "button-primary" : "",
  );
}
function actionBar(r: Review, s: UIState): string {
  if (r.state === "verified")
    return btn(
      "View outcome",
      "view",
      'data-value="execution"',
      "button-primary",
    );
  if (r.state === "applied")
    return workflowButton(r, s, "verify", "Verify result", true);
  if (allowed(r, "apply"))
    return workflowButton(r, s, "apply", `Apply Plan ${r.plan!.number}`, true);
  if (allowed(r, "approve"))
    return workflowButton(
      r,
      s,
      "approve",
      `Approve Plan ${r.plan!.number}`,
      true,
    );
  if (
    r.plan &&
    !["stale", "partial", "incomplete", "unplanned"].includes(r.state)
  )
    return btn(
      "Review policies →",
      "view",
      'data-value="policies"',
      "button-primary",
    );
  return workflowButton(r, s, "plan", "Run plan", true);
}
function stateNotice(r: Review): string {
  if (
    currentEvidence(r) &&
    r.policy?.violations.some(
      (v) => v.blocking && currentAcceptance(r, v)?.status === "expired",
    )
  ) {
    return '<section class="notice notice--warning" role="status"><strong>Acceptance expired; apply is blocked</strong><p>The plan approval remains recorded. Resolve the violation or obtain renewed acceptance before applying.</p></section>';
  }
  const messages: Record<string, [string, string]> = {
    stale: [
      "A new commit needs a new plan",
      `The head is ${r.headSha}; displayed evidence belongs to ${r.plan?.commitSha}. Earlier approval does not cover this change.`,
    ],
    incomplete: [
      "The proposal is incomplete",
      "The network root failed to plan. Other roots are available to inspect, but approval and apply remain blocked.",
    ],
    partial: [
      "Execution stopped after some infrastructure changed",
      "API applied; network failed; observability has not started. Run a new plan from the resulting state before proceeding.",
    ],
    applied: [
      "Execution completed; verification is pending",
      "Check the resulting state separately. A successful command does not establish application health.",
    ],
    verified: [
      "Simulated state matches the plan",
      "Operational health and data recovery remain unverified. No live infrastructure was contacted.",
    ],
  };
  if (!Object.hasOwn(messages, r.state)) return "";
  return `<section class="notice notice--${tone(r.state)}" role="status"><strong>${e(messages[r.state][0])}</strong><p>${e(messages[r.state][1])}</p></section>`;
}
function lifecycle(r: Review): string {
  const planned = r.roots.filter((x) => x.status === "planned").length,
    approved = currentApproval(r);
  const blockers =
    r.policy?.violations.filter(
      (v) => v.blocking && currentAcceptance(r, v)?.status !== "granted",
    ).length ?? 0;
  const applied = ["applied", "verified"].includes(r.state);
  const steps = [
    [
      "Plan",
      `${planned}/${r.roots.length} roots planned`,
      !!r.plan && planned === r.roots.length,
    ],
    [
      "Review",
      approved
        ? blockers && !applied
          ? `Approved · ${blockers} blocker`
          : "Approved plan"
        : blockers
          ? `${blockers} blocking violation`
          : "Awaiting decision",
      approved && (blockers === 0 || applied),
    ],
    [
      "Apply",
      r.state === "partial"
        ? "Partial failure"
        : applied
          ? "Completed"
          : "Not started",
      applied,
    ],
    [
      "Verify",
      r.verification === "state_matches" ? "State matches" : "Not verified",
      r.verification === "state_matches",
    ],
  ];
  return `<ol class="lifecycle" aria-label="Review lifecycle">${steps.map(([name, detail, done], i) => `<li class="${done ? "is-done" : ""}"><span class="step-number">${done ? "✓" : i + 1}</span><span><strong>${name}</strong><small>${detail}</small></span></li>`).join("")}</ol>`;
}
export function visibleChanges(r: Review, s: UIState): Change[] {
  const rank: Record<string, number> = {
    critical: 0,
    high: 1,
    medium: 2,
    low: 3,
  };
  return r.changes
    .filter(
      (c) =>
        (s.root === "all" || c.rootId === s.root) &&
        `${c.name} ${c.address} ${c.summary}`
          .toLowerCase()
          .includes(s.query.toLowerCase()),
    )
    .sort((a, b) => (rank[a.risk] ?? 4) - (rank[b.risk] ?? 4));
}
function relationships(r: Review, c: Change): string {
  return c.relationships
    .map(
      (rel) =>
        `<div class="relationship"><span class="muted">${e(rel.kind)} →</span>${r.changes.some((x) => x.id === rel.resourceId) ? btn(rel.label, "inspect", `data-value="${e(rel.resourceId)}"`, "button-link") : `<span>${e(rel.label)}</span>`}<small>${e(rel.evidence)}</small></div>`,
    )
    .join("");
}
function resourceDetail(r: Review, s: UIState, c: Change): string {
  const findings =
    r.policy?.violations.filter((v) => v.resourceId === c.id) ?? [];
  let body = `<h3>What changes</h3><table class="diff-table"><thead><tr><th>Property</th><th>Before</th><th>After</th></tr></thead><tbody>${c.properties.map((p) => `<tr><th scope="row"><code>${e(p.name)}</code></th><td class="${p.before !== p.after ? "before" : ""}">${p.before !== p.after ? "− " : ""}${e(p.before)}</td><td class="${p.before !== p.after ? "after" : ""}">${p.before !== p.after ? "+ " : ""}${e(p.after)}</td></tr>`).join("")}</tbody></table><h3 class="subheading">Connected resources</h3>${relationships(r, c)}<p class="evidence-label">Evidence: ${e(r.plan?.id)} / ${e(c.rootId)}</p>`;
  if (s.detail === "plan")
    body = `<h3>Plan evidence</h3><p class="muted">${e(r.plan?.id)} · commit ${e(r.plan?.commitSha)}</p><pre>${e(c.planText)}</pre><p class="muted">Synthetic plan output for this demo.</p>`;
  if (s.detail === "source")
    body = `<h3>Configuration</h3><code>${e(c.sourcePath)}</code><pre>${e(c.sourceText)}</pre>`;
  if (s.detail === "relationships")
    body = `<h3>Dependency neighborhood</h3><p class="muted">Illustrative relationships, including unchanged context. External consumers may be missing.</p><div class="neighborhood"><strong>${e(c.name)}</strong>${relationships(r, c)}</div>`;
  return `<aside class="inspector" aria-label="Selected resource" aria-live="polite"><div class="section-heading"><span class="eyebrow">Resource inspector</span>${badge(label(c.action), c.action)}</div><h2>${e(c.name)}</h2><code class="resource-address">${e(c.address)}</code><div class="resource-context"><span>azure / prod / ${e(c.rootId)}</span>${badge(c.risk, c.risk)}</div>${findings.map((v) => `<div class="finding-callout ${v.blocking ? "is-blocking" : ""}"><strong>${v.blocking ? "Blocking" : "Advisory"} · ${e(v.title)}</strong><p>${e(v.consequence)}</p>${btn("Inspect policy and evidence →", "violation", `data-value="${e(v.id)}"`, "button-link")}</div>`).join("")}<nav class="detail-tabs" aria-label="Resource evidence">${["impact", "plan", "source", "relationships"].map((x) => btn(x === "relationships" ? "Dependencies" : label(x), "detail", `data-value="${x}" aria-pressed="${s.detail === x}"`)).join("")}</nav>${body}</aside>`;
}
function workspace(r: Review, s: UIState): string {
  const list = visibleChanges(r, s),
    selected = list.find((c) => c.id === s.selected) ?? list[0];
  return `<section class="workspace"><aside class="roots" aria-label="Infrastructure roots"><p class="eyebrow">Scope</p>${[{ id: "all", name: "All roots" }, ...r.roots.map((root) => ({ id: root.id, name: root.name.split("/").at(-1)! }))].map((root) => btn(`${root.name} · ${root.id === "all" ? r.changes.length : r.changes.filter((c) => c.rootId === root.id).length}`, "root", `data-value="${e(root.id)}" aria-pressed="${s.root === root.id}"`)).join("")}<p class="root-caption">azure / prod<br>${r.roots.length} expected roots<br><br>${r.roots.filter((x) => x.status === "planned").length} plans current</p></aside><section class="resource-list"><div class="list-heading"><h2>Changed resources</h2><p>${list.length} changes · consequences first</p><label class="search"><span aria-hidden="true">⌕</span><input type="search" id="resource-search" aria-label="Filter resources" placeholder="Filter resources" value="${e(s.query)}"></label></div>${list.map((c) => `<button type="button" class="resource-row" data-ui="select" data-value="${e(c.id)}" aria-pressed="${selected?.id === c.id}"><span class="resource-title">${e(c.name)}<span class="resource-glyph" aria-hidden="true">${c.id === "c3" ? "▤" : "◇"}</span></span><code>${e(c.address)}</code><span class="resource-summary">${e(c.summary)}</span><span class="resource-meta">${badge(label(c.action), c.action)}<span>${e(c.rootId)} · ${e(c.risk)}</span></span></button>`).join("") || '<p class="empty">No matching resource changes. Try another root or clear the filter.</p>'}</section>${selected ? resourceDetail(r, s, selected) : '<aside class="inspector"><h2>No resource selected</h2><p class="muted">Select a root with current evidence or clear the filter. A failed root still needs a successful plan.</p></aside>'}</section>`;
}
function readiness(r: Review, s: UIState): string {
  return `<section class="readiness"><h3>Next decision</h3>${[
    "plan",
    "approve",
    "apply",
    "verify",
  ]
    .map((action) => {
      const d = gate(r, action as Action);
      return `<div class="requirement"><span class="requirement-mark ${d?.outcome === "permitted" ? "is-good" : ""}" aria-hidden="true">${d?.outcome === "permitted" ? "✓" : "•"}</span><div><strong>${label(action)}</strong><p>${e(d?.reason ?? "Assessment unavailable")}</p></div></div>`;
    })
    .join("")}${actionBar(r, s)}</section>`;
}
function policyView(r: Review, s: UIState): string {
  const p = r.policy;
  if (!p)
    return '<section class="content"><h2>Assessment not available</h2><p>Run a plan to collect evidence.</p></section>';
  return `<section class="content policy-layout"><div><div class="section-heading"><div><p class="eyebrow">Policy assessment</p><h2>Requirements and consequences</h2></div>${badge(`${p.coverage} coverage`, p.coverage === "complete" ? "planned" : "incomplete")}</div><p class="muted">${e(p.policySetId)} · ${e(p.planSetId)} · ${e(date(p.evaluatedAt))}</p>${
    p.violations
      .map((v) => {
        const a = currentAcceptance(r, v),
          active = a?.status === "granted",
          pending = a?.status === "requested";
        const available =
          currentEvidence(r) && v.acceptanceAllowed && !active && !s.busy;
        return `<article class="violation" id="violation-${e(v.id)}"><div class="section-heading"><span class="eyebrow">${e(v.policyId)} / v${e(v.policyVersion)}</span>${badge(v.blocking ? (active ? "Blocking risk accepted" : "Blocks approval and apply") : "Advisory", v.blocking && !active ? "critical" : "high")}</div><h3>${e(v.title)}</h3><p>${e(v.explanation)}</p><dl class="consequences"><dt>Potential consequence</dt><dd>${e(v.consequence)}</dd><dt>Still unknown</dt><dd>${e(v.unknown)}</dd></dl><details><summary>Inspect supporting evidence</summary><ul>${v.evidence.map((item) => `<li>${e(item)}</li>`).join("")}</ul></details>${btn("Inspect resource →", "inspect", `data-value="${e(v.resourceId)}"`, "button-link")}${a ? `<section class="acceptance"><div class="section-heading"><strong>${active ? "Violation accepted" : `Acceptance ${e(label(a.status).toLowerCase())}`}</strong>${badge(label(a.status), a.status)}</div><p>${e(a.reason)}</p><p><strong>Evidence:</strong> ${e(a.evidence)}</p><small>${e(a.authorizedBy || `Requested by ${a.requestedBy}`)}${a.expiresAt ? ` · expires ${e(date(a.expiresAt))}` : " · database-owner decision pending"}</small></section>` : ""}${v.acceptanceAllowed ? `<div class="acceptance-action">${btn(pending ? "Simulate owner acceptance" : a?.status === "expired" ? "Request renewed acceptance" : "Request acceptance", "command", `data-action="${pending ? "grant_acceptance" : "request_acceptance"}" data-value="${e(v.id)}" ${available ? "" : "disabled"}`)}<p class="muted">${active ? "Acceptance preserves this violation. A separate plan approval is required." : pending ? "Morgan, the demo database owner, must authorize this request." : "Requires rationale, recovery evidence, and a database owner. Valid for one hour on this plan only."}</p></div>` : '<p class="muted">This advisory remains visible; no acceptance is required.</p>'}</article>`;
      })
      .join("") ||
    '<p class="empty">No violations found. Assessment coverage still applies.</p>'
  }</div><aside class="policy-sidebar"><h3>Evaluated policies</h3>${p.results.map((result) => `<div class="policy-result"><strong>${e(result.title)}</strong><code>${e(result.id)} · v${e(result.version)}</code>${badge(label(result.outcome), result.outcome)}</div>`).join("")}<h3 class="subheading">What acceptance means</h3><p class="muted">An authorized person accepts a specific risk for a limited time. The violation stays visible. Acknowledging a finding does not approve the plan.</p>${readiness(r, s)}</aside></section>`;
}
function overview(r: Review, s: UIState): string {
  const blocking = r.policy?.violations.some((v) => v.blocking);
  return `<section class="content overview"><div><p class="eyebrow">Decision brief</p><h2>${blocking ? "One replacement needs your attention." : "Review the consequences across every root."}</h2><p class="intro">${r.changes.length} resource changes across ${new Set(r.changes.map((c) => c.rootId)).size} roots. ${blocking ? "The database replacement needs recovery evidence and authorized acceptance." : "Inspect capacity, connectivity, and diagnostics before approving."}</p>${visibleChanges(
    r,
    { ...s, root: "all", query: "" },
  )
    .map(
      (c) =>
        `<article class="brief-change"><div class="section-heading"><h3>${e(c.name)}</h3>${badge(label(c.action), c.action)}</div><p>${e(c.summary)}</p>${c.properties[0] ? `<div class="inline-diff"><code>${e(c.properties[0].name)}</code><span>${e(c.properties[0].before)} → <strong>${e(c.properties[0].after)}</strong></span></div>` : ""}${btn("Inspect resource and evidence →", "inspect", `data-value="${e(c.id)}"`, "button-link")}</article>`,
    )
    .join(
      "",
    )}</div><aside>${readiness(r, s)}<h3 class="subheading">Roots in scope</h3>${r.roots.map((root) => `<div class="root-status"><span>${e(root.name)}</span>${badge(label(root.status), root.status)}</div>`).join("")}</aside></section>`;
}
function execution(r: Review, s: UIState): string {
  return `<section class="content"><div class="section-heading"><div><p class="eyebrow">Execution</p><h2>${r.state === "partial" ? "Understand what changed before retrying" : "Follow each root through the workflow"}</h2></div>${badge(label(r.verification), r.verification === "state_matches" ? "verified" : "neutral")}</div><p class="intro">Every attempt is simulated. Applied state, remaining work, and verification are tracked separately.</p><div class="execution-grid">${r.roots.map((root) => `<article class="execution-root"><div class="section-heading"><h3>${e(root.name)}</h3>${badge(label(root.applyStatus), root.applyStatus)}</div><p class="muted">Plan: ${e(label(root.status))}</p><details ${root.applyStatus === "failed" || root.status === "failed" ? "open" : ""}><summary>Latest output</summary><pre>${e(root.log)}</pre></details></article>`).join("")}</div><div class="execution-summary"><h3>Verification</h3><p>${r.verification === "state_matches" ? "The simulated resulting state matches the applied plan. Operational health and data recovery remain unverified." : r.state === "partial" ? "Verification is incomplete because execution stopped partway. Re-plan from the resulting state." : r.state === "applied" ? "Application completed. Verify the simulated resulting state next." : "No resulting-state verification has completed."}</p>${workflowButton(r, s, r.state === "applied" ? "verify" : "plan", r.state === "applied" ? "Verify resulting state" : "Run a new plan")}</div><h3 class="subheading">Attempt history</h3>${[
    ...r.attempts,
  ]
    .reverse()
    .map(
      (a) =>
        `<details class="attempt"><summary>${e(label(a.operation))} · ${e(a.rootId)} · ${e(a.planSetId)} · ${e(label(a.status))}</summary><small>${e(date(a.createdAt))}</small><pre>${e(a.log)}</pre></details>`,
    )
    .join("")}</section>`;
}
function decisions(r: Review): string {
  const outcomes: Record<string, string> = {
    approved: "approved",
    changes_requested: "requested changes",
    commented: "commented",
    dismissed: "had a review dismissed",
    pending: "has a pending review",
  };
  return (
    [...r.decisions]
      .reverse()
      .map(
        (d) =>
          `<article class="decision"><strong>${e(d.actor)} ${e(Object.hasOwn(outcomes, d.decision) ? outcomes[d.decision] : "has an unknown decision")}</strong>${badge(decisionContext(r, d) || "Current assessment", decisionContext(r, d) ? "stale" : d.decision)}<p>${e(d.planSetId)} · ${e(d.commitSha)} · ${e(date(d.createdAt))}</p>${d.message ? `<p>${e(d.message)}</p>` : ""}</article>`,
      )
      .join("") || '<p class="muted">No human decisions recorded.</p>'
  );
}
function history(r: Review): string {
  return `<section class="content history-layout"><div><p class="eyebrow">Review history</p><h2>Plans, decisions, and recovery</h2><div class="timeline">${
    [...r.history]
      .reverse()
      .map(
        (item) =>
          `<article><time>${e(date(item.createdAt))}</time><div><h3>${e(item.title)}</h3><p>${e(item.detail)}</p></div></article>`,
      )
      .join("") || '<p class="empty">Run a plan to start this review.</p>'
  }</div></div><aside><h3>Plan-bound decisions</h3>${decisions(r)}<h3 class="subheading">Previous plan versions</h3>${r.planHistory.map((plan) => `<div class="policy-result"><strong>Plan ${plan.number}</strong><code>${e(plan.commitSha)}</code><p>${plan.changeIds.length} changes · ${plan.rootIds.length} roots</p></div>`).join("") || '<p class="muted">No earlier versions in this session.</p>'}<h3 class="subheading">Acceptance history</h3>${r.acceptances.map((a) => `<div class="policy-result"><strong>${e(label(a.status))} · ${e(a.planSetId)}</strong><p>${e(a.reason)}</p><small>${e(a.authorizedBy || a.requestedBy)}${a.expiresAt ? ` · ${e(date(a.expiresAt))}` : ""}</small></div>`).join("") || '<p class="muted">No acceptances requested.</p>'}</aside></section>`;
}
function commandForm(r: Review, s: UIState): string {
  if (!s.form) return "";
  const { action, violationId, reason, evidence } = s.form;
  const names: Record<Action, string> = {
    plan: "Run a new plan",
    approve: `Approve Plan ${r.plan?.number}`,
    request_changes: "Request changes",
    request_acceptance: "Request violation acceptance",
    grant_acceptance: "Simulate database-owner acceptance",
    apply: `Apply Plan ${r.plan?.number} to production`,
    verify: "Verify the resulting state",
  };
  const note: Record<Action, string> = {
    plan: "Collect fresh evidence across all four roots. This creates a new plan version; earlier approvals and acceptances remain in history.",
    approve:
      "Your demo decision covers this exact plan, commit, and policy assessment. It does not start an apply.",
    request_changes:
      "Explain the required changes. This decision supersedes your earlier approval on this plan.",
    request_acceptance:
      "Explain why this risk is acceptable and identify recovery evidence. A request does not clear the policy gate.",
    grant_acceptance:
      "Simulate Morgan, the database owner, granting this request for one hour. The violation remains recorded; approval is a separate decision.",
    apply:
      "Simulate applying the approved plan to the listed roots. No real infrastructure is contacted. Resulting-state verification follows separately.",
    verify:
      "Compare simulated state with the applied plan. This does not verify application health or data recovery.",
  };
  const required = ["request_changes", "request_acceptance"].includes(action),
    v = r.policy?.violations.find((v) => v.id === violationId);
  return `<section class="command-panel" aria-label="Decision details"><div class="section-heading"><h2>${e(names[action])}</h2>${btn("Cancel", "cancel", s.busy ? "disabled" : "")}</div><p>${e(note[action])}</p>${v ? `<p class="decision-target">${e(v.title)} · ${e(v.policyId)} v${e(v.policyVersion)}</p>` : ""}<div class="decision-scope"><span>Commit <code>${e(r.headSha)}</code></span><span>${r.plan ? `Plan ${r.plan.number}` : "No plan yet"}</span><span>${r.roots.length} expected roots</span></div>${action === "apply" ? `<ul>${r.roots.map((root) => `<li>${e(root.name)} · ${r.changes.filter((c) => c.rootId === root.id).length} changes</li>`).join("")}</ul>` : ""}<form id="decision-form">${required || action === "approve" ? `<label for="decision-reason">${action === "request_acceptance" ? "Acceptance rationale" : "Review comment"}${required ? " (required)" : " (optional)"}</label><textarea id="decision-reason" name="reason" maxlength="4000" ${required ? "required" : ""} ${s.busy ? "disabled" : ""}>${e(reason)}</textarea>` : ""}${action === "request_acceptance" ? `<label for="decision-evidence">Recovery evidence (required)</label><textarea id="decision-evidence" name="evidence" maxlength="4000" required ${s.busy ? "disabled" : ""}>${e(evidence)}</textarea><p class="muted">Use a demo reference or description; evidence is not independently verified.</p>` : ""}<button class="button button-primary" type="submit" ${s.busy ? "disabled" : ""}>${s.busy ? "Working…" : e(action === "grant_acceptance" ? "Simulate acceptance" : action === "apply" ? "Simulate apply" : names[action])}</button></form></section>`;
}
export function renderReview(r: Review, s: UIState = initialUI()): string {
  const views: [View, string][] = [
    ["overview", "Overview"],
    ["changes", "Changes"],
    ["policies", "Policies"],
    ["execution", "Execution"],
    ["history", "History"],
  ];
  const content = !r.plan
    ? `<section class="empty-state"><p class="eyebrow">Start with evidence</p><h2>Understand the proposal before deciding.</h2><p>Four infrastructure roots are in scope. Run a plan to collect resource changes and evaluate the demo policies.</p>${workflowButton(r, s, "plan", "Run plan for 4 roots", true)}</section>`
    : s.view === "overview"
      ? overview(r, s)
      : s.view === "policies"
        ? policyView(r, s)
        : s.view === "execution"
          ? execution(r, s)
          : s.view === "history"
            ? history(r)
            : workspace(r, s);
  return `<div class="app-shell"><header class="app-header"><a class="brand" href="/" aria-label="Statecraft home"><span class="brand-mark" aria-hidden="true">◈</span> STATECRAFT</a><span class="repo-label">${e(r.repository)}</span><span class="demo-label">Mock workspace · no live infrastructure</span></header><div class="demo-toolbar"><label for="demo-scenario">Explore a workflow</label><select id="demo-scenario" ${s.busy ? "disabled" : ""}>${scenarios.map(([value, name]) => `<option value="${value}" ${r.scenario === value ? "selected" : ""}>${name}</option>`).join("")}</select><span>Demo reviewer: Alex</span>${btn("Reset scenario", "reset", s.busy ? "disabled" : "", "button-link")}</div><section class="review-heading"><div><p class="eyebrow">Pull request #${e(r.pullRequest)}</p><h1>${e(r.title)}</h1><div class="review-meta">${badge(label(r.state), r.state)}<span>main ← capacity-and-access</span><code>${e(r.headSha)}</code></div></div><div class="heading-actions">${actionBar(r, s)}${r.plan && !["stale", "partial", "incomplete"].includes(r.state) && allowed(r, "plan") ? workflowButton(r, s, "plan", "Run plan") : ""}${r.plan ? workflowButton(r, s, "request_changes", "Request changes") : ""}</div></section>${lifecycle(r)}${stateNotice(r)}${s.error ? `<div class="notice notice--danger" role="alert">${e(s.error)}</div>` : ""}${s.notice ? `<div class="notice" role="status">${e(s.notice)}</div>` : ""}${commandForm(r, s)}<nav class="view-tabs" aria-label="Review views">${views.map(([value, name]) => btn(`${name}${value === "changes" && r.plan ? ` ${r.changes.length}` : value === "policies" && r.policy ? ` ${r.policy.violations.length}` : ""}`, "view", `data-value="${value}" aria-pressed="${s.view === value}"`)).join("")}</nav>${content}<footer class="app-footer"><span>${r.plan ? `Plan ${r.plan.number} · commit ${e(r.plan.commitSha)}` : "No plan collected"} · ${r.roots.length} expected roots</span><span>Decisions follow the exact plan · session ${e(r.id.slice(-6))}</span></footer><div class="sr-only">${r.actions.map((d) => `<p id="reason-${d.action}">${e(d.reason)}</p>`).join("")}</div></div>`;
}
