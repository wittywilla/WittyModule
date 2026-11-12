// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors

package sdui

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The kit refuses what the server refuses: here a settings row's label longer than the
// server allows, which a lighter check once let through.
func TestCheckIsTheServersValidator(t *testing.T) {
	m := boardGames()
	m.Settings[0].Groups[0].Rows = append(m.Settings[0].Groups[0].Rows, SettingRow{Kind: "info", Label: strings.Repeat("x", 104)})
	found := false
	for _, is := range Check(m, "wittyboardgames") {
		found = found || (is.Severity == "error" && strings.HasPrefix(is.Path, "/settings/0/groups/0/rows/3/label"))
	}
	if !found {
		t.Fatalf("a 104-character label passed: %v", Check(m, "wittyboardgames"))
	}
}

// internal/core is a copy of the server's pkg/sdui. Beside a server checkout (or with
// WMS_SERVER naming one), it must be the same, file for file: run kit/sync-sdui-core.sh.
// Without one there is nothing to compare with, and the test says so.
func TestCoreIsTheServersCopy(t *testing.T) {
	server := os.Getenv("WMS_SERVER")
	if server == "" {
		server = filepath.Join("..", "..", "WittyMediaServer")
		if _, err := os.Stat(filepath.Join(server, "pkg", "sdui", "module.go")); err != nil {
			server = filepath.Join("..", "..", "..", "WittyMediaServer")
		}
	}
	src := filepath.Join(server, "pkg", "sdui")
	if _, err := os.Stat(filepath.Join(src, "module.go")); err != nil {
		t.Skip("no WittyMediaServer checkout beside this one, or in WMS_SERVER, to compare internal/core with")
	}
	want := map[string][]byte{}
	files, _ := filepath.Glob(filepath.Join(src, "*.go"))
	for _, f := range files {
		if !strings.HasSuffix(f, "_test.go") {
			b, _ := os.ReadFile(f)
			want[filepath.Base(f)] = b
		}
	}
	have, _ := filepath.Glob(filepath.Join("internal", "core", "*.go"))
	if len(have) != len(want) {
		t.Errorf("internal/core has %d files and the server's pkg/sdui %d: run kit/sync-sdui-core.sh", len(have), len(want))
	}
	for _, f := range have {
		b, _ := os.ReadFile(f)
		if w, ok := want[filepath.Base(f)]; !ok || !bytes.Equal(b, w) {
			t.Errorf("internal/core/%s differs from the server's: run kit/sync-sdui-core.sh", filepath.Base(f))
		}
	}
}
