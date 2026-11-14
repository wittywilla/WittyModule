// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors

package host

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wittywilla/wittymodule"
	"github.com/wittywilla/wittymodule/service"
)

func TestClientSendsTokenAndPerson(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Witty-Host", "1")
		if r.Header.Get(HeaderToken) != "secret" {
			w.WriteHeader(403)
			_, _ = w.Write([]byte(`{"error":"no"}`))
			return
		}
		switch r.URL.Path {
		case Base + "/items/7":
			if r.Header.Get(HeaderOnUser) != "3" {
				w.WriteHeader(403)
				_, _ = w.Write([]byte(`{"error":"not theirs"}`))
				return
			}
			_ = json.NewEncoder(w).Encode(service.Item{ID: 7, Kind: "game", Files: []service.File{{Path: "/x.sfc", Size: 1}}})
		case Base + "/may":
			_, _ = w.Write([]byte(`{"may": true}`))
		default:
			w.WriteHeader(404)
			_, _ = w.Write([]byte(`{"error":"no such item"}`))
		}
	}))
	defer srv.Close()
	c := NewAt(srv.URL, "secret", map[string]int{wittymodule.HostAPI: 1})
	ctx := context.Background()
	it, err := c.Item(ctx, On{User: 3}, 7)
	if err != nil || it.ID != 7 || len(it.Files) != 1 {
		t.Fatalf("%+v %v", it, err)
	}
	if _, err := c.Item(ctx, On{User: 4}, 7); !IsForbidden(err) {
		t.Fatalf("someone else's item: %v", err)
	}
	if _, err := c.Item(ctx, On{User: 3}, 8); !IsNotFound(err) {
		t.Fatalf("a missing item: %v", err)
	}
	if ok, err := c.May(ctx, On{User: 3}, "games"); !ok || err != nil {
		t.Fatalf("may: %v %v", ok, err)
	}
	if !c.Has(wittymodule.HostAPI, 1) || c.Has(wittymodule.HostAPI, 2) {
		t.Fatal("Has")
	}
}

func TestNotOffered(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	if _, err := NewAt(srv.URL, "s", nil).Hello(context.Background()); err != ErrNotOffered {
		t.Fatalf("%v", err)
	}
}

func TestNeeds(t *testing.T) {
	in := &wittymodule.Integration{Host: map[string]string{"http": ">=1 <2", "web": ">=5"}}
	if err := Needs(in, map[string]int{"http": 1, "web": 5}); err != nil {
		t.Fatal(err)
	}
	if err := Needs(in, map[string]int{"http": 2, "web": 5}); err == nil {
		t.Fatal("http 2 passed")
	}
	if err := Needs(in, map[string]int{"http": 1}); err == nil {
		t.Fatal("no web passed")
	}
}
