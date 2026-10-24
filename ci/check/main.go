// Command check runs the automated part of SUBMITTING.md against the plugin
// entries a pull request adds or changes, and writes a markdown report.
//
//	go run ./ci/check -lerd /path/to/lerd plugins/<name>.yaml...
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/lerd-env/plugins/ci/registry"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

func main() {
	lerd := flag.String("lerd", "", "lerd binary that validates and installs each plugin")
	plugins := flag.String("plugins", "plugins", "folder holding every listed entry")
	out := flag.String("report", "", "file the markdown report is appended to (stdout when empty)")
	only := flag.String("only", "", "check one area: entry, "+strings.Join(areas, ", "))
	flag.Parse()
	if *lerd == "" || flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: check -lerd <binary> [-only <area>] plugins/<name>.yaml...")
		os.Exit(2)
	}
	if *only != "" && *only != "entry" && !slices.Contains(areas, *only) {
		fmt.Fprintf(os.Stderr, "unknown area %q, want entry, %s\n", *only, strings.Join(areas, ", "))
		os.Exit(2)
	}

	w := io.Writer(os.Stdout)
	if *out != "" {
		f, err := os.OpenFile(*out, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		defer f.Close()
		w = f
	}

	failed := false
	for _, path := range flag.Args() {
		name, r := checkEntry(path, *plugins, *lerd, *only)
		writeReport(w, name, *only, r)
		annotate(os.Stdout, name, r)
		failed = failed || len(r.Fail) > 0
	}
	if failed {
		os.Exit(1)
	}
}

// checkEntry checks one entry; only names a single area, or "" for all of them.
// The entry itself, the repository being public and the commit fetching are
// checked whatever the area, since nothing else can be judged without them.
func checkEntry(path, plugins, lerd, only string) (string, report) {
	var r report
	name, e, err := registry.ReadEntry(path)
	if err != nil {
		r.fail("%v", err)
		return name, r
	}
	if p := e.Problems(name); len(p) > 0 {
		r.Fail = append(r.Fail, p...)
		return name, r
	}
	// A revocation pulls the plugin, so there is nothing to install or judge.
	if !e.Listed() {
		r.flag("revoked: %s", e.Revoked.Reason)
		return name, r
	}
	if err := publicCheck(e.Repo); err != nil {
		r.fail("%v (rule 1)", err)
		return name, r
	}
	dir, err := registry.FetchCommit(e.Repo, e.Commit)
	if err != nil {
		r.fail("%v", err)
		return name, r
	}
	defer os.RemoveAll(dir)

	switch only {
	case "":
		found := inspect(dir, name, e)
		r.Fail = append(r.Fail, found.Fail...)
		r.Flag = append(r.Flag, found.Flag...)
		installAlongside(&r, lerd, plugins, name, dir)
	case "install":
		installAlongside(&r, lerd, plugins, name, dir)
	case "entry":
	default:
		inspectArea(&r, dir, name, e, only)
	}
	return name, r
}

// publicCheck is a var so tests can stand in for GitHub.
var publicCheck = checkPublic

// checkPublic asks GitHub whether the repository is public. GITHUB_TOKEN, when
// set, only raises the rate limit.
func checkPublic(repo string) error {
	api := "https://api.github.com/repos/" + strings.TrimPrefix(repo, "https://github.com/")
	req, err := http.NewRequest(http.MethodGet, api, nil)
	if err != nil {
		return err
	}
	if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return fmt.Errorf("%s does not exist or is private", repo)
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("asking GitHub about %s: %s", repo, res.Status)
	}
	var info struct {
		Private bool `json:"private"`
	}
	if err := json.NewDecoder(res.Body).Decode(&info); err != nil {
		return err
	}
	if info.Private {
		return fmt.Errorf("%s is private", repo)
	}
	return nil
}

// installAlongside installs every other listed plugin into a throwaway lerd,
// then this one, so lerd's own validator refuses a broken manifest, a newer
// lerd requirement or a name another plugin already ships (rule 18).
func installAlongside(r *report, lerd, plugins, name, dir string) {
	home, err := os.MkdirTemp("", "lerd-home-")
	if err != nil {
		r.fail("%v", err)
		return
	}
	defer os.RemoveAll(home)
	env := append(os.Environ(),
		"XDG_CONFIG_HOME="+filepath.Join(home, "config"),
		"XDG_DATA_HOME="+filepath.Join(home, "data"),
	)

	others, _ := filepath.Glob(filepath.Join(plugins, "*.yaml"))
	for _, path := range others {
		other, e, err := registry.ReadEntry(path)
		if err != nil || other == name {
			continue
		}
		odir, err := registry.FetchCommit(e.Repo, e.Commit)
		if err != nil {
			r.flag("could not fetch listed plugin %s to check for clashes: %v", other, err)
			continue
		}
		if out, err := installHere(lerd, env, odir); err != nil {
			r.flag("listed plugin %s no longer installs on this lerd: %s", other, out)
		}
		os.RemoveAll(odir)
	}
	idir, aside, err := installableHere(dir)
	if err != nil {
		r.fail("%v", err)
		return
	}
	if aside {
		defer os.RemoveAll(idir)
		r.flag("it lists only macos, so it was installed on Linux with platforms set aside; run it on a Mac before approving")
	}
	if out, err := lerdInstall(lerd, env, idir); err != nil {
		r.fail("lerd refused to install it: %s", out)
	}
}

// installHere installs a listed plugin the way installAlongside installs the
// submission, platforms set aside when it runs only on macOS.
func installHere(lerd string, env []string, dir string) (string, error) {
	idir, aside, err := installableHere(dir)
	if err != nil {
		return err.Error(), err
	}
	if aside {
		defer os.RemoveAll(idir)
	}
	return lerdInstall(lerd, env, idir)
}

func lerdInstall(lerd string, env []string, dir string) (string, error) {
	cmd := exec.Command(lerd, "plugin", "install", dir)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// annotate writes each finding as a GitHub workflow command, so it shows in the
// job's log and on the pull request rather than only on the summary page.
func annotate(w io.Writer, name string, r report) {
	esc := strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A")
	for _, f := range r.Fail {
		fmt.Fprintf(w, "::error title=%s::%s\n", name, esc.Replace(f))
	}
	for _, f := range r.Flag {
		fmt.Fprintf(w, "::warning title=%s::%s\n", name, esc.Replace(f))
	}
}

func writeReport(w io.Writer, name, only string, r report) {
	if only != "" {
		name += ": " + only
	}
	fmt.Fprintf(w, "## %s\n\n", name)
	if len(r.Fail) > 0 {
		fmt.Fprintln(w, "**Blocking**")
		fmt.Fprintln(w)
		for _, f := range r.Fail {
			fmt.Fprintf(w, "- %s\n", f)
		}
		fmt.Fprintln(w)
	}
	if len(r.Flag) > 0 {
		fmt.Fprintln(w, "**For the reviewer to read**")
		fmt.Fprintln(w)
		for _, f := range r.Flag {
			fmt.Fprintf(w, "- %s\n", f)
		}
		fmt.Fprintln(w)
	}
	if len(r.Fail) == 0 && (only == "" || only == "install") {
		fmt.Fprintln(w, "The automated checks pass. A maintainer still reads the code for rules 9 to 15 and 19 to 23 before approving.")
		fmt.Fprintln(w)
	}
}
