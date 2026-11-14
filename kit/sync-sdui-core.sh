#!/usr/bin/env bash
# SPDX-License-Identifier: AGPL-3.0-or-later
# Copyright (C) 2026 WittyWilla and contributors
# Copies the server's SDUI library (WittyMediaServer pkg/sdui, standard library only) into
# sdui/core, so that sdui.Check is the server's own validator and a module that
# passes `wittymodule check` is never refused at install. Run it after the server's pkg/sdui
# changes; sdui's tests fail while the copy differs from a server checkout beside this one.
#
#   kit/sync-sdui-core.sh [path/to/WittyMediaServer]
set -euo pipefail
here=$(cd "$(dirname "$0")/.." && pwd)
server=${1:-${WMS_SERVER:-$here/../WittyMediaServer}}
src="$server/pkg/sdui"
[ -f "$src/module.go" ] || { echo "no server SDUI library at $src" >&2; exit 1; }
dst="$here/sdui/core"
mkdir -p "$dst"
find "$dst" -maxdepth 1 -name '*.go' -delete
for f in "$src"/*.go; do
  case "$f" in *_test.go) continue;; esac
  cp "$f" "$dst/"
done
# the data tables and experiences the library embeds
for d in data experiences; do
  rm -rf "$dst/$d"
  mkdir -p "$dst/$d"
  cp "$src/$d"/*.json "$dst/$d/"
done
echo "copied $(ls "$dst"/*.go | wc -l | tr -d ' ') files from $src"
