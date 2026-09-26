export type Review = {
  id:string; repository:string; pullRequest:number; title:string; headSha:string; state:string;
  roots:{id:string;name:string;status:string}[];
  changes:{id:string;rootId:string;address:string;resourceType:string;action:string;risk:string;summary:string}[];
  findings:{id:string;severity:string;category:string;title:string;resourceAddress:string;blocking:boolean}[];
  decisions:{actor:string;decision:string;planSetId:string;commitSha:string;createdAt:string}[];
};

type Decision = Review["decisions"][number];

function escapeHTML(value:string|number):string {
  const entities:Record<string,string>={"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;","'":"&#39;"};
  return String(value).replace(/[&<>"']/g,character=>entities[character]);
}

function decisionContext(review:Review, decision:Decision):string {
  if (review.state === "stale") return "Stale evidence";
  if (!decision.commitSha || !review.headSha) return "Commit not recorded";
  if (decision.commitSha !== review.headSha) return "Previous commit";
  if (!decision.planSetId) return "Plan not recorded";
  return "";
}

function isCurrentApproval(review:Review, decision:Decision):boolean {
  return decision.decision === "approved" && decisionContext(review, decision) === "";
}

function renderDecision(review:Review, decision:Decision):string {
  const labels:Record<string,string>={
    approved:"approved",
    changes_requested:"requested changes on",
    commented:"commented on",
    dismissed:"had a review dismissed for",
    pending:"has a pending review for"
  };
  const label = Object.hasOwn(labels, decision.decision) ? labels[decision.decision] : "has an unknown decision for";
  const note = decisionContext(review, decision);
  const approved = isCurrentApproval(review, decision);
  const tone = approved ? "approved" : decision.decision === "changes_requested" && !note ? "changes-requested" : "muted";
  const icon = approved ? "✓" : decision.decision === "changes_requested" ? "!" : "•";
  return `<div class="decision decision--${tone}">${icon} <strong>${escapeHTML(decision.actor)}</strong> ${label} <code>${escapeHTML(decision.planSetId)}</code>${note ? ` <small>(${note})</small>` : ""}</div>`;
}

export function renderReview(r:Review):string {
  const attention=r.changes.filter(c=>c.risk==="critical"||c.risk==="high").length;
  const planned=r.roots.filter(root=>root.status==="planned").length;
  const approvals=r.decisions.filter(decision=>isCurrentApproval(r, decision)).length;
  const stale=r.state==="stale";
  return `
    <header><div><strong>STATECRAFT</strong><span>Infrastructure change, understood.</span></div><code>${escapeHTML(r.repository)} · PR #${escapeHTML(r.pullRequest)}</code></header>
    <section class="hero"><div><p class="eyebrow">${escapeHTML(r.state.replaceAll("_"," "))}</p><h1>${escapeHTML(r.title)}</h1><p>${escapeHTML(r.headSha)} · ${planned}/${r.roots.length} roots planned</p></div><button${stale ? " disabled" : ""}>Approve plan</button></section>
    ${stale ? '<p class="stale-notice" role="status">This review contains evidence from a previous commit. Re-plan before reviewing or approving.</p>' : ""}
    <section class="metrics"><article><b>${r.changes.length}</b><span>Changes</span></article><article><b>${attention}</b><span>Require attention</span></article><article><b>${r.findings.length}</b><span>Findings</span></article><article><b>${approvals}</b><span>Approvals on this commit</span></article></section>
    <section class="grid"><article class="panel"><h2>Changed resources</h2><div class="changes">${r.changes.map(c=>`<button class="change" data-id="${escapeHTML(c.id)}"><span><code>${escapeHTML(c.address)}</code><small>${escapeHTML(c.summary)}</small></span><em class="${escapeHTML(c.risk)}">${escapeHTML(c.risk)}</em></button>`).join("")}</div></article>
    <article class="panel"><h2>Findings</h2>${r.findings.map(f=>`<div class="finding"><em class="${escapeHTML(f.severity)}">${escapeHTML(f.severity)}</em><strong>${escapeHTML(f.title)}</strong><code>${escapeHTML(f.resourceAddress)}</code></div>`).join("")}<h2>Review decisions</h2>${r.decisions.map(decision=>renderDecision(r, decision)).join("")}</article></section>
    <section class="panel roots"><h2>Roots</h2>${r.roots.map(x=>`<div><code>${escapeHTML(x.name)}</code><span class="${x.status === "planned" ? "planned" : "unplanned"}">${x.status === "planned" ? "✓" : "•"} ${escapeHTML(x.status)}</span></div>`).join("")}</section>`;
}
