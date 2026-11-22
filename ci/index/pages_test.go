package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const tmpl = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>Lerd Plugins: the home title</title>
  <meta name="description" content="Plugins for Lerd.">
  <meta property="og:title" content="Lerd Plugins">
</head>
<body>
  <p class="label"><b>00</b> / Reviewed plugins</p>
  <h1>Plugins<span>for Lerd</span></h1>
  <p class="lead">Home intro.</p>
  <section class="plugin"><div class="wrap" id="plugin-body"></div></section>
  <section class="docs"><div class="wrap" id="plugin-docs"></div></section>
  <div id="grid" class="grid" aria-live="polite"></div>
  <nav id="pager" class="pager" aria-label="Pages"></nav>
</body>
</html>
`

func TestPluginPage_CarriesItsOwnSocialCard(t *testing.T) {
	l := listing{Name: "sail", Description: `Bring "Sail" <data> into Lerd`, Author: "lerd-env", Version: "0.1.0", Images: []string{"images/sail/abc/1.png"}}
	page, err := pluginPage([]byte(tmpl), l)
	if err != nil {
		t.Fatal(err)
	}
	s := string(page)
	for _, want := range []string{
		`<base href="/">`,
		`<title>sail: Bring &#34;Sail&#34; &lt;data&gt; into Lerd · Lerd plugin</title>`,
		`<body class="on-plugin">`,
		`<meta name="description" content="Bring &#34;Sail&#34; &lt;data&gt; into Lerd">`,
		`<link rel="canonical" href="https://plugins.lerd.sh/sail/">`,
		`<meta property="og:title" content="sail, a Lerd plugin by lerd-env">`,
		`<meta property="og:image" content="https://plugins.lerd.sh/images/sail/abc/1.png">`,
		`<meta name="twitter:card" content="summary_large_image">`,
		`<meta name="lerd-plugin" content="sail">`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("page lacks %s\n%s", want, s)
		}
	}
	// The home page's own social tags would contradict the plugin's.
	if strings.Contains(s, `<meta property="og:title" content="Lerd Plugins">`) || strings.Contains(s, "Plugins for Lerd.") {
		t.Errorf("the home page's tags stayed in:\n%s", s)
	}
}

func TestPluginPage_FallsBackToTheSiteCardWithoutAScreenshot(t *testing.T) {
	page, err := pluginPage([]byte(tmpl), listing{Name: "quiet", Description: "d", Author: "a"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(page), `<meta property="og:image" content="https://plugins.lerd.sh/og.png">`) {
		t.Errorf("no fallback image:\n%s", page)
	}
}

func TestWritePluginPages(t *testing.T) {
	out := t.TempDir()
	c := catalog{Plugins: []listing{{Name: "sail"}, {Name: "notes"}}}
	if err := writePluginPages([]byte(tmpl), c, out); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"sail", "notes"} {
		if _, err := os.Stat(filepath.Join(out, name, "index.html")); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	// A plugin named like a folder the site serves would replace it.
	err := writePluginPages([]byte(tmpl), catalog{Plugins: []listing{{Name: "images"}}}, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "images") {
		t.Errorf("err = %v, want one naming images", err)
	}
}

// Search engines find each plugin's page through the sitemap, dated by when
// its entry last changed.
func TestWriteSitemap_ListsHomeAndEveryPluginPage(t *testing.T) {
	out := t.TempDir()
	c := catalog{Plugins: []listing{{Name: "sail", UpdatedAt: "2026-11-02"}, {Name: "notes"}}}
	if err := writeSitemap(c, out); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(out, "sitemap.xml"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{
		`<?xml version="1.0" encoding="UTF-8"?>`,
		`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`,
		`<url><loc>https://plugins.lerd.sh/</loc></url>`,
		`<url><loc>https://plugins.lerd.sh/sail/</loc><lastmod>2026-11-02</lastmod></url>`,
		`<url><loc>https://plugins.lerd.sh/notes/</loc></url>`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("sitemap lacks %s\n%s", want, s)
		}
	}
}

// Each tag and category gets its own page listing the plugins that carry it,
// one page per slug however the entries spell it.
func TestWriteFilterPages(t *testing.T) {
	out := t.TempDir()
	c := catalog{Plugins: []listing{
		{Name: "sail", Category: "Migration", Tags: []string{"laravel", "Docker Compose"}, Author: "lerd-env"},
		{Name: "notes", Category: "migration", Tags: []string{"laravel"}},
		{Name: "quiet", Tags: []string{"!!!"}},
	}}
	if err := writeFilterPages([]byte(tmpl), c, out); err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]string{
		"tag/laravel":        `<meta name="lerd-filter" content="tag:laravel">`,
		"tag/docker-compose": `<h1>Docker Compose<span>plugins for Lerd</span></h1>`,
		"category/migration": `<title>Lerd plugins for Migration</title>`,
		"author/lerd-env":    `<title>Lerd plugins by lerd-env</title>`,
	} {
		b, err := os.ReadFile(filepath.Join(out, path, "index.html"))
		if err != nil {
			t.Errorf("%s: %v", path, err)
			continue
		}
		s := string(b)
		for _, w := range []string{want, `<base href="/">`} {
			if !strings.Contains(s, w) {
				t.Errorf("%s lacks %s\n%s", path, w, s)
			}
		}
		// Crawlers read the heading and intro without running the script.
		if !regexp.MustCompile(`<p class="lead">(A|\d+) Lerd plugins? `).MatchString(s) || strings.Contains(s, "Home intro.") || strings.Contains(s, "<h1>Plugins<span>") {
			t.Errorf("%s kept the home page's heading or intro\n%s", path, s)
		}
		if strings.Contains(s, "Plugins for Lerd.") {
			t.Errorf("%s kept the home page's description", path)
		}
	}
	// A tag with nothing usable in it gets no page rather than an empty folder name.
	entries, _ := os.ReadDir(filepath.Join(out, "tag"))
	if len(entries) != 2 {
		t.Errorf("tag pages = %d, want 2", len(entries))
	}
}

func TestSlug(t *testing.T) {
	for in, want := range map[string]string{"AI": "ai", "Docker Compose": "docker-compose", "host-proxy": "host-proxy", " ../x ": "x", "!!!": ""} {
		if got := slug(in); got != want {
			t.Errorf("slug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestWritePluginPages_ReservesTheFilterFolders(t *testing.T) {
	for _, name := range []string{"tag", "category", "author", "page"} {
		err := writePluginPages([]byte(tmpl), catalog{Plugins: []listing{{Name: name}}}, t.TempDir())
		if err == nil || !strings.Contains(err.Error(), name) {
			t.Errorf("%s: err = %v, want one naming it", name, err)
		}
	}
}

func TestWriteSitemap_ListsFilterPages(t *testing.T) {
	out := t.TempDir()
	c := catalog{Plugins: []listing{{Name: "sail", Category: "Migration", Tags: []string{"laravel"}}, {Name: "notes", Tags: []string{"laravel"}}}}
	if err := writeSitemap(c, out); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(out, "sitemap.xml"))
	for _, want := range []string{"<loc>https://plugins.lerd.sh/tag/laravel/</loc>", "<loc>https://plugins.lerd.sh/category/migration/</loc>"} {
		if !strings.Contains(string(b), want) {
			t.Errorf("sitemap lacks %s\n%s", want, b)
		}
	}
}

// h1s lists a page's top headings; a page has exactly one, the thing it is about.
func h1s(s string) []string {
	return regexp.MustCompile(`<h1[^>]*>(.*?)</h1>`).FindAllString(s, -1)
}

// ldJSON decodes the page's structured data, failing the test when it is not valid JSON.
func ldJSON(t *testing.T, s string) map[string]any {
	t.Helper()
	m := regexp.MustCompile(`(?s)<script type="application/ld\+json">(.*?)</script>`).FindStringSubmatch(s)
	if m == nil {
		t.Fatalf("no structured data in\n%s", s)
	}
	var v map[string]any
	if err := json.Unmarshal([]byte(m[1]), &v); err != nil {
		t.Fatalf("structured data is not JSON: %v\n%s", err, m[1])
	}
	return v
}

// Crawlers that run no script still read what the plugin is.
func TestPluginPage_ReadsWithoutTheScript(t *testing.T) {
	l := listing{Name: "sail", Description: "Bring Sail into Lerd </script><b>", Author: "lerd-env", Category: "Migration", Version: "0.1.0", Platforms: []string{"linux"}, UpdatedAt: "2026-10-23", Repo: "https://github.com/lerd-env/plugin-sail", Commit: "abc", LicenseFile: "LICENSE"}
	page, err := pluginPage([]byte(tmpl), l)
	if err != nil {
		t.Fatal(err)
	}
	s := string(page)
	if got := h1s(s); len(got) != 1 || !strings.Contains(got[0], ">sail<") {
		t.Errorf("h1s = %q, want the plugin's name alone", got)
	}
	if !strings.Contains(s, `<div class="wrap" id="plugin-body"><h1 id="d-name">sail</h1><p>Bring Sail into Lerd &lt;/script&gt;&lt;b&gt;</p></div>`) {
		t.Errorf("no static name and description in the plugin body\n%s", s)
	}
	// A description cannot close the structured data's script tag.
	if strings.Count(s, "</script>") != 1 {
		t.Errorf("the description closed a script tag\n%s", s)
	}
	graph := ldJSON(t, s)["@graph"].([]any)
	app := graph[0].(map[string]any)
	if app["@type"] != "SoftwareApplication" || app["name"] != "sail" || app["operatingSystem"] != "Linux, Windows (WSL)" || app["softwareVersion"] != "0.1.0" {
		t.Errorf("software = %v", app)
	}
	if app["license"] != "https://github.com/lerd-env/plugin-sail/blob/abc/LICENSE" {
		t.Errorf("license = %v", app["license"])
	}
	crumbs := graph[1].(map[string]any)["itemListElement"].([]any)
	if len(crumbs) != 3 || crumbs[1].(map[string]any)["item"] != "https://plugins.lerd.sh/category/migration/" {
		t.Errorf("breadcrumbs = %v", crumbs)
	}
}

// A filter page names its plugins and links each one, so crawlers find them.
func TestFilterPage_ListsItsPluginsAsLinks(t *testing.T) {
	out := t.TempDir()
	c := catalog{Plugins: []listing{
		{Name: "db-browser", Description: "Browse a database", Tags: []string{"mysql", "sqlite"}},
		{Name: "db-import-host", Description: "Import a database", Tags: []string{"mysql"}},
	}}
	if err := writeFilterPages([]byte(tmpl), c, out); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(out, "tag/mysql/index.html"))
	s := string(b)
	for _, want := range []string{
		`<body class="on-filter">`,
		`<title>Lerd plugins tagged mysql</title>`,
		`<meta name="description" content="2 Lerd plugins tagged mysql: db-browser and db-import-host, each reviewed at the commit Lerd installs.">`,
		`<a class="card" href="db-browser/">`,
		`<a class="card" href="db-import-host/">`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("tag page lacks %s\n%s", want, s)
		}
	}
	if strings.Contains(s, "noindex") {
		t.Error("a tag two plugins share was kept out of search")
	}
	if len(h1s(s)) != 1 {
		t.Errorf("h1s = %q", h1s(s))
	}
	if ldJSON(t, s)["@graph"] == nil {
		t.Error("no structured data graph")
	}
	// A tag one plugin carries is a thin page: crawlers follow it but leave it out.
	b, _ = os.ReadFile(filepath.Join(out, "tag/sqlite/index.html"))
	s = string(b)
	if !strings.Contains(s, `<meta name="robots" content="noindex, follow">`) || strings.Contains(s, `href="db-import-host/"`) {
		t.Errorf("single-plugin tag page:\n%s", s)
	}
}

func TestWriteSitemap_LeavesOutThinTags(t *testing.T) {
	out := t.TempDir()
	c := catalog{Plugins: []listing{{Name: "a", Tags: []string{"shared", "lonely"}, Category: "Solo"}, {Name: "b", Tags: []string{"shared"}}}}
	if err := writeSitemap(c, out); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(out, "sitemap.xml"))
	s := string(b)
	if !strings.Contains(s, "/tag/shared/") || strings.Contains(s, "/tag/lonely/") || !strings.Contains(s, "/category/solo/") {
		t.Errorf("sitemap:\n%s", s)
	}
}

// The home page carries every plugin as a link and the site's own structured data.
func TestHomePage(t *testing.T) {
	c := catalog{Plugins: []listing{{Name: "sail", Description: "Bring Sail"}, {Name: "sync", Description: "Pull a site"}}}
	page, err := homePage([]byte(tmpl), c, 1)
	if err != nil {
		t.Fatal(err)
	}
	s := string(page)
	for _, want := range []string{`<a class="card" href="sail/">`, `<a class="card" href="sync/">`, `<title>Lerd Plugins: the home title</title>`} {
		if !strings.Contains(s, want) {
			t.Errorf("home lacks %s\n%s", want, s)
		}
	}
	if ldJSON(t, s)["@graph"] == nil {
		t.Error("no structured data graph")
	}
}

// The real template keeps every anchor the generator replaces.
func TestTheSiteTemplateHasEveryAnchor(t *testing.T) {
	real, err := os.ReadFile("../../site/index.html")
	if err != nil {
		t.Fatal(err)
	}
	c := catalog{Plugins: []listing{{Name: "sail", Description: "d", Author: "lerd-env", Category: "Migration", Tags: []string{"laravel"}}}}
	if _, err := homePage(real, c, 1); err != nil {
		t.Error(err)
	}
	if _, err := pluginPage(real, c.Plugins[0]); err != nil {
		t.Error(err)
	}
	if err := writeFilterPages(real, c, t.TempDir()); err != nil {
		t.Error(err)
	}
}

// seven plugins, p7 updated most recently, so six fill the first page.
func seven() catalog {
	var c catalog
	for i := 1; i <= 7; i++ {
		c.Plugins = append(c.Plugins, listing{Name: fmt.Sprintf("p%d", i), Author: "one", UpdatedAt: fmt.Sprintf("2026-11-0%d", i)})
	}
	return c
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// The list is split into real pages, newest first, each with plain links to the others.
func TestHomePages_SplitIntoPages(t *testing.T) {
	out := t.TempDir()
	if err := writeHomePages([]byte(tmpl), seven(), out); err != nil {
		t.Fatal(err)
	}
	first := read(t, filepath.Join(out, "index.html"))
	if !strings.Contains(first, `href="p7/"`) || !strings.Contains(first, `href="p2/"`) || strings.Contains(first, `href="p1/"`) {
		t.Errorf("first page holds the wrong plugins\n%s", first)
	}
	for _, want := range []string{`<a href="page/2/">02</a>`, `<link rel="next" href="https://plugins.lerd.sh/page/2/">`, `<link rel="canonical" href="https://plugins.lerd.sh/">`} {
		if !strings.Contains(first, want) {
			t.Errorf("first page lacks %s\n%s", want, first)
		}
	}
	second := read(t, filepath.Join(out, "page/2/index.html"))
	for _, want := range []string{
		`href="p1/"`, `<base href="/">`, `<meta name="lerd-page" content="2">`,
		`<link rel="canonical" href="https://plugins.lerd.sh/page/2/">`,
		`<link rel="prev" href="https://plugins.lerd.sh/">`,
		`<title>Lerd Plugins: the home title · page 2</title>`,
		`<a href="./">01</a>`, `<a href="page/2/" aria-current="page">02</a>`,
	} {
		if !strings.Contains(second, want) {
			t.Errorf("second page lacks %s\n%s", want, second)
		}
	}
	// The base comes before anything with an address, or the stylesheet loads from /page/2/.
	if strings.Index(second, `<base href="/">`) > strings.Index(second, "<title>") {
		t.Error("the second page's base comes after its title")
	}
	if strings.Contains(second, `href="p2/"`) {
		t.Error("the second page repeats the first one's plugins")
	}
	// Six or fewer plugins make one page and no pager.
	out = t.TempDir()
	if err := writeHomePages([]byte(tmpl), catalog{Plugins: seven().Plugins[:6]}, out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "page")); err == nil {
		t.Error("six plugins made a second page")
	}
	if strings.Contains(read(t, filepath.Join(out, "index.html")), `<a href="page/`) {
		t.Error("a single page has a pager")
	}
}

func TestFilterPages_SplitIntoPages(t *testing.T) {
	out := t.TempDir()
	if err := writeFilterPages([]byte(tmpl), seven(), out); err != nil {
		t.Fatal(err)
	}
	second := read(t, filepath.Join(out, "author/one/page/2/index.html"))
	for _, want := range []string{`href="p1/"`, `<link rel="canonical" href="https://plugins.lerd.sh/author/one/page/2/">`, `<a href="author/one/">01</a>`, `<title>Lerd plugins by one · page 2</title>`} {
		if !strings.Contains(second, want) {
			t.Errorf("author page 2 lacks %s\n%s", want, second)
		}
	}
	if err := writeSitemap(seven(), out); err != nil {
		t.Fatal(err)
	}
	sm := read(t, filepath.Join(out, "sitemap.xml"))
	for _, want := range []string{"<loc>https://plugins.lerd.sh/page/2/</loc>", "<loc>https://plugins.lerd.sh/author/one/page/2/</loc>"} {
		if !strings.Contains(sm, want) {
			t.Errorf("sitemap lacks %s", want)
		}
	}
}
