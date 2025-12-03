# Complete GitHub Repository Setup Guide

This guide walks you through setting up your GitHub repository for automated releases, including all required secrets, variables, and permissions.

---

## Overview

Fluxbase uses GitHub Actions to automatically:

- Build binaries for multiple platforms
- Build and push Docker images to GHCR (and optionally Docker Hub)
- Package and publish Helm charts to GHCR and gh-pages
- Publish TypeScript SDK to NPM
- Create GitHub releases with all artifacts

---

## Table of Contents

1. [Secrets Configuration](#secrets-configuration)
2. [Variables Configuration](#variables-configuration)
3. [Repository Settings](#repository-settings)
4. [GitHub Pages Setup](#github-pages-setup)
5. [Testing the Setup](#testing-the-setup)

---

## Secrets Configuration

### Navigation

Go to: **Settings** → **Secrets and variables** → **Actions** → **Secrets** tab

### Required Secrets

#### 1. NPM_TOKEN ✅ Required

**Purpose**: Publish TypeScript SDK (`@fluxbase/sdk`) to NPM registry

**How to get it**:

1. Log in to [npmjs.com](https://www.npmjs.com/)
2. Click your profile → **Access Tokens**
3. Click **Generate New Token** → **Classic Token**
4. Select **Automation** type (for CI/CD usage)
5. Copy the token (starts with `npm_`)

**Add to GitHub**:

```
Name: NPM_TOKEN
Secret: npm_xxxxxxxxxxxxxxxxxxxxxxxx
```

**Scope needed**: Publish access to `@fluxbase` organization (or your org)

---

### Optional Secrets (Docker Hub)

These are **only needed if you want to publish to Docker Hub in addition to GitHub Container Registry (ghcr.io)**.

If you don't add these secrets, images will only be published to ghcr.io (which is free and works great).

#### 2. DOCKERHUB_USERNAME ❌ Optional

**Purpose**: Authenticate with Docker Hub to push images

**How to get it**:

- This is your Docker Hub username (e.g., `wayliapp`, `mycompany`, etc.)

**Add to GitHub**:

```
Name: DOCKERHUB_USERNAME
Secret: your-dockerhub-username
```

#### 3. DOCKERHUB_TOKEN ❌ Optional

**Purpose**: Docker Hub access token for authentication

**How to get it**:

1. Log in to [hub.docker.com](https://hub.docker.com/)
2. Go to **Account Settings** → **Security**
3. Click **New Access Token**
4. Name: `Fluxbase GitHub Actions`
5. Permissions: **Read, Write, Delete**
6. Copy the token (starts with `dckr_pat_`)

**Add to GitHub**:

```
Name: DOCKERHUB_TOKEN
Secret: dckr_pat_xxxxxxxxxxxxxxxxxxxxxxxx
```

---

### Automatic Secrets (No Action Needed)

#### GITHUB_TOKEN ✅ Automatic

**Purpose**: Authenticate with GitHub services

**Used for**:

- Pushing Docker images to `ghcr.io`
- Pushing Helm charts to `ghcr.io`
- Creating GitHub releases
- Uploading release binaries
- Committing to `gh-pages` branch

**Action needed**: None - automatically provided by GitHub Actions

---

## Variables Configuration

Variables are like secrets, but they're not encrypted and can be read in workflow logs.

### Navigation

Go to: **Settings** → **Secrets and variables** → **Actions** → **Variables** tab

### Optional Variables

#### 1. DOCKERHUB_ORG ❌ Optional

**Purpose**: Customize Docker Hub organization name

**Default**: `wayli-app` (if not set)

**When to set**: If you want images pushed to a different Docker Hub organization

**Example**:

```
Name: DOCKERHUB_ORG
Value: mycompany
```

This will push images to `mycompany/fluxbase` instead of `fluxbase-eu/fluxbase`.

---

## Repository Settings

### 1. Workflow Permissions ✅ Required

This allows workflows to push Docker images and Helm charts to GitHub Container Registry.

**Navigation**: **Settings** → **Actions** → **General** → **Workflow permissions**

**Configuration**:

- Select: ✅ **Read and write permissions**
- Check: ✅ **Allow GitHub Actions to create and approve pull requests**
- Click: **Save**

### 2. Package Visibility ✅ Required (after first push)

After the first Docker image is pushed, you need to make the package public.

**Navigation**: **Packages** (from repository main page) → **fluxbase** → **Package settings**

**Configuration**:

- Scroll to **Danger Zone**
- Click **Change visibility**
- Select **Public**
- Type package name to confirm
- Click **I understand, change package visibility**

**Repeat for**:

- `fluxbase` (Docker image)
- `charts/fluxbase` (Helm chart)

---

## GitHub Pages Setup

### Purpose

Hosts Helm chart repository at `https://yourusername.github.io/fluxbase`

### Navigation

**Settings** → **Pages**

### Configuration

**Source**:

- Select: **Deploy from a branch**

**Branch**:

- Branch: **gh-pages**
- Folder: **/ (root)**

**Click**: **Save**

### After First Release

After your first release completes:

1. Wait 2-5 minutes for Pages to deploy
2. Visit `https://yourusername.github.io/fluxbase`
3. You should see `index.yaml` (Helm repository index)

### Using the Helm Repository

```bash
# Add repository
helm repo add fluxbase https://yourusername.github.io/fluxbase
helm repo update

# Install chart
helm install fluxbase fluxbase/fluxbase
```

---

## Summary Checklist

Use this checklist to verify your setup:

### Secrets ✅

- [ ] **NPM_TOKEN** - Added (required)
- [ ] **DOCKERHUB_USERNAME** - Added (optional, for Docker Hub)
- [ ] **DOCKERHUB_TOKEN** - Added (optional, for Docker Hub)
- [x] **GITHUB_TOKEN** - Automatic ✅

### Variables (Optional)

- [ ] **DOCKERHUB_ORG** - Set if using custom Docker Hub org

### Settings ✅

- [ ] **Workflow permissions** - Set to "Read and write permissions"
- [ ] **Allow GitHub Actions to create/approve PRs** - Enabled
- [ ] **GitHub Pages** - Enabled with gh-pages branch

### After First Release

- [ ] **Package visibility** - Set `fluxbase` package to public
- [ ] **Package visibility** - Set `charts/fluxbase` package to public
- [ ] **GitHub Pages** - Verify Helm repository is accessible

---

## Testing the Setup

### 1. Test NPM Token

```bash
# Create a test commit
git checkout -b test-setup
echo "test" >> README.md
git add README.md
git commit -m "feat: test automated release"
git push origin test-setup
```

Create a PR and merge to `main`. Release Please will create a release PR.

### 2. Test Release Workflow

1. Merge the Release Please PR
2. Go to **Actions** → **Release** workflow
3. Monitor the workflow run
4. Check each job:
   - ✅ Release Please
   - ✅ Publish Binaries (should succeed)
   - ✅ Publish Docker (should succeed)
   - ✅ Publish NPM SDK (should succeed if NPM_TOKEN is correct)
   - ✅ Publish Go Module (should succeed)
   - ✅ Publish Helm Chart (should succeed)

### 3. Verify Artifacts

After successful release:

**Binaries**:

```bash
# Check GitHub Releases
# Should see fluxbase-linux-amd64, fluxbase-darwin-arm64, etc.
```

**Docker Images**:

```bash
# GHCR (always published)
docker pull ghcr.io/yourusername/fluxbase:0.1.0

# Docker Hub (if configured)
docker pull yourusername/fluxbase:0.1.0
```

**Helm Chart**:

```bash
# OCI registry
helm pull oci://ghcr.io/yourusername/charts/fluxbase --version 0.1.0

# HTTP repository
helm repo add fluxbase https://yourusername.github.io/fluxbase
helm search repo fluxbase
```

**NPM Package**:

```bash
npm view @fluxbase/sdk
```

---

## Troubleshooting

### NPM Publish Fails

**Error**: `npm ERR! code ENEEDAUTH`

**Solutions**:

1. Verify `NPM_TOKEN` is set correctly in GitHub secrets
2. Ensure token has "Automation" scope
3. Check token hasn't expired
4. Verify you have publish permissions for `@fluxbase` organization

### Docker Push to GHCR Fails

**Error**: `unauthorized: authentication required`

**Solutions**:

1. Check **Settings** → **Actions** → **General** → **Workflow permissions**
2. Must be set to "Read and write permissions"
3. Ensure **Allow GitHub Actions to create and approve pull requests** is checked
4. Save and re-run workflow

### Docker Push to Docker Hub Fails

**Error**: `unauthorized: incorrect username or password`

**Solutions**:

1. Verify `DOCKERHUB_USERNAME` matches your Docker Hub username exactly
2. Verify `DOCKERHUB_TOKEN` is a valid access token (not password)
3. Ensure token has "Read, Write, Delete" permissions
4. Check token hasn't expired or been revoked

### Helm Chart Push Fails

**Error**: `Error: failed to authorize`

**Solutions**:

1. Same as "Docker Push to GHCR Fails" above
2. Ensure workflow has write permissions
3. Check that packages can be created in your organization

### GitHub Pages Not Working

**Error**: 404 when visiting Helm repository URL

**Solutions**:

1. Go to **Settings** → **Pages**
2. Verify **gh-pages** branch is selected as source
3. Wait 2-5 minutes after first push to gh-pages
4. Check **Actions** tab for **pages-build-deployment** workflow
5. Ensure gh-pages branch exists: `git fetch origin gh-pages`

### Package Not Public

**Error**: Cannot pull Docker image or Helm chart

**Solutions**:

1. Go to repository main page → **Packages**
2. Click on package name
3. Click **Package settings** (gear icon)
4. Scroll to **Danger Zone**
5. Click **Change visibility** → Select **Public**

---

## Security Best Practices

### Token Management

1. **Use automation tokens** (not personal access tokens)
2. **Rotate tokens regularly** (every 90-180 days)
3. **Use minimum required permissions**
4. **Never commit tokens** to repository

### Audit

Regularly review:

- **Actions** → Recent workflow runs
- **Settings** → **Secrets and variables** → Last updated
- **Packages** → Download stats and access logs

### Revoke Compromised Tokens

If a token is compromised:

1. **Immediately revoke** the token in its service (NPM, Docker Hub)
2. **Generate a new token**
3. **Update GitHub secret** with new token
4. **Review recent workflow runs** for unauthorized access

---

## Support

- **Documentation**: [VERSIONING.md](../VERSIONING.md)
- **Secrets Reference**: [SECRETS.md](SECRETS.md)
- **GitHub Actions Docs**: https://docs.github.com/en/actions
- **Issues**: https://github.com/fluxbase-eu/fluxbase/issues

---

## Quick Reference

### Minimal Setup (GHCR + NPM only)

If you want the simplest setup without Docker Hub:

**Required secrets**:

1. ✅ `NPM_TOKEN`

**Required settings**:

1. ✅ Workflow permissions: "Read and write permissions"
2. ✅ GitHub Pages: Enabled with gh-pages branch

**Result**:

- ✅ Docker images on ghcr.io
- ✅ Helm charts on ghcr.io and GitHub Pages
- ✅ NPM package on npmjs.com
- ✅ Binaries on GitHub Releases
- ❌ No Docker Hub images

### Full Setup (GHCR + Docker Hub + NPM)

**Required secrets**:

1. ✅ `NPM_TOKEN`
2. ✅ `DOCKERHUB_USERNAME`
3. ✅ `DOCKERHUB_TOKEN`

**Optional variables**:

- `DOCKERHUB_ORG` (if different from default)

**Required settings**:

1. ✅ Workflow permissions: "Read and write permissions"
2. ✅ GitHub Pages: Enabled with gh-pages branch

**Result**:

- ✅ Docker images on ghcr.io AND Docker Hub
- ✅ Helm charts on ghcr.io and GitHub Pages
- ✅ NPM package on npmjs.com
- ✅ Binaries on GitHub Releases

---

**Setup complete!** Your repository is now ready for automated releases. 🚀
