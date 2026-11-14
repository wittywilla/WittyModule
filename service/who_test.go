// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors

package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wittywilla/wittymodule"
)

func TestWho(t *testing.T) {
	r := httptest.NewRequest("GET", "/x", nil)
	if Who(r).User != 0 {
		t.Fatal("nobody is somebody")
	}
	r.Header.Set(HeaderUser, "12")
	r.Header.Set(HeaderAdmin, "1")
	r.Header.Set(HeaderItem, EncodeItem(Item{ID: 5, Kind: "game", Facets: map[string]string{"console": "snes"}}))
	id := Who(r)
	if id.User != 12 || !id.Admin || id.Item == nil || id.Item.ID != 5 || id.Item.Facets["console"] != "snes" {
		t.Fatalf("%+v", id)
	}
	r.Header.Set(HeaderItem, "!!!")
	if Who(r).Item != nil {
		t.Fatal("a broken item was read")
	}
}

func TestPeopleAndAdmins(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {})
	for _, c := range []struct {
		h          http.Handler
		user, adm  string
		wantStatus int
	}{
		{People(ok), "", "", 401}, {People(ok), "3", "", 200},
		{Admins(ok), "3", "", 403}, {Admins(ok), "3", "1", 200}, {Admins(ok), "", "1", 403},
	} {
		r := httptest.NewRequest("GET", "/", nil)
		if c.user != "" {
			r.Header.Set(HeaderUser, c.user)
		}
		if c.adm != "" {
			r.Header.Set(HeaderAdmin, c.adm)
		}
		w := httptest.NewRecorder()
		c.h.ServeHTTP(w, r)
		if w.Code != c.wantStatus {
			t.Errorf("user %q admin %q: %d, want %d", c.user, c.adm, w.Code, c.wantStatus)
		}
	}
}

func TestHTTPRoutesAndLifecycle(t *testing.T) {
	var heard []string
	c := Config{
		HTTP: func(mux *http.ServeMux) {
			mux.Handle("POST /games/{id}/play", People(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("play " + r.PathValue("id")))
			})))
		},
		Lifecycle: func(_ context.Context, call string, a wittymodule.LifecycleArgs) error {
			heard = append(heard, call+" "+a.From+">"+a.To)
			if call == wittymodule.CallUninstall {
				return errors.New("not now")
			}
			return nil
		},
	}
	started, err := listen(c)
	if err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest("POST", started.URL+"/http/games/9/play", nil)
	req.Header.Set(TokenHeader, started.Token)
	req.Header.Set(HeaderUser, "2")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(b) != "play 9" {
		t.Fatalf("%d %q", resp.StatusCode, b)
	}
	req, _ = http.NewRequest("POST", started.URL+"/http/games/9/play", nil) // no token
	if resp, _ := http.DefaultClient.Do(req); resp.StatusCode != 403 {
		t.Fatalf("without the token: %d", resp.StatusCode)
	}

	calls := helperCalls(c, started)
	if _, _, err := calls(MethodStart, json.RawMessage(`{"host":"http://127.0.0.1:1","hostToken":"t","offers":{"host":1}}`)); err != nil {
		t.Fatal(err)
	}
	if h := Host(); h.Host != "http://127.0.0.1:1" || h.HostToken != "t" || h.Offers["host"] != 1 {
		t.Fatalf("host %+v", h)
	}
	// The server's check that the process is there carries nothing, and changes nothing.
	if _, _, err := calls(MethodStart, nil); err != nil || Host().HostToken != "t" {
		t.Fatalf("after a check: %+v %v", Host(), err)
	}
	if _, _, err := calls(wittymodule.CallUpgrade, json.RawMessage(`{"from":"0.4.2","to":"0.5.0"}`)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := calls(wittymodule.CallUninstall, nil); err == nil {
		t.Fatal("a refused uninstall passed")
	}
	if len(heard) != 2 || heard[0] != "lifecycle.upgrade 0.4.2>0.5.0" {
		t.Fatalf("%v", heard)
	}
}
