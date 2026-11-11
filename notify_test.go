// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors

package wittymodule

import (
	"encoding/json"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

// pair runs a helper over pipes and returns the server's end of it. The test ends it by
// closing what it returns.
func pair(t *testing.T, h Handler) (*Conn, func()) {
	t.Helper()
	toHelper, helperIn := io.Pipe()
	fromHelper, helperOut := io.Pipe()
	done := make(chan error, 1)
	go func() { done <- Serve(toHelper, helperOut, Hello{Module: "wittytest", Version: "1.0.0", API: API}, h) }()
	c := NewConn(fromHelper, helperIn)
	if _, err := c.ReadHello(); err != nil {
		t.Fatal(err)
	}
	return c, func() {
		helperIn.Close()
		if err := <-done; err != nil {
			t.Error(err)
		}
		fromHelper.Close()
	}
}

func TestNotifyDuringACall(t *testing.T) {
	c, stop := pair(t, func(method string, _ json.RawMessage) (any, []byte, error) {
		for i := int64(1); i <= 3; i++ {
			if err := ReportTask(TaskReport{Key: "scan", Title: "Scanning", State: TaskRunning, Done: i, Total: 3}); err != nil {
				return nil, nil, err
			}
		}
		return map[string]string{"said": method}, []byte("blob"), nil
	})
	defer stop()
	var got []TaskReport
	c.OnNotify = func(method string, params json.RawMessage) {
		var r TaskReport
		if method != NotifyTask || json.Unmarshal(params, &r) != nil {
			t.Errorf("heard %s %s", method, params)
		}
		got = append(got, r)
	}
	var out struct{ Said string }
	blob, err := c.Call("hello", nil, &out)
	if err != nil || out.Said != "hello" || string(blob) != "blob" {
		t.Fatal(out, string(blob), err)
	}
	if len(got) != 3 || got[2].Done != 3 {
		t.Fatalf("heard %+v", got)
	}
}

// A server that doesn't care still gets its answers.
func TestNotifyNobodyListening(t *testing.T) {
	c, stop := pair(t, func(string, json.RawMessage) (any, []byte, error) {
		return nil, nil, Notify("anything", 1)
	})
	defer stop()
	if _, err := c.Call("x", nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestListenHearsBetweenCalls(t *testing.T) {
	go1 := make(chan struct{})
	c, stop := pair(t, func(method string, _ json.RawMessage) (any, []byte, error) {
		if method == "begin" {
			go func() {
				<-go1
				for i := 0; i < 1000; i++ { // more than a pipe holds, with no call being made
					_ = Notify("step", strings.Repeat("x", 512))
				}
				_ = Notify("end", nil)
			}()
		}
		return "ok", nil, nil
	})
	defer stop()
	heard := make(chan string, 1)
	steps := 0
	var mu sync.Mutex
	c.OnNotify = func(method string, _ json.RawMessage) {
		mu.Lock()
		defer mu.Unlock()
		if method == "step" {
			steps++
		}
		if method == "end" {
			heard <- method
		}
	}
	c.Listen()
	if _, err := c.Call("begin", nil, nil); err != nil {
		t.Fatal(err)
	}
	close(go1)
	select {
	case <-heard:
	case <-time.After(10 * time.Second):
		t.Fatal("the notifications stopped with no call being made")
	}
	mu.Lock()
	if steps != 1000 {
		t.Errorf("heard %d steps", steps)
	}
	mu.Unlock()
	var out string
	if _, err := c.Call("again", nil, &out); err != nil || out != "ok" {
		t.Fatal(out, err)
	}
	if _, err := c.Stream("x", nil, nil); err == nil {
		t.Fatal("a listening Conn was allowed to stream")
	}
}

// Many goroutines at once mustn't tear an answer's blob or each other's lines.
func TestNotifyKeepsAnswersWhole(t *testing.T) {
	blob := []byte(strings.Repeat("b", 256<<10))
	var wg sync.WaitGroup
	stopNoise := make(chan struct{})
	c, stop := pair(t, func(string, json.RawMessage) (any, []byte, error) { return nil, blob, nil })
	defer stop()
	c.OnNotify = func(string, json.RawMessage) {}
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stopNoise:
					return
				default:
					_ = Notify("noise", "x")
				}
			}
		}()
	}
	for range 20 {
		got, err := c.Call("x", nil, nil)
		if err != nil || len(got) != len(blob) {
			t.Fatal(len(got), err)
		}
	}
	// Noise left in the pipe has to be read, or the writers won't stop.
	c.Listen()
	close(stopNoise)
	wg.Wait()
}

func TestNotifyWithNoServe(t *testing.T) {
	if err := Notify("x", nil); err != ErrNotServing {
		t.Fatal(err)
	}
}

func TestTaskReportValidate(t *testing.T) {
	good := TaskReport{Key: "scan.films", Title: "Scanning", State: TaskRunning, Done: 1, Total: 2, Item: "a file"}
	if err := good.Validate(); err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(r *TaskReport){
		"key":          func(r *TaskReport) { r.Key = "../other" },
		"another's":    func(r *TaskReport) { r.Key = "wittyplay/scan" },
		"no title":     func(r *TaskReport) { r.Title = "" },
		"long title":   func(r *TaskReport) { r.Title = strings.Repeat("x", 201) },
		"control":      func(r *TaskReport) { r.Item = "a\x1bb" },
		"line break":   func(r *TaskReport) { r.Title = "a\nb" },
		"state":        func(r *TaskReport) { r.State = "paused" },
		"negative":     func(r *TaskReport) { r.Done = -1 },
		"past the end": func(r *TaskReport) { r.Done = 3 },
	} {
		r := good
		change(&r)
		if r.Validate() == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestManifestPages(t *testing.T) {
	base := func() *Manifest {
		return &Manifest{ID: "wittystats", Name: "WittyStats", Version: "0.1.0", Repo: "wittywilla/WittyStats", Delivery: DeliveryWeb, Entry: "index.js",
			API: API, MinServer: "0.6.0-alpha", MaxServer: "0.6.x", Pages: []Page{{ID: "stats", Title: "Statistics", Where: PageMain, Icon: "chart", Admin: true}}}
	}
	if err := base().ValidateRelease(); err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(m *Manifest){
		"bad id":       func(m *Manifest) { m.Pages[0].ID = "../x" },
		"two of an id": func(m *Manifest) { m.Pages = append(m.Pages, m.Pages[0]) },
		"no title":     func(m *Manifest) { m.Pages[0].Title = "" },
		"bad place":    func(m *Manifest) { m.Pages[0].Where = "anywhere" },
		"bad icon":     func(m *Manifest) { m.Pages[0].Icon = "<svg>" },
		"not web":      func(m *Manifest) { m.Delivery, m.Entry = DeliveryHelper, "wittystats-module" },
	} {
		m := base()
		change(m)
		if m.Validate() == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}
