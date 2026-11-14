// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors

package wittymodule

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Module API 5 makes a module own its whole feature: its HTTP routes, the screens it adds to the
// app, its data and what happens when it is installed, upgraded or taken off. It says all of that
// in integration.json beside module.json and sdui.json: the module's "install script", as data a
// server checks and applies rather than code it runs. Code that has to run at install (a data
// migration) runs in the module's own sandbox, as lifecycle calls to its service.
//
// The server's side is WittyMediaServer docs/MODULES-API5.md; the plan is
// docs/plans/MODULAR-MODULES.md.

// IntegrationFile is its name in a module's folder and at the root of every archive.
const IntegrationFile = "integration.json"

// IntegrationFromAPI is the module API from which a module that adds anything to a server must
// carry integration.json.
const IntegrationFromAPI = 5

// MaxIntegration is the most a server reads of one.
const MaxIntegration = 256 << 10

// Integration is integration.json.
type Integration struct {
	Schema        string `json:"$schema,omitempty"`
	Format        string `json:"format"`        // always IntegrationFormat
	FormatVersion int    `json:"formatVersion"` // always 1
	Module        string `json:"module"`        // the module's id, as in module.json

	// Host is the host features it needs, as ranges: "http", "web", "sdui", "host". A server
	// that offers less (or more than a range allows) doesn't run this version, and says why.
	Host map[string]string `json:"host,omitempty"`
	// Feature is the parental and profile-limits feature everything here belongs to, such as
	// "games" or "livetv": a profile without it can't reach the routes or see the screens.
	Feature string `json:"feature,omitempty"`

	HTTP      []Route    `json:"http,omitempty"`
	Web       *WebPart   `json:"web,omitempty"`
	SDUI      *SDUIPart  `json:"sdui,omitempty"`
	Data      *DataPart  `json:"data,omitempty"`
	Lifecycle *Lifecycle `json:"lifecycle,omitempty"`
	Live      *LivePart  `json:"live,omitempty"`
	// Network is what the module reaches on the internet by itself.
	Network *NetworkPart `json:"network,omitempty"`
}

// NetworkPart lists every host a module contacts by itself, each with why: shown at install,
// on the server's privacy list, and enforced where a system's sandbox can.
type NetworkPart struct {
	Hosts []NetworkHost `json:"hosts,omitempty"`
}

// NetworkHost is one host.
type NetworkHost struct {
	Host string `json:"host"` // api.example.com, or *.example.com
	Why  string `json:"why"`  // in plain words, at most 200 characters
}

// IntegrationFormat is integration.json's format.
const IntegrationFormat = "wittymodule-integration"

// Route is one HTTP route the module answers. The server mounts it at /api/modules/<id><path>
// (and /api/v1/modules/...), checks who may call it before the module hears of the request, and
// passes it to the module's service at /http<path>. Apps call it there: a module's API is its own.
type Route struct {
	Path    string   `json:"path"`    // "/games/{id}/play": Go ServeMux style, starting with "/"
	Methods []string `json:"methods"` // GET, HEAD, POST, PUT, PATCH, DELETE
	// Auth is who may call it: user (anyone signed in), admin, ticket (no sign-in; the module
	// checks a secret in the path, and the answer is only ever shown in a sandboxed frame).
	Auth string `json:"auth"`
	// Item names a path parameter holding a library item's id. The server checks the profile
	// may see that item, and tells the module about it (X-Witty-Item) with its files.
	Item string `json:"item,omitempty"`
	// Upgrade allows a WebSocket on this route.
	Upgrade string `json:"upgrade,omitempty"` // "websocket"
	// Stream: the answer is a stream, sent on as it comes, with no size or time limit.
	Stream bool `json:"stream,omitempty"`
	// Plays: calling it starts or goes on playing something, so it counts towards a profile's
	// screen time and is refused when that time is up (leaving, a DELETE, never is).
	Plays bool `json:"plays,omitempty"`
	// Limits on a request and its answer; the server's defaults when left out.
	Limits *Limits `json:"limits,omitempty"`
	// CSP is a policy for an answer that is a page: "sandboxed-page" only, for ticket routes.
	CSP string `json:"csp,omitempty"`
}

// Limits on one route.
type Limits struct {
	Body   string `json:"body,omitempty"`   // "256KB", "4MB": the most a request may carry
	Answer string `json:"answer,omitempty"` // the most an answer may carry (not for streams)
	Time   string `json:"time,omitempty"`   // "45s", "10m": how long an answer may take
}

// Route auth values.
const (
	AuthUser   = "user"
	AuthAdmin  = "admin"
	AuthTicket = "ticket"
)

// WebPart is what the module's web bundle adds to the web app. Its bundle exports register(host)
// to hand over the components; this says what it claims, so a server can check it and apps can
// tell what's missing before any code loads.
type WebPart struct {
	Routes   []WebRoute    `json:"routes,omitempty"`
	Takeover []Takeover    `json:"takeover,omitempty"`
	Cards    []CardHook    `json:"cards,omitempty"`
	Players  []PlayerHook  `json:"players,omitempty"`
	Readers  []ReaderHook  `json:"readers,omitempty"`
	Settings []SettingsTab `json:"settings,omitempty"`
	Setup    []SetupStep   `json:"setup,omitempty"`
	NavFold  []NavFold     `json:"navFold,omitempty"`
}

// WebRoute is a place in the web app the module draws.
type WebRoute struct {
	Path string `json:"path"` // "/wittyplay/*", "/game/:id": react-router style
	// Frame: fullscreen (replaces the app's frame), app (inside it), tv (a screen of the TV frame).
	Frame string `json:"frame"`
}

// Frames.
const (
	FrameFullscreen = "fullscreen"
	FrameApp        = "app"
	FrameTV         = "tv"
)

// Takeover has the module draw every library of a kind, at its own route.
type Takeover struct {
	LibraryKind string `json:"libraryKind"`
	Route       string `json:"route"` // with {library}
}

// CardHook draws cards of a media kind: their art, their detail panel's bio, where they link.
type CardHook struct {
	MediaKind string `json:"mediaKind"`
	Art       bool   `json:"art,omitempty"`
	Detail    bool   `json:"detail,omitempty"`
	Link      bool   `json:"link,omitempty"`
}

// PlayerHook plays items of a media kind.
type PlayerHook struct {
	MediaKind string `json:"mediaKind"`
}

// ReaderHook reads a book format.
type ReaderHook struct {
	Format string `json:"format"` // epub, pdf, cbz
}

// SettingsTab is a section of the web app's Settings the module draws.
type SettingsTab struct {
	ID     string   `json:"id"`
	Label  string   `json:"label"`
	Admin  bool     `json:"admin,omitempty"`
	After  string   `json:"after,omitempty"`  // another section's id
	Search []string `json:"search,omitempty"` // words Settings' search finds it by
	// Launch makes the row open a place of the module's own instead of drawing a section in
	// Settings: one interface, the module's (WittyPlay's store for cores and emulators).
	Launch string `json:"launch,omitempty"`
	// Hidden takes a built-in section of this id out of Settings' menu; its address goes to Launch.
	Hidden bool `json:"hidden,omitempty"`
}

// SetupStep is a step of the first-run setup the module adds (and a "Finish setting up" step
// later, for a module installed or upgraded after setup).
type SetupStep struct {
	ID     string     `json:"id"`
	Title  string     `json:"title,omitempty"`
	After  string     `json:"after,omitempty"`  // a built-in step it goes after
	Before string     `json:"before,omitempty"` // or before; not both
	When   *SetupWhen `json:"when,omitempty"`
	// Page is an SDUI page of the module's (sdui.json) that is the step's form, so every app
	// can draw it; a web bundle may register a richer component for the same id.
	Page string `json:"page,omitempty"`
	// Saves names what the step may change: the module's own settings keys.
	Saves []string `json:"saves,omitempty"`
	// Later says where the step lives in Settings afterwards.
	Later *SetupLater `json:"later,omitempty"`
	// Version is raised when the step changes enough to be shown again to a server that did it.
	Version int `json:"version,omitempty"`
}

// SetupWhen is when a step is shown.
type SetupWhen struct {
	LibraryKinds []string `json:"libraryKinds,omitempty"` // such a library exists or is being added
}

// SetupLater is a step's place in Settings.
type SetupLater struct {
	Where string `json:"where"`
	To    string `json:"to"`
}

// SetupAnchors are the built-in steps a module's step may name.
var SetupAnchors = []string{"modules", "libraries", "details", "people", "remote", "ready"}

// NavFold folds the menu entries of a kind of library into one of the module's.
type NavFold struct {
	LibraryKind string `json:"libraryKind"`
	Into        string `json:"into"` // a navigation entry id from sdui.json
}

// SDUIPart is the module's SDUI ops: commands its pages send, each answered by one of its routes.
type SDUIPart struct {
	Ops []Op `json:"ops,omitempty"`
}

// Op is one SDUI command.
type Op struct {
	ID   string `json:"id"`   // "<module>.<name>"
	HTTP string `json:"http"` // "POST /rooms": a method and one of the module's own route paths
}

// DataPart is the module's own data.
type DataPart struct {
	Store  string `json:"store"`            // "sqlite": a database in its data folder
	Backup string `json:"backup,omitempty"` // which part of a backup it goes in: settings, saves, library
}

// Lifecycle says which lifecycle calls the module answers.
type Lifecycle struct {
	Install   bool   `json:"install,omitempty"`
	Upgrade   bool   `json:"upgrade,omitempty"`
	Enable    bool   `json:"enable,omitempty"`
	Disable   bool   `json:"disable,omitempty"`
	Uninstall string `json:"uninstall,omitempty"` // "", "ask-keep-data", "keep-data", "remove-data"
}

// Lifecycle calls, on the helper protocol.
const (
	CallInstall   = "lifecycle.install"
	CallUpgrade   = "lifecycle.upgrade"
	CallEnable    = "lifecycle.enable"
	CallDisable   = "lifecycle.disable"
	CallUninstall = "lifecycle.uninstall"
)

// LifecycleArgs is what a lifecycle call is given.
type LifecycleArgs struct {
	From     string `json:"from,omitempty"`     // the version before, for upgrade
	To       string `json:"to,omitempty"`       // the version now
	KeepData bool   `json:"keepData,omitempty"` // for uninstall
}

// LivePart is what the module reports while it runs.
type LivePart struct {
	Sessions bool `json:"sessions,omitempty"` // games, streams: counted for metrics, screen time and "playing now"
	Doctor   bool `json:"doctor,omitempty"`   // it answers doctor checks
}

// Host features a server offers, and their versions now.
const (
	HostHTTP = "http" // routes, and what the server checks for them
	HostWeb  = "web"  // the web app's host object (web host API)
	HostSDUI = "sdui" // the SDUI catalogue revision
	HostAPI  = "host" // the host API a module calls back on
)

// NeedsIntegration reports whether a version must carry integration.json.
func (m *Manifest) NeedsIntegration() bool {
	switch m.Delivery {
	case DeliveryHelper, DeliveryProgram, DeliveryWeb:
		return m.API >= IntegrationFromAPI
	}
	return false
}

// ParseIntegration reads integration.json strictly: one object, no unknown field anywhere, no
// more than MaxIntegration.
func ParseIntegration(b []byte) (*Integration, error) {
	if len(b) > MaxIntegration {
		return nil, fmt.Errorf("%s is larger than %d bytes", IntegrationFile, MaxIntegration)
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	var in Integration
	if err := dec.Decode(&in); err != nil {
		return nil, fmt.Errorf("%s: %w", IntegrationFile, err)
	}
	if dec.More() {
		return nil, fmt.Errorf("%s: more than one object", IntegrationFile)
	}
	return &in, nil
}

var (
	routePath  = regexp.MustCompile(`^(/([a-z0-9][a-z0-9._-]*|\{[a-z][A-Za-z0-9]*(\.\.\.)?\}))+/?$`)
	webPath    = regexp.MustCompile(`^(/([a-z0-9][a-z0-9._-]*\*?|:[a-z][A-Za-z0-9]*|\*))+$`) // a segment may end in * (/page/wittyplay-*)
	idName     = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)
	kindName   = regexp.MustCompile(`^[a-z][a-z0-9]{0,31}$`)
	rangeExpr  = regexp.MustCompile(`^(>=|<=|>|<|=)?\d+( (>=|<=|>|<)\d+)?$`)
	sizeExpr   = regexp.MustCompile(`^(\d+)(B|KB|MB|GB)$`)
	hostName   = regexp.MustCompile(`^(\*\.)?([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,63}$`)
	pathParams = regexp.MustCompile(`\{([a-z][A-Za-z0-9]*)(\.\.\.)?\}`)
)

var httpMethods = []string{"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE"}

// Check reports every problem with the file for this manifest, as problems with a field each.
func (in *Integration) Check(m *Manifest) []Problem {
	var ps []Problem
	add := func(field, f string, a ...any) { ps = append(ps, Problem{field, fmt.Sprintf(f, a...)}) }
	if in.Format != IntegrationFormat {
		add("format", "must be %q", IntegrationFormat)
	}
	if in.FormatVersion != 1 {
		add("formatVersion", "must be 1")
	}
	if in.Module != m.ID {
		add("module", "must be this module's id, %q", m.ID)
	}
	for k, v := range in.Host {
		if !slices.Contains([]string{HostHTTP, HostWeb, HostSDUI, HostAPI}, k) {
			add("host."+k, "isn't a host feature (http, web, sdui, host)")
		} else if !rangeExpr.MatchString(v) {
			add("host."+k, "%q isn't a range such as \">=1 <2\"", v)
		}
	}
	if in.Feature != "" && !kindName.MatchString(in.Feature) {
		add("feature", "%q isn't a feature name", in.Feature)
	}

	seen := map[string]bool{}
	for i, r := range in.HTTP {
		f := fmt.Sprintf("http[%d]", i)
		if !routePath.MatchString(r.Path) {
			add(f+".path", "%q isn't a path such as /games/{id}", r.Path)
		}
		if len(r.Methods) == 0 {
			add(f+".methods", "needs at least one")
		}
		for _, meth := range r.Methods {
			if !slices.Contains(httpMethods, meth) {
				add(f+".methods", "%q isn't one of %s", meth, strings.Join(httpMethods, ", "))
			}
			if seen[meth+" "+r.Path] {
				add(f, "%s %s is listed twice", meth, r.Path)
			}
			seen[meth+" "+r.Path] = true
		}
		switch r.Auth {
		case AuthUser, AuthAdmin:
		case AuthTicket:
			if r.Item != "" {
				add(f+".item", "a ticket route can't check an item: nobody is signed in")
			}
			if slices.ContainsFunc(r.Methods, func(s string) bool { return s != "GET" && s != "HEAD" }) {
				add(f+".methods", "a ticket route can only be read")
			}
		default:
			add(f+".auth", "must be user, admin or ticket")
		}
		params := map[string]bool{}
		for _, p := range pathParams.FindAllStringSubmatch(r.Path, -1) {
			params[p[1]] = true
		}
		if r.Item != "" && !params[r.Item] {
			add(f+".item", "%q isn't a parameter of its path", r.Item)
		}
		if r.Upgrade != "" && r.Upgrade != "websocket" {
			add(f+".upgrade", "can only be websocket")
		}
		if r.Upgrade != "" && !slices.Contains(r.Methods, "GET") {
			add(f+".upgrade", "a WebSocket starts with GET")
		}
		if r.CSP != "" && (r.CSP != "sandboxed-page" || r.Auth != AuthTicket) {
			add(f+".csp", "only a ticket route's page can have a policy, and only sandboxed-page")
		}
		if r.Limits != nil {
			for _, s := range []struct{ name, v string }{{"body", r.Limits.Body}, {"answer", r.Limits.Answer}} {
				if s.v != "" {
					if _, err := ParseSize(s.v); err != nil {
						add(f+".limits."+s.name, "%v", err)
					}
				}
			}
			if r.Limits.Time != "" {
				if d, err := time.ParseDuration(r.Limits.Time); err != nil || d <= 0 || d > time.Hour {
					add(f+".limits.time", "%q isn't a time from 1s to 1h", r.Limits.Time)
				}
			}
			if r.Stream && r.Limits.Answer != "" {
				add(f+".limits.answer", "a stream has no answer limit")
			}
		}
	}

	if w := in.Web; w != nil {
		if !m.HasWeb() {
			add("web", "the module has no web part (webEntry or delivery web)")
		}
		for i, r := range w.Routes {
			f := fmt.Sprintf("web.routes[%d]", i)
			if !webPath.MatchString(r.Path) {
				add(f+".path", "%q isn't a path such as /wittyplay/*", r.Path)
			}
			if !slices.Contains([]string{FrameFullscreen, FrameApp, FrameTV}, r.Frame) {
				add(f+".frame", "must be fullscreen, app or tv")
			}
		}
		for i, t := range w.Takeover {
			f := fmt.Sprintf("web.takeover[%d]", i)
			if !kindName.MatchString(t.LibraryKind) {
				add(f+".libraryKind", "%q isn't a kind of library", t.LibraryKind)
			}
			if !strings.Contains(t.Route, "{library}") || !strings.HasPrefix(t.Route, "/") {
				add(f+".route", "must be a path with {library}")
			}
		}
		for i, c := range w.Cards {
			if !kindName.MatchString(c.MediaKind) {
				add(fmt.Sprintf("web.cards[%d].mediaKind", i), "%q isn't a kind", c.MediaKind)
			}
			if !c.Art && !c.Detail && !c.Link {
				add(fmt.Sprintf("web.cards[%d]", i), "draws nothing: set art, detail or link")
			}
		}
		for i, p := range w.Players {
			if !kindName.MatchString(p.MediaKind) {
				add(fmt.Sprintf("web.players[%d].mediaKind", i), "%q isn't a kind", p.MediaKind)
			}
		}
		for i, r := range w.Readers {
			if !kindName.MatchString(r.Format) {
				add(fmt.Sprintf("web.readers[%d].format", i), "%q isn't a format", r.Format)
			}
		}
		for i, s := range w.Settings {
			f := fmt.Sprintf("web.settings[%d]", i)
			if !idName.MatchString(s.ID) {
				add(f+".id", "%q isn't an id", s.ID)
			}
			if s.Label == "" || len(s.Label) > 60 {
				add(f+".label", "needs a label of at most 60 characters")
			}
			if s.Launch != "" && !strings.HasPrefix(s.Launch, "/") {
				add(f+".launch", "must be a place in the app, starting with /")
			}
			if s.Hidden && s.Launch == "" {
				add(f+".hidden", "a hidden section needs launch: where its address goes")
			}
			if len(s.Search) > 32 {
				add(f+".search", "at most 32 words")
			}
		}
		for i, s := range w.Setup {
			f := fmt.Sprintf("web.setup[%d]", i)
			if !idName.MatchString(s.ID) {
				add(f+".id", "%q isn't an id", s.ID)
			}
			if len(s.Title) > 40 {
				add(f+".title", "at most 40 characters")
			}
			if s.After != "" && s.Before != "" {
				add(f, "after or before, not both")
			}
			for _, a := range []string{s.After, s.Before} {
				if a != "" && !slices.Contains(SetupAnchors, a) {
					add(f, "%q isn't a built-in step a module can name (%s)", a, strings.Join(SetupAnchors, ", "))
				}
			}
			if s.Before == "modules" {
				add(f+".before", "a module's step can't come before modules are chosen")
			}
			if s.After == "ready" {
				add(f+".after", "nothing comes after ready")
			}
			if s.Page != "" && s.Page != m.ID && !strings.HasPrefix(s.Page, m.ID+"-") {
				add(f+".page", "must be one of this module's pages (%s or %s-…)", m.ID, m.ID)
			}
			if s.Later != nil && (s.Later.Where == "" || !strings.HasPrefix(s.Later.To, "/settings")) {
				add(f+".later", "needs where, and a place under /settings")
			}
		}
		for i, n := range w.NavFold {
			f := fmt.Sprintf("web.navFold[%d]", i)
			if !kindName.MatchString(n.LibraryKind) {
				add(f+".libraryKind", "%q isn't a kind of library", n.LibraryKind)
			}
			if n.Into != m.ID && !strings.HasPrefix(n.Into, m.ID+"-") {
				add(f+".into", "must be one of this module's menu entries (%s or %s-…)", m.ID, m.ID)
			}
		}
	}

	if s := in.SDUI; s != nil {
		for i, op := range s.Ops {
			f := fmt.Sprintf("sdui.ops[%d]", i)
			if !strings.HasPrefix(op.ID, m.ID+".") {
				add(f+".id", "must start with %q", m.ID+".")
			}
			meth, path, ok := strings.Cut(op.HTTP, " ")
			if !ok || !slices.ContainsFunc(in.HTTP, func(r Route) bool { return r.Path == path && slices.Contains(r.Methods, meth) }) {
				add(f+".http", "%q isn't one of this module's routes", op.HTTP)
			}
		}
	}

	if d := in.Data; d != nil {
		if d.Store != "sqlite" {
			add("data.store", "must be sqlite")
		}
		if !m.Asks(PermissionData) {
			add("data", "needs the data permission")
		}
		if d.Backup != "" && !slices.Contains([]string{"settings", "saves", "library"}, d.Backup) {
			add("data.backup", "must be settings, saves or library")
		}
		if m.DataFormat == 0 {
			add("data", "module.json must say its dataFormat")
		}
	}
	if n := in.Network; n != nil {
		if len(n.Hosts) > 0 && !m.Asks(PermissionNetwork) {
			add("network", "lists hosts but doesn't ask for the network permission")
		}
		for i, h := range n.Hosts {
			f := fmt.Sprintf("network.hosts[%d]", i)
			if !hostName.MatchString(h.Host) {
				add(f+".host", "%q isn't a host name such as api.example.com or *.example.com", h.Host)
			}
			if h.Why == "" || len(h.Why) > 200 {
				add(f+".why", "say why, in at most 200 characters")
			}
		}
	}
	if l := in.Lifecycle; l != nil && l.Uninstall != "" && !slices.Contains([]string{"ask-keep-data", "keep-data", "remove-data"}, l.Uninstall) {
		add("lifecycle.uninstall", "must be ask-keep-data, keep-data or remove-data")
	}
	if (len(in.HTTP) > 0 || in.Lifecycle != nil || in.Data != nil) && m.Delivery != DeliveryHelper {
		add("http", "routes, data and lifecycle need a module with a service (delivery helper)")
	}
	return ps
}

// ValidateIntegration is Check as one error, or nil.
func (in *Integration) Validate(m *Manifest) error {
	ps := in.Check(m)
	if len(ps) == 0 {
		return nil
	}
	lines := make([]string, len(ps))
	for i, p := range ps {
		lines[i] = p.String()
	}
	return errors.New(strings.Join(lines, "\n"))
}

func keysOf(m map[string]bool) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	slices.Sort(ks)
	return ks
}

// ParseSize reads "256KB", "4MB".
func ParseSize(s string) (int64, error) {
	m := sizeExpr.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("%q isn't a size such as 256KB or 4MB", s)
	}
	n, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("%q isn't a size", s)
	}
	mult := map[string]int64{"B": 1, "KB": 1 << 10, "MB": 1 << 20, "GB": 1 << 30}[m[2]]
	if n > (8<<30)/mult {
		return 0, fmt.Errorf("%q is more than 8GB", s)
	}
	return n * mult, nil
}

// InRange reports whether version v (a host feature's number) is within a range such as
// ">=1 <2" or "5".
func InRange(v int, r string) bool {
	for _, part := range strings.Fields(r) {
		op := strings.TrimRight(part, "0123456789")
		n, err := strconv.Atoi(part[len(op):])
		if err != nil {
			return false
		}
		ok := map[string]bool{"": v == n, "=": v == n, ">=": v >= n, "<=": v <= n, ">": v > n, "<": v < n}[op]
		if !ok {
			return false
		}
	}
	return r != ""
}

// Summary is what a catalogue says of a version's integration.json, for the store to show.
type IntegrationSummary struct {
	Routes   int      `json:"routes,omitempty"`
	Screens  []string `json:"screens,omitempty"` // web routes
	Settings []string `json:"settings,omitempty"`
	Data     bool     `json:"data,omitempty"`
	Feature  string   `json:"feature,omitempty"`
}

// Summary sums the file up.
func (in *Integration) Summary() IntegrationSummary {
	s := IntegrationSummary{Routes: len(in.HTTP), Data: in.Data != nil, Feature: in.Feature}
	if in.Web != nil {
		for _, r := range in.Web.Routes {
			s.Screens = append(s.Screens, r.Path)
		}
		for _, t := range in.Web.Settings {
			s.Settings = append(s.Settings, t.Label)
		}
	}
	return s
}
