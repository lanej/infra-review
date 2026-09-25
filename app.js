const [state, plan] = await Promise.all([
  fetch('./fixtures/state.json').then(r => r.json()),
  fetch('./fixtures/plan.json').then(r => r.json())
]);

const stateResources = state.values.root_module.resources || [];
const stateByAddress = new Map(stateResources.map(r => [r.address, r]));

const changes = (plan.resource_changes || []).map((c, id) => {
  const actions = c.change.actions || [];
  const rawAction = actions.includes('delete') && actions.includes('create') ? 'replace' : (actions[0] || 'no-op');
  const action = rawAction === 'update' ? 'modify' : rawAction;
  const before = c.change.before || {};
  const after = c.change.after || {};
  const changed = [...new Set([...Object.keys(before), ...Object.keys(after)])]
    .filter(k => JSON.stringify(before[k]) !== JSON.stringify(after[k]));
  const risk = action === 'replace' ? 'critical'
    : /security_rule|firewall|iam|role/.test(c.type) ? 'high'
    : action === 'delete' ? 'high'
    : changed.some(k => /capacity|node_count|size|sku|storage/.test(k)) ? 'medium'
    : 'low';
  return { ...c, id, action, before, after, changed, risk };
});

const refs = [];
for (const r of stateResources) {
  for (const dep of (r.depends_on || [])) refs.push([dep, r.address]);
}
for (const r of plan.configuration?.root_module?.resources || []) {
  for (const expr of Object.values(r.expressions || {})) {
    for (const ref of expr.references || []) {
      const target = ref.split('.').slice(0,2).join('.');
      refs.push([target, r.address]);
    }
  }
}

const countBy = (items, key) => items.reduce((m, x) => ((m[x[key]] = (m[x[key]] || 0) + 1), m), {});
const actionCounts = countBy(changes, 'action');

document.querySelector('#resource-count').textContent = changes.length;
document.querySelector('#attention-count').textContent = changes.filter(c => ['critical','high'].includes(c.risk)).length;
document.querySelector('#tab-count').textContent = changes.length;
document.querySelector('#table-count').textContent = changes.length;
document.querySelector('#action-summary').textContent =
  ['modify','create','delete','replace'].map(a => String(actionCounts[a] || 0) + ' ' + a).join(' · ');
document.querySelector('#attention-summary').textContent =
  changes.filter(c => ['critical','high'].includes(c.risk)).map(c => c.type.replace('azurerm_','')).join(' · ');

const graph = document.querySelector('#graph');
const graphNodes = changes.slice(0, 9);
const positions = [[42,7],[20,28],[43,28],[68,28],[20,55],[45,55],[69,55],[31,80],[59,80]];

graphNodes.forEach((c, i) => {
  const el = document.createElement('div');
  el.className = 'node ' + c.action;
  el.dataset.id = c.id;
  el.style.left = positions[i][0] + '%';
  el.style.top = positions[i][1] + '%';
  el.innerHTML = '<strong>' + shortName(c.address) + '</strong><small>' + c.action + ' · ' + c.changed.length + ' fields</small>';
  el.addEventListener('click', () => select(c.id));
  graph.appendChild(el);
});

requestAnimationFrame(drawEdges);

function drawEdges() {
  graph.querySelectorAll('.edge').forEach(e => e.remove());
  const nodes = [...graph.querySelectorAll('.node')];
  for (const [fromAddr, toAddr] of refs) {
    const from = nodes.find(n => changes[Number(n.dataset.id)]?.address.startsWith(fromAddr));
    const to = nodes.find(n => changes[Number(n.dataset.id)]?.address.startsWith(toAddr));
    if (!from || !to) continue;
    const fr = from.getBoundingClientRect(), tr = to.getBoundingClientRect(), gr = graph.getBoundingClientRect();
    const x1 = fr.left - gr.left + fr.width/2, y1 = fr.top - gr.top + fr.height/2;
    const x2 = tr.left - gr.left + tr.width/2, y2 = tr.top - gr.top + tr.height/2;
    const edge = document.createElement('div');
    edge.className = 'edge';
    const length = Math.hypot(x2-x1,y2-y1);
    const angle = Math.atan2(y2-y1,x2-x1) * 180 / Math.PI;
    edge.style.left = x1 + 'px'; edge.style.top = y1 + 'px'; edge.style.width = length + 'px';
    edge.style.transform = 'rotate(' + angle + 'deg)';
    graph.prepend(edge);
  }
}

window.addEventListener('resize', drawEdges);

const tbody = document.querySelector('#changes');
for (const c of changes) {
  const tr = document.createElement('tr');
  tr.dataset.id = c.id;
  tr.dataset.viewruleKey = c.address;
  tr.innerHTML =
    '<td data-viewrule="cell"><code data-viewrule="label">' + c.address + '</code></td>' +
    '<td data-viewrule="cell">' + c.type + '</td>' +
    '<td data-viewrule="cell"><span class="action ' + c.action + '">' + c.action + '</span></td>' +
    '<td data-viewrule="cell">' + summary(c) + '</td>' +
    '<td data-viewrule="cell"><span class="risk ' + c.risk + '">' + c.risk + '</span></td>';
  tr.addEventListener('click', () => select(c.id));
  tbody.appendChild(tr);
}

function select(id) {
  const c = changes[id];
  document.querySelectorAll('[data-id]').forEach(el =>
    el.classList.toggle('selected', Number(el.dataset.id) === id));
  document.querySelector('#detail-position').textContent = (id+1) + ' of ' + changes.length;
  const keys = c.changed.slice(0, 7);
  document.querySelector('#details').innerHTML =
    '<div class="resource-title"><div><strong>' + c.type + '</strong><br><code>' + c.address + '</code></div>' +
    '<span class="action ' + c.action + '">' + c.action + '</span></div>' +
    '<p>' + summary(c) + '</p>' +
    (c.risk === 'critical' ? '<div class="alert"><strong>This resource will be destroyed and recreated.</strong><br>Verify migration, backup strategy, and expected downtime.</div>' : '') +
    '<div class="kv-grid"><div class="kv"><h3>Before</h3>' + keys.map(k => row(k,c.before[k],false)).join('') + '</div>' +
    '<div class="kv"><h3>After</h3>' + keys.map(k => row(k,c.after[k],true)).join('') + '</div></div>' +
    '<p class="micro">Evidence: mock state + plan · commit abc1234 · project api/production</p>';
}

function row(k, v, changed) {
  return '<div class="row"><span>' + k + '</span><span class="' + (changed ? 'changed' : '') + '">' + fmt(v) + '</span></div>';
}
function fmt(v) {
  if (v === null || v === undefined) return '—';
  return typeof v === 'object' ? JSON.stringify(v) : String(v);
}
function shortName(address) { return address.split('.').at(-1).replace(/_/g,' '); }
function summary(c) {
  if (c.action === 'replace') return 'Replacement required by configuration changes';
  if (c.type.includes('security_rule')) return 'Network access rules changed';
  if (c.changed.includes('node_count')) return 'Node count ' + c.before.node_count + ' → ' + c.after.node_count;
  if (c.changed.includes('storage_mb')) return 'Storage capacity changed';
  if (c.action === 'create') return 'New managed resource';
  if (c.action === 'delete') return 'Resource removed';
  return c.changed.length + ' attributes changed';
}

select(changes.find(c => c.action === 'replace')?.id ?? 0);