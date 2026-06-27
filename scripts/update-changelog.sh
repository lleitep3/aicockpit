#!/bin/bash

# Update CHANGELOG.md with unreleased changes since the latest tag.
# Usage: scripts/update-changelog.sh [--dry-run | --pr]

set -euo pipefail

DRY_RUN=false
PR_MODE=false
while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run)
      DRY_RUN=true
      shift
      ;;
    --pr)
      PR_MODE=true
      shift
      ;;
    *)
      echo "Unknown option: $1" >&2
      exit 1
      ;;
  esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
if [ -z "$LATEST_TAG" ]; then
  echo "No tags found, skipping changelog update"
  exit 0
fi

LATEST_COMMIT=$(git log -1 --pretty=format:'%s')
if [[ "$LATEST_COMMIT" =~ ^docs\(changelog\):.*\[skip\ ci\] ]]; then
  echo "Latest commit is already a changelog update, skipping"
  exit 0
fi

NEW_CHANGES=$(mktemp)
bash "${SCRIPT_DIR}/generate-changelog.sh" "" "${LATEST_TAG}..HEAD" > "$NEW_CHANGES"

if [ ! -s "$NEW_CHANGES" ] || grep -q "No commits found" "$NEW_CHANGES"; then
  echo "No new commits since ${LATEST_TAG}, skipping changelog update"
  rm -f "$NEW_CHANGES"
  exit 0
fi

HISTORICAL=$(mktemp)
if [ -f CHANGELOG.md ]; then
  HISTORICAL_START=$(grep -n "^## \[[0-9]" CHANGELOG.md | head -n 1 | cut -d: -f1 || true)
  if [ -n "$HISTORICAL_START" ]; then
    tail -n +"${HISTORICAL_START}" CHANGELOG.md > "$HISTORICAL"
  else
    tail -n +8 CHANGELOG.md > "$HISTORICAL"
  fi
else
  : > "$HISTORICAL"
fi

OUTPUT=$(mktemp)
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
  cat "$NEW_CHANGES"
  if [ -s "$HISTORICAL" ]; then
    echo ""
    cat "$HISTORICAL"
  fi
} > "$OUTPUT"

if [ "$DRY_RUN" = true ]; then
  echo "Dry-run: generated CHANGELOG.md would be:"
  cat "$OUTPUT"
  rm -f "$NEW_CHANGES" "$HISTORICAL" "$OUTPUT"
  exit 0
fi

if [ "$PR_MODE" = true ]; then
  BRANCH="chore/changelog-update-$(date +%Y%m%d%H%M%S)"
  git checkout -b "$BRANCH"
fi

cat "$OUTPUT" > CHANGELOG.md

git config user.name "github-actions[bot]"
git config user.email "github-actions[bot]@users.noreply.github.com"
git add CHANGELOG.md
git commit -m "docs(changelog): update CHANGELOG.md for changes since ${LATEST_TAG} [skip ci]"

if [ "$PR_MODE" = true ]; then
  git push origin "$BRANCH"
  PR_URL=$(gh pr create --base main --title "docs(changelog): update CHANGELOG.md for changes since ${LATEST_TAG} [skip ci]" --body "Automated changelog update.")
  PR_NUMBER=$(echo "${PR_URL}" | sed 's/.*\/pull\/\([0-9]*\).*/\1/')
  echo "Created PR #${PR_NUMBER}: ${PR_URL}"

  for i in $(seq 1 120); do
    STATUS=$(gh pr view "${PR_NUMBER}" --json mergeStateStatus --jq '.mergeStateStatus')
    echo "PR #${PR_NUMBER} status: ${STATUS}"
    if [ "${STATUS}" = "CLEAN" ]; then
      gh pr merge "${PR_NUMBER}" --squash --delete-branch
      echo "Merged PR #${PR_NUMBER}"
      break
    fi
    sleep 10
  done

  if [ "${STATUS}" != "CLEAN" ]; then
    echo "Timeout waiting for PR #${PR_NUMBER} to become mergeable" >&2
    exit 1
  fi
else
  git push origin main
fi

rm -f "$NEW_CHANGES" "$HISTORICAL" "$OUTPUT"
