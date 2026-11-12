// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors

package sdui

import core "github.com/wittywilla/wittymodule/sdui/internal/core"

// What a module's page puts into the shell while it shows (catalogue revision 8): the status bar's
// content and the round system dock's destinations. The experience decides where and how they're
// drawn. These are the core's own types, so a module builds exactly what the server checks.
type (
	// PageRegions is a page's contribution to the shell's regions.
	PageRegions = core.PageRegions
	// StatusItem is one thing in the status bar: kind profile, server or text.
	StatusItem = core.StatusItem
	// DockEntry is one destination of the system dock: a route, shown as a layer when As is "layer".
	DockEntry = core.DockEntry
)

// The status bar's kinds of item, and the places a settings page may be put into.
var (
	StatusKinds   = core.StatusKinds
	SettingsSlots = core.SettingsSlots
)
