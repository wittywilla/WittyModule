# The WittyMediaServer module specification

**Specification 4.0, module API 4.** Written for WittyModule's `wittymodule` package and tool and for
WittyMediaServer 0.8.0-alpha. It also says where module API 3, which servers still take, differs.

This is the whole contract a module is held to: what it is called, what its manifest may say, what a
release holds and how it is signed, how its programs talk to a server, what it adds to every app, how
it is listed in a catalogue, how it is taken back, and what a server lets an administrator override.
It is written for people who make modules and for people who write a server or a store that reads
them.

Three rules decide any question this document doesn't settle:

1. **The code wins over the prose.** Where this document and WittyModule's code disagree, the code
   is right and this document is wrong: `manifest.go`, `protocol.go`, `catalog.go`, `withdraw.go`,
   `task.go`, `diag.go`, `service/`, `sdui/` and `cmd/wittymodule` in WittyModule, and
   `internal/modules` and `pkg/sdui` in WittyMediaServer.
2. **The server is authoritative.** A server decides what it installs, runs and allows. A module,
   its manifest, a catalogue or a store can only ask. Everything a module says is untrusted input
   to a server.
3. **Guides elsewhere explain; this specifies.** WittyMediaServer's `docs/MODULES.md` (making a
   module), `docs/SDUI-MODULES.md` (the full `sdui.json` format), `docs/SDUI.md` (server-driven UI)
   and `docs/MODULE-STORE.md` (the store, for administrators) have the reasoning and the pictures.

"Must" and "must not" are requirements: a module or release that breaks one is refused. "Should" is
strong advice. Sizes are in bytes unless they say characters (Unicode code points).

## Contents

1. [Numbers that version things](#1-numbers-that-version-things)
2. [Identity and naming](#2-identity-and-naming)
3. [The manifest](#3-the-manifest)
4. [Status: saying a version is unfinished](#4-status-saying-a-version-is-unfinished)
5. [Delivery kinds](#5-delivery-kinds)
6. [Compatibility](#6-compatibility)
7. [Permissions and limits](#7-permissions-and-limits)
8. [The helper protocol](#8-the-helper-protocol)
9. [Services](#9-services)
10. [Web parts](#10-web-parts)
11. [The SDUI contribution](#11-the-sdui-contribution)
12. [Settings](#12-settings)
13. [Theme scope](#13-theme-scope)
14. [Data formats](#14-data-formats)
15. [Logging and capabilities](#15-logging-and-capabilities)
16. [Releases](#16-releases)
17. [What a server does with a release](#17-what-a-server-does-with-a-release)
18. [The catalogue](#18-the-catalogue)
19. [Stores other than this project's](#19-stores-other-than-this-projects)
20. [Withdrawals](#20-withdrawals)
21. [Forks](#21-forks)
22. [Overrides](#22-overrides)
23. [Checking a module](#23-checking-a-module)
24. [Module API 3](#24-module-api-3)
25. [Change log](#25-change-log)

## 1. Numbers that version things

Several numbers move independently. None of them is the server's version.

| Number | Where | Today | What raises it |
|---|---|---|---|
| Specification | This document's title | 4.0 | The major part follows the module API. The minor part rises when the specification is corrected or extended without changing the API. |
| Module API | `api` in `module.json`; `wittymodule.API` and `MinAPI`; the server's own `API` and `MinAPI` in `internal/modules/sdui.go` | 4, and servers take 3 to 4 | Adding to the contract between a server and its modules raises `API`. Breaking it raises `MinAPI` too, which disables older modules, so that is rare. |
| Catalogue format | `format` in `catalog.json`; `wittymodule.CatalogFormat` | 1 | A change readers can't ignore. Adding a field doesn't raise it. |
| SDUI file format | `format` and `formatVersion` in `sdui.json` | `wittymodule-sdui`, 1 | A change in the file's shape. |
| SDUI catalogue revision | `catalog` in `sdui.json`; `sdui.CatalogRevision` | 2 | New components, sources or properties in the server's SDUI catalogue. |
| Client API revision | `apiRevision` in WittyMediaServer's Client API | (the server's) | Nothing to do with modules. |

The module API, in short:

| API | Added | Taken by servers |
|---|---|---|
| 1 | The manifest, helpers and the helper protocol | No longer |
| 2 | Web bundles, programs, a helper's attached part, streams | No longer |
| 3 | Notifications, services, permissions, pages, a helper's part in the browser; nothing of a module compiled into the server | 0.6.0-alpha and later |
| 4 | `sdui.json` in every installed module; `status` and `statusNote` | 0.8.0-alpha and later |

Nothing in API 3 changed in API 4, so a version built for API 3 still runs on a server that takes 4.
[Section 24](#24-module-api-3) lists what is different for one.

## 2. Identity and naming

**The module ID** (`id`) is what everything else is named after. It must match
`^[a-z][a-z0-9]{1,63}$`: a lower-case letter, then 1 to 63 lower-case letters and digits, 2 to 64 in
all. No hyphens, underscores or capitals. It never changes for the life of a module; a fork keeps it
([section 21](#21-forks)). The folder a module's checkout is in must have the same name, compared in
lower case, unless the folder is the top of a git checkout or worktree (which may be named anything).

**The name** (`name`) is what people see, such as `WittyPhotos`. It must not be empty.

**The repository** (`repo`) is `owner/name` on GitHub, matching `^[A-Za-z0-9-]+/[A-Za-z0-9._-]+$`.
It is where releases are fetched from when a module is used without a catalogue, and what a fork
changes.

**Versions** follow semantic versioning: `MAJOR.MINOR.PATCH`, optionally `-PRERELEASE` (dot-separated
parts, none empty) and `+BUILD`. A leading `v` is ignored. Only ASCII letters, digits, `.`, `-` and
`+` may appear, because a version becomes a folder name. Ordering is semantic versioning's: a
pre-release comes before its release (`0.8.0-alpha` < `0.8.0`), numeric parts compare as numbers
and sort before words, and build metadata is ignored. A version that can't be read sorts before
every other. Versions stay below 1.0 while the APIs settle.

**Names inside a release** are fixed by the ID and version (`V` is the version without a leading `v`):

| What | Name |
|---|---|
| Archive for a system | `<id>-V-<goos>-<arch>.tar.gz`, or `.zip` on Windows. `<arch>` is GOARCH, except `armv7` for `linux/arm` |
| Its one top folder | `<id>-V-<goos>-<arch>` |
| Web bundle archive | `<id>-V-web.tar.gz`, top folder `<id>-V-web` |
| Checksum list | `<id>-V-SHA256SUMS.txt`, and its signature `<id>-V-SHA256SUMS.txt.sig` |
| Manifest beside the archives | `<id>-V-module.json` |
| Withdrawal list | `<id>-withdrawn.json` and `.sig`, on the release tagged `withdrawals` |

**Names in what a module adds to the apps** all begin with the module's ID, so two modules never
clash: pages, inserted sections, presets, menu entries and settings pages are `<id>` or `<id>-…`;
sources and views are `<id>.<camelName>`; components are `<id>.<PascalName>`
([section 11](#11-the-sdui-contribution)).

## 3. The manifest

Every module has `module.json` at the root of its repository and of every archive, covered by the
release's signature. It is JSON, UTF-8, one object. Fields a reader doesn't know are ignored, which
is how an older server reads a newer manifest. `wittymodule.ParseManifest` reads it and
`Manifest.Problems` lists everything wrong with it.

```json
{
  "id": "wittyphotos",
  "name": "WittyPhotos",
  "version": "0.4.0",
  "repo": "wittywilla/WittyPhotos",
  "license": "AGPL-3.0-or-later",
  "delivery": "helper",
  "entry": "wittyphotos-module",
  "api": 4,
  "minServer": "0.8.0-alpha",
  "maxServer": "0.8.x",
  "group": "photos",
  "summary": "Photo libraries: what a photo says about itself.",
  "libraries": ["photos"],
  "updates": "Reading EXIF, picture sizes, and the preview inside RAW files.",
  "stays": "The photo pages in the app.",
  "status": "beta",
  "statusNote": "RAW previews from some cameras are still slow."
}
```

| Field | Type | Required | Rules and limits |
|---|---|---|---|
| `id` | string | yes | [The module ID](#2-identity-and-naming). |
| `name` | string | yes | Not empty. |
| `version` | string | yes | A [version](#2-identity-and-naming). |
| `repo` | string | yes | `owner/name` on GitHub. |
| `license` | string | for a release | Its licence, as an SPDX expression, shown in the store. A catalogue entry needs one, and it must equal the catalogue's. |
| `delivery` | string | yes | `helper`, `program`, `web`, `library` or `builtin` ([section 5](#5-delivery-kinds)). |
| `entry` | string | `helper`, `program`; a released `web` | For a helper or program: the program's name without `.exe`, built from `cmd/<entry>`. For a web bundle: its script, a file in the bundle. Must match `^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`. A web bundle may leave it empty until its first release. |
| `style` | string | no | A web bundle's stylesheet, a file in the bundle; same pattern. |
| `programs` | list of strings | no | More programs that come with a helper or program, each built from `cmd/<name>`, same pattern, at most 64. Not for other deliveries. |
| `webEntry`, `webStyle` | string | no | The script and stylesheet of a helper's or program's part in the browser, in `web/` inside each archive. Same pattern. Not for other deliveries. |
| `api` | int | yes | The module API it was built for: at least 1 to parse; 3 or 4 to be released or run. |
| `minServer` | string | yes | The oldest server it fits, a version. At least `0.6.0-alpha` (`wittymodule.FirstServer`); for API 4, `0.8.0-alpha` in practice, the first server that reads `sdui.json`. |
| `maxServer` | string | yes | The newest server it fits: an exact version, or a series such as `0.8.x` or `0.x`. A lone `x` is refused. Must not be below `minServer`. See [section 6](#6-compatibility). |
| `capabilities` | object of string lists | no | Facts a server needs without starting the program, such as `rawExtensions` and `logTopics` ([section 15](#15-logging-and-capabilities)). Keys, values and lengths together at most 4,096 bytes. |
| `permissions` | list of strings | no | `network`, `programs`, `devices`, `data` ([section 7](#7-permissions-and-limits)); at most 64 entries. Any other word is refused by `wittymodule`; a server that doesn't know a permission won't run the version. |
| `dataFormat`, `dataFrom` | int | no | The newest data format it writes and the oldest it reads ([section 14](#14-data-formats)). Both 0 or more, and `dataFrom` ≤ `dataFormat`. |
| `group` | string | no | Which shelf of the store it goes on, at most 2,000 bytes. A catalogue needs it to be a word: `^[a-z][a-z0-9-]{0,63}$`. |
| `summary` | string | no | One line for the store, at most 2,000 bytes. |
| `needs` | list of strings | no | What it needs in order to work, each at most 2,000 bytes, at most 64. |
| `updates`, `stays` | string | no | Honestly: which parts a release of its own changes, and which stay with the server. At most 2,000 bytes each. |
| `libraries` | list of strings | no | Kinds of library that need it, at most 64. Only the kinds a signed catalogue lists for a module count. |
| `requires` | list of strings | no | Module IDs it is built on, installed first; at most 64. |
| `core` | bool | no | Other modules can't do without it, so it can't be taken off a server. |
| `pages` | list | no | Pages its web part draws in the web app (API 3's way): at most 16, only with a part in the browser. Each: `id` (`^[a-z][a-z0-9-]{0,63}$`, unique), `title` (one line of plain text, 1 to 64 bytes), `where` (`main` or `settings`), `icon` (a word, optional), `admin` (bool). |
| `status` | string | no | `stable` (the same as leaving it out), `beta`, `experimental` or `incomplete` ([section 4](#4-status-saying-a-version-is-unfinished)). |
| `statusNote` | string | for `experimental`, `incomplete` | One line of plain text, at most 500 characters, with no control characters: what isn't done or may break. |

"Plain text" here and below means valid UTF-8 with no control characters (nothing below U+0020, and
not U+007F); "one line" also means no line feed or tab.

## 4. Status: saying a version is unfinished

A version says how finished it is with `status`:

| Status | Means | What a server does |
|---|---|---|
| `stable`, or none | Nothing known to be missing | Installs it as usual |
| `beta` | Complete, still being tried out | Installs it as usual, and shows the status and any note |
| `experimental` | Works in parts; may change or break | Shows the note, and installs it only with an [override](#22-overrides) of kind `incomplete` |
| `incomplete` | Parts are missing, which the note names | The same |

`statusNote` is required for `experimental` and `incomplete`, because it is what the administrator
reads before agreeing; it is allowed, and shown, for the others. Write it for an administrator:
what doesn't work yet, what may break, and what is safe.

The status is the release's own: a server goes by the signed manifest it downloaded, never by what
a catalogue says. A catalogue entry carries a copy ([section 18](#18-the-catalogue)) so a store can
show it before anything is downloaded. A server older than module API 4 ignores both fields.

`wittymodule check` passes an `experimental` or `incomplete` module and prints a warning naming its
status and note. `Manifest.Unfinished()` says whether a version is one of those two, and
`StatusOrStable()` gives `stable` for a version that says nothing.

## 5. Delivery kinds

| `delivery` | A release holds | How it runs | `sdui.json` from API 4 |
|---|---|---|---|
| `helper` | An archive per system: the program `entry`, any `programs`, an optional part in the browser under `web/`, `module.json`, `sdui.json`, `LICENSE`, `NOTICE`, `THIRD-PARTY-NOTICES.txt`, `README.md` if there is one | A separate process the server starts on first use and talks to over pipes ([section 8](#8-the-helper-protocol)). It may also be a [service](#9-services). The only kind that may answer SDUI data sources | Required |
| `program` | The same as a helper | The server starts it for one job, in that job's own protocol. It must also answer the hello when started as a helper, so an install can check it runs | Required |
| `web` | One archive for every system: the built bundle (`entry`, `style`, and what they load), `module.json`, `sdui.json`, `LICENSE`, `NOTICE` | The browser loads it when a page first needs it, under the page's content security policy ([section 10](#10-web-parts)) | Required |
| `library` | Nothing to install | Go code linked into other modules or the server; reaches a server inside them. A catalogue can't list a version of one | Not read |
| `builtin` | Nothing to install | Part of the server, updated with it. Not released on its own | Not read |

A web bundle may hold only `.js`, `.mjs`, `.css`, `.wasm`, `.json`, `.svg`, `.png`, `.woff2` and
`.map` files; `wittymodule pack -web` adds a gzip copy (`.gz`) beside each script or stylesheet larger
than 1 KiB, and refuses links and special files. A helper's part in the browser follows the same
rules inside `web/`.

## 6. Compatibility

A version fits a server when all of these hold (`Manifest.Supports`, with the server's reading of
`maxServer` below):

1. **The server's version can be compared.** A server built from a commit with no tag installs
   nothing, and says so.
2. **`minServer` ≤ the server**, in [version order](#2-identity-and-naming). A pre-release comes
   before its release, so a module that works on `0.8.0-alpha` says `minServer: 0.8.0-alpha`, not
   `0.8.0`.
3. **The server ≤ `maxServer`.** An exact version means exactly that, pre-releases of it included
   (`0.8.0-rc1` ≤ `0.8.0`, `0.8.1` is past it). A series `A.B.x` covers every release and
   pre-release whose first two numbers are `A.B`; `A.x` covers every one whose first is `A`.
4. **`MinAPI` ≤ `api` ≤ `API`**, the server's own numbers: 3 to 4 on 0.8.0-alpha. An older `api`
   is refused with words saying it was built before modules became separate from the server; a
   newer one with words saying it needs a newer server.

**How a server reads `0.N.x`.** `wittymodule` reads a range exactly as above. A server reads any
pre-1.0 series `0.N.x` as `0.x` (`internal/modules/widen.go`), because before 1.0 the module API
number decides whether a version fits: a module released beside 0.6 for API 3, with `maxServer:
0.6.x`, runs on 0.8.0-alpha, which takes API 3 and 4, and a store offers it there. An exact
`maxServer` is kept as written, and a 1.x server is another line. So set `maxServer` to the series
you tested with; on a pre-1.0 server it is advice, and the API is the rule.

**Pre-release modules** are offered only to a server that is itself a pre-release, or that already
runs that module's pre-release. An administrator may still choose one from the list of versions,
after a warning. A version with a pre-release label counts as one whatever the catalogue says.

**When a server leaves a module behind** (it was upgraded past a version's range, or that version
was refused), nothing breaks: at start it uses the chosen version, else the previous installed
version, else goes without the module, whose pages say so and offer a version that fits. A version
out of range stays on disk in case the server is rolled back. An administrator may run a version
whose range doesn't cover the server, within the API range, with an [override](#22-overrides) of kind
`out-of-range`.

## 7. Permissions and limits

A helper or program runs boxed in. Its manifest's `permissions` ask for what the box leaves open; a
manifest can't grant itself anything. A server has its own list of which modules may have each
(`mayHave` in `internal/modules/service.go`), and a permission is given only when the manifest asks
for it **and** the server's list allows it for that module.

| Permission | Opens | Official modules a server allows it today |
|---|---|---|
| `network` | Sockets | WittyTuner, WittyPlay, WittyCouch, WittyRigs |
| `programs` | Starting other programs, such as FFmpeg or its own `programs`; `PATH` in its environment; no memory limit | WittyTuner, WittyPlay, WittyCouch, WittyRigs, WittyBluray |
| `devices` | This machine's devices: capture cards, tuners, graphics | WittyTuner, WittyPlay |
| `data` | Writing in a folder of its own, named in `WITTYMODULE_DATA`, and those in `WITTYMODULE_WRITE` | WittyBluray, WittyPlay, WittyCouch, WittyRigs |

A fork, a module from another store, and a module installed with a trust override (`unsigned`,
`untrusted-key`, `custom`, `weak-sandbox`) get **none** of these, except what an administrator ticks
explicitly when agreeing to the override ([section 22](#22-overrides)). A permission a server doesn't
know means the version can't run there, and it is shown with that reason.

Every program runs with a small environment (`GOMEMLIMIT` of 256 MiB, `GOMAXPROCS` of 2), reads
wherever media is, and is boxed in by what each system offers: on Linux a launcher with no new
privileges, an address-space limit, Landlock for writes, a seccomp filter on 64-bit systems and
namespaces where allowed; on Windows a low-integrity token and a job with a 512 MB limit; on macOS
`sandbox-exec` with a profile written by the server; on FreeBSD resource limits an ordinary account
can give up for good, and a jail as root. A module asking for `programs`, `devices` or `data` can't
have namespaces or Windows' low integrity. The server reports the result as `sandbox.level`: `full`,
`partial`, `limited` or `unconfined`. WittyMediaServer's `docs/MODULE-STORE.md`, "What limits a
module", has the table for each system; the server's code is the authority.

Rules every module follows whatever it is given: no reaching into the server except through this
contract; heavy parts load only when used; untrusted files (books, PDFs, disc menus) stay boxed in,
with size and time limits; **encrypted media stays refused**, with no extension point for it; and
everything from outside, including what the server sends, is treated as untrusted.

## 8. The helper protocol

A helper (and a program, for its hello) speaks this over its standard input and output.
`wittymodule.Main`, `Serve`, `ServeStream` and `Notify` are the helper's side; `Conn` is the server's.

**Starting.** The server starts the program with the one argument `--wms-helper`
(`wittymodule.HelperFlag`), from its version's folder, with the environment of
[section 7](#7-permissions-and-limits) and these variables:

| Variable | Holds |
|---|---|
| `WITTYMODULE_LOG` | `debug`, `info`, `warn`, `error` or `off` |
| `WITTYMODULE_LOG_TOPICS` | The log topics that are on, comma-separated |
| `WITTYMODULE_SETTINGS` | The module's server settings with defaults, secrets included, as a JSON object (API 4, [section 12](#12-settings)) |
| `WITTYMODULE_DATA` | Its own folder to write in, with `data` |
| `WITTYMODULE_WRITE` | Other folders it may write in, with `data` |

Started any other way, `wittymodule.Main` explains what it is and exits with status 2.

**Framing.** Each message is one line of JSON ending in `\n`, at most 1 MiB (`MaxLine`) including
the line feed. An answer may be followed by a binary blob of exactly the size it announces, at most
128 MiB (`MaxBlob`). Both sides check sizes before setting memory aside, and treat the other as
untrusted.

**Hello.** The helper's first line names itself:

```
helper → {"module":"wittyphotos","version":"0.4.0","api":4}
```

`module`, `version` and `api` must equal the manifest's, or an install fails and a start is refused.
A helper has 45 seconds to say it (a virus scanner may hold a new program the first time); one that
was still starting gets a second start with twice as long before it counts as failing.

**Calls.** The server sends one call at a time and waits for its answer:

```
server → {"id":1,"method":"read","params":{"path":"..."}}
helper → {"id":1,"result":{...}}
helper → {"id":2,"result":{},"blob":5120}<5120 bytes>
helper → {"id":3,"error":"this file has no preview","code":"no-preview"}
```

| Field | In | Meaning |
|---|---|---|
| `id` | call, answer | A number the server picks; the answer carries the same one. A different one is a failure |
| `method`, `params` | call | What is asked, with any JSON parameters |
| `result` | answer | Any JSON; may be left out |
| `blob` | answer | The size of the bytes after this line; 0 or left out for none |
| `error`, `code` | answer | The helper answered no. `code` is a word the caller may act on, and may be empty |

An answer with `error` is a **call error** (`*wittymodule.CallError`): the helper is fine and said
no, such as for a file it can't read. It never counts against the version. Anything else that goes
wrong — the process ending, an unreadable line, a size out of range, a wrong `id`, no answer in time
— is a **failure**. A panic in a `Serve` handler becomes a call error with code `panic`. A method the
helper doesn't know should be answered with a call error (a service answers `unknown-method`).

**Notifications** (API 3). A helper may say things unasked, at any time while it is serving, from
any goroutine (`wittymodule.Notify`). A notification has `notify` and no `id`, and nothing answers it:

```
helper → {"notify":"task","params":{"key":"scan","title":"Reading photos","state":"running","done":40,"total":120}}
```

| `notify` | `params` |
|---|---|
| `task` | `TaskReport`: `key` (`^[a-z0-9][a-z0-9._-]{0,63}$`, stable for the task), `title` (one line, 1 to 200 bytes), `state` (`running`, `done`, `failed`), `done` and `total` (0 or more, `done` ≤ `total` when `total` is set), `item` (one line, at most 200 bytes). Reports replace each other; send them as things change, a few a second at most |
| `diag.step` | `{"phase": "...", "fraction": 0..1}`: what a disc check is doing |
| `diag.progress` | A number from 0 to 1 |
| `diag.label` | The disc's name |
| `diag.finding` | `DiagFinding`: `severity` (`ok`, `info`, `warn`, `problem`), `title`, and what it found, in words about the disc's structure, never its content |

A server checks every notification, may ignore any, and may drop them for arriving too fast. A task
is named by its module as well as its key, so a module only ever speaks about its own. Between calls
a server reads the pipe only for a helper it keeps running in the background; otherwise
notifications arrive during calls.

**Streams.** A process may answer one call with a stream (`ServeStream`): it says hello, reads one
call, writes the answer line, then writes the stream's bytes until it ends or the server closes
standard input. Nothing else can be said on that pipe, and the process exits when the stream ends.
`Notify` is unavailable there.

**Ending.** A helper exits when its standard input closes, so it can't outlive the server. Its
standard error is kept, bounded, and logged when it fails; print one thing per line there, never
tokens, passwords or file contents.

**Time and failure.** A call has 60 seconds by default. A helper idle for a minute is stopped, and
started again on the next call. A job that keeps state, such as an open disc, has a helper of its own
for as long as it lasts. Three failures in a row block the version, with the reason: the previous
version takes over, or the module goes without. A blocked version isn't installed automatically
again; an administrator may still choose it.

## 9. Services

A service is a helper for what one call at a time down a pipe can't do: calls that overlap, and
connections a browser makes straight to it. Its manifest says `delivery: helper`. The package
`wittymodule/service` is both ends.

1. It starts as a helper and says hello.
2. The server calls `service` (`service.MethodStart`). It answers `{"url": "http://127.0.0.1:<port>",
   "token": "<64 hex digits>"}`: an address on this machine only, and a secret made for this start.
3. Every request from the server carries the token in `X-Witty-Service-Token`; a request without it
   is refused with 403.
4. A call is `POST /call/{name}` with `{"recv": <receiver>, "args": [<arguments>]}`, at most 64 MiB,
   answered with `{"rets": [...], "back": <receiver if changed>, "err": "...", "code": "...",
   "errType": "...", "errData": ...}`. An unknown name answers `code: "unknown-method"`. Values
   cross by their Go field names, including those hidden from JSON (`service.Marshal`).
5. A module may add routes of its own (`Config.Routes`), such as a stream a browser connects to,
   reached through the server's proxy under `/api/modules/<id>/...`. The server passes on who is
   asking in `X-Witty-User` (the account's ID) and `X-Witty-Admin` (`1` for an administrator), after
   removing every `X-Witty-` header, `Authorization` and its own sign-in cookie that a browser sent.
   A request with no `X-Witty-User` didn't come from a signed-in person and should be refused.
6. It answers `service.busy` with whether it has work going on (`Config.Busy`). It is stopped about
   five minutes after its last use unless busy, and one that dies is started again on the next call.
7. From API 4 it answers the SDUI calls of [section 11](#11-the-sdui-contribution) on the helper
   protocol through `Config.UI`.

A service is held to a helper's limits, except where its permissions and the server's list allow
more.

## 10. Web parts

A web bundle, or a helper's part in the browser, runs in the browser with the access of whoever is
signed in, under the page's content security policy. It takes the app's React and shared parts from
`globalThis.__wmsModuleHost` and must not bring its own copy of React. Adding to that object raises
the module API; removing anything from it breaks bundles already released.

A bundle that fails to load or crashes is reported to the server. Three reports in a day from an
administrator's browser take that version out of use; anyone else's go in the module's log.

API 3's manifest `pages` are drawn by the bundle at `/m/<module>/<page>`, in the web app only. Each
page component gets `host`: `request(method, path, body?)` to this server's API with the viewer's
access (any other address is refused), `user` (`{admin}`), `store` (`get`, `set`: at most 64 KB a key
and 32 keys, per person and module), and `token()` and `base()` for a stream. `admin: true` only hides
a page; what it calls must check for itself. From API 4 a module should describe its pages in
`sdui.json` instead, which every app draws, and use a `NativeView` where only its web part can draw
something.

## 11. The SDUI contribution

From module API 4, **every module a server installs** (`helper`, `program`, `web`) **must carry
`sdui.json`**: what it adds to every app, as data the server checks against its catalogue and every
app draws with its own components. A version built for API 4 without one is refused, and one whose
file fails the server's checks is refused, unless an administrator agrees to run it without its SDUI
([section 22](#22-overrides), kind `incomplete`). The full format is WittyMediaServer's
**`docs/SDUI-MODULES.md`**; the code is `pkg/sdui/module.go` there, and WittyModule's `sdui` package
carries a copy of the server's validator (`sdui/internal/core`, kept current with
`kit/sync-sdui-core.sh`), so `sdui.Check` checks exactly as a server does.

**Where it is.** Beside `module.json`: at the root of the repository and of every archive.
`wittymodule pack` puts it in each one, and it is covered by the checksum list like every file.

**The smallest file**, for a module with nothing to add:

```json
{ "format": "wittymodule-sdui", "formatVersion": 1, "module": "wittyx", "catalog": 2,
  "theme": { "scope": "wittyx" } }
```

**The rules, in summary:**

- At most 1 MiB; one JSON object with nothing after it; **an unknown field anywhere is refused**.
- `format` is `wittymodule-sdui`, `formatVersion` 1, `module` the manifest's `id`, `catalog` the SDUI
  catalogue revision it was written against (1 to the server's; 2 today; a newer one is refused).
- `theme` is required ([section 13](#13-theme-scope)). Everything else is optional: `sources`,
  `components`, `views`, `pages`, `inserts`, `presets`, `navigation`, `settings`.
- Every name begins with the module's ID ([section 2](#2-identity-and-naming)), and a module uses only
  its own components, sources and views, and the catalogue's.
- **Only a helper may declare sources**, since only a helper can answer them.
- Pages are `custom`, `library`, `detail`, `livetv`, `music`, `photos`, `books` or `games`, never
  `home`, `search`, `libraries` or `collections`; a `library` or `detail` page narrows by
  `match.libraryKind` or `match.mediaKind`, never `match.library`. An administrator's page always
  wins over a module's, and a module's over a built-in one.
- Inserts go only into built-in pages, never an administrator's. Menu entries go just before
  Settings. A module never adds anything to Settings except by describing its settings.
- It is data, never code: no HTML, scripts or addresses of its own; pictures only from this server.

| Limit | At most |
|---|---|
| Pages | 32 |
| Sources, components, presets | 64 each |
| Views, inserts | 32 each |
| Menu entries | 16 (the menu, 40 in all) |
| Settings pages | 16, each with 1 to 64 rows |
| A label | 80 characters (the theme's, 60; a menu entry's, 40) |
| A description, help or footer | 500 characters |

**When it is checked.** At install, before the version is switched to; and every time a version is
picked for use (at start, on a switch, when the module is turned on). A failing file at install
fails the install; at use, that version isn't used and the one before takes over.

**The calls a server makes for it**, on the helper protocol (`sdui.Answers` answers them; a helper
that declares no sources, pictures or settings needn't):

| Method | Params | Answer |
|---|---|---|
| `sdui.items` | `ItemsRequest`: `source` (the section's source, variables filled in), `filter` (the page's filter and sort, when any), `offset`, `limit` (at most 500, and never past the source's own), `viewer` (`profileKind` `adult`/`kids`/`guest`, `admin`, `cannot`, `formFactor`, `platform`, `locale`), `settings` (the module's settings for this profile, with defaults, when it has any) | `ItemsAnswer`: `items` (a list of `sdui.Item`, never null), `more`, `total`. A call error puts the section in its error state, with Retry |
| `sdui.image` | `{"name": "<name>"}`, a name matching a letter or digit then up to 127 letters, digits, `.`, `_`, `-`, with no `..` | The picture as the call's blob (`result` may be `{}` or null): PNG, JPEG, WebP or GIF by content, at most 4 MiB. A call error is a 404 to the app; anything that isn't a picture, a 502 |
| `settings.changed` | `{"values": {...}}`: the server settings after an administrator saved them | `{}`. Only sent to a running helper; one that doesn't know it may answer a call error and read `WITTYMODULE_SETTINGS` when it next starts |

The server trusts nothing in an item: it keeps an item only with an `id` (at most 200 bytes), a
`kind` the source's `yields` lists and a non-blank `title`; it drops items the profile may not see;
keeps routes only within the app, pictures only on this server under `/api/`, and `playId` and
`nextId` only for library items the profile may play. A module's own pictures are served from
`/api/v1/sdui/modules/<id>/images/<name>` (`sdui.ImagePath`).

**Writing it.** In Go: `sdui.New`, the `Add…` helpers, `sdui.WriteFile` and a golden test with
`sdui/sduitest`. For a web bundle: `kit/sdui/` (copied into the module at `sdui/kit/` by
`kit/module-kit.sh`): `wittymodule-sdui.ts`, `write-sdui.mjs` and the JSON Schema. Either way,
`wittymodule check` checks it.

## 12. Settings

A module **describes** its settings in `sdui.json` (`settings`) and never draws them: every app's own
Settings shows them under Settings → Modules → the module, in that app's own style. Settings is never
an SDUI page.

- A settings page has `id`, `title` (≤ 80 characters), `icon`, `admin` and `groups`; a group has
  `title` (≤ 80), `footer` (≤ 500) and `rows`.
- A row has `key` (`^[a-z][A-Za-z0-9_.-]{0,63}$`, unique across the module's settings; none on an
  `info` row), `kind` (`bool`, `enum`, `int`, `string`, `secret`, `info`), `label` (≤ 80), `help`
  (≤ 500), `scope` (`server`, the default, or `profile`), `default` (checked like a value; never on
  `secret` or `info`), `options` (`{value, label}`, only and always on `enum`), `min` and `max` (on
  `int`), `unit` (≤ 20) and `maxLength` (on `string` and `secret`, 0 to 4,000; 0 means 1,000).
- A `string` or `secret` value is one line with no NUL, escape, carriage return or line feed. A
  `profile` row is never `secret` and never on an `admin` page.
- `server` values are set by administrators and kept by the server; `profile` values by each profile
  for itself. Secrets are never sent to an app.
- The helper is told its `server` values in `WITTYMODULE_SETTINGS` when it starts, through
  `settings.changed` when they change, and with this profile's values in every `sdui.items`
  (`sdui.Settings()` reads the first).

## 13. Theme scope

`theme` in `sdui.json` is required. Its `scope` is always the module's ID; `label` (≤ 60 characters)
names the scope in the theme editor and the module in Settings → Modules; `tokens` maps colour tokens
(`sdui.ColorTokens`: `accent`, `bg`, `fg`, …) to colours (`#rgb`, `#rrggbb`, `#rrggbbaa`, `rgb()`,
`rgba()`). A module's tokens are the bottom layer: an administrator's overrides for the scope replace
them token by token. A module's pages, inserted sections and items are in its scope unless they say
otherwise.

## 14. Data formats

A version that keeps data of its own says so in its signed manifest: `dataFormat`, the newest format
it writes, and `dataFrom`, the oldest it can read. The server remembers the newest format used so far.
A version that can't read what is there is refused, with the reason, both when chosen and at every
start, and no override lets it run. Zero, or leaving them out, means it keeps none of its own. Today
no module declares a format; the first that keeps data of its own must.

## 15. Logging and capabilities

A program logs to standard error, one thing per line, at the level in `WITTYMODULE_LOG` and for the
topics in `WITTYMODULE_LOG_TOPICS`. It offers topics in its manifest:

```json
"capabilities": { "logTopics": ["scan|Scanning|Each file looked at", "cache|Cache|Hits and misses"] }
```

Each topic is `name|label|description`. Other capabilities a server reads today: `rawExtensions`
(WittyPhotos: the file extensions it reads RAW previews from). A module needing file types should
read the server's `GET /api/media-types` (or `docs/media-types.json` when it is built) rather than
keep a list of its own.

## 16. Releases

A release of version `V` is a GitHub release tagged `vV` (a tag with a suffix is a pre-release) whose
assets are:

```
<id>-V-SHA256SUMS.txt       the checksum list
<id>-V-SHA256SUMS.txt.sig   its signature
<id>-V-module.json          the manifest, byte for byte the one in the archives
<id>-V-<goos>-<arch>.tar.gz an archive per system (.zip on Windows), or
<id>-V-web.tar.gz           a web bundle's one archive
```

**Archives.** Each holds exactly one top folder ([section 2](#2-identity-and-naming)) and regular
files only: no links, devices, absolute paths or paths that climb out. Programs have mode 0755, other
files 0644. `wittymodule pack` writes every file with the same date, so the same code packs to the
same bytes. Systems are `windows/amd64`, `windows/arm64`, `darwin/amd64`, `darwin/arm64`,
`linux/amd64`, `linux/arm64`, `linux/arm` (armv7), `linux/386`, `freebsd/amd64` and `freebsd/arm64`;
a release needn't have all of them (a program built with something other than Go may not), and a
server with no build for its own system says so.

**The checksum list** has one line per file, sorted by name: the SHA-256 in lower-case hex, two
spaces, the name. It names the manifest copy, every archive, and **every file inside every archive**
as `<top folder>/<path>`. Nothing else may be named: `wittymodule sign -expect <id>-V` refuses a list
naming anything that isn't a file of that module and version.

**The signature** is the base64 of an Ed25519 signature over the list's exact bytes, followed by a
line feed. Official modules are signed with the project's one **module key**, which a server trusts
for modules and never for a server; a fork or another store's module is signed with its publisher's
own key ([sections 19](#19-stores-other-than-this-projects) and [21](#21-forks)). The signing job
should run on a clean machine that sees only the list and the key, behind an approval.

**A repository** carries `README.md` ending in a Legal section, `LICENSE` (AGPL-3.0-or-later for this
project's modules), `NOTICE` crediting `<name>, developed by WittyWilla`, `module.json` and, from API
4, `sdui.json`. `wittymodule pack` writes `THIRD-PARTY-NOTICES.txt` into each archive from the
licences of every library its programs are built from, and stops if one has none. `kit/module-kit.sh`
writes the workflows (CI, release, withdraw) that do all of this.

**Versions** go up in small steps (`wittymodule bump`), never reuse a tag, and a version never changes
once released. CI's `wittymodule needs-bump` fails when code changed since the last tag and the
version didn't.

## 17. What a server does with a release

**Install**, each step leaving the running version untouched if it fails:

1. Verify the checksum list's signature with a key trusted for this module, and that the manifest
   copy matches the list, is for this module, version and repository, and agrees with what the
   catalogue said (`CatalogVersion.MatchesManifest`: `delivery`, `api`, `minServer`, `maxServer`,
   `license` and `permissions` must be equal).
2. Download the archive for this system, hashing as it arrives, stopping at its listed size (at most
   256 MiB).
3. Unpack it to a work folder, refusing links, devices, absolute paths and climbing paths.
4. Check every file against the list, refusing any extra or missing file; then read `sdui.json` and
   check it against the server's SDUI catalogue (required from API 4).
5. Start the program once with `--wms-helper`; its hello must name the same module, version and API.
6. Move the folder to `versions/V` in the module's data folder, and save the state atomically.
7. Switch over: the next call uses the new version, with no restart.

Everything is under `<data>/modules/`, never the program folder. The checksum list stays beside the
files and is checked again every time a program starts, so a file changed afterwards is refused. The
active, previous and pinned versions are kept.

**Updates.** A server checks for new versions (`moduleUpdateCheck`, twice a day when allowed) and
installs what fits only when `moduleAutoInstall` is on. Forks, modules from other stores, pinned and
overridden modules never update by themselves.

## 18. The catalogue

A catalogue is one signed JSON file, `catalog.json`, listing modules and every version of each, with
what a store shows and where each version's files are. This project's is WittyModuleCatalog,
published on its release tagged `catalog`. `wittymodule.Catalog` reads and checks one;
`wittymodule catalog-add`, `catalog-check` and `catalog-sign` write them.

**A catalogue only offers.** An install still checks the release's own signed list, manifest and every
file, and that the catalogue's description matches the manifest, so a catalogue's key alone can't
make anything run.

The file: `format` (int, 1; a higher one is refused in words and the last good catalogue kept),
`name` (one line, 1 to 200 bytes), `serial` (1 or more; only ever goes up, and a server keeps the
highest it has seen, so an older file can't be replayed), `generated` (RFC 3339), `modules` (at most
1,000, IDs unique).

**A module:**

| Field | Type | Rules |
|---|---|---|
| `id` | string | A module ID |
| `name` | string | One line, 1 to 64 bytes |
| `summary` | string | One line, at most 2,000 bytes |
| `description` | string | Plain text with line breaks, at most 16 KiB |
| `icon` | file | Optional. A PNG of at most 256 KiB, never a brand's logo |
| `screenshots` | list of files | Optional. At most 8 PNG or JPEG pictures of at most 2 MiB each |
| `publisher` | object | `name` (one line, 1 to 200 bytes), `url` (https, optional), `key` (base64 Ed25519 public key its releases are signed with when not the catalogue's own, optional) |
| `repo` | string | `owner/name` |
| `group` | string | Optional, a word |
| `libraries`, `requires` | lists | Words, at most 64 each |
| `core` | bool | |
| `needs` | list | One line each, at most 2,000 bytes, at most 64 |
| `updates`, `stays` | string | Plain text, at most 2,000 bytes |
| `versions` | list | Newest first, at most 100, unique; may be empty before a first release |

The module's own words come from its newest version's manifest when a version is added; words a
version leaves out stay as they were.

**A version:**

| Field | Type | Rules |
|---|---|---|
| `version` | string | A version |
| `delivery` | string | `helper`, `program` or `web` (a library or builtin has nothing to list) |
| `api`, `minServer`, `maxServer` | | As in the manifest |
| `license` | string | One line, 1 to 200 bytes |
| `permissions` | list | Words, at most 64. One a server doesn't know means the version can't run there |
| `released` | string | `YYYY-MM-DD` |
| `prerelease` | bool | Also true of any version with a pre-release label |
| `changelog` | string | Plain text with line breaks, at most 16 KiB; may be empty |
| `sums`, `sig`, `manifest` | string | Where the release's files are |
| `platforms` | object | `GOOS-GOARCH` (`linux-arm` is armv7), or `all` for a web bundle → file; 1 to 32 |
| `withdrawn` | object | Optional: `severity` (`broken` or `security`) and `reason` (plain text, 1 to 2,000 bytes) |
| `status` | string | Optional, as in the manifest ([section 4](#4-status-saying-a-version-is-unfinished)) |
| `statusNote` | string | As in the manifest: required for `experimental` and `incomplete`, one line, at most 500 characters |
| `ui` | object | Optional: what the version's `sdui.json` adds, below |

`delivery`, `api`, `minServer`, `maxServer`, `license` and `permissions` must equal the release's
signed manifest, or an install is refused. `status`, `statusNote` and `ui` are copies for the store:
a server goes by the release itself and never installs or refuses because of them.

**`ui`**, written by `wittymodule catalog-add` from the `sdui.json` inside the signed archive
(`sdui.Summary`):

| Field | Type | Rules |
|---|---|---|
| `pages` | list of `{id, title}` | At most 32. `id` an SDUI page ID (`^[a-z][a-z0-9_-]{0,63}$`, unique), `title` one line of 1 to 120 characters |
| `navigation` | list of strings | The menu entries' labels (a page entry without one takes its page's title). At most 16, each one line of 1 to 40 characters |
| `settings` | bool | It describes settings |
| `sources` | int | Its data sources, 0 to 64 |
| `inserts` | int | Its sections in built-in pages, 0 to 32 |

**A file** is `{url, size, sha256}`: `url` is `https`, or a relative path below the catalogue's own
folder (no `http`, `file`, `..`, query, fragment, user or backslash; at most 2,048 bytes); `size` 1 to
the limit for its kind (256 MiB for an archive); `sha256` lower-case hex.

**The signature** is the same as a release's, in `catalog.json.sig`. `catalog-sign` also writes
`catalog.json.key`, the public key a store suggests the first time an administrator adds it.

**A server**: checks the signature before reading anything (`VerifyCatalog`), then the serial; keeps
the last good file and its signature and checks them again when read back; reads at most 8 MiB;
offers a version only if it fits ([section 6](#6-compatibility)) and shows the reason on one that
doesn't; fetches one file at a time with its limit in force; and never lets a page fetch from an
address in a catalogue. Text is always shown as text, never as HTML or Markdown.

**Adding a version.** `wittymodule catalog-add -catalog catalog.json [-base url] [-changelog file]
[-pub key] [-publisher name] <release folder>` checks the release's signature (with `-pub`), the
manifest and every archive against the signed list, reads `sdui.json` from the first archive that has
it (checked against the signed list, the way a server checks it), and refuses a version built for API
4 with none or one a server would refuse. It fills `status` and `statusNote` from the manifest and
`ui` from `sdui.json`. A version already listed must have the same files; adding it again changes
nothing, and the serial goes up only when something changed.

## 19. Stores other than this project's

A store is a catalogue at an `https` address, signed with the store's own key. To run one, put
`catalog.json` and the release files in a folder and run `wittymodule catalog-sign -key <file>
catalog.json`; give people the address and, separately, the public key.

What a server does with one:

- An administrator adds it after a warning, with its key pinned at that moment (pasted, or offered in
  `catalog.json.key` for the administrator to check). From then on a different key is a refusal.
- Its signature is checked against the pinned key before anything is read. Its serial only goes up,
  and any change needs a higher serial: the same serial with different contents is refused.
- Its modules are held to the rules for forks: an administrator chooses each version, nothing updates
  by itself, no permissions, the strict sandbox. Releases must be signed with the store's key or the
  `publisher.key` it gives.
- An `id` this project's catalogue also lists is a fork of that module; a store can't replace a module
  by listing it, and WittyTuner can't be replaced at all. An `id` nobody knows is a custom module,
  shown as in development and installable only through the `custom` override.
- Its addresses are fetched like any typed address: no private, local or link-local addresses unless
  the administrator allowed them for this store, no redirects that break that. Sizes are enforced
  while reading: 8 MiB for a catalogue, 4 KiB for a signature, 256 KiB for an icon, 2 MiB for a
  screenshot.
- Characters that reorder or hide text are removed from what it says, and a module using the name of
  one of this project's modules is called out.

## 20. Withdrawals

A release can't be taken back from everyone who saw it, and GitHub isn't trusted to say which are bad.
So a module publishes a signed list, `<id>-withdrawn.json` and `.sig`, as assets of a release tagged
`withdrawals` (not a version, so nothing installs it):

```json
{ "module": "wittyphotos", "serial": 3,
  "versions": [ { "version": "0.3.1", "severity": "security", "reason": "Reads past the end of some RAW files." } ] }
```

- `module` is inside what is signed, so one module's list can't be used for another's.
- `serial` is 1 or more and only goes up; a server keeps the highest seen, so neither an old list nor
  hiding the list brings a version back. A newer list may restore one.
- At most 1,000 entries, each a valid version, a severity and a reason of at most 2,000 bytes.

| Severity | Choosing it | Where it is in use | Automatic updates |
|---|---|---|---|
| `broken` | A warning | Stays, with a warning | Never picked |
| `security` | Refused; no override | Goes out of use at once, even if pinned or overridden | Never picked |

`wittymodule withdraw` and `restore` edit the list (with `-list-key`, only a list carrying that key's
signature), and the module's **Withdraw a version** workflow signs and publishes it. Set `withdrawn` on
the version in the catalogue too.

## 21. Forks

A fork is someone's own build of a module this server knows, with the same `id` and their own `repo`
and key (`wittymodule keygen`). Choosing one replaces that module on a server, and everything else
treats it as that module.

- Its key is trusted for that one module, and it never updates by itself.
- It gets **no permissions** and the **strict sandbox**: on Linux no sockets and no changing what's in
  files, and it doesn't run where that can't be applied (Landlock ABI 3, kernel 6.2, 64-bit).
- No forks where nothing can box them in: a server refuses them on systems without a sandbox for other
  publishers, and whenever `WMS_NO_MODULE_FORKS=1` is set, from the form or a store.
- **WittyTuner can't be replaced**: nothing outside it could tell whether a fork still refused
  encrypted channels.

## 22. Overrides

An administrator may install a version a server would otherwise refuse, for a reason the server can
name, after a warning that grows with the risk. This is the server's decision and its code is the
authority (WittyMediaServer `internal/modules`, behind the feature `module-overrides`; an older server
simply refuses as before). This section is the contract a module and a store can rely on.

**Kinds**, each with a plain warning, by severity:

| Kind | When | Severity |
|---|---|---|
| `incomplete` | `status` is `experimental` or `incomplete`; or an API-4 version has no `sdui.json`, or one that fails the server's catalogue (the module then runs **without** its SDUI) | 1 |
| `out-of-range` | This server is outside `minServer`..`maxServer` while the API is within the server's range ("compatibility mode": the range is ignored) | 2 |
| `local-file` | Installed from an uploaded file rather than a catalogue | 2 |
| `custom` | A module ID this server doesn't know | 3 |
| `untrusted-key` | Validly signed, by a key not trusted for this module (its fingerprint is shown) | 3 |
| `weak-sandbox` | Another publisher's module on macOS, where `sandbox-exec` is the only box: it must start inside it, strictly, with no fallback | 3 |
| `unsigned` | No signature at all | 4 |

**Never overridable**, refused with the reason: a module API outside the server's range (3 to 4);
a delivery other than the one this server expects for the module; a protocol mismatch (a hello that
doesn't match); a data format that would lose data; a version withdrawn for security; no build for
this system; a fork of WittyTuner; `WMS_NO_MODULE_FORKS=1` for anything from another publisher; a
system with no sandbox for other publishers (FreeBSD, and others); a sandbox that won't start.

**Friction.** Every override needs a confirmation naming each bypassed kind with its warning, the
administrator's password (and two-factor code) again, and, at severity 3 or more, the module's ID
typed. `accept` must name exactly the kinds needed. A version whose need is only known after download
(its status, its `sdui.json`) is downloaded, verified and held, and the install ends asking for the
override; agreeing installs the held copy without downloading again.

**Afterwards.** The override is recorded on the module (kinds, reasons, who, when, version, key
fingerprint, granted permissions, and whether it came from a catalogue, an upload or a store), shown
as **Overridden** with its reasons in the store and on the Modules page, and written to the module
audit log. An overridden module is pinned and never updates by itself; a manual update to a version
that needs no override clears it, and one that needs an override asks again. Removing an override
goes back to the last version that needed none, or takes the module off the server.

**Permissions.** A module installed with `unsigned`, `untrusted-key`, `custom` or `weak-sandbox` gets
no permissions and the strict sandbox, like a fork, except permissions the administrator explicitly
ticks (`grants`), each one its manifest asks for, and never under `WMS_NO_MODULE_FORKS=1`.
`incomplete`, `out-of-range` and `local-file` on an otherwise trusted official module keep the
official rules. An override never widens anything silently.

**Unsigned files** are hashed at install and signed with the server's own local key, trusted only for
the versions its override records name, so a file changed afterwards is refused like any other.

**For a module author** this means: declare an unfinished version honestly with `status` and
`statusNote`, rather than relying on an administrator to find out; and know that an override is an
administrator's exception, not a way to ship. A store should copy `status`, `statusNote` and `ui`
into its entries (`catalog-add` does) so the warning can be shown before anything is downloaded.

The HTTP routes are WittyMediaServer's to document (`docs/MODULE-STORE.md`, `docs/openapi.json`).

## 23. Checking a module

`wittymodule check [-server version] <folder>...` checks each folder and lists **every** problem, one
line each, located by file and field or by file and JSON path:

```
module.json: maxServer: maxServer "0.y.x" is not a version or a series such as 0.5.x
module.json: permissions[1]: "root" is not a permission
sdui.json: /theme/label: a label is at most 60 characters (Shorten it.)
warning: module.json: status: this version says it is incomplete: No settings page yet. ...
```

It checks the manifest on its own (`Manifest.Problems`), that the folder is named for the module, that
`api` is one this WittyModule knows and `minServer` isn't older than `0.6.0-alpha`, that a helper's
`cmd/<entry>` exists (or `CMakeLists.txt` for a program built otherwise), that `package.json`'s
version matches, `sdui.json` exactly as a server checks it (required from API 4; data sources only on
a helper), and with `-server` that the version fits that server under the strict reading of
[section 6](#6-compatibility). The SDUI checker's warnings are printed and don't fail it, and nor does
an unfinished status, which is printed as a warning. `pack` runs the same checks first.

For a catalogue: `wittymodule catalog-check [-pub key] catalog.json`, and in WittyModuleCatalog
`scripts/check-api.mjs`, which checks what a server decides: the module API, an API-4 version's
`minServer`, and `status` with its note.

## 24. Module API 3

A version built for API 3 still runs on a server that takes API 4. It differs in these ways:

- **`sdui.json` is optional.** One that carries it anyway has it checked and used exactly as for API
  4; one that doesn't adds nothing to the apps through SDUI, and its manifest `pages` are drawn by its
  web part in the web app only.
- **No `incomplete` override for a missing `sdui.json`**, since none is required.
- `status` and `statusNote` mean the same in an API 3 manifest; a server older than 0.8.0-alpha
  ignores them.
- `minServer` is at least `0.6.0-alpha`; `maxServer` is read by a 0.8 server as described in
  [section 6](#6-compatibility), so an API 3 version with `0.6.x` fits 0.8.0-alpha.
- Everything else here — identity, the manifest, delivery, permissions, the helper protocol,
  services, releases, signing, the catalogue, withdrawals, forks and overrides — is the same.

API 1 and 2 versions are refused by every current server, with words saying to install a version made
for WittyMediaServer 0.6.0-alpha or newer.

## 25. Change log

**4.0** (module API 4, WittyMediaServer 0.8.0-alpha)

- First written specification, covering module API 4 and where API 3 differs.
- Module API 4: every installed module carries `sdui.json` ([section 11](#11-the-sdui-contribution)).
- `status` and `statusNote` in the manifest and in catalogue versions
  ([section 4](#4-status-saying-a-version-is-unfinished)).
- `ui` in catalogue versions, written by `catalog-add` from the release's `sdui.json`.
- The server's overrides ([section 22](#22-overrides)).
- `wittymodule check` lists every problem, located by file and field or path.
