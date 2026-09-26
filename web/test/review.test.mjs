import assert from "node:assert/strict";
import test from "node:test";
import {
  renderReview,
  initialUI,
  currentApproval,
  currentAcceptance,
  decisionContext,
  visibleChanges,
} from "../src/review.ts";

function decision(overrides = {}) {
  return {
    actor: "alex",
    decision: "approved",
    planSetId: "plan-7",
    commitSha: "head",
    evaluationId: "eval-7",
    message: "",
    createdAt: "2026-09-26T12:00:00Z",
    ...overrides,
  };
}
function review(overrides = {}) {
  return {
    id: "demo-42",
    repository: "example/infra",
    pullRequest: 42,
    title: "Review infrastructure",
    headSha: "head",
    state: "ready_for_review",
    demo: true,
    version: 1,
    scenario: "review",
    verification: "not_started",
    plan: {
      id: "plan-7",
      number: 7,
      commitSha: "head",
      rootIds: ["api"],
      changeIds: ["c3"],
      createdAt: "2026-09-26T12:00:00Z",
    },
    policy: {
      id: "eval-7",
      planSetId: "plan-7",
      policySetId: "policies-v1",
      status: "completed",
      coverage: "complete",
      evaluatedAt: "2026-09-26T12:00:00Z",
      results: [],
      violations: [],
    },
    roots: [
      {
        id: "api",
        name: "prod/api",
        status: "planned",
        applyStatus: "not_started",
        log: "plan output",
      },
    ],
    changes: [
      {
        id: "c3",
        rootId: "api",
        name: "Primary database",
        address: "database.main",
        action: "replace",
        risk: "critical",
        summary: "Replacement",
        properties: [{ name: "name", before: "old", after: "new" }],
        relationships: [],
        planText: "plan evidence",
        sourcePath: "api/main.tf",
        sourceText: "configuration",
      },
    ],
    findings: [],
    decisions: [],
    acceptances: [],
    attempts: [],
    history: [],
    planHistory: [],
    actions: [
      {
        action: "approve",
        outcome: "denied",
        reason: "Resolve the blocking violation",
      },
    ],
    ...overrides,
  };
}

test("only an approval bound to the current plan and assessment counts", () => {
  assert.equal(currentApproval(review({ decisions: [decision()] })), true);
  for (const changed of [
    { decision: "commented" },
    { decision: "dismissed" },
    { decision: "pending" },
    { decision: "unknown" },
    { decision: "toString" },
    { commitSha: "old" },
    { commitSha: "" },
    { planSetId: "plan-6" },
    { planSetId: "" },
    { evaluationId: "old" },
    { evaluationId: "" },
  ]) {
    assert.equal(
      currentApproval(review({ decisions: [decision(changed)] })),
      false,
      JSON.stringify(changed),
    );
  }
  assert.equal(
    currentApproval(review({ state: "stale", decisions: [decision()] })),
    false,
  );
  assert.equal(
    currentApproval(
      review({
        decisions: [decision(), decision({ decision: "changes_requested" })],
      }),
    ),
    false,
  );
});
test("decision history retains each outcome and its original identity", () => {
  const r = review({
    decisions: [
      decision(),
      decision({ actor: "old", planSetId: "plan-6" }),
      decision({ actor: "requester", decision: "changes_requested" }),
      decision({ actor: "commenter", decision: "commented" }),
      decision({ actor: "dismissed", decision: "dismissed" }),
      decision({ actor: "unknown", decision: "toString" }),
    ],
  });
  const html = renderReview(r, { ...initialUI(), view: "history" });
  for (const text of [
    "alex approved",
    "Previous plan",
    "requester requested changes",
    "commenter commented",
    "dismissed had a review dismissed",
    "unknown has an unknown decision",
  ])
    assert.ok(html.includes(text), text);
  assert.equal(
    decisionContext(r, decision({ commitSha: "old" })),
    "Previous commit",
  );
});
test("accepted risk never becomes plan approval or compliance", () => {
  const v = {
    id: "v1",
    policyId: "data-protection",
    policyVersion: "1",
    resourceId: "c3",
    title: "Replacement",
    severity: "critical",
    blocking: true,
    acceptanceAllowed: true,
    explanation: "Replace database",
    consequence: "Data loss",
    unknown: "Recovery readiness",
    evidence: ["Plan evidence"],
  };
  const r = review();
  r.policy.violations = [v];
  r.policy.results = [
    {
      id: v.policyId,
      version: "1",
      title: "Protect data",
      outcome: "violated",
    },
  ];
  r.acceptances = [
    {
      id: "a1",
      violationId: "v1",
      planSetId: "plan-7",
      evaluationId: "eval-7",
      policyVersion: "1",
      status: "granted",
      authorizedBy: "owner",
      reason: "Migration",
      evidence: "Recovery drill",
      expiresAt: "2026-09-26T13:00:00Z",
    },
  ];
  const html = renderReview(r, { ...initialUI(), view: "policies" });
  assert.match(html, /Violation accepted/);
  assert.match(html, /Violated/);
  assert.equal(currentApproval(r), false);
  assert.equal(
    currentAcceptance({ ...r, plan: { ...r.plan, id: "plan-8" } }, v),
    undefined,
  );
  r.acceptances[0].status = "expired";
  const expired = renderReview(r, { ...initialUI(), view: "policies" });
  assert.match(expired, /Request renewed acceptance/);
  assert.doesNotMatch(expired, /Violation accepted|Violation expired/);
  assert.match(expired, /Acceptance expired; apply is blocked/);
});
test("selection, filters and inspector evidence remain connected", () => {
  const r = review();
  r.changes.push({
    ...r.changes[0],
    id: "c2",
    rootId: "network",
    name: "Network rule",
    address: "network.rule",
    risk: "high",
    planText: "network-only evidence",
  });
  const state = { ...initialUI(), root: "network", detail: "plan" };
  assert.deepEqual(
    visibleChanges(r, state).map((c) => c.id),
    ["c2"],
  );
  const html = renderReview(r, state);
  assert.match(html, /<h2>Network rule<\/h2>/);
  assert.match(html, /network-only evidence/);
  assert.deepEqual(
    r.changes.map((c) => c.id),
    ["c3", "c2"],
  );
  assert.equal(visibleChanges(r, { ...state, query: "not present" }).length, 0);
});
test("incomplete and stale evidence stay visibly unready", () => {
  const r = review({
    state: "incomplete",
    roots: [
      {
        id: "api",
        name: "api",
        status: "failed",
        applyStatus: "not_started",
        log: "failed",
      },
    ],
  });
  const html = renderReview(r);
  assert.match(html, /0\/1 roots planned/);
  assert.match(html, /proposal is incomplete/);
  assert.doesNotMatch(html, /data-action="approve"(?![^>]*disabled)/);
  r.state = "stale";
  r.headSha = "new-head";
  assert.match(renderReview(r), /Earlier approval does not cover this change/);
});
test("no plan does not display speculative resource changes as evidence", () => {
  const html = renderReview(
    review({ plan: null, policy: null, state: "unplanned" }),
  );
  assert.match(html, /Start with evidence/);
  assert.doesNotMatch(html, /name="Selected resource"/);
  assert.doesNotMatch(html, /<code>database.main/);
});
test("provider text, comments, policy text and evidence are escaped in every view", () => {
  const payload = '<img src=x onerror="alert(1)">';
  const r = review({
    title: payload,
    decisions: [
      decision({ actor: payload, message: payload, decision: payload }),
    ],
    history: [{ title: payload, detail: payload, createdAt: "" }],
  });
  r.changes[0].name = payload;
  r.changes[0].planText = payload;
  r.changes[0].sourceText = payload;
  r.policy.violations = [
    {
      id: "v",
      policyId: "p",
      policyVersion: "1",
      resourceId: "c3",
      title: payload,
      explanation: payload,
      consequence: payload,
      unknown: payload,
      evidence: [payload],
    },
  ];
  for (const view of [
    "overview",
    "changes",
    "policies",
    "execution",
    "history",
  ]) {
    const html = renderReview(r, { ...initialUI(), view });
    assert.ok(!html.includes("<img"));
    assert.ok(html.includes("&lt;img"));
  }
});
