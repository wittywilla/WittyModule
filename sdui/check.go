// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors

// Package sdui is the kit a module describes its part of every app with: its pages,
// sections, components, data sources, views, menu entries, settings and theme scope, in the
// sdui.json that ships in each release (WittyMediaServer docs/SDUI-MODULES.md).
//
// A module built for module API 4 must carry one; the server checks it against its
// catalogue before the module is used and refuses a version without it. This package
// writes the file from Go, checks it the way the server will as far as that can be done
// without a server (Check), and answers the server's calls for the module's sources,
// pictures and settings (Answers).
package sdui

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// File is the name of a module's SDUI file, beside module.json.
const File = "sdui.json"

// Format and FormatVersion mark the file.
const (
	Format        = "wittymodule-sdui"
	FormatVersion = 1
)

// MaxBytes bounds the file.
const MaxBytes = 1 << 20

// CatalogRevision is the catalogue revision this kit's copy of the catalogue is.
const CatalogRevision = 2

// catalogJSON is a copy of the server's catalogue (GET /api/v1/sdui/catalog), revision 2.
//
//go:embed catalog.json
var catalogJSON []byte

// SchemaJSON is the JSON Schema of the file, as the server writes it.
//
//go:embed sdui-module.schema.json
var SchemaJSON []byte

type catalogue struct {
	Revision   int `json:"revision"`
	Components []struct {
		Type      string     `json:"type"`
		Authored  bool       `json:"authored"`
		Container bool       `json:"container"`
		Data      string     `json:"data"`
		Props     []PropSpec `json:"props"`
	} `json:"components"`
	CardStyles []struct {
		Style string `json:"style"`
	} `json:"cardStyles"`
	Sources []struct {
		Type   string     `json:"type"`
		Params []PropSpec `json:"params"`
	} `json:"sources"`
	PageKinds []struct {
		Kind string `json:"kind"`
	} `json:"pageKinds"`
	Icons        []string `json:"icons"`
	Platforms    []string `json:"platforms"`
	FormFactors  []string `json:"formFactors"`
	Placements   []EnumOption
	NativeViews  []EnumOption `json:"nativeViews"`
	MediaKinds   []string     `json:"mediaKinds"`
	LibraryKinds []string     `json:"libraryKinds"`
}

var cat = sync.OnceValue(func() *catalogue {
	var c catalogue
	if err := json.Unmarshal(catalogJSON, &c); err != nil {
		panic("sdui: the kit's catalogue is unreadable: " + err.Error())
	}
	return &c
})

// Issue is one thing wrong, where, and what to do about it.
type Issue struct {
	Path     string `json:"path"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Fix      string `json:"fix,omitempty"`
}

func (i Issue) String() string {
	s := i.Path + ": " + i.Message
	if i.Fix != "" {
		s += " (" + i.Fix + ")"
	}
	return s
}

// Errors are the errors among issues, as one error; nil when there are none.
func Errors(issues []Issue) error {
	var msgs []string
	for _, i := range issues {
		if i.Severity == "error" {
			msgs = append(msgs, i.String())
		}
	}
	if len(msgs) == 0 {
		return nil
	}
	return errors.New(strings.Join(msgs, "\n"))
}

var (
	moduleID    = regexp.MustCompile(`^[a-z][a-z0-9]{1,63}$`)
	idPattern   = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)
	camelName   = regexp.MustCompile(`^[a-z][A-Za-z0-9]{0,47}$`)
	pascalName  = regexp.MustCompile(`^[A-Z][A-Za-z0-9]{0,47}$`)
	settingKey  = regexp.MustCompile(`^[a-z][A-Za-z0-9_.-]{0,63}$`)
	placeholder = regexp.MustCompile(`\{\{([A-Za-z][A-Za-z0-9_]{0,47})\}\}`)
	colour      = regexp.MustCompile(`^(#[0-9A-Fa-f]{3}|#[0-9A-Fa-f]{6}|#[0-9A-Fa-f]{8}|rgba?\([0-9., %]+\))$`)
)

// ModulePageKinds are the kinds of page a module may add.
var ModulePageKinds = []string{"custom", "library", "detail", "livetv", "music", "photos", "books", "games"}

// ItemKinds are the kinds of item a source may give.
var ItemKinds = []string{"movie", "episode", "show", "season", "video", "disc", "album", "artist", "track", "book", "audiobook", "podcast", "game", "photo", "channel", "recording", "collection", "library", "genre", "view"}

// SettingKinds and the scopes of a setting.
var (
	SettingKinds  = []string{"bool", "enum", "int", "string", "secret", "info"}
	SettingScopes = []string{"", "server", "profile"}
)

func owned(mod, id string) bool { return id == mod || strings.HasPrefix(id, mod+"-") }

func has(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

type checker struct {
	issues []Issue
	mod    string
	own    struct{ components, sources, views, pages map[string]bool }
}

func (c *checker) err(path, fix, format string, a ...any) {
	c.issues = append(c.issues, Issue{Path: path, Severity: "error", Message: fmt.Sprintf(format, a...), Fix: fix})
}

// Check checks a module's SDUI the way the server will, as far as that can be done
// without a server: the format, the names, that everything is the catalogue's or the
// module's own, and the settings. The server checks every property's value besides, and
// refuses a version it finds wrong; `wittymodule check` and the golden test run this.
func Check(m *Module, module string) []Issue {
	c := &checker{mod: m.Module}
	c.own.components, c.own.sources, c.own.views, c.own.pages = map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	if m.Format != Format {
		c.err("/format", `Write "format": "`+Format+`".`, "%q is not the format of a module's SDUI file", m.Format)
	}
	if m.FormatVersion != FormatVersion {
		c.err("/formatVersion", "Write \"formatVersion\": 1.", "format version %d is not one this kit knows", m.FormatVersion)
	}
	if !moduleID.MatchString(m.Module) {
		c.err("/module", "Use the module's ID from module.json.", "%q is not a module ID", m.Module)
	} else if module != "" && m.Module != module {
		c.err("/module", "Use the module's ID from module.json.", "the file is for %q and the module is %q", m.Module, module)
	}
	if m.Catalog < 1 || m.Catalog > CatalogRevision {
		c.err("/catalog", fmt.Sprintf("Write 1 to %d.", CatalogRevision), "catalog %d is not a revision this kit knows", m.Catalog)
	}
	if m.Theme.Scope != m.Module {
		c.err("/theme/scope", "Use the module's ID.", "a module's theme scope is its own ID")
	}
	for k, v := range m.Theme.Tokens {
		if !colour.MatchString(v) {
			c.err("/theme/tokens/"+k, "Use #rrggbb, #rrggbbaa or rgb()/rgba().", "%q is not a colour", v)
		}
	}
	for _, l := range []struct {
		p       string
		n, most int
	}{{"/pages", len(m.Pages), 32}, {"/sources", len(m.Sources), 64}, {"/components", len(m.Components), 64}, {"/navigation", len(m.Navigation), 16},
		{"/settings", len(m.Settings), 16}, {"/views", len(m.Views), 32}, {"/inserts", len(m.Inserts), 32}, {"/presets", len(m.Presets), 64}} {
		if l.n > l.most {
			c.err(l.p, "Leave some out.", "a module adds at most %d of these", l.most)
		}
	}
	for i, s := range m.Sources {
		p := fmt.Sprintf("/sources/%d", i)
		if !c.named(s.Type, camelName) {
			c.err(p+"/type", "Name it "+c.mod+".something, in camelCase.", "%q is not a source name of this module's", s.Type)
		}
		c.own.sources[s.Type] = true
		if strings.TrimSpace(s.Label) == "" {
			c.err(p+"/label", "Give it a label.", "a source needs a label")
		}
		if len(s.Yields) == 0 {
			c.err(p+"/yields", `List the kinds of item it gives, such as ["game"].`, "a source says what it gives")
		}
		for j, k := range s.Yields {
			if !has(ItemKinds, k) {
				c.err(fmt.Sprintf("%s/yields/%d", p, j), "", "%q is not a kind of item", k)
			}
		}
		if !c.cardStyle(s.CardStyle) {
			c.err(p+"/cardStyle", "", "%q is not a card style", s.CardStyle)
		}
	}
	for i, comp := range m.Components {
		if !c.named(comp.Type, pascalName) {
			c.err(fmt.Sprintf("/components/%d/type", i), "Name it "+c.mod+".Something, in PascalCase.", "%q is not a component name of this module's", comp.Type)
		}
		c.own.components[comp.Type] = true
	}
	for i, v := range m.Views {
		if !c.named(v.ID, camelName) {
			c.err(fmt.Sprintf("/views/%d/id", i), "Name it "+c.mod+".something.", "%q is not a view name of this module's", v.ID)
		}
		c.own.views[v.ID] = true
	}
	for _, p := range m.Pages {
		c.own.pages[p.ID] = true
	}
	for i, comp := range m.Components {
		p := fmt.Sprintf("/components/%d", i)
		if strings.TrimSpace(comp.Label) == "" {
			c.err(p+"/label", "Give it a label for the editor.", "a component needs a label")
		}
		if c.own.components[comp.Section.Component] {
			c.err(p+"/section/component", "Build it from the catalogue's components.", "a component is made of the catalogue's components")
		}
		props := map[string]bool{}
		for _, ps := range comp.Props {
			props[ps.Name] = true
		}
		for _, ph := range placeholders(comp.Section) {
			if !props[ph] {
				c.err(p+"/section", "Add a property called "+ph+".", "{{%s}} names no property of the component", ph)
			}
		}
		c.sections([]Section{comp.Section}, p+"/section", true)
	}
	for i, pg := range m.Pages {
		p := fmt.Sprintf("/pages/%d", i)
		if !owned(c.mod, pg.ID) || !idPattern.MatchString(pg.ID) {
			c.err(p+"/id", "Call it "+c.mod+" or "+c.mod+"-something.", "%q is not a page ID of this module's", pg.ID)
		}
		if !has(ModulePageKinds, pg.Kind) {
			c.err(p+"/kind", "Use one of "+strings.Join(ModulePageKinds, ", ")+".", "a module can't add a %s page", pg.Kind)
		}
		if (pg.Kind == "library" || pg.Kind == "detail") && (pg.Match == nil || (pg.Match.LibraryKind == "" && pg.Match.MediaKind == "")) {
			c.err(p+"/match", "Narrow it with match.libraryKind or match.mediaKind.", "a module's %s page is for a kind of library or title", pg.Kind)
		}
		if strings.TrimSpace(pg.Title) == "" {
			c.err(p+"/title", "Give it a title.", "a page needs a title")
		}
		if pg.Icon != "" && !has(cat().Icons, pg.Icon) {
			c.err(p+"/icon", "", "%q is not one of the built-in icons", pg.Icon)
		}
		c.sections(pg.Sections, p+"/sections", false)
		for j, v := range pg.Variants {
			c.sections(v.Sections, fmt.Sprintf("%s/variants/%d/sections", p, j), false)
		}
	}
	for i, in := range m.Inserts {
		p := fmt.Sprintf("/inserts/%d", i)
		if !owned(c.mod, in.Section.ID) {
			c.err(p+"/section/id", "Call it "+c.mod+"-something.", "%q is not a section ID of this module's", in.Section.ID)
		}
		if in.After != "" && in.Before != "" {
			c.err(p, "Keep one of after and before.", "an insert goes after a section or before one")
		}
		c.sections([]Section{in.Section}, p+"/section", false)
	}
	for i, ps := range m.Presets {
		p := fmt.Sprintf("/presets/%d", i)
		if !owned(c.mod, ps.ID) {
			c.err(p+"/id", "Call it "+c.mod+"-something.", "%q is not a preset ID of this module's", ps.ID)
		}
		c.sections([]Section{ps.Section}, p+"/section", false)
	}
	for i, n := range m.Navigation {
		p := fmt.Sprintf("/navigation/%d", i)
		if !owned(c.mod, n.ID) {
			c.err(p+"/id", "Call it "+c.mod+" or "+c.mod+"-something.", "%q is not an entry ID of this module's", n.ID)
		}
		switch n.Type {
		case "page":
			if !c.own.pages[n.Page] {
				c.err(p+"/page", "Name one of the module's pages.", "the module has no page %q", n.Page)
			}
		case "route":
			if !strings.HasPrefix(n.Route, "/") || strings.HasPrefix(n.Route, "//") || n.Label == "" {
				c.err(p+"/route", "Give a place in the app and a label.", "a route entry needs a place beginning with one / and a label")
			}
		case "native":
			if n.Screen == "settings" {
				c.err(p+"/screen", "Describe settings under settings.", "Settings is the server's entry")
			}
		default:
			c.err(p+"/type", "Use page, route or native.", "%q is not a kind of entry a module adds", n.Type)
		}
	}
	keys := map[string]bool{}
	for i, sp := range m.Settings {
		p := fmt.Sprintf("/settings/%d", i)
		if !owned(c.mod, sp.ID) {
			c.err(p+"/id", "Call it "+c.mod+" or "+c.mod+"-something.", "%q is not a settings page ID of this module's", sp.ID)
		}
		rows := 0
		for g, grp := range sp.Groups {
			for r, row := range grp.Rows {
				rows++
				c.row(row, fmt.Sprintf("%s/groups/%d/rows/%d", p, g, r), keys, sp.Admin)
			}
		}
		if rows == 0 {
			c.err(p+"/groups", "Add a row.", "a settings page needs at least one row")
		}
	}
	return c.issues
}

func (c *checker) named(name string, rest *regexp.Regexp) bool {
	mod, n, ok := strings.Cut(name, ".")
	return ok && mod == c.mod && rest.MatchString(n)
}

func (c *checker) cardStyle(s string) bool {
	for _, x := range cat().CardStyles {
		if x.Style == s {
			return true
		}
	}
	return false
}

// sections checks the components and sources sections use: the catalogue's, or this
// module's own; another module's never.
func (c *checker) sections(list []Section, path string, template bool) {
	for i, s := range list {
		p := fmt.Sprintf("%s/%d", path, i)
		if !idPattern.MatchString(s.ID) {
			c.err(p+"/id", "Use lower-case letters, digits, - and _.", "%q is not a section ID", s.ID)
		}
		switch {
		case c.own.components[s.Component]:
			if template {
				c.err(p+"/component", "Use the catalogue's components.", "a component's template can't use another of the module's components")
			}
		case strings.Contains(s.Component, "."):
			c.err(p+"/component", "Use the catalogue's, or this module's own.", "%q is another module's component: a module uses only its own", s.Component)
		default:
			found := false
			for _, comp := range cat().Components {
				if comp.Type == s.Component && comp.Authored {
					found = true
					if !template {
						for k := range s.Props {
							if !propNamed(comp.Props, k) {
								c.err(p+"/props/"+k, "", "%s has no property %q", s.Component, k)
							}
						}
					}
				}
			}
			if !found {
				c.err(p+"/component", "", "%q is not a component of the catalogue", s.Component)
			}
		}
		if s.Component == "NativeView" {
			if v, _ := s.Props["view"].(string); strings.Contains(v, ".") && !c.own.views[v] {
				c.err(p+"/props/view", "Use one of the module's own views.", "%q is not a view of this module's", v)
			}
		}
		if t := s.Source.Type(); t != "" {
			switch {
			case c.own.sources[t]:
			case strings.Contains(t, "."):
				c.err(p+"/source/type", "Use the catalogue's, or this module's own.", "%q is another module's source: a module uses only its own", t)
			default:
				found := false
				for _, src := range cat().Sources {
					found = found || src.Type == t
				}
				if !found {
					c.err(p+"/source/type", "", "%q is not a source of the catalogue", t)
				}
			}
		}
		c.sections(s.Children, p+"/children", template)
	}
}

func propNamed(specs []PropSpec, name string) bool {
	for _, s := range specs {
		if s.Name == name {
			return true
		}
	}
	return false
}

func (c *checker) row(r SettingRow, p string, keys map[string]bool, admin bool) {
	if !has(SettingKinds, r.Kind) {
		c.err(p+"/kind", "Use one of "+strings.Join(SettingKinds, ", ")+".", "%q is not a kind of setting", r.Kind)
		return
	}
	if strings.TrimSpace(r.Label) == "" {
		c.err(p+"/label", "Give it a label.", "a setting needs a label")
	}
	if r.Kind == "info" {
		return
	}
	if !settingKey.MatchString(r.Key) {
		c.err(p+"/key", "Use a short camelCase name.", "%q is not a setting's key", r.Key)
	} else if keys[r.Key] {
		c.err(p+"/key", "Give each its own key.", "%q is used twice", r.Key)
	}
	keys[r.Key] = true
	if !has(SettingScopes, r.Scope) {
		c.err(p+"/scope", "Use server or profile.", "%q is not where a setting is kept", r.Scope)
	}
	if r.Scope == "profile" && (admin || r.Kind == "secret") {
		c.err(p+"/scope", "", "a profile's own setting is on a page every profile sees, and is never a secret")
	}
	if (r.Kind == "enum") != (len(r.Options) > 0) {
		c.err(p+"/options", "", "an enum has options, and only an enum")
	}
	if r.Default != nil && r.Kind == "secret" {
		c.err(p+"/default", "Leave it out.", "a secret has no default")
	}
	if r.Default != nil && r.Kind == "int" {
		n, ok := r.Default.(float64)
		if i, isInt := r.Default.(int); isInt {
			n, ok = float64(i), true
		}
		if !ok || (r.Min != nil && n < float64(*r.Min)) || (r.Max != nil && n > float64(*r.Max)) {
			c.err(p+"/default", "", "the default is a whole number within min and max")
		}
	}
}

func placeholders(t Section) []string {
	seen := map[string]bool{}
	var walk func(v any)
	walk = func(v any) {
		switch x := v.(type) {
		case string:
			for _, m := range placeholder.FindAllStringSubmatch(x, -1) {
				seen[m[1]] = true
			}
		case map[string]any:
			for _, e := range x {
				walk(e)
			}
		case Source:
			for _, e := range x {
				walk(e)
			}
		case []any:
			for _, e := range x {
				walk(e)
			}
		}
	}
	var sec func(s Section)
	sec = func(s Section) {
		walk(s.Props)
		walk(map[string]any(s.Source))
		for _, ch := range s.Children {
			sec(ch)
		}
	}
	sec(t)
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Parse reads a module's SDUI file as strictly as the server does.
func Parse(b []byte) (*Module, error) {
	if len(b) > MaxBytes {
		return nil, fmt.Errorf("%s is %d bytes; the most is %d", File, len(b), MaxBytes)
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	var m Module
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("%s can't be read: %w", File, err)
	}
	if dec.More() {
		return nil, fmt.Errorf("%s has something after its end", File)
	}
	return &m, nil
}

// ReadFile reads and checks a module's SDUI file.
func ReadFile(path, module string) (*Module, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	m, err := Parse(b)
	if err != nil {
		return nil, err
	}
	return m, Errors(Check(m, module))
}

// JSON is the file's contents for a module: indented, with a newline at the end, the
// same bytes every time for the same module.
func JSON(m *Module) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(m); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// WriteFile checks a module's SDUI and writes it.
func WriteFile(m *Module, path string) error {
	if err := Errors(Check(m, m.Module)); err != nil {
		return err
	}
	b, err := JSON(m)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
