package registry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestReadEntry(t *testing.T) {
	dir := write(t, t.TempDir(), map[string]string{
		"queue-board.yaml": "repo: https://github.com/acme/queue-board\ncommit: " + goodCommit + "\nauthor: acme\ndescription: Queued jobs\n",
		"typo.yaml":        "repo: https://github.com/acme/x\ncommit: " + goodCommit + "\nauthor: acme\ndescription: x\ncomit: x\n",
	})
	name, e, err := ReadEntry(filepath.Join(dir, "queue-board.yaml"))
	if err != nil || name != "queue-board" || e.Commit != goodCommit {
		t.Fatalf("got %q %+v %v", name, e, err)
	}
	if _, _, err := ReadEntry(filepath.Join(dir, "typo.yaml")); err == nil || !strings.Contains(err.Error(), "comit") {
		t.Errorf("a misspelt key must be named, got %v", err)
	}
}

func TestEntryProblems(t *testing.T) {
	good := Entry{Repo: "https://github.com/acme/qb", Commit: goodCommit, Author: "acme", Description: "d"}
	if p := good.Problems("qb"); len(p) != 0 {
		t.Fatalf("unexpected %v", p)
	}
	cases := map[string]struct {
		name string
		e    Entry
	}{
		"Qb":          {"Qb", good},
		"github.com":  {"qb", Entry{Repo: "https://gitlab.com/acme/qb", Commit: goodCommit, Author: "a", Description: "d"}},
		"40":          {"qb", Entry{Repo: good.Repo, Commit: "main", Author: "a", Description: "d"}},
		"author":      {"qb", Entry{Repo: good.Repo, Commit: goodCommit, Description: "d"}},
		"description": {"qb", Entry{Repo: good.Repo, Commit: goodCommit, Author: "a"}},
	}
	for want, c := range cases {
		if p := c.e.Problems(c.name); !has(p, want) {
			t.Errorf("%+v: problems %v, want one naming %q", c.e, p, want)
		}
	}
}

// A revocation without its reason or a dated record would reach users as an
// unexplained stop, so the entry refuses it.
func TestEntryProblems_Revocation(t *testing.T) {
	base := Entry{Repo: "https://github.com/acme/qb", Commit: goodCommit, Author: "a", Description: "d"}
	ok := base
	ok.Revoked = &Revocation{Reason: "sends .env away", At: "2026-10-24"}
	if p := ok.Problems("qb"); len(p) != 0 {
		t.Fatalf("a full revocation: %v", p)
	}
	for want, r := range map[string]Revocation{
		"reason": {At: "2026-10-24"},
		"YYYY":   {Reason: "x", At: "yesterday"},
		"hash":   {Reason: "x", At: "2026-10-24", Commits: []string{"abc"}},
	} {
		e := base
		e.Revoked = &r
		if p := e.Problems("qb"); !has(p, want) {
			t.Errorf("%+v: problems %v, want one naming %q", r, p, want)
		}
	}
}

// Covering no commits pulls every install; covering some lets a fixed commit be
// listed again while the ones named stay pulled.
func TestEntryListed(t *testing.T) {
	fixed := "1111111111111111111111111111111111111111"
	e := Entry{Commit: goodCommit}
	if !e.Listed() {
		t.Error("an entry with no revocation is listed")
	}
	e.Revoked = &Revocation{Reason: "x", At: "2026-10-24"}
	if e.Listed() {
		t.Error("a revocation covering every commit unlists it")
	}
	e.Revoked.Commits = []string{goodCommit}
	if e.Listed() {
		t.Error("the entry's own commit is covered, so it is not listed")
	}
	e.Commit = fixed
	if !e.Listed() {
		t.Error("a fixed commit outside the revocation is listed again")
	}
}
