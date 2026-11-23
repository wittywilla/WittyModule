// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors

package wittymodule

import (
	"crypto/ed25519"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Everything a server reads from a catalogue, a release or a withdrawals list is parsed
// before anyone has looked at it. None of it may panic, and what is accepted must be valid.

func seedJSON(f *testing.F, pattern string) {
	f.Helper()
	files, _ := filepath.Glob(pattern)
	for _, p := range files {
		if b, err := os.ReadFile(p); err == nil && len(b) < 1<<20 {
			f.Add(b)
		}
	}
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"format":1,"name":"x","serial":1,"generated":"2026-01-01T00:00:00Z","modules":[]}`))
	f.Add([]byte(`[`))
}

func FuzzParseCatalog(f *testing.F) {
	seedJSON(f, "testdata/*catalog*.json")
	f.Fuzz(func(t *testing.T, b []byte) {
		c, err := ParseCatalog(b)
		if err != nil {
			return
		}
		if err := c.Validate(); err != nil {
			t.Fatalf("accepted, then invalid: %v", err)
		}
	})
}

func FuzzVerifyCatalog(f *testing.F) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	raw := []byte(`{"format":1,"name":"x","serial":1,"generated":"2026-01-01T00:00:00Z","modules":[]}`)
	good := base64.StdEncoding.EncodeToString(ed25519.Sign(priv, raw))
	f.Add(raw, []byte(good))
	f.Add(raw, []byte("AAAA"))
	f.Fuzz(func(t *testing.T, b, sig []byte) {
		c, err := VerifyCatalog(b, sig, []ed25519.PublicKey{pub})
		if err != nil {
			return
		}
		s, _ := base64.StdEncoding.DecodeString(strings.TrimSpace(string(sig)))
		if !ed25519.Verify(pub, b, s) || c == nil {
			t.Fatal("a catalogue the key didn't sign was taken")
		}
	})
}

func FuzzParseManifest(f *testing.F) {
	seedJSON(f, "testdata/*.json")
	f.Add([]byte(`{"id":"x","name":"X","version":"1.0.0","repo":"a/b","api":5,"minServer":"0.8.0","maxServer":"0.8.x","entry":"x"}`))
	f.Fuzz(func(t *testing.T, b []byte) {
		m, err := ParseManifest(b)
		if err != nil || m == nil {
			return
		}
		if !plainText(m.Name, 64, false) {
			t.Fatalf("a manifest named %q was taken", m.Name)
		}
	})
}

func FuzzParseWithdrawals(f *testing.F) {
	f.Add([]byte(`{"module":"x","serial":1,"versions":[{"version":"1.0.0","severity":"security","reason":"r"}]}`))
	f.Fuzz(func(t *testing.T, b []byte) {
		w, err := ParseWithdrawals(b)
		if err != nil || w == nil {
			return
		}
		for _, v := range w.Versions {
			_ = w.Find(v.Version)
		}
	})
}

func FuzzParseIntegration(f *testing.F) {
	seedJSON(f, "testdata/*integration*.json")
	f.Fuzz(func(t *testing.T, b []byte) { _, _ = ParseIntegration(b) })
}

func TestPlainTextRefusesHiddenCharacters(t *testing.T) {
	r := func(c rune) string { return string(c) }
	for _, s := range []string{
		"Witty" + r(0x202e) + "Tuner", "WittyTuner" + r(0x200b), r(0x2066) + "WittyTuner" + r(0x2069), "Witty" + r(0x85) + "Tuner",
		"Witty" + r(0x9b) + "Tuner", r(0xfeff) + "WittyTuner", "a" + r(0x61c) + "b", "Witty" + r(0x7f),
	} {
		if plainText(s, 64, false) || plainText(s, 64, true) {
			t.Errorf("%q was taken as plain text", s)
		}
	}
	family := r(0x1f468) + r(0x200d) + r(0x1f469) + r(0x200d) + r(0x1f467) + " Family"
	for _, s := range []string{"WittyTuner", "Caf" + r(0xe9) + " T" + r(0xe9) + "l" + r(0xe9), r(0x65e5) + r(0x672c) + r(0x8a9e), family} {
		if !plainText(s, 64, false) {
			t.Errorf("%q was refused", s)
		}
	}
	if !plainText("line one"+r(10)+"line two"+r(9)+"end", 64, true) || plainText("line one"+r(10)+"line two", 64, false) {
		t.Error("new lines are for long text only")
	}
}

func TestManifestNameIsPlainText(t *testing.T) {
	m := Manifest{ID: "x", Name: "Witty" + string(rune(0x202e)) + "Tuner", Version: "1.0.0", Repo: "a/b", API: 5, MinServer: "0.8.0", MaxServer: "0.8.x", Entry: "x"}
	found := false
	for _, p := range m.Problems() {
		if p.Field == "name" {
			found = true
		}
	}
	if !found {
		t.Fatal("a manifest name with a direction override was taken")
	}
}
