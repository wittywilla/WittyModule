package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A program built with something other than Go is packed like any other, for the systems it was
// built for and no others.
func TestPackPrebuiltLeavesOutWhatWasntBuilt(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "wittyother")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, text := range map[string]string{
		"CMakeLists.txt": "",
		"module.json":    `{"id":"wittyother","name":"WittyOther","version":"1.2.3","repo":"someone/WittyOther","license":"GPL-3.0","delivery":"program","entry":"other-host","api":3,"minServer":"0.6.0-alpha","maxServer":"0.6.x"}`,
		"LICENSE":        "license",
		"NOTICE":         "notice",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	built := t.TempDir()
	for _, name := range []string{"other-host-linux-amd64", "other-host-windows-amd64.exe"} {
		if err := os.WriteFile(filepath.Join(built, name), []byte(name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	out := t.TempDir()
	if err := pack([]string{dir}, out, "", "", "", built); err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	entries, err := os.ReadDir(out)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		got[e.Name()] = true
	}
	for _, want := range []string{"wittyother-1.2.3-linux-amd64.tar.gz", "wittyother-1.2.3-windows-amd64.zip", "wittyother-1.2.3-module.json", "wittyother-1.2.3-SHA256SUMS.txt"} {
		if !got[want] {
			t.Errorf("no %s in %v", want, got)
		}
	}
	if len(got) != 4 {
		t.Errorf("more than was built: %v", got)
	}
	sums, err := os.ReadFile(filepath.Join(out, "wittyother-1.2.3-SHA256SUMS.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(sums), "wittyother-1.2.3-linux-amd64/other-host\n") {
		t.Errorf("the program isn't in the list:\n%s", sums)
	}

	if err := pack([]string{dir}, t.TempDir(), "", "", "", t.TempDir()); err == nil {
		t.Error("nothing was built, and it packed")
	}
}
