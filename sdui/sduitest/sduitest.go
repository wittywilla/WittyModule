// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors

// Package sduitest keeps a module's sdui.json equal to its Go definition, in the module's
// own tests:
//
//	func TestSDUIFileIsCurrent(t *testing.T) { sduitest.Golden(t, mymodule.UI(), "sdui.json") }
//
// Run `go test -run TestSDUIFileIsCurrent -update-sdui` to write it after a change.
package sduitest

import (
	"bytes"
	"flag"
	"os"
	"testing"

	"github.com/wittywilla/wittymodule/sdui"
)

var update = flag.Bool("update-sdui", false, "write sdui.json from the module's Go definition")

// Golden checks the module's definition and that the file at path is exactly it.
func Golden(t testing.TB, m *sdui.Module, path string) {
	t.Helper()
	for _, is := range sdui.Check(m, m.Module) {
		t.Errorf("%s: %s", is.Severity, is)
	}
	want, err := sdui.JSON(m)
	if err != nil {
		t.Fatal(err)
	}
	if *update {
		if err := os.WriteFile(path, want, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(bytes.ReplaceAll(got, []byte("\r\n"), []byte("\n")), want) {
		t.Errorf("%s is not the module's Go definition: run go test -run %s -update-sdui", path, t.Name())
	}
}
