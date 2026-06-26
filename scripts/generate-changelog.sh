#!/bin/bash

# Script to generate automated changelog from conventional commits
# Usage: 
#   ./scripts/generate-changelog.sh [version]           # For releases (since last tag)
#   ./scripts/generate-changelog.sh [version] [range] # For specific commit range
#   ./scripts/generate-changelog.sh "" [range]         # For PRs (no version header)

set -e

VERSION="${1:-}"
COMMIT_RANGE="${2:-}"
VERSION_FILE="VERSION"

# Determine commit range
if [ -z "$COMMIT_RANGE" ]; then
    if [ -n "$VERSION" ]; then
        # For releases: get commits since previous tag
        PREVIOUS_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
        if [ -n "$PREVIOUS_TAG" ]; then
            COMMIT_RANGE="${PREVIOUS_TAG}..HEAD"
        else
            COMMIT_RANGE="HEAD"
        fi
    else
        # No version and no range: use current HEAD
        if [ -f "$VERSION_FILE" ]; then
            VERSION=$(cat "$VERSION_FILE")
        else
            echo "Error: No version provided and VERSION file not found"
            exit 1
        fi
        PREVIOUS_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
        if [ -n "$PREVIOUS_TAG" ]; then
            COMMIT_RANGE="${PREVIOUS_TAG}..HEAD"
        else
            COMMIT_RANGE="HEAD"
        fi
    fi
fi

# Get commits
if [ "$COMMIT_RANGE" = "HEAD" ]; then
    COMMITS=$(git log --pretty=format:"%s")
else
    COMMITS=$(git log $COMMIT_RANGE --pretty=format:"%s")
fi

# Skip if no commits
if [ -z "$COMMITS" ]; then
    echo "No commits found in range: $COMMIT_RANGE"
    exit 0
fi

# Add version header if version is provided
if [ -n "$VERSION" ]; then
    echo "## [${VERSION}] - $(date +%Y-%m-%d)"
    echo ""
    
    if [ -n "$COMMIT_RANGE" ] && [ "$COMMIT_RANGE" != "HEAD" ]; then
        PREVIOUS_TAG=$(echo $COMMIT_RANGE | cut -d'.' -f1)
        if [ -n "$PREVIOUS_TAG" ]; then
            echo "### Changed since ${PREVIOUS_TAG}"
        fi
    fi
    echo ""
fi

# Categorize commits by type
echo "### Features"
FEATURES=$(echo "$COMMITS" | grep -E "^feat" | sed 's/^feat(\(.*\))?: /- /' || true)
if [ -n "$FEATURES" ]; then
    echo "$FEATURES"
else
    echo "No features"
fi
echo ""

echo "### Bug Fixes"
FIXES=$(echo "$COMMITS" | grep -E "^fix" | sed 's/^fix(\(.*\))?: /- /' || true)
if [ -n "$FIXES" ]; then
    echo "$FIXES"
else
    echo "No bug fixes"
fi
echo ""

echo "### Performance"
PERF=$(echo "$COMMITS" | grep -E "^perf" | sed 's/^perf(\(.*\))?: /- /' || true)
if [ -n "$PERF" ]; then
    echo "$PERF"
else
    echo "No performance improvements"
fi
echo ""

echo "### Breaking Changes"
BREAKING=$(echo "$COMMITS" | grep -E "!" | sed 's/^.*!: /- /' || true)
if [ -n "$BREAKING" ]; then
    echo "$BREAKING"
else
    echo "No breaking changes"
fi
echo ""

echo "### Documentation"
DOCS=$(echo "$COMMITS" | grep -E "^docs" | sed 's/^docs(\(.*\))?: /- /' || true)
if [ -n "$DOCS" ]; then
    echo "$DOCS"
else
    echo "No documentation changes"
fi
echo ""

echo "### Testing"
TESTS=$(echo "$COMMITS" | grep -E "^test" | sed 's/^test(\(.*\))?: /- /' || true)
if [ -n "$TESTS" ]; then
    echo "$TESTS"
else
    echo "No test changes"
fi
echo ""

echo "### Other Changes"
OTHER=$(echo "$COMMITS" | grep -vE "^(feat|fix|perf|docs|style|refactor|test|chore|ci)" | sed 's/^/ - /' || true)
if [ -n "$OTHER" ]; then
    echo "$OTHER"
else
    echo "No other changes"
fi
echo ""