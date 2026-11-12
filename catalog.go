// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors

package wittymodule

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// A catalogue is how a server learns what modules exist, with nothing compiled in: one
// signed file listing every module and every version, with the facts a store shows and
// where each version's files are. The catalogue only offers. What runs is still decided
// by the signed list and manifest of the release itself, and the two must agree
// (MatchesManifest), so someone holding only the catalogue's key can't make anything run.
//
// docs/MODULES.md in WittyMediaServer, "The catalogue", has the reasoning.

// CatalogFormat is the format this package writes and reads. A file with a higher one is
// refused in words, and the last good catalogue stays in use. Adding a field doesn't raise
// it, since readers ignore fields they don't know.
const CatalogFormat = 1

const (
	// CatalogName is the catalogue's file; its signature adds .sig.
	CatalogName = "catalog.json"
	// CatalogKeyName is the public key a store offers beside its catalogue, in base64. It is
	// only ever a suggestion for the first time an admin adds the store: from then on the key
	// that was agreed to is the one that counts, and a different one here is a refusal.
	CatalogKeyName = CatalogName + ".key"
	// PlatformAll is the one platform of a web bundle.
	PlatformAll = "all"
	// MaxCatalog is the largest catalogue a reader takes, checked before reading it all.
	MaxCatalog = 8 << 20
)

// Limits on what a catalogue may carry: all of it is kept and shown.
const (
	maxCatalogModules  = 1000
	maxCatalogVersions = 100
	maxCatalogLong     = 16 << 10 // a description, or one version's changelog
	maxCatalogURL      = 2048
	maxIcon            = 256 << 10
	maxScreenshots     = 8
	maxScreenshot      = 2 << 20
)

// CatalogFile is one downloadable file, with what it must turn out to be.
type CatalogFile struct {
	URL    string `json:"url"` // https, or relative to the catalogue's own location
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

// Publisher says who a module comes from. Key is the base64 Ed25519 public key its
// releases are signed with, when that isn't the catalogue's own key. A server believes it
// only from a store an admin added, and never for a stock module.
type Publisher struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
	Key  string `json:"key,omitempty"`
}

// CatalogWithdrawn takes a listed version back, as a module's own signed list does.
type CatalogWithdrawn struct {
	Severity string `json:"severity"`
	Reason   string `json:"reason"`
}

// CatalogVersion is one release of a module. Everything down to Permissions has to match
// the release's own signed manifest.
type CatalogVersion struct {
	Version     string   `json:"version"`
	Delivery    string   `json:"delivery"`
	API         int      `json:"api"`
	MinServer   string   `json:"minServer"`
	MaxServer   string   `json:"maxServer"`
	License     string   `json:"license"`
	Permissions []string `json:"permissions,omitempty"`

	Released   string `json:"released"`             // YYYY-MM-DD
	Prerelease bool   `json:"prerelease,omitempty"` // also true of any version with a pre-release label
	Changelog  string `json:"changelog,omitempty"`  // plain text; empty when there's nothing to report

	// The release's signed files, and an archive for each platform, keyed by GOOS-GOARCH
	// ("linux-arm" is the armv7 build), or PlatformAll for a web bundle.
	Sums      string                 `json:"sums"`
	Sig       string                 `json:"sig"`
	Manifest  string                 `json:"manifest"`
	Platforms map[string]CatalogFile `json:"platforms"`

	Withdrawn *CatalogWithdrawn `json:"withdrawn,omitempty"`

	// Status and StatusNote are the release manifest's own, for the store to show before
	// anything is downloaded. What a server goes by is the signed manifest, which an install
	// reads anyway: these are never a reason to install or refuse.
	Status     string `json:"status,omitempty"`
	StatusNote string `json:"statusNote,omitempty"`
	// UI sums up what the version's sdui.json adds to the apps, for the store's page. It is
	// written by `wittymodule catalog-add` from the file in the signed release; a version
	// with no sdui.json has none.
	UI *CatalogUI `json:"ui,omitempty"`
}

// CatalogUI sums up a version's sdui.json: its pages, its menu entries by label, whether it
// has settings, and how many data sources and inserts into built-in pages it has.
type CatalogUI struct {
	Pages      []CatalogUIPage `json:"pages,omitempty"`
	Navigation []string        `json:"navigation,omitempty"`
	Settings   bool            `json:"settings,omitempty"`
	Sources    int             `json:"sources,omitempty"`
	Inserts    int             `json:"inserts,omitempty"`
}

// CatalogUIPage is one page a version adds.
type CatalogUIPage struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// Limits on a version's UI summary, which match what sdui.json itself may carry.
const (
	MaxUIPages      = 32
	MaxUIPageTitle  = 120 // characters
	MaxUINavigation = 16
	MaxUILabel      = 40 // characters
	MaxUISources    = 64
	MaxUIInserts    = 32
)

var uiPageRe = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`) // an SDUI page id

func (u *CatalogUI) validate() error {
	switch {
	case len(u.Pages) > MaxUIPages:
		return fmt.Errorf("ui: more than %d pages", MaxUIPages)
	case len(u.Navigation) > MaxUINavigation:
		return fmt.Errorf("ui: more than %d menu entries", MaxUINavigation)
	case u.Sources < 0 || u.Sources > MaxUISources:
		return fmt.Errorf("ui: sources is between 0 and %d", MaxUISources)
	case u.Inserts < 0 || u.Inserts > MaxUIInserts:
		return fmt.Errorf("ui: inserts is between 0 and %d", MaxUIInserts)
	}
	seen := map[string]bool{}
	for i, p := range u.Pages {
		switch {
		case !uiPageRe.MatchString(p.ID) || seen[p.ID]:
			return fmt.Errorf("ui: pages[%d]: %q is not a page's id, or there are two of it", i, p.ID)
		case strings.TrimSpace(p.Title) == "" || utf8.RuneCountInString(p.Title) > MaxUIPageTitle || !plainText(p.Title, 4*MaxUIPageTitle, false):
			return fmt.Errorf("ui: pages[%d]: its title is one line of plain text of 1 to %d characters", i, MaxUIPageTitle)
		}
		seen[p.ID] = true
	}
	for i, l := range u.Navigation {
		if strings.TrimSpace(l) == "" || utf8.RuneCountInString(l) > MaxUILabel || !plainText(l, 4*MaxUILabel, false) {
			return fmt.Errorf("ui: navigation[%d]: a label is one line of plain text of 1 to %d characters", i, MaxUILabel)
		}
	}
	return nil
}

// CatalogModule is one module and its versions, newest first.
type CatalogModule struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Summary     string       `json:"summary"`               // a line
	Description string       `json:"description,omitempty"` // plain text
	Icon        *CatalogFile `json:"icon,omitempty"`        // PNG, at most 256 KiB; never a brand's logo
	// Screenshots are PNG or JPEG pictures of the module at work, at most 8 of 2 MiB each. They
	// are fetched by the server when someone opens the module's page, and served from there.
	Screenshots []CatalogFile `json:"screenshots,omitempty"`
	Publisher   Publisher     `json:"publisher"`
	Repo        string        `json:"repo"` // owner/name on GitHub, where its source and releases are
	Group       string        `json:"group,omitempty"`
	Libraries   []string      `json:"libraries,omitempty"`
	Requires    []string      `json:"requires,omitempty"`
	Core        bool          `json:"core,omitempty"`
	Needs       []string      `json:"needs,omitempty"`
	Updates     string        `json:"updates,omitempty"`
	Stays       string        `json:"stays,omitempty"`

	Versions []CatalogVersion `json:"versions"`
}

// Catalog is the whole file. Serial only ever goes up, and a server keeps the highest it
// has seen, so nobody can bring back a withdrawn version by replaying an older catalogue.
type Catalog struct {
	Format    int             `json:"format"`
	Name      string          `json:"name"`
	Serial    int64           `json:"serial"`
	Generated string          `json:"generated"` // RFC 3339
	Modules   []CatalogModule `json:"modules"`
}

// ErrCatalogSignature is a catalogue that none of the keys signed.
var ErrCatalogSignature = errors.New("the catalogue's signature doesn't check out")

// VerifyCatalog checks a catalogue's signature, then reads it. Nothing in it is looked at
// before the signature passes.
func VerifyCatalog(raw, sig []byte, keys []ed25519.PublicKey) (*Catalog, error) {
	if int64(len(raw)) > MaxCatalog {
		return nil, fmt.Errorf("the catalogue is larger than %d MiB", MaxCatalog>>20)
	}
	s, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(sig)))
	if err != nil || len(s) != ed25519.SignatureSize {
		return nil, ErrCatalogSignature
	}
	for _, k := range keys {
		if ed25519.Verify(k, raw, s) {
			return ParseCatalog(raw)
		}
	}
	return nil, ErrCatalogSignature
}

// ParseCatalog reads and checks a catalogue without looking at its signature, which is
// for the tools that write one. A server uses VerifyCatalog.
func ParseCatalog(b []byte) (*Catalog, error) {
	var head struct {
		Format int `json:"format"`
	}
	if err := json.Unmarshal(b, &head); err != nil {
		return nil, fmt.Errorf("unreadable catalogue: %w", err)
	}
	if head.Format > CatalogFormat {
		return nil, fmt.Errorf("this catalogue is format %d, and this server reads up to format %d: a newer server can read it", head.Format, CatalogFormat)
	}
	var c Catalog
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("unreadable catalogue: %w", err)
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

var (
	platformRe = regexp.MustCompile(`^(?:windows|darwin|linux|freebsd)-[a-z0-9]{2,16}$`)
	sha256Re   = regexp.MustCompile(`^[0-9a-f]{64}$`)
	dateRe     = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)
	wordRe     = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)
)

// Validate checks that a catalogue makes sense on its own.
func (c *Catalog) Validate() error {
	switch {
	case c.Format < 1 || c.Format > CatalogFormat:
		return fmt.Errorf("catalogue: format %d isn't one this reads", c.Format)
	case !plainText(c.Name, 200, false) || c.Name == "":
		return errors.New("catalogue: it needs a name, in one line of plain text")
	case c.Serial < 1:
		return errors.New("catalogue: serial must be 1 or more")
	case len(c.Modules) > maxCatalogModules:
		return errors.New("catalogue: too many modules")
	}
	if _, err := time.Parse(time.RFC3339, c.Generated); err != nil {
		return fmt.Errorf("catalogue: generated %q isn't a time", c.Generated)
	}
	seen := map[string]bool{}
	for i := range c.Modules {
		mod := &c.Modules[i]
		if seen[mod.ID] {
			return fmt.Errorf("catalogue: %s is listed twice", mod.ID)
		}
		seen[mod.ID] = true
		if err := mod.validate(); err != nil {
			return fmt.Errorf("catalogue: %w", err)
		}
	}
	return nil
}

func (m *CatalogModule) validate() error {
	switch {
	case !idRe.MatchString(m.ID):
		return fmt.Errorf("%q is not a module id (lowercase letters and digits)", m.ID)
	case m.Name == "" || !plainText(m.Name, 64, false):
		return fmt.Errorf("%s: it needs a name, in one line of plain text", m.ID)
	case !plainText(m.Summary, maxText, false):
		return fmt.Errorf("%s: its summary isn't one line of plain text", m.ID)
	case !plainText(m.Description, maxCatalogLong, true):
		return fmt.Errorf("%s: its description isn't plain text of at most %d KiB", m.ID, maxCatalogLong>>10)
	case m.Publisher.Name == "" || !plainText(m.Publisher.Name, 200, false):
		return fmt.Errorf("%s: it needs a publisher's name, in one line of plain text", m.ID)
	case !repoRe.MatchString(m.Repo):
		return fmt.Errorf("%s: %q is not a GitHub repository (owner/name)", m.ID, m.Repo)
	case m.Group != "" && !wordRe.MatchString(m.Group):
		return fmt.Errorf("%s: %q is not a group", m.ID, m.Group)
	case !plainText(m.Updates, maxText, true) || !plainText(m.Stays, maxText, true):
		return fmt.Errorf("%s: what it says about updates isn't plain text", m.ID)
	case len(m.Libraries) > maxListed || len(m.Requires) > maxListed || len(m.Needs) > maxListed:
		return fmt.Errorf("%s: too many entries in a list", m.ID)
	case len(m.Versions) > maxCatalogVersions:
		return fmt.Errorf("%s: more than %d versions", m.ID, maxCatalogVersions)
	}
	if m.Publisher.URL != "" {
		if u, err := url.Parse(m.Publisher.URL); err != nil || u.Scheme != "https" || u.Host == "" {
			return fmt.Errorf("%s: its publisher's address isn't https", m.ID)
		}
	}
	if m.Publisher.Key != "" {
		if k, err := base64.StdEncoding.DecodeString(m.Publisher.Key); err != nil || len(k) != ed25519.PublicKeySize {
			return fmt.Errorf("%s: its publisher's key isn't a public key", m.ID)
		}
	}
	if m.Icon != nil {
		if err := m.Icon.validate(maxIcon); err != nil {
			return fmt.Errorf("%s: its icon: %w", m.ID, err)
		}
	}
	if len(m.Screenshots) > maxScreenshots {
		return fmt.Errorf("%s: more than %d screenshots", m.ID, maxScreenshots)
	}
	for i := range m.Screenshots {
		if err := m.Screenshots[i].validate(maxScreenshot); err != nil {
			return fmt.Errorf("%s: a screenshot: %w", m.ID, err)
		}
	}
	for _, s := range append(slices.Clone(m.Libraries), m.Requires...) {
		if !wordRe.MatchString(s) {
			return fmt.Errorf("%s: %q is not a library kind or a module", m.ID, s)
		}
	}
	for _, s := range m.Needs {
		if !plainText(s, maxText, false) {
			return fmt.Errorf("%s: a need isn't one line of plain text", m.ID)
		}
	}
	seen := map[string]bool{}
	for i := range m.Versions {
		v := &m.Versions[i]
		if seen[v.Version] {
			return fmt.Errorf("%s: version %s is listed twice", m.ID, v.Version)
		}
		seen[v.Version] = true
		if err := v.validate(); err != nil {
			return fmt.Errorf("%s %s: %w", m.ID, v.Version, err)
		}
	}
	return nil
}

func (v *CatalogVersion) validate() error {
	man := Manifest{ID: "catalog", Name: "catalog", Version: v.Version, Repo: "a/b", Delivery: v.Delivery, Entry: "entry",
		API: v.API, MinServer: v.MinServer, MaxServer: v.MaxServer}
	if err := man.ValidateRelease(); err != nil {
		return errors.New(strings.TrimPrefix(err.Error(), "manifest of catalog: "))
	}
	switch {
	case v.License == "" || !plainText(v.License, 200, false):
		return errors.New("it needs a licence")
	case !dateRe.MatchString(v.Released):
		return fmt.Errorf("released %q isn't a date (YYYY-MM-DD)", v.Released)
	case !plainText(v.Changelog, maxCatalogLong, true):
		return fmt.Errorf("its changelog isn't plain text of at most %d KiB", maxCatalogLong>>10)
	case len(v.Permissions) > maxListed:
		return errors.New("too many permissions")
	case len(v.Platforms) == 0 || len(v.Platforms) > 32:
		return errors.New("it needs an archive for at least one platform")
	}
	for _, p := range v.Permissions {
		if !wordRe.MatchString(p) { // one this server doesn't know is for CanRun to turn down
			return fmt.Errorf("%q is not a permission", p)
		}
	}
	if _, err := time.Parse(time.DateOnly, v.Released); err != nil {
		return fmt.Errorf("released %q isn't a date", v.Released)
	}
	for what, ref := range map[string]string{"sums": v.Sums, "sig": v.Sig, "manifest": v.Manifest} {
		if !ValidCatalogURL(ref) {
			return fmt.Errorf("its %s is at %q, which isn't https or a path inside the catalogue's folder", what, ref)
		}
	}
	for p, f := range v.Platforms {
		web := v.Delivery == DeliveryWeb
		if (web && p != PlatformAll) || (!web && !platformRe.MatchString(p)) {
			return fmt.Errorf("%q isn't a platform for something delivered as %s", p, v.Delivery)
		}
		if err := f.validate(maxArchiveSize); err != nil {
			return fmt.Errorf("its %s archive: %w", p, err)
		}
	}
	if ps := statusProblems(v.Status, v.StatusNote); len(ps) > 0 {
		return errors.New(ps[0].String())
	}
	if v.UI != nil {
		if err := v.UI.validate(); err != nil {
			return err
		}
	}
	if w := v.Withdrawn; w != nil {
		if (w.Severity != SeverityBroken && w.Severity != SeveritySecurity) || w.Reason == "" || !plainText(w.Reason, maxText, true) {
			return errors.New("it is withdrawn with no readable severity and reason")
		}
	}
	return nil
}

// maxArchiveSize is the largest archive a server takes.
const maxArchiveSize = 256 << 20

func (f *CatalogFile) validate(limit int64) error {
	switch {
	case !ValidCatalogURL(f.URL):
		return fmt.Errorf("%q isn't https or a path inside the catalogue's folder", f.URL)
	case f.Size < 1 || f.Size > limit:
		return fmt.Errorf("a size of %d isn't between 1 and %d", f.Size, limit)
	case !sha256Re.MatchString(f.SHA256):
		return errors.New("no SHA-256, in lowercase hex")
	}
	return nil
}

// ValidCatalogURL says whether a catalogue may point at this: an https address, or a path
// below the folder the catalogue is in.
func ValidCatalogURL(s string) bool {
	if s == "" || len(s) > maxCatalogURL || strings.ContainsAny(s, "\\\x00 ") {
		return false
	}
	u, err := url.Parse(s)
	if err != nil || u.User != nil || u.Fragment != "" {
		return false
	}
	if u.IsAbs() {
		return u.Scheme == "https" && u.Host != ""
	}
	if u.Host != "" || u.RawQuery != "" || strings.HasPrefix(s, "/") {
		return false
	}
	for _, part := range strings.Split(u.Path, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

// ResolveCatalogURL turns what a catalogue says into an address, given where the catalogue
// itself came from.
func ResolveCatalogURL(catalog, ref string) (string, error) {
	if !ValidCatalogURL(ref) {
		return "", fmt.Errorf("%q isn't https or a path inside the catalogue's folder", ref)
	}
	base, err := url.Parse(catalog)
	if err != nil {
		return "", err
	}
	r, err := url.Parse(ref)
	if err != nil {
		return "", err
	}
	return base.ResolveReference(r).String(), nil
}

// plainText says whether a catalogue's text is what a page can show as it is: valid UTF-8
// with no control characters, and line breaks only where they're expected.
func plainText(s string, limit int, lines bool) bool {
	if len(s) > limit || !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if r == 0x7f || (r < ' ' && !(lines && (r == '\n' || r == '\t'))) {
			return false
		}
	}
	return true
}

// Find looks a module up.
func (c *Catalog) Find(id string) *CatalogModule {
	for i := range c.Modules {
		if c.Modules[i].ID == id {
			return &c.Modules[i]
		}
	}
	return nil
}

// Find looks a version up.
func (m *CatalogModule) Find(version string) *CatalogVersion {
	for i := range m.Versions {
		if Compare(m.Versions[i].Version, version) == 0 {
			return &m.Versions[i]
		}
	}
	return nil
}

// Platform is a version's archive for a system, or nil if it has none.
func (v *CatalogVersion) Platform(goos, goarch string) *CatalogFile {
	key := goos + "-" + goarch
	if v.Delivery == DeliveryWeb {
		key = PlatformAll
	}
	if f, ok := v.Platforms[key]; ok {
		return &f
	}
	return nil
}

// Supports says, in words an owner can read, why this version can't run on a server of the
// given version and module API range; nil when it can. It answers as the manifest would.
func (v *CatalogVersion) Supports(name, server string, minAPI, maxAPI int) error {
	m := Manifest{Name: name, Version: v.Version, API: v.API, MinServer: v.MinServer, MaxServer: v.MaxServer}
	return m.Supports(server, minAPI, maxAPI)
}

// MatchesManifest compares what the catalogue said with what the release itself signed.
// Any difference refuses the install, so a catalogue can't describe a version as asking for
// less than it does, or as fitting a server it doesn't.
func (v *CatalogVersion) MatchesManifest(id string, m *Manifest) error {
	a, b := slices.Clone(v.Permissions), slices.Clone(m.Permissions)
	sort.Strings(a)
	sort.Strings(b)
	switch {
	case m.ID != id:
		return fmt.Errorf("the catalogue lists it as %s, and the release says %s", id, m.ID)
	case m.Version != v.Version:
		return fmt.Errorf("the catalogue lists it as version %s, and the release says %s", v.Version, m.Version)
	case m.Delivery != v.Delivery:
		return fmt.Errorf("the catalogue says it is delivered as %s, and the release says %s", v.Delivery, m.Delivery)
	case m.API != v.API:
		return fmt.Errorf("the catalogue says it uses module API %d, and the release says %d", v.API, m.API)
	case m.MinServer != v.MinServer || m.MaxServer != v.MaxServer:
		return fmt.Errorf("the catalogue says it fits servers %s to %s, and the release says %s to %s", v.MinServer, v.MaxServer, m.MinServer, m.MaxServer)
	case m.License != v.License:
		return fmt.Errorf("the catalogue says its licence is %s, and the release says %s", v.License, m.License)
	case !slices.Equal(a, b):
		return fmt.Errorf("the catalogue says it asks for [%s], and the release asks for [%s]", strings.Join(a, " "), strings.Join(b, " "))
	}
	return nil
}

// Put adds a version to a module, adding the module if it's new, and keeps the newest first.
// A version already there has to be the same one, since versions never change after they
// are released; the one thing that may change is its being withdrawn. It reports whether
// anything changed, and only then does the serial go up.
func (c *Catalog) Put(mod CatalogModule, v CatalogVersion) (bool, error) {
	if err := v.validate(); err != nil {
		return false, fmt.Errorf("%s %s: %w", mod.ID, v.Version, err)
	}
	have := c.Find(mod.ID)
	if have == nil {
		c.Modules = append(c.Modules, mod)
		sort.Slice(c.Modules, func(i, j int) bool { return c.Modules[i].ID < c.Modules[j].ID })
		have = c.Find(mod.ID)
		have.Versions = nil
	}
	if old := have.Find(v.Version); old != nil {
		a, _ := json.Marshal(old.Platforms)
		b, _ := json.Marshal(v.Platforms)
		if !bytes.Equal(a, b) {
			return false, fmt.Errorf("%s %s is already listed with other files, and a version never changes", mod.ID, v.Version)
		}
		return false, nil
	}
	have.Versions = append(have.Versions, v)
	sort.SliceStable(have.Versions, func(i, j int) bool { return Compare(have.Versions[i].Version, have.Versions[j].Version) > 0 })
	// What a module says about itself comes from its newest version. Words it leaves out stay as they
	// are, so a version with less to say can't blank what someone wrote for the store.
	if have.Versions[0].Version == v.Version {
		have.Name, have.Repo, have.Core, have.Requires = mod.Name, mod.Repo, mod.Core, mod.Requires
		for _, f := range []struct {
			to   *string
			from string
		}{{&have.Summary, mod.Summary}, {&have.Group, mod.Group}, {&have.Updates, mod.Updates}, {&have.Stays, mod.Stays}} {
			if f.from != "" {
				*f.to = f.from
			}
		}
		if len(mod.Libraries) > 0 {
			have.Libraries = mod.Libraries
		}
		if len(mod.Needs) > 0 {
			have.Needs = mod.Needs
		}
	}
	c.Serial++
	return true, nil
}

// HashHex is the form a catalogue writes a SHA-256 in.
func HashHex(sum []byte) string { return hex.EncodeToString(sum) }
