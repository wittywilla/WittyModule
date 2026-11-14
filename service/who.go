// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors

package service

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
)

// Headers a server tells a module's routes who is asking by. Only the server sets them: it drops
// every X-Witty- header a browser sends before it sets its own, and only the server holds the
// token that reaches a service at all.
const (
	HeaderUser    = "X-Witty-User"    // the signed-in account's id
	HeaderAdmin   = "X-Witty-Admin"   // "1" for an administrator
	HeaderProfile = "X-Witty-Profile" // the profile in use, when it isn't the account's own
	// HeaderItem is the item a route's "item" parameter names, already checked for this
	// profile: base64url JSON of Item.
	HeaderItem = "X-Witty-Item"
	// HeaderFeatures is the features this profile may use, space-separated; set on every
	// request to a module route.
	HeaderFeatures = "X-Witty-Features"
	// HeaderKind is the profile's kind: adult, kids or guest (a guest can only look).
	HeaderKind = "X-Witty-Kind"
)

// Identity is who a request to a module route is from.
type Identity struct {
	User    int64
	Admin   bool
	Profile string
	Kind    string // adult, kids, guest
	// Item is the item its route names, when it names one.
	Item *Item
}

// Item is a library item as a server describes it to a module that was asked about it.
type Item struct {
	ID      int64             `json:"id"`
	Library int64             `json:"library"`
	Kind    string            `json:"kind"` // the item kind: game, movie, …
	Title   string            `json:"title"`
	Facets  map[string]string `json:"facets,omitempty"` // console, …
	Files   []File            `json:"files,omitempty"`
}

// File is one of an item's files. Path is for reading only: the module never writes there.
type File struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// Who reads who is asking. A request with no user wasn't a signed-in person's (a ticket route,
// or something else on this machine), and a route for people refuses it.
func Who(r *http.Request) Identity {
	id := Identity{Admin: r.Header.Get(HeaderAdmin) == "1", Profile: r.Header.Get(HeaderProfile), Kind: r.Header.Get(HeaderKind)}
	id.User, _ = strconv.ParseInt(r.Header.Get(HeaderUser), 10, 64)
	if h := r.Header.Get(HeaderItem); h != "" {
		if b, err := base64.RawURLEncoding.DecodeString(h); err == nil {
			var it Item
			if json.Unmarshal(b, &it) == nil {
				id.Item = &it
			}
		}
	}
	return id
}

// EncodeItem is HeaderItem's value for an item: the server's half of Who.
func EncodeItem(it Item) string {
	b, _ := json.Marshal(it)
	return base64.RawURLEncoding.EncodeToString(b)
}

// People wraps a route for signed-in people only: what the server checked, checked again.
func People(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if Who(r).User <= 0 {
			http.Error(w, `{"error":"sign in first"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Admins wraps a route for administrators only.
func Admins(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if id := Who(r); id.User <= 0 || !id.Admin {
			http.Error(w, `{"error":"only an administrator can do that"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
