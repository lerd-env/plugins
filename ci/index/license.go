package main

import (
	"os"
	"path/filepath"
	"strings"
)

// licenses name a license by words only its text carries, most specific first:
// the GPL's name is inside the LGPL's, so the LGPL is tried before it.
var licenses = []struct{ name, needs, also string }{
	{"AGPL-3.0", "gnu affero general public license", ""},
	{"LGPL-3.0", "gnu lesser general public license", "version 3"},
	{"LGPL-2.1", "gnu lesser general public license", "version 2.1"},
	{"GPL-3.0", "gnu general public license", "version 3"},
	{"GPL-2.0", "gnu general public license", "version 2"},
	{"Apache-2.0", "apache license", "version 2.0"},
	{"MPL-2.0", "mozilla public license version 2.0", ""},
	{"BSD-3-Clause", "bsd 3-clause", ""},
	{"BSD-2-Clause", "bsd 2-clause", ""},
	{"ISC", "isc license", ""},
	{"Unlicense", "unencumbered software released into the public domain", ""},
	{"MIT", "mit license", ""},
	{"MIT", "permission is hereby granted, free of charge", ""},
}

// licenseName is the license a license file's text is, "" when it is none the
// catalog recognises.
func licenseName(text string) string {
	t := strings.Join(strings.Fields(strings.ToLower(text)), " ")
	for _, l := range licenses {
		if strings.Contains(t, l.needs) && strings.Contains(t, l.also) {
			return l.name
		}
	}
	return ""
}

// licenseFile finds the license at the root of the approved commit: its path,
// and its name when the catalog recognises it.
func licenseFile(dir string) (file, name string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", ""
	}
	for _, f := range entries {
		base := strings.ToLower(f.Name())
		if f.IsDir() || !(strings.HasPrefix(base, "license") || strings.HasPrefix(base, "licence") || strings.HasPrefix(base, "copying")) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, f.Name()))
		if err != nil {
			continue
		}
		return f.Name(), licenseName(string(data))
	}
	return "", ""
}
