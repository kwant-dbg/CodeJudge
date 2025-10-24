# CI/CD Workflows

This directory contains GitHub Actions workflows for automated building, testing, and deployment of CodeJudge services.

## Workflows

### 1. Docker Build and Push (`docker-build-push.yml`)

Automatically builds and pushes Docker images for both Monolith and Judge services.

**Triggers:**
- Push to `mono` or `main` branches (when relevant files change)
- Pull requests
- Manual dispatch via GitHub Actions UI

**Features:**
- ✅ Multi-architecture builds (amd64, arm64 for monolith)
- ✅ Automated testing of built images
- ✅ Security scanning with Trivy
- ✅ Caching for faster builds
- ✅ Pushes to GitHub Container Registry (ghcr.io)
- ✅ Optional Docker Hub support

**Images Published:**
- `ghcr.io/<owner>/codejudge-monolith:latest`
- `ghcr.io/<owner>/codejudge-monolith:<branch>-<sha>`
- `ghcr.io/<owner>/codejudge-judge:latest`
- `ghcr.io/<owner>/codejudge-judge:<branch>-<sha>`

### 2. Azure Deployment (`deploy-monolith-azure.yml`)

Deploys to Azure Container Instances (currently disabled by default).

## Setup Instructions

### GitHub Container Registry (Default)

No setup needed! The workflow uses `GITHUB_TOKEN` which is automatically available.

**To pull images:**
```bash
# Login (if private repo)
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin

# Pull images
docker pull ghcr.io/<owner>/codejudge-monolith:latest
docker pull ghcr.io/<owner>/codejudge-judge:latest
```

### Docker Hub (Optional)

To also push to Docker Hub, add these secrets to your repository:

1. Go to: `Settings` → `Secrets and variables` → `Actions`
2. Add these secrets:
   - `DOCKERHUB_USERNAME`: Your Docker Hub username
   - `DOCKERHUB_TOKEN`: Docker Hub access token ([create one here](https://hub.docker.com/settings/security))

### Azure Deployment (Optional)

To enable Azure deployment:

1. Create Azure service principal:
```bash
az ad sp create-for-rbac --name "codejudge-deploy" --role contributor \
    --scopes /subscriptions/<subscription-id>/resourceGroups/<resource-group> \
    --sdk-auth
```

2. Add these secrets:
   - `AZURE_CREDENTIALS`: Output from above command
   - `AZURE_SUBSCRIPTION_ID`: Your Azure subscription ID
   - `POSTGRES_PASSWORD`: Database password
   - `JWT_SECRET`: JWT secret key

3. Uncomment the trigger in `deploy-monolith-azure.yml`

## Using the Built Images

### With Docker Compose

Update your `docker-compose.yml`:

```yaml
services:
  monolith:
    image: ghcr.io/<owner>/codejudge-monolith:latest
    # ... rest of config

  judge:
    image: ghcr.io/<owner>/codejudge-judge:latest
    # ... rest of config
```

Then run:
```bash
docker-compose pull
docker-compose up -d
```

### Standalone

```bash
# Run monolith
docker run -d \
  -p 8080:8080 \
  -e DATABASE_URL="postgres://user:pass@host:5432/db" \
  -e REDIS_URL="redis://host:6379" \
  -e JWT_SECRET="your-secret" \
  ghcr.io/<owner>/codejudge-monolith:latest

# Run judge
docker run -d \
  --privileged \
  -e DATABASE_URL="postgres://user:pass@host:5432/db" \
  -e REDIS_HOST="host" \
  -e REDIS_PORT="6379" \
  ghcr.io/<owner>/codejudge-judge:latest
```

## Manual Workflow Dispatch

You can manually trigger builds from GitHub:

1. Go to `Actions` tab
2. Select "Build and Push Docker Images"
3. Click "Run workflow"
4. Choose whether to push images to registry

## Monitoring Builds

- Check the `Actions` tab for build status
- Security scan results appear in the `Security` tab
- Build summaries show in the workflow run page

## Troubleshooting

### Build fails with authentication error

Make sure GitHub Actions has permission to write packages:
1. Go to `Settings` → `Actions` → `General`
2. Under "Workflow permissions", select "Read and write permissions"

### Images not appearing in ghcr.io

1. Check that the package is public (or you're authenticated)
2. Go to your profile → `Packages` to see all published packages

### Security scan fails

This is informational only and won't block the build. Review the security findings in the `Security` → `Code scanning alerts` tab.

## Local Testing

Test the workflow locally using [act](https://github.com/nektos/act):

```bash
# Install act
# macOS: brew install act
# Linux: see https://github.com/nektos/act

# Run the workflow
act push --secret GITHUB_TOKEN=$GITHUB_TOKEN
```
