// Package registry reads the entries in plugins/ and fetches the commits they
// approve, for the submission check and the index builder alike.
package registry

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// Entry is one listing in plugins/<name>.yaml.
type Entry struct {
	Repo        string   `yaml:"repo"`
	Commit      string   `yaml:"commit"`
	Author      string   `yaml:"author"`
	Description string   `yaml:"description"`
	Category    string   `yaml:"category,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`
	// Images are paths inside the plugin repository at the approved commit,
	// the first one being the preview.
	Images []string `yaml:"images,omitempty"`
	// Revoked pulls the plugin for a security problem: Lerd stops it on every
	// machine that has a commit it covers and tells the user why.
	Revoked *Revocation `yaml:"revoked,omitempty"`
}

// Revocation is why a plugin was pulled and which approved commits it covers.
// With no commits it covers every install; listing them lets a fixed commit be
// approved again while the ones before it stay stopped.
type Revocation struct {
	Reason  string   `yaml:"reason" json:"reason"`
	At      string   `yaml:"at" json:"at"`
	Commits []string `yaml:"commits,omitempty" json:"commits,omitempty"`
}

// Listed reports whether the entry's commit is offered at all: not when its
// revocation covers it.
func (e Entry) Listed() bool {
	if e.Revoked == nil {
		return true
	}
	return len(e.Revoked.Commits) > 0 && !slices.Contains(e.Revoked.Commits, e.Commit)
}

var (
	nameRe = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	// RepoPattern is a var so tests can list a local repository.
	RepoPattern = regexp.MustCompile(`^https://github\.com/[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
	commitRe    = regexp.MustCompile(`^[0-9a-f]{40}$`)
	dateRe      = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

// ReadEntry parses an entry strictly, so a misspelt key fails instead of
// silently dropping what it meant to say.
func ReadEntry(path string) (string, Entry, error) {
	name := strings.TrimSuffix(filepath.Base(path), ".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return name, Entry{}, err
	}
	var e Entry
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&e); err != nil {
		return name, Entry{}, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	return name, e, nil
}

// Problems lists what is wrong with the entry itself, before anything is fetched.
func (e Entry) Problems(name string) []string {
	var out []string
	if !nameRe.MatchString(name) {
		out = append(out, fmt.Sprintf("entry name %q must be lowercase letters, digits and dashes", name))
	}
	if !RepoPattern.MatchString(e.Repo) {
		out = append(out, fmt.Sprintf("repo %q must be https://github.com/<owner>/<repo>", e.Repo))
	}
	if !commitRe.MatchString(e.Commit) {
		out = append(out, fmt.Sprintf("commit %q must be the full 40-character hash", e.Commit))
	}
	if strings.TrimSpace(e.Author) == "" {
		out = append(out, "author is required")
	}
	if strings.TrimSpace(e.Description) == "" {
		out = append(out, "description is required")
	}
	if r := e.Revoked; r != nil {
		if strings.TrimSpace(r.Reason) == "" {
			out = append(out, "revoked needs a reason, which every user who has the plugin is shown")
		}
		if !dateRe.MatchString(r.At) {
			out = append(out, fmt.Sprintf("revoked at %q must be a YYYY-MM-DD date", r.At))
		}
		for _, c := range r.Commits {
			if !commitRe.MatchString(c) {
				out = append(out, fmt.Sprintf("revoked commit %q must be the full 40-character hash", c))
			}
		}
	}
	return out
}
