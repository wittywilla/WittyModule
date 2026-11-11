// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors

package main

import (
	"bufio"
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wittywilla/wittymodule"
)

// catalogAdd lists one release in a catalogue, from the folder its files are in. The release
// has to check out first, since a catalogue only ever offers what its own files back up.
func catalogAdd(path, base, changelog, pubKey, publisher string, dirs []string) error {
	if path == "" || len(dirs) != 1 {
		return errors.New("wittymodule catalog-add -catalog file [-base url] [-changelog file] [-pub key] <release folder>")
	}
	dir := dirs[0]
	found, err := filepath.Glob(filepath.Join(dir, "*-module.json"))
	if err != nil {
		return err
	}
	if len(found) != 1 {
		return fmt.Errorf("%s: expected one manifest, found %d", dir, len(found))
	}
	rawMan, err := os.ReadFile(found[0])
	if err != nil {
		return err
	}
	man, err := wittymodule.ParseManifest(rawMan)
	if err != nil {
		return err
	}
	if err := man.ValidateRelease(); err != nil {
		return err
	}
	sumsName := wittymodule.SumsName(man.ID, man.Version)
	sums, err := os.ReadFile(filepath.Join(dir, sumsName))
	if err != nil {
		return err
	}
	sig, err := os.ReadFile(filepath.Join(dir, sumsName+".sig"))
	if err != nil {
		return fmt.Errorf("the release isn't signed: %w", err)
	}
	if pubKey != "" {
		if err := checkSignature(pubKey, sums, sig); err != nil {
			return fmt.Errorf("%s: %w", sumsName, err)
		}
	}
	list := readSums(sums)
	if sum := sha256.Sum256(rawMan); list[filepath.Base(found[0])] != wittymodule.HashHex(sum[:]) {
		return errors.New("the manifest doesn't match the signed list")
	}

	if base == "" {
		base = "https://github.com/" + man.Repo + "/releases/download/v" + man.Version
	}
	at := func(name string) string { return strings.TrimSuffix(base, "/") + "/" + name }

	platforms := map[string]wittymodule.CatalogFile{}
	addArchive := func(key, name string) error {
		st, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			return nil // a release needn't have every platform
		}
		want, ok := list[name]
		if !ok {
			return fmt.Errorf("%s isn't in the signed list", name)
		}
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		if sum := sha256.Sum256(b); wittymodule.HashHex(sum[:]) != want {
			return fmt.Errorf("%s doesn't match the signed list", name)
		}
		platforms[key] = wittymodule.CatalogFile{URL: at(name), Size: st.Size(), SHA256: want}
		return nil
	}
	if man.PerPlatform() {
		for _, t := range wittymodule.Targets {
			if err := addArchive(t.GOOS+"-"+t.GOARCH, man.Archive(t.GOOS, t.GOARCH)); err != nil {
				return err
			}
		}
	} else if err := addArchive(wittymodule.PlatformAll, man.Archive("", "")); err != nil {
		return err
	}
	if len(platforms) == 0 {
		return fmt.Errorf("%s: no archives of %s %s", dir, man.ID, man.Version)
	}

	notes := ""
	if changelog != "" {
		b, err := os.ReadFile(changelog)
		if err != nil {
			return err
		}
		notes = changelogFor(string(b), man.Version)
	}
	if publisher == "" {
		publisher, _, _ = strings.Cut(man.Repo, "/")
	}
	owner, _, _ := strings.Cut(man.Repo, "/")
	mod := wittymodule.CatalogModule{ID: man.ID, Name: man.Name, Summary: man.Summary, Publisher: wittymodule.Publisher{Name: publisher, URL: "https://github.com/" + owner},
		Repo: man.Repo, Group: man.Group, Libraries: man.Libraries, Requires: man.Requires, Core: man.Core, Needs: man.Needs, Updates: man.Updates, Stays: man.Stays}
	ver := wittymodule.CatalogVersion{Version: man.Version, Delivery: man.Delivery, API: man.API, MinServer: man.MinServer, MaxServer: man.MaxServer,
		License: man.License, Permissions: man.Permissions, Released: time.Now().UTC().Format(time.DateOnly), Prerelease: wittymodule.Prerelease(man.Version), Changelog: notes,
		Sums: at(sumsName), Sig: at(sumsName + ".sig"), Manifest: at(filepath.Base(found[0])), Platforms: platforms}

	c, err := readCatalog(path)
	if err != nil {
		return err
	}
	changed, err := c.Put(mod, ver)
	if err != nil {
		return err
	}
	if !changed {
		fmt.Printf("%s %s is already listed\n", man.ID, man.Version)
		return nil
	}
	if err := writeCatalog(path, c); err != nil {
		return err
	}
	fmt.Printf("%s %s listed in %s, serial %d\n", man.ID, man.Version, path, c.Serial)
	return nil
}

// readCatalog reads a catalogue, or starts one if there's no file yet.
func readCatalog(path string) (*wittymodule.Catalog, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &wittymodule.Catalog{Format: wittymodule.CatalogFormat, Name: "Modules", Generated: time.Now().UTC().Format(time.RFC3339), Modules: []wittymodule.CatalogModule{}}, nil
	}
	if err != nil {
		return nil, err
	}
	return wittymodule.ParseCatalog(b)
}

func writeCatalog(path string, c *wittymodule.Catalog) error {
	c.Generated = time.Now().UTC().Format(time.RFC3339)
	if err := c.Validate(); err != nil {
		return err
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(c); err != nil {
		return err
	}
	return os.WriteFile(path, b.Bytes(), 0o644)
}

// catalogCheck says whether a catalogue is sound, and with a key whether that key signed it.
func catalogCheck(pubKey string, files []string) error {
	if len(files) != 1 {
		return errors.New("wittymodule catalog-check [-pub key] <catalog.json>")
	}
	b, err := os.ReadFile(files[0])
	if err != nil {
		return err
	}
	c, err := wittymodule.ParseCatalog(b)
	if err != nil {
		return err
	}
	signed := "signature not checked"
	if pubKey != "" {
		sig, err := os.ReadFile(files[0] + ".sig")
		if err != nil {
			return err
		}
		if err := checkSignature(pubKey, b, sig); err != nil {
			return err
		}
		signed = "signature good"
	}
	versions := 0
	for i := range c.Modules {
		versions += len(c.Modules[i].Versions)
	}
	fmt.Printf("ok %s: %q, serial %d, %d modules, %d versions, %s\n", files[0], c.Name, c.Serial, len(c.Modules), versions, signed)
	return nil
}

// catalogSign signs a catalogue that passes its checks, and nothing else.
func catalogSign(keyPath string, files []string) error {
	if len(files) != 1 {
		return errors.New("wittymodule catalog-sign [-key file] <catalog.json>")
	}
	b, err := os.ReadFile(files[0])
	if err != nil {
		return err
	}
	if _, err := wittymodule.ParseCatalog(b); err != nil {
		return err
	}
	if err := sign(keyPath, "", files); err != nil {
		return err
	}
	// What a server shows an admin the first time they add this as a store.
	key, err := readKey(keyPath)
	if err != nil {
		return err
	}
	pub := base64.StdEncoding.EncodeToString(key.Public().(ed25519.PublicKey)) + "\n"
	return os.WriteFile(filepath.Join(filepath.Dir(files[0]), wittymodule.CatalogKeyName), []byte(pub), 0o644)
}

func checkSignature(pubKey string, data, sig []byte) error {
	k, err := base64.StdEncoding.DecodeString(strings.TrimSpace(pubKey))
	if err != nil || len(k) != ed25519.PublicKeySize {
		return errors.New("that isn't a public key: it should be the 44 characters keygen prints")
	}
	s, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(sig)))
	if err != nil || !ed25519.Verify(ed25519.PublicKey(k), data, s) {
		return errors.New("the signature doesn't check out with that key")
	}
	return nil
}

// readSums reads a checksum list into name and hash.
func readSums(b []byte) map[string]string {
	out := map[string]string{}
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		sum, name, ok := strings.Cut(strings.TrimSpace(sc.Text()), " ")
		if ok {
			out[strings.TrimPrefix(strings.TrimSpace(name), "*")] = strings.ToLower(sum)
		}
	}
	return out
}

// changelogFor is one version's section of a changelog, without its heading: the part under
// "## 0.3.0", "## v0.3.0" or "## [0.3.0] - 2026-11-01", up to the next heading of that level.
func changelogFor(md, version string) string {
	var out []string
	inside := false
	for _, line := range strings.Split(strings.ReplaceAll(md, "\r\n", "\n"), "\n") {
		if rest, ok := strings.CutPrefix(line, "## "); ok {
			if inside {
				break
			}
			first, _, _ := strings.Cut(strings.TrimSpace(rest), " ")
			inside = strings.TrimPrefix(strings.Trim(first, "[]"), "v") == version
			continue
		}
		if inside {
			out = append(out, line)
		}
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}
