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
// A category, tag or author page names its filter, as the slug its folder has.
const [filterKind, filterSlug] = (document.querySelector('meta[name="lerd-filter"]')?.content ?? '').split(':');
// A page of the list: two rows of three cards. Each page is its own address,
// /page/<n>/ under the list's folder, written by ci/index with the same size.
const PAGE_SIZE = 6;
const page = Number(document.querySelector('meta[name="lerd-page"]')?.content ?? 1) - 1;

const esc = (s) => String(s ?? '').replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c]);
const short = (sha) => sha.slice(0, 7);
const commitURL = (p) => (p.repo.startsWith('https://github.com/') ? `${p.repo}/tree/${p.commit}` : p.repo);
// The folder a category, tag or author page lives in, as ci/index's slug builds it.
const slug = (s) => String(s ?? '').toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '');
const filterURL = (kind, value) => `${kind}/${slug(value)}/`;
const FILTER_VALUES = { category: (p) => [p.category], tag: (p) => p.tags ?? [], author: (p) => [p.author] };
// Each plugin's own page, the address a shared link carries.
const pageURL = (p) => `${location.origin}/${encodeURIComponent(p.name)}/`;

function preview(p) {
  return p.images?.length
    ? `<div class="shot"><img src="${esc(p.images[0])}" alt="" loading="lazy"></div>`
    : `<div class="shot none">${PUZZLE}</div>`;
}

const pad = (n) => String(n).padStart(2, '0');

// Lerd runs inside WSL on Windows, so a plugin for linux runs there too. The
// marks are Simple Icons', as on lerd.sh; each names itself for screen readers.
const OS_MARKS = {
  linux: ['Linux', '<path fill="currentColor" d="M12.504 0q-.232 0-.48.021c-4.226.333-3.105 4.807-3.17 6.298c-.076 1.092-.3 1.953-1.05 3.02c-.885 1.051-2.127 2.75-2.716 4.521c-.278.832-.41 1.684-.287 2.489a.4.4 0 0 0-.11.135c-.26.268-.45.6-.663.839c-.199.199-.485.267-.797.4c-.313.136-.658.269-.864.68c-.09.189-.136.394-.132.602c0 .199.027.4.055.536c.058.399.116.728.04.97c-.249.68-.28 1.145-.106 1.484c.174.334.535.47.94.601c.81.2 1.91.135 2.774.6c.926.466 1.866.67 2.616.47c.526-.116.97-.464 1.208-.946c.587-.003 1.23-.269 2.26-.334c.699-.058 1.574.267 2.577.2c.025.134.063.198.114.333l.003.003c.391.778 1.113 1.132 1.884 1.071s1.592-.536 2.257-1.306c.631-.765 1.683-1.084 2.378-1.503c.348-.199.629-.469.649-.853c.023-.4-.2-.811-.714-1.376v-.097l-.003-.003c-.17-.2-.25-.535-.338-.926c-.085-.401-.182-.786-.492-1.046h-.003c-.059-.054-.123-.067-.188-.135a.36.36 0 0 0-.19-.064c.431-1.278.264-2.55-.173-3.694c-.533-1.41-1.465-2.638-2.175-3.483c-.796-1.005-1.576-1.957-1.56-3.368c.026-2.152.236-6.133-3.544-6.139m.529 3.405h.013c.213 0 .396.062.584.198c.19.135.33.332.438.533c.105.259.158.459.166.724c0-.02.006-.04.006-.06v.105l-.004-.021l-.004-.024a1.8 1.8 0 0 1-.15.706a.95.95 0 0 1-.213.335a1 1 0 0 0-.088-.042c-.104-.045-.198-.064-.284-.133a1.3 1.3 0 0 0-.22-.066c.05-.06.146-.133.183-.198q.08-.193.088-.402v-.02a1.2 1.2 0 0 0-.061-.4c-.045-.134-.101-.2-.183-.333c-.084-.066-.167-.132-.267-.132h-.016c-.093 0-.176.03-.262.132a.8.8 0 0 0-.205.334a1.2 1.2 0 0 0-.09.4v.019q.002.134.02.267c-.193-.067-.438-.135-.607-.202a2 2 0 0 1-.018-.2v-.02a1.8 1.8 0 0 1 .15-.768a1.08 1.08 0 0 1 .43-.533a1 1 0 0 1 .594-.2zm-2.962.059h.036c.142 0 .27.048.399.135c.146.129.264.288.344.465c.09.199.14.4.153.667v.004c.007.134.006.2-.002.266v.08c-.03.007-.056.018-.083.024c-.152.055-.274.135-.393.2q.018-.136.003-.267v-.015c-.012-.133-.04-.2-.082-.333a.6.6 0 0 0-.166-.267a.25.25 0 0 0-.183-.064h-.021c-.071.006-.13.04-.186.132a.55.55 0 0 0-.12.27a1 1 0 0 0-.023.33v.015c.012.135.037.2.08.334c.046.134.098.2.166.268q.014.014.034.024c-.07.057-.117.07-.176.136a.3.3 0 0 1-.131.068a2.6 2.6 0 0 1-.275-.402a1.8 1.8 0 0 1-.155-.667a1.8 1.8 0 0 1 .08-.668a1.4 1.4 0 0 1 .283-.535c.128-.133.26-.2.418-.2m1.37 1.706c.332 0 .733.065 1.216.399c.293.2.523.269 1.052.468h.003c.255.136.405.266.478.399v-.131a.57.57 0 0 1 .016.47c-.123.31-.516.643-1.063.842v.002c-.268.135-.501.333-.775.465c-.276.135-.588.292-1.012.267a1.1 1.1 0 0 1-.448-.067a4 4 0 0 1-.322-.198c-.195-.135-.363-.332-.612-.465v-.005h-.005c-.4-.246-.616-.512-.686-.71q-.104-.403.193-.6c.224-.135.38-.271.483-.336c.104-.074.143-.102.176-.131h.002v-.003c.169-.202.436-.47.839-.601c.139-.036.294-.065.466-.065zm2.8 2.142c.358 1.417 1.196 3.475 1.735 4.473c.286.534.855 1.659 1.102 3.024c.156-.005.33.018.513.064c.646-1.671-.546-3.467-1.089-3.966c-.22-.2-.232-.335-.123-.335c.59.534 1.365 1.572 1.646 2.757c.13.535.16 1.104.021 1.67c.067.028.135.06.205.067c1.032.534 1.413.938 1.23 1.537v-.043c-.06-.003-.12 0-.18 0h-.016c.151-.467-.182-.825-1.065-1.224c-.915-.4-1.646-.336-1.77.465c-.008.043-.013.066-.018.135c-.068.023-.139.053-.209.064c-.43.268-.662.669-.793 1.187c-.13.533-.17 1.156-.205 1.869v.003c-.02.334-.17.838-.319 1.35c-1.5 1.072-3.58 1.538-5.348.334a2.7 2.7 0 0 0-.402-.533a1.5 1.5 0 0 0-.275-.333c.182 0 .338-.03.465-.067a.62.62 0 0 0 .314-.334c.108-.267 0-.697-.345-1.163s-.931-.995-1.788-1.521c-.63-.4-.986-.87-1.15-1.396c-.165-.534-.143-1.085-.015-1.645c.245-1.07.873-2.11 1.274-2.763c.107-.065.037.135-.408.974c-.396.751-1.14 2.497-.122 3.854a8.1 8.1 0 0 1 .647-2.876c.564-1.278 1.743-3.504 1.836-5.268c.048.036.217.135.289.202c.218.133.38.333.59.465c.21.201.477.335.876.335q.058.005.11.006c.412 0 .73-.134.997-.268c.29-.134.52-.334.74-.4h.005c.467-.135.835-.402 1.044-.7zm2.185 8.958c.037.6.343 1.245.882 1.377c.588.134 1.434-.333 1.791-.765l.211-.01c.315-.007.577.01.847.268l.003.003c.208.199.305.53.391.876c.085.4.154.78.409 1.066c.486.527.645.906.636 1.14l.003-.007v.018l-.003-.012c-.015.262-.185.396-.498.595c-.63.401-1.746.712-2.457 1.57c-.618.737-1.37 1.14-2.036 1.191c-.664.053-1.237-.2-1.574-.898l-.005-.003c-.21-.4-.12-1.025.056-1.69c.176-.668.428-1.344.463-1.897c.037-.714.076-1.335.195-1.814c.12-.465.308-.797.641-.984l.045-.022zm-10.814.049h.01q.08 0 .157.014c.376.055.706.333 1.023.752l.91 1.664l.003.003c.243.533.754 1.064 1.189 1.637c.434.598.77 1.131.729 1.57v.006c-.057.744-.48 1.148-1.125 1.294c-.645.135-1.52.002-2.395-.464c-.968-.536-2.118-.469-2.857-.602q-.553-.1-.723-.4c-.11-.2-.113-.602.123-1.23v-.004l.002-.003c.117-.334.03-.752-.027-1.118c-.055-.401-.083-.71.043-.94c.16-.334.396-.4.69-.533c.294-.135.64-.202.915-.47h.002v-.002c.256-.268.445-.601.668-.838c.19-.201.38-.336.663-.336m7.159-9.074c-.435.201-.945.535-1.488.535c-.542 0-.97-.267-1.28-.466c-.154-.134-.28-.268-.373-.335c-.164-.134-.144-.333-.074-.333c.109.016.129.134.199.2c.096.066.215.2.36.333c.292.2.68.467 1.167.467c.485 0 1.053-.267 1.398-.466c.195-.135.445-.334.648-.467c.156-.136.149-.267.279-.267c.128.016.034.134-.147.332a8 8 0 0 1-.69.468zm-1.082-1.583V5.64c-.006-.02.013-.042.029-.05c.074-.043.18-.027.26.004c.063 0 .16.067.15.135c-.006.049-.085.066-.135.066c-.055 0-.092-.043-.141-.068c-.052-.018-.146-.008-.163-.065m-.551 0c-.02.058-.113.049-.166.066c-.047.025-.086.068-.14.068c-.05 0-.13-.02-.136-.068c-.01-.066.088-.133.15-.133c.08-.031.184-.047.259-.005c.019.009.036.03.03.05v.02h.003z"/>'],
  macos: ['macOS', '<path fill="currentColor" d="M12.152 6.896c-.948 0-2.415-1.078-3.96-1.04c-2.04.027-3.91 1.183-4.961 3.014c-2.117 3.675-.546 9.103 1.519 12.09c1.013 1.454 2.208 3.09 3.792 3.039c1.52-.065 2.09-.987 3.935-.987c1.831 0 2.35.987 3.96.948c1.637-.026 2.676-1.48 3.676-2.948c1.156-1.688 1.636-3.325 1.662-3.415c-.039-.013-3.182-1.221-3.22-4.857c-.026-3.04 2.48-4.494 2.597-4.559c-1.429-2.09-3.623-2.324-4.39-2.376c-2-.156-3.675 1.09-4.61 1.09zM15.53 3.83c.843-1.012 1.4-2.427 1.245-3.83c-1.207.052-2.662.805-3.532 1.818c-.78.896-1.454 2.338-1.273 3.714c1.338.104 2.715-.688 3.559-1.701"/>'],
  windows: ['Windows (WSL)', '<path fill="currentColor" d="M0 0h11.377v11.372H0Zm12.623 0H24v11.372H12.623ZM0 12.623h11.377V24H0Zm12.623 0H24V24H12.623"/>'],
};
function runsOn(p) {
  const on = p.platforms?.length ? p.platforms : ['linux', 'macos'];
  const keys = ['linux', 'macos'].filter((k) => on.includes(k));
  if (on.includes('linux')) keys.push('windows');
  return `<span class="os">${keys.map((k) => `<svg viewBox="0 0 24 24" role="img" aria-label="${OS_MARKS[k][0]}"><title>${OS_MARKS[k][0]}</title>${OS_MARKS[k][1]}</svg>`).join('')}</span>`;
}

function matches(p, term) {
  if (filterKind && !FILTER_VALUES[filterKind]?.(p).some((v) => slug(v) === filterSlug)) return false;
  if (!term) return true;
  return [p.name, p.description, p.author, ...(p.tags ?? [])].join(' ').toLowerCase().includes(term);
}

function render() {
  const term = q.value.trim().toLowerCase();
  const shown = plugins.filter((p) => matches(p, term)).sort(SORTS[sortSel.value] ?? SORTS.updated);
  const dateLabel = sortSel.value === 'added' ? 'Added' : 'Updated';
  const dateOf = (p) => (sortSel.value === 'added' ? p.added_at : p.updated_at);
  // A search shows every match at once; the pages are for browsing.
  const pages = term ? 1 : Math.max(1, Math.ceil(shown.length / PAGE_SIZE));
  const first = term ? 0 : Math.min(page, pages - 1) * PAGE_SIZE;
  count.innerHTML = `<b>${pad(shown.length)}</b> / ${plugins.length} plugins`;
  grid.innerHTML = shown.length
    ? shown
        .slice(first, term ? undefined : first + PAGE_SIZE)
        .map(
          (p, i) => `
      <a class="card" href="${encodeURIComponent(p.name)}/">
        ${preview(p)}
        <div class="card-body">
          <div class="card-top"><span class="num">${pad(first + i + 1)}</span><span>${esc(p.author)}</span></div>
          <h3>${esc(p.name)}${version(p)}</h3>
          <p>${esc(p.description)}</p>
          <span class="runs">${runsOn(p)}</span>
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

// Page n (from 0) of this list, relative to the site root, as ci/index names it.
function listPage(n) {
  const base = filterKind ? `${filterKind}/${filterSlug}/` : '';
  return n > 0 ? `${base}page/${n + 1}/` : base || './';
}

function renderPager(pages) {
  if (pages < 2) {
    pager.innerHTML = '';
    return;
  }
  const step = (n, label, name) =>
    n < 0 || n >= pages ? `<span aria-label="${name}">${label}</span>` : `<a href="${listPage(n)}" aria-label="${name}">${label}</a>`;
  pager.innerHTML = [
    step(page - 1, '←', 'Previous page'),
    ...Array.from({ length: pages }, (_, n) => `<a href="${listPage(n)}"${n === page ? ' aria-current="page"' : ''}>${pad(n + 1)}</a>`),
    step(page + 1, '→', 'Next page'),
  ].join('');
}

// The approved commit's own version, beside the name.
const version = (p) => (p.version ? ` <span class="ver">v${esc(p.version)}</span>` : '');

function renderCategories() {
  const all = [...new Set(plugins.map((p) => p.category).filter(Boolean))].sort();
  const current = (c) => (c ? filterKind === 'category' && slug(c) === filterSlug : !filterKind);
  cats.innerHTML = ['', ...all]
    .map((c) => `<a class="chip" href="${c ? esc(filterURL('category', c)) : './'}"${current(c) ? ' aria-current="page"' : ''}>${c ? esc(c) : 'All'}</a>`)
    .join('');
}

const GITHUB = '<svg viewBox="0 0 16 16" fill="currentColor" aria-hidden="true"><path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z"/></svg>';

// The repository as a button anyone can spot: GitHub's mark and owner/name,
// with its stars, the way the header shows Lerd's own.
function repoLink(p) {
  const gh = /^https:\/\/github\.com\/([^/]+\/[^/]+?)\/?$/.exec(p.repo);
  const stars = p.stars != null ? `<span class="stars" title="GitHub stars">${STAR} ${p.stars.toLocaleString()}</span>` : '';
  return gh
    ? `<a class="btn repo" href="${esc(p.repo)}">${GITHUB}${esc(gh[1])}${stars}</a>`
    : `<a class="btn repo" href="${esc(p.repo)}">Repository</a>`;
}

function shareMenu(p) {
  const url = encodeURIComponent(pageURL(p));
  const text = encodeURIComponent(`${p.name}, a Lerd plugin: ${p.description}`);
  const to = [
    ['X', `https://x.com/intent/post?text=${text}&url=${url}`],
    ['Bluesky', `https://bsky.app/intent/compose?text=${text}%20${url}`],
    ['LinkedIn', `https://www.linkedin.com/sharing/share-offsite/?url=${url}`],
    ['Reddit', `https://www.reddit.com/submit?url=${url}&title=${text}`],
  ];
  return `<div class="share">
      <button class="btn" type="button" aria-haspopup="true" data-share="${esc(pageURL(p))}" data-title="${esc(p.name)}">Share</button>
      <div class="share-menu" hidden>
        <button type="button" data-copy="${esc(pageURL(p))}" data-label="Copy link">Copy link</button>
        ${to.map(([n, href]) => `<a href="${esc(href)}" target="_blank" rel="noopener">${n}</a>`).join('')}
      </div>
    </div>`;
}

// The screenshots as the dashboard's plugin page shows them: one at a time in
// a fixed frame, so other shapes never move the page, with thumbnails below.
function slider(images) {
  if (images.length === 1) return `<figure class="slider"><div class="frame"><img src="${esc(images[0])}" alt=""></div></figure>`;
  return `<figure class="slider" data-shown="0">
      <div class="frame">
        <img src="${esc(images[0])}" alt="">
        <button type="button" class="arrow prev" data-step="-1" aria-label="Previous screenshot">←</button>
        <button type="button" class="arrow next" data-step="1" aria-label="Next screenshot">→</button>
        <span class="counter">1 / ${images.length}</span>
      </div>
      <div class="thumbs">${images
        .map((src, i) => `<button type="button" data-slide="${i}" data-src="${esc(src)}" aria-label="Screenshot ${i + 1} of ${images.length}" aria-pressed="${i === 0}"><img src="${esc(src)}" alt="" loading="lazy">${i === 0 ? '<span class="slide-bar"></span>' : ''}</button>`)
        .join('')}</div>
    </figure>`;
}

// showSlide moves the slider to screenshot i; the active thumbnail gets a fresh
// bar, whose fill restarts and whose end steps to the next one.
function showSlide(fig, i) {
  const thumbs = fig.querySelectorAll('[data-slide]');
  const n = (i + thumbs.length) % thumbs.length;
  fig.dataset.shown = n;
  fig.querySelector('.frame img').src = thumbs[n].dataset.src;
  fig.querySelector('.counter').textContent = `${n + 1} / ${thumbs.length}`;
  fig.querySelector('.slide-bar')?.remove();
  thumbs.forEach((t, j) => t.setAttribute('aria-pressed', j === n));
  thumbs[n].insertAdjacentHTML('beforeend', '<span class="slide-bar"></span>');
}

// The license the reviewed commit ships, linked to its text at that commit.
function licenseLink(p) {
  if (!p.license_file) return '';
  const href = p.repo.startsWith('https://github.com/') ? `${p.repo}/blob/${p.commit}/${p.license_file}` : p.repo;
  return `<a href="${esc(href)}">${p.license ? `${esc(p.license)} license` : 'License'}</a>`;
}

function detail(p) {
  const cmd = `lerd plugin install ${p.name}`;
  return `
    <p class="label crumbs"><a href="./">All plugins</a> / ${p.category ? `<a href="${esc(filterURL('category', p.category))}"><b>${esc(p.category)}</b></a> / ` : ''}<span class="by">by ${p.author ? `<a href="${esc(filterURL('author', p.author))}">${esc(p.author)}</a>` : ''}</span></p>
    <h1 id="d-name">${esc(p.name)}${version(p)}</h1>
    <p>${esc(p.description)}</p>
    <div class="links">
      ${repoLink(p)}
      ${shareMenu(p)}
    </div>
    <div class="facts">
      <span class="runs-on">Runs on ${runsOn(p)}</span>
      ${p.updated_at ? `<span>Updated ${esc(fmtDate(p.updated_at))}</span>` : ''}
      <span>Reviewed at <a href="${esc(commitURL(p))}">${short(p.commit)}</a></span>
      ${licenseLink(p)}
    </div>
    ${p.tags?.length ? `<div class="badges tags">${p.tags.map((t) => `<a class="badge" href="${esc(filterURL('tag', t))}">${esc(t)}</a>`).join('')}</div>` : ''}
    ${p.images?.length ? slider(p.images) : ''}
    <div class="install">
      <div class="cmd"><i>$</i><code>${esc(cmd)}</code><button class="copy" data-copy="${esc(cmd)}">Copy</button></div>
      <a class="btn ink" href="http://lerd.localhost/#system/plugins/browse/${encodeURIComponent(p.name)}">Install in Lerd</a>
    </div>
    ${p.requires?.length ? `<div class="reqs"><span>Requires</span><div class="badges">${p.requires.map((r) => `<span class="badge">${esc(r)}</span>`).join('')}</div></div>` : ''}
    <p class="note">Lerd installs commit <a href="${esc(p.repo.startsWith('https://github.com/') ? `${p.repo}/commit/${p.commit}` : commitURL(p))}"><code>${short(p.commit)}</code></a> and nothing newer, even if the repository has moved on.</p>`;
}

// showPlugin turns the page into the plugin's own: its detail in place of the list.
function showPlugin(name) {
  const p = plugins.find((x) => x.name === name);
  document.body.classList.add('on-plugin');
  pluginBody.innerHTML = p
    ? detail(p)
    : `<a class="back" href="./">← All plugins</a><p class="empty">No plugin named ${esc(name)} is listed.</p>`;
  window.scrollTo(0, 0);
}

document.addEventListener('click', (e) => {
  const fig = e.target.closest('.slider[data-shown]');
  const btn = e.target.closest('[data-step], [data-slide]');
  if (!fig || !btn) return;
  const at = Number(fig.dataset.shown);
  showSlide(fig, btn.dataset.slide != null ? Number(btn.dataset.slide) : at + Number(btn.dataset.step));
});
document.addEventListener('animationend', (e) => {
  if (!e.target.classList.contains('slide-bar')) return;
  const fig = e.target.closest('.slider');
  showSlide(fig, Number(fig.dataset.shown) + 1);
});
// Arrow keys step the slider, unless a field has the focus.
document.addEventListener('keydown', (e) => {
  const fig = document.querySelector('body.on-plugin .slider[data-shown]');
  if (!fig || e.target.closest?.('input, textarea, select')) return;
  if (e.key === 'ArrowLeft') showSlide(fig, Number(fig.dataset.shown) - 1);
  if (e.key === 'ArrowRight') showSlide(fig, Number(fig.dataset.shown) + 1);
});
// On a phone Share opens the system's share sheet; on a desktop, where that
// sheet is sparse or missing, it opens our own menu of the usual places.
const touch = matchMedia('(pointer: coarse)');
document.addEventListener('click', async (e) => {
  const open = document.querySelector('.share-menu:not([hidden])');
  const btn = e.target.closest('[data-share]');
  if (open && !e.target.closest('.share-menu')) open.hidden = true;
  if (!btn) return;
  if (touch.matches && navigator.share) {
    try {
      await navigator.share({ title: `${btn.dataset.title} · Lerd Plugins`, url: btn.dataset.share });
      return;
    } catch (err) {
      if (err?.name === 'AbortError') return;
    }
  }
  const menu = btn.nextElementSibling;
  menu.hidden = menu === open;
});
document.addEventListener('keydown', (e) => {
  if (e.key === 'Escape') document.querySelectorAll('.share-menu').forEach((m) => (m.hidden = true));
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
  setTimeout(() => (btn.textContent = btn.dataset.label ?? 'Copy'), 1500);
});
q.addEventListener('input', render);
sortSel.addEventListener('change', () => {
  try {
    localStorage.setItem('lerd-plugins-sort', sortSel.value);
  } catch {
    // Not remembered, but still applied.
  }
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
    document.body.classList.toggle('on-filter', Boolean(filterKind));
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
