// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors

package wittymodule

import _ "embed"

// ManifestJSON is this module's manifest (docs/MODULES.md, "The manifest").
//
//go:embed module.json
var ManifestJSON []byte
