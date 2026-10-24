package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/lerd-env/plugins/ci/registry"
)

func write(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, body := range files {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		mode := os.FileMode(0o644)
		if strings.HasPrefix(rel, "bin/") {
			mode = 0o755
		}
		if err := os.WriteFile(path, []byte(body), mode); err != nil {
			t.Fatal(err)
		}
	}
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir, "-c", "user.email=t@t", "-c", "user.name=t"}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func commitAll(t *testing.T, dir string) string {
	t.Helper()
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-q", "-m", "x")
	return git(t, dir, "rev-parse", "HEAD")
}

// fixture is a registry holding one entry for a plugin in its own repository.
func fixture(t *testing.T) (registryDir string, commit string) {
	t.Helper()
	plugin := t.TempDir()
	git(t, plugin, "init", "-q")
	write(t, plugin, map[string]string{
		"plugin.yaml":   "api_version: 1\nname: qb\nversion: 1.2.0\nlerd: \">=1.37\"\nrequires: [jq]\nplatforms: [linux]\ncommands:\n  - {name: a, run: bin/a}\nhooks:\n  site.linked: bin/b\n",
		"bin/a":         "#!/usr/bin/env python3\n",
		"bin/b":         "#!/bin/sh\n",
		"docs/shot.png": "\x89PNG",
	})
	commit = commitAll(t, plugin)

	registryDir = t.TempDir()
	git(t, registryDir, "init", "-q")
	write(t, registryDir, map[string]string{
		"plugins/qb.yaml": "repo: file://" + plugin + "\ncommit: " + commit + "\nauthor: acme\ndescription: Queues\ncategory: Debugging\ntags: [queues]\nimages: [docs/shot.png]\n",
	})
	commitAll(t, registryDir)
	return registryDir, commit
}

func TestBuild(t *testing.T) {
	orig := registry.RepoPattern
	registry.RepoPattern = regexp.MustCompile(`.`)
	t.Cleanup(func() { registry.RepoPattern = orig })
	origStars := starsOf
	starsOf = func(repo string) (int, error) {
		if repo == lerdRepo {
			return 5000, nil
		}
		return 7, nil
	}
	t.Cleanup(func() { starsOf = origStars })

	reg, commit := fixture(t)
	out := t.TempDir()
	if err := build(filepath.Join(reg, "plugins"), reg, out); err != nil {
		t.Fatal(err)
	}
	var c catalog
	data, _ := os.ReadFile(filepath.Join(out, "index.json"))
	if err := json.Unmarshal(data, &c); err != nil || len(c.Plugins) != 1 {
		t.Fatalf("%v %s", err, data)
	}
	if c.LerdStars != 5000 {
		t.Errorf("lerd stars %d, want lerd's own count for the page header", c.LerdStars)
	}
	l := c.Plugins[0]
	today := time.Now().Format("2006-01-02")
	if l.Name != "qb" || l.Commit != commit || l.Author != "acme" || l.Category != "Debugging" || l.Stars != 7 || l.Lerd != ">=1.37" || l.Version != "1.2.0" {
		t.Errorf("listing %+v", l)
	}
	if strings.Join(l.Platforms, ",") != "linux" {
		t.Errorf("platforms %v", l.Platforms)
	}
	if strings.Join(l.Requires, ",") != "/bin/sh,jq,python3" {
		t.Errorf("requires %v", l.Requires)
	}
	if l.AddedAt != today || l.UpdatedAt != today {
		t.Errorf("dates %s %s", l.AddedAt, l.UpdatedAt)
	}
	want := "images/qb/" + commit + "/1.png"
	if len(l.Images) != 1 || l.Images[0] != want {
		t.Fatalf("images %v", l.Images)
	}
	if _, err := os.Stat(filepath.Join(out, want)); err != nil {
		t.Errorf("the image is published beside the index: %v", err)
	}
}

// A listing whose commit can't be fetched would be served as installable and
// fail for everyone, so the build stops and names it.
func TestBuild_RefusesAnEntryItCannotFetch(t *testing.T) {
	orig := registry.RepoPattern
	registry.RepoPattern = regexp.MustCompile(`.`)
	t.Cleanup(func() { registry.RepoPattern = orig })
	reg := t.TempDir()
	git(t, reg, "init", "-q")
	write(t, reg, map[string]string{"plugins/ghost.yaml": "repo: file:///nonexistent\ncommit: 0123456789abcdef0123456789abcdef01234567\nauthor: a\ndescription: d\n"})
	commitAll(t, reg)
	if err := build(filepath.Join(reg, "plugins"), reg, t.TempDir()); err == nil || !strings.Contains(err.Error(), "ghost") {
		t.Fatalf("err = %v", err)
	}
}

// A revoked entry leaves the listings and goes into the revoked list with its
// reason, without its commit ever being fetched again.
func TestBuild_PublishesARevocation(t *testing.T) {
	orig := registry.RepoPattern
	registry.RepoPattern = regexp.MustCompile(`.`)
	t.Cleanup(func() { registry.RepoPattern = orig })
	origStars := starsOf
	starsOf = func(string) (int, error) { return 0, nil }
	t.Cleanup(func() { starsOf = origStars })

	reg := t.TempDir()
	write(t, reg, map[string]string{"plugins/bad.yaml": "repo: https://example.test/gone\ncommit: 0123456789abcdef0123456789abcdef01234567\nauthor: a\ndescription: d\nrevoked:\n  reason: sends .env away\n  at: \"2026-10-24\"\n"})
	out := t.TempDir()
	if err := build(filepath.Join(reg, "plugins"), reg, out); err != nil {
		t.Fatal(err)
	}
	var c catalog
	data, _ := os.ReadFile(filepath.Join(out, "index.json"))
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatal(err)
	}
	if len(c.Plugins) != 0 || len(c.Revoked) != 1 || c.Revoked[0].Name != "bad" || c.Revoked[0].Reason != "sends .env away" {
		t.Fatalf("index %s", data)
	}
}
