#!/usr/bin/env bash
set -euo pipefail

RETRY=false
if [[ "${1:-}" == "--retry" ]]; then
    RETRY=true
    shift
fi

VERSION="${1:?Usage: $0 [--retry] <version> (e.g. v1.0.0)}"

if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$ ]]; then
    echo "Error: Version must match format vX.Y.Z or vX.Y.Z-suffix (e.g. v1.0.0, v1.0.0-rc1)"
    exit 1
fi

if git rev-parse "$VERSION" >/dev/null 2>&1; then
    if [ "$RETRY" = true ]; then
        echo "Deleting existing release and tag $VERSION..."
        gh release delete "$VERSION" --yes --cleanup-tag 2>/dev/null || true
        git tag -d "$VERSION" 2>/dev/null || true
        git push origin ":refs/tags/$VERSION" 2>/dev/null || true
    else
        echo "Error: Tag $VERSION already exists. Use --retry to delete and recreate."
        exit 1
    fi
fi

git tag -a "$VERSION" -m "Release $VERSION"
git push origin "$VERSION"

echo "Tag $VERSION pushed. GitHub Actions will create the release."
