// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors

// Package host is a module's way back to the server that runs it (module API 5): library items
// and their files, whether a person may do something, the module's settings and secrets,
// pictures, live sessions, notifications, tasks and events. Both ends speak the paths below; the server's end is
// WittyMediaServer internal/modules/hostapi.go.
//
// Every call that is on someone's behalf says so (On), and the server answers with that
// person's access, never more: a module can't see an item through the host that the person
// asking couldn't.
package host

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/wittywilla/wittymodule"
	"github.com/wittywilla/wittymodule/service"
)

// Version is the host API this package speaks.
const Version = 1

// Base is where the host API's paths start.
const Base = "/host/v1"

// Headers on a host call.
const (
	HeaderToken  = "X-Witty-Host-Token" // the secret the server gave this start
	HeaderOnUser = "X-Witty-On-User"    // the person a call is for
	HeaderOnProf = "X-Witty-On-Profile" // and their profile, if not their own
)

// ErrNoHost is returned when the server is older than module API 5 and has no host API.
var ErrNoHost = errors.New("this server has no host API (it is older than module API 5)")

// ErrNotOffered is returned for a call the server doesn't offer.
var ErrNotOffered = errors.New("this server doesn't offer that")

// Client calls the host API.
type Client struct {
	base   string
	token  string
	offers map[string]int
	http   *http.Client
}

// New is a client for the server that started this service, from what it said at the start.
func New() (*Client, error) {
	p := service.Host()
	if p.Host == "" {
		return nil, ErrNoHost
	}
	return NewAt(p.Host, p.HostToken, p.Offers), nil
}

// NewAt is a client for a host API at an address, for tests.
func NewAt(base, token string, offers map[string]int) *Client {
	return &Client{base: base, token: token, offers: offers, http: &http.Client{Timeout: 2 * time.Minute}}
}

// Has reports whether the server offers a host feature at a version at least v.
func (c *Client) Has(feature string, v int) bool { return c.offers[feature] >= v }

// On is who a call is for.
type On struct {
	User    int64
	Profile string
}

// For is the person a module route's request is from.
func For(r *http.Request) On {
	id := service.Who(r)
	return On{User: id.User, Profile: id.Profile}
}

// Error is what the server said when it refused a call.
type Error struct {
	Status  int
	Message string `json:"error"`
}

func (e *Error) Error() string { return fmt.Sprintf("host: %d %s", e.Status, e.Message) }

// IsNotFound and IsForbidden tell the usual refusals apart.
func IsNotFound(err error) bool  { var e *Error; return errors.As(err, &e) && e.Status == 404 }
func IsForbidden(err error) bool { var e *Error; return errors.As(err, &e) && e.Status == 403 }

// Do makes one call: a JSON body in (unless body is an io.Reader), JSON out into out.
func (c *Client) Do(ctx context.Context, on On, method, path string, body, out any) error {
	var rd io.Reader
	ctype := "application/json"
	switch b := body.(type) {
	case nil:
	case io.Reader:
		rd, ctype = b, "application/octet-stream"
	default:
		buf, err := json.Marshal(b)
		if err != nil {
			return err
		}
		rd = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+Base+path, rd)
	if err != nil {
		return err
	}
	req.Header.Set(HeaderToken, c.token)
	if rd != nil {
		req.Header.Set("Content-Type", ctype)
	}
	if on.User > 0 {
		req.Header.Set(HeaderOnUser, strconv.FormatInt(on.User, 10))
	}
	if on.Profile != "" {
		req.Header.Set(HeaderOnProf, on.Profile)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound && resp.Header.Get("X-Witty-Host") == "" {
		return ErrNotOffered // not a refusal of the host API's: no such call on this server
	}
	if resp.StatusCode >= 300 {
		e := &Error{Status: resp.StatusCode}
		_ = json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(e)
		if e.Message == "" {
			e.Message = resp.Status
		}
		return e
	}
	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// ---------------------------------------------------------------- the calls

// Hello says which module the server thinks this is, and what it offers.
type Hello struct {
	Module string         `json:"module"`
	Server string         `json:"server"`
	Offers map[string]int `json:"offers"`
	// Machine is what whoever runs the machine has allowed, beyond the server's own settings:
	// "gamingCustomCores" (the config file allows custom cores, and nothing locks them off) and
	// "gamingTestCore" (the built-in test console is offered: a developer's server).
	Machine map[string]bool `json:"machine,omitempty"`
}

// Hello asks.
func (c *Client) Hello(ctx context.Context) (Hello, error) {
	var h Hello
	return h, c.Do(ctx, On{}, "GET", "/hello", nil, &h)
}

// Item is one item, with its files, if the person may see it.
func (c *Client) Item(ctx context.Context, on On, id int64) (service.Item, error) {
	var it service.Item
	return it, c.Do(ctx, on, "GET", "/items/"+strconv.FormatInt(id, 10), nil, &it)
}

// Query picks items.
type Query struct {
	LibraryKind string            // games
	Library     int64             // one library
	Kind        string            // an item kind
	Facets      map[string]string // console=snes
	Search      string
	IDs         []int64 // only these items (at most 500)
	Sort        string  // added, rating, oldest or title; "" for by title
	Desc        bool    // the other way round
	MinRating   float64 // at least this score out of 10
	Genre       string
	Series      string // games of one franchise
	Offset      int
	Limit       int // at most 500
}

// values is the query's address.
func (q Query) values() url.Values {
	v := url.Values{}
	if q.LibraryKind != "" {
		v.Set("libraryKind", q.LibraryKind)
	}
	if q.Library > 0 {
		v.Set("library", strconv.FormatInt(q.Library, 10))
	}
	if q.Kind != "" {
		v.Set("kind", q.Kind)
	}
	for k, f := range q.Facets {
		v.Set("facet."+k, f)
	}
	if q.Search != "" {
		v.Set("search", q.Search)
	}
	if len(q.IDs) > 0 {
		ids := make([]string, len(q.IDs))
		for i, id := range q.IDs {
			ids[i] = strconv.FormatInt(id, 10)
		}
		v.Set("ids", strings.Join(ids, ","))
	}
	if q.Sort != "" {
		v.Set("sort", q.Sort)
	}
	if q.Desc {
		v.Set("desc", "1")
	}
	if q.MinRating > 0 {
		v.Set("minRating", strconv.FormatFloat(q.MinRating, 'f', -1, 64))
	}
	if q.Genre != "" {
		v.Set("genre", q.Genre)
	}
	if q.Series != "" {
		v.Set("series", q.Series)
	}
	if q.Offset > 0 {
		v.Set("offset", strconv.Itoa(q.Offset))
	}
	if q.Limit > 0 {
		v.Set("limit", strconv.Itoa(q.Limit))
	}
	return v
}

// Items lists the items a person may see that match, without their files.
func (c *Client) Items(ctx context.Context, on On, q Query) ([]service.Item, error) {
	var out struct {
		Items []service.Item `json:"items"`
	}
	return out.Items, c.Do(ctx, on, "GET", "/items?"+q.values().Encode(), nil, &out)
}

// ItemsFull lists the same items as the server's own brief item JSON, as its GET /api/v1/items
// gives each one (poster, rating, genres, progress …), for a module answering in that shape.
func (c *Client) ItemsFull(ctx context.Context, on On, q Query) ([]json.RawMessage, error) {
	v := q.values()
	v.Set("full", "1")
	var out struct {
		Items []json.RawMessage `json:"items"`
	}
	return out.Items, c.Do(ctx, on, "GET", "/items?"+v.Encode(), nil, &out)
}

// Facets is what one games library holds as a person may see it (consoles, genres, decades …),
// as the server's GET /api/v1/games/facets gives it.
func (c *Client) Facets(ctx context.Context, on On, library int64) (json.RawMessage, error) {
	var out json.RawMessage
	return out, c.Do(ctx, on, "GET", "/items/facets?library="+strconv.FormatInt(library, 10), nil, &out)
}

// Library is one of the server's libraries of a kind the module serves.
type Library struct {
	ID      int64    `json:"id"`
	Name    string   `json:"name"`
	Kind    string   `json:"kind"`              // games, …
	Console string   `json:"console,omitempty"` // a games library's console, or "auto"
	Paths   []string `json:"paths"`             // its folders, for reading only (a BIOS sweep)
}

// Libraries are the libraries of the kinds the module serves (all of them: this isn't on anyone's
// behalf, and the folders are for the module's own work, never shown to a person by it).
func (c *Client) Libraries(ctx context.Context) ([]Library, error) {
	var out struct {
		Libraries []Library `json:"libraries"`
	}
	return out.Libraries, c.Do(ctx, On{}, "GET", "/libraries", nil, &out)
}

// Counts counts the items a person may see in the module's kinds of library, by a facet
// (console): what a console page says it has.
func (c *Client) Counts(ctx context.Context, on On, libraryKind, facet string) (map[string]int, error) {
	var out struct {
		Counts map[string]int `json:"counts"`
	}
	return out.Counts, c.Do(ctx, on, "GET", "/items/counts?libraryKind="+url.QueryEscape(libraryKind)+"&by="+url.QueryEscape(facet), nil, &out)
}

// CountsIn is Counts in one of the module's libraries.
func (c *Client) CountsIn(ctx context.Context, on On, library int64, facet string) (map[string]int, error) {
	var out struct {
		Counts map[string]int `json:"counts"`
	}
	return out.Counts, c.Do(ctx, on, "GET", "/items/counts?library="+strconv.FormatInt(library, 10)+"&by="+url.QueryEscape(facet), nil, &out)
}

// May reports whether a person may use a feature (games, livetv …) now: their profile's
// limits, parental rules and screen time.
func (c *Client) May(ctx context.Context, on On, feature string) (bool, error) {
	var out struct {
		May bool `json:"may"`
	}
	err := c.Do(ctx, on, "GET", "/may?feature="+url.QueryEscape(feature), nil, &out)
	return out.May, err
}

// Settings are the module's settings (its sdui.json settings), for everyone, or a person's own.
func (c *Client) Settings(ctx context.Context, on On) (map[string]json.RawMessage, error) {
	var out map[string]json.RawMessage
	return out, c.Do(ctx, on, "GET", "/settings", nil, &out)
}

// Secret is a secret an administrator gave the module (a metadata service's key), or "".
func (c *Client) Secret(ctx context.Context, name string) (string, error) {
	var out struct {
		Value string `json:"value"`
	}
	err := c.Do(ctx, On{}, "GET", "/secrets/"+url.PathEscape(name), nil, &out)
	return out.Value, err
}

// PutImage keeps a picture in the server's image cache, for an item's role (cover, logo, …) or
// for the module itself (item 0), and says the address pages can show it at.
func (c *Client) PutImage(ctx context.Context, item int64, role string, img io.Reader) (string, error) {
	var out struct {
		URL string `json:"url"`
	}
	err := c.Do(ctx, On{}, "POST", "/images?item="+strconv.FormatInt(item, 10)+"&role="+url.QueryEscape(role), img, &out)
	return out.URL, err
}

// Session is something live: a game being played, a stream being watched.
type Session struct {
	ID    string `json:"id,omitempty"` // the server's
	Kind  string `json:"kind"`         // game, stream
	User  int64  `json:"user"`
	Item  int64  `json:"item,omitempty"`
	Title string `json:"title,omitempty"`
	// SignIn is the request's X-Witty-SignIn (service.Identity.SignIn): the sign-in that started
	// it, so that signing out there ends it. Opaque; empty for none.
	SignIn string `json:"signIn,omitempty"`
}

// EventSessionStop is the event the server sends when it ends a session a module started: its
// person signed out or was deleted, their screen time ran out, or games were turned off for them.
// Data is SessionStop. The module stops it (the server has already let it go).
const EventSessionStop = "session.stop"

// SessionStop says which session the server ended, and why: signed-out, time-up, feature-off, gone.
type SessionStop struct {
	ID     string `json:"id"`
	User   int64  `json:"user"`
	Reason string `json:"reason"`
}

// User is an account on the server, as a module may know it: for showing names, and for a
// module's own lists of people (friends, who's online).
type User struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName,omitempty"`
	Kind        string `json:"kind"` // adult, kids, guest
	Admin       bool   `json:"admin,omitempty"`
	LastSeen    int64  `json:"lastSeen,omitempty"` // Unix seconds a sign-in of theirs was last used; 0 for none
}

// Users is every account on the server.
func (c *Client) Users(ctx context.Context) ([]User, error) {
	var out struct {
		Users []User `json:"users"`
	}
	return out.Users, c.Do(ctx, On{}, "GET", "/users", nil, &out)
}

// Tools is the server's FFmpeg, and which encoders it found working.
type Tools struct {
	FFmpeg   string          `json:"ffmpeg"`
	FFprobe  string          `json:"ffprobe"`
	Version  string          `json:"version,omitempty"`
	Encoders map[string]bool `json:"encoders"`
	// Hardware is the hardware encoders' test, once it has finished (HardwareDone).
	Hardware     []json.RawMessage `json:"hardware,omitempty"`
	HardwareDone bool              `json:"hardwareDone"`
}

// Tools asks for the server's FFmpeg.
func (c *Client) Tools(ctx context.Context) (Tools, error) {
	var t Tools
	return t, c.Do(ctx, On{}, "GET", "/tools", nil, &t)
}

// StartSession tells the server something live started; it counts it for metrics, screen time
// and "playing now", and may end it (EndSession is then called by the server through the
// module's lifecycle events).
func (c *Client) StartSession(ctx context.Context, s Session) (string, error) {
	var out struct {
		ID string `json:"id"`
	}
	err := c.Do(ctx, On{User: s.User}, "POST", "/sessions", s, &out)
	return out.ID, err
}

// EndSession says it ended.
func (c *Client) EndSession(ctx context.Context, id string) error {
	return c.Do(ctx, On{}, "DELETE", "/sessions/"+url.PathEscape(id), nil, nil)
}

// Notify tells a person something, with a place in the app to go to.
func (c *Client) Notify(ctx context.Context, user int64, text, link string) error {
	return c.Do(ctx, On{User: user}, "POST", "/notify", map[string]string{"text": text, "link": link}, nil)
}

// Event is something that happened on the server that a module may care about.
type Event struct {
	Seq  int64           `json:"seq"`
	Type string          `json:"type"` // scan.finished, item.added, item.removed, user.deleted, profile.changed, session.stop, server.stopping
	Data json.RawMessage `json:"data,omitempty"`
}

// Events waits up to wait for events after seq, and returns them with the next seq to ask from.
func (c *Client) Events(ctx context.Context, after int64, wait time.Duration) ([]Event, int64, error) {
	var out struct {
		Events []Event `json:"events"`
		Next   int64   `json:"next"`
	}
	err := c.Do(ctx, On{}, "GET", fmt.Sprintf("/events?after=%d&wait=%s", after, wait), nil, &out)
	return out.Events, out.Next, err
}

// Task reports a long piece of work, shown with the server's tasks.
func (c *Client) Task(ctx context.Context, name string, done, total int, err error) error {
	body := map[string]any{"done": done, "total": total}
	if err != nil {
		body["error"] = err.Error()
	}
	return c.Do(ctx, On{}, "POST", "/tasks/"+url.PathEscape(name), body, nil)
}

// Needs is a module's host range check: whether this server offers every host feature the
// integration asks for, in range.
func Needs(in *wittymodule.Integration, offers map[string]int) error {
	for k, r := range in.Host {
		if !wittymodule.InRange(offers[k], r) {
			return fmt.Errorf("it needs %s %s from the server, which offers %d", k, r, offers[k])
		}
	}
	return nil
}

// Audit writes a row of the server's audit log: who (on), what, to what. The server keeps it, so it
// stays whatever happens to the module.
func (c *Client) Audit(ctx context.Context, on On, action, core, detail string) error {
	return c.Do(ctx, on, "POST", "/audit", map[string]string{"action": action, "core": core, "detail": detail}, nil)
}

// AuditRow is one row of the server's audit log.
type AuditRow struct {
	At     int64  `json:"at"` // Unix seconds
	User   string `json:"user"`
	Action string `json:"action"`
	Core   string `json:"core,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// AuditLog is the module's own rows of the audit log, newest first: at most limit, up to 500.
func (c *Client) AuditLog(ctx context.Context, limit int) ([]AuditRow, error) {
	var out struct {
		Rows []AuditRow `json:"rows"`
	}
	return out.Rows, c.Do(ctx, On{}, "GET", "/audit?limit="+strconv.Itoa(limit), nil, &out)
}

// Report is a module's word on its own version in use: Working after something that proves it
// (a game started), or a failure the server can't see itself. Three failures in a day take the
// version out of use.
type Report struct {
	Working bool   `json:"working"`
	Reason  string `json:"reason,omitempty"`
}

// Report tells the server.
func (c *Client) Report(ctx context.Context, r Report) error {
	return c.Do(ctx, On{}, "POST", "/report", r, nil)
}
