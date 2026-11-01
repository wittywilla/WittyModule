# Changelog

What changed in WittyModule, newest first. What changed before it had a repository of its own is in the
[WittyMediaServer changelog](https://github.com/wittywilla/WittyMediaServer/blob/master/CHANGELOG.md).

## Unreleased

- A service and the server send every field across, by its Go name, including those hidden from JSON. Only what is live stays behind.
- `pack` writes THIRD-PARTY-NOTICES.txt into each archive: the license of every library in a module's programs. A library with no license file stops it.
- The kit: a helper with a part in the browser builds that first and packs them together, and modules without a lock file install their packages unlocked.

- `DiagJob` and the `diag.*` notifications: a module can check what it reads, and report as it goes.
- Module API 3 is also the oldest a server now takes, and a module's `minServer` starts at 0.6.0-alpha: nothing of a module is compiled into the server from there on.
- A helper can ask for the `programs`, `devices` and `data` permissions, bring more programs with it (`programs`), and have a part in the browser (`webEntry`, `webStyle`, built with `pack -web`).
- Module API 3. A helper can tell the server things unasked with `Notify`, and report background work with `ReportTask`. A web module can add pages to the app, listed in its manifest.
- The catalogue: `Catalog` and its checks, and `wittymodule catalog-add`, `catalog-check` and `catalog-sign`. One signed file saying which modules exist and where each version's files are.
- WittyModule is now a project of its own, developed and released from this repository.
