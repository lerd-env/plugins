// Renders the catalog from index.json, the same file lerd installs from.
const grid = document.getElementById('grid');
const count = document.getElementById('count');
const q = document.getElementById('q');
const cats = document.getElementById('categories');
const pager = document.getElementById('pager');
const pluginBody = document.getElementById('plugin-body');
const sortSel = document.getElementById('sort');

// Each order sorts newest first on its own date, falling back to the name so
// listings approved the same day stay in a stable order.
const SORTS = {
  updated: (a, b) => (b.updated_at ?? '').localeCompare(a.updated_at ?? '') || a.name.localeCompare(b.name),
  stars: (a, b) => (b.stars ?? 0) - (a.stars ?? 0) || a.name.localeCompare(b.name),
  added: (a, b) => (b.added_at ?? '').localeCompare(a.added_at ?? '') || a.name.localeCompare(b.name),
  name: (a, b) => a.name.localeCompare(b.name)
};
try {
  const saved = localStorage.getItem('lerd-plugins-sort');
  if (saved in SORTS) sortSel.value = saved;
} catch {
  // Storage can be blocked; the default order still works.
}
const fmtDate = (iso) => (iso ? new Date(iso + 'T00:00:00').toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' }) : '');

// A plugin with no screenshot shows the piece Lerd's dashboard marks plugins with.
const PUZZLE = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round"><path d="M14.25 6.09c0-.36.19-.68.4-.96.22-.29.35-.63.35-1 0-1.04-1-1.88-2.25-1.88S10.5 3.09 10.5 4.13c0 .37.13.71.35 1 .21.28.4.6.4.96a.64.64 0 01-.66.64 48.4 48.4 0 01-4.16-.3c.19 1.61.3 3.25.32 4.91a.66.66 0 01-.66.66c-.36 0-.68-.19-.96-.4a1.65 1.65 0 00-1-.35c-1.04 0-1.88 1.01-1.88 2.25s.84 2.25 1.88 2.25c.37 0 .71-.13 1-.35.28-.21.6-.4.96-.4.31 0 .55.26.53.57a48 48 0 01-.64 5.06c1.52.19 3.06.31 4.62.35a.64.64 0 00.66-.64c0-.36-.19-.68-.4-.96a1.65 1.65 0 01-.35-1c0-1.04 1.01-1.88 2.25-1.88s2.25.84 2.25 1.88c0 .37-.13.71-.35 1-.21.28-.4.6-.4.96 0 .33.28.6.61.58a48.1 48.1 0 005.43-.63 48 48 0 00.58-4.72.53.53 0 00-.53-.57c-.36 0-.68.19-.96.4-.29.22-.63.35-1 .35-1.04 0-1.88-1.01-1.88-2.25s.84-2.25 1.88-2.25c.37 0 .71.13 1 .35.28.21.6.4.96.4a.66.66 0 00.66-.66 48.4 48.4 0 00-.37-5.36c-1.89.34-3.81.57-5.77.69a.58.58 0 01-.61-.58z"/></svg>';

const STAR = '<svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M12 2.5l2.9 6.1 6.6.8-4.9 4.6 1.3 6.6L12 17.3l-5.9 3.3 1.3-6.6-4.9-4.6 6.6-.8z"/></svg>';

let plugins = [];
let category = '';
// A page of the list, as many cards as the dashboard's Browse shows at once.
const PAGE_SIZE = 12;
let page = 0;

const esc = (s) => String(s ?? '').replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c]);
const short = (sha) => sha.slice(0, 7);
const commitURL = (p) => (p.repo.startsWith('https://github.com/') ? `${p.repo}/tree/${p.commit}` : p.repo);
// The author's GitHub profile, from whoever owns the plugin's repository: the
// author field is free text, the repository's owner is always a real account.
const profileURL = (p) => {
  const m = /^https:\/\/github\.com\/([^/]+)\//.exec(p.repo + '/');
  return m ? `https://github.com/${m[1]}` : null;
};
// Each plugin's own page, the address a shared link carries.
const pageURL = (p) => `${location.origin}/${encodeURIComponent(p.name)}/`;

function preview(p) {
  return p.images?.length
    ? `<div class="shot"><img src="${esc(p.images[0])}" alt="" loading="lazy"></div>`
    : `<div class="shot none">${PUZZLE}</div>`;
}

const pad = (n) => String(n).padStart(2, '0');

// Lerd runs inside WSL on Windows, so a plugin for linux runs there too.
function runsOn(p) {
  const on = p.platforms?.length ? p.platforms : ['linux', 'macos'];
  return [on.includes('linux') && 'Linux', on.includes('macos') && 'macOS', on.includes('linux') && 'Windows (WSL)'].filter(Boolean).join(' · ');
}

function matches(p, term) {
  if (category && p.category !== category) return false;
  if (!term) return true;
  return [p.name, p.description, p.author, ...(p.tags ?? [])].join(' ').toLowerCase().includes(term);
}

function render() {
  const term = q.value.trim().toLowerCase();
  const shown = plugins.filter((p) => matches(p, term)).sort(SORTS[sortSel.value] ?? SORTS.updated);
  const dateLabel = sortSel.value === 'added' ? 'Added' : 'Updated';
  const dateOf = (p) => (sortSel.value === 'added' ? p.added_at : p.updated_at);
  const pages = Math.max(1, Math.ceil(shown.length / PAGE_SIZE));
  page = Math.min(page, pages - 1);
  const first = page * PAGE_SIZE;
  count.innerHTML = `<b>${pad(shown.length)}</b> / ${plugins.length} plugins`;
  grid.innerHTML = shown.length
    ? shown
        .slice(first, first + PAGE_SIZE)
        .map(
          (p, i) => `
      <a class="card" href="${encodeURIComponent(p.name)}/">
        ${preview(p)}
        <div class="card-body">
          <div class="card-top"><span class="num">${pad(first + i + 1)}</span><span>${esc(p.author)}</span></div>
          <h3>${esc(p.name)}${version(p)}</h3>
          <p>${esc(p.description)}</p>
          <span class="runs">${esc(runsOn(p))}</span>
          <div class="badges">${(p.tags ?? []).map((t) => `<span class="badge">${esc(t)}</span>`).join('')}</div>
          <div class="foot-row">
            ${dateOf(p) ? `<span>${dateLabel} ${esc(fmtDate(dateOf(p)))}</span>` : '<span></span>'}
            ${p.stars != null ? `<span class="stars" title="GitHub stars">${STAR} ${p.stars.toLocaleString()}</span>` : ''}
          </div>
        </div>
      </a>`
        )
        .join('')
    : plugins.length
      ? '<p class="empty">No plugin matches that.</p>'
      : '<p class="empty">No plugins are listed yet. <a href="#_make">Build the first one</a>.</p>';
  renderPager(pages);
}

function renderPager(pages) {
  if (pages < 2) {
    pager.innerHTML = '';
    return;
  }
  const btn = (n, label, extra = '') => `<button type="button" data-page="${n}" ${extra}>${label}</button>`;
  pager.innerHTML = [
    btn(page - 1, '←', `aria-label="Previous page" ${page === 0 ? 'disabled' : ''}`),
    ...Array.from({ length: pages }, (_, n) => btn(n, pad(n + 1), n === page ? 'aria-current="page"' : '')),
    btn(page + 1, '→', `aria-label="Next page" ${page === pages - 1 ? 'disabled' : ''}`),
  ].join('');
}

// The approved commit's own version, beside the name.
const version = (p) => (p.version ? ` <span class="ver">v${esc(p.version)}</span>` : '');

function renderCategories() {
  const all = [...new Set(plugins.map((p) => p.category).filter(Boolean))].sort();
  cats.innerHTML = ['', ...all]
    .map((c) => `<button class="chip" data-cat="${esc(c)}" aria-pressed="${c === category}">${c ? esc(c) : 'All'}</button>`)
    .join('');
}

function detail(p) {
  const cmd = `lerd plugin install ${p.name}`;
  const profile = profileURL(p);
  const author = profile ? `<a href="${esc(profile)}">${esc(p.author || profile.split('/').pop())}</a>` : esc(p.author);
  return `
    <a class="back" href="./">← All plugins</a>
    <p class="label">${p.category ? `<b>${esc(p.category)}</b> / ` : ''}<span class="by">by ${author}</span></p>
    <h2 id="d-name">${esc(p.name)}${version(p)}</h2>
    <p>${esc(p.description)}</p>
    <div class="facts">
      <span>Runs on ${esc(runsOn(p))}</span>
      ${p.updated_at ? `<span>Updated ${esc(fmtDate(p.updated_at))}</span>` : ''}
      ${p.stars != null ? `<span>${p.stars.toLocaleString()} stars</span>` : ''}
      <span>Reviewed at <a href="${esc(commitURL(p))}">${short(p.commit)}</a></span>
      <a href="${esc(p.repo)}">Repository</a>
      <button class="copy" type="button" data-share="${esc(pageURL(p))}" data-title="${esc(p.name)}">Share</button>
    </div>
    ${p.requires?.length ? `<div class="reqs"><span>Requires</span><div class="badges">${p.requires.map((r) => `<span class="badge">${esc(r)}</span>`).join('')}</div></div>` : ''}
    ${p.images?.length ? `<div class="shots">${p.images.map((i) => `<img src="${esc(i)}" alt="" loading="lazy">`).join('')}</div>` : ''}
    <div class="install">
      <div class="cmd"><i>$</i><code>${esc(cmd)}</code><button class="copy" data-copy="${esc(cmd)}">Copy</button></div>
      <a class="btn ink" href="http://lerd.localhost/#system/plugins/browse/${encodeURIComponent(p.name)}">Install in Lerd</a>
    </div>
    <p class="note">Lerd installs commit <code>${short(p.commit)}</code> and nothing newer, even if the repository has moved on.</p>`;
}

// showPlugin turns the page into the plugin's own: its detail in place of the list.
function showPlugin(name) {
  const p = plugins.find((x) => x.name === name);
  document.body.classList.add('on-plugin');
  pluginBody.innerHTML = p
    ? detail(p)
    : `<a class="back" href="./">← All plugins</a><p class="empty">No plugin named ${esc(name)} is listed.</p>`;
  if (p) document.title = `${p.name} · Lerd Plugins`;
  window.scrollTo(0, 0);
}

pager.addEventListener('click', (e) => {
  const btn = e.target.closest('[data-page]');
  if (!btn || btn.disabled) return;
  page = Number(btn.dataset.page);
  render();
  document.querySelector('.list').scrollIntoView({ behavior: 'smooth', block: 'start' });
});
cats.addEventListener('click', (e) => {
  const chip = e.target.closest('.chip');
  if (!chip) return;
  category = chip.dataset.cat;
  page = 0;
  renderCategories();
  render();
});
// Share hands the plugin's own page to the system's share sheet where there is
// one, and copies it everywhere else.
document.addEventListener('click', async (e) => {
  const btn = e.target.closest('[data-share]');
  if (!btn) return;
  const url = btn.dataset.share;
  if (navigator.share) {
    try {
      await navigator.share({ title: `${btn.dataset.title} · Lerd Plugins`, url });
      return;
    } catch (err) {
      if (err?.name === 'AbortError') return;
    }
  }
  try {
    await navigator.clipboard.writeText(url);
    btn.textContent = 'Link copied';
  } catch {
    btn.textContent = 'Not copied';
  }
  setTimeout(() => (btn.textContent = 'Share'), 1500);
});
document.addEventListener('click', async (e) => {
  const btn = e.target.closest('[data-copy]');
  if (!btn) return;
  // A browser can refuse the clipboard; say so rather than claim it copied.
  try {
    await navigator.clipboard.writeText(btn.dataset.copy);
    btn.textContent = 'Copied';
  } catch {
    btn.textContent = 'Not copied';
  }
  setTimeout(() => (btn.textContent = 'Copy'), 1500);
});
q.addEventListener('input', () => {
  page = 0;
  render();
});
sortSel.addEventListener('change', () => {
  try {
    localStorage.setItem('lerd-plugins-sort', sortSel.value);
  } catch {
    // Not remembered, but still applied.
  }
  page = 0;
  render();
});

fetch('index.json')
  .then((r) => r.json())
  .then((c) => {
    plugins = c.plugins ?? [];
    // Lerd's own star count, baked into the index at each deploy, beside GitHub.
    if (c.lerd_stars) {
      const s = document.getElementById('lerd-stars');
      s.textContent = '★ ' + (c.lerd_stars >= 1000 ? (c.lerd_stars / 1000).toFixed(1).replace(/\.0$/, '') + 'k' : c.lerd_stars);
      s.hidden = false;
    }
    // An empty catalog shows only the call to submit, not zero counts and filters.
    document.body.classList.toggle('none-listed', plugins.length === 0);
    document.getElementById('n-plugins').textContent = plugins.length;
    document.getElementById('n-authors').textContent = new Set(plugins.map((p) => p.author)).size;
    renderCategories();
    render();
    // A plugin's page names it; an old #name link moves to that page.
    const named = document.querySelector('meta[name="lerd-plugin"]')?.content;
    if (named) showPlugin(named);
    else if (location.hash) {
      const name = decodeURIComponent(location.hash.slice(1));
      if (plugins.some((p) => p.name === name)) location.replace(`${encodeURIComponent(name)}/`);
    }
  })
  .catch(() => (grid.innerHTML = '<p class="empty">The catalog could not be loaded.</p>'));

// The theme switch: a click remembers the choice; until there is one, the page
// follows the system as it changes.
(() => {
  const root = document.documentElement;
  const button = document.getElementById('theme');
  const label = () => button.setAttribute('aria-label', root.classList.contains('dark') ? 'Switch to light theme' : 'Switch to dark theme');
  label();
  button.addEventListener('click', () => {
    const dark = !root.classList.contains('dark');
    root.classList.toggle('dark', dark);
    try { localStorage.setItem('lerd-theme', dark ? 'dark' : 'light'); } catch {}
    label();
  });
  matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
    let saved = null;
    try { saved = localStorage.getItem('lerd-theme'); } catch {}
    if (!saved) { root.classList.toggle('dark', e.matches); label(); }
  });
})();
