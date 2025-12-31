# Fluxbase Versioning & Release Strategy

This document describes the versioning, building, and release automation strategy for Fluxbase.

## Table of Contents

- [Overview](#overview)
- [Version Management](#version-management)
- [Automated Build Process](#automated-build-process)
- [Release Workflow](#release-workflow)
- [Component Versioning](#component-versioning)
- [Local Development](#local-development)
- [CI/CD Integration](#cicd-integration)

---

## Overview

Fluxbase uses **Semantic Versioning (SemVer)** across all components:

```
MAJOR.MINOR.PATCH (e.g., 1.2.3)
```

- **MAJOR**: Breaking changes
- **MINOR**: New features (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

**Single Source of Truth**: The [VERSION](VERSION) file at the root of the repository.

---

## Version Management

### VERSION File

The `/VERSION` file contains the current version number:

```
0.1.0
```

This file is:

- ✅ **Read by**: Makefile, GitHub Actions, Docker builds, Helm charts
- ✅ **Updated by**: `make bump-*` commands or manually
- ✅ **Tracked in**: Git (committed to main branch)

### Version Information in Binary

Every Fluxbase binary includes embedded version metadata:

```bash
./fluxbase --version
# Output:
# Fluxbase 0.1.0
# Commit: abc1234
# Build Date: 2025-10-31T14:30:00Z
```

This is achieved via Go `ldflags` during compilation:

```go
var (
    Version   = "dev"       // Set via -X main.Version
    Commit    = "unknown"   // Set via -X main.Commit
    BuildDate = "unknown"   // Set via -X main.BuildDate
)
```

---

## Automated Build Process

### Building Locally

#### Quick Build (Development)

```bash
# Build with version information
make build

# Output: dist/fluxbase-linux-amd64
```

#### Production Build (With Admin UI)

```bash
# Build production binary with embedded admin UI
make build

# Or build production Docker image
make docker-build-production
```

### Build Variables

All builds automatically inject version information:

| Variable     | Source                          | Example                |
| ------------ | ------------------------------- | ---------------------- |
| `VERSION`    | [VERSION](VERSION) file         | `0.1.0`                |
| `COMMIT`     | `git rev-parse --short HEAD`    | `abc1234`              |
| `BUILD_DATE` | `date -u +"%Y-%m-%dT%H:%M:%SZ"` | `2025-10-31T14:30:00Z` |

---

## Release Workflow

### Manual Release Process

#### 1. Bump Version

```bash
# Patch release (0.1.0 -> 0.1.1)
make bump-patch

# Minor release (0.1.0 -> 0.2.0)
make bump-minor

# Major release (0.1.0 -> 1.0.0)
make bump-major
```

This updates the `VERSION` file.

#### 2. Commit and Push Version Bump

```bash
git add VERSION
git commit -m "chore: bump version to $(cat VERSION)"
git push origin main
```

#### 3. Create Release

```bash
# Run full release process:
# - Run tests
# - Build binaries
# - Build Docker image
# - Push Docker image
# - Create git tag
make release
```

Or do it step-by-step:

```bash
# 1. Test
make test

# 2. Build production image
make docker-build-production

# 3. Push to registry
make docker-push

# 4. Create and push git tag
make release-tag
```

### Automated Release (via GitHub Actions)

The project uses [Release Please](https://github.com/googleapis/release-please-action) for automated releases.

**When you merge to `main`**:

1. Release Please creates a PR with changelog and version bump
2. When you merge the Release PR:
   - Binaries are built for all platforms
   - Docker images are pushed to `ghcr.io`
   - TypeScript SDK is published to NPM
   - Go module is updated on pkg.go.dev
   - GitHub release is created with artifacts

**Triggering a release**:

```bash
# Use conventional commit messages
git commit -m "feat: add new API endpoint"      # Minor version bump
git commit -m "fix: resolve authentication bug" # Patch version bump
git commit -m "feat!: breaking API change"      # Major version bump
```

See [.github/workflows/release.yml](.github/workflows/release.yml) for details.

---

## Component Versioning

### 1. Docker Images

**Registry**: `ghcr.io/fluxbase-eu/fluxbase`

**Tags**:

- `latest` - Latest stable release
- `0.1.0` - Specific version (SemVer)
- `0.1` - Latest patch of minor version
- `0` - Latest minor of major version

**Building**:

```bash
# Build and tag
make docker-build-production

# Push to registry
make docker-push
```

**Using a specific version**:

```bash
# docker-compose
export FLUXBASE_VERSION=0.1.0
docker compose -f deploy/docker-compose.production.yml up

# Docker CLI
docker pull ghcr.io/fluxbase-eu/fluxbase:latest
docker run ghcr.io/fluxbase-eu/fluxbase:latest
```

### 2. Helm Chart

**Registries**:

- **OCI Registry**: `oci://ghcr.io/wayli-app/charts/fluxbase`
- **HTTP Repository**: `https://wayli-app.github.io/fluxbase` (gh-pages)

**Chart.yaml**:

```yaml
version: 0.1.0 # Chart version (auto-updated on release)
appVersion: "0.1.0" # Fluxbase version (auto-updated on release)
```

**Installing from OCI registry** (Recommended):

```bash
# Add Helm repository
helm registry login ghcr.io

# Install latest
helm install fluxbase oci://ghcr.io/wayli-app/charts/fluxbase

# Install specific version
helm install fluxbase oci://ghcr.io/wayli-app/charts/fluxbase --version 0.1.0
```

**Installing from HTTP repository**:

```bash
# Add Helm repository
helm repo add fluxbase https://wayli-app.github.io/fluxbase
helm repo update

# Install latest
helm install fluxbase fluxbase/fluxbase

# Install specific version
helm install fluxbase fluxbase/fluxbase --version 0.1.0
```

**Installing from local chart**:

```bash
# Install from local directory
helm install fluxbase deploy/helm/fluxbase \
  --set image.tag=0.1.0
```

The Helm chart automatically uses `appVersion` from `Chart.yaml` if no `image.tag` is specified.

### 3. TypeScript SDK

**Package**: `@fluxbase/sdk` (NPM)

**Versioning**: Synced with Fluxbase core version

**Publishing** (automated via GitHub Actions):

```bash
# On release, GitHub Actions runs:
cd sdk/typescript
npm version $VERSION --no-git-tag-version
npm publish --access public
```

**Using in projects**:

```bash
npm install @fluxbase/sdk@0.1.0
```

### 4. Documentation

**Docs version** matches Fluxbase version. Each release creates a versioned documentation snapshot.

**Building docs**:

```bash
make docs-build
```

---

## Local Development

### Development Builds (No Versioning)

```bash
# Quick dev build (version=dev)
go run cmd/fluxbase/main.go

# Or use air for hot reload
make dev
```

### Development with Version

```bash
# Build with custom version
make build VERSION=0.2.0-dev

# Or set via environment
export VERSION=0.2.0-rc1
make build
```

---

## CI/CD Integration

### GitHub Actions

#### CI Workflow (`.github/workflows/ci.yml`)

Runs on every push/PR:

- ✅ Linting
- ✅ Tests with coverage
- ✅ Multi-platform builds (Linux, macOS)
- ✅ Docker image build

**Version injection**:

```yaml
- name: Build binary
  run: |
    go build -ldflags="-X main.Version=${GITHUB_SHA::8}" \
      -o fluxbase cmd/fluxbase/main.go
```

#### Release Workflow (`.github/workflows/release.yml`)

Triggered by Release Please:

- ✅ Builds binaries for all platforms
- ✅ Creates checksums
- ✅ Builds and pushes Docker images
- ✅ Publishes TypeScript SDK to NPM
- ✅ Updates Go module on pkg.go.dev
- ✅ Creates GitHub release

**Version from Release Please**:

```yaml
VERSION=${{ needs.release-please.outputs.version }}
```

---

## Best Practices

### Version Bumping

1. **Development**: Work on `develop` branch or feature branches
2. **Pre-release**: Use pre-release versions like `0.2.0-rc1`, `1.0.0-beta2`
3. **Release**: Merge to `main` with conventional commits
4. **Post-release**: Release Please creates PR → Merge to trigger release

### Git Tags

Tags follow the format `v{VERSION}`:

```
v0.1.0
v0.2.0
v1.0.0
```

**Creating tags**:

```bash
# Automated (via make release)
make release-tag

# Manual
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

### Docker Image Tags

Always tag images with:

- **Specific version**: `0.1.0` (immutable, recommended for production)
- **Latest**: `latest` (mutable, for testing)

**Avoid** using `latest` in production. Always pin to a specific version:

```yaml
# ❌ Bad
image: ghcr.io/fluxbase-eu/fluxbase:latest

# ✅ Good
image: ghcr.io/fluxbase-eu/fluxbase:0.1.0
```

### Helm Chart Updates

Update `Chart.yaml` when:

- **Chart version**: Changes to Helm templates or values
- **App version**: New Fluxbase release

```yaml
# Helm template change (bump chart version)
version: 0.1.1
appVersion: "0.1.0"

# Fluxbase release (bump both)
version: 0.2.0
appVersion: "0.2.0"
```

---

## Troubleshooting

### Version Mismatch

**Problem**: Binary reports wrong version

```bash
./fluxbase --version
# Fluxbase dev
```

**Solution**: Rebuild with version information:

```bash
make build
# or
go build -ldflags="-X main.Version=$(cat VERSION)" cmd/fluxbase/main.go
```

### Docker Image Not Found

**Problem**: `docker pull ghcr.io/fluxbase-eu/fluxbase:0.1.0` fails

**Solution**: Check if the version exists:

```bash
# List available tags
gh api /orgs/wayli-app/packages/container/fluxbase/versions

# Or build locally
make docker-build-production
```

### Helm Chart Version Conflict

**Problem**: Helm using wrong image version

**Solution**: Explicitly set image tag:

```bash
helm install fluxbase deploy/helm/fluxbase --set image.tag=0.1.0
```

---

## Summary

| Component           | Version Source | Update Method                  |
| ------------------- | -------------- | ------------------------------ |
| **Fluxbase Binary** | `VERSION` file | `make bump-*`                  |
| **Docker Image**    | `VERSION` file | `make docker-build-production` |
| **Helm Chart**      | `Chart.yaml`   | Manual edit                    |
| **TypeScript SDK**  | `VERSION` file | GitHub Actions                 |
| **Go Module**       | Git tags       | `make release-tag`             |

**Key Commands**:

```bash
# Show current version
make version

# Bump version
make bump-patch   # or bump-minor, bump-major

# Build with version
make build

# Create release
make release

# Build Docker image
make docker-build-production

# Push to registry
make docker-push
```

---

## References

- [Semantic Versioning](https://semver.org/)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [Release Please](https://github.com/googleapis/release-please-action)
- [Docker Multi-Stage Builds](https://docs.docker.com/build/building/multi-stage/)
- [Helm Chart Versioning](https://helm.sh/docs/topics/charts/)
