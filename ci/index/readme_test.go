package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lerd-env/plugins/ci/registry"
)

const readmeCommit = "0123456789abcdef0123456789abcdef01234567"

func TestRenderReadme(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	write(t, dir, map[string]string{"docs/shot.png": "\x89PNG"})
	src := `# qb for Lerd

Intro with [Lerd](https://lerd.sh), [the guide](docs/GUIDE.md), [top](#usage) and [mail](mailto:a@b.c).

<script>alert(1)</script>

[bad](javascript:alert(1))

![shot](docs/shot.png) ![ext](https://tracker.example/x.png) ![escape](../../etc/passwd.png)

# Usage

## Options
`
	got, err := renderReadme([]byte(src), dir, out, "qb", registry.Entry{Repo: "https://github.com/acme/qb", Commit: readmeCommit})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`<a href="https://lerd.sh" rel="nofollow ugc noopener">Lerd</a>`,
		`<a href="https://github.com/acme/qb/blob/` + readmeCommit + `/docs/GUIDE.md" rel="nofollow ugc noopener">the guide</a>`,
		`<a href="#usage">top</a>`,
		`<img src="images/qb/` + readmeCommit + `/readme/docs/shot.png" alt="shot">`,
		// The page's own h1 is the plugin's name, so the readme's sections step down under it.
		`<h2 id="usage">Usage</h2>`,
		`<h3 id="options">Options</h3>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("readme lacks %s\n%s", want, got)
		}
	}
	for _, bad := range []string{"qb for Lerd", "<script", "javascript:", "mailto:", "tracker.example", "passwd"} {
		if strings.Contains(got, bad) {
			t.Errorf("readme keeps %q\n%s", bad, got)
		}
	}
	// A dropped link keeps its words.
	if !strings.Contains(got, "and mail.") {
		t.Errorf("the mail link's text went with it\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(out, "images/qb", readmeCommit, "readme/docs/shot.png")); err != nil {
		t.Errorf("the readme's image was not copied: %v", err)
	}
}

// A repository off GitHub has no address for a file at a commit, so a relative
// link keeps only its words.
func TestRenderReadme_RelativeLinkOffGitHub(t *testing.T) {
	got, err := renderReadme([]byte("See [the guide](GUIDE.md)."), t.TempDir(), t.TempDir(), "qb", registry.Entry{Repo: "file:///x", Commit: readmeCommit})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "<a") || !strings.Contains(got, "See the guide.") {
		t.Errorf("got %s", got)
	}
}

func TestReadmeHTML_NoReadme(t *testing.T) {
	got, err := readmeHTML(t.TempDir(), t.TempDir(), "qb", registry.Entry{Commit: readmeCommit})
	if err != nil || got != "" {
		t.Errorf("got %q, %v; want nothing", got, err)
	}
}

func TestPluginPage_CarriesTheReadme(t *testing.T) {
	page, err := pluginPage([]byte(tmpl), listing{Name: "qb", Readme: "<p>From the readme</p>", Repo: "https://github.com/acme/qb", Commit: readmeCommit})
	if err != nil {
		t.Fatal(err)
	}
	// It says which commit the readme is read at, the one Lerd installs.
	want := `<div class="wrap" id="plugin-docs"><p class="label">README at reviewed commit <a href="https://github.com/acme/qb/commit/` + readmeCommit + `"><code>0123456</code></a></p><p>From the readme</p></div>`
	if !strings.Contains(string(page), want) {
		t.Errorf("no readme\n%s", page)
	}
	// A plugin without a readme gets no label either.
	page, _ = pluginPage([]byte(tmpl), listing{Name: "qb", Repo: "https://github.com/acme/qb", Commit: readmeCommit})
	if !strings.Contains(string(page), `<div class="wrap" id="plugin-docs"></div>`) {
		t.Errorf("an empty readme got a label\n%s", page)
	}
}

// A picture the carousel above already shows is not shown twice.
func TestRenderReadme_SkipsTheListingsOwnScreenshots(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, map[string]string{"docs/shot.png": "\x89PNG", "docs/other.png": "\x89PNG"})
	got, err := renderReadme([]byte("![a](docs/shot.png)\n\n![b](./docs/other.png)"), dir, t.TempDir(), "qb", registry.Entry{Commit: readmeCommit, Images: []string{"docs/shot.png"}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "shot.png") || !strings.Contains(got, "readme/docs/other.png") {
		t.Errorf("got %s", got)
	}
}
