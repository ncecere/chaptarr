#!/usr/bin/env bash
# Refuses a release tag ($TAG) that is not a semantic version, is not on
# main, or has no section in CHANGELOG.md. Run by the `tag` job in
# .github/workflows/ci.yml before anything is built, and by `make release-check`.
set -euo pipefail

fail() {
  echo "::error::check-tag: $*" >&2
  exit 1
}

: "${TAG:?TAG is not set}"

# vMAJOR.MINOR.PATCH, optionally with a pre-release (v1.2.0-rc.1). No build
# metadata: "+" is not valid in an image tag.
semver='^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z]+(\.[0-9A-Za-z]+)*)?$'
[[ "$TAG" =~ $semver ]] || fail "'${TAG}' is not a semantic version (vMAJOR.MINOR.PATCH or vMAJOR.MINOR.PATCH-rc.N)"

base="${TAG%%-*}" # v1.2.0-rc.1 -> v1.2.0
"$(dirname "$0")/changelog-section.sh" "${base#v}" >/dev/null ||
  fail "CHANGELOG.md has no section '## [${base#v}] - YYYY-MM-DD' with content"

# Releases are cut from main. Skipped where origin/main is unknown (a
# shallow clone).
if git rev-parse --verify -q origin/main >/dev/null; then
  git merge-base --is-ancestor "${TAG_COMMIT:-HEAD}" origin/main ||
    fail "${TAG} is not on main"
fi

echo "${TAG}: ok"
