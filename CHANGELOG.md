# Changelog

What changed in WittyModule, newest first. What changed before it had a repository of its own is in the
[WittyMediaServer changelog](https://github.com/wittywilla/WittyMediaServer/blob/master/CHANGELOG.md).

## Unreleased

- A catalogue entry can list screenshots. `catalog-sign` also writes `catalog.json.key`, the public key a store offers the first time an admin adds it.
- Module API 3. A helper can tell the server things unasked with `Notify`, and report background work with `ReportTask`. A web module can add pages to the app, listed in its manifest.
- The catalogue: `Catalog` and its checks, and `wittymodule catalog-add`, `catalog-check` and `catalog-sign`. One signed file saying which modules exist and where each version's files are.
- WittyModule is now a project of its own, developed and released from this repository.
