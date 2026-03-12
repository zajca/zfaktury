# Release Process

ZFaktury uses GitHub Actions to automatically build cross-platform binaries and create GitHub Releases when a version tag is pushed.

## Quick Release

```bash
./scripts/release.sh v1.0.0
```

This validates the tag format, creates an annotated git tag, and pushes it to origin. GitHub Actions takes over from there.

## Manual Release

```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

## Pre-releases

Tags containing a hyphen (e.g. `v1.0.0-rc1`, `v2.0.0-beta.1`) are automatically marked as pre-release on GitHub.

```bash
./scripts/release.sh v1.0.0-rc1
```

## What Happens on Tag Push

The release workflow (`.github/workflows/release.yml`) runs 3 jobs:

1. **build-frontend** - Builds the SvelteKit frontend (`npm ci && npm run build`)
2. **build-binaries** - Cross-compiles Go binaries for 5 targets using the frontend artifact
3. **release** - Creates a GitHub Release with binaries, changelog, and SHA256 checksums

## Target Platforms

| OS | Arch | Archive |
|----|------|---------|
| Linux | amd64 | `.tar.gz` |
| Linux | arm64 | `.tar.gz` |
| macOS | amd64 | `.tar.gz` |
| macOS | arm64 | `.tar.gz` |
| Windows | amd64 | `.zip` |

## Version Info

Version, commit hash, and build date are injected at build time via Go ldflags into `internal/version/version.go`.

```bash
# Dev build (no ldflags)
./zfaktury --version
# zfaktury version dev (none, unknown)

# Release build
./zfaktury --version
# zfaktury version v1.0.0 (abc1234, 2026-03-12T10:00:00Z)
```

The Makefile also supports ldflags for local builds:

```bash
make build VERSION=v1.0.0
```

## Tag Format

Tags must match `vX.Y.Z` or `vX.Y.Z-suffix` (e.g. `v1.0.0`, `v2.1.0-rc1`, `v0.5.0-beta.2`). The release script validates this before creating the tag.
