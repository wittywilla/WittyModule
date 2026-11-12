// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors

package sdui

// Helpers for writing a module's SDUI in Go. They only fill in the structs; anything
// they don't cover is written as the struct itself.

// New starts a module's SDUI: its ID, the catalogue revision it is written for, and its
// theme scope with a label for the theme editor.
func New(module, label string) *Module {
	return &Module{Format: Format, FormatVersion: FormatVersion, Module: module, Catalog: CatalogRevision, Theme: ModuleTheme{Scope: module, Label: label}}
}

// Colours sets the colours the module would like in its scope, by token (accent, ...).
func (m *Module) Colours(tokens map[string]string) *Module {
	m.Theme.Tokens = tokens
	return m
}

// AddPage, AddSource, AddComponent, AddView, AddInsert, AddPreset, AddNav and AddSettings
// add one of each, and answer the module for the next.
func (m *Module) AddPage(p Page) *Module { m.Pages = append(m.Pages, p); return m }
func (m *Module) AddSource(s SourceSpec) *Module {
	if s.Since == 0 {
		s.Since = CatalogRevision
	}
	m.Sources = append(m.Sources, ModuleSource{SourceSpec: s})
	return m
}
func (m *Module) AddComponent(c ModuleComponent) *Module {
	m.Components = append(m.Components, c)
	return m
}
func (m *Module) AddView(v ModuleView) *Module { m.Views = append(m.Views, v); return m }
func (m *Module) AddInsert(page, after string, s Section) *Module {
	m.Inserts = append(m.Inserts, ModuleInsert{Page: page, After: after, Section: s})
	return m
}
func (m *Module) AddPreset(p ModulePreset) *Module { m.Presets = append(m.Presets, p); return m }
func (m *Module) AddNav(n NavItem) *Module         { m.Navigation = append(m.Navigation, n); return m }
func (m *Module) AddSettings(p SettingsPage) *Module {
	m.Settings = append(m.Settings, p)
	return m
}

// Src is a data source: its type and parameters in pairs, Src("recentlyAdded", "limit", 20).
func Src(typ string, kv ...any) Source {
	s := Source{"type": typ}
	for i := 0; i+1 < len(kv); i += 2 {
		if k, ok := kv[i].(string); ok {
			s[k] = kv[i+1]
		}
	}
	return s
}

// Props are a section's properties in pairs.
func Props(kv ...any) map[string]any {
	if len(kv) == 0 {
		return nil
	}
	p := map[string]any{}
	for i := 0; i+1 < len(kv); i += 2 {
		if k, ok := kv[i].(string); ok {
			p[k] = kv[i+1]
		}
	}
	return p
}

// Sec is one section.
func Sec(id, component string, props map[string]any, src Source, children ...Section) Section {
	return Section{ID: id, Component: component, Props: props, Source: src, Children: children}
}

// Shelf is a row of cards from a source, with a title.
func Shelf(id, title string, src Source) Section {
	return Sec(id, "Shelf", Props("title", title), src)
}

// Grid is a grid of cards from a source; it belongs at the end of a page.
func Grid(id, title string, src Source) Section {
	return Sec(id, "Grid", Props("title", title), src)
}

// NativeView is a view of the module's own where an app has it, and children elsewhere.
func NativeView(id, view string, children ...Section) Section {
	return Sec(id, "NativeView", Props("view", view), nil, children...)
}

// Notice is a short message.
func Notice(id, tone, text string) Section {
	return Sec(id, "Notice", Props("tone", tone, "text", text), nil)
}

// Int is a pointer to n, for a property's or setting's Min and Max.
func Int(n int) *int { return &n }
