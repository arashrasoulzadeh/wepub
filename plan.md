# wepub — Minimal EPUB Reader with Private Cloud Sync

A minimal, offline-first EPUB reader for phone and desktop that syncs library, reading position, annotations and settings with a self-hosted server.

## Decisions (from planning Q&A)

| Area | Decision |
|---|---|
| Client | Flutter: Android, iOS, macOS, Windows, Linux |
| Render engine | **Locked: Webview + foliate-js** (MIT, pin a commit; spiked on macOS). Priority is rendering fidelity: Persian/RTL, fixed-layout, image-heavy. epub.js is the fallback |
| Server | New, Go + SQLite (pure-Go driver), files on disk; Docker image built in CI only, no Docker on local dev |
| Process | TDD; one step = one commit = one push (see Working rules and Implementation checklist) |
| Users | Single user, token auth |
| Offline | Offline-first: local DB is the source of truth for the UI, sync in background |
| Sync scope | Book files, reading position, highlights and notes, bookmarks, shelves, settings |
| Conflict model | Op-log + CRDT (see Sync design) |
| Reading modes | Paginated and continuous scroll, switchable (global default, per-book override) |
| v1 extras | Metadata and covers (with online lookup), themes and typography, in-book search and TOC |
| Platform phasing | v1: Android, iOS, macOS. v1.1: Windows, Linux (after webview proven) |
| Engine fallback | If foliate-js fails on iOS/Android or real books: fall back to epub.js, then Readium, behind a thin bridge |

## Scope split

- **v1:** reader (both modes), library, import, server, full sync (files, position, annotations, bookmarks, shelves, settings), metadata + online lookup, themes/typography, search/TOC, dyslexia font, high contrast.
- **v1.1:** Windows + Linux clients, PDF/MOBI conversion, reading stats/goals/streaks, text-to-speech.

## Non-goals (v1)

- Multi-user / sharing, DRM-protected books, audio/video media overlays and scripted EPUB content, social features, a web client.

## Architecture

```
┌────────────── Flutter app ──────────────┐        ┌──────── Go server ────────┐
│ UI (library, reader, settings)          │        │ HTTP API (REST + SSE)     │
│ Reader view: WebView + foliate-js       │        │ Auth (static token / key) │
│ Local DB (SQLite/drift): books, ops,    │ <────> │ SQLite: ops, books meta   │
│   materialized state, sync cursor       │  HTTPS │ Blob store: EPUB + covers │
│ Sync engine: op queue, push/pull        │        │ Converter worker          │
│ File cache: epubs, covers               │        │  (calibre ebook-convert)  │
└─────────────────────────────────────────┘        └───────────────────────────┘
```

### Client modules
- `library`: grid/list, search, sort, shelves, import from file picker, cover thumbnails.
- `reader`: host for foliate-js (`<foliate-view>`) in a WebView, JS bridge (locations, selections, taps, search results), theme injection.
- `data`: local SQLite (drift), repositories, file cache with eviction.
- `sync`: op log writer, push/pull loop, connectivity and retry, conflict-free merge.
- `settings`: server URL and token, typography, theme, reading mode.

### Server modules
- `api`: handlers, auth middleware, request limits.
- `store`: SQLite (ops, books, tombstones), content-addressed blob storage (sha256).
- `ingest`: validate EPUB, extract OPF metadata and cover, dedupe by hash.
- `convert`: queue converting PDF/MOBI/AZW3 to EPUB (calibre in the Docker image).
- `sync`: append-only op log, per-device cursor.

## Sync design

- Every change is an immutable **op**: `{id (ULID), device_id, hlc timestamp, entity, entity_id, field, value}`.
- Clocks: hybrid logical clock (HLC) so ordering is stable despite skewed device clocks.
- Entity CRDTs:
  - Scalar fields (title, position, settings): **LWW-register** keyed by HLC.
  - Collections (shelf membership, bookmarks): **OR-Set** with tombstones.
  - Highlights and notes: one record per annotation (LWW fields: range, color, note text); deletes via tombstone. Note text is a candidate for text-CRDT later; LWW per note in v1.
  - Reading position: LWW per book, anchored by EPUB CFI plus progress %.
- Protocol: `POST /sync/push` (batch of ops), `GET /sync/pull?cursor=N` (ops after server sequence N). Server assigns a monotonically increasing sequence, dedupes by op id, never rewrites history.
- Client materializes current state from ops; periodic compaction/snapshot to bound log size.
- Book files sync separately (blobs by hash): upload on import, lazy download on open, optional "download all".

## Op catalogue (v1 wire format)

Op value is UTF-8 JSON. HLC string is `{wall_ms:013d}-{counter:04x}-{device_id}` so plain string comparison orders correctly. ULIDs for record ids.

| entity | entity_id | fields / kinds |
|---|---|---|
| `book` | sha256 of the EPUB file (content-addressed, so a client import and a server watched-folder ingest of the same file merge instead of duplicating) | LWW: `title`, `author`, `series`, `series_index`, `language`, `direction`, `fixed`, `cover_sha256`, `added_at`, `deleted`; OR-Set: `tags` (`{"op":"add"|"remove","v":"..."}`) |
| `position` | book sha256 | LWW: `cfi`, `fraction` (written together as one `pos` field so they never split) |
| `annotation` | ULID | LWW: `book`, `cfi`, `quote`, `prefix`, `suffix`, `color`, `note`, `deleted` |
| `bookmark` | ULID | LWW: `book`, `cfi`, `label`, `deleted` (unique ids plus tombstone behave as an OR-Set) |
| `shelf` | ULID | LWW: `name`, `deleted`; OR-Set: `members` (book sha256s) |
| `setting` | key string | LWW: `value` |

Rules: a tombstone (`deleted:true`) is a normal LWW field, so a later edit with a higher HLC does not resurrect a record unless it also sets `deleted:false`. The server never interprets ops beyond validation; only clients materialize. The one exception is server-authored `book` ops from the watched folder (device id `server`).

## API sketch

All routes live under `/v1` except `/healthz`. The server returns its API version in `GET /v1/version` so clients can refuse an incompatible server.

```
GET    /v1/version                 {api: 1, build: "..."}
POST   /v1/auth/check              verify token
GET    /v1/sync/pull?cursor=N      ops since N
POST   /v1/sync/push               ops batch
PUT    /v1/blobs/{sha256}          upload epub/cover; EPUBs are ingested (metadata, cover, index row) on upload
GET    /v1/blobs/{sha256}          download
GET    /v1/books                   server index (search, paging)
GET    /v1/lookup?q=               online metadata candidates
POST   /v1/import                  upload non-EPUB, returns job id (v1.1)
GET    /v1/import/{job}            conversion status (v1.1)
GET    /opds                       OPDS catalog (token via basic auth)
GET    /healthz
```

Auth: single long random bearer token set via env/config, TLS expected (reverse proxy or built-in via autocert). Rate limit and body size caps.

## Reader features

- Pagination and scroll modes, tap zones, swipe, keyboard/mouse on desktop.
- Typography: font family (bundle Persian fonts, e.g. Vazirmatn), size, line height, margins, justify, publisher styles on/off.
- Themes: light, dark, sepia, black (OLED).
- RTL and Persian digits, vertical text where foliate-js allows. Set page direction explicitly from the book language, since foliate-js reported no direction for the real Persian book.
- Fixed-layout EPUBs: spread handling, zoom and pan.
- TOC, in-book search with result list, jump by progress, bookmarks.
- Highlights (colors), notes, annotation list per book.

## Milestones

**Phase 0: Foundations**
- Repo layout (`app/`, `server/`, `docs/`), CI, lint. No Docker on local dev: Docker is used only by CI to build the release image.

**Phase 1: Reader spike (de-risk)**
- Done on macOS (WKWebView, foliate-js vs epub.js). Remaining: Flutter WebView + foliate-js on iOS and Android with real books, including a real fixed-layout one; re-measure turn latency with a visible window.
- Test set: Persian novel, fixed-layout comic, image-heavy textbook, plain novel.
- Exit criteria: RTL correct, fixed layout renders, page turn under ~150 ms, memory acceptable. If it fails, evaluate alternative engine before continuing.

**Phase 2: Local app**
- Library DB, import, metadata/covers (client-side parse), reader with progress, TOC, themes and typography, scroll/paginated toggle.

**Phase 3: Server MVP**
- Auth, blob store, ingest and metadata, op log, push/pull; release binaries (Docker image built in CI only).

**Phase 4: Sync**
- Client op log, HLC, materialization, background sync, book file upload/download, offline queue, conflict tests.

**Phase 5: Annotations and search**
- Highlights, notes, bookmarks, shelves synced; in-book search.

**Phase 6: Format import**
- Server conversion queue (PDF/MOBI to EPUB), status UI.

**Phase 7: Polish and release**
- Fixed-layout polish, performance, backup/export, packaging for all platforms, docs.

## Working rules (apply to every step below)

- **TDD:** red, green, refactor. Write the failing test first and see it fail for the right reason, then the smallest code to pass, then clean up. A step with no test says why (e.g. config files).
- **One step = one commit = one push.** Tick the step's checkbox in this file in the same commit. Never batch steps. If a step grows past ~1 hour of work, split it and add the new steps here first.
- **Commit message:** `<id>: <imperative summary>` (e.g. `S1.03: reject requests without bearer token`). Body: what and why in a line or two. Attribution trailer as per the session's commit rules.
- **Push after every commit** (`git push`; one-time `git config push.autoSetupRemote true`). Never push a red build: run the step's test command and the linter first.
- **No Docker locally.** Server runs as a plain binary (`go run ./cmd/wepub-server`), SQLite via pure-Go `modernc.org/sqlite` (no cgo, no services). Integration tests use `httptest` and temp dirs. Dockerfile exists but is only built in CI and never required for a local step.
- **Test commands:** server `cd server && go test ./... && go vet ./...`; client `cd app && flutter analyze && flutter test`; reader JS `cd app/assets/reader && node --test`.
- **Branching:** work on `main` with short steps, or a branch per phase merged when the phase's exit test is green. Decide at F0.2.
- **Proxy note:** on this machine pub.dev/GitHub/npm may need `HTTPS_PROXY=http://127.0.0.1:10808` with `NO_PROXY=localhost,127.0.0.1` (needed so Flutter's debug connection to localhost still works).
- **Shared test vectors:** anything both Go and Dart implement (ULID, HLC ordering, LWW/OR-Set merge, CFI quote anchoring) is specified once as JSON in `docs/vectors/` and tested by both sides.

## Implementation checklist

Legend: `Test:` the failing test to write first. `Do:` the minimal change. `Done:` how to know it is finished. Check the box in the same commit.

### F: Foundations

- [ ] **F0.1 Init repo hygiene** — Do: root `.gitignore` (Flutter, Go, macOS, `*.db`, `/data`), `README.md` (one paragraph + link to plan.md), `LICENSE` (MIT, 2026). Done: `git status` clean, commit pushed.
- [ ] **F0.2 Remote and push setup** — Do: create the remote repo, `git remote add origin ...`, `git config push.autoSetupRemote true`, push `main`. Decide branching rule (see Working rules). Done: `git push` works with no arguments.
- [ ] **F0.3 Commit the spike** — Do: commit `spike/` as reference only (README note: "throwaway, results in plan.md"); make sure `spike/assets/books/divan_hafez.epub` stays ignored (CC BY-SA). Done: spike builds from a clean checkout after `flutter pub get`.
- [ ] **F0.4 Record foliate-js pin** — Do: add `docs/vendor.md` with foliate-js repo, commit `78914ae`, license (MIT), date vendored, and the update procedure. Done: file committed.
- [ ] **F0.5 Layout dirs** — Do: create `server/`, `app/`, `docs/vectors/` each with a `.gitkeep`. Done: tree matches Architecture section.
- [ ] **F0.6 CI: server job** — Do after S0.1 (a CI job with no Go module fails): `.github/workflows/server.yml` running `go vet` and `go test ./...` on push and PR. Done: first green run on `main`.
- [ ] **F0.7 CI: client job** — Do: `.github/workflows/app.yml` running `flutter analyze` and `flutter test` on push and PR (Linux runner). Do after C0.1. Done: green run.
- [ ] **F0.8 CI: reader JS job** — Do: workflow step running `node --test` in `app/assets/reader`. Do after C4.3 (first JS tests). Done: green.
- [ ] **F0.9 CI: release image (CI-only Docker)** — Do: `server/Dockerfile` (multi-stage, static Go binary, non-root user, `ca-certificates`), workflow job that builds it (no push yet). Done: CI builds the image; nothing needed locally.
- [ ] **F0.10 Author shared vectors** — Do: write `docs/vectors/{ulid,hlc,lww,orset,epub-meta,anchor,textnorm}.json` by hand from the Op catalogue (inputs and expected outputs, including ties, duplicates, out-of-order, resurrect attempts, Persian text). Test: a tiny schema-check script (`node docs/vectors/check.mjs`) validates JSON shape in CI. Done: vectors committed before any code that consumes them (do this before S3.1 and C1.1).
- [ ] **F0.11 Pin toolchains** — Do: Go version in `go.mod` and CI, Flutter version (stable 3.38.9 as used in the spike) in CI and `docs/vendor.md`, Node LTS for JS tests. Done: CI uses the pinned versions.

### S: Server (Go + SQLite)

**S0 Skeleton**
- [ ] **S0.1 Module and health test** — Test: `GET /healthz` returns 200 `{"ok":true}` via `httptest`. Do: `go mod init`, `internal/api` router, handler. Done: `go test ./...` green.
- [ ] **S0.2 Main binary** — Do: `cmd/wepub-server/main.go` starting `http.Server` on `:8080`, graceful shutdown on SIGINT. Test: unit test of `Run(ctx, cfg)` starting and stopping on a random port. Done: `go run ./cmd/wepub-server` then `curl localhost:8080/healthz`.
- [ ] **S0.3 Config from env** — Test: table test for defaults and overrides (`WEPUB_ADDR`, `WEPUB_DATA_DIR`, `WEPUB_TOKEN`, `WEPUB_MAX_UPLOAD_MB`); missing token is an error. Do: `internal/config`. Done: tests green.
- [ ] **S0.4 Structured logging** — Test: handler logs method, path, status, duration without the Authorization header. Do: `log/slog` middleware. Done: test asserts no token in log output.
- [ ] **S0.5 Request body limit** — Test: body over limit returns 413. Do: `http.MaxBytesReader` middleware using config. Done: green.
- [ ] **S0.6 Version endpoint and /v1 prefix** — Test: `GET /v1/version` returns `{api:1}`; all other routes are mounted under `/v1`, unversioned paths 404 except `/healthz` and `/opds`. Done: green. (Do this before S1.)

**S1 Auth**
- [ ] **S1.1 Reject missing token** — Test: protected route without header returns 401. Do: bearer middleware wrapping all routes except `/healthz` and `/v1/version`. Done: green.
- [ ] **S1.2 Reject wrong token, constant time** — Test: wrong token 401; correct token 200. Do: `subtle.ConstantTimeCompare`. Done: green.
- [ ] **S1.3 POST /v1/auth/check** — Test: 200 with valid token, 401 otherwise. Do: handler. Done: green.
- [ ] **S1.4 Basic-auth variant for OPDS** — Test: `/opds` accepts HTTP Basic where password is the token. Do: middleware accepts either scheme on `/opds` only. Done: green.

**S2 Storage (SQLite)**
- [ ] **S2.1 Open DB and pragmas** — Test: opening a temp DB sets WAL, foreign keys on, busy timeout. Do: `internal/store` with `modernc.org/sqlite`. Done: green.
- [ ] **S2.2 Migration runner** — Test: applies numbered migrations once, idempotent on rerun, records version. Do: embed `migrations/*.sql`. Done: green.
- [ ] **S2.3 Migration 001: ops table** — Test: insert one op, read it back. Schema: `seq INTEGER PRIMARY KEY AUTOINCREMENT, id TEXT UNIQUE NOT NULL, device_id TEXT, hlc TEXT, entity TEXT, entity_id TEXT, field TEXT, value BLOB, created_at`. Done: green.
- [ ] **S2.4 Migration 002: blobs table** — Test: row per sha256 with size, mime, created_at. Done: green.
- [ ] **S2.5 Migration 003: books index** — Test: `books(sha256 PRIMARY KEY, title, author, language, direction, fixed, cover_sha256, added_at)`. This is a server-side index of ingested files, not the CRDT source of truth. Done: green.

**S3 Op log and sync API**
- [ ] **S3.1 Op validation** — Test: table test rejecting empty id, bad ULID, empty entity, oversize value (>64 KiB), malformed hlc, hlc device part different from `device_id`, and hlc wall time more than 24 h ahead of server time (a device with a far-future clock would otherwise win every LWW field forever). Do: `internal/sync.ValidateOp`. Done: green.
- [ ] **S3.2 Append ops, assign seq** — Test: appending N ops yields increasing seq. Do: `store.AppendOps`. Done: green.
- [ ] **S3.3 Dedupe by op id** — Test: pushing the same op twice stores one row and returns the same seq. Do: `INSERT OR IGNORE` plus lookup. Done: green.
- [ ] **S3.4 Pull after cursor** — Test: `PullOps(cursor, limit)` returns ops with seq > cursor in order, plus `next_cursor` and `has_more`. Done: green.
- [ ] **S3.5 POST /v1/sync/push** — Test: valid batch returns `{accepted, seq_max}`; invalid op rejects the whole batch with 400 and stores none (transaction). Done: green.
- [ ] **S3.6 GET /v1/sync/pull** — Test: query `cursor`, `limit` (default 500, max 1000); empty log returns empty array and same cursor. Done: green.
- [ ] **S3.7 Idempotent batches under concurrency** — Test: two concurrent pushes of overlapping ops, no duplicate rows, no deadlock (`-race`). Do: single-writer transaction. Done: `go test -race` green.
- [ ] **S3.8 Cross-device convergence test (server view)** — Test: ops pushed by devices A and B in different orders produce the identical pull stream order only by seq, never mutated. Done: green.
- [ ] **S3.9 Server-side LWW compaction** — Test: after compaction, for every (entity, entity_id, field) of an LWW kind only the highest-HLC op remains; a fold of a full pull before and after compaction gives the identical latest-value map; OR-Set kinds (`tags`, `members`) and tombstones are untouched; cursors of devices that already pulled stay valid (seq numbers are never reused). Do: `wepub-server compact` command, never automatic in v1. Done: green. Reason: without this the op log grows forever (positions, settings, note edits).

**S4 Blob store**
- [ ] **S4.1 Path layout** — Test: sha256 `ab12...` maps to `data/blobs/ab/12/ab12...`; rejects non-hex or wrong-length input (path traversal). Do: `internal/blob.Path`. Done: green.
- [ ] **S4.2 Put streams and verifies hash** — Test: uploading bytes under the wrong hash is rejected and leaves no file; correct hash stored via temp file + atomic rename. Done: green.
- [ ] **S4.3 Get with Range support** — Test: full read, `Range: bytes=0-9`, missing blob 404. Do: `http.ServeContent`. Done: green.
- [ ] **S4.4 PUT /v1/blobs/{sha256}** — Test: auth required, size limit, idempotent re-upload returns 200. Done: green.
- [ ] **S4.5 GET /v1/blobs/{sha256} + HEAD** — Test: `ETag` is the hash, `Cache-Control: immutable`, HEAD returns size. Done: green.
- [ ] **S4.6 Orphan accounting** — Test: `blob.List` returns hash, size, mtime for GC and backup use. Done: green (GC itself is S9).

**S5 EPUB ingest**
- [ ] **S5.1 Fixture set** — Do: copy `spike/tool/make_epubs.py` to `server/testdata/make_epubs.py`, generate persian, fixed, plain, plus one corrupt zip and one zip without `container.xml`. Done: fixtures committed (small).
- [ ] **S5.2 Open EPUB zip safely** — Test: rejects non-zip, zip-slip entries (`../`), entry count > 10k, uncompressed size > 500 MB (zip bomb). Do: `internal/ingest.OpenEPUB`. Done: green.
- [ ] **S5.3 Locate OPF** — Test: parses `META-INF/container.xml` to the rootfile path; error if missing. Done: green.
- [ ] **S5.4 Parse metadata** — Test: title, authors, language, identifier, series (calibre meta) for persian/plain fixtures, including Persian title as UTF-8. Done: green.
- [ ] **S5.5 Detect layout and direction** — Test: `rendition:layout` pre-paginated flag; `page-progression-direction`; falls back to language (`fa`, `ar`, `he`, `ur` => rtl). This feeds the client's explicit page direction. Done: green.
- [ ] **S5.6 Extract cover** — Test: cover from EPUB3 `properties="cover-image"`, EPUB2 `<meta name="cover">`, and none (returns nil). Do: write cover to blob store, downscale to max 600px JPEG. Done: green.
- [ ] **S5.7 Book identity** — Test: same file twice yields same sha256 and one `books` row. Done: green.
- [ ] **S5.8 Ingest on EPUB blob upload** — Test: `PUT /v1/blobs/{sha}` of an EPUB (detected by content, not header) indexes a `books` row (title, author, language, direction, fixed, cover_sha256) and stores the cover; a non-EPUB blob (cover image) is stored without indexing; invalid EPUB returns 422 and stores nothing; duplicate returns 200 unchanged. Do: shared `ingest.Process(file)` used by upload and by the watched folder. The upload path appends no ops (the client authors its own `book` ops, S5.8 only builds the server index). Done: green.
- [ ] **S5.8b Server-authored book ops (watched folder only)** — Test: ingesting from the watched folder appends `book` ops with `device_id=server`, `entity_id=sha256`; later a client that imports the same file produces ops that merge on the same entity (no duplicate). Done: green.
- [ ] **S5.9 Fuzz the parser** — Test: Go fuzz target for `OpenEPUB` and OPF parse on mutated fixtures; no panics in 60 s. Done: committed corpus.

**S6 Book listing and OPDS**
- [ ] **S6.1 GET /v1/books** — Test: paged list sorted by `added_at`, `?q=` title/author search with Persian normalization (Arabic ي/ك match ی/ک, ZWNJ and diacritics ignored, Persian/Arabic digits equal ASCII digits). Do: `internal/textnorm` shared by search. Done: green.
- [ ] **S6.2 OPDS root and acquisition feed** — Test: valid Atom XML with `application/atom+xml;profile=opds-catalog`, one entry per book with acquisition link to `/v1/blobs/{sha}` and cover link. Done: green.
- [ ] **S6.3 OPDS search** — Test: `/opds/search?q=` and OpenSearch description. Done: green.

**S7 Watched folder**
- [ ] **S7.1 Scan once** — Test: temp dir with 2 EPUBs and 1 txt: ingests 2, ignores 1, records failures in a log, idempotent on rescan. Done: green.
- [ ] **S7.2 Poll loop** — Test: new file appears after N seconds (fake clock/ticker interface); file still being written (size changing) is skipped until stable. Do: polling, not fsnotify, for portability. Done: green.
- [ ] **S7.3 Wire in main** — Do: `WEPUB_WATCH_DIR` optional env; moves processed files to `.done/`. Done: manual run with a sample file shows it in `GET /v1/books`.

**S8 Online metadata lookup (server)**
- [ ] **S8.1 Provider interface and fake** — Test: `Lookup(ctx, query)` returns candidates; fake provider used by tests. Done: green.
- [ ] **S8.2 Open Library provider** — Test: parse a recorded JSON fixture into candidates (title, authors, year, cover URL). No network in tests. Done: green.
- [ ] **S8.3 GET /v1/lookup?q=** — Test: auth required, timeout 5 s, provider errors become 502 with message, response cached 24 h in memory. Done: green. (Client applies a candidate only after user confirms.)

**S9 Backup and ops**
- [ ] **S9.1 SQLite online backup** — Test: backup file opens and has identical op count while writes continue. Do: `VACUUM INTO` temp then rename. Done: green.
- [ ] **S9.2 Snapshot archive** — Test: a snapshot is the DB backup plus a manifest (hash, size) of every blob; blobs are immutable and content-addressed, so they are copied incrementally into `<target>/blobs/` (only missing hashes) rather than re-archived each time. Restore into an empty dir reproduces the same `GET /v1/sync/pull`, and fails loudly if any manifest hash is missing or has a wrong checksum. Done: green.
- [ ] **S9.3 Retention** — Test: keeps newest K snapshots, deletes older. Done: green.
- [ ] **S9.4 Scheduler** — Test: runs at interval using injected clock; skips if previous still running. Do: `WEPUB_BACKUP_EVERY`, `WEPUB_BACKUP_KEEP`, `WEPUB_BACKUP_DIR`. Done: green.
- [ ] **S9.5 S3-compatible target (optional)** — Test: target interface (put object, list, get) with a local-dir fake used for S9.2 too; uploader copies DB snapshot + missing blobs. Do: minio-go implementation behind `WEPUB_BACKUP_S3_*`. Done: unit tests green; manual test deferred.
- [ ] **S9.6 CLI: backup and restore** — Test: `wepub-server backup` and `restore --from X --into Y` against temp dirs; restore refuses non-empty target. Done: green.
- [ ] **S9.7 Blob GC (dry-run by default)** — Test: blobs not referenced by any book/cover/op are listed; `--delete` removes only those older than 7 days. Done: green.

**S10 Server hardening**
- [ ] **S10.1 Rate limit** — Test: >N req/s per IP gets 429; limit is per token-failure to blunt brute force. Done: green.
- [ ] **S10.2 Security headers and CORS off** — Test: `X-Content-Type-Options`, no CORS headers. Done: green.
- [ ] **S10.3 Release binary build** — Do: `make build` produces static binaries for linux/amd64 and arm64 and darwin/arm64 with version stamp. Done: runs locally, no Docker.
- [ ] **S10.4 Deployment docs** — Do: `docs/deploy.md` with systemd unit, Caddy reverse-proxy example, env table, backup/restore. Done: reviewed.

### C: Client (Flutter)

**C0 Skeleton**
- [ ] **C0.1 Create `app/`** — Do: `flutter create --platforms=android,ios,macos --org dev.wepub app`; delete the sample counter test; set `minSdk`, iOS 13, macOS 12 deployment targets (Xcode needs >= 12). Test: smoke widget test builds `WepubApp`. Done: `flutter test` green, `flutter analyze` clean.
- [ ] **C0.2 Lints** — Do: `flutter_lints` + stricter rules in `analysis_options.yaml`. Done: analyze clean.
- [ ] **C0.3 Folder structure** — Do: `lib/{core,data,domain,sync,reader,library,settings,l10n}` with barrel files. Done: compiles.
- [ ] **C0.4 State management and DI** — Do: add `flutter_riverpod`; root `ProviderScope`. Test: provider override works in widget test. Done: green.
- [ ] **C0.5 App theme (M3)** — Test: golden for light/dark `ColorScheme` seeded theme. Do: `core/theme.dart`. Done: golden committed.
- [ ] **C0.6 Adaptive scaffold** — Test: width < 600 shows bottom nav; >= 600 shows rail (widget tests with sized surface). Done: green.

**C1 Core primitives (shared vectors with server)**
- [ ] **C1.1 ULID** — Test: monotonic within same ms, sortable, parses vectors from `docs/vectors/ulid.json`. Do: `core/ulid.dart`. Done: green. (Server mirrors in S3.1 using same vectors.)
- [ ] **C1.2 HLC type** — Test: ordering by (wall, counter, device); string round-trip `0000017c...-0001-<device>` per `docs/vectors/hlc.json`. Do: `core/hlc.dart`. Done: green.
- [ ] **C1.3 HLC tick and receive** — Test: local tick increases; receive from the future advances; clock going backwards keeps monotonic; max drift guard (reject > 24 h ahead). Done: green.
- [ ] **C1.4 Device ID** — Test: generated once, persisted, stable across restarts (fake storage). Done: green.
- [ ] **C1.5 Cross-check with server vectors** — Test: CI step fails if the Dart or Go test does not read every file in `docs/vectors/` (script lists files, greps both suites). Done: green. (The Go side is written in S3.1, which must read `ulid.json` and `hlc.json`.)

**C2 Local database (drift)**
- [ ] **C2.1 Add drift and in-memory DB for tests** — Do: `drift`, `drift_dev`, `sqlite3_flutter_libs`, `build_runner`. Test: open in-memory DB, run a trivial query. Done: green.
- [ ] **C2.2 Table `ops`** — Test: insert/read an op; `synced` flag and `server_seq` nullable. Done: green.
- [ ] **C2.3 Table `books`** — Test: CRUD; columns sha256 (primary key, equals the book entity id), title, author, language, direction, fixed, cover path, local file path, size, added_at, last_read_at, pinned. Done: green.
- [ ] **C2.4 Table `positions`** — Test: one row per book, cfi + fraction + hlc. Done: green.
- [ ] **C2.5 Tables `annotations`, `bookmarks`, `shelves`, `shelf_books`, `settings`** — Test: each CRUD with tombstone column where relevant. Done: green.
- [ ] **C2.6 Table `sync_state`** — Test: stores `last_pulled_seq`, `server_url` hash. Done: green.
- [ ] **C2.7 Migrations test** — Test: schema v1 -> v2 stub migration keeps data. Do: drift schema dump + `drift_dev` verifier. Done: green.

**C3 Merge engine (pure Dart, no Flutter)**
- [ ] **C3.1 LWW register** — Test: highest HLC wins, ties broken by device id, idempotent re-apply, vectors from `docs/vectors/lww.json`. Do: `domain/crdt/lww.dart`. Done: green.
- [ ] **C3.2 OR-Set** — Test: add/remove/re-add semantics; concurrent add + remove keeps add; vectors from `docs/vectors/orset.json`. Done: green.
- [ ] **C3.3 Op model and (de)serialization** — Test: JSON round trip matches the server's schema; unknown fields preserved. Done: green.
- [ ] **C3.4 Materializer: books** — Test: `book` field ops (title, author, deleted, tags OR-Set) produce `books` rows; replaying in any order converges. Done: green.
- [ ] **C3.5 Materializer: positions** — Test: older op never overwrites newer position. Done: green.
- [ ] **C3.6 Materializer: annotations** — Test: create, edit color, edit note, delete (tombstone) then late edit does not resurrect. Done: green.
- [ ] **C3.7 Materializer: bookmarks and shelves** — Test: OR-Set membership; shelf rename LWW. Done: green.
- [ ] **C3.8 Materializer: settings** — Test: per-key LWW. Done: green.
- [ ] **C3.9 Property test: convergence** — Test: generate random op sets, shuffle into 3 device orders with duplicates; final materialized state identical (use `glados` or hand-rolled seeded RNG, 200 cases). Done: green.
- [ ] **C3.10 Snapshot/compaction** — Test: compacting ops below a watermark yields the same state and fewer rows; drops LWW ops superseded by a newer HLC for the same (entity, entity_id, field); keeps OR-Set ops and tombstones until S3.9-style safe horizon; never compacts unsynced ops. Done: green.

**C4 Reader core (WebView + foliate-js)**
- [ ] **C4.1 Vendor foliate-js** — Do: copy pinned files from F0.4 into `app/assets/foliate/` (minus pdfjs), register assets, add `LICENSE` copy. Done: `flutter pub get` ok, app builds on macOS.
- [ ] **C4.2 Bridge protocol spec** — Do: `docs/reader-bridge.md` listing messages. JS->Dart: `ready`, `relocate{cfi,fraction,sectionIndex,tocItem}`, `tap{zone}`, `selection{cfi,text,rect}`, `error`. Dart->JS: `open{url,direction,flow}`, `next`, `prev`, `goTo{cfi|fraction|href}`, `setStyle{...}`, `addHighlight`, `removeHighlight`, `search{q}`. Done: reviewed.
- [ ] **C4.3 Bridge JS module with tests** — Test: `node --test` on `bridge.js` using a stubbed `foliate-view` object: each Dart->JS message calls the right API; each event serializes to the spec. Do: `assets/reader/bridge.js` (pure functions, no DOM where possible). Done: green; add to CI (F0.8).
- [ ] **C4.4 Local file server for assets** — Do: serve app files and book files to the WebView from a loopback server with a random per-run port and path token (not a fixed port; the spike showed fixed ports collide). Test: Dart test fetches a file through the server; wrong token 403; binds only 127.0.0.1. Done: green.
- [ ] **C4.5 ReaderController interface** — Test: Dart-side fake implementing `open/next/prev/goTo/setStyle` and a stream of `ReaderEvent`; widget tests drive UI against the fake. Do: `reader/reader_controller.dart`. Done: green.
- [ ] **C4.6 WebView implementation** — Do: `flutter_inappwebview` implementation of `ReaderController` forwarding via `JavaScriptHandler`; macOS entitlements for network client/server. Test: integration test (`integration_test`, macOS) opens `plain.epub`, expects `relocate` event within 3 s. Security: disable scripts inside book content (foliate sandbox, no `allowScripts`), block external network requests from the reader page except the loopback server, and test that a fixture EPUB with a `<script>` does not execute it. Done: green locally; documented as not run in CI. **Gate: run C4.14 (iOS) and C4.15 (Android) immediately after this step, before C4.7**, since mobile is the biggest remaining risk.
- [ ] **C4.7 Explicit page direction** — Test: `direction` from book metadata (S5.5) is passed to `open`; Persian book with null direction gets `rtl` from language. Do: set `dir` on the paginator/`book.dir` before display. Done: integration test with `persian.epub` asserts first `next` moves forward in RTL sense (location fraction increases).
- [ ] **C4.8 Paginated and scroll modes** — Test: bridge unit test for `flow` attribute; integration test toggles and still relocates. Done: green.
- [ ] **C4.9 Fixed-layout open** — Test: integration with `fixed_rtl.epub`: layout detected, `next` changes position. Done: green.
- [ ] **C4.10 Tap zones** — Test: pure function mapping (x,width,direction) to prev/next/menu, RTL flips sides. Done: green; wired to `tap` event.
- [ ] **C4.11 Reader screen shell** — Test: widget test with fake controller shows progress, back button, menu toggle on center tap. Done: green.
- [ ] **C4.12 Progress persistence (local)** — Test: `relocate` events are debounced (500 ms) and written to the local `positions` row only (no op); a position op is emitted at most once per 30 s, plus on pause, app background and reader close (fake clock; 200 page turns in 10 minutes produce at most ~20 ops). Reopening restores via `goTo(cfi)`. Reason: one op per page turn would bloat the log and every device's sync. Done: green.
- [ ] **C4.13 Visible-window latency check** — Do: run spike-style timing with a focused window on real books; record in plan.md. Done: p95 page turn < 150 ms or issue filed.
- [ ] **C4.14 iOS run** — Do: build `app` for iOS simulator; run the integration test. Fix ATS exceptions for loopback if needed. Done: reader opens `plain.epub` and `persian.epub`; result recorded in plan.md.
- [ ] **C4.15 Android run** — Do: emulator run, cleartext loopback allowed via network security config. Done: same as C4.14.

**C5 Library**
- [ ] **C5.1 EPUB metadata parser (Dart)** — Test: same fixtures as S5 give same title/author/language/direction/layout; mirrors `docs/vectors/epub-meta.json`. Do: `archive` package + `xml`. Done: green.
- [ ] **C5.2 Cover extraction (Dart)** — Test: returns bytes for fixtures with cover, null otherwise. Done: green.
- [ ] **C5.3 BookRepository** — Test: `importBytes` stores file under app dir by sha256, inserts `books` row, appends `book` field ops with `entity_id = sha256` (see Op catalogue), duplicate import is a no-op. Done: green with fake file system.
- [ ] **C5.4 Library list UI** — Test: golden grid and list with 0, 1, 50 books; Persian title renders RTL. Done: green.
- [ ] **C5.5 Sort and search** — Test: sort by recent/title/author (Persian collation, not byte order); search filters by title/author using the same normalization as the server (ي/ی, ك/ک, ZWNJ, diacritics, digit variants; vectors in `docs/vectors/textnorm.json`). Done: green.
- [ ] **C5.6 File picker import** — Test: use a `FilePicker` interface with a fake; error toast on invalid EPUB. Done: green.
- [ ] **C5.7 Drag-and-drop (macOS)** — Test: drop handler converts paths to import calls (unit). Done: manual check on macOS.
- [ ] **C5.8 Open-with / share-sheet intents** — Do: Android intent filters (`application/epub+zip`), iOS document types, macOS file association. Test: URI-to-import handler unit-tested. Done: manual check on each platform.
- [ ] **C5.9 Metadata edit screen** — Test: editing title/author/series/tags appends field ops and updates the list; cancel changes nothing. Done: green.
- [ ] **C5.10 Online lookup UI** — Test: with fake server client, shows candidates; nothing applies until "Use this" is tapped. Done: green.
- [ ] **C5.11 Shelves** — Test: create, rename, delete shelf; add/remove book; OR-Set ops appended. Done: green.
- [ ] **C5.12 Delete book** — Test: delete sets `deleted:true` on the book entity, removes the local file and cached cover, hides the book from the library, keeps annotations (restorable); server blob is untouched until GC (S9.7). Done: green.

**C6 Reader features**
- [ ] **C6.1 Style model** — Test: `ReaderStyle` (theme, font, size, line height, margins, justify, publisher styles) serializes to CSS variables string; defaults per language. Done: green.
- [ ] **C6.2 Themes** — Test: light, dark, sepia, black, high-contrast produce expected CSS; contrast ratio >= 7:1 for high-contrast (computed in test). Done: green.
- [ ] **C6.3 Fonts** — Do: bundle Vazirmatn and OpenDyslexic (check licenses, add to `docs/vendor.md`); inject `@font-face` through the local server. Test: style CSS references bundled font URL. Done: manual check with Persian text.
- [ ] **C6.4 Typography sheet UI** — Test: widget test changes size and emits `setStyle`; persisted as settings ops. Done: green.
- [ ] **C6.5 Persian digits option** — Test: text transform helper maps 0-9 to ۰-۹ for UI strings only (not book content). Done: green.
- [ ] **C6.6 TOC panel** — Test: nested TOC from bridge `toc` event renders; tapping calls `goTo(href)`; current item highlighted from `relocate`. Done: green.
- [ ] **C6.7 Progress slider** — Test: dragging emits `goTo(fraction)` on release only; shows percent. Done: green.
- [ ] **C6.8 In-book search** — Test: bridge unit test for `search` streaming results; UI lists excerpts, tap jumps to CFI. Done: green.
- [ ] **C6.9 Bookmarks** — Test: add/remove at current CFI, list per book, ops appended. Done: green.
- [ ] **C6.10 Highlights: create** — Test: `selection` event shows color toolbar; choosing a color appends `annotation` field ops (book, cfi, quote, prefix, suffix, color) with a new ULID, with CFI, quote text, prefix/suffix context. Done: green.
- [ ] **C6.11 Highlights: render** — Test: on `open` and `relocate` the app sends visible annotations to `addHighlight`; deleted ones are removed. Done: green.
- [ ] **C6.12 Notes and edit/delete** — Test: note text edit appends LWW op; delete tombstones. Done: green.
- [ ] **C6.13 Annotation list per book** — Test: sorted by position; tap navigates. Done: green.
- [ ] **C6.14 Re-anchoring** — Test: given quote + prefix/suffix and a modified HTML fixture, finds the new range; reports "orphaned" when missing (vectors in `docs/vectors/anchor.json`). Do: JS function in `bridge.js`. Done: green.
- [ ] **C6.15 Fixed-layout zoom/pan** — Test: gesture math (pure) for pinch-zoom bounds. Done: manual check on a fixed book.
- [ ] **C6.16 Large tap targets and high-contrast mode** — Test: widget test asserts min 48 dp targets with the large-targets setting. Done: green.

**C7 Settings**
- [ ] **C7.1 Settings repository** — Test: typed get/set, defaults, each set appends an op. Done: green.
- [ ] **C7.2 Server connection screen** — Test: validates URL (https required except localhost), token field obscured, "Test connection" calls `/auth/check` via fake client and shows result. Done: green.
- [ ] **C7.3 Secure token storage** — Test: token stored via `flutter_secure_storage` interface (fake in tests), never in the DB or logs. Done: green.
- [ ] **C7.4 Reading mode and storage limit** — Test: toggles persist; cache limit setting read by eviction logic. Done: green.
- [ ] **C7.5 App language** — Do: ARB files `en`, `fa`; `flutter gen-l10n`. Test: widget test in `fa` locale is RTL and shows Persian strings; missing-key check in CI. Done: green.

### X: Sync (client + server together)

- [ ] **X1.1 API client** — Test: against `httptest` server stub (Dart `HttpServer`) for `auth/check`, `sync/push`, `sync/pull`, error mapping (401, 413, 5xx, timeouts). Do: `sync/api_client.dart` with injected `http.Client`. Done: green.
- [ ] **X1.2 Push queue** — Test: unsynced ops are sent in batches of 200; marked synced only after 200 OK; failure leaves them unsynced; partial failure retried idempotently. Done: green.
- [ ] **X1.3 Pull loop** — Test: pulls from `last_pulled_seq`, applies ops via materializer in one DB transaction, advances cursor only after commit; handles `has_more`. Done: green.
- [ ] **X1.4 Echo suppression** — Test: ops authored locally and returned by pull are not re-applied twice (idempotent by id). Done: green.
- [ ] **X1.5 Backoff and connectivity** — Test: exponential backoff with jitter (fake timer), pauses offline, resumes on connectivity event. Done: green.
- [ ] **X1.6 Sync triggers** — Test: sync runs on app start, on resume, 5 s after a local write (debounced), and every 5 min while open. Done: green.
- [ ] **X1.7 Sync status UI** — Test: indicator shows idle, syncing, offline, error with last-synced time. Done: green.
- [ ] **X2.1 Book blob upload** — Test: after import, uploads blob by hash with resume-on-failure; `HEAD` first to skip existing. Done: green with fake server.
- [ ] **X2.2 Lazy download on open** — Test: opening a book not present locally downloads it, verifies sha256, shows progress; corrupt download is deleted and retried. Done: green.
- [ ] **X2.3 Cover sync** — Test: covers are fetched on demand and cached. Done: green.
- [ ] **X2.4 Cache eviction** — Test: when over limit, evicts least-recently-read unpinned books (never the open one); keeps metadata. Done: green.
- [ ] **X2.5 Pin "keep offline"** — Test: pinned books are exempt and downloaded in background on Wi-Fi setting. Done: green.
- [ ] **X3.1 Local end-to-end** — Do: `tool/e2e_local.sh` starts `go run` server on a temp data dir and runs the Dart sync tests against it (no Docker). Test: two simulated devices (two in-memory DBs) push/pull and converge. Done: script passes locally.
- [ ] **X3.2 Offline conflict scenarios** — Test: device A and B edit same highlight color and note offline, then sync both ways; both end identical, LWW by HLC. Done: green.
- [ ] **X3.3 Position sync across devices** — Test: reading on A updates B's start position; older position never regresses. Done: green.
- [ ] **X3.4 First-run restore** — Test: new device with empty DB pulls full log (several pages) and rebuilds library, positions, annotations. Done: green.
- [ ] **X3.5 Large log** — Test: 100k ops pull in < N s and bounded memory; compaction (C3.10) triggers. Done: benchmark recorded in plan.md.

### R: Release (v1)

- [ ] **R1 Error reporting** — Do: local-only error log screen with "copy report"; no analytics. Test: log redacts tokens. Done: green.
- [ ] **R2 Performance pass** — Do: profile library with 1,000 books and reader memory on a large EPUB; fix hot spots. Done: numbers in plan.md meet targets (cold start < 2 s, scroll 60 fps).
- [ ] **R3 App icons and names** — Do: icons, bundle ids, display name "wepub". Done: builds on three platforms.
- [ ] **R4 Android release** — Do: keystore (stored outside repo), `flutter build appbundle`, CI artifact. Done: installs on device.
- [ ] **R5 iOS release** — Do: signing, TestFlight upload. Done: build installs via TestFlight.
- [ ] **R6 macOS release** — Do: sign and notarize `.app`/dmg. Done: opens without Gatekeeper warning.
- [ ] **R7 GitHub release automation** — Do: tag-triggered workflow building server binaries and client artifacts, attaching them to the release, pushing the server image from CI. Done: dry-run on a pre-release tag.
- [ ] **R8 Docs** — Do: README quick start (run server, set token, connect app), `docs/deploy.md`, `docs/reader-bridge.md`. Done: a fresh machine follows them successfully.

### V1.1 backlog (not scheduled)

- [ ] Windows and Linux clients (choose webview per Open questions, then repeat C4).
- [ ] PDF/MOBI to EPUB conversion queue (server `convert` module, calibre in CI image only; local dev uses a fake converter).
- [ ] Reading stats, goals, streaks (ops: `session.add`).
- [ ] Text-to-speech.

## Testing strategy

- Server: unit tests for ops/merge, API integration tests, fuzz EPUB ingest.
- Sync: property-based tests (random op interleavings across N devices converge to identical state).
- Client: repository and sync unit tests, golden tests for library UI, manual device matrix for reader.
- Rendering corpus: fixed set of EPUBs checked on every release.

## Risks

| Risk | Mitigation |
|---|---|
| WebView differences across platforms (esp. Linux/Windows) | Phase 1 spike; pick supported webview plugin per platform early |
| foliate-js is self-described as unstable; slower than epub.js on reflowable books | Pin a commit, vendor it, keep a thin bridge so the engine can be swapped (epub.js, readium) |
| CFI stability for annotations across edits/re-imports | Store CFI plus text quote context for re-anchoring |
| CRDT complexity | Keep per-field LWW + OR-Set; no text CRDT in v1 |
| Conversion quality / calibre size in image | Optional converter container; clear failure states |
| Large libraries on mobile storage | Lazy download, cache eviction, "keep offline" flag |

## Deployment, backup, security (decided)

- TLS terminated by a reverse proxy (Caddy/nginx/Traefik); server listens on plain HTTP behind it.
- Auth: single bearer token, no extra encryption beyond TLS.
- Backups: scheduled in-server snapshots (consistent SQLite backup + blob manifest) with retention count, written to a local path or S3-compatible target; restore command in the server CLI.
- Name: wepub. License: MIT.

## UX, import, storage, i18n (decided)

- UI: Material 3, adaptive layout (bottom nav on phone, navigation rail on desktop), follows system light/dark.
- Import: file picker and drag-drop; OS share-sheet / "Open with" / file association; server watched folder (auto-ingest, then appears via sync); OPDS feed exposed by the server for third-party readers.
- Storage: books download lazily on open; least-recently-read are evicted when over a configurable size limit; "keep offline" pin exempts a book. Covers and metadata always cached.
- App language: English + Persian, full RTL UI (Flutter l10n, ARB files).

## Metadata, stats, accessibility, distribution (decided)

- Metadata: edit title, author, series, cover, tags (synced as ops). Online lookup (Open Library / Google Books) for metadata and covers, done server-side or client-side with user confirmation before applying.
- Stats: reading sessions (time, pages/progress) recorded as append-only ops; books finished; daily goals and streaks computed from the log. Stats screen with simple charts.
- Accessibility: dyslexia-friendly font (OpenDyslexic) in typography options; text-to-speech via system TTS (read-aloud from current position, sentence highlighting if feasible); large tap targets and a high-contrast theme. Screen reader support not targeted in v1 beyond Flutter defaults.
- Distribution: GitHub releases built by CI (APK, dmg, exe/msi, AppImage), Google Play and App Store listings, TestFlight/internal testing tracks, and plain sideload builds. Requires signing setup (Android keystore, Apple certs/notarization) in Phase 7.

## Open questions

- Linux/Windows webview choice. Findings so far: flutter_inappwebview has a Windows package (WebView2, 0.6.0) and a Linux package (WPE WebKit, 0.1.0-beta.1, young, heavy build deps); webview_win_floating covers Windows (WebView2) and Linux (webkit2gtk) but its view floats above Flutter widgets. Pick in the Phase 1 spike.
- Spike result (macOS, WKWebView via flutter_inappwebview, epub.js 0.3.93, synthetic books): Persian RTL reflowable paginates and turns pages in 21-120 ms (first open 6.1 s cold, 81 ms warm in scroll mode); plain EN and scroll mode fine (~30-70 ms). Fixed-layout RTL in paginated mode did NOT advance: CFI stayed at the first page after three `next()` calls, matching epub.js issue #1069; scroll mode advanced. Inferred from CFIs, not yet confirmed visually. iOS, Android not yet run.
- Spike result, foliate-js (same macOS setup, commit 78914ae, MIT): RTL detected for Persian; paginated Persian turns 76-240 ms except one 76.9 s stall that must be explained; scroll mode 300-700 ms; fixed-layout RTL navigated (spread to spread, prev worked), unlike epub.js. Slower than epub.js on reflowable books. Position/spread behaviour inferred from CFIs, not yet checked visually.
- Stall follow-up: rerun with instrumentation showed no stall (Persian paginated turns 106-117 ms, scroll 173-287 ms, fixed 72-78 ms). The webview reported document.hidden=true and no focus for the whole run, so all spike timings were taken with a hidden window and are not representative of real reading. The 76.9 s stall was not reproduced and is not explained (hidden window alone is ruled out as the cause: this run was hidden too). Re-measure with a visible window.
- Real-book check (Divan-e Hafez, fa.wikisource export, 1.1 MB, window visible this time): foliate-js paginated turns 101-297 ms (first 297 ms), scroll 102-158 ms, advanced, no stall; but it reported dir=null (no RTL detected from the package). epub.js: body dir rtl detected, first paginated turn 251 ms then 24-29 ms, but its location did not change between the last turn and prev (stuck or inconclusive); scroll mode advanced. Neither verified visually.
- Manual visual check on macOS by the user (both engines, real Persian book and fixed RTL): "seems good", no per-case detail recorded, so the epub.js paginated and fixed-layout doubts are not individually confirmed or cleared.
- Engine leaning: foliate-js primary (fixed-layout RTL works), epub.js as fallback; confirm after the stall is explained and a visual check on real books.
- Engine fixed-layout/RTL: epub.js has open RTL fixed-layout spread bugs (issues #1069, #1053, reports from 2020, resolution unconfirmed); foliate-js calls itself unstable with an unfinished fixed-layout renderer. Test both on the corpus in Phase 1.
