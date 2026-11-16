# WittyModule

What a [WittyMediaServer](https://github.com/wittywilla/WittyMediaServer) module needs to be a project of its own: the manifest, the rules for version ranges, the protocol a helper process speaks, what a module adds to a server and to every app, and the tool that builds and signs a release. It's for people who make modules. WittyMediaServer's site is https://wittymediaserver.com.

**The specification** of a module, everything one is held to at module API 5, is [docs/SPEC.md](docs/SPEC.md): its manifest, delivery kinds, permissions, the helper protocol, its SDUI, releases and signing, catalogues and stores, withdrawals, forks, its status, and the overrides a server allows.

<!-- pictures -->

![The module store in WittyMediaServer](docs/images/module-store.webp)

<sub>What a module becomes: its entry in the store.</sub>

![A module's page in the store](docs/images/store-module.webp)

<sub>A module's own page in the store.</sub>

<!-- /pictures -->

## How it fits

WittyMediaServer 0.8.0-alpha takes module API 5 only, with no backwards compatibility: a module built for an older API is refused and needs a new release. This version of WittyModule is module API 5 (`api: 5` in [module.json](module.json)). Every module in the store is built with it, and the server uses the same code to read and check what a module says about itself.

It's a library, not a module you install: a server gets it inside the modules built with it, so there's nothing to add from the store. Its own manifest gives WittyMediaServer 0.6.0-alpha through 0.8.x as its range, and a module built with it runs on a server that takes module API 5, which is 0.8.0-alpha and later in the 0.8 series. People add the modules themselves in **Settings → Modules → Browse the store → Add**.

## What it does

- **The manifest:** `module.json`, parsed and checked here.
- **Version ranges:** which servers a version fits.
- **The helper protocol:** what a helper program and a server say to each other. A helper is a small program:

```go
func main() {
	wittymodule.Main(manifestJSON, func(method string, params json.RawMessage) (any, []byte, error) {
		// answer one call
	})
}
```

- **What a module adds to a server:** `integration.json`, which every installed module carries from module API 5: its routes, its web part, its own data, its lifecycle calls and its scan hooks. WittyMediaServer's docs/MODULES-API5.md describes it.
- **The module's SDUI:** `sdui/` writes and checks the `sdui.json` every installed module carries: what it adds to every app, as data the server checks and every app draws. `kit/sdui/` is the same for web bundles in TypeScript. WittyMediaServer's docs/SDUI-MODULES.md describes it.
- **Services:** `service/` is both ends of a helper that takes calls that overlap and connections a browser makes straight to it.
- **The catalogue:** the code that reads and checks a store's signed list of modules.
- **Withdrawing a version:** the signed list that takes a version back.
- **The `wittymodule` tool:**

| Command | What it does |
|---|---|
| `check` | Checks a folder's manifest, `sdui.json` and `integration.json`, listing every problem by file and field |
| `pack` | Builds a release: a program for every system, or a browser bundle with `-web` |
| `sign`, `keygen` | Signs a checksum list, and makes a key |
| `bump`, `needs-bump` | Raises the version everywhere it's written, and fails if code changed since the last tag with no new version |
| `withdraw`, `restore` | Takes a version back, and undoes that |
| `catalog-add`, `catalog-check`, `catalog-sign` | Works on a store's catalogue |

- **`kit/`:** `kit/module-kit.sh` writes a module's workflows and the Releasing section of its README, `kit/facade` writes the stubs a server uses to call a module, and `kit/web` builds a module's web part.

The reasoning behind the specification, including what each version range means, is in Making a module, in **Help** inside WittyMediaServer, and in the server's docs/MODULES.md.

## How to get it

Modules bring it in as a Go dependency:

```bash
go get github.com/wittywilla/wittymodule
```

```bash
go run github.com/wittywilla/wittymodule/cmd/wittymodule check .
```

## What it needs

Go 1.26 or newer. It has no dependencies, and no cgo. The TypeScript kit's tests need Node.js (CI uses 24).

## Platforms

It's plain Go, and CI builds it for Windows, macOS, Linux and FreeBSD. Its tests have been run on Linux, and `pack` has built modules for all four from there, including FreeBSD on amd64 and arm64. Those have not been run on a real Mac or on FreeBSD. `kit/module-kit.sh` needs bash.

## Building and testing

What CI runs:

```bash
go run ./cmd/wittymodule check .
```

```bash
go vet ./...
```

```bash
go test ./... -count=1
```

```bash
node --test kit/sdui/tests/*.test.mjs
```

A module's own `pack` and `sign` go like this:

```bash
go run github.com/wittywilla/wittymodule/cmd/wittymodule pack -out release .
```

```bash
go run github.com/wittywilla/wittymodule/cmd/wittymodule sign release/<id>-<version>-SHA256SUMS.txt
```

To make or refresh a module's workflows:

```bash
kit/module-kit.sh ../WittyPhotos
```

## Licence

GNU Affero General Public License v3.0 (or later): see [LICENSE](LICENSE) and [NOTICE](NOTICE). If you redistribute it or build on it, please keep the NOTICE attribution.

<!-- module-kit -->
## Releasing

WittyModule reaches a server inside the projects built with it, so a release has no files of its own: a version tag runs the tests and publishes a release that those projects can require.

Versions move in small steps as work lands. `go run github.com/wittywilla/wittymodule/cmd/wittymodule@v0.5.5 bump .` raises it
everywhere it is written (add `-part minor` for a bigger step). A bump isn't a release: nothing is
tagged or published until you tag it. CI fails when code changed since the last tag and the version
didn't, because tags never move.

To release:

1. Check that `version` in `module.json` is the one to publish, with the bump committed.
2. Tag that commit and push the tag: `git tag vX.Y.Z`, then `git push origin vX.Y.Z`. A tag with a
   suffix, such as `v1.2.0-beta`, is published as a prerelease.
3. Watch the *Actions* tab. There is nothing to sign.

Licensed under AGPL-3.0-or-later: see [LICENSE](LICENSE) and [NOTICE](NOTICE).
<!-- /module-kit -->

## Legal

WittyModule is an independent open-source project. It isn't associated with, sponsored by or
endorsed by any publisher, rights holder, service or other media software project. A module built
with it comes from whoever published that module. Any third-party names here only describe what it
works with, and belong to their owners. It's provided “as is”, with no warranty; see
[LICENSE](LICENSE). Local law applies to how you use it. This isn't legal advice.
