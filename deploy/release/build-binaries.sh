#!/usr/bin/env bash
# Cross-compiles the release binaries for $TAG into the directory given as
# the first argument, one archive per platform:
#   chaptarr-<tag>-<os>-<arch>.tar.gz   (.zip for Windows)
# web/dist must already be built (`make web`): the UI is embedded.
set -euo pipefail

: "${TAG:?TAG is not set}"
out="${1:?usage: build-binaries.sh <output directory>}"
commit="${COMMIT:-$(git rev-parse HEAD)}"
[ -f web/dist/index.html ] || { echo "::error::web/dist is not built (run make web)" >&2; exit 1; }

mkdir -p "$out"
out="$(cd "$out" && pwd)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64; do
  os="${target%/*}" arch="${target#*/}"
  name="chaptarr-${TAG}-${os}-${arch}"
  dir="${work}/${name}"
  mkdir -p "$dir"
  bin=chaptarr
  [ "$os" = windows ] && bin=chaptarr.exe
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -trimpath \
    -ldflags "-s -w -X main.version=${TAG} -X main.commit=${commit}" \
    -o "${dir}/${bin}" ./cmd/chaptarr
  cp README.md CHANGELOG.md "$dir/"
  if [ "$os" = windows ]; then
    (cd "$work" && zip -qr "${out}/${name}.zip" "$name")
  else
    tar -C "$work" -czf "${out}/${name}.tar.gz" "$name"
  fi
  echo "built ${name}"
done
