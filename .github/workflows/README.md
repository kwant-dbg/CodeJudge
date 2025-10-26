# CI/CD Workflows# CI/CD Workflows



Automated building, testing, and deployment for CodeJudge services.This directory contains GitHub Actions workflows for automated building, testing, and deployment of CodeJudge services.



## Active Workflows## Workflows



### `build-push-acr.yml` - Azure Container Registry CI/CD### `build-push-acr.yml` - Azure Container Registry CI/CD



**Triggers:** Push to `mono` branch, manual dispatchAutomatically builds, tests, and pushes Docker images to Azure Container Registry.



**Actions:****Triggers:**

- ✅ Builds Monolith & Judge Docker images- Push to `mono` branch (when relevant files change)

- ✅ Pushes to `codejudgeacr9519.azurecr.io`- Manual dispatch via GitHub Actions UI

- ✅ Runs health checks & API tests

- ✅ Security scanning (Trivy)**What it does:**

- ✅ Builds Docker images for Monolith and Judge services

**Images:** `codejudgeacr9519.azurecr.io/codejudge-{monolith|judge}:latest`- ✅ Pushes to Azure Container Registry (`codejudgeacr9519.azurecr.io`)

- ✅ Runs health checks and API tests

## Setup- ✅ Security scanning with Trivy

- ✅ Build caching for faster builds

### GitHub Secrets Required

**Images Published:**

In **Settings → Secrets → Actions**, add:- `codejudgeacr9519.azurecr.io/codejudge-monolith:latest`

- `codejudgeacr9519.azurecr.io/codejudge-monolith:<commit-sha>`

- `ACR_USERNAME` - Get: `az acr credential show --name codejudgeacr9519 --query username -o tsv`- `codejudgeacr9519.azurecr.io/codejudge-judge:latest`

- `ACR_PASSWORD` - Get: `az acr credential show --name codejudgeacr9519 --query "passwords[0].value" -o tsv`- `codejudgeacr9519.azurecr.io/codejudge-judge:<commit-sha>`



## Deployment Flow## Setup Instructions



1. Push code → GitHub Actions builds & tests → Images in ACR### Required GitHub Secrets

2. Deploy: `.\deploy\azure-deploy.ps1`

Add these secrets in your repository settings (`Settings` → `Secrets and variables` → `Actions`):

## Free Tier

1. **`ACR_USERNAME`**: Your Azure Container Registry username

- **GitHub Actions**: 2,000 min/month (each build ~5-10 min)   - Get it: `az acr credential show --name codejudgeacr9519 --query username -o tsv`

- **ACR Basic**: 10 GB storage included

2. **`ACR_PASSWORD`**: Your Azure Container Registry password
   - Get it: `az acr credential show --name codejudgeacr9519 --query "passwords[0].value" -o tsv`

3. **`AZURE_CREDENTIALS`**: Service principal credentials (for future automated deployment)
   - Create it: `az ad sp create-for-rbac --name "github-actions-codejudge" --role contributor --scopes /subscriptions/<subscription-id>/resourceGroups/codejudge-sea-rg --sdk-auth`

## Deployment Workflow

1. **Push code** to `mono` branch
2. **GitHub Actions** automatically builds and tests
3. **Images pushed** to Azure Container Registry
4. **Manual deploy** using PowerShell: `.\deploy\azure-deploy.ps1`

## Pull Images Locally

```bash
# Login to ACR
az acr login --name codejudgeacr9519

# Pull images
docker pull codejudgeacr9519.azurecr.io/codejudge-monolith:latest
docker pull codejudgeacr9519.azurecr.io/codejudge-judge:latest
```

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
