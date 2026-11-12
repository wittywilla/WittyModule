// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors

package sdui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// boardGames is the worked example of docs/SDUI-MODULES.md, written with the kit.
func boardGames() *Module {
	m := New("wittyboardgames", "Board games").Colours(map[string]string{"accent": "#2E7D32"})
	m.AddSource(SourceSpec{Type: "wittyboardgames.shelf", Label: "Board games on the shelf", Description: "The household's board games, from the module's own list.", Icon: "game",
		Params: []PropSpec{
			{Name: "players", Type: "int", Label: "Players", Help: "Only games for this many players; 0 for any.", Min: Int(0), Max: Int(12), Since: 2},
			{Name: "sort", Type: "enum", Label: "Order", Default: "name", Options: []EnumOption{{"name", "By name"}, {"played", "Last played first"}, {"shuffle", "Shuffled"}}, Since: 2},
			{Name: "limit", Type: "int", Label: "At most", Min: Int(1), Max: Int(100), Since: 2},
		},
		Yields: []string{"game"}, CardStyle: "square", Paged: true, SeeAll: "/page/wittyboardgames", Title: "Board games"})
	m.AddComponent(ModuleComponent{Type: "wittyboardgames.GameNight", Label: "Game night", Description: "A row of games picked for tonight's players, shuffled every time.", Icon: "game",
		Props: []PropSpec{{Name: "title", Type: "text", Label: "Title", Default: "Game night", Since: 2}, {Name: "players", Type: "int", Label: "Players", Default: 4, Min: Int(1), Max: Int(12), Since: 2}},
		Section: Sec("night", "Shelf", Props("title", "{{title}} for {{players}}", "cardSize", "l", "seeAll", "always"),
			Src("wittyboardgames.shelf", "players", "{{players}}", "sort", "shuffle", "limit", 12))})
	m.AddView(ModuleView{ID: "wittyboardgames.scoreboard", Label: "Scoreboard", Description: "The module's live scoreboard, drawn by its web bundle.", Platforms: []string{"web", "desktop"}})
	m.AddPage(Page{ID: "wittyboardgames", Kind: "custom", Title: "Board games", Icon: "game", Sections: []Section{
		Sec("night", "wittyboardgames.GameNight", Props("players", 4), nil),
		Sec("all", "Shelf", Props("title", "Every game", "cardStyle", "square", "seeAll", "never"), Src("wittyboardgames.shelf", "sort", "name")),
		NativeView("scores", "wittyboardgames.scoreboard", Notice("scores-note", "info", "The scoreboard is on the web app.")),
	}})
	m.AddInsert("home", "jump", Section{ID: "wittyboardgames-tonight", Component: "wittyboardgames.GameNight", When: &Condition{Weekday: []string{"fri", "sat"}}})
	m.AddPreset(ModulePreset{ID: "wittyboardgames-recent", Label: "Board games played lately", Section: Shelf("boardgames", "Played lately", Src("wittyboardgames.shelf", "sort", "played", "limit", 20))})
	m.AddNav(NavItem{ID: "wittyboardgames", Type: "page", Page: "wittyboardgames", Placement: map[string]string{"phone": "more", "tv": "more"}})
	m.AddSettings(SettingsPage{ID: "wittyboardgames", Title: "Board games", Icon: "game", Groups: []SettingsGroup{
		{Title: "The shelf", Rows: []SettingRow{
			{Key: "listUrl", Kind: "string", Label: "Where the list is", Help: "A web address of a BoardGameGeek collection, or empty for the list kept here.", MaxLength: 300},
			{Key: "apiKey", Kind: "secret", Label: "API key"},
			{Key: "household", Kind: "int", Label: "Players in the house", Default: 4, Min: Int(1), Max: Int(12)}}},
		{Title: "For you", Footer: "Each profile chooses its own.", Rows: []SettingRow{
			{Key: "ages", Kind: "enum", Scope: "profile", Label: "Games for", Default: "all", Options: []EnumOption{{"all", "Everyone"}, {"kids", "Children"}}}}},
	}})
	return m
}

func canon(t *testing.T, v any) any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out any
	_ = json.Unmarshal(b, &out)
	return out
}

// The kit writes exactly the file the server's own example is.
func TestTheKitWritesTheWorkedExample(t *testing.T) {
	m, err := ReadFile(filepath.Join("testdata", "module-boardgames.json"), "wittyboardgames")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(canon(t, m), canon(t, boardGames())) {
		a, _ := JSON(m)
		b, _ := JSON(boardGames())
		t.Fatalf("different:\nfile %s\nkit  %s", a, b)
	}
	path := filepath.Join(t.TempDir(), File)
	if err := WriteFile(boardGames(), path); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadFile(path, "wittyboardgames"); err != nil {
		t.Fatal(err)
	}
}

func TestCheckCatchesWhatTheServerWouldRefuse(t *testing.T) {
	cases := map[string]func(m *Module){
		"/module":                             func(m *Module) { m.Module = "Bad" },
		"/theme/scope":                        func(m *Module) { m.Theme.Scope = "wittytv" },
		"/sources/0/type":                     func(m *Module) { m.Sources[0].Type = "shelf" },
		"/components/0/type":                  func(m *Module) { m.Components[0].Type = "wittyother.X" },
		"/components/0/section":               func(m *Module) { m.Components[0].Section.Props["subtitle"] = "{{nope}}" },
		"/pages/0/id":                         func(m *Module) { m.Pages[0].ID = "games" },
		"/pages/0/kind":                       func(m *Module) { m.Pages[0].Kind = "home" },
		"/pages/0/sections/1/source/type":     func(m *Module) { m.Pages[0].Sections[1].Source["type"] = "wittyother.x" },
		"/pages/0/sections/1/component":       func(m *Module) { m.Pages[0].Sections[1].Component = "Shelve" },
		"/pages/0/sections/1/props/titel":     func(m *Module) { m.Pages[0].Sections[1].Props["titel"] = "x" },
		"/pages/0/sections/2/props/view":      func(m *Module) { m.Pages[0].Sections[2].Props["view"] = "wittyother.view" },
		"/navigation/0/page":                  func(m *Module) { m.Navigation[0].Page = "wittyboardgames-x" },
		"/settings/0/groups/0/rows/1/scope":   func(m *Module) { m.Settings[0].Groups[0].Rows[1].Scope = "profile" },
		"/settings/0/groups/0/rows/2/default": func(m *Module) { m.Settings[0].Groups[0].Rows[2].Default = 99 },
	}
	for path, change := range cases {
		m := boardGames()
		change(m)
		found := false
		for _, is := range Check(m, "") {
			found = found || strings.HasPrefix(is.Path, path)
		}
		if !found {
			t.Errorf("%s: not caught; got %v", path, Check(m, ""))
		}
	}
}

func TestParseIsStrict(t *testing.T) {
	if _, err := Parse([]byte(`{"format":"wittymodule-sdui","formatVersion":1,"module":"ab","catalog":2,"theme":{"scope":"ab"},"extra":1}`)); err == nil {
		t.Error("an unknown field was read")
	}
}

func TestAnswers(t *testing.T) {
	var heard map[string]any
	a := Answers{
		Items: func(r ItemsRequest) (ItemsAnswer, error) {
			return ItemsAnswer{Items: []Item{{ID: "g1", Kind: "game", Title: r.Source.Type() + " for " + r.Viewer.ProfileKind}}, Total: 1}, nil
		},
		Image:           func(name string) ([]byte, error) { return []byte("png " + name), nil },
		SettingsChanged: func(v map[string]any) { heard = v },
	}
	res, _, err, ok := a.Handle(MethodItems, json.RawMessage(`{"source":{"type":"wittyx.picks"},"limit":5,"viewer":{"profileKind":"kids"}}`))
	if !ok || err != nil || res.(ItemsAnswer).Items[0].Title != "wittyx.picks for kids" {
		t.Fatalf("items %v %v %v", res, err, ok)
	}
	_, blob, _, ok := a.Handle(MethodImage, json.RawMessage(`{"name":"g1.png"}`))
	if !ok || string(blob) != "png g1.png" {
		t.Fatalf("image %q", blob)
	}
	if _, _, _, ok := a.Handle(MethodSettingsChanged, json.RawMessage(`{"values":{"household":3}}`)); !ok || heard["household"] != float64(3) {
		t.Fatalf("settings %v", heard)
	}
	if _, _, _, ok := a.Handle("read", nil); ok {
		t.Error("handled another method")
	}
	t.Setenv(SettingsEnv, `{"household":5}`)
	if Settings()["household"] != float64(5) {
		t.Error("settings from the environment")
	}
	_ = os.Getenv
}
