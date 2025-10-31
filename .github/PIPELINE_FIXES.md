# GitHub Actions Pipeline Fixes

## Issues Fixed

### 1. TypeScript Build Errors ✅ FIXED
- **Files**: `admin/src/routes/__root.tsx`, `admin/src/routes/_authenticated/settings/index.tsx`
- **Changes**:
  - Removed unused imports
  - Fixed deprecated React Query v5 API (`onSuccess` → `useEffect`)
  - Added proper TypeScript types

### 2. Go Version Mismatch ✅ FIXED
- **Issue**: Go 1.25 required by dependencies but workflows used 1.22/1.23
- **Files Updated**:
  - `go.mod` → `go 1.25`
  - `Dockerfile` → `golang:1.25-alpine`
  - `.devcontainer/Dockerfile` → `golang:1.25-bookworm`
  - `.github/workflows/ci.yml` → `GO_VERSION: "1.25"`
  - `.github/workflows/release.yml` → `go-version: "1.25"`

### 3. DevContainer Path Issues ✅ FIXED
- **File**: `.devcontainer/docker-compose.yml`
- **Change**: Updated path from `../example/create_tables.sql` to `../examples/sql-scripts/create_tables.sql`

### 4. GitHub Actions Secrets Access ✅ FIXED
- **File**: `.github/workflows/release.yml`
- **Issue**: Cannot access `secrets` directly in `if` conditions
- **Solution**: Added `dockerhub-check` step that outputs a boolean flag

### 5. Release Please Configuration ✅ FIXED
- **File**: `.github/workflows/release.yml`
- **Issue**: `package-name` is not a valid parameter in release-please v4
- **Solution**: Removed `package-name: fluxbase` parameter

### 6. CI Build Directory Creation ✅ FIXED
- **File**: `.github/workflows/ci.yml`
- **Issue**: `dist/` directory not created before build
- **Solution**: Added `mkdir -p dist` before build command

## Release Workflow Updated ✅

**Changed from Release Please to Manual Releases**

The release workflow now triggers only when you manually create a release or push a version tag:

- ✅ **Automatic CI builds** on every push to main
- ✅ **Automatic Docker images** pushed to GHCR on every push
- ✅ **Manual releases** - you control when versions are published

### How to Create a Release

**Option 1: Using GitHub UI**
1. Go to https://github.com/wayli-app/fluxbase/releases/new
2. Click "Choose a tag" → Type new version (e.g., `v0.1.0`)
3. Write release notes
4. Click "Publish release"

**Option 2: Using Git Tags**
```bash
git tag v0.1.0
git push origin v0.1.0
```

**Option 3: Using GitHub CLI**
```bash
gh release create v0.1.0 --title "Release 0.1.0" --notes "Release notes here"
```

When you create a release, the workflow will automatically:
- Build binaries for all platforms
- Build and push Docker images with version tags
- Publish Go module to pkg.go.dev
- Publish NPM packages (if configured)
- Publish Helm charts (if configured)

## Verification

After enabling permissions or adding the PAT:

1. **Test Release Please**:
   ```bash
   git commit -m "feat: test conventional commit"
   git push
   ```
   - Should create a Release PR

2. **Test Docker Build**:
   ```bash
   make docker-build
   ```
   - Should build successfully

3. **Test DevContainer**:
   - Rebuild container in VS Code
   - Should start without errors

## Commit Message Format

Release Please requires [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add new feature (minor version bump)
fix: bug fix (patch version bump)
feat!: breaking change (major version bump)
docs: documentation only
chore: maintenance tasks
```

## Current Status

- ✅ Docker build: **WORKING** (`make docker-build` succeeds)
- ✅ TypeScript compilation: **WORKING** (admin UI builds)
- ✅ Go compilation: **WORKING** (Go 1.25 everywhere)
- ⚠️ Release Please: **NEEDS PERMISSION** (see above)
- ⚠️ CI Workflow: May have syntax issue - checking logs

## Files Changed

```
Modified:
- .github/workflows/ci.yml
- .github/workflows/release.yml
- .devcontainer/Dockerfile
- .devcontainer/docker-compose.yml
- Dockerfile
- go.mod
- admin/src/routes/__root.tsx
- admin/src/routes/_authenticated/settings/index.tsx

Created:
- .github/RELEASE_SETUP.md
- .github/PIPELINE_FIXES.md (this file)
```
