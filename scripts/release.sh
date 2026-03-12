#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:?Usage: $0 <version> (e.g. v1.0.0)}"

if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$ ]]; then
    echo "Error: Version must match format vX.Y.Z or vX.Y.Z-suffix (e.g. v1.0.0, v1.0.0-rc1)"
    exit 1
fi

if git rev-parse "$VERSION" >/dev/null 2>&1; then
    echo "Error: Tag $VERSION already exists"
    exit 1
fi

git tag -a "$VERSION" -m "Release $VERSION"
git push origin "$VERSION"

echo "Tag $VERSION pushed. GitHub Actions will create the release."
