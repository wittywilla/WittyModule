// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors

package sdui

import (
	"encoding/json"
	"os"
)

// The calls a server makes to a module's helper for its SDUI.
const (
	MethodItems           = "sdui.items"
	MethodImage           = "sdui.image"
	MethodSettingsChanged = "settings.changed"
)

// SettingsEnv names the environment a helper is started with: its server settings, as a
// JSON object, defaults filled in.
const SettingsEnv = "WITTYMODULE_SETTINGS"

// ItemsRequest is a server asking for one of the module's sources.
type ItemsRequest struct {
	// Source is the section's source, with the page's variables filled in.
	Source Source            `json:"source"`
	Filter map[string]string `json:"filter,omitempty"`
	Offset int               `json:"offset"`
	Limit  int               `json:"limit"`
	Viewer Viewer            `json:"viewer"`
	// Settings are the module's settings for this profile: the server's, then the
	// profile's own, defaults filled in.
	Settings map[string]any `json:"settings,omitempty"`
}

// Viewer is who the page is for.
type Viewer struct {
	// User is the account the page is for (its id on the server), for a module's own data about
	// that person: what they played, their friends, their saves.
	User        int64    `json:"user,omitempty"`
	ProfileKind string   `json:"profileKind"`
	Admin       bool     `json:"admin,omitempty"`
	Cannot      []string `json:"cannot,omitempty"`
	FormFactor  string   `json:"formFactor,omitempty"`
	Platform    string   `json:"platform,omitempty"`
	Locale      string   `json:"locale,omitempty"`
}

// ItemsAnswer is the module's answer. The server keeps only what a card may carry and the
// profile may see: give contentRating and genres where they apply, routes in the app,
// and pictures on the server (the module's own: ImagePath).
type ItemsAnswer struct {
	Items []Item `json:"items"`
	More  bool   `json:"more,omitempty"`
	Total int    `json:"total,omitempty"`
}

// ImagePath is where a server serves a picture of the module's own, answered by Image.
func ImagePath(module, name string) string {
	return "/api/v1/sdui/modules/" + module + "/images/" + name
}

// Answers are a helper's answers to the server's SDUI calls. Any may be nil.
type Answers struct {
	Items func(ItemsRequest) (ItemsAnswer, error)
	// Image answers a picture by name: PNG, JPEG, WebP or GIF, at most 4 MiB.
	Image func(name string) ([]byte, error)
	// SettingsChanged hears the server's settings after an administrator saved them.
	SettingsChanged func(values map[string]any)
}

// Handle answers one call if it is one of these: handled is false for any other method,
// for the helper's own handler to answer. Use it first in a wittymodule.Handler:
//
//	if res, blob, err, ok := answers.Handle(method, params); ok {
//		return res, blob, err
//	}
func (a Answers) Handle(method string, params json.RawMessage) (result any, blob []byte, err error, handled bool) {
	switch method {
	case MethodItems:
		if a.Items == nil {
			return nil, nil, nil, false
		}
		var req ItemsRequest
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, nil, err, true
		}
		ans, err := a.Items(req)
		if ans.Items == nil {
			ans.Items = []Item{}
		}
		return ans, nil, err, true
	case MethodImage:
		if a.Image == nil {
			return nil, nil, nil, false
		}
		var req struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, nil, err, true
		}
		b, err := a.Image(req.Name)
		return map[string]any{}, b, err, true
	case MethodSettingsChanged:
		var req struct {
			Values map[string]any `json:"values"`
		}
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, nil, err, true
		}
		if a.SettingsChanged != nil {
			a.SettingsChanged(req.Values)
		}
		return map[string]any{}, nil, nil, true
	}
	return nil, nil, nil, false
}

// Settings are the server settings the helper was started with.
func Settings() map[string]any {
	out := map[string]any{}
	_ = json.Unmarshal([]byte(os.Getenv(SettingsEnv)), &out)
	return out
}
