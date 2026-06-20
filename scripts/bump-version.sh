#!/bin/bash

# Script to bump version based on commit type
# Usage: ./scripts/bump-version.sh [major|minor|patch]

set -e

VERSION_FILE="VERSION"

if [ ! -f "$VERSION_FILE" ]; then
    echo "Error: VERSION file not found"
    exit 1
fi

CURRENT_VERSION=$(cat "$VERSION_FILE")
IFS='.' read -r MAJOR MINOR PATCH <<< "$CURRENT_VERSION"

BUMP_TYPE="${1:-patch}"

case "$BUMP_TYPE" in
    major)
        MAJOR=$((MAJOR + 1))
        MINOR=0
        PATCH=0
        ;;
    minor)
        MINOR=$((MINOR + 1))
        PATCH=0
        ;;
    patch)
        PATCH=$((PATCH + 1))
        ;;
    *)
        echo "Invalid bump type: $BUMP_TYPE"
        echo "Usage: $0 [major|minor|patch]"
        exit 1
        ;;
esac

NEW_VERSION="$MAJOR.$MINOR.$PATCH"

echo "Bumping version from $CURRENT_VERSION to $NEW_VERSION"

# Update VERSION file
echo "$NEW_VERSION" > "$VERSION_FILE"

# Update version.go
if [ -f "internal/version/version.go" ]; then
    sed -i "s/const Version = .*/const Version = \"$NEW_VERSION\"/" internal/version/version.go
fi

# Update go.mod if version is referenced there
if grep -q "version = " go.mod 2>/dev/null; then
    sed -i "s/version = .*/version = $NEW_VERSION/" go.mod
fi

echo "Version updated to $NEW_VERSION"
