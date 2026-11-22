package main

import (
	"bytes"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/lerd-env/plugins/ci/registry"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// readmeImages are the picture types a readme may show. SVG is left out: served
// from the site's own address, one could run script there.
var readmeImages = map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true}

// readmeHTML renders the README at the root of the approved commit for the
// plugin's page, "" when it has none.
func readmeHTML(dir, out, name string, e registry.Entry) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, f := range entries {
		if !f.IsDir() && strings.EqualFold(f.Name(), "README.md") {
			src, err := os.ReadFile(filepath.Join(dir, f.Name()))
			if err != nil {
				return "", err
			}
			return renderReadme(src, dir, out, name, e)
		}
	}
	return "", nil
}

// renderReadme renders the way Lerd's dashboard renders plugin docs: raw HTML
// is dropped, the readme's images are copied beside the plugin's screenshots,
// and links leave the site only to the web or to the repository at the
// approved commit, marked as the author's rather than ours.
func renderReadme(src []byte, dir, out, name string, e registry.Entry) (string, error) {
	t := &readmeLinks{dir: dir, name: name, e: e}
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(parser.WithAutoHeadingID(), parser.WithASTTransformers(util.Prioritized(t, 100))),
	)
	var buf bytes.Buffer
	if err := md.Convert(src, &buf); err != nil {
		return "", err
	}
	for _, rel := range t.images {
		if err := copyFile(filepath.Join(dir, filepath.FromSlash(rel)), filepath.Join(out, t.imageURL(rel))); err != nil {
			return "", err
		}
	}
	return buf.String(), nil
}

type readmeLinks struct {
	dir, name string
	e         registry.Entry
	images    []string
}

func (t *readmeLinks) imageURL(rel string) string {
	return "images/" + t.name + "/" + t.e.Commit + "/readme/" + rel
}

// repoPath is a relative address inside the repository, "" for anything else:
// another scheme, or a path climbing out of it.
func repoPath(dest string) string {
	if dest == "" || strings.Contains(strings.SplitN(dest, "/", 2)[0], ":") {
		return ""
	}
	p := path.Clean(strings.TrimPrefix(dest, "/"))
	if p == "." || p == ".." || strings.HasPrefix(p, "../") {
		return ""
	}
	return p
}

func isWeb(dest string) bool {
	return strings.HasPrefix(dest, "https://") || strings.HasPrefix(dest, "http://")
}

var outbound = []byte("nofollow ugc noopener")

func (t *readmeLinks) Transform(doc *ast.Document, reader text.Reader, _ parser.Context) {
	source := reader.Source()
	// The readme's own title repeats the page's heading, the plugin's name.
	if h, ok := doc.FirstChild().(*ast.Heading); ok && h.Level == 1 {
		doc.RemoveChild(doc, h)
	}
	// The page's one h1 is that name, so a readme with more h1s steps down under it.
	shift := false
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if h, ok := n.(*ast.Heading); ok && entering && h.Level == 1 {
			shift = true
		}
		return ast.WalkContinue, nil
	})
	var drop []ast.Node
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch v := n.(type) {
		case *ast.Heading:
			if shift && v.Level < 6 {
				v.Level++
			}
		case *ast.Image:
			// Pictures from elsewhere would let a third party see every visit.
			rel := repoPath(string(v.Destination))
			if rel == "" || isWeb(string(v.Destination)) || !readmeImages[strings.ToLower(path.Ext(rel))] {
				drop = append(drop, v)
				return ast.WalkSkipChildren, nil
			}
			// The carousel above already shows the listing's own screenshots.
			if slices.ContainsFunc(t.e.Images, func(img string) bool { return path.Clean(img) == rel }) {
				drop = append(drop, v)
				return ast.WalkSkipChildren, nil
			}
			if fi, err := os.Stat(filepath.Join(t.dir, filepath.FromSlash(rel))); err != nil || !fi.Mode().IsRegular() {
				drop = append(drop, v)
				return ast.WalkSkipChildren, nil
			}
			t.images = append(t.images, rel)
			v.Destination = []byte(t.imageURL(rel))
		case *ast.Link:
			dest := string(v.Destination)
			if strings.HasPrefix(dest, "#") {
				return ast.WalkContinue, nil
			}
			if isWeb(dest) {
				v.SetAttributeString("rel", outbound)
				return ast.WalkContinue, nil
			}
			file, anchor, _ := strings.Cut(dest, "#")
			rel := repoPath(file)
			if rel == "" || !strings.HasPrefix(t.e.Repo, "https://github.com/") {
				drop = append(drop, v)
				return ast.WalkContinue, nil
			}
			dest = t.e.Repo + "/blob/" + t.e.Commit + "/" + rel
			if anchor != "" {
				dest += "#" + anchor
			}
			v.Destination = []byte(dest)
			v.SetAttributeString("rel", outbound)
		case *ast.AutoLink:
			if v.AutoLinkType == ast.AutoLinkURL && isWeb(string(v.URL(source))) {
				v.SetAttributeString("rel", outbound)
				return ast.WalkContinue, nil
			}
			drop = append(drop, v)
		}
		return ast.WalkContinue, nil
	})
	// A dropped link keeps its words; a dropped image goes, alt text and all.
	for _, n := range drop {
		parent := n.Parent()
		if auto, ok := n.(*ast.AutoLink); ok {
			parent.InsertBefore(parent, n, ast.NewString(auto.Label(source)))
		}
		if _, link := n.(*ast.Link); link {
			for c := n.FirstChild(); c != nil; {
				next := c.NextSibling()
				parent.InsertBefore(parent, n, c)
				c = next
			}
		}
		parent.RemoveChild(parent, n)
	}
}
