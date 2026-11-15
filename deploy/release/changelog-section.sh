#!/usr/bin/env bash
# Prints the body of one version's section of CHANGELOG.md, e.g.
#   deploy/release/changelog-section.sh 0.2.0
# Exits 1 if the section is missing, undated or empty.
set -euo pipefail

version="${1:?usage: changelog-section.sh X.Y.Z}"
file="$(dirname "$0")/../../CHANGELOG.md"

body="$(awk -v v="$version" '
  /^## \[/ {
    if (found) exit
    if (index($0, "## [" v "] - ") == 1) { found = 1; next }
  }
  /^\[[^]]+\]: / { if (found) exit }   # the link references at the end
  found { print }
' "$file")"

# Trim blank lines at both ends.
body="$(printf '%s\n' "$body" | awk 'NF { for (i = 0; i < blank; i++) print ""; blank = 0; print; seen = 1; next } seen { blank++ }')"
[ -n "$body" ] || exit 1
printf '%s\n' "$body"
