# Release Please Setup Guide

The Release Please workflow requires special permissions to create pull requests. You have two options to resolve the permission error:

## Option 1: Enable Workflow Permissions (Easiest)

1. Go to your repository on GitHub
2. Navigate to **Settings** → **Actions** → **General**
3. Scroll down to **Workflow permissions**
4. Select **"Read and write permissions"**
5. Check the box **"Allow GitHub Actions to create and approve pull requests"**
6. Click **Save**

This allows the default `GITHUB_TOKEN` to create pull requests.

## Option 2: Use a Personal Access Token (More Secure)

1. Create a fine-grained Personal Access Token:
   - Go to GitHub → **Settings** → **Developer settings** → **Personal access tokens** → **Fine-grained tokens**
   - Click **Generate new token**
   - Give it a name: `Release Please Token`
   - Select repository access: Choose this repository
   - Grant these permissions:
     - **Contents**: Read and write
     - **Pull requests**: Read and write
     - **Metadata**: Read-only (automatically included)
   - Click **Generate token** and copy it

2. Add the token as a repository secret:
   - Go to your repository → **Settings** → **Secrets and variables** → **Actions**
   - Click **New repository secret**
   - Name: `RELEASE_PLEASE_TOKEN`
   - Value: Paste the token you copied
   - Click **Add secret**

The workflow is already configured to use `RELEASE_PLEASE_TOKEN` if available, falling back to `GITHUB_TOKEN` otherwise.

## How Release Please Works

Once configured, Release Please will:

1. **On every push to main**: Analyze commits since the last release
2. **Create/Update a Release PR**: Opens a PR with version bump and changelog
3. **When you merge the Release PR**:
   - Creates a GitHub release
   - Publishes Docker images to GHCR (and Docker Hub if configured)
   - Publishes Go module
   - Publishes NPM packages (if configured)
   - Publishes Helm chart

## Commit Message Convention

Release Please uses [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` - New feature (minor version bump)
- `fix:` - Bug fix (patch version bump)
- `feat!:` or `fix!:` - Breaking change (major version bump)
- `docs:`, `chore:`, `style:`, `refactor:`, `test:` - No version bump

Example:
```
feat: add user authentication
fix: resolve database connection timeout
feat!: change API response format (breaking change)
```

## Docker Hub Configuration (Optional)

To publish to Docker Hub in addition to GHCR:

1. Create Docker Hub credentials as repository secrets:
   - `DOCKERHUB_USERNAME`: Your Docker Hub username
   - `DOCKERHUB_TOKEN`: Docker Hub access token

2. Optionally set a repository variable:
   - `DOCKERHUB_ORG`: Your Docker Hub organization (defaults to 'wayli-app')

If Docker Hub credentials are not configured, images will only be pushed to GitHub Container Registry.

## NPM Publishing (Optional)

To publish the TypeScript SDK to NPM:

1. Create an NPM access token
2. Add it as a repository secret: `NPM_TOKEN`

## Troubleshooting

### "GitHub Actions is not permitted to create or approve pull requests"
- Follow Option 1 or Option 2 above

### Release PR not created
- Check that your commits follow the Conventional Commits format
- Check the Actions tab for error logs

### Docker build fails
- Ensure all required secrets are set
- Check that Go version matches (currently 1.25)

### Multiple releases triggered
- Only merge the Release PR when you're ready to release
- Don't manually create tags
