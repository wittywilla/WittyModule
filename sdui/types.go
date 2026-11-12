// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors

package sdui

import "encoding/json"

// The types of a module's SDUI file, as the server reads it. They are copies of the
// server's own (WittyMediaServer pkg/sdui: model.go, catalog.go, module.go), so a module
// needs nothing of the server's to describe itself; the JSON is the contract, and
// docs/SDUI-MODULES.md in the server describes every field.
//
// Written by copying; a test in this package checks every field against the JSON
// Schema the server writes (sdui-module.schema.json, beside this file).

// Page is one page of the app as it is authored: an ordered list of sections,
// each a component from the catalogue, most of them filled from a data source.
// A page is rendered for one profile on one device into a Document.
type Page struct {
	// ID names the page: lower-case letters, digits, - and _. A page with the
	// ID of a built-in page replaces it.
	ID string `json:"id"`
	// Kind says what the page is for, which decides when the app opens it and
	// what its sources may refer to ($library on a library page, $item on a
	// detail page). See PageKinds.
	Kind string `json:"kind"`
	// Title is shown in the page's heading and the window title. It may hold
	// placeholders ({library}, {title}, {query}, {user}).
	Title string `json:"title"`
	// Icon is one of Icons, for a navigation entry made from the page.
	Icon string `json:"icon,omitempty"`
	// Description is for whoever edits the page; clients never see it.
	Description string `json:"description,omitempty"`
	// Match narrows a library or detail page to one library, one kind of
	// library or one kind of media. The most specific match wins.
	Match *Match `json:"match,omitempty"`
	// Scope is the theme scope the page belongs to: a module id (wittymusic)
	// or another name with colours under theme.modules. The whole page takes
	// that scope's colour overrides.
	Scope string `json:"scope,omitempty"`
	// Sections are the page, top to bottom.
	Sections []Section `json:"sections"`
	// Variants replace Sections for the first variant whose condition holds:
	// a kids home, a television home.
	Variants []Variant `json:"variants,omitempty"`
	// Personal says what a profile may change about the page for itself. nil:
	// everything in PersonalKinds.
	Personal *PersonalPolicy `json:"personal,omitempty"`
	// TTLSeconds suggests how long a client may show the page before asking
	// again. 0: the default for the page's kind.
	TTLSeconds int `json:"ttlSeconds,omitempty"`
	// Regions are what the page puts into the shell's own regions (the status bar, the system
	// dock) while it, or its theme scope, is showing.
	Regions *PageRegions `json:"regions,omitempty"`
}

// Match narrows a page. Zero fields match anything.
type Match struct {
	// Library is one library's ID.
	Library int64 `json:"library,omitempty"`
	// LibraryKind is a kind of library: movies, shows, anime, music, ...
	LibraryKind string `json:"libraryKind,omitempty"`
	// MediaKind is a kind of title on a detail page: movie, show, episode,
	// album, book, game, collection, ...
	MediaKind string `json:"mediaKind,omitempty"`
}

// Variant is another list of sections for the same page, used when its
// condition holds. Variants are tried in order; the first that holds wins.
type Variant struct {
	ID       string     `json:"id"`
	Label    string     `json:"label,omitempty"`
	When     *Condition `json:"when"`
	Sections []Section  `json:"sections"`
}

// PersonalPolicy lists what profiles may change: any of PersonalKinds. An
// empty list means nothing.
type PersonalPolicy struct {
	Allow []string `json:"allow"`
}

// Section is one component on a page.
type Section struct {
	// ID is unique on the page (across variants too): it is how a profile's
	// own order refers to the section and how a client asks for more of it.
	ID string `json:"id"`
	// Component is a type from the catalogue: Shelf, Hero, Grid, ...
	Component string `json:"component"`
	// Props are the component's properties, as the catalogue describes them.
	Props map[string]any `json:"props,omitempty"`
	// Source says where the component's items come from.
	Source Source `json:"source,omitempty"`
	// When must hold for the section to be shown.
	When *Condition `json:"when,omitempty"`
	// Repeat makes one copy of the section for each library, pinned view or
	// collection, with $each standing for it in Source and {each} in titles.
	Repeat *Repeat `json:"repeat,omitempty"`
	// Scope is a theme scope for this section alone (see Page.Scope).
	Scope string `json:"scope,omitempty"`
	// Locked keeps profiles from hiding or moving the section.
	Locked bool `json:"locked,omitempty"`
	// Children are the sections inside a layout component (Stack, Split, Tab).
	Children []Section `json:"children,omitempty"`
}

// Repeat makes a section once per thing.
type Repeat struct {
	// Over is "libraries", "pinnedViews" or "collections".
	Over string `json:"over"`
	// Kinds keeps only libraries of these kinds (Over "libraries").
	Kinds []string `json:"kinds,omitempty"`
	// Limit is the most copies made; 0 is the device's limit.
	Limit int `json:"limit,omitempty"`
}

// Source is a data source and its parameters, written flat:
// {"type": "recentlyAdded", "library": 3, "limit": 20}. The catalogue lists
// every type and its parameters. A string parameter beginning with $ stands
// for something the page was opened with ($library, $item, $show, $query) or
// for the thing a repeated section is repeated over ($each).
type Source map[string]any

// Condition says when a section, variant or navigation entry is shown. Every
// field that is set must hold; All, Any and Not combine conditions. The zero
// Condition always holds.
type Condition struct {
	All []Condition `json:"all,omitempty"`
	Any []Condition `json:"any,omitempty"`
	Not *Condition  `json:"not,omitempty"`
	// FormFactor: the device is one of these (tv, phone, tablet, desktop, web).
	FormFactor []string `json:"formFactor,omitempty"`
	// Platform: the app is one of these (web, tvos, ios, android, androidtv, desktop).
	Platform []string `json:"platform,omitempty"`
	// ProfileKind: the profile is one of these (adult, kids, guest).
	ProfileKind []string `json:"profileKind,omitempty"`
	// Admin: the profile is (true) or isn't (false) an administrator.
	Admin *bool `json:"admin,omitempty"`
	// Feature: the server has this feature (/capabilities features).
	Feature string `json:"feature,omitempty"`
	// Module: this module is installed and on.
	Module string `json:"module,omitempty"`
	// Can: the profile may use this (games, recordings, downloads, sharing,
	// comments, syncplay, remoteControl, remote, liveTv, editLayout): its
	// parental feature switches.
	Can string `json:"can,omitempty"`
	// HasLibrary: the profile can open at least one library of this kind.
	HasLibrary string `json:"hasLibrary,omitempty"`
	// LibraryKind: the page's own library is of this kind (library pages).
	LibraryKind string `json:"libraryKind,omitempty"`
	// Capability: the client said it has this (capabilities=hdr,touch).
	Capability string `json:"capability,omitempty"`
	// Flag: something the server knows is so: setupIncomplete, scanning,
	// librariesEmpty, hasQuery, screenTimeUp.
	Flag string `json:"flag,omitempty"`
	// MinWidth and MaxWidth bound the window's width in points, inclusive.
	// They pass when the client didn't say how wide it is.
	MinWidth int `json:"minWidth,omitempty"`
	MaxWidth int `json:"maxWidth,omitempty"`
	// TimeFrom and TimeTo bound the time of day on the server, "HH:MM",
	// from inclusive and to exclusive; a range may pass midnight.
	TimeFrom string `json:"timeFrom,omitempty"`
	TimeTo   string `json:"timeTo,omitempty"`
	// Weekday: today is one of these (mon, tue, wed, thu, fri, sat, sun).
	Weekday []string `json:"weekday,omitempty"`
	// Locale: the client's language is this or a kind of it (en matches en-GB).
	Locale string `json:"locale,omitempty"`
}

// NavItem is one entry of the main menu.
type NavItem struct {
	ID string `json:"id"`
	// Label and Icon may be left out for a page (its title and icon are used)
	// and for the libraries (each library's name and its kind's icon).
	Label string `json:"label,omitempty"`
	Icon  string `json:"icon,omitempty"`
	// Type is "page" (Page names it), "libraries" (one entry per library the
	// profile can open), "library" (Library is its ID), "native" (Screen is a
	// screen the app draws itself: settings, guide, profiles, downloads,
	// remote) or "route" (Route is a place in the app).
	Type    string `json:"type"`
	Page    string `json:"page,omitempty"`
	Library int64  `json:"library,omitempty"`
	Screen  string `json:"screen,omitempty"`
	Route   string `json:"route,omitempty"`
	// When must hold for the entry to be shown. The settings entry has none.
	When *Condition `json:"when,omitempty"`
	// Placement says where the entry sits on each form factor: "primary" (the
	// bar itself), "more" (behind More or in the sidebar's lower group), "end"
	// (pinned to the far end) or "hidden". A form factor left out is primary.
	Placement map[string]string `json:"placement,omitempty"`
}

// Item is one thing a data source found, in the one shape every card style
// reads. A source fills what it knows; a card style shows what it has room
// for (see CardStyles).
type Item struct {
	// ID is unique among the items of one source, and stable.
	ID string `json:"id"`
	// Kind is movie, episode, show, season, video, disc, album, artist, track,
	// book, audiobook, podcast, game, photo, channel, recording, collection,
	// library, genre or view.
	Kind     string `json:"kind"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle,omitempty"`
	Overview string `json:"overview,omitempty"`
	Tagline  string `json:"tagline,omitempty"`
	Year     int    `json:"year,omitempty"`
	// Rating is the community score out of 10; ContentRating the certificate.
	Rating        float64  `json:"rating,omitempty"`
	ContentRating string   `json:"contentRating,omitempty"`
	RuntimeSec    float64  `json:"runtimeSec,omitempty"`
	Genres        []string `json:"genres,omitempty"`
	Studio        string   `json:"studio,omitempty"`
	// Progress is how far through it the profile is, 0 to 1; Watched that it
	// finished; Unwatched how many episodes of a show are left.
	Progress  float64 `json:"progress,omitempty"`
	Watched   bool    `json:"watched,omitempty"`
	Unwatched int     `json:"unwatched,omitempty"`
	// InWatchLater: the title is on the profile's Watch later list.
	InWatchLater bool `json:"inWatchLater,omitempty"`
	// ResumeSec is where Play picks up, 0 to start over.
	ResumeSec float64 `json:"resumeSec,omitempty"`
	// Season and Episode number an episode; ShowTitle names its show.
	Season    int    `json:"season,omitempty"`
	Episode   int    `json:"episode,omitempty"`
	ShowTitle string `json:"showTitle,omitempty"`
	ShowID    string `json:"showId,omitempty"`
	// Artist and Track are a song's; Count is how many things an album,
	// artist, library, collection or season holds.
	Artist string `json:"artist,omitempty"`
	Track  int    `json:"track,omitempty"`
	Count  int    `json:"count,omitempty"`
	// Number is a channel's number. Now and Next are what is on it.
	Number string     `json:"number,omitempty"`
	Now    *Programme `json:"now,omitempty"`
	Next   *Programme `json:"next,omitempty"`
	// Images are addresses on this server by kind: poster (2/3), backdrop
	// (16/9), square (1/1), logo. Render adds the width to ask for.
	Images map[string]string `json:"images,omitempty"`
	// Route is the item's place in the app; PlayID what to play, when it can
	// be played from a card.
	Route  string `json:"route,omitempty"`
	PlayID string `json:"playId,omitempty"`
	// Badges are short labels: 4K, HDR, NEW, LIVE.
	Badges []string `json:"badges,omitempty"`
	// Icon is one of Icons, for items without pictures (libraries, genres).
	Icon string `json:"icon,omitempty"`
	// Scope is the theme scope the item belongs to (wittytv for a channel).
	Scope string `json:"scope,omitempty"`
	// Can lists what DetailHeader may offer for the item: play, download,
	// share, watchLater, watched, playOn, watchParty.
	Can []string `json:"can,omitempty"`
	// NextID is the item Play should start for a show: its next episode.
	NextID string `json:"nextId,omitempty"`
}

// Programme is something on a channel.
type Programme struct {
	Title string `json:"title"`
	// Start and End are Unix seconds.
	Start int64 `json:"start,omitempty"`
	End   int64 `json:"end,omitempty"`
}

// EnumOption is one choice of a property or facet.
type EnumOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// PropSpec describes one property of a component or parameter of a source,
// with everything an editor needs to offer it and a validator to check it.
type PropSpec struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Label string `json:"label"`
	Help  string `json:"help,omitempty"`
	// Default is what a renderer and the server assume when it is left out.
	Default  any  `json:"default,omitempty"`
	Required bool `json:"required,omitempty"`
	// Options are the choices of an enum or enumList.
	Options []EnumOption `json:"options,omitempty"`
	// Min and Max bound an int; MaxLength a string.
	Min       *int `json:"min,omitempty"`
	Max       *int `json:"max,omitempty"`
	MaxLength int  `json:"maxLength,omitempty"`
	// FormFactors lists where the property has an effect; empty means all.
	FormFactors []string `json:"formFactors,omitempty"`
	// Since is the catalogue revision the property appeared in.
	Since int `json:"since"`
}

// SourceSpec describes one data source.
type SourceSpec struct {
	Type        string `json:"type"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	// Category groups the editor's list: personal, library, discover, live,
	// music, context, structure.
	Category string     `json:"category"`
	Params   []PropSpec `json:"params,omitempty"`
	// Yields are the item kinds it gives; CardStyle the style "auto" means.
	Yields    []string `json:"yields"`
	CardStyle string   `json:"cardStyle"`
	// PageKinds lists the kinds of page it can be used on; empty means any.
	PageKinds []string `json:"pageKinds,omitempty"`
	// Paged sources give more through a section's cursor URL.
	Paged bool `json:"paged"`
	// SeeAll is the route "See all" opens, with {param} placeholders; "" when
	// the source has no page of its own.
	SeeAll string `json:"seeAll,omitempty"`
	// Title is a good default title for a section using it.
	Title   string          `json:"title,omitempty"`
	Since   int             `json:"since"`
	Example json.RawMessage `json:"example,omitempty"`
	// Module is the module that answers the source; "" for the server.
	Module string `json:"module,omitempty"`
	// TopShelf offers the source for tvOS Top Shelf, as a row of its own: only for a
	// source with no required parameters and no PageKinds.
	TopShelf bool `json:"topShelf,omitempty"`
}

// Module is one module's SDUI file.
type Module struct {
	// Schema may name the JSON Schema an editor checks the file with
	// (docs/sdui-module.schema.json); the server ignores it.
	Schema string `json:"$schema,omitempty"`
	// Format is ModuleFormat, FormatVersion ModuleFormatVersion.
	Format        string `json:"format"`
	FormatVersion int    `json:"formatVersion"`
	// Module is the module's ID, the same as its manifest's.
	Module string `json:"module"`
	// Catalog is the catalogue revision the module was written against. A
	// server refuses a file newer than its own catalogue.
	Catalog int `json:"catalog"`
	// Theme is the module's theme scope. Required.
	Theme ModuleTheme `json:"theme"`
	// Pages are whole pages the module adds: its own (kind custom, opened at
	// /page/{id}), or a page for a kind of library or title (a games library
	// page, the detail page of a game), or the page of a kind the module is
	// about (games, photos, books, music, livetv).
	Pages []Page `json:"pages,omitempty"`
	// Inserts put one of the module's sections into a built-in page, such as a
	// row on Home. An administrator's own page is never changed by a module.
	Inserts []ModuleInsert `json:"inserts,omitempty"`
	// Presets are sections offered in the page editor's gallery.
	Presets []ModulePreset `json:"presets,omitempty"`
	// Components are the module's named sections: a section of catalogue
	// components with properties of its own, offered in the editor's palette
	// and expanded by the server, so no app needs to know them.
	Components []ModuleComponent `json:"components,omitempty"`
	// Sources are the data sources the module answers, through its helper.
	Sources []ModuleSource `json:"sources,omitempty"`
	// Views are views of the module's own that an app may draw in a
	// NativeView (the module's web bundle, for one); every other app draws the
	// NativeView's sections instead.
	Views []ModuleView `json:"views,omitempty"`
	// Navigation entries the module adds to the main menu.
	Navigation []NavItem `json:"navigation,omitempty"`
	// Settings describe the module's settings, which every app's own Settings
	// draws in its own style under Settings, Modules. They are never SDUI pages.
	Settings []SettingsPage `json:"settings,omitempty"`
}

// ModuleTheme is a module's theme scope: its ID, a name to show in the theme
// editor, and the colours the module would like, which an administrator's own
// overrides for the scope replace token by token.
type ModuleTheme struct {
	Scope  string            `json:"scope"`
	Label  string            `json:"label,omitempty"`
	Tokens map[string]string `json:"tokens,omitempty"`
}

// ModuleInsert puts a section into a built-in page.
type ModuleInsert struct {
	// Page is the built-in page's ID: home, music, games, ...
	Page string `json:"page"`
	// After or Before name a section of that page; neither: at the end.
	After   string  `json:"after,omitempty"`
	Before  string  `json:"before,omitempty"`
	Section Section `json:"section"`
}

// ModulePreset is a section for the editor's gallery.
type ModulePreset struct {
	ID          string  `json:"id"`
	Label       string  `json:"label"`
	Description string  `json:"description,omitempty"`
	Section     Section `json:"section"`
}

// ModuleComponent is a named section with properties of its own. Its Section
// is a template: a value written "{{name}}" becomes the property's value, and
// "{{name}}" inside text is replaced by it.
type ModuleComponent struct {
	// Type is "<module>.<Name>": wittyplay.ConsoleShelf.
	Type        string     `json:"type"`
	Label       string     `json:"label"`
	Description string     `json:"description,omitempty"`
	Icon        string     `json:"icon,omitempty"`
	Props       []PropSpec `json:"props,omitempty"`
	// PageKinds are the kinds of page it may be used on; empty: any.
	PageKinds []string `json:"pageKinds,omitempty"`
	Section   Section  `json:"section"`
}

// ModuleSource is a data source a module answers: the catalogue's description
// of a source, with Type "<module>.<name>".
type ModuleSource struct {
	SourceSpec
}

// ModuleView is a view of the module's own for a NativeView.
type ModuleView struct {
	// ID is "<module>.<name>"; it is what a NativeView's view prop names and
	// what an app that can draw it sends in views=.
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	// Platforms that can draw it, as a hint for the editor; empty: unknown.
	Platforms []string `json:"platforms,omitempty"`
}

// SettingsPage is one page of a module's settings, as every app's own
// Settings draws it: a title, and groups of rows.
type SettingsPage struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Icon  string `json:"icon,omitempty"`
	// Admin pages are shown only to administrators.
	Admin bool `json:"admin,omitempty"`
	// Slot puts the page into a place of the hand-built Settings: "you.games" (You, Games) or
	// "people.profile" (Settings, People, a profile; administrators only, so the page must be
	// Admin). See SettingsSlots.
	Slot   string          `json:"slot,omitempty"`
	Groups []SettingsGroup `json:"groups"`
}

// SettingsGroup is a titled group of rows.
type SettingsGroup struct {
	Title  string       `json:"title,omitempty"`
	Footer string       `json:"footer,omitempty"`
	Rows   []SettingRow `json:"rows"`
}

// SettingRow is one setting.
type SettingRow struct {
	// Key names the value: lower-case first letter, then letters, digits, . _ -.
	Key   string `json:"key"`
	Kind  string `json:"kind"`
	Label string `json:"label"`
	Help  string `json:"help,omitempty"`
	// Scope is server (the default) or profile.
	Scope   string       `json:"scope,omitempty"`
	Default any          `json:"default,omitempty"`
	Options []EnumOption `json:"options,omitempty"`
	Min     *int         `json:"min,omitempty"`
	Max     *int         `json:"max,omitempty"`
	// Unit follows a number: "MB", "seconds".
	Unit      string `json:"unit,omitempty"`
	MaxLength int    `json:"maxLength,omitempty"`
}

// Type is the source's type, "" when the section has no source.
func (s Source) Type() string {
	t, _ := s["type"].(string)
	return t
}
