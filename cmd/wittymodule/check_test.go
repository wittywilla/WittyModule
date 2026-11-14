// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withManifest rewrites the test module's manifest with more fields, or other values.
func withManifest(t *testing.T, dir string, edit func(string) string) {
	t.Helper()
	path := filepath.Join(dir, "module.json")
	if err := os.WriteFile(path, []byte(edit(string(mustRead(t, path)))), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Every problem is listed, one line each, located by file and field or JSON path.
func TestCheckListsEveryProblemWhereItIs(t *testing.T) {
	ui := strings.Replace(smallUI, `"theme":{"scope":"wittyx"}`, `"theme":{"scope":"wittytv","tokens":{"accent":"red"}}`, 1)
	dir := webModule(t, 5, ui)
	withManifest(t, dir, func(s string) string {
		s = strings.Replace(s, `"maxServer":"0.8.x"`, `"maxServer":"0.y.x","permissions":["network","root"],"status":"done"`, 1)
		return strings.Replace(s, `"name":"X"`, `"name":""`, 1)
	})
	problems, warnings := inspect(dir, "")
	man, file := filepath.Join(dir, "module.json"), filepath.Join(dir, "sdui.json")
	for _, want := range []string{
		man + ": name: ",
		man + ": maxServer: ",
		man + ": permissions[1]: ",
		man + ": status: ",
		file + ": /theme/scope: ",
		file + ": /theme/tokens/accent: ",
	} {
		found := false
		for _, p := range problems {
			found = found || strings.HasPrefix(p, want)
		}
		if !found {
			t.Errorf("no problem at %q in:\n%s", want, strings.Join(problems, "\n"))
		}
	}
	for _, p := range problems {
		if strings.Contains(p, "\n") || strings.Contains(p, "manifest of") {
			t.Errorf("not one plain line: %q", p)
		}
	}
	if len(warnings) != 0 {
		t.Errorf("warnings: %v", warnings)
	}
	err := check([]string{dir}, "")
	if err == nil || !strings.Contains(err.Error(), "in 1 of 1 folders") {
		t.Errorf("check: %v", err)
	}
	if err := checkOne(dir, ""); err == nil || strings.Count(err.Error(), "\n")+1 != len(problems) {
		t.Errorf("checkOne doesn't carry every problem: %v", err)
	}
}

func TestCheckLocatesTheFolderTheSDUIAndTheRange(t *testing.T) {
	dir := webModule(t, 5, "")
	problems, _ := inspect(dir, "")
	if len(problems) != 1 || !strings.HasPrefix(problems[0], filepath.Join(dir, "sdui.json")+": it has no sdui.json") {
		t.Errorf("no sdui.json: %v", problems)
	}

	dir = webModule(t, 5, `{"format":"wittymodule-sdui","formatVersion":1,"module":"wittyx","catalog":2,"theme":{"scope":"wittyx"},"script":"x"}`)
	problems, _ = inspect(dir, "")
	if len(problems) != 1 || !strings.HasPrefix(problems[0], filepath.Join(dir, "sdui.json")+": can't be read: ") {
		t.Errorf("unknown field: %v", problems)
	}

	dir = webModule(t, 5, smallUI)
	for server, field := range map[string]string{"0.7.0": "minServer", "0.9.0": "maxServer", "dev": ""} {
		problems, _ = inspect(dir, server)
		want := filepath.Join(dir, "module.json") + ": "
		if field != "" {
			want += field + ": "
		}
		if len(problems) != 1 || !strings.HasPrefix(problems[0], want) {
			t.Errorf("server %s: %v", server, problems)
		}
	}
	if problems, _ := inspect(dir, "0.8.1"); len(problems) != 0 {
		t.Errorf("in range: %v", problems)
	}

	if problems, _ := inspect(filepath.Join(t.TempDir(), "nothing"), ""); len(problems) != 1 || !strings.Contains(problems[0], "module.json: can't be read") {
		t.Errorf("no manifest: %v", problems)
	}
}

// An unfinished module passes, with a warning naming its status and note.
func TestCheckWarnsOfAnUnfinishedModule(t *testing.T) {
	for _, status := range []string{"experimental", "incomplete"} {
		dir := webModule(t, 5, smallUI)
		withManifest(t, dir, func(s string) string {
			return strings.Replace(s, `"maxServer":"0.8.x"`, `"maxServer":"0.8.x","status":"`+status+`","statusNote":"No settings page yet."`, 1)
		})
		problems, warnings := inspect(dir, "")
		if len(problems) != 0 {
			t.Fatalf("%s: %v", status, problems)
		}
		if len(warnings) != 1 || !strings.Contains(warnings[0], "module.json: status: ") || !strings.Contains(warnings[0], status) || !strings.Contains(warnings[0], "No settings page yet") {
			t.Errorf("%s: warnings %v", status, warnings)
		}
		if err := check([]string{dir}, ""); err != nil {
			t.Errorf("%s: check failed: %v", status, err)
		}
	}
	// One with no note doesn't pass: the note is what the warning is made of.
	dir := webModule(t, 5, smallUI)
	withManifest(t, dir, func(s string) string {
		return strings.Replace(s, `"maxServer":"0.8.x"`, `"maxServer":"0.8.x","status":"incomplete"`, 1)
	})
	if problems, _ := inspect(dir, ""); len(problems) != 1 || !strings.Contains(problems[0], "module.json: statusNote: ") {
		t.Errorf("no note: %v", problems)
	}
	// Beta and stable are nothing to warn of.
	dir = webModule(t, 5, smallUI)
	withManifest(t, dir, func(s string) string {
		return strings.Replace(s, `"maxServer":"0.8.x"`, `"maxServer":"0.8.x","status":"beta"`, 1)
	})
	if problems, warnings := inspect(dir, ""); len(problems)+len(warnings) != 0 {
		t.Errorf("beta: %v %v", problems, warnings)
	}
}
