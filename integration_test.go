// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors

package wittymodule

import (
	"strings"
	"testing"
)

const goodIntegration = `{
  "format": "wittymodule-integration", "formatVersion": 1, "module": "wittyplay",
  "host": {"http": ">=1 <2", "web": ">=5"},
  "feature": "games",
  "http": [
    {"path": "/games/{id}/play", "methods": ["POST"], "auth": "user", "item": "id", "plays": true,
     "alias": ["/api/games/{id}/play"], "limits": {"body": "256KB", "time": "3m"}},
    {"path": "/play/{sid}/stream", "methods": ["GET", "POST"], "auth": "user", "upgrade": "websocket", "stream": true,
     "alias": ["/api/play/{sid}/stream"]},
    {"path": "/rooms", "methods": ["POST"], "auth": "user"},
    {"path": "/kasm/{sid}/{ticket}/{rest...}", "methods": ["GET"], "auth": "ticket", "csp": "sandboxed-page"}
  ],
  "web": {
    "routes": [{"path": "/wittyplay/*", "frame": "fullscreen"}, {"path": "/game/:id", "frame": "fullscreen"}],
    "takeover": [{"libraryKind": "games", "route": "/wittyplay/library/{library}"}],
    "cards": [{"mediaKind": "game", "art": true, "detail": true, "link": true}],
    "players": [{"mediaKind": "game"}],
    "settings": [{"id": "gaming", "label": "Games & emulators", "admin": true, "search": ["cores", "BIOS"]}],
    "setup": [{"id": "games", "title": "Games", "after": "details", "when": {"libraryKinds": ["games"]}, "page": "wittyplay-setup",
               "saves": ["titles"], "later": {"where": "Settings, Games", "to": "/settings/gaming"}}],
    "navFold": [{"libraryKind": "games", "into": "wittyplay-online"}]
  },
  "sdui": {"ops": [{"id": "wittyplay.room.create", "http": "POST /rooms"}]},
  "data": {"store": "sqlite", "import": "server-games-v1", "backup": "saves"},
  "lifecycle": {"install": true, "upgrade": true, "uninstall": "ask-keep-data"},
  "live": {"sessions": true},
  "network": {"hosts": [{"host": "retroachievements.org", "why": "Achievements for the games you play"}]}
}`

func playManifest() *Manifest {
	return &Manifest{ID: "wittyplay", Name: "WittyPlay", Version: "0.5.0", Delivery: DeliveryHelper, WebEntry: "index.js",
		API: 5, MinServer: "0.9.0-alpha", MaxServer: "0.9.x", Permissions: []string{PermissionData, PermissionNetwork}, DataFormat: 1, DataFrom: 1}
}

func TestIntegrationGood(t *testing.T) {
	in, err := ParseIntegration([]byte(goodIntegration))
	if err != nil {
		t.Fatal(err)
	}
	if err := in.Validate(playManifest()); err != nil {
		t.Fatalf("a good file has problems:\n%v", err)
	}
	s := in.Summary()
	if s.Routes != 4 || s.Aliases != 2 || !s.Data || len(s.Screens) != 2 || s.Feature != "games" {
		t.Fatalf("summary %+v", s)
	}
}

func TestIntegrationStrict(t *testing.T) {
	if _, err := ParseIntegration([]byte(`{"format":"wittymodule-integration","formatVersion":1,"module":"x","htttp":[]}`)); err == nil {
		t.Fatal("a misspelt field was taken")
	}
	if _, err := ParseIntegration([]byte(`{} {}`)); err == nil {
		t.Fatal("two objects were taken")
	}
	if _, err := ParseIntegration(make([]byte, MaxIntegration+1)); err == nil {
		t.Fatal("an oversized file was taken")
	}
}

func TestIntegrationProblems(t *testing.T) {
	cases := map[string]struct{ from, to, want string }{
		"other module":         {`"module": "wittyplay"`, `"module": "wittytuner"`, "module"},
		"bad path":             {`"/rooms"`, `"rooms"`, "http[2].path"},
		"bad method":           {`"methods": ["POST"], "auth": "user"}`, `"methods": ["BREW"], "auth": "user"}`, "http[2].methods"},
		"bad auth":             {`"methods": ["POST"], "auth": "user"}`, `"methods": ["POST"], "auth": "anyone"}`, "http[2].auth"},
		"item not in path":     {`"item": "id"`, `"item": "game"`, "http[0].item"},
		"alias outside api":    {`"/api/games/{id}/play"`, `"/games/{id}/play"`, "http[0].alias[0]"},
		"alias v1":             {`"/api/games/{id}/play"`, `"/api/v1/games/{id}/play"`, "http[0].alias[0]"},
		"alias params":         {`"/api/games/{id}/play"`, `"/api/games/{game}/play"`, "http[0].alias[0]"},
		"bad size":             {`"256KB"`, `"lots"`, "http[0].limits.body"},
		"bad time":             {`"3m"`, `"3h"`, "http[0].limits.time"},
		"ticket writes":        {`"methods": ["GET"], "auth": "ticket"`, `"methods": ["POST"], "auth": "ticket"`, "http[3].methods"},
		"csp on user route":    {`"auth": "ticket", "csp"`, `"auth": "user", "csp"`, "http[3].csp"},
		"bad frame":            {`"frame": "fullscreen"}, {"path": "/game`, `"frame": "window"}, {"path": "/game`, "web.routes[0].frame"},
		"takeover no lib":      {`"/wittyplay/library/{library}"`, `"/wittyplay/library"`, "web.takeover[0].route"},
		"fold into other":      {`"into": "wittyplay-online"`, `"into": "wittytv"`, "web.navFold[0].into"},
		"op of another":        {`"id": "wittyplay.room.create"`, `"id": "wittycouch.room.create"`, "sdui.ops[0].id"},
		"op not a route":       {`"POST /rooms"}`, `"DELETE /rooms"}`, "sdui.ops[0].http"},
		"bad host feature":     {`"web": ">=5"`, `"gpu": ">=5"`, "host.gpu"},
		"bad host range":       {`"web": ">=5"`, `"web": "five"`, "host.web"},
		"bad backup":           {`"backup": "saves"`, `"backup": "cloud"`, "data.backup"},
		"bad uninstall":        {`"ask-keep-data"`, `"forget"`, "lifecycle.uninstall"},
		"websocket not GET":    {`"methods": ["GET", "POST"], "auth": "user", "upgrade"`, `"methods": ["POST"], "auth": "user", "upgrade"`, "http[1].upgrade"},
		"setup before modules": {`"after": "details"`, `"before": "modules"`, "web.setup[0].before"},
		"setup unknown anchor": {`"after": "details"`, `"after": "look"`, "web.setup[0]"},
		"setup other page":     {`"page": "wittyplay-setup"`, `"page": "wittytuner-setup"`, "web.setup[0].page"},
		"bad host":             {`"host": "retroachievements.org"`, `"host": "http://x"`, "network.hosts[0].host"},
		"no why":               {`"why": "Achievements for the games you play"`, `"why": ""`, "network.hosts[0].why"},
		"hidden without launch": {`"search": ["cores", "BIOS"]}`, `"search": ["cores", "BIOS"], "hidden": true}`, "web.settings[0].hidden"},
		"duplicate route":      {`{"path": "/rooms", "methods": ["POST"], "auth": "user"}`, `{"path": "/rooms", "methods": ["POST"], "auth": "user"}, {"path": "/rooms", "methods": ["POST"], "auth": "user"}`, "http[3]"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if !strings.Contains(goodIntegration, c.from) {
				t.Fatalf("the good file has no %s", c.from)
			}
			in, err := ParseIntegration([]byte(strings.Replace(goodIntegration, c.from, c.to, 1)))
			if err != nil {
				t.Fatal(err)
			}
			ps := in.Check(playManifest())
			for _, p := range ps {
				if p.Field == c.want {
					return
				}
			}
			t.Fatalf("no problem with %s; got %v", c.want, ps)
		})
	}
}

func TestIntegrationNeedsManifest(t *testing.T) {
	in, _ := ParseIntegration([]byte(goodIntegration))
	m := playManifest()
	m.Permissions, m.DataFormat, m.WebEntry = nil, 0, ""
	fields := map[string]bool{}
	for _, p := range in.Check(m) {
		fields[p.Field] = true
	}
	for _, f := range []string{"data", "web"} {
		if !fields[f] {
			t.Errorf("no problem with %s", f)
		}
	}
	m = playManifest()
	m.Delivery, m.WebEntry, m.Entry = DeliveryWeb, "", "index.js"
	if ps := in.Check(m); !strings.Contains(psString(ps), "http:") {
		t.Errorf("a web module with routes passed: %v", ps)
	}
}

func psString(ps []Problem) string {
	var b strings.Builder
	for _, p := range ps {
		b.WriteString(p.String() + "\n")
	}
	return b.String()
}

func TestNeedsIntegration(t *testing.T) {
	m := playManifest()
	if !m.NeedsIntegration() {
		t.Error("API 5 helper needs integration.json")
	}
	m.API = 4
	if m.NeedsIntegration() {
		t.Error("API 4 doesn't")
	}
	m.API, m.Delivery = 5, DeliveryLibrary
	if m.NeedsIntegration() {
		t.Error("a library doesn't")
	}
}

func TestInRange(t *testing.T) {
	for r, want := range map[string][]bool{ // versions 1, 2, 5
		">=1 <2": {true, false, false},
		"5":      {false, false, true},
		">1":     {false, true, true},
		"<=2":    {true, true, false},
		"":       {false, false, false},
		"x":      {false, false, false},
	} {
		for i, v := range []int{1, 2, 5} {
			if InRange(v, r) != want[i] {
				t.Errorf("InRange(%d, %q) = %v", v, r, !want[i])
			}
		}
	}
}

func TestParseSize(t *testing.T) {
	for s, want := range map[string]int64{"256KB": 256 << 10, "4MB": 4 << 20, "1B": 1, "8GB": 8 << 30} {
		if n, err := ParseSize(s); err != nil || n != want {
			t.Errorf("%s: %d %v", s, n, err)
		}
	}
	for _, s := range []string{"9GB", "0KB", "4mb", "-1MB", ""} {
		if _, err := ParseSize(s); err == nil {
			t.Errorf("%q was taken", s)
		}
	}
}
