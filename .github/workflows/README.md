# CI/CD Workflows

This directory contains GitHub Actions workflows for CodeJudge.

## `build-push-acr.yml`

Builds and pushes Docker images for the `monolith` and `judge` services to Azure Container Registry (ACR).

- **Triggers:**
  - Push to `mono` branch (path-filtered)
  - Manual dispatch (`workflow_dispatch`)

- **Jobs:**
  - Builds `monolith` and `judge` Docker images.
  - Pushes images to `codejudgeacr9519.azurecr.io` with tags `latest` and the commit SHA.
  - Runs health checks and API tests.
  - Performs security scanning with Trivy.

### Setup

Configure the following secrets in the repository's `Settings > Secrets and variables > Actions`:

- `ACR_USERNAME`: Azure Container Registry username.
  - `az acr credential show --name codejudgeacr9519 --query username -o tsv`
- `ACR_PASSWORD`: Azure Container Registry password.
  - `az acr credential show --name codejudgeacr9519 --query "passwords[0].value" -o tsv`

### Deployment

After a successful workflow run, images are available in ACR. To deploy, run the deployment script:

```powershell
.\deploy\azure-deploy.ps1
```

### Local Usage

To pull images from ACR for local use:

```bash
# 1. Login to ACR
az acr login --name codejudgeacr9519

# 2. Pull images
docker pull codejudgeacr9519.azurecr.io/codejudge-monolith:latest
docker pull codejudgeacr9519.azurecr.io/codejudge-judge:latest
```