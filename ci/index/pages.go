package main

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// siteURL is where plugins.lerd.sh serves the catalog, for the absolute links
// social cards need.
const siteURL = "https://plugins.lerd.sh"

// reservedNames are the folders the site itself serves, which a plugin's own
// page must not replace.
var reservedNames = map[string]bool{"images": true}

// The home page's own description and social tags, replaced on a plugin page.
var homeTags = regexp.MustCompile(`(?m)^\s*<meta (name="description"|property="og:[a-z:]+"|name="twitter:[a-z:]+") content="[^"]*">\n`)

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
		dir := filepath.Join(out, l.Name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, "index.html"), page, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func pluginPage(tmpl []byte, l listing) ([]byte, error) {
	s := string(tmpl)
	if !strings.Contains(s, "<title>Lerd Plugins</title>") {
		return nil, fmt.Errorf("the site template has no <title>Lerd Plugins</title> to replace")
	}
	e := html.EscapeString
	url := siteURL + "/" + l.Name + "/"
	image := siteURL + "/og.png"
	if len(l.Images) > 0 {
		image = siteURL + "/" + l.Images[0]
	}
	title := l.Name + ", a Lerd plugin"
	if l.Author != "" {
		title += " by " + l.Author
	}
	tags := strings.Join([]string{
		// The page lives one folder down, so its relative links resolve from the root.
		`<base href="/">`,
		fmt.Sprintf(`<meta name="description" content="%s">`, e(l.Description)),
		fmt.Sprintf(`<link rel="canonical" href="%s">`, e(url)),
		`<meta property="og:type" content="website">`,
		`<meta property="og:site_name" content="Lerd Plugins">`,
		fmt.Sprintf(`<meta property="og:url" content="%s">`, e(url)),
		fmt.Sprintf(`<meta property="og:title" content="%s">`, e(title)),
		fmt.Sprintf(`<meta property="og:description" content="%s">`, e(l.Description)),
		fmt.Sprintf(`<meta property="og:image" content="%s">`, e(image)),
		`<meta name="twitter:card" content="summary_large_image">`,
		fmt.Sprintf(`<meta name="twitter:title" content="%s">`, e(title)),
		fmt.Sprintf(`<meta name="twitter:description" content="%s">`, e(l.Description)),
		fmt.Sprintf(`<meta name="twitter:image" content="%s">`, e(image)),
		fmt.Sprintf(`<meta name="lerd-plugin" content="%s">`, e(l.Name)),
	}, "\n  ")
	s = homeTags.ReplaceAllString(s, "")
	s = strings.Replace(s, "<title>Lerd Plugins</title>", fmt.Sprintf("<title>%s · Lerd Plugins</title>\n  %s", e(l.Name), tags), 1)
	return []byte(s), nil
}
