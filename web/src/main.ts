import "./style.css";

type Review = {
  id:string; repository:string; pullRequest:number; title:string; headSha:string; state:string;
  roots:{id:string;name:string;status:string}[];
  changes:{id:string;rootId:string;address:string;resourceType:string;action:string;risk:string;summary:string}[];
  findings:{id:string;severity:string;category:string;title:string;resourceAddress:string;blocking:boolean}[];
  decisions:{actor:string;decision:string;planSetId:string;createdAt:string}[];
};

async function load():Promise<Review>{
  const response=await fetch("/api/reviews/pr-1842");
  if(!response.ok) throw new Error(await response.text());
  return response.json();
}

function escapeHTML(value:string|number):string{
  const entities:Record<string,string>={"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;","'":"&#39;"};
  return String(value).replace(/[&<>"']/g,character=>entities[character]);
}

function render(r:Review){
  const attention=r.changes.filter(c=>c.risk==="critical"||c.risk==="high").length;
  document.querySelector<HTMLDivElement>("#app")!.innerHTML=`
    <header><div><strong>STATECRAFT</strong><span>Infrastructure change, understood.</span></div><code>${escapeHTML(r.repository)} · PR #${escapeHTML(r.pullRequest)}</code></header>
    <section class="hero"><div><p class="eyebrow">${escapeHTML(r.state.replaceAll("_"," "))}</p><h1>${escapeHTML(r.title)}</h1><p>${escapeHTML(r.headSha)} · ${escapeHTML(r.roots.length)}/${escapeHTML(r.roots.length)} roots planned</p></div><button>Approve plan</button></section>
    <section class="metrics"><article><b>${escapeHTML(r.changes.length)}</b><span>Changes</span></article><article><b>${escapeHTML(attention)}</b><span>Require attention</span></article><article><b>${escapeHTML(r.findings.length)}</b><span>Findings</span></article><article><b>${escapeHTML(r.decisions.length)}</b><span>Approvals</span></article></section>
    <section class="grid"><article class="panel"><h2>Changed resources</h2><div class="changes">${r.changes.map(c=>`<button class="change" data-id="${escapeHTML(c.id)}"><span><code>${escapeHTML(c.address)}</code><small>${escapeHTML(c.summary)}</small></span><em class="${escapeHTML(c.risk)}">${escapeHTML(c.risk)}</em></button>`).join("")}</div></article>
    <article class="panel"><h2>Findings</h2>${r.findings.map(f=>`<div class="finding"><em class="${escapeHTML(f.severity)}">${escapeHTML(f.severity)}</em><strong>${escapeHTML(f.title)}</strong><code>${escapeHTML(f.resourceAddress)}</code></div>`).join("")}<h2>Approvals</h2>${r.decisions.map(d=>`<div class="decision">✓ <strong>${escapeHTML(d.actor)}</strong> approved <code>${escapeHTML(d.planSetId)}</code></div>`).join("")}</article></section>
    <section class="panel roots"><h2>Roots</h2>${r.roots.map(x=>`<div><code>${escapeHTML(x.name)}</code><span>✓ ${escapeHTML(x.status)}</span></div>`).join("")}</section>`;
}

load().then(render).catch(e=>{document.querySelector("#app")!.textContent="Failed to load review: "+e.message});
