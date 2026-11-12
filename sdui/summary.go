// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors

package sdui

import (
	"strings"

	"github.com/wittywilla/wittymodule"
)

// Summary sums up what a module's SDUI adds to the apps, as a catalogue lists it for a
// version (wittymodule.CatalogUI): its pages, its menu entries by label, whether it has
// settings, and how many sources and inserts it has. A store shows it before anything is
// installed. It is cut to the catalogue's limits, which a file that passes Check is within.
func Summary(m *Module) wittymodule.CatalogUI {
	var ui wittymodule.CatalogUI
	titles := map[string]string{}
	for _, p := range m.Pages {
		title := cut(p.Title, wittymodule.MaxUIPageTitle)
		if title == "" {
			title = p.ID
		}
		titles[p.ID] = title
		if len(ui.Pages) < wittymodule.MaxUIPages {
			ui.Pages = append(ui.Pages, wittymodule.CatalogUIPage{ID: p.ID, Title: title})
		}
	}
	for _, n := range m.Navigation {
		if len(ui.Navigation) == wittymodule.MaxUINavigation {
			break
		}
		// A page entry left without a label takes its page's title, as the menu does.
		label := cut(n.Label, wittymodule.MaxUILabel)
		if label == "" {
			label = cut(titles[n.Page], wittymodule.MaxUILabel)
		}
		if label == "" {
			label = cut(n.ID, wittymodule.MaxUILabel)
		}
		ui.Navigation = append(ui.Navigation, label)
	}
	ui.Settings = len(m.Settings) > 0
	ui.Sources = min(len(m.Sources), wittymodule.MaxUISources)
	ui.Inserts = min(len(m.Inserts), wittymodule.MaxUIInserts)
	return ui
}

// cut keeps one line of at most n characters.
func cut(s string, n int) string {
	s = strings.Join(strings.FieldsFunc(s, func(r rune) bool { return r < ' ' || r == 0x7f }), " ")
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > n {
		s = strings.TrimSpace(string(r[:n]))
	}
	return s
}
