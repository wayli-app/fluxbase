# GitHub Secrets Configuration

This document lists all the secrets that need to be configured in your GitHub repository for the automated release workflow to function properly.

## Required Secrets

### 1. NPM_TOKEN (Required for TypeScript SDK publishing)

**Purpose**: Authenticate with NPM to publish the `@fluxbase/sdk` package.

**How to get it**:
1. Log in to [npmjs.com](https://www.npmjs.com/)
2. Go to your profile → Access Tokens
3. Click "Generate New Token" → "Classic Token"
4. Select "Automation" type (for CI/CD)
5. Copy the token

**Add to GitHub**:
```
Settings → Secrets and variables → Actions → New repository secret
Name: NPM_TOKEN
Value: npm_xxx...
```

### 2. DOCKERHUB_USERNAME (Optional - for Docker Hub publishing)

**Purpose**: Authenticate with Docker Hub to push images.

**Note**: Not required if you only use GitHub Container Registry (ghcr.io). The workflow already publishes to ghcr.io using `GITHUB_TOKEN`.

**How to get it**:
- Your Docker Hub username

**Add to GitHub**:
```
Name: DOCKERHUB_USERNAME
Value: your-dockerhub-username
```

### 3. DOCKERHUB_TOKEN (Optional - for Docker Hub publishing)

**Purpose**: Authenticate with Docker Hub to push images.

**How to get it**:
1. Log in to [hub.docker.com](https://hub.docker.com/)
2. Go to Account Settings → Security → New Access Token
3. Give it a descriptive name (e.g., "GitHub Actions Fluxbase")
4. Copy the token

**Add to GitHub**:
```
Name: DOCKERHUB_TOKEN
Value: dckr_pat_xxx...
```

---

## Automatically Available Secrets

These secrets are automatically provided by GitHub Actions and **do not need to be configured**:

### GITHUB_TOKEN

**Purpose**: Authenticate with GitHub API, push to ghcr.io, create releases, update gh-pages branch.

**Used for**:
- ✅ Pushing Docker images to `ghcr.io`
- ✅ Pushing Helm charts to `ghcr.io`
- ✅ Creating GitHub releases
- ✅ Uploading release assets (binaries)
- ✅ Committing to `gh-pages` branch
- ✅ Updating package metadata

**Automatically available**: Yes (provided by GitHub)

---

## Summary Table

### Secrets

| Secret Name | Required | Purpose | Where to Get |
|-------------|----------|---------|--------------|
| `NPM_TOKEN` | ✅ Yes | Publish TypeScript SDK to NPM | [npmjs.com](https://www.npmjs.com/) → Access Tokens |
| `DOCKERHUB_USERNAME` | ❌ Optional | Push to Docker Hub (in addition to ghcr.io) | Your Docker Hub username |
| `DOCKERHUB_TOKEN` | ❌ Optional | Push to Docker Hub (in addition to ghcr.io) | [hub.docker.com](https://hub.docker.com/) → Security → Access Tokens |
| `GITHUB_TOKEN` | ✅ Auto | All GitHub operations | Automatically provided |

### Variables

| Variable Name | Required | Purpose | Default |
|---------------|----------|---------|---------|
| `DOCKERHUB_ORG` | ❌ Optional | Docker Hub organization name | `wayli-app` |

**Note**: Variables are like secrets but not encrypted. Set them if you need to customize behavior.

---

## Configuration Steps

### Step 1: Add NPM_TOKEN

1. Go to your GitHub repository
2. Click **Settings** → **Secrets and variables** → **Actions**
3. Click **New repository secret**
4. Name: `NPM_TOKEN`
5. Value: Your NPM automation token
6. Click **Add secret**

### Step 2: (Optional) Add Docker Hub Credentials

Only if you want to publish to Docker Hub in addition to ghcr.io:

1. Click **New repository secret**
2. Name: `DOCKERHUB_USERNAME`
3. Value: Your Docker Hub username
4. Click **Add secret**

5. Click **New repository secret** again
6. Name: `DOCKERHUB_TOKEN`
7. Value: Your Docker Hub access token
8. Click **Add secret**

### Step 3: Enable GitHub Pages (for Helm Chart)

1. Go to **Settings** → **Pages**
2. Under "Source", select **Deploy from a branch**
3. Select branch: `gh-pages`
4. Select folder: `/ (root)`
5. Click **Save**

After the first release, your Helm chart repository will be available at:
```
https://<your-org>.github.io/fluxbase
```

---

## Verifying Secrets

After adding secrets, you can verify they're configured:

1. Go to **Settings** → **Secrets and variables** → **Actions**
2. You should see:
   - `NPM_TOKEN` (green checkmark)
   - `DOCKERHUB_USERNAME` (if added)
   - `DOCKERHUB_TOKEN` (if added)

**Note**: You cannot view the actual secret values after creation, only update or delete them.

---

## Testing the Workflow

To test the release workflow:

1. Make sure all required secrets are configured
2. Push a conventional commit to `main`:
   ```bash
   git commit -m "feat: test release workflow"
   git push origin main
   ```
3. Release Please will create a PR
4. Merge the PR to trigger the release
5. Monitor the workflow at **Actions** → **Release**

---

## Troubleshooting

### NPM Publish Fails

**Error**: `npm ERR! code ENEEDAUTH`

**Solution**:
- Verify `NPM_TOKEN` is set correctly
- Ensure token has "Automation" scope
- Token should not be expired

### Docker Push Fails (ghcr.io)

**Error**: `unauthorized: authentication required`

**Solution**:
- Check that repository has **write** permissions for packages
- Go to **Settings** → **Actions** → **General** → **Workflow permissions**
- Select "Read and write permissions"
- Click **Save**

### Helm Chart Push Fails

**Error**: `Error: failed to authorize: failed to fetch anonymous token`

**Solution**:
- Same as Docker Push (above) - check workflow permissions
- Ensure `gh-pages` branch exists or can be created

### GitHub Pages Not Working

**Error**: Helm chart URL returns 404

**Solution**:
1. Go to **Settings** → **Pages**
2. Ensure source is set to `gh-pages` branch
3. Wait a few minutes for first deployment
4. Check **Actions** tab for Pages deployment status

---

## Security Best Practices

### Token Rotation

Rotate secrets periodically:
- NPM tokens: Every 90-180 days
- Docker Hub tokens: Every 90-180 days

### Least Privilege

- Use automation tokens (not personal tokens)
- Grant minimum required permissions
- Don't share tokens across repositories

### Audit

Review secret usage:
- Check **Actions** logs for unexpected access
- Monitor NPM package publishes
- Review Docker image tags

---

## Additional Resources

- [GitHub Encrypted Secrets](https://docs.github.com/en/actions/security-guides/encrypted-secrets)
- [NPM Access Tokens](https://docs.npmjs.com/creating-and-viewing-access-tokens)
- [Docker Hub Access Tokens](https://docs.docker.com/security/for-developers/access-tokens/)
- [GitHub Container Registry](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
- [Helm Chart Repository](https://helm.sh/docs/topics/chart_repository/)
