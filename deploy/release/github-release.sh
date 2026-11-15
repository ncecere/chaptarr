#!/usr/bin/env bash
# Creates (or updates) the GitHub Release for $TAG with notes from
# CHANGELOG.md (the base version's section for -rc tags), the image digest
# and verification steps, and attaches the release assets:
#   chaptarr-<tag>-<os>-<arch>.tar.gz|.zip     the binaries (build-binaries.sh),
#                                              read from the directory given as $1
#   chaptarr-<tag>-<os>-<arch>.sbom.spdx.json  the image's SPDX SBOM, per platform
#                                              (from its BuildKit attestations)
#   chaptarr-<tag>.digest.txt                  the image reference by digest
#   checksums.txt                              SHA-256 of the files above
# Run by the `release` job in .github/workflows/ci.yml (needs GH_TOKEN, TAG,
# DIGEST, and read access to the image). Tags with a hyphen (v0.1.0-rc.1)
# become pre-releases. Any failure stops the script with a message saying
# what went wrong.
set -euo pipefail

fail() {
  echo "::error::github-release: $*" >&2
  exit 1
}

: "${TAG:?TAG is not set}" "${DIGEST:?DIGEST is not set}"
assets="${1:?usage: github-release.sh <directory holding the binaries>}"
[[ "$DIGEST" =~ ^sha256:[0-9a-f]{64}$ ]] || fail "DIGEST is not a sha256 digest: '${DIGEST}'"
for cmd in gh docker jq sha256sum; do
  command -v "$cmd" >/dev/null || fail "${cmd} is not installed"
done
here="$(cd "$(dirname "$0")" && pwd)"
assets="$(cd "$assets" && pwd)"
ls "$assets"/chaptarr-"${TAG}"-* >/dev/null 2>&1 || fail "no binaries for ${TAG} in ${assets}"

image=ghcr.io/ncecere/chaptarr
ref="${image}@${DIGEST}"
base="${TAG%%-*}" # v0.1.0-rc.1 -> v0.1.0
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

# ---- assets ------------------------------------------------------------------

# The SBOM. For a multi-platform index, .SBOM maps each platform
# ("linux/amd64") to its attestations; a single-platform image has .SPDX at
# the top. Each platform's SPDX document becomes one file.
sboms="${work}/sboms.json"
docker buildx imagetools inspect "$ref" --format '{{ json .SBOM }}' >"$sboms" ||
  fail "could not read the SBOM attestations of ${ref}"
if jq -e 'type == "object" and has("SPDX")' "$sboms" >/dev/null; then
  jq '{"image": .}' "$sboms" >"${sboms}.tmp" && mv "${sboms}.tmp" "$sboms"
fi
platforms="$(jq -r 'if type == "object" then keys[] else empty end' "$sboms")"
[ -n "$platforms" ] || fail "${ref} has no SBOM attestation (was it built with sbom: true?)"
while IFS= read -r platform; do
  name="chaptarr-${TAG}-${platform//\//-}.sbom.spdx.json" # linux/amd64 -> linux-amd64
  [ "$platform" = image ] && name="chaptarr-${TAG}.sbom.spdx.json"
  jq --arg p "$platform" '.[$p].SPDX' "$sboms" >"${assets}/${name}"
  jq -e '.spdxVersion | type == "string" and startswith("SPDX-")' "${assets}/${name}" >/dev/null ||
    fail "the SBOM for ${platform} is not an SPDX document"
done <<<"$platforms"

echo "$ref" >"${assets}/chaptarr-${TAG}.digest.txt"

rm -f "${assets}/checksums.txt"
(cd "$assets" && sha256sum -- * >"${work}/checksums.txt" && mv "${work}/checksums.txt" checksums.txt)

# ---- notes -------------------------------------------------------------------

notes="${work}/notes.md"
{
  if [ "$TAG" != "$base" ]; then
    echo "> **Release candidate** for ${base}. Test it before relying on it; the final release follows."
    echo
  fi
  "${here}/changelog-section.sh" "${base#v}" || fail "CHANGELOG.md has no section for ${base#v}"
  echo
  echo "## Image"
  echo
  echo '```'
  echo "${image}:${TAG}"
  echo "$ref"
  echo '```'
  echo
  echo "Verify the signature (keyless, GitHub Actions OIDC):"
  echo
  echo '```sh'
  echo "cosign verify ${ref} \\"
  echo "  --certificate-identity-regexp '^https://github.com/ncecere/chaptarr/\\.github/workflows/.+@refs/tags/${TAG}\$' \\"
  echo "  --certificate-oidc-issuer https://token.actions.githubusercontent.com"
  echo '```'
  echo
  echo "## Release assets"
  echo
  echo "- \`chaptarr-${TAG}-<os>-<arch>.tar.gz\` (\`.zip\` for Windows): the binary, with the UI embedded."
  echo "- \`chaptarr-${TAG}-<os>-<arch>.sbom.spdx.json\`: the image's SBOM (SPDX JSON) for each platform."
  echo "- \`chaptarr-${TAG}.digest.txt\`: the image reference by digest."
  echo "- \`checksums.txt\`: SHA-256 of the files above (\`sha256sum -c checksums.txt\`)."
} >"$notes"

# ---- release -----------------------------------------------------------------

repo="${GITHUB_REPOSITORY:-$(gh repo view --json nameWithOwner -q .nameWithOwner)}" || fail "could not tell the repository (set GITHUB_REPOSITORY)"
flags=(--repo "$repo" --title "Chaptarr ${TAG}" --notes-file "$notes" --verify-tag)
[ "$TAG" != "$base" ] && flags+=(--prerelease)
if gh release view "$TAG" --repo "$repo" >/dev/null 2>&1; then
  gh release edit "$TAG" --repo "$repo" --title "Chaptarr ${TAG}" --notes-file "$notes" || fail "could not update the release ${TAG}"
else
  gh release create "$TAG" "${flags[@]}" || fail "could not create the release ${TAG}"
fi
(cd "$assets" && gh release upload "$TAG" --repo "$repo" --clobber -- *) || fail "could not upload the release assets"
echo "release ${TAG}: ${ref}"
ls -l "$assets"
