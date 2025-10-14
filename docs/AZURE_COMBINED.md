# CodeJudge — Azure Deployment (Combined)

This consolidated guide combines quick-start, step-by-step, and deployment options for deploying CodeJudge to Microsoft Azure.

## Overview

This document covers three deployment approaches:

- Option 1 — Azure Container Instances (ACI): simplest for testing and quick deployments
- Option 2 — Azure App Service: suitable for production web apps with scaling
- Option 3 — Azure Kubernetes Service (AKS): advanced; for high scale and microservices

Prerequisites
- An Azure account (trial or paid) and Azure CLI installed (`az login`)
- Docker Desktop (for building and pushing container images)

---

## Option 1 — Azure Container Instances (ACI)

Cost: ~$10–20/month (estimate)

Steps (high level):

1. Install Azure CLI and login

```powershell
# Install Azure CLI (Windows)
winget install Microsoft.AzureCLI
az login
```

2. Create resource group

```powershell
az group create --name codejudge-rg --location eastus
```

3. Create Azure Container Registry (ACR)

```powershell
az acr create --resource-group codejudge-rg --name codejudgeacr --sku Basic
az acr login --name codejudgeacr
```

4. Build and push images

```powershell
docker-compose -f docker-compose.monolith.yml build monolith
docker tag codejudge-monolith:latest codejudgeacr.azurecr.io/codejudge-monolith:latest
docker push codejudgeacr.azurecr.io/codejudge-monolith:latest
```

5. Create PostgreSQL (Flexible Server) and Redis

See `deploy/azure.ps1` for a script; example commands:

```powershell
az postgres flexible-server create --resource-group codejudge-rg --name codejudge-db --admin-user codejudgeadmin --admin-password "YourStrongPassword" --sku-name Standard_B1ms --version 14 --storage-size 32 --public-access 0.0.0.0

az redis create --resource-group codejudge-rg --name codejudge-redis --sku Basic --vm-size c0
```

6. Deploy the container instance

```powershell
# Replace placeholders with real values
$DB_URL = "postgres://codejudgeadmin:YourStrongPassword@codejudge-db.postgres.database.azure.com:5432/codejudgedb?sslmode=require"
$REDIS_KEY = az redis list-keys --resource-group codejudge-rg --name codejudge-redis --query primaryKey -o tsv
$REDIS_URL = "rediss://:$REDIS_KEY@codejudge-redis.redis.cache.windows.net:6380"

az container create --resource-group codejudge-rg --name codejudge-monolith --image codejudgeacr.azurecr.io/codejudge-monolith:latest --registry-login-server codejudgeacr.azurecr.io --registry-username codejudgeacr --registry-password $(az acr credential show --name codejudgeacr --query passwords[0].value -o tsv) --dns-name-label codejudge-app --ports 8080 --environment-variables DATABASE_URL="$DB_URL" REDIS_URL="$REDIS_URL" JWT_SECRET="your-production-jwt-secret" PORT=8080 GIN_MODE=release --cpu 1 --memory 1.5
```

7. Access the app (retrieve FQDN):

```powershell
az container show --resource-group codejudge-rg --name codejudge-monolith --query ipAddress.fqdn -o tsv
```

---

## Option 2 — Azure App Service (recommended for production web apps)

Cost: ~$15–30/month (estimate)

1. Create an App Service plan and web app

```powershell
az appservice plan create --name codejudge-plan --resource-group codejudge-rg --location eastus --is-linux --sku B1
az webapp create --resource-group codejudge-rg --plan codejudge-plan --name codejudge-app --deployment-container-image-name codejudgeacr.azurecr.io/codejudge-monolith:latest
```

2. Configure app settings

```powershell
az webapp config appsettings set --resource-group codejudge-rg --name codejudge-app --settings DATABASE_URL="$DB_URL" REDIS_URL="$REDIS_URL" JWT_SECRET="your-production-jwt-secret" PORT=8080 WEBSITES_PORT=8080 GIN_MODE=release
```

3. Configure container registry access

```powershell
az webapp config container set --name codejudge-app --resource-group codejudge-rg --docker-custom-image-name codejudgeacr.azurecr.io/codejudge-monolith:latest --docker-registry-server-url https://codejudgeacr.azurecr.io --docker-registry-server-user $(az acr credential show --name codejudgeacr --query username -o tsv) --docker-registry-server-password $(az acr credential show --name codejudgeacr --query passwords[0].value -o tsv)
```

4. Access via: https://codejudge-app.azurewebsites.net

---

## Option 3 — AKS (advanced)

Use AKS if you need Kubernetes features, multi-replica deployments, or privileged judge containers. See `docs/AZURE_DEPLOYMENT_MANUAL.md` for examples and a sample `azure-k8s-deployment.yaml`.

---

## Quick deployment script

There is a `deploy/azure.ps1` script included that automates many steps (ACR, DB, Redis, push image, deploy). Use it as a starting point and review variables and generated secrets before running in production.

## Environment and secrets

Create `.env.azure` with the production values (never commit secrets):

```
DATABASE_URL=postgres://codejudgeadmin:PASSWORD@codejudge-db.postgres.database.azure.com:5432/codejudgedb?sslmode=require
REDIS_URL=rediss://:REDIS_KEY@codejudge-redis.redis.cache.windows.net:6380
JWT_SECRET=generate-strong-random-secret-here
PORT=8080
GIN_MODE=release
LOG_LEVEL=info
```

## Clean up

To remove resources created for the deployment:

```powershell
az group delete --name codejudge-rg --yes --no-wait
```

---

## Troubleshooting notes

- If images are not found: rebuild locally and push to ACR
- If DB firewall issues occur: ensure firewall allows Azure services or configure proper rules
- For judge service (C++ sandbox): due to privileged requirements, keep local judge connected to Azure Redis if you host web app in Azure

---

If you want I can: commit this file to the repo (create/commit/push) under `docs/AZURE_COMBINED.md`, or further tailor it (e.g., regional defaults, simplified script examples). Tell me to proceed and I'll commit + push.
