// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors

package wittymodule

// Scan hooks (module API 5): a module that owns a kind of library (WittyPlay: games) says so in
// integration.json's scan part. The server still walks the library's folders, keeps the items
// and checks who may see them; the module says what the files are (identify) and what's known
// about each item (describe), and keeps whatever else it learns in its own database.
//
// The calls go to the module's service (service.Config.Scan), each in its own request, so one
// can take a while and a scan that's stopped stops them (the request's context ends).
//
//	scan.choices   which values a library of the kind can be held to (consoles), for the
//	               library's settings and the apps
//	scan.identify  batches of the files the walk found; the last call (Done) answers the items
//	scan.describe  batches of new or changed items, by id: the fields the module knows now.
//	               What it finds out later it sets through the host API (PATCH /items/{id})

// ScanPart is integration.json's scan part.
type ScanPart struct {
	// LibraryKinds are the kinds of library the module scans ("games"). Each must be one of
	// module.json's libraries. A library of such a kind can't be made while the module is
	// missing.
	LibraryKinds []string `json:"libraryKinds"`
	// Identify: the module answers scan.identify (needed).
	Identify bool `json:"identify"`
	// Describe: the module answers scan.describe.
	Describe bool `json:"describe,omitempty"`
	// Facet is the facet a library of the kind can be held to ("console": a library of one
	// console's games), whose values scan.choices gives. "" for none.
	Facet string `json:"facet,omitempty"`
}

// Scan calls, to the module's service.
const (
	CallScanChoices  = "scan.choices"
	CallScanIdentify = "scan.identify"
	CallScanDescribe = "scan.describe"
)

// ScanAuto is the facet value of a library that holds any of the module's choices.
const ScanAuto = "auto"

// ScanChoicesArgs asks which values a kind of library can be held to.
type ScanChoicesArgs struct {
	LibraryKind string `json:"libraryKind"`
	// Testing: the server shows test choices (a test console) too.
	Testing bool `json:"testing,omitempty"`
}

// ScanChoice is one of them: a console.
type ScanChoice struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Short string `json:"short,omitempty"` // SNES, Mega Drive: for a card
}

// ScanChoices is scan.choices' answer.
type ScanChoices struct {
	Facet  string       `json:"facet"`
	Values []ScanChoice `json:"values"`
}

// ScanLibrary is the library a scan is of.
type ScanLibrary struct {
	ID    int64    `json:"id"`
	Kind  string   `json:"kind"`
	Facet string   `json:"facet,omitempty"` // the value it's held to, or ScanAuto
	Paths []string `json:"paths"`
}

// ScanFile is a file the walk found. Path is for reading only.
type ScanFile struct {
	Path  string `json:"path"`
	Root  string `json:"root"` // the library folder it's under
	Size  int64  `json:"size"`
	MTime int64  `json:"mtime"` // Unix seconds
}

// ScanIdentifyArgs is one call of scan.identify. A scan sends its files in batches, in the order
// the walk found them, and then a last call with Done and no files. A scan that's stopped sends
// no last call; the next scan has another Scan id.
type ScanIdentifyArgs struct {
	Scan    string      `json:"scan"` // the same for every call of one scan
	Library ScanLibrary `json:"library"`
	Files   []ScanFile  `json:"files,omitempty"`
	// Total is how many files the walk found in all, for the module's own progress.
	Total int  `json:"total,omitempty"`
	Done  bool `json:"done,omitempty"`
	// Offline are folders that couldn't be read this time: what was under them stays.
	Offline []string `json:"offline,omitempty"`
}

// ScanIdentified is scan.identify's answer: nothing until the last call, which lists every item
// the library holds now. An item the library had whose path isn't listed (and isn't under an
// offline folder) is missing from now on.
type ScanIdentified struct {
	Items []ScanItem `json:"items,omitempty"`
	// Problems is how many files the module couldn't make sense of, for the scan's result.
	Problems int `json:"problems,omitempty"`
	// Result is what the scan's task says when it's done ("Found 2,292 games in 41s."); "" for
	// the server's own words.
	Result string `json:"result,omitempty"`
}

// ScanItem is an item a scan found.
type ScanItem struct {
	// Files are the item's files, the first the one it's known by (its path): a game on several
	// discs is one item with a file for each.
	Files     []string          `json:"files"`
	Kind      string            `json:"kind"` // the item kind: game
	Title     string            `json:"title"`
	SortTitle string            `json:"sortTitle,omitempty"`
	Tagline   string            `json:"tagline,omitempty"` // region, revision
	Year      int               `json:"year,omitempty"`
	Container string            `json:"container,omitempty"` // the file's type: 7z, nes
	Size      int64             `json:"size,omitempty"`
	MTime     int64             `json:"mtime,omitempty"`
	Facets    map[string]string `json:"facets,omitempty"` // console
	// VersionOf is the path of the item this one is another version of (a region, a revision),
	// shown under it; "" when it stands on its own. VersionLabel says which version it is.
	VersionOf    string `json:"versionOf,omitempty"`
	VersionLabel string `json:"versionLabel,omitempty"`
}

// ScanDescribeArgs is one call of scan.describe: items a scan added or found changed, or one an
// administrator asked to be looked up again (Refresh).
type ScanDescribeArgs struct {
	Library ScanLibrary        `json:"library"`
	Items   []ScanDescribeItem `json:"items"`
	Refresh bool               `json:"refresh,omitempty"`
}

// ScanDescribeItem is an item to describe.
type ScanDescribeItem struct {
	ID     int64             `json:"id"`
	Kind   string            `json:"kind"`
	Title  string            `json:"title"`
	Files  []string          `json:"files"`
	Facets map[string]string `json:"facets,omitempty"`
	// Edited are the fields a person typed: the server keeps them whatever the module says.
	Edited []string `json:"edited,omitempty"`
}

// ScanDescribed is scan.describe's answer: what the module knows now, item by item. What it
// finds out later (from the internet) it sets with the host API's PATCH /items/{id}.
type ScanDescribed struct {
	Items []ItemFields `json:"items,omitempty"`
}

// ItemFields are an item's generic fields as a module sets them. A field left empty stays as it
// is; Clear names fields to empty. The server never changes a field a person typed.
type ItemFields struct {
	ID            int64       `json:"id"`
	Title         string      `json:"title,omitempty"`
	SortTitle     string      `json:"sortTitle,omitempty"`
	Tagline       string      `json:"tagline,omitempty"`
	Overview      string      `json:"overview,omitempty"`
	Year          int         `json:"year,omitempty"`
	Genres        []string    `json:"genres,omitempty"`
	Rating        *float64    `json:"rating,omitempty"` // 0-10
	ContentRating string      `json:"contentRating,omitempty"`
	Studio        string      `json:"studio,omitempty"`
	Poster        string      `json:"poster,omitempty"`   // an address PutImage gave
	Backdrop      string      `json:"backdrop,omitempty"` // likewise
	Provider      string      `json:"provider,omitempty"` // where it came from: "game:libretro"
	ProviderID    string      `json:"providerId,omitempty"`
	Titles        *ItemTitles `json:"titles,omitempty"`
	Clear         []string    `json:"clear,omitempty"`
}

// ItemTitles are an item's other names: in its own script, and in English.
type ItemTitles struct {
	Native  string `json:"native,omitempty"`
	English string `json:"english,omitempty"`
}

// ItemFieldNames are the fields ItemFields can set or clear, as Clear and Edited name them.
var ItemFieldNames = []string{"title", "tagline", "overview", "year", "genres", "rating", "contentRating", "studio", "poster", "backdrop", "provider", "titles"}
