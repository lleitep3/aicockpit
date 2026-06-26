#!/bin/bash

# Script to generate automated changelog from conventional commits
# Usage: ./scripts/generate-changelog.sh [version]

set -e

VERSION="${1:-}"
VERSION_FILE="VERSION"

if [ -z "$VERSION" ]; then
    if [ -f "$VERSION_FILE" ]; then
        VERSION=$(cat "$VERSION_FILE")
    else
        echo "Error: No version provided and VERSION file not found"
        exit 1
    fi
fi

# Get the previous tag
PREVIOUS_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")

echo "## [${VERSION}] - $(date +%Y-%m-%d)"
echo ""

if [ -n "$PREVIOUS_TAG" ]; then
    echo "### Changed since ${PREVIOUS_TAG}"
else
    echo "### Initial Release"
fi
echo ""

# Get commits since previous tag (or all commits if no tag)
if [ -n "$PREVIOUS_TAG" ]; then
    COMMITS=$(git log ${PREVIOUS_TAG}..HEAD --pretty=format:"%s")
else
    COMMITS=$(git log --pretty=format:"%s")
fi

# Categorize commits by type
echo "### Features"
echo "$COMMITS" | grep -E "^feat" | sed 's/^feat(\(.*\))?: /- /' || echo "No features"
echo ""

echo "### Bug Fixes"
echo "$COMMITS" | grep -E "^fix" | sed 's/^fix(\(.*\))?: /- /' || echo "No bug fixes"
echo ""

echo "### Performance"
echo "$COMMITS" | grep -E "^perf" | sed 's/^perf(\(.*\))?: /- /' || echo "No performance improvements"
echo ""

echo "### Breaking Changes"
echo "$COMMITS" | grep -E "!" | sed 's/^.*!: /- /' || echo "No breaking changes"
echo ""

echo "### Other Changes"
echo "$COMMITS" | grep -vE "^(feat|fix|perf|docs|style|refactor|test|chore|ci)" | sed 's/^/ - /' || echo "No other changes"
echo ""

echo "### Documentation"
echo "$COMMITS" | grep -E "^docs" | sed 's/^docs(\(.*\))?: /- /' || echo "No documentation changes"
echo ""

echo "### Testing"
echo "$COMMITS" | grep -E "^test" | sed 's/^test(\(.*\))?: /- /' || echo "No test changes"
echo ""