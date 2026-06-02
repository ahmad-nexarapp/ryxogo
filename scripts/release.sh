#!/usr/bin/env bash
# release.sh — tag, push, and create a GitHub Release in one step.
#
# Usage:
#   ./scripts/release.sh v0.3.3 "Short title" "Markdown body / notes"
#
# Requires a GitHub token in the GITHUB_TOKEN env var (do NOT hardcode it):
#   export GITHUB_TOKEN=ghp_xxx
#   ./scripts/release.sh v0.3.3 "New feature" "## What changed ..."
#
# The script:
#   1. Creates an annotated git tag with the notes
#   2. Pushes the tag
#   3. Creates a GitHub Release via the API
#   4. Reminds you to update CHANGELOG.md

set -euo pipefail

TAG="${1:?usage: release.sh <tag> <title> <body>}"
TITLE="${2:?missing title}"
BODY="${3:?missing body}"

REPO="ahmad-nexarapp/ryxogo"

if [ -z "${GITHUB_TOKEN:-}" ]; then
  echo "Error: GITHUB_TOKEN not set. Run: export GITHUB_TOKEN=ghp_xxx"
  exit 1
fi

echo "→ Creating annotated tag $TAG"
git tag -a "$TAG" -m "$TITLE

$BODY"

echo "→ Pushing tag"
git push origin "$TAG"

echo "→ Creating GitHub Release"
PAYLOAD=$(python3 -c "import json,sys; print(json.dumps({
  'tag_name': sys.argv[1],
  'name': sys.argv[1] + ' — ' + sys.argv[2],
  'body': sys.argv[3],
  'draft': False,
  'prerelease': False
}))" "$TAG" "$TITLE" "$BODY")

curl -s -X POST \
  -H "Authorization: token $GITHUB_TOKEN" \
  -H "Accept: application/vnd.github+json" \
  "https://api.github.com/repos/$REPO/releases" \
  -d "$PAYLOAD" \
  | python3 -c "import sys,json; d=json.load(sys.stdin); print('✓ Release:', d.get('html_url', d.get('message','ERROR')))"

echo ""
echo "Don't forget to add a section to CHANGELOG.md for $TAG"
