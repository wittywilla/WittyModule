// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors

package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/wittywilla/wittymodule"
)

// playUI is the SDUI a release of the test helper carries: a page, a menu entry and settings.
const playUI = `{"format":"wittymodule-sdui","formatVersion":1,"module":"wittyplay","catalog":2,"theme":{"scope":"wittyplay","label":"Games"},
"pages":[{"id":"wittyplay","kind":"custom","title":"Games","sections":[{"id":"n","component":"Notice","props":{"tone":"info","text":"Hello"}}]}],
"navigation":[{"id":"wittyplay","type":"page","page":"wittyplay"}],
"settings":[{"id":"wittyplay","title":"Games","groups":[{"rows":[{"key":"fast","kind":"bool","label":"Fast"}]}]}]}`

// fakeRelease writes the files of a release of a helper, and returns its folder.
func fakeRelease(t *testing.T, ver string) string {
	t.Helper()
	return fakeReleaseWith(t, ver, playUI, "")
}

// fakeReleaseWith writes a release whose archive holds ui as its sdui.json ("" for none), and
// whose manifest has more fields (`,"status":"beta"`).
func fakeReleaseWith(t *testing.T, ver, ui, more string) string {
	t.Helper()
	dir := t.TempDir()
	man := fmt.Sprintf(`{"id":"wittyplay","name":"WittyPlay","version":%q,"license":"Apache-2.0","repo":"wittywilla/WittyPlay","delivery":"helper","entry":"wittyplay-module","api":%d,"minServer":"0.6.0-alpha","maxServer":"0.6.x","summary":"Games.","group":"play"%s}`, ver, wittymodule.API, more)
	manName := wittymodule.ManifestName("wittyplay", ver)
	top := wittymodule.TopFolder("wittyplay", ver, "linux", "amd64")
	inside := map[string][]byte{wittymodule.ManifestFile: []byte(man), "wittyplay-module": []byte("a program")}
	if ui != "" {
		inside[wittymodule.SDUIFile] = []byte(ui)
	}
	archive, err := archiveBytes("linux", top, map[string]bool{"wittyplay-module": true}, inside)
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{manName: []byte(man), wittymodule.ArchiveName("wittyplay", ver, "linux", "amd64"): archive}
	sums := ""
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), content, 0o644); err != nil {
			t.Fatal(err)
		}
		sums += hash(content) + "  " + name + "\n"
	}
	for name, content := range inside {
		sums += hash(content) + "  " + top + "/" + name + "\n"
	}
	sumsName := wittymodule.SumsName("wittyplay", ver)
	if err := os.WriteFile(filepath.Join(dir, sumsName), []byte(sums), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func signRelease(t *testing.T, dir, ver string) string {
	t.Helper()
	pub, priv, _ := ed25519.GenerateKey(nil)
	sumsName := wittymodule.SumsName("wittyplay", ver)
	b, err := os.ReadFile(filepath.Join(dir, sumsName))
	if err != nil {
		t.Fatal(err)
	}
	sig := base64.StdEncoding.EncodeToString(ed25519.Sign(priv, b)) + "\n"
	if err := os.WriteFile(filepath.Join(dir, sumsName+".sig"), []byte(sig), 0o644); err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(pub)
}

func TestCatalogAddFromARelease(t *testing.T) {
	dir := fakeRelease(t, "0.3.0")
	path := filepath.Join(t.TempDir(), wittymodule.CatalogName)

	if err := catalogAdd(path, "", "", "", "", []string{dir}); err == nil {
		t.Fatal("an unsigned release was listed")
	}
	pub := signRelease(t, dir, "0.3.0")

	other, _, _ := ed25519.GenerateKey(nil)
	if err := catalogAdd(path, "", "", base64.StdEncoding.EncodeToString(other), "", []string{dir}); err == nil {
		t.Fatal("a release signed with another key was listed")
	}

	log := filepath.Join(t.TempDir(), "CHANGELOG.md")
	if err := os.WriteFile(log, []byte("# Changelog\n\n## Unreleased\n\n- Soon.\n\n## 0.3.0\n\n- Plays games.\r\n\n## 0.2.0\n\n- Old.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := catalogAdd(path, "", log, pub, "WittyWilla", []string{dir}); err != nil {
		t.Fatal(err)
	}
	c, err := readCatalog(path)
	if err != nil {
		t.Fatal(err)
	}
	v := c.Find("wittyplay").Find("0.3.0")
	if v == nil || v.Changelog != "- Plays games." || c.Serial != 1 || c.Find("wittyplay").Publisher.Name != "WittyWilla" {
		t.Fatalf("%+v serial %d", v, c.Serial)
	}
	if !strings.HasPrefix(v.Sums, "https://github.com/wittywilla/WittyPlay/releases/download/v0.3.0/") {
		t.Fatalf("sums at %q", v.Sums)
	}
	if v.Platform("linux", "amd64") == nil || v.Platform("windows", "amd64") != nil {
		t.Fatalf("platforms %v", v.Platforms)
	}
	// What its sdui.json adds, summed up from the file in the signed archive.
	want := wittymodule.CatalogUI{Pages: []wittymodule.CatalogUIPage{{ID: "wittyplay", Title: "Games"}}, Navigation: []string{"Games"}, Settings: true}
	if v.UI == nil || !reflect.DeepEqual(*v.UI, want) || v.Status != "" {
		t.Fatalf("ui %+v, status %q", v.UI, v.Status)
	}
	man, err := wittymodule.ParseManifest(mustRead(t, filepath.Join(dir, wittymodule.ManifestName("wittyplay", "0.3.0"))))
	if err != nil {
		t.Fatal(err)
	}
	if err := v.MatchesManifest("wittyplay", man); err != nil {
		t.Fatal(err)
	}

	// Adding it again changes nothing, including the serial a server compares.
	if err := catalogAdd(path, "", log, pub, "", []string{dir}); err != nil {
		t.Fatal(err)
	}
	if c, _ := readCatalog(path); c.Serial != 1 {
		t.Fatalf("serial went to %d", c.Serial)
	}
}

func TestCatalogAddRefusesAChangedArchive(t *testing.T) {
	dir := fakeRelease(t, "0.3.0")
	signRelease(t, dir, "0.3.0")
	if err := os.WriteFile(filepath.Join(dir, wittymodule.ArchiveName("wittyplay", "0.3.0", "linux", "amd64")), []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := catalogAdd(filepath.Join(t.TempDir(), "c.json"), "", "", "", "", []string{dir}); err == nil {
		t.Fatal("an archive that doesn't match the signed list was listed")
	}
}

func TestCatalogSignAndCheck(t *testing.T) {
	dir := fakeRelease(t, "0.3.0")
	pub := signRelease(t, dir, "0.3.0")
	path := filepath.Join(t.TempDir(), wittymodule.CatalogName)
	if err := catalogAdd(path, "wittyplay", "", pub, "", []string{dir}); err != nil {
		t.Fatal(err)
	}
	keyFile := filepath.Join(t.TempDir(), "k.key")
	if err := keygen(keyFile); err != nil {
		t.Fatal(err)
	}
	if err := catalogSign(keyFile, []string{path}); err != nil {
		t.Fatal(err)
	}
	key, err := readKey(keyFile)
	if err != nil {
		t.Fatal(err)
	}
	catPub := base64.StdEncoding.EncodeToString(key.Public().(ed25519.PublicKey))
	if err := catalogCheck(catPub, []string{path}); err != nil {
		t.Fatal(err)
	}
	if err := catalogCheck(pub, []string{path}); err == nil {
		t.Fatal("the wrong key passed")
	}
	bad := filepath.Join(t.TempDir(), wittymodule.CatalogName)
	if err := os.WriteFile(bad, []byte(`{"format":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := catalogSign(keyFile, []string{bad}); err == nil {
		t.Fatal("a file that isn't a catalogue was signed")
	}
}

func TestCatalogAddTakesStatusAndRefusesWhatAServerWould(t *testing.T) {
	// An unfinished version is listed with its status and note, for the store to warn with.
	dir := fakeReleaseWith(t, "0.3.0", playUI, `,"status":"incomplete","statusNote":"No saves yet."`)
	pub := signRelease(t, dir, "0.3.0")
	path := filepath.Join(t.TempDir(), wittymodule.CatalogName)
	if err := catalogAdd(path, "", "", pub, "", []string{dir}); err != nil {
		t.Fatal(err)
	}
	c, err := readCatalog(path)
	if err != nil {
		t.Fatal(err)
	}
	if v := c.Find("wittyplay").Find("0.3.0"); v.Status != wittymodule.StatusIncomplete || v.StatusNote != "No saves yet." || v.UI == nil {
		t.Fatalf("%+v", v)
	}

	for name, ui := range map[string]string{
		"no sdui.json":           "",
		"another module's scope": strings.Replace(playUI, `"scope":"wittyplay"`, `"scope":"wittytv"`, 1),
		"an unknown field":       strings.Replace(playUI, `"catalog":2`, `"catalog":2,"script":"x"`, 1),
	} {
		dir := fakeReleaseWith(t, "0.3.1", ui, "")
		pub := signRelease(t, dir, "0.3.1")
		err := catalogAdd(filepath.Join(t.TempDir(), "c.json"), "", "", pub, "", []string{dir})
		if err == nil || !strings.Contains(err.Error(), "sdui.json") {
			t.Errorf("%s: %v", name, err)
		}
	}

	// An sdui.json changed after the list was signed is refused, as the archive would be.
	dir = fakeReleaseWith(t, "0.3.2", playUI, "")
	sumsName := filepath.Join(dir, wittymodule.SumsName("wittyplay", "0.3.2"))
	b := mustRead(t, sumsName)
	top := wittymodule.TopFolder("wittyplay", "0.3.2", "linux", "amd64")
	var lines []string
	for _, l := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		if strings.HasSuffix(l, top+"/sdui.json") {
			l = strings.Repeat("0", 64) + "  " + top + "/sdui.json"
		}
		lines = append(lines, l)
	}
	if err := os.WriteFile(sumsName, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	pub = signRelease(t, dir, "0.3.2")
	if err := catalogAdd(filepath.Join(t.TempDir(), "c.json"), "", "", pub, "", []string{dir}); err == nil || !strings.Contains(err.Error(), "doesn't match") {
		t.Errorf("a changed sdui.json: %v", err)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// sdui.json is read from either kind of archive, and only as large as one may be.
func TestFileInArchive(t *testing.T) {
	for _, goos := range []string{"windows", "linux"} {
		b, err := archiveBytes(goos, "top", nil, map[string][]byte{"sdui.json": []byte("{}"), "big": make([]byte, 2<<20)})
		if err != nil {
			t.Fatal(err)
		}
		name := wittymodule.ArchiveName("x", "1.0.0", goos, "amd64")
		if got, err := fileInArchive(name, b, "top/sdui.json"); err != nil || string(got) != "{}" {
			t.Errorf("%s: %q %v", goos, got, err)
		}
		if got, err := fileInArchive(name, b, "top/missing"); err != nil || got != nil {
			t.Errorf("%s: a missing file: %q %v", goos, got, err)
		}
		if _, err := fileInArchive(name, b, "top/big"); err == nil {
			t.Errorf("%s: a file past the limit was read", goos)
		}
	}
}
