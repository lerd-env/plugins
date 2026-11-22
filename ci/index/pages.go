package main

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// siteURL is where plugins.lerd.sh serves the catalog, for the absolute links
// social cards need.
const siteURL = "https://plugins.lerd.sh"

// reservedNames are the folders the site itself serves, which a plugin's own
// page must not replace.
var reservedNames = map[string]bool{"images": true, "tag": true, "category": true, "author": true, "page": true}

var (
	// The home page's own description and social tags, replaced on every other page.
	homeTags  = regexp.MustCompile(`(?m)^\s*<meta (name="description"|property="og:[a-z:]+"|name="twitter:[a-z:]+") content="[^"]*">\n`)
	titleTag  = regexp.MustCompile(`<title>[^<]*</title>`)
	homeLead  = regexp.MustCompile(`<p class="lead">[^<]*</p>`)
	homeH1    = `<h1>Plugins<span>for Lerd</span></h1>`
	homeLabel = `<p class="label"><b>00</b> / Reviewed plugins</p>`
	emptyBody = `<div class="wrap" id="plugin-body"></div>`
	emptyGrid = `<div id="grid" class="grid" aria-live="polite"></div>`
	emptyNav  = `<nav id="pager" class="pager" aria-label="Pages"></nav>`
	emptyDocs = `<div class="wrap" id="plugin-docs"></div>`
)

// head is what a generated page says about itself to browsers, crawlers and
// the cards a shared link unfolds into.
type head struct {
	title, socialTitle, desc, url, image string
	extra                                []string
	ld                                   any
}

// apply puts the head into the template in place of the home page's own.
func (h head) apply(tmpl []byte) (string, error) {
	s := string(tmpl)
	if !titleTag.MatchString(s) {
		return "", fmt.Errorf("the site template has no <title> to replace")
	}
	e := html.EscapeString
	tags := append([]string{
		// The page lives in a folder, so its relative links resolve from the root.
		`<base href="/">`,
		fmt.Sprintf(`<meta name="description" content="%s">`, e(h.desc)),
		`<meta property="og:type" content="website">`,
		`<meta property="og:site_name" content="Lerd Plugins">`,
		fmt.Sprintf(`<meta property="og:url" content="%s">`, e(h.url)),
		fmt.Sprintf(`<meta property="og:title" content="%s">`, e(h.socialTitle)),
		fmt.Sprintf(`<meta property="og:description" content="%s">`, e(h.desc)),
		fmt.Sprintf(`<meta property="og:image" content="%s">`, e(h.image)),
		`<meta name="twitter:card" content="summary_large_image">`,
		fmt.Sprintf(`<meta name="twitter:title" content="%s">`, e(h.socialTitle)),
		fmt.Sprintf(`<meta name="twitter:description" content="%s">`, e(h.desc)),
		fmt.Sprintf(`<meta name="twitter:image" content="%s">`, e(h.image)),
	}, h.extra...)
	tags = append(tags, ldScript(h.ld))
	s = homeTags.ReplaceAllString(s, "")
	loc := titleTag.FindStringIndex(s)
	return s[:loc[0]] + "<title>" + e(h.title) + "</title>\n  " + strings.Join(tags, "\n  ") + s[loc[1]:], nil
}

// replaceOnce swaps the template's one copy of old, failing loudly when a
// template change removed what the generator fills in.
func replaceOnce(s, old, new string) (string, error) {
	if !strings.Contains(s, old) {
		return "", fmt.Errorf("the site template has no %s to replace", old)
	}
	return strings.Replace(s, old, new, 1), nil
}

func replaceAll(s string, pairs ...string) (string, error) {
	var err error
	for i := 0; i < len(pairs) && err == nil; i += 2 {
		s, err = replaceOnce(s, pairs[i], pairs[i+1])
	}
	return s, err
}

func writePage(out, rel string, page []byte) error {
	dir := filepath.Join(out, rel)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "index.html"), page, 0o644)
}

// cards are the list as plain links, for crawlers and for anyone before the
// script draws the real cards over them.
func cards(ls []listing) string {
	var b strings.Builder
	b.WriteString(`<div id="grid" class="grid" aria-live="polite">`)
	for _, l := range ls {
		fmt.Fprintf(&b, `<a class="card" href="%s/"><div class="card-body"><h3>%s</h3><p>%s</p></div></a>`, html.EscapeString(l.Name), html.EscapeString(l.Name), html.EscapeString(l.Description))
	}
	b.WriteString(`</div>`)
	return b.String()
}

// pageSize is how many plugins a page of the list holds, as the site's script pages them.
const pageSize = 6

// byUpdate is the list's default order, as the site's script sorts it: most
// recently updated first, then by name.
func byUpdate(ls []listing) []listing {
	out := slices.Clone(ls)
	slices.SortStableFunc(out, func(a, b listing) int {
		if c := strings.Compare(b.UpdatedAt, a.UpdatedAt); c != 0 {
			return c
		}
		return strings.Compare(a.Name, b.Name)
	})
	return out
}

func pageCount(n int) int { return max(1, (n+pageSize-1)/pageSize) }

// pageOf is page n's share of the list, counted from 1.
func pageOf(ls []listing, n int) []listing {
	return ls[min(len(ls), (n-1)*pageSize):min(len(ls), n*pageSize)]
}

// pagePath is page n of the list at base ("" for the home page), relative to the site root.
func pagePath(base string, n int) string {
	if n > 1 {
		return base + "page/" + strconv.Itoa(n) + "/"
	}
	if base == "" {
		return "./"
	}
	return base
}

func pageURL(base string, n int) string {
	return siteURL + "/" + strings.TrimPrefix(pagePath(base, n), "./")
}

// pagerNav links every page of the list, the same markup the site's script draws.
func pagerNav(base string, n, pages int) string {
	if pages < 2 {
		return emptyNav
	}
	step := func(to int, label, name string) string {
		if to < 1 || to > pages {
			return fmt.Sprintf(`<span aria-label="%s">%s</span>`, name, label)
		}
		return fmt.Sprintf(`<a href="%s" aria-label="%s">%s</a>`, pagePath(base, to), name, label)
	}
	var b strings.Builder
	b.WriteString(`<nav id="pager" class="pager" aria-label="Pages">` + step(n-1, "←", "Previous page"))
	for i := 1; i <= pages; i++ {
		current := ""
		if i == n {
			current = ` aria-current="page"`
		}
		fmt.Fprintf(&b, `<a href="%s"%s>%02d</a>`, pagePath(base, i), current, i)
	}
	b.WriteString(step(n+1, "→", "Next page") + `</nav>`)
	return b.String()
}

// pageLinks tell crawlers where page n sits: its own address and its neighbours.
func pageLinks(base string, n, pages int) []string {
	links := []string{fmt.Sprintf(`<link rel="canonical" href="%s">`, pageURL(base, n))}
	if n > 1 {
		links = append(links, fmt.Sprintf(`<link rel="prev" href="%s">`, pageURL(base, n-1)), fmt.Sprintf(`<meta name="lerd-page" content="%d">`, n))
	}
	if n < pages {
		links = append(links, fmt.Sprintf(`<link rel="next" href="%s">`, pageURL(base, n+1)))
	}
	return links
}

// writeHomePages writes the home page and the list's further pages at /page/<n>/.
func writeHomePages(tmpl []byte, c catalog, out string) error {
	for n := 1; n <= pageCount(len(c.Plugins)); n++ {
		page, err := homePage(tmpl, c, n)
		if err != nil {
			return err
		}
		path := filepath.Join(out, "index.html")
		if n > 1 {
			path = filepath.Join(out, "page", strconv.Itoa(n), "index.html")
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, page, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// homePage is page n of the site's template with its plugins linked and the
// site's structured data; its title and social tags are the template's own.
func homePage(tmpl []byte, c catalog, n int) ([]byte, error) {
	all := byUpdate(c.Plugins)
	pages := pageCount(len(all))
	head := append(pageLinks("", n, pages), ldScript(graph(
		map[string]any{"@type": "WebSite", "name": "Lerd Plugins", "url": siteURL + "/"},
		itemList(all),
	)))
	s, err := replaceAll(string(tmpl),
		emptyGrid, cards(pageOf(all, n)),
		emptyNav, pagerNav("", n, pages),
		"</head>", "  "+strings.Join(head, "\n  ")+"\n</head>",
	)
	if err != nil || n == 1 {
		return []byte(s), err
	}
	// A page one folder down resolves its relative links from the root, so the
	// base comes before the stylesheet and script.
	loc := titleTag.FindStringIndex(s)
	return []byte(s[:loc[0]] + `<base href="/">` + "\n  " + s[loc[0]:loc[1]-len("</title>")] + fmt.Sprintf(" · page %d</title>", n) + s[loc[1]:]), nil
}

// writePluginPages gives every plugin a page at /<name>/: the site itself, told
// which plugin to show, with that plugin's title, description and screenshot
// as the card a shared link unfolds into.
func writePluginPages(tmpl []byte, c catalog, out string) error {
	for _, l := range c.Plugins {
		if reservedNames[l.Name] {
			return fmt.Errorf("a plugin named %q would replace the site's own %s/ folder", l.Name, l.Name)
		}
		page, err := pluginPage(tmpl, l)
		if err != nil {
			return fmt.Errorf("%s: %w", l.Name, err)
		}
		if err := writePage(out, l.Name, page); err != nil {
			return err
		}
	}
	return nil
}

func pluginPage(tmpl []byte, l listing) ([]byte, error) {
	url := siteURL + "/" + l.Name + "/"
	image := siteURL + "/og.png"
	if len(l.Images) > 0 {
		image = siteURL + "/" + l.Images[0]
	}
	social := l.Name + ", a Lerd plugin"
	if l.Author != "" {
		social += " by " + l.Author
	}
	title := l.Name + " · Lerd plugin"
	if l.Description != "" {
		title = l.Name + ": " + clip(l.Description, 60) + " · Lerd plugin"
	}
	crumbs := [][2]string{{"Plugins", siteURL + "/"}}
	if l.Category != "" {
		crumbs = append(crumbs, [2]string{l.Category, filterURL("category", l.Category)})
	}
	crumbs = append(crumbs, [2]string{l.Name, url})
	h := head{
		title: title, socialTitle: social, desc: l.Description, url: url, image: image,
		extra: []string{
			fmt.Sprintf(`<link rel="canonical" href="%s">`, html.EscapeString(url)),
			fmt.Sprintf(`<meta name="lerd-plugin" content="%s">`, html.EscapeString(l.Name)),
		},
		ld: graph(software(l, url, image), breadcrumbs(crumbs)),
	}
	s, err := h.apply(tmpl)
	if err != nil {
		return nil, err
	}
	// The plugin's name is the page's one heading; the home one, hidden here, steps down.
	s, err = replaceAll(s,
		"<body>", `<body class="on-plugin">`,
		homeH1, `<p class="h1">Plugins<span>for Lerd</span></p>`,
		emptyBody, fmt.Sprintf(`<div class="wrap" id="plugin-body"><h1 id="d-name">%s</h1><p>%s</p></div>`, html.EscapeString(l.Name), html.EscapeString(l.Description)),
		// The readme sits outside the body the script redraws, so it stays.
		emptyDocs, `<div class="wrap" id="plugin-docs">`+readmeSection(l)+`</div>`,
	)
	return []byte(s), err
}

// readmeSection is the readme under a label naming the commit it was read at,
// the one Lerd installs, so a newer one on GitHub is not mistaken for it.
func readmeSection(l listing) string {
	if l.Readme == "" {
		return ""
	}
	sha := l.Commit[:min(7, len(l.Commit))]
	at := "<code>" + html.EscapeString(sha) + "</code>"
	if strings.HasPrefix(l.Repo, "https://github.com/") {
		at = `<a href="` + html.EscapeString(l.Repo+"/commit/"+l.Commit) + `">` + at + `</a>`
	}
	return `<p class="label">README at reviewed commit ` + at + `</p>` + l.Readme
}

// clip shortens s to about n characters at a word, for titles search results show whole.
func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	cut := strings.LastIndex(s[:n], " ")
	if cut < 0 {
		cut = n
	}
	return strings.TrimRight(s[:cut], ",;:") + "…"
}

// filters are the listing fields that get a page per value, at /<kind>/<slug>/,
// each listing the plugins that carry it.
var filters = []struct {
	kind   string
	values func(listing) []string
}{
	{"category", func(l listing) []string { return []string{l.Category} }},
	{"tag", func(l listing) []string { return l.Tags }},
	{"author", func(l listing) []string { return []string{l.Author} }},
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

// slug is the folder a filter value's page lives in; the site's script builds
// the same one for its links.
func slug(s string) string {
	return strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(s), "-"), "-")
}

func filterURL(kind, value string) string {
	return siteURL + "/" + kind + "/" + slug(value) + "/"
}

// filterValue is one page's filter: the spelling first seen, its plugins and
// the newest update among them.
type filterValue struct {
	kind, slug, label, updated string
	plugins                    []listing
}

// thin is a tag only one plugin carries: a page crawlers follow but leave out
// of search, so a few dozen near-empty pages don't stand for the site.
func (v filterValue) thin() bool { return v.kind == "tag" && len(v.plugins) < 2 }

// base is the folder the filter's pages live under, relative to the site root.
func (v filterValue) base() string { return v.kind + "/" + v.slug + "/" }

func filterValues(c catalog) []filterValue {
	var vals []filterValue
	for _, f := range filters {
		at := map[string]int{}
		for _, l := range c.Plugins {
			seen := map[string]bool{}
			for _, v := range f.values(l) {
				s := slug(v)
				if s == "" || seen[s] {
					continue
				}
				seen[s] = true
				i, ok := at[s]
				if !ok {
					i = len(vals)
					at[s] = i
					vals = append(vals, filterValue{kind: f.kind, slug: s, label: v})
				}
				vals[i].plugins = append(vals[i].plugins, l)
				if l.UpdatedAt > vals[i].updated {
					vals[i].updated = l.UpdatedAt
				}
			}
		}
	}
	return vals
}

// writeFilterPages gives every category, tag and author a page: the site's
// list, told which filter to apply, with its own title and description.
func writeFilterPages(tmpl []byte, c catalog, out string) error {
	for _, v := range filterValues(c) {
		for n := 1; n <= pageCount(len(v.plugins)); n++ {
			page, err := filterPage(tmpl, v, n)
			if err != nil {
				return fmt.Errorf("%s %s: %w", v.kind, v.label, err)
			}
			if err := writePage(out, strings.TrimSuffix(pagePath(v.base(), n), "/"), page); err != nil {
				return err
			}
		}
	}
	return nil
}

func filterPage(tmpl []byte, v filterValue, n int) ([]byte, error) {
	url := pageURL(v.base(), n)
	all := byUpdate(v.plugins)
	pages := pageCount(len(all))
	phrase := map[string]string{"category": "for ", "tag": "tagged ", "author": "by "}[v.kind] + v.label
	title := "Lerd plugins " + phrase
	if n > 1 {
		title += fmt.Sprintf(" · page %d", n)
	}
	names := make([]string, len(v.plugins))
	for i, l := range v.plugins {
		names[i] = l.Name
	}
	desc := fmt.Sprintf("A Lerd plugin %s: %s, reviewed at the commit Lerd installs.", phrase, names[0])
	if len(names) > 1 {
		desc = fmt.Sprintf("%d Lerd plugins %s: %s, each reviewed at the commit Lerd installs.", len(names), phrase, andList(names))
	}
	extra := append(pageLinks(v.base(), n, pages), fmt.Sprintf(`<meta name="lerd-filter" content="%s:%s">`, v.kind, html.EscapeString(v.slug)))
	if v.thin() {
		extra = append(extra, `<meta name="robots" content="noindex, follow">`)
	}
	h := head{
		title: title, socialTitle: title, desc: desc, url: url, image: siteURL + "/og.png", extra: extra,
		ld: graph(
			map[string]any{"@type": "CollectionPage", "name": title, "url": url, "mainEntity": itemList(all)},
			breadcrumbs([][2]string{{"Plugins", siteURL + "/"}, {v.label, pageURL(v.base(), 1)}}),
		),
	}
	s, err := h.apply(tmpl)
	if err != nil {
		return nil, err
	}
	// The heading and intro are in the page itself, so crawlers read them
	// without running the script.
	s, err = replaceAll(s,
		"<body>", `<body class="on-filter">`,
		homeLabel, fmt.Sprintf(`<p class="label"><b>00</b> / %s</p>`, strings.ToUpper(v.kind[:1])+v.kind[1:]),
		homeH1, fmt.Sprintf(`<h1>%s<span>plugins for Lerd</span></h1>`, html.EscapeString(v.label)),
		emptyGrid, cards(pageOf(all, n)),
		emptyNav, pagerNav(v.base(), n, pages),
	)
	if err != nil {
		return nil, err
	}
	lead := homeLead.FindStringIndex(s)
	if lead == nil {
		return nil, fmt.Errorf(`the site template has no <p class="lead"> to replace`)
	}
	return []byte(s[:lead[0]] + `<p class="lead">` + html.EscapeString(desc) + `</p>` + s[lead[1]:]), nil
}

func andList(s []string) string {
	if len(s) < 2 {
		return strings.Join(s, "")
	}
	return strings.Join(s[:len(s)-1], ", ") + " and " + s[len(s)-1]
}

// The structured data search engines read: what each page is and where it sits.

func ldScript(v any) string {
	// Marshal escapes <, > and &, so no value can close the script tag.
	b, _ := json.Marshal(v)
	return `<script type="application/ld+json">` + string(b) + `</script>`
}

func graph(nodes ...any) map[string]any {
	return map[string]any{"@context": "https://schema.org", "@graph": nodes}
}

func software(l listing, url, image string) map[string]any {
	m := map[string]any{
		"@type": "SoftwareApplication", "name": l.Name, "description": l.Description, "url": url, "image": image,
		"applicationCategory": "DeveloperApplication", "operatingSystem": osNames(l.Platforms),
		"offers": map[string]any{"@type": "Offer", "price": "0", "priceCurrency": "USD"},
	}
	if l.Version != "" {
		m["softwareVersion"] = l.Version
	}
	if l.UpdatedAt != "" {
		m["dateModified"] = l.UpdatedAt
	}
	if l.LicenseFile != "" && strings.HasPrefix(l.Repo, "https://github.com/") {
		m["license"] = l.Repo + "/blob/" + l.Commit + "/" + l.LicenseFile
	}
	return m
}

// osNames mirrors the site's: Lerd runs inside WSL on Windows, so a plugin for
// linux runs there too.
func osNames(platforms []string) string {
	on := map[string]bool{"linux": true, "macos": true}
	if len(platforms) > 0 {
		on = map[string]bool{}
		for _, p := range platforms {
			on[p] = true
		}
	}
	var names []string
	if on["linux"] {
		names = append(names, "Linux")
	}
	if on["macos"] {
		names = append(names, "macOS")
	}
	if on["linux"] {
		names = append(names, "Windows (WSL)")
	}
	return strings.Join(names, ", ")
}

func breadcrumbs(items [][2]string) map[string]any {
	list := make([]any, len(items))
	for i, it := range items {
		list[i] = map[string]any{"@type": "ListItem", "position": i + 1, "name": it[0], "item": it[1]}
	}
	return map[string]any{"@type": "BreadcrumbList", "itemListElement": list}
}

func itemList(ls []listing) map[string]any {
	list := make([]any, len(ls))
	for i, l := range ls {
		list[i] = map[string]any{"@type": "ListItem", "position": i + 1, "name": l.Name, "url": siteURL + "/" + l.Name + "/"}
	}
	return map[string]any{"@type": "ItemList", "itemListElement": list}
}

// writeSitemap lists the home page, every plugin's page and every filter page
// worth a search result, for search engines, which robots.txt points at.
func writeSitemap(c catalog, out string) error {
	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	b.WriteString("<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">\n")
	b.WriteString("<url><loc>" + siteURL + "/</loc></url>\n")
	url := func(loc, lastmod string) {
		b.WriteString("<url><loc>" + html.EscapeString(loc) + "</loc>")
		if lastmod != "" {
			b.WriteString("<lastmod>" + html.EscapeString(lastmod) + "</lastmod>")
		}
		b.WriteString("</url>\n")
	}
	for n := 2; n <= pageCount(len(c.Plugins)); n++ {
		url(pageURL("", n), "")
	}
	for _, l := range c.Plugins {
		url(siteURL+"/"+l.Name+"/", l.UpdatedAt)
	}
	for _, v := range filterValues(c) {
		if v.thin() {
			continue
		}
		for n := 1; n <= pageCount(len(v.plugins)); n++ {
			url(pageURL(v.base(), n), v.updated)
		}
	}
	b.WriteString("</urlset>\n")
	return os.WriteFile(filepath.Join(out, "sitemap.xml"), []byte(b.String()), 0o644)
}
