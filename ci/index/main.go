// Command index builds the catalog Lerd and plugins.lerd.sh read: index.json
// and the images it lists, one listing per entry in plugins/.
//
//	go run ./ci/index -out _site
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lerd-env/plugins/ci/registry"
)

// listing is one plugin as Lerd's catalog reads it; the field names are the
// ones its Listing type decodes.
type listing struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Repo        string   `json:"repo"`
	Commit      string   `json:"commit"`
	Author      string   `json:"author,omitempty"`
	Category    string   `json:"category,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Images      []string `json:"images,omitempty"`
	AddedAt     string   `json:"added_at,omitempty"`
	UpdatedAt   string   `json:"updated_at,omitempty"`
	Stars       int      `json:"stars,omitempty"`
	Lerd        string   `json:"lerd,omitempty"`
	Requires    []string `json:"requires,omitempty"`
	Platforms   []string `json:"platforms,omitempty"`
}

type catalog struct {
	Plugins []listing `json:"plugins"`
}

func main() {
	plugins := flag.String("plugins", "plugins", "folder holding every listed entry")
	out := flag.String("out", "_site", "folder index.json and the images are written to")
	flag.Parse()
	if err := build(*plugins, ".", *out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// build writes the catalog for every entry in plugins, reading their dates
// from the registry's history in repoDir.
func build(plugins, repoDir, out string) error {
	paths, err := filepath.Glob(filepath.Join(plugins, "*.yaml"))
	if err != nil {
		return err
	}
	sort.Strings(paths)
	c := catalog{Plugins: []listing{}}
	for _, path := range paths {
		l, err := buildListing(path, repoDir, out)
		if err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(path), err)
		}
		c.Plugins = append(c.Plugins, l)
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(out, "index.json"), append(data, '\n'), 0o644)
}

func buildListing(path, repoDir, out string) (listing, error) {
	name, e, err := registry.ReadEntry(path)
	if err != nil {
		return listing{}, err
	}
	if p := e.Problems(name); len(p) > 0 {
		return listing{}, fmt.Errorf("%s", strings.Join(p, "; "))
	}
	dir, err := registry.FetchCommit(e.Repo, e.Commit)
	if err != nil {
		return listing{}, err
	}
	defer os.RemoveAll(dir)

	m, err := readManifest(dir)
	if err != nil {
		return listing{}, err
	}
	images, err := publishImages(dir, out, name, e)
	if err != nil {
		return listing{}, err
	}
	l := listing{
		Name: name, Description: e.Description, Repo: e.Repo, Commit: e.Commit,
		Author: e.Author, Category: e.Category, Tags: e.Tags, Images: images,
		Lerd: m.Lerd, Requires: m.requirements(dir), Platforms: m.Platforms,
	}
	rel, _ := filepath.Rel(repoDir, path)
	l.AddedAt = gitDate(repoDir, rel, true)
	l.UpdatedAt = gitDate(repoDir, rel, false)
	if stars, err := starsOf(e.Repo); err == nil {
		l.Stars = stars
	} else {
		fmt.Fprintf(os.Stderr, "%s: no star count: %v\n", name, err)
	}
	return l, nil
}

// publishImages copies the entry's images out of the approved commit, named by
// their order so the first is the preview, under a folder keyed on the commit
// so an update never serves a stale picture from a cache.
func publishImages(dir, out, name string, e registry.Entry) ([]string, error) {
	var rels []string
	for i, img := range e.Images {
		src := filepath.Join(dir, img)
		if !strings.HasPrefix(src, filepath.Clean(dir)+string(os.PathSeparator)) {
			return nil, fmt.Errorf("image %s points outside the repository", img)
		}
		rel := fmt.Sprintf("images/%s/%s/%d%s", name, e.Commit, i+1, strings.ToLower(filepath.Ext(img)))
		if err := copyFile(src, filepath.Join(out, rel)); err != nil {
			return nil, fmt.Errorf("image %s: %w", img, err)
		}
		rels = append(rels, rel)
	}
	return rels, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, in); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// gitDate is the day the entry was first added, or last changed, by the
// registry's own history.
func gitDate(repoDir, rel string, first bool) string {
	args := []string{"-C", repoDir, "log", "--format=%cs", "--", rel}
	if first {
		args = []string{"-C", repoDir, "log", "--diff-filter=A", "--format=%cs", "--", rel}
	}
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return ""
	}
	lines := strings.Fields(string(out))
	if len(lines) == 0 {
		return ""
	}
	if first {
		return lines[len(lines)-1]
	}
	return lines[0]
}

// starsOf asks GitHub for the repository's star count; a var so tests need no
// network. GITHUB_TOKEN, when set, only raises the rate limit.
var starsOf = func(repo string) (int, error) {
	api := "https://api.github.com/repos/" + strings.TrimPrefix(repo, "https://github.com/")
	req, err := http.NewRequest(http.MethodGet, api, nil)
	if err != nil {
		return 0, err
	}
	if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("%s", res.Status)
	}
	var info struct {
		Stars int `json:"stargazers_count"`
	}
	err = json.NewDecoder(res.Body).Decode(&info)
	return info.Stars, err
}
