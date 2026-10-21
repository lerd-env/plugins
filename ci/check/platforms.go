package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"

	"gopkg.in/yaml.v3"
)

// knownPlatforms are the values Lerd accepts in `platforms`.
var knownPlatforms = map[string]bool{"linux": true, "macos": true}

// installableHere returns a folder the check's Linux lerd can install. A plugin
// that does not list linux is copied with its platforms key removed, so the
// rest of its manifest is still validated; aside reports that it was.
func installableHere(dir string) (string, bool, error) {
	path := filepath.Join(dir, "plugin.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return dir, false, nil
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil || len(doc.Content) == 0 {
		return dir, false, nil
	}
	root := doc.Content[0]
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value != "platforms" {
			continue
		}
		var list []string
		if err := root.Content[i+1].Decode(&list); err != nil || len(list) == 0 || slices.Contains(list, "linux") {
			return dir, false, nil
		}
		root.Content = append(root.Content[:i], root.Content[i+2:]...)
		out, err := yaml.Marshal(&doc)
		if err != nil {
			return dir, false, err
		}
		copyDir, err := os.MkdirTemp("", "lerd-plugin-")
		if err != nil {
			return dir, false, err
		}
		if cp, err := exec.Command("cp", "-a", dir+"/.", copyDir).CombinedOutput(); err != nil {
			os.RemoveAll(copyDir)
			return dir, false, fmt.Errorf("copying the plugin: %v: %s", err, cp)
		}
		return copyDir, true, os.WriteFile(filepath.Join(copyDir, "plugin.yaml"), out, 0o644)
	}
	return dir, false, nil
}
