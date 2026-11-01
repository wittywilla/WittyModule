// Package wittymodule is what a WittyMediaServer plugin needs to be a project of
// its own: the manifest, the rules for version ranges, the protocol a helper
// process speaks, and the names of a release's files. It has no dependencies, so
// a plugin that uses it still has none of its own.
package wittymodule

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// The module API: the contract between the server and its plugins. A plugin
// names the version it was built for, and a server takes MinAPI to API. Adding
// to the contract raises API only; breaking it raises MinAPI too, which disables
// older plugins, so that is done rarely.
//
// API 2 adds web bundles, programs, attached helpers and streams to API 1. Nothing
// in API 1 changed, so a version built for it still runs. API 3 adds notifications from
// a helper and the pages a module adds to the app, and is the first API where nothing of a
// module is compiled into the server, so it is also the oldest one a server takes: a module
// has to be built for WittyMediaServer 0.6.0-alpha or newer.
const (
	API    = 3
	MinAPI = 3
)

// FirstServer is the first server this API is for, and what a module's minServer starts from.
const FirstServer = "0.6.0-alpha"

// How a release of a plugin reaches the server.
const (
	DeliveryHelper  = "helper"  // a program for each platform, run as a separate process
	DeliveryProgram = "program" // a program for each platform that the server runs for one job, in its own protocol
	DeliveryWeb     = "web"     // a bundle the browser loads
	DeliveryLibrary = "library" // source only, linked into others: it reaches a server inside them
	DeliveryBuiltin = "builtin" // not converted yet: it updates with the server
)

// Permissions a version can ask for. A helper has neither unless it asks, in its
// signed manifest, and the server also agrees that its kind of plugin needs it.
const (
	PermissionNetwork  = "network"  // it may reach the network
	PermissionPrograms = "programs" // it may start other programs, such as FFmpeg
	PermissionDevices  = "devices"  // it may open this machine's devices, such as a tuner or a graphics card
	PermissionData     = "data"     // it may write to a folder of its own, which it is told of in WITTYMODULE_DATA
)

// ManifestFile is the manifest's name in a plugin's folder and in a release.
const ManifestFile = "module.json"

// NoticesFile holds the licenses of the libraries in a module's programs.
const NoticesFile = "THIRD-PARTY-NOTICES.txt"

// Manifest describes one version of a plugin.
type Manifest struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Version  string `json:"version"`
	Repo     string `json:"repo"` // owner/name on GitHub, where its releases are
	License  string `json:"license,omitempty"`
	Delivery string `json:"delivery"`
	// Entry is the program, without .exe, or a web bundle's script.
	Entry string `json:"entry,omitempty"`
	Style string `json:"style,omitempty"` // a web bundle's stylesheet, if it has one
	// Programs are more programs that come with a helper, without .exe, built from cmd/<name>.
	Programs []string `json:"programs,omitempty"`
	// WebEntry and WebStyle are the script and stylesheet of a helper that also has a part in
	// the browser. Those files sit in web/ inside each platform's archive.
	WebEntry  string `json:"webEntry,omitempty"`
	WebStyle  string `json:"webStyle,omitempty"`
	API       int    `json:"api"`
	MinServer string `json:"minServer"`
	MaxServer string `json:"maxServer"`
	// Capabilities are facts the server needs without starting the helper, such
	// as "rawExtensions". Signed along with everything else.
	Capabilities map[string][]string `json:"capabilities,omitempty"`
	Permissions  []string            `json:"permissions,omitempty"`

	// DataFormat numbers the shape of the data this version keeps, and DataFrom is
	// the oldest it can still read. Going back to a version that can't read what is
	// there is refused, with the reason. Zero means it keeps none of its own.
	DataFormat int `json:"dataFormat,omitempty"`
	DataFrom   int `json:"dataFrom,omitempty"`

	// For the people choosing versions: what it does, which group it belongs in,
	// what it needs in order to work, which parts of it a release of its own
	// really changes, and which stay with the server.
	Group   string   `json:"group,omitempty"`
	Summary string   `json:"summary,omitempty"`
	Needs   []string `json:"needs,omitempty"`
	Updates string   `json:"updates,omitempty"`
	Stays   string   `json:"stays,omitempty"`

	// Libraries are the kinds of library that need this plugin (any one plugin listing
	// a kind is enough). Requires names plugins it is built on. Core is one that other
	// plugins can't do without, so it can't be taken off a server.
	Libraries []string `json:"libraries,omitempty"`
	Requires  []string `json:"requires,omitempty"`
	Core      bool     `json:"core,omitempty"`

	// Pages are the pages it adds to the app, which its web bundle draws.
	Pages []Page `json:"pages,omitempty"`
}

// Where a module's page is listed in the app.
const (
	PageMain     = "main"     // with the app's sections
	PageSettings = "settings" // in Settings
)

// Page is a page a module adds to the app, at /m/<module>/<id>. Whether someone may see it
// is the server's to check, and Admin only asks for that check: it never widens what the
// bundle can do, since a bundle runs with the access of whoever is signed in.
type Page struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Where string `json:"where"`
	Icon  string `json:"icon,omitempty"` // a name from the app's own set of icons
	Admin bool   `json:"admin,omitempty"`
}

var (
	idRe    = regexp.MustCompile(`^[a-z][a-z0-9]{1,63}$`)
	entryRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
	repoRe  = regexp.MustCompile(`^[A-Za-z0-9-]+/[A-Za-z0-9._-]+$`)
)

// Limits on what a manifest may carry: each is shown or kept by the server.
const (
	maxText         = 2000
	maxListed       = 64
	maxCapabilities = 4096
)

// ParseManifest reads and checks a manifest.
func ParseManifest(b []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("unreadable manifest: %w", err)
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return &m, nil
}

// Validate checks that a manifest makes sense on its own.
func (m *Manifest) Validate() error {
	name := m.ID
	if name == "" {
		name = "the module"
	}
	switch {
	case !idRe.MatchString(m.ID):
		return fmt.Errorf("manifest: %q is not a module id (lowercase letters and digits)", m.ID)
	case m.Name == "":
		return fmt.Errorf("manifest of %s: no name", name)
	case !Valid(m.Version):
		return fmt.Errorf("manifest of %s: %q is not a version", name, m.Version)
	case !repoRe.MatchString(m.Repo):
		return fmt.Errorf("manifest of %s: %q is not a GitHub repository (owner/name)", name, m.Repo)
	case m.API < 1:
		return fmt.Errorf("manifest of %s: api must be 1 or more", name)
	case !Valid(m.MinServer):
		return fmt.Errorf("manifest of %s: minServer %q is not a version", name, m.MinServer)
	}
	if _, _, ok := parseMax(m.MaxServer); !ok {
		return fmt.Errorf("manifest of %s: maxServer %q is not a version or a series such as 0.5.x", name, m.MaxServer)
	}
	if !atMost(m.MinServer, m.MaxServer) {
		return fmt.Errorf("manifest of %s: minServer %s is past maxServer %s", name, m.MinServer, m.MaxServer)
	}
	switch m.Delivery {
	case DeliveryHelper, DeliveryProgram:
		if !entryRe.MatchString(m.Entry) {
			return fmt.Errorf("manifest of %s: entry %q is not a program name", name, m.Entry)
		}
	case DeliveryWeb:
		// Empty is allowed while a bundle has never been released; a release needs one
		// (ValidateRelease).
		for _, f := range []string{m.Entry, m.Style} {
			if f != "" && !entryRe.MatchString(f) {
				return fmt.Errorf("manifest of %s: %q is not a file name in the bundle", name, f)
			}
		}
	case DeliveryLibrary, DeliveryBuiltin:
	default:
		return fmt.Errorf("manifest of %s: delivery %q is not helper, program, web, library or builtin", name, m.Delivery)
	}
	if m.DataFormat < 0 || m.DataFrom < 0 || m.DataFrom > m.DataFormat {
		return fmt.Errorf("manifest of %s: dataFrom %d and dataFormat %d don't make sense", name, m.DataFrom, m.DataFormat)
	}
	for _, s := range []string{m.Group, m.Summary, m.Updates, m.Stays} {
		if len(s) > maxText {
			return fmt.Errorf("manifest of %s: a description is too long", name)
		}
	}
	if len(m.Needs) > maxListed || len(m.Permissions) > maxListed || len(m.Libraries) > maxListed || len(m.Requires) > maxListed {
		return fmt.Errorf("manifest of %s: too many entries in a list", name)
	}
	for _, s := range m.Needs {
		if len(s) > maxText {
			return fmt.Errorf("manifest of %s: a need is described at too great a length", name)
		}
	}
	for _, p := range m.Permissions {
		switch p {
		case PermissionNetwork, PermissionPrograms, PermissionDevices, PermissionData:
		default:
			return fmt.Errorf("manifest of %s: %q is not a permission", name, p)
		}
	}
	if len(m.Pages) > 0 && !m.HasWeb() {
		return fmt.Errorf("manifest of %s: only a web bundle can draw a page", name)
	}
	if len(m.Programs) > maxListed {
		return fmt.Errorf("manifest of %s: too many programs", name)
	}
	for _, p := range m.Programs {
		if !entryRe.MatchString(p) || !m.PerPlatform() {
			return fmt.Errorf("manifest of %s: %q is not a program that can come with it", name, p)
		}
	}
	for _, f := range []string{m.WebEntry, m.WebStyle} {
		if f != "" && (!entryRe.MatchString(f) || !m.PerPlatform()) {
			return fmt.Errorf("manifest of %s: %q is not a file of a helper's web part", name, f)
		}
	}
	if len(m.Pages) > 16 {
		return fmt.Errorf("manifest of %s: too many pages", name)
	}
	seen := map[string]bool{}
	for _, p := range m.Pages {
		switch {
		case !wordRe.MatchString(p.ID) || seen[p.ID]:
			return fmt.Errorf("manifest of %s: %q is not a page's id, or there are two of it", name, p.ID)
		case p.Title == "" || !plainText(p.Title, 64, false):
			return fmt.Errorf("manifest of %s: page %s needs a title, in one line of plain text", name, p.ID)
		case p.Where != PageMain && p.Where != PageSettings:
			return fmt.Errorf("manifest of %s: page %s is listed in %q, not %s or %s", name, p.ID, p.Where, PageMain, PageSettings)
		case p.Icon != "" && !wordRe.MatchString(p.Icon):
			return fmt.Errorf("manifest of %s: page %s has %q for an icon", name, p.ID, p.Icon)
		}
		seen[p.ID] = true
	}
	total := 0
	for k, v := range m.Capabilities {
		total += len(k) + len(v)
		for _, s := range v {
			total += len(s)
		}
	}
	if total > maxCapabilities {
		return fmt.Errorf("manifest of %s: its capabilities are too large", name)
	}
	return nil
}

// Asks reports whether this version asks for a permission.
func (m *Manifest) Asks(permission string) bool {
	for _, p := range m.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// HasWeb says whether a version has a part that runs in the browser.
func (m *Manifest) HasWeb() bool { return m.Delivery == DeliveryWeb || m.WebEntry != "" }

// WebFolder is ProgramFile's folder for that part: where its files sit inside the version.
func (m *Manifest) WebFolder() string {
	if m.Delivery == DeliveryWeb {
		return ""
	}
	return "web/"
}

// ProgramFile is a program's file name on a system.
func ProgramFile(name, goos string) string {
	if goos == "windows" {
		return name + ".exe"
	}
	return name
}

// PerPlatform says whether a version has a build for each platform, rather than
// one bundle for all.
func (m *Manifest) PerPlatform() bool {
	return m.Delivery == DeliveryHelper || m.Delivery == DeliveryProgram
}

// ValidateRelease adds what only a release must have: something a server can run.
func (m *Manifest) ValidateRelease() error {
	if err := m.Validate(); err != nil {
		return err
	}
	switch m.Delivery {
	case DeliveryHelper, DeliveryProgram:
	case DeliveryWeb:
		if m.Entry == "" {
			return fmt.Errorf("manifest of %s: a web bundle needs an entry", m.ID)
		}
	default:
		return fmt.Errorf("%s is delivered as %q, so a server has nothing to install from a release of it", m.ID, m.Delivery)
	}
	return nil
}

// Supports says, in words an owner can read, why this version can't run on a
// server of the given version and module API range; nil when it can.
func (m *Manifest) Supports(server string, minAPI, maxAPI int) error {
	if !Valid(server) {
		return fmt.Errorf("this server's version (%q) can't be compared, so it uses its built-in %s", server, m.Name)
	}
	if Compare(server, m.MinServer) < 0 {
		return fmt.Errorf("%s %s needs WittyMediaServer %s or newer, and this server is %s", m.Name, m.Version, m.MinServer, server)
	}
	if !atMost(server, m.MaxServer) {
		return fmt.Errorf("%s %s supports WittyMediaServer up to %s, and this server is %s", m.Name, m.Version, m.MaxServer, server)
	}
	if m.API > maxAPI {
		return fmt.Errorf("%s %s needs a newer server: it uses module API %d, and this server has up to %d", m.Name, m.Version, m.API, maxAPI)
	}
	if m.API < minAPI {
		return fmt.Errorf("%s %s is too old for this server: it uses module API %d, and this server needs at least %d. It was built before modules became separate from the server, so install a version made for WittyMediaServer %s or newer", m.Name, m.Version, m.API, minAPI, FirstServer)
	}
	return nil
}

// EntryFile is the helper program's file name on a system.
func (m *Manifest) EntryFile(goos string) string {
	if goos == "windows" {
		return m.Entry + ".exe"
	}
	return m.Entry
}

// Target is a platform a helper is built for.
type Target struct{ GOOS, GOARCH string }

// Targets are the platforms the server ships to, as in scripts/build-release.sh.
var Targets = []Target{{"windows", "amd64"}, {"windows", "arm64"}, {"darwin", "amd64"}, {"darwin", "arm64"},
	{"linux", "amd64"}, {"linux", "arm64"}, {"linux", "arm"}, {"linux", "386"}, {"freebsd", "amd64"}, {"freebsd", "arm64"}}

// The names of a release's files.

// TopFolder is the one folder inside a release's archive.
func TopFolder(id, ver, goos, goarch string) string {
	label := goarch
	if goos == "linux" && goarch == "arm" {
		label = "armv7"
	}
	return id + "-" + strings.TrimPrefix(ver, "v") + "-" + goos + "-" + label
}

// ArchiveName is the release archive for a platform.
func ArchiveName(id, ver, goos, goarch string) string {
	ext := ".tar.gz"
	if goos == "windows" {
		ext = ".zip"
	}
	return TopFolder(id, ver, goos, goarch) + ext
}

// SumsName is the signed list of a release's hashes; its signature adds .sig.
func SumsName(id, ver string) string {
	return id + "-" + strings.TrimPrefix(ver, "v") + "-SHA256SUMS.txt"
}

// ManifestName is the copy of the manifest attached to a release, so it can be
// judged before anything large is downloaded.
func ManifestName(id, ver string) string {
	return id + "-" + strings.TrimPrefix(ver, "v") + "-module.json"
}

// WebTopFolder is the one folder inside a web bundle's archive, the same for every
// platform.
func WebTopFolder(id, ver string) string {
	return id + "-" + strings.TrimPrefix(ver, "v") + "-web"
}

// WebArchiveName is the archive of a web bundle.
func WebArchiveName(id, ver string) string { return WebTopFolder(id, ver) + ".tar.gz" }

// Top is the folder a version's files sit in inside its archive, and in its signed list.
func (m *Manifest) Top(goos, goarch string) string {
	if m.Delivery == DeliveryWeb {
		return WebTopFolder(m.ID, m.Version)
	}
	return TopFolder(m.ID, m.Version, goos, goarch)
}

// Archive is the archive holding this version for a platform.
func (m *Manifest) Archive(goos, goarch string) string {
	if m.Delivery == DeliveryWeb {
		return WebArchiveName(m.ID, m.Version)
	}
	return ArchiveName(m.ID, m.Version, goos, goarch)
}
