package wittymodule

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func testVersion(ver string) CatalogVersion {
	return CatalogVersion{Version: ver, Delivery: DeliveryHelper, API: API, MinServer: "0.6.0-alpha", MaxServer: "0.6.x", License: "Apache-2.0",
		Released: "2026-11-01", Sums: "a-SHA256SUMS.txt", Sig: "a-SHA256SUMS.txt.sig", Manifest: "a-module.json",
		Platforms: map[string]CatalogFile{"linux-amd64": {URL: "https://github.com/o/r/releases/download/v1/a.tar.gz", Size: 10, SHA256: strings.Repeat("a", 64)}}}
}

func testCatalog() *Catalog {
	return &Catalog{Format: CatalogFormat, Name: "Test", Serial: 1, Generated: "2026-11-01T12:00:00Z", Modules: []CatalogModule{{
		ID: "wittyplay", Name: "WittyPlay", Summary: "Games.", Publisher: Publisher{Name: "WittyWilla"}, Repo: "wittywilla/WittyPlay",
		Versions: []CatalogVersion{testVersion("0.3.0")},
	}}}
}

func TestCatalogRoundTripAndSignature(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	other, _, _ := ed25519.GenerateKey(nil)
	raw, err := json.Marshal(testCatalog())
	if err != nil {
		t.Fatal(err)
	}
	sig := []byte(base64.StdEncoding.EncodeToString(ed25519.Sign(priv, raw)) + "\n")
	c, err := VerifyCatalog(raw, sig, []ed25519.PublicKey{other, pub})
	if err != nil {
		t.Fatal(err)
	}
	if c.Find("wittyplay").Find("0.3.0").Platform("linux", "amd64") == nil {
		t.Fatal("the version's archive wasn't found")
	}
	if c.Find("wittyplay").Find("0.3.0").Platform("windows", "amd64") != nil {
		t.Fatal("a platform with no archive was found")
	}
	if _, err := VerifyCatalog(raw, sig, []ed25519.PublicKey{other}); err != ErrCatalogSignature {
		t.Fatalf("wrong key: %v", err)
	}
	if _, err := VerifyCatalog(append(raw, ' '), sig, []ed25519.PublicKey{pub}); err != ErrCatalogSignature {
		t.Fatalf("changed file: %v", err)
	}
}

func TestCatalogRefusals(t *testing.T) {
	for name, change := range map[string]func(c *Catalog){
		"no serial":               func(c *Catalog) { c.Serial = 0 },
		"bad time":                func(c *Catalog) { c.Generated = "soon" },
		"two of a module":         func(c *Catalog) { c.Modules = append(c.Modules, c.Modules[0]) },
		"bad id":                  func(c *Catalog) { c.Modules[0].ID = "../x" },
		"control characters":      func(c *Catalog) { c.Modules[0].Summary = "a\x1b[2Jb" },
		"line break in a summary": func(c *Catalog) { c.Modules[0].Summary = "a\nb" },
		"no publisher":            func(c *Catalog) { c.Modules[0].Publisher.Name = "" },
		"bad publisher key":       func(c *Catalog) { c.Modules[0].Publisher.Key = "abc" },
		"http publisher":          func(c *Catalog) { c.Modules[0].Publisher.URL = "http://example.test" },
		"two of a version":        func(c *Catalog) { c.Modules[0].Versions = append(c.Modules[0].Versions, c.Modules[0].Versions[0]) },
		"no licence":              func(c *Catalog) { c.Modules[0].Versions[0].License = "" },
		"bad date":                func(c *Catalog) { c.Modules[0].Versions[0].Released = "2026-13-40" },
		"library has no release":  func(c *Catalog) { c.Modules[0].Versions[0].Delivery = DeliveryLibrary },
		"no archives":             func(c *Catalog) { c.Modules[0].Versions[0].Platforms = nil },
		"bad platform": func(c *Catalog) {
			c.Modules[0].Versions[0].Platforms["../x"] = CatalogFile{URL: "a", Size: 1, SHA256: strings.Repeat("a", 64)}
		},
		"bad hash": func(c *Catalog) {
			c.Modules[0].Versions[0].Platforms["linux-amd64"] = CatalogFile{URL: "a", Size: 1, SHA256: "AB"}
		},
		"archive too big": func(c *Catalog) {
			c.Modules[0].Versions[0].Platforms["linux-amd64"] = CatalogFile{URL: "a", Size: 1 << 40, SHA256: strings.Repeat("a", 64)}
		},
		"climbing path": func(c *Catalog) { c.Modules[0].Versions[0].Sums = "../../etc/passwd" },
		"http file":     func(c *Catalog) { c.Modules[0].Versions[0].Sig = "http://example.test/a.sig" },
		"file address":  func(c *Catalog) { c.Modules[0].Versions[0].Manifest = "file:///etc/passwd" },
		"bad withdrawal": func(c *Catalog) {
			c.Modules[0].Versions[0].Withdrawn = &CatalogWithdrawn{Severity: "maybe", Reason: "x"}
		},
	} {
		c := testCatalog()
		c.Modules[0].Versions[0].Platforms = map[string]CatalogFile{"linux-amd64": c.Modules[0].Versions[0].Platforms["linux-amd64"]}
		change(c)
		if err := c.Validate(); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if err := testCatalog().Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestCatalogNewerFormatSaysSo(t *testing.T) {
	_, err := ParseCatalog([]byte(`{"format":99,"modules":"anything at all"}`))
	if err == nil || !strings.Contains(err.Error(), "a newer server") {
		t.Fatalf("got %v", err)
	}
}

func TestCatalogUnknownFieldsAndPermissionsAreRead(t *testing.T) {
	c := testCatalog()
	c.Modules[0].Versions[0].Permissions = []string{"from-the-future"}
	raw, _ := json.Marshal(c)
	raw = []byte(strings.Replace(string(raw), `"format":1`, `"format":1,"added":"later"`, 1))
	if _, err := ParseCatalog(raw); err != nil {
		t.Fatal(err)
	}
}

func TestCatalogURLs(t *testing.T) {
	for s, want := range map[string]bool{
		"https://github.com/o/r/releases/download/v1/a.zip": true,
		"wittyplay/a.tar.gz":               true,
		"a.tar.gz":                         true,
		"":                                 false,
		"/etc/passwd":                      false,
		"../a":                             false,
		"a/../../b":                        false,
		"a//b":                             false,
		`a\b`:                              false,
		"http://example.test/a":            false,
		"https://user:pass@example.test/a": false,
		"file:///etc/passwd":               false,
		"//example.test/a":                 false,
		"a?x=1":                            false,
	} {
		if got := ValidCatalogURL(s); got != want {
			t.Errorf("%q: got %v", s, got)
		}
	}
	got, err := ResolveCatalogURL("https://github.com/o/r/releases/download/catalog/catalog.json", "icons/a.png")
	if err != nil || got != "https://github.com/o/r/releases/download/catalog/icons/a.png" {
		t.Fatalf("%v %v", got, err)
	}
	got, err = ResolveCatalogURL("file:///srv/starter/catalog.json", "https://example.test/a.zip")
	if err != nil || got != "https://example.test/a.zip" {
		t.Fatalf("%v %v", got, err)
	}
}

func TestCatalogMatchesManifest(t *testing.T) {
	v := testVersion("0.3.0")
	v.Permissions = []string{PermissionNetwork}
	good := func() *Manifest {
		return &Manifest{ID: "wittyplay", Version: "0.3.0", Delivery: DeliveryHelper, API: API, MinServer: "0.6.0-alpha", MaxServer: "0.6.x", License: "Apache-2.0", Permissions: []string{PermissionNetwork}}
	}
	if err := v.MatchesManifest("wittyplay", good()); err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(m *Manifest){
		"id":          func(m *Manifest) { m.ID = "wittydvd" },
		"version":     func(m *Manifest) { m.Version = "0.3.1" },
		"delivery":    func(m *Manifest) { m.Delivery = DeliveryProgram },
		"api":         func(m *Manifest) { m.API = 1 },
		"range":       func(m *Manifest) { m.MaxServer = "0.9.x" },
		"licence":     func(m *Manifest) { m.License = "GPL-3.0" },
		"permissions": func(m *Manifest) { m.Permissions = nil },
	} {
		m := good()
		change(m)
		if v.MatchesManifest("wittyplay", m) == nil {
			t.Errorf("a different %s was accepted", name)
		}
	}
}

func TestCatalogPut(t *testing.T) {
	c := testCatalog()
	mod := CatalogModule{ID: "wittyplay", Name: "WittyPlay 2", Summary: "Newer.", Publisher: Publisher{Name: "WittyWilla"}, Repo: "wittywilla/WittyPlay"}

	changed, err := c.Put(mod, testVersion("0.3.0"))
	if err != nil || changed || c.Serial != 1 {
		t.Fatalf("the same version again: %v %v serial %d", changed, err, c.Serial)
	}
	different := testVersion("0.3.0")
	different.Platforms["linux-amd64"] = CatalogFile{URL: "b", Size: 2, SHA256: strings.Repeat("b", 64)}
	if _, err := c.Put(mod, different); err == nil {
		t.Fatal("a released version was changed")
	}
	if changed, err := c.Put(mod, testVersion("0.2.0")); err != nil || !changed {
		t.Fatal(changed, err)
	}
	if m := c.Find("wittyplay"); m.Versions[1].Version != "0.2.0" || m.Name != "WittyPlay" {
		t.Fatalf("an older version changed the order or the name: %+v", m)
	}
	if changed, err := c.Put(mod, testVersion("0.4.0")); err != nil || !changed || c.Serial != 3 {
		t.Fatal(changed, err, c.Serial)
	}
	if m := c.Find("wittyplay"); m.Versions[0].Version != "0.4.0" || m.Name != "WittyPlay 2" {
		t.Fatalf("the newest version didn't lead: %+v", m)
	}
	mod.ID = "wittynew"
	if changed, err := c.Put(mod, testVersion("0.1.0")); err != nil || !changed || c.Find("wittynew") == nil {
		t.Fatal(changed, err)
	}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
}

// A version with less to say can't blank what's listed for the store.
func TestCatalogPutKeepsWords(t *testing.T) {
	c := testCatalog()
	mod := c.Modules[0]
	mod.Summary, mod.Group = "What the store says.", "photos"
	c.Modules[0] = mod
	bare := CatalogModule{ID: mod.ID, Name: mod.Name, Repo: mod.Repo}
	if changed, err := c.Put(bare, testVersion("9.0.0")); err != nil || !changed {
		t.Fatal(changed, err)
	}
	if got := c.Find(mod.ID); got.Summary != "What the store says." || got.Group != "photos" {
		t.Fatalf("%+v", got)
	}
}
