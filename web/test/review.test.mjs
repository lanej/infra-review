import assert from "node:assert/strict";
import test from "node:test";
import { renderReview } from "../src/review.ts";

function decision(overrides = {}) {
  return {actor:"reviewer", decision:"approved", planSetId:"plan-7", commitSha:"head", createdAt:"2026-09-25T20:10:00Z", ...overrides};
}

function review(overrides = {}) {
  return {id:"review-42", repository:"acme/infra", pullRequest:42, title:"Change API", headSha:"head", state:"ready_for_review", roots:[], changes:[], findings:[], decisions:[], ...overrides};
}

function decisionCards(html) {
  return html.match(/<div class="decision [^"]+">.*?<\/div>/g) ?? [];
}

function approvalCount(html) {
  return Number(html.match(/<b>(\d+)<\/b><span>Approvals on this commit<\/span>/)?.[1]);
}

for (const [state, label, count] of [
  ["approved", "approved", 1],
  ["changes_requested", "requested changes on", 0],
  ["commented", "commented on", 0],
  ["dismissed", "had a review dismissed for", 0],
  ["pending", "has a pending review for", 0],
  ["unknown", "has an unknown decision for", 0],
  ["toString", "has an unknown decision for", 0],
]) {
  test(`renders ${state} without turning it into an approval`, () => {
    const html = renderReview(review({decisions:[decision({decision:state})]}));
    const [card] = decisionCards(html);
    assert.ok(card.includes(label));
    assert.equal(approvalCount(html), count);
    assert.equal(card.includes("✓"), state === "approved");
    if (state !== "approved") assert.ok(!card.includes(" approved "));
  });
}

test("keeps previous and unbound decisions visible without counting them as current approvals", () => {
  const html = renderReview(review({decisions:[
    decision(),
    decision({actor:"old-reviewer", commitSha:"previous-head"}),
    decision({actor:"unbound-reviewer", commitSha:""}),
    decision({actor:"no-plan", planSetId:""}),
    decision({actor:"requester", decision:"changes_requested"}),
  ]}));
  const cards = decisionCards(html);
  assert.equal(cards.length, 5);
  assert.equal(approvalCount(html), 1);
  assert.match(cards[1], /Previous commit/);
  assert.match(cards[2], /Commit not recorded/);
  assert.match(cards[3], /Plan not recorded/);
  assert.ok(cards.slice(1).every(card => !card.includes("✓")));
});

test("a stale review shows retained evidence without readiness or approvals", () => {
  const html = renderReview(review({
    headSha:"new-head", state:"stale",
    roots:[{id:"api",name:"prod/api",status:"stale"},{id:"db",name:"prod/db",status:"stale"}],
    // Even a same-commit record cannot make explicitly stale evidence ready.
    decisions:[decision({commitSha:"new-head"})],
  }));
  assert.match(html, /0\/2 roots planned/);
  assert.match(html, /<button disabled>Approve plan<\/button>/);
  assert.match(html, /Re-plan before reviewing or approving/);
  assert.match(html, /Stale evidence/);
  assert.equal(approvalCount(html), 0);
  assert.ok(!html.includes("✓"));
});

test("root completion counts only planned roots", () => {
  const html = renderReview(review({roots:[
    {id:"api",name:"prod/api",status:"planned"},
    {id:"db",name:"prod/db",status:"failed"},
  ]}));
  assert.match(html, /1\/2 roots planned/);
  assert.match(html, /✓ planned/);
  assert.match(html, /• failed/);
});

test("decision labels and source text remain escaped", () => {
  const payload = '<img src=x onerror="alert(1)">';
  const html = renderReview(review({title:payload, decisions:[decision({actor:payload,planSetId:payload,decision:payload})]}));
  assert.ok(!html.includes("<img"));
  assert.match(html, /&lt;img src=x onerror=&quot;alert\(1\)&quot;&gt;/);
  assert.match(decisionCards(html)[0], /has an unknown decision for/);
  assert.equal(approvalCount(html), 0);
});
