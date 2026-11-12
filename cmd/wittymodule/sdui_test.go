// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func webModule(t *testing.T, api int, ui string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "wittyx")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	man := `{"id":"wittyx","name":"X","version":"0.1.0","repo":"wittywilla/WittyX","delivery":"web","entry":"index.js","api":` + string(rune('0'+api)) + `,"minServer":"0.8.0-alpha","maxServer":"0.8.x"}`
	files := map[string]string{"module.json": man, "LICENSE": "l", "NOTICE": "n"}
	if ui != "" {
		files["sdui.json"] = ui
	}
	for n, b := range files {
		if err := os.WriteFile(filepath.Join(dir, n), []byte(b), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

const smallUI = `{"format":"wittymodule-sdui","formatVersion":1,"module":"wittyx","catalog":2,"theme":{"scope":"wittyx"}}`

func TestCheckWantsSDUIFromAPI4(t *testing.T) {
	if err := checkOne(webModule(t, 4, ""), ""); err == nil || !strings.Contains(err.Error(), "sdui.json") {
		t.Errorf("API 4 without sdui.json: %v", err)
	}
	if err := checkOne(webModule(t, 4, smallUI), ""); err != nil {
		t.Errorf("API 4 with sdui.json: %v", err)
	}
	if err := checkOne(webModule(t, 3, ""), ""); err != nil {
		t.Errorf("API 3 without: %v", err)
	}
	if err := checkOne(webModule(t, 3, strings.Replace(smallUI, `"scope":"wittyx"`, `"scope":"wittytv"`, 1)), ""); err == nil {
		t.Error("a wrong sdui.json passed because the API is 3")
	}
}

func TestPackWebCarriesSDUI(t *testing.T) {
	dir := webModule(t, 4, smallUI)
	built := t.TempDir()
	_ = os.WriteFile(filepath.Join(built, "index.js"), []byte("export default 1"), 0o644)
	out := t.TempDir()
	if err := packWeb([]string{dir}, out, built); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(out, "wittyx-0.1.0-SHA256SUMS"))
	if err != nil {
		matches, _ := filepath.Glob(filepath.Join(out, "*SUMS*"))
		if len(matches) == 0 {
			t.Fatal("no checksum list")
		}
		b, _ = os.ReadFile(matches[0])
	}
	if !strings.Contains(string(b), "/sdui.json") {
		t.Errorf("sdui.json isn't in the release: %s", b)
	}
}
