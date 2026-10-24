package main

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// manifest is the part of plugin.yaml the catalog shows before an install.
type manifest struct {
	Version   string                 `yaml:"version"`
	Lerd      string                 `yaml:"lerd"`
	Requires  []string               `yaml:"requires"`
	Platforms []string               `yaml:"platforms"`
	Commands  []struct{ Run string } `yaml:"commands"`
	SiteCmds  []struct{ Run string } `yaml:"site_commands"`
	Hooks     map[string]string      `yaml:"hooks"`
	MCP       struct {
		Tools []struct{ Run string } `yaml:"tools"`
	} `yaml:"mcp"`
	API      []struct{ Run string } `yaml:"api"`
	Provides map[string]string      `yaml:"provides"`
}

func readManifest(dir string) (manifest, error) {
	var m manifest
	data, err := os.ReadFile(filepath.Join(dir, "plugin.yaml"))
	if err != nil {
		return m, err
	}
	return m, yaml.Unmarshal(data, &m)
}

// requirements mirrors how Lerd reads them before installing: the interpreter
// each executable's first line names, plus what the manifest declares.
func (m manifest) requirements(dir string) []string {
	var runs []string
	for _, group := range [][]struct{ Run string }{m.Commands, m.SiteCmds, m.MCP.Tools, m.API} {
		for _, c := range group {
			runs = append(runs, c.Run)
		}
	}
	for _, run := range m.Hooks {
		runs = append(runs, run)
	}
	for _, run := range m.Provides {
		runs = append(runs, run)
	}
	seen := map[string]bool{}
	for _, run := range runs {
		if in := interpreter(filepath.Join(dir, run)); in != "" {
			seen[in] = true
		}
	}
	for _, r := range m.Requires {
		seen[r] = true
	}
	out := make([]string, 0, len(seen))
	for r := range seen {
		out = append(out, r)
	}
	sort.Strings(out)
	return out
}

func interpreter(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	line, _ := bufio.NewReader(f).ReadString('\n')
	if !strings.HasPrefix(line, "#!") {
		return ""
	}
	fields := strings.Fields(strings.TrimPrefix(line, "#!"))
	if len(fields) == 0 {
		return ""
	}
	if filepath.Base(fields[0]) != "env" {
		return fields[0]
	}
	for _, f := range fields[1:] {
		if !strings.HasPrefix(f, "-") && !strings.Contains(f, "=") {
			return f
		}
	}
	return ""
}
