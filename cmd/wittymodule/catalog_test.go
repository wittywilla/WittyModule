// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors

package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wittywilla/wittymodule"
)

// fakeRelease writes the files of a release of a helper, and returns its folder.
func fakeRelease(t *testing.T, ver string) string {
	t.Helper()
	dir := t.TempDir()
	man := fmt.Sprintf(`{"id":"wittyplay","name":"WittyPlay","version":%q,"license":"Apache-2.0","repo":"wittywilla/WittyPlay","delivery":"helper","entry":"wittyplay-module","api":%d,"minServer":"0.6.0-alpha","maxServer":"0.6.x","summary":"Games.","group":"play"}`, ver, wittymodule.API)
	manName := wittymodule.ManifestName("wittyplay", ver)
	archive := wittymodule.ArchiveName("wittyplay", ver, "linux", "amd64")
	files := map[string]string{manName: man, archive: "the archive"}
	sums := ""
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		h := sha256.Sum256([]byte(content))
		sums += wittymodule.HashHex(h[:]) + "  " + name + "\n"
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

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
