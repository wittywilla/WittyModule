# Changelog

What changed in WittyModule, newest first. What changed before it had a repository of its own is in the
[WittyMediaServer changelog](https://github.com/wittywilla/WittyMediaServer/blob/master/CHANGELOG.md).

## Unreleased

- The SDUI kit follows WittyMediaServer's catalogue revision 4: `sdui.CatalogRevision` and the TypeScript kit's lists (with `Dock` and `KeyLegend`), the copies of the catalogue and schema, and the server's validator in `sdui/internal/core`, with `watchLater.remove` and `inWatchLater`. A file the kit writes says `"catalog": 4`; files made for an earlier revision are still taken.
- The SDUI kit's copies of the server's catalogue and schema (`sdui/`, `kit/sdui/`) are WittyMediaServer 0.8.0-alpha's: the `books`, `statistics` and `liked` sources, rows for each series or author of a books library (`bookSeries`, `bookAuthors`), the layouts and theme tokens, and `topShelf` on a module's source, which offers it for Apple TV's Top Shelf (only a source with no required parameters and no `pageKinds`).
- **The module specification**, [docs/SPEC.md](docs/SPEC.md): Specification 4.0, module API 4. Everything a module is held to, in one place, with what API 3 differs in, the server's overrides, and a change log of its own.
- `wittymodule check` lists every problem, not just the first, one line each and located by file and field or JSON path: `module.json: maxServer: ...`, `sdui.json: /pages/0/title: ...`. The SDUI checker's warnings are printed too, and don't fail it. A module whose `status` is `experimental` or `incomplete` passes, with a warning naming its status and note.
- **A catalogue entry says how finished a version is, and what it adds to the apps:** `status`, `statusNote` and `ui` on a version (`ui`: its pages by id and title, its menu entries by label, whether it has settings, and how many sources and inserts it has; at most 32 pages with titles of 120 characters and 16 labels of 40). `wittymodule catalog-add` takes the status from the release's manifest and sums up the `sdui.json` inside its signed archive (`sdui.Summary`), checked against the signed list; it refuses an API-4 release with no `sdui.json` or one a server would refuse. Readers that don't know the fields ignore them, so the format stays 1.
- **A version says how finished it is:** `status` in `module.json` (`stable` when left out, `beta`, `experimental` or `incomplete`) and `statusNote`, one line of at most 500 characters saying what isn't done or may break, required for `experimental` and `incomplete`. A server warns before installing an unfinished version and asks an administrator to agree. `Manifest.Problems` lists everything wrong with a manifest, each with its field; `Validate` still reports the first.
- `sdui.Check` (and so `wittymodule check` and the golden test) is the server's own validator: `sdui/internal/core` is a copy of WittyMediaServer's `pkg/sdui`, kept current with `kit/sync-sdui-core.sh` and compared with a server checkout beside this one by the tests. A module that passes is no longer refused at install for its SDUI (a settings label over 80 characters once passed). `wittymodule check` also refuses data sources on a module that isn't a helper.
- **A service module answers the server's SDUI calls:** `service.Config.UI` (`sdui.Answers`) answers `sdui.items`, `sdui.image` and `settings.changed` on the helper protocol, which a service otherwise refused as unknown methods.

- Module API 4: a module carries its own SDUI, `sdui.json` beside its manifest: its pages, sections, components, data sources, views, menu entries, settings and theme scope (WittyMediaServer docs/SDUI-MODULES.md). `wittymodule check` requires it from API 4 and checks it; `pack` puts it in every archive. The kit is `sdui/` for Go (write, check, golden test, answer the server's `sdui.items`, `sdui.image` and `settings.changed`) and `kit/sdui/` for web bundles. Servers that take API 4 still take API 3.
- The kit writes the licence each project's own `module.json` or `package.json` names, instead of Apache-2.0 for all of them. Running it again no longer undoes WittyCouch Host's README.
- `pack -prebuilt <folder>` packs a program built with something other than Go, for the systems it was built for. The kit leaves such a repository's release workflow alone.
- A catalogue entry can list screenshots. `catalog-sign` also writes `catalog.json.key`, the public key a store offers the first time an admin adds it.
- A service and the server send every field across, by its Go name, including those hidden from JSON. Only what is live stays behind.
- `pack` writes THIRD-PARTY-NOTICES.txt into each archive: the license of every library in a module's programs. A library with no license file stops it.
- The kit: a helper with a part in the browser builds that first and packs them together, and modules without a lock file install their packages unlocked.

- `DiagJob` and the `diag.*` notifications: a module can check what it reads, and report as it goes.
- Module API 3 is also the oldest a server now takes, and a module's `minServer` starts at 0.6.0-alpha: nothing of a module is compiled into the server from there on.
- A helper can ask for the `programs`, `devices` and `data` permissions, bring more programs with it (`programs`), and have a part in the browser (`webEntry`, `webStyle`, built with `pack -web`).
- Module API 3. A helper can tell the server things unasked with `Notify`, and report background work with `ReportTask`. A web module can add pages to the app, listed in its manifest.
- The catalogue: `Catalog` and its checks, and `wittymodule catalog-add`, `catalog-check` and `catalog-sign`. One signed file saying which modules exist and where each version's files are.
- WittyModule is now a project of its own, developed and released from this repository.
