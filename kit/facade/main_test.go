// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors

package main

import "testing"

// A package counts as used only where code uses it: a comment or a string that names it once
// had an import written that the file didn't use, and the server didn't build.
func TestUsesPackage(t *testing.T) {
	for _, c := range []struct {
		text, name string
		want       bool
	}{
		{"type A struct {\n\tC wittyplay.Console\n}\n", "wittyplay", true},
		{"// Like wittyplay.Console, but not.\ntype A struct{}\n", "wittyplay", false},
		{"var s = \"wittyplay.Console\"\n", "wittyplay", false},
		{"type A struct {\n\tX other.wittyplay\n}\n", "wittyplay", false},
		{"*wittyplay.Console", "wittyplay", true},
		{"func(netip.Addr)", "netip", true},
		{"[]byte", "netip", false},
	} {
		if got := usesPackage(c.text, c.name); got != c.want {
			t.Errorf("%q uses %s: %v, want %v", c.text, c.name, got, c.want)
		}
	}
}
