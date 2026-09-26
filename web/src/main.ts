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

function render(r:Review){
  const attention=r.changes.filter(c=>c.risk==="critical"||c.risk==="high").length;
  document.querySelector<HTMLDivElement>("#app")!.innerHTML=`
    <header><div><strong>STATECRAFT</strong><span>Infrastructure change, understood.</span></div><code>${r.repository} · PR #${r.pullRequest}</code></header>
    <section class="hero"><div><p class="eyebrow">${r.state.replaceAll("_"," ")}</p><h1>${r.title}</h1><p>${r.headSha} · ${r.roots.length}/${r.roots.length} roots planned</p></div><button>Approve plan</button></section>
    <section class="metrics"><article><b>${r.changes.length}</b><span>Changes</span></article><article><b>${attention}</b><span>Require attention</span></article><article><b>${r.findings.length}</b><span>Findings</span></article><article><b>${r.decisions.length}</b><span>Approvals</span></article></section>
    <section class="grid"><article class="panel"><h2>Changed resources</h2><div class="changes">${r.changes.map(c=>`<button class="change" data-id="${c.id}"><span><code>${c.address}</code><small>${c.summary}</small></span><em class="${c.risk}">${c.risk}</em></button>`).join("")}</div></article>
    <article class="panel"><h2>Findings</h2>${r.findings.map(f=>`<div class="finding"><em class="${f.severity}">${f.severity}</em><strong>${f.title}</strong><code>${f.resourceAddress}</code></div>`).join("")}<h2>Approvals</h2>${r.decisions.map(d=>`<div class="decision">✓ <strong>${d.actor}</strong> approved <code>${d.planSetId}</code></div>`).join("")}</article></section>
    <section class="panel roots"><h2>Roots</h2>${r.roots.map(x=>`<div><code>${x.name}</code><span>✓ ${x.status}</span></div>`).join("")}</section>`;
}

load().then(render).catch(e=>{document.querySelector("#app")!.textContent="Failed to load review: "+e.message});
