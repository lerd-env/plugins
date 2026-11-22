package main

import "testing"

func TestLicenseName(t *testing.T) {
	for text, want := range map[string]string{
		"MIT License\n\nCopyright (c) 2026 A\n\nPermission is hereby granted, free of charge":                   "MIT",
		"Permission is hereby granted, free of charge, to any person":                                           "MIT",
		"                                 Apache License\n                           Version 2.0, January 2004": "Apache-2.0",
		"                    GNU GENERAL PUBLIC LICENSE\n                       Version 3, 29 June 2007":        "GPL-3.0",
		"                    GNU GENERAL PUBLIC LICENSE\n                       Version 2, June 1991":           "GPL-2.0",
		"                   GNU LESSER GENERAL PUBLIC LICENSE\n                       Version 3":                "LGPL-3.0",
		"Mozilla Public License Version 2.0":                                                                    "MPL-2.0",
		"ISC License\n\nCopyright":                                                                              "ISC",
		"BSD 3-Clause License\n\nCopyright":                                                                     "BSD-3-Clause",
		"This is free and unencumbered software released into the public domain.":                               "Unlicense",
		"All rights reserved. Do what the author says.":                                                         "",
	} {
		if got := licenseName(text); got != want {
			t.Errorf("licenseName(%.30q) = %q, want %q", text, got, want)
		}
	}
}

func TestLicenseFile(t *testing.T) {
	dir := t.TempDir()
	if f, name := licenseFile(dir); f != "" || name != "" {
		t.Errorf("no license: got %q %q", f, name)
	}
	write(t, dir, map[string]string{"LICENSE.md": "MIT License\n"})
	if f, name := licenseFile(dir); f != "LICENSE.md" || name != "MIT" {
		t.Errorf("got %q %q", f, name)
	}
}
