// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors

package service

import (
	"encoding/json"
	"io"
	"log/slog"
	"reflect"
	"testing"
	"time"
)

type wireInner struct {
	Name    string   `json:"name"`
	Aliases []string `json:"-"`
	Hidden  bool     `json:"-"`
}

type wireThing struct {
	ID       int64             `json:"id"`
	Path     string            `json:"-"`
	Inner    wireInner         `json:"inner"`
	List     []wireInner       `json:"list,omitempty"`
	Ptr      *wireInner        `json:"ptr,omitempty"`
	ByName   map[string]string `json:"byName,omitempty"`
	ByNumber map[int]wireInner `json:"-"`
	Blob     []byte            `json:"blob"`
	When     time.Time         `json:"when"`
	Took     time.Duration     `json:"took"`
	Any      any               `json:"any"`
	Fixed    [2]int            `json:"fixed"`
	Log      *slog.Logger      `json:"-"`
	Out      io.Writer         `json:"-"`
	Do       func() int        `json:"-"`
	secret   string
}

// What's hidden from JSON still crosses, and what's live doesn't.
func TestWireCarriesHiddenFields(t *testing.T) {
	when := time.Date(2026, 11, 1, 3, 4, 5, 0, time.UTC)
	in := wireThing{ID: 1 << 60, Path: "/cores/a.so", Inner: wireInner{Name: "n", Aliases: []string{"a", "b"}, Hidden: true},
		List: []wireInner{{Name: "x", Hidden: true}}, Ptr: &wireInner{Aliases: []string{"p"}}, ByName: map[string]string{"k": "v"},
		ByNumber: map[int]wireInner{7: {Name: "seven"}}, Blob: []byte{1, 2, 3}, When: when, Took: time.Second, Any: "text", Fixed: [2]int{4, 5},
		Log: slog.Default(), Out: io.Discard, Do: func() int { return 1 }, secret: "no"}
	raw, err := Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out wireThing
	if err := Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	want := in
	want.Log, want.Out, want.Do, want.secret = nil, nil, nil, ""
	if !reflect.DeepEqual(out, want) {
		t.Fatalf("%+v\nwant %+v", out, want)
	}
	// Its ordinary JSON, for a browser, still leaves them out.
	if b, _ := json.Marshal(wireInner{Name: "n", Aliases: []string{"secret"}}); string(b) != `{"name":"n"}` {
		t.Fatal(string(b))
	}
}

func TestWireNils(t *testing.T) {
	for _, v := range []any{nil, (*wireThing)(nil), []string(nil), map[string]string(nil)} {
		raw, err := Marshal(v)
		if err != nil || string(raw) != "null" {
			t.Fatalf("%v: %s %v", v, raw, err)
		}
	}
	got := []string{"stays?"}
	if err := Unmarshal([]byte("null"), &got); err != nil || got != nil {
		t.Fatal(got, err)
	}
	empty, err := Marshal([]string{})
	if err != nil || string(empty) != "[]" {
		t.Fatal(string(empty), err)
	}
}

type wireOwnForm struct{ N int }

func (w wireOwnForm) MarshalJSON() ([]byte, error) { return json.Marshal(map[string]int{"n": w.N * 2}) }
func (w *wireOwnForm) UnmarshalJSON(b []byte) error {
	var m map[string]int
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	w.N = m["n"] / 2
	return nil
}

// A type that says how it's written is written that way.
func TestWireRespectsAFormOfItsOwn(t *testing.T) {
	type holder struct {
		A wireOwnForm
		B *wireOwnForm
		C []wireOwnForm
	}
	in := holder{A: wireOwnForm{1}, B: &wireOwnForm{2}, C: []wireOwnForm{{3}}}
	raw, err := Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out holder
	if err := Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("%+v %s", out, raw)
	}
}

// One that contains itself is refused, not followed for ever.
func TestWireTooDeep(t *testing.T) {
	type loop struct{ Next *loop }
	l := &loop{}
	l.Next = l
	if _, err := Marshal(l); err == nil {
		t.Fatal("no error")
	}
}

// An embedded struct's fields are written as the struct's own, whatever embeds it on the other side.
func TestWireEmbedded(t *testing.T) {
	type holder struct {
		wireInner
		Extra []string
	}
	raw, err := Marshal(holder{wireInner: wireInner{Name: "n", Hidden: true}, Extra: []string{"e"}})
	if err != nil {
		t.Fatal(err)
	}
	var flat struct {
		Name   string
		Hidden bool
		Extra  []string
	}
	if err := Unmarshal(raw, &flat); err != nil || flat.Name != "n" || !flat.Hidden || len(flat.Extra) != 1 {
		t.Fatalf("%+v %v %s", flat, err, raw)
	}
	var back holder
	if err := Unmarshal(raw, &back); err != nil || back.Name != "n" || !back.Hidden {
		t.Fatalf("%+v %v", back, err)
	}
}
