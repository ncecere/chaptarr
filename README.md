# Chaptarr

A self-hosted audiobook manager in the spirit of Sonarr, Radarr and Lidarr.
Follow authors, and Chaptarr finds their audiobooks on your indexers, hands
them to your download client, and files the finished downloads into a tidy
library folder that Audiobookshelf, Plex or Jellyfin can read.

One Go binary, with the React UI embedded and SQLite for storage. No external
services to run besides the indexers and download client you already have.

## Features

- **Follow authors** – add an author and their catalogue is imported, one
  entry per work (translations, abridgements and dramatisations are filtered
  out). New releases are picked up on a schedule.
- **Or add single books** – search the catalogue and add just the titles you want.
- **Series** – books are grouped and ordered by series.
- **Indexers** – any Torznab or Newznab feed: Prowlarr, Jackett, NZBHydra, or
  an indexer directly.
- **Download clients** – qBittorrent and SABnzbd.
- **Automatic and interactive search** – releases are scored on title/author
  match, preferred format (M4B, MP3, …), seeders, and a size-versus-runtime
  sanity check; anything doubtful is left for you to pick by hand.
- **Import** – finished downloads are hardlinked (or copied/moved) into
  `{Author}/{Series}/{Title}`, configurable. Torrents keep seeding.
- **Manual import** – when a download will not import on its own, pick the
  folder yourself and file it under the book.
- **Library scan** – audiobooks already on disk are matched to books you add.
- **Queue, history, wanted list, scheduled tasks and a log viewer.**

Metadata comes from Audible's public catalogue; author photos and bios from
[Audnexus](https://github.com/laxamentumtech/audnexus). Neither needs an account.
Under Settings → Metadata you can switch on more sources beside Audible
(iTunes, Open Library, and Google Books with an API key) for books it does
not list, and point any of them at a mirror or self-hosted server.

## Quick start

Requirements: Go 1.26+, Node 22+, pnpm.

```sh
make run
```

Then open <http://localhost:8789>. The first visit asks you to create the
admin account. After that:

1. **Settings → Root Folders** – choose where your audiobooks live.
2. **Settings → Indexers** – add a Torznab/Newznab feed. From Prowlarr, use
   the indexer's Torznab URL (`http://prowlarr:9696/1/api`) and Prowlarr's API key.
3. **Settings → Download Clients** – add qBittorrent or SABnzbd.
4. **Add New** – search for an author or a book.

### Docker

Images are published to `ghcr.io/ncecere/chaptarr` for amd64 and arm64:

| Tag               | What it is                                       |
| ----------------- | ------------------------------------------------ |
| `latest`          | The newest release                               |
| `v0.1.0`, `v0.1`  | One release; the newest patch of a minor version |
| `edge`            | The current `main` branch, for testing           |
| `sha-<commit>`    | One commit of `main`                             |

```sh
docker compose up -d            # pull the published image
docker compose up -d --build    # or build from source
```

While the repository is private, so is the image: run
`docker login ghcr.io` with a token that has `read:packages` first.

Binaries for Linux, macOS and Windows are attached to each
[release](https://github.com/ncecere/chaptarr/releases).

See `docker-compose.yml`. Chaptarr and the download client must see finished
downloads at the **same path**; mount one parent directory containing both
your downloads and your library into both containers. That also lets imports
use hardlinks, so a seeding torrent and the library copy share disk space.

## Configuration

| Flag       | Environment      | Default                    |                           |
| ---------- | ---------------- | -------------------------- | ------------------------- |
| `-addr`    | `CHAPTARR_ADDR`  | `:8789`                    | Listen address            |
| `-data`    | `CHAPTARR_DATA`  | OS config dir + `chaptarr` | Database directory        |
| `-debug`   | `CHAPTARR_DEBUG` | off                        | Debug logging             |

Everything else is configured in the UI. Back up the data directory to back
up Chaptarr; it holds a single SQLite database.

Behind a reverse proxy, forward `X-Forwarded-Proto` and `X-Forwarded-Host` so
session cookies are marked secure and same-origin checks pass.

## API

The UI's API is available to scripts: send your API key (Settings → Account)
in an `X-Api-Key` header. Routes and types are documented in
[`web/src/lib/api.ts`](web/src/lib/api.ts).

```sh
curl -H "X-Api-Key: $KEY" http://localhost:8789/api/v1/books?status=missing
```

## Development

```sh
make dev              # backend on :8789, database in ./data
cd web && pnpm dev    # UI with hot reload on :5173, proxying /api
make test             # go vet, go test, frontend lint
```

Layout:

```
cmd/chaptarr        entry point
internal/api        HTTP routes, auth, embedded UI
internal/service    application logic and the task scheduler
internal/store      SQLite
internal/metadata   Audible catalogue + Audnexus
internal/indexer    Torznab/Newznab client and release scoring
internal/download   qBittorrent and SABnzbd
internal/library    folder naming, import, disk scan
web                 React + Vite UI
```

## Releasing

Versions follow [Semantic Versioning](https://semver.org): a patch release
(`v0.1.1`) only fixes bugs, a minor release (`v0.2.0`) adds features, and a
major release changes something you have to act on. Before 1.0, a minor
release may also include breaking changes. Every change is recorded in
[`CHANGELOG.md`](CHANGELOG.md).

To cut a release from `main`:

1. In `CHANGELOG.md`, rename `## [Unreleased]` to `## [X.Y.Z] - YYYY-MM-DD`,
   add a new empty `## [Unreleased]` above it and update the links at the
   bottom. Commit and push.
2. Check, then tag:

   ```sh
   make release-check TAG=vX.Y.Z
   git tag -a vX.Y.Z -m "Chaptarr vX.Y.Z"
   git push origin vX.Y.Z
   ```

The tag starts the pipeline in `.github/workflows/ci.yml`. It refuses a tag
that is not a semantic version, is not on `main` or has no changelog section;
runs the tests; builds the image, starts it, scans it for known
vulnerabilities and signs it; and creates the GitHub Release with the
changelog section as its notes, plus the binaries, the image's SBOMs and
checksums. A tag with a hyphen (`vX.Y.Z-rc.1`) makes a pre-release, which
does not move `latest` or `vX.Y`.

## Known limitations

- No remote path mapping: the download client and Chaptarr must share paths.
- Downloads that arrive as archives (`.rar`, `.zip`) are not unpacked;
  SABnzbd unpacks its own, but a torrent of archives will fail to import.
- One quality rule (format preference) rather than full quality profiles.
- Single admin user.
