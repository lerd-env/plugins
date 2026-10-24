package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const tmpl = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>Lerd Plugins</title>
  <meta name="description" content="Plugins for Lerd.">
  <meta property="og:title" content="Lerd Plugins">
</head>
<body></body>
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
		`<title>sail · Lerd Plugins</title>`,
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
