package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lerd-env/plugins/ci/registry"
	_ "golang.org/x/image/webp"
	"gopkg.in/yaml.v3"
)

// report holds what blocks a submission and what a reviewer should look at.
type report struct {
	Fail []string
	Flag []string
}

func (r *report) fail(format string, a ...any) { r.Fail = append(r.Fail, fmt.Sprintf(format, a...)) }
func (r *report) flag(format string, a ...any) { r.Flag = append(r.Flag, fmt.Sprintf(format, a...)) }

// inspect checks a plugin checked out at its approved commit in dir.
func inspect(dir, name string, e registry.Entry) report {
	var r report
	if !anyFile(dir, "LICENSE", "LICENSE.md", "LICENSE.txt", "COPYING") {
		r.fail("no LICENSE file (rule 2)")
	}
	readme, ok := readFirst(dir, "README.md", "README", "readme.md")
	if !ok {
		r.fail("no README (rule 3)")
	}
	checkManifest(&r, dir, name, len(e.Images) > 0)
	checkServices(&r, dir, readme)
	checkImages(&r, dir, e.Images)
	scanFiles(&r, dir)
	return r
}

// lowerBoundRe accepts only a lower bound, so a new lerd never strands the plugin.
var lowerBoundRe = regexp.MustCompile(`^>=\s*\d+\.\d+(\.\d+)?$`)

func checkManifest(r *report, dir, name string, hasImages bool) {
	data, err := os.ReadFile(filepath.Join(dir, "plugin.yaml"))
	if err != nil {
		r.fail("no plugin.yaml")
		return
	}
	var m struct {
		Name      string         `yaml:"name"`
		Lerd      string         `yaml:"lerd"`
		UI        map[string]any `yaml:"ui"`
		Platforms []string       `yaml:"platforms"`
	}
	if err := yaml.Unmarshal(data, &m); err != nil {
		r.fail("plugin.yaml: %v", err)
		return
	}
	if m.Name != name {
		r.fail("plugin.yaml name %q must match the entry name %q", m.Name, name)
	}
	switch {
	case m.Lerd == "":
		r.fail("plugin.yaml must declare lerd, the oldest release it runs on (rule 4)")
	case !lowerBoundRe.MatchString(strings.TrimSpace(m.Lerd)):
		r.fail("plugin.yaml lerd %q must be a lower bound only, like \">=1.37\" (rule 4)", m.Lerd)
	}
	for _, p := range m.Platforms {
		if !knownPlatforms[p] {
			r.fail("plugin.yaml platforms %q is not one Lerd knows, use linux or macos (Windows runs Lerd in WSL, so linux covers it)", p)
		}
	}
	// A plugin with something to see is listed with a picture of it; one that
	// only adds commands or hooks has nothing to capture and shows its icon.
	if len(m.UI) > 0 && !hasImages {
		r.fail("the plugin adds a page, site tab or page script, so its entry lists at least one image to show it (rule 20)")
	}
}

func checkServices(r *report, dir, readme string) {
	paths, _ := filepath.Glob(filepath.Join(dir, "services", "*.yaml"))
	for _, path := range paths {
		rel := "services/" + filepath.Base(path)
		data, err := os.ReadFile(path)
		if err != nil {
			r.fail("%s: %v", rel, err)
			continue
		}
		var svc struct {
			Image    string           `yaml:"image"`
			Versions []serviceVersion `yaml:"versions"`
		}
		if err := yaml.Unmarshal(data, &svc); err != nil {
			r.fail("%s: %v", rel, err)
			continue
		}
		seen := map[string]bool{"": true}
		for _, img := range append([]string{svc.Image}, versionImages(svc.Versions)...) {
			if !seen[img] {
				seen[img] = true
				checkImage(r, rel, img, readme)
			}
		}
	}
}

type serviceVersion struct {
	Image string `yaml:"image"`
}

func versionImages(vs []serviceVersion) []string {
	out := make([]string, 0, len(vs))
	for _, v := range vs {
		out = append(out, v.Image)
	}
	return out
}

func checkImage(r *report, rel, image, readme string) {
	repo, tag := splitImage(image)
	switch {
	case tag == "":
		r.fail("%s: image %s has no tag, pin a version (rule 17)", rel, image)
	case tag == "latest":
		r.fail("%s: image %s uses latest, pin a version (rule 17)", rel, image)
	}
	if !strings.Contains(readme, repo) {
		r.fail("%s: the README must name the image %s and who publishes it (rule 16)", rel, repo)
	}
}

// splitImage separates an image reference into its repository and its tag or
// digest; a port in the registry host is not a tag.
func splitImage(image string) (repo, tag string) {
	if i := strings.Index(image, "@"); i != -1 {
		return image[:i], image[i+1:]
	}
	slash := strings.LastIndex(image, "/")
	if i := strings.LastIndex(image, ":"); i > slash {
		return image[:i], image[i+1:]
	}
	return image, ""
}

var imageExts = map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".webp": true}

// Every Browse card in Lerd loads its plugin's first image, and GitHub Pages
// caps the catalog site at 1 GB, so images are kept to what a page of cards
// can carry: a full dashboard screenshot fits well inside these.
const (
	maxImages     = 6
	maxImageBytes = 1 << 20
	maxImageWidth = 2560
)

func checkImages(r *report, dir string, images []string) {
	if len(images) > maxImages {
		r.fail("the entry lists %d images, at most %d (rule 20)", len(images), maxImages)
	}
	for _, img := range images {
		if !imageExts[strings.ToLower(filepath.Ext(img))] {
			r.fail("image %s must be a png, jpg or webp", img)
			continue
		}
		if !insideFile(dir, img) {
			r.fail("image %s is not in the repository at this commit", img)
			continue
		}
		checkImageFile(r, filepath.Join(dir, img), img)
	}
}

func checkImageFile(r *report, path, rel string) {
	info, err := os.Stat(path)
	if err != nil {
		r.fail("image %s: %v", rel, err)
		return
	}
	if info.Size() > maxImageBytes {
		r.fail("image %s is %.1f MB, keep each image under 1 MB (rule 20)", rel, float64(info.Size())/(1<<20))
	}
	f, err := os.Open(path)
	if err != nil {
		r.fail("image %s: %v", rel, err)
		return
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		r.fail("image %s is not a readable png, jpg or webp: %v", rel, err)
		return
	}
	if cfg.Width > maxImageWidth {
		r.fail("image %s is %d px wide, at most %d (rule 20)", rel, cfg.Width, maxImageWidth)
	}
}

func insideFile(dir, rel string) bool {
	abs := filepath.Join(dir, rel)
	if !strings.HasPrefix(abs, filepath.Clean(dir)+string(os.PathSeparator)) {
		return false
	}
	info, err := os.Stat(abs)
	return err == nil && info.Mode().IsRegular()
}

func anyFile(dir string, names ...string) bool {
	_, ok := readFirst(dir, names...)
	return ok
}

func readFirst(dir string, names ...string) (string, bool) {
	for _, n := range names {
		if data, err := os.ReadFile(filepath.Join(dir, n)); err == nil {
			return string(data), true
		}
	}
	return "", false
}
