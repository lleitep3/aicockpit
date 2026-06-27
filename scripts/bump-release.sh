#!/bin/bash

# Bump version, update CHANGELOG.md with a versioned release section, and create a tag.
# Usage: scripts/bump-release.sh [--dry-run] [--version X.Y.Z]

set -euo pipefail

DRY_RUN=false
VERSION=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run)
      DRY_RUN=true
      shift
      ;;
    --version)
      VERSION="$2"
      shift 2
      ;;
    *)
      echo "Unknown option: $1" >&2
      exit 1
      ;;
  esac
done

LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
if [ -z "$LATEST_TAG" ]; then
  echo "No tags found, cannot create release" >&2
  exit 1
fi

if [ -z "$VERSION" ]; then
  if [ ! -f VERSION ]; then
    echo "VERSION file not found and no version provided" >&2
    exit 1
  fi
  CURRENT_VERSION=$(cat VERSION)
  VERSION=$(awk -F. '{$NF += 1}1' OFS=. <<< "$CURRENT_VERSION")
  echo "Auto-bumped version: ${VERSION}"
fi

UNRELEASED=$(mktemp)
if [ -f CHANGELOG.md ]; then
  awk '/^## \[Unreleased\]/{flag=1; next} flag && /^## \[[0-9]/{flag=0} flag{print}' CHANGELOG.md > "$UNRELEASED" || true
else
  : > "$UNRELEASED"
fi

if ! grep -qE '^- ' "$UNRELEASED"; then
  echo "No unreleased changes found in CHANGELOG.md, skipping release"
  rm -f "$UNRELEASED"
  exit 0
fi

HISTORICAL=$(mktemp)
if [ -f CHANGELOG.md ]; then
  awk '/^## \[[0-9]/{flag=1} flag{print}' CHANGELOG.md > "$HISTORICAL" || true
else
  : > "$HISTORICAL"
fi

NEW_CL=$(mktemp)
RELEASE_NOTES=$(mktemp)
RELEASE_DATE=$(date +%Y-%m-%d)

{
  echo "# Changelog"
  echo ""
  echo "All notable changes to this project will be documented in this file."
  echo ""
  echo "The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),"
  echo "and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html)."
  echo ""
  echo "## [Unreleased]"
  echo ""
  echo "## [${VERSION}] - ${RELEASE_DATE}"
  echo ""
  echo "### Changed since ${LATEST_TAG}"
  echo ""
  cat "$UNRELEASED"
  if [ -s "$HISTORICAL" ]; then
    echo ""
    cat "$HISTORICAL"
  fi
} > "$NEW_CL"

{
  echo "## [${VERSION}] - ${RELEASE_DATE}"
  echo ""
  echo "### Changed since ${LATEST_TAG}"
  echo ""
  cat "$UNRELEASED"
} > "$RELEASE_NOTES"

if [ "$DRY_RUN" = true ]; then
  echo "Dry-run: generated CHANGELOG.md would be:"
  cat "$NEW_CL"
  echo ""
  echo "Dry-run: generated RELEASE_NOTES.md would be:"
  cat "$RELEASE_NOTES"
  echo ""
  echo "Dry-run: version would be ${VERSION}"
  rm -f "$UNRELEASED" "$HISTORICAL" "$NEW_CL" "$RELEASE_NOTES"
  exit 0
fi

echo "${VERSION}" > VERSION
if [ -f internal/version/version.go ]; then
  sed -i "s/const Version = .*/const Version = \"${VERSION}\"/" internal/version/version.go
fi

cat "$NEW_CL" > CHANGELOG.md
cat "$RELEASE_NOTES" > RELEASE_NOTES.md

git config user.name "github-actions[bot]"
git config user.email "github-actions[bot]@users.noreply.github.com"
git add VERSION internal/version/version.go CHANGELOG.md
git commit -m "chore(release): bump version and update CHANGELOG.md to v${VERSION} [skip ci]"
git tag -a "v${VERSION}" -m "Release v${VERSION}"
git push origin main
git push origin "v${VERSION}"

if [ -n "${GITHUB_OUTPUT:-}" ]; then
  echo "tag_name=v${VERSION}" >> "$GITHUB_OUTPUT"
  echo "release_name=Release v${VERSION}" >> "$GITHUB_OUTPUT"
fi

rm -f "$UNRELEASED" "$HISTORICAL" "$NEW_CL" "$RELEASE_NOTES"
