# GitHub Setup - Quick Reference Card

## 🔑 Required Secrets

Navigate to: **Settings → Secrets and variables → Actions → Secrets tab**

```
Name: NPM_TOKEN
Value: npm_xxxxxxxxxxxxxxxxxxxxxxxx
Source: npmjs.com → Access Tokens → Generate (Automation type)
```

## ⚙️ Required Settings

### 1. Workflow Permissions
**Settings → Actions → General → Workflow permissions**
- ✅ Read and write permissions
- ✅ Allow GitHub Actions to create and approve pull requests

### 2. GitHub Pages
**Settings → Pages**
- Source: Deploy from a branch
- Branch: **gh-pages**
- Folder: **/ (root)**

## 📦 Optional Secrets (Docker Hub)

Only if you want to publish to Docker Hub in addition to ghcr.io:

```
Name: DOCKERHUB_USERNAME
Value: your-dockerhub-username

Name: DOCKERHUB_TOKEN
Value: dckr_pat_xxxxxxxxxxxxxxxxxxxxxxxx
Source: hub.docker.com → Security → New Access Token
```

## 🏷️ Optional Variables

**Settings → Secrets and variables → Actions → Variables tab**

```
Name: DOCKERHUB_ORG
Value: your-org-name
Default: wayli-app
```

## ✅ Post-First-Release Checklist

After your first successful release:

1. **Make packages public**
   - Go to **Packages** → **fluxbase** → **Package settings**
   - Change visibility → Public
   - Repeat for **charts/fluxbase**

2. **Verify Helm repository**
   - Visit: https://yourusername.github.io/fluxbase
   - Should see: index.yaml file

3. **Test installations**
   ```bash
   # Docker
   docker pull ghcr.io/yourusername/fluxbase:0.1.0

   # Helm (OCI)
   helm pull oci://ghcr.io/yourusername/charts/fluxbase --version 0.1.0

   # Helm (HTTP)
   helm repo add fluxbase https://yourusername.github.io/fluxbase
   helm search repo fluxbase

   # NPM
   npm view @fluxbase/sdk
   ```

## 🚀 Triggering a Release

```bash
# 1. Commit with conventional message
git commit -m "feat: add new feature"     # Minor bump
git commit -m "fix: resolve bug"          # Patch bump
git commit -m "feat!: breaking change"    # Major bump

# 2. Push to main
git push origin main

# 3. Release Please creates PR automatically

# 4. Merge the Release PR
# → Full release workflow runs automatically
```

## 🔍 Verifying Setup

```bash
# Check workflow status
# Go to: Actions → Release workflow

# Monitor jobs:
✅ Release Please
✅ Publish Binaries
✅ Publish Docker
✅ Publish NPM SDK
✅ Publish Go Module
✅ Publish Helm Chart
```

## 📚 Full Documentation

- Setup Guide: [.github/SETUP_GUIDE.md](.github/SETUP_GUIDE.md)
- Secrets Reference: [.github/SECRETS.md](.github/SECRETS.md)
- Versioning Guide: [VERSIONING.md](../VERSIONING.md)

## 🆘 Quick Troubleshooting

**NPM publish fails**
→ Check NPM_TOKEN is set and valid

**Docker push to ghcr.io fails**
→ Check workflow permissions are "Read and write"

**Helm repository 404**
→ Check GitHub Pages is enabled with gh-pages branch

**Package private error**
→ Make package public in Package settings

---

**Minimal Setup**: Just `NPM_TOKEN` + Workflow permissions + GitHub Pages
**Full Setup**: Add Docker Hub secrets for dual registry publishing
