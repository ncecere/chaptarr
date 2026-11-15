# Changelog

All notable changes to Chaptarr are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html). Before 1.0, minor releases may include breaking changes; patch releases never do.

## [Unreleased]

### Added

- **Add missing books from a series.** The series page lists the books the catalogue has for the series that are not in your library, one edition per position, and adds the ones you tick.
- **Add all books by an author.** An author who is in the library because you added one of their books now has an "Add All Books" action that imports the rest of their catalogue.
- `GET /api/v1/series/{id}/missing` and `POST /api/v1/series/{id}/books`.

### Changed

- Status badges on covers are opaque, so they can be read over any artwork.
- `mp4` is in the default preferred formats. An existing installation keeps its saved list; add MP4 under Settings → General.

### Fixed

- Following an author who was added without a catalogue id found none of their books.
- The version in the sidebar was shown with a doubled "v".

## [0.1.0] - 2026-11-15

The first release: follow authors, find their audiobooks on your indexers, download them and file them into a library.

### Added

- **Authors, books and series.** Follow an author to import their catalogue from Audible (one entry per work; translations, abridgements and dramatisations are left out), or add single books. Books are grouped and ordered by series, and new releases are picked up daily.
- **Indexers.** Any Torznab or Newznab feed: Prowlarr, Jackett, NZBHydra or an indexer directly.
- **Download clients.** qBittorrent and SABnzbd.
- **Automatic and interactive search.** Releases are scored on title and author match, preferred format, seeders and a size-versus-runtime check; a release that fails a check is listed with the reason and can still be grabbed by hand.
- **Import.** Finished downloads are hardlinked, copied or moved into `{Author}/{Series}/{Title}` (configurable). Torrents keep seeding. Recognised audio: `m4b`, `mp3`, `m4a`, `flac`, `opus`, `ogg`, `aac`, `wma`, `wav`, `mp4` and `mka`. A failed import says what the download held instead.
- **Library scan.** Audiobooks already on disk are matched to the books you add.
- **Queue, history, wanted list, scheduled tasks and a log viewer.**
- **API.** Everything the UI does is available under `/api/v1` with an `X-Api-Key` header.
- **Container image** `ghcr.io/ncecere/chaptarr` for linux/amd64 and linux/arm64, signed with cosign, with an SBOM and build provenance; and binaries for Linux, macOS and Windows.

[Unreleased]: https://github.com/ncecere/chaptarr/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/ncecere/chaptarr/releases/tag/v0.1.0
