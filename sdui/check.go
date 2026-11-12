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
	"slices"
	"strings"

	core "github.com/wittywilla/wittymodule/sdui/internal/core"
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
const CatalogRevision = 4

// CatalogJSON is a copy of the server's catalogue (GET /api/v1/sdui/catalog), revision 4.
//
//go:embed catalog.json
var CatalogJSON []byte

// SchemaJSON is the JSON Schema of the file, as the server writes it.
//
//go:embed sdui-module.schema.json
var SchemaJSON []byte

// ColorTokens are the colour tokens themes have, which a module's theme may set.
var ColorTokens = []string{"accent", "accentFg", "accentHover", "accentLine", "accentSoft", "accentText", "bg", "bg2", "bg3", "bg4", "border", "borderStrong",
	"brandAccent", "controlLine", "danger", "dangerSoft", "fg", "fg2", "fg3", "focus", "hover", "ok", "okSoft", "onImage", "overlay", "pressed", "scrim", "warn", "warnSoft"}

var colour = regexp.MustCompile(`^(#[0-9A-Fa-f]{3}|#[0-9A-Fa-f]{6}|#[0-9A-Fa-f]{8}|rgba?\(\s*\d{1,3}\s*,\s*\d{1,3}\s*,\s*\d{1,3}\s*(,\s*(0|1|0?\.\d+)\s*)?\))$`)

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

// Check checks a module's SDUI exactly as the server does before it uses a module: it is
// the server's own validator (internal/core, a copy of WittyMediaServer's pkg/sdui that
// kit/sync-sdui-core.sh keeps current), so a module that passes here is not refused at
// install for its SDUI. module is the module's ID ("" to take the file's). What the server
// adds from outside the file, such as that only a helper may declare sources, is checked by
// `wittymodule check`, which has the manifest.
func Check(m *Module, module string) []Issue {
	b, err := json.Marshal(m)
	if err != nil {
		return []Issue{{Path: "", Severity: "error", Message: err.Error()}}
	}
	ui, err := core.ParseModuleUI(b)
	if err != nil {
		return []Issue{{Path: "", Severity: "error", Message: err.Error()}}
	}
	if module == "" {
		module = ui.Module
	}
	checks := core.ModuleChecks{
		ColorToken: func(s string) bool { return slices.Contains(ColorTokens, s) },
		Color:      colour.MatchString,
	}
	var out []Issue
	for _, is := range core.ValidateModuleUI(ui, module, checks) {
		out = append(out, Issue{Path: is.Path, Severity: is.Severity, Message: is.Message, Fix: is.Fix})
	}
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
