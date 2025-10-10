# CodeJudge - Azure Deployment Guide

## ✅ Azure Readiness Status

**YES**, your project is ready for Azure deployment! Here's how to deploy it.

## Azure Student Account Benefits

With Azure for Students, you get:
- $100 in Azure credits (12 months)
- Free services: App Service, Azure Database for PostgreSQL, Azure Container Registry
- No credit card required

## Deployment Options

### Option 1: Azure Container Instances (Simplest - Recommended for Testing)

**Cost**: ~$10-20/month with student credits
**Complexity**: Low
**Best for**: Quick deployment, testing, low traffic

#### Steps:

1. **Install Azure CLI**
```powershell
# Install Azure CLI
winget install Microsoft.AzureCLI

# Login
az login
```

2. **Create Resource Group**
```powershell
az group create --name codejudge-rg --location eastus
```

3. **Create Azure Container Registry (ACR)**
```powershell
# Create registry
az acr create --resource-group codejudge-rg --name codejudgeacr --sku Basic

# Login to ACR
az acr login --name codejudgeacr
```

4. **Build and Push Images**
```powershell
# Tag images for ACR
$ACR_NAME="codejudgeacr"
docker tag codejudge-monolith:latest $ACR_NAME.azurecr.io/codejudge-monolith:latest

# Push to ACR
docker push $ACR_NAME.azurecr.io/codejudge-monolith:latest
```

5. **Create PostgreSQL Database**
```powershell
# Create PostgreSQL Flexible Server
az postgres flexible-server create `
  --resource-group codejudge-rg `
  --name codejudge-db `
  --location eastus `
  --admin-user codejudgeadmin `
  --admin-password "YourStrongPassword123!" `
  --sku-name Standard_B1ms `
  --tier Burstable `
  --version 14 `
  --storage-size 32 `
  --public-access 0.0.0.0

# Create database
az postgres flexible-server db create `
  --resource-group codejudge-rg `
  --server-name codejudge-db `
  --database-name codejudgedb
```

6. **Create Redis Cache**
```powershell
az redis create `
  --resource-group codejudge-rg `
  --name codejudge-redis `
  --location eastus `
  --sku Basic `
  --vm-size c0
```

7. **Deploy Container Instances**
```powershell
# Get database connection string
$DB_HOST = "codejudge-db.postgres.database.azure.com"
$DB_URL = "postgres://codejudgeadmin:YourStrongPassword123!@$DB_HOST:5432/codejudgedb?sslmode=require"

# Get Redis connection
$REDIS_KEY = az redis list-keys --resource-group codejudge-rg --name codejudge-redis --query primaryKey -o tsv
$REDIS_URL = "rediss://:$REDIS_KEY@codejudge-redis.redis.cache.windows.net:6380"

# Deploy monolith
az container create `
  --resource-group codejudge-rg `
  --name codejudge-monolith `
  --image codejudgeacr.azurecr.io/codejudge-monolith:latest `
  --registry-login-server codejudgeacr.azurecr.io `
  --registry-username codejudgeacr `
  --registry-password $(az acr credential show --name codejudgeacr --query passwords[0].value -o tsv) `
  --dns-name-label codejudge-app `
  --ports 8080 `
  --environment-variables `
    DATABASE_URL="$DB_URL" `
    REDIS_URL="$REDIS_URL" `
    JWT_SECRET="your-production-jwt-secret-change-this" `
    PORT=8080 `
    GIN_MODE=release `
  --cpu 1 --memory 1.5
```

8. **Access Your App**
```powershell
# Get the FQDN
az container show --resource-group codejudge-rg --name codejudge-monolith --query ipAddress.fqdn -o tsv
# Access at: http://codejudge-app.eastus.azurecontainer.io:8080
```

---

### Option 2: Azure App Service (Best for Production)

**Cost**: ~$15-30/month with student credits
**Complexity**: Medium
**Best for**: Production, auto-scaling, custom domains

#### Steps:

1. **Create App Service Plan**
```powershell
az appservice plan create `
  --name codejudge-plan `
  --resource-group codejudge-rg `
  --location eastus `
  --is-linux `
  --sku B1
```

2. **Create Web App**
```powershell
az webapp create `
  --resource-group codejudge-rg `
  --plan codejudge-plan `
  --name codejudge-app `
  --deployment-container-image-name codejudgeacr.azurecr.io/codejudge-monolith:latest
```

3. **Configure App Settings**
```powershell
az webapp config appsettings set `
  --resource-group codejudge-rg `
  --name codejudge-app `
  --settings `
    DATABASE_URL="$DB_URL" `
    REDIS_URL="$REDIS_URL" `
    JWT_SECRET="your-production-jwt-secret" `
    PORT=8080 `
    WEBSITES_PORT=8080
```

4. **Enable Container Registry Access**
```powershell
az webapp config container set `
  --name codejudge-app `
  --resource-group codejudge-rg `
  --docker-custom-image-name codejudgeacr.azurecr.io/codejudge-monolith:latest `
  --docker-registry-server-url https://codejudgeacr.azurecr.io `
  --docker-registry-server-user codejudgeacr `
  --docker-registry-server-password $(az acr credential show --name codejudgeacr --query passwords[0].value -o tsv)
```

5. **Access Your App**
```
https://codejudge-app.azurewebsites.net
```

---

### Option 3: Azure Kubernetes Service (AKS) - Advanced

**Cost**: ~$30-50/month with student credits
**Complexity**: High
**Best for**: Microservices, high scalability, production grade

#### Steps:

1. **Create AKS Cluster**
```powershell
az aks create `
  --resource-group codejudge-rg `
  --name codejudge-aks `
  --node-count 1 `
  --node-vm-size Standard_B2s `
  --enable-managed-identity `
  --generate-ssh-keys `
  --attach-acr codejudgeacr
```

2. **Get Credentials**
```powershell
az aks get-credentials --resource-group codejudge-rg --name codejudge-aks
```

3. **Create Kubernetes Deployment**
Create `azure-k8s-deployment.yaml`:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: codejudge-monolith
spec:
  replicas: 2
  selector:
    matchLabels:
      app: codejudge
  template:
    metadata:
      labels:
        app: codejudge
    spec:
      containers:
      - name: monolith
        image: codejudgeacr.azurecr.io/codejudge-monolith:latest
        ports:
        - containerPort: 8080
        env:
        - name: DATABASE_URL
          value: "postgres://codejudgeadmin:YourStrongPassword123!@codejudge-db.postgres.database.azure.com:5432/codejudgedb?sslmode=require"
        - name: REDIS_URL
          value: "rediss://:REDIS_KEY@codejudge-redis.redis.cache.windows.net:6380"
        - name: JWT_SECRET
          value: "your-production-jwt-secret"
        - name: PORT
          value: "8080"
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
---
apiVersion: v1
kind: Service
metadata:
  name: codejudge-service
spec:
  type: LoadBalancer
  ports:
  - port: 80
    targetPort: 8080
  selector:
    app: codejudge
```

4. **Deploy to AKS**
```powershell
kubectl apply -f azure-k8s-deployment.yaml

# Get external IP
kubectl get service codejudge-service
```

---

## Cost Estimates (with Azure Student Credits)

| Service | Option 1 (ACI) | Option 2 (App Service) | Option 3 (AKS) |
|---------|----------------|------------------------|----------------|
| Container/Compute | $10-15 | $15-20 | $30-40 |
| PostgreSQL (Flexible) | $15-20 | $15-20 | $15-20 |
| Redis Cache (Basic) | $16 | $16 | $16 |
| Container Registry | Free (Basic) | Free (Basic) | Free (Basic) |
| **Total/month** | **~$41-50** | **~$46-56** | **~$61-76** |
| **Student Credits** | 2+ months free | 2 months free | 1.5 months free |

## Judge Service Deployment (C++ Sandbox)

⚠️ **Important**: The judge service requires privileged access for sandboxing. For Azure:

### Option A: Deploy Judge as Separate Container Instance
```powershell
# Build and push judge image
docker-compose -f docker-compose.monolith.yml build judge
docker tag codejudge-judge:latest codejudgeacr.azurecr.io/codejudge-judge:latest
docker push codejudgeacr.azurecr.io/codejudge-judge:latest

# Deploy with privileged mode (requires AKS with security context)
```

### Option B: Use Azure Container Apps (Recommended)
Azure Container Apps supports background processing and Redis queues better than ACI.

---

## Environment Configuration

Create `.env.azure` file:
```bash
# Azure Production
DATABASE_URL=postgres://codejudgeadmin:PASSWORD@codejudge-db.postgres.database.azure.com:5432/codejudgedb?sslmode=require
REDIS_URL=rediss://:REDIS_KEY@codejudge-redis.redis.cache.windows.net:6380
JWT_SECRET=generate-strong-random-secret-here
PORT=8080
GIN_MODE=release
LOG_LEVEL=info
```

Generate strong JWT secret:
```powershell
# PowerShell
[Convert]::ToBase64String([System.Security.Cryptography.RandomNumberGenerator]::GetBytes(32))
```

---

## Quick Deployment Script

Save as `deploy-azure.ps1`:
```powershell
# Azure Deployment Script
param(
    [string]$ResourceGroup = "codejudge-rg",
    [string]$Location = "southeastasia",
    [string]$ACRName = "codejudgeacr",
    [string]$DBPassword = "ChangeMe123!"
)

Write-Host "🚀 Deploying CodeJudge to Azure..." -ForegroundColor Cyan

# Login
az login

# Create resource group
Write-Host "Creating resource group..." -ForegroundColor Yellow
az group create --name $ResourceGroup --location $Location

# Create ACR
Write-Host "Creating Container Registry..." -ForegroundColor Yellow
az acr create --resource-group $ResourceGroup --name $ACRName --sku Basic

# Build and push
Write-Host "Building and pushing images..." -ForegroundColor Yellow
az acr login --name $ACRName
docker-compose -f docker-compose.monolith.yml build monolith
docker tag codejudge-monolith:latest "$ACRName.azurecr.io/codejudge-monolith:latest"
docker push "$ACRName.azurecr.io/codejudge-monolith:latest"

# Create PostgreSQL
Write-Host "Creating PostgreSQL database..." -ForegroundColor Yellow
az postgres flexible-server create `
  --resource-group $ResourceGroup `
  --name codejudge-db `
  --location $Location `
  --admin-user codejudgeadmin `
  --admin-password $DBPassword `
  --sku-name Standard_B1ms `
  --tier Burstable `
  --version 14 `
  --storage-size 32 `
  --public-access 0.0.0.0

az postgres flexible-server db create `
  --resource-group $ResourceGroup `
  --server-name codejudge-db `
  --database-name codejudgedb

# Create Redis
Write-Host "Creating Redis Cache..." -ForegroundColor Yellow
az redis create `
  --resource-group $ResourceGroup `
  --name codejudge-redis `
  --location $Location `
  --sku Basic `
  --vm-size c0

Write-Host "✅ Deployment complete!" -ForegroundColor Green
Write-Host "Configure your app with the connection strings from Azure Portal" -ForegroundColor Cyan
```

---

## Recommended Approach for Azure Student Account

1. **Start with Option 1 (Container Instances)** - Quickest to test
2. **Monitor costs** in Azure Portal
3. **Upgrade to App Service** if you need better performance
4. **Use AKS** only if you plan to scale the microservices architecture

---

## Monitoring & Management

### View Logs
```powershell
# Container Instances
az container logs --resource-group codejudge-rg --name codejudge-monolith

# App Service
az webapp log tail --resource-group codejudge-rg --name codejudge-app

# AKS
kubectl logs -l app=codejudge --tail=100
```

### Scale Resources
```powershell
# App Service
az appservice plan update --name codejudge-plan --resource-group codejudge-rg --sku B2

# AKS
kubectl scale deployment codejudge-monolith --replicas=3
```

---

## Cleanup (When Done Testing)

```powershell
# Delete entire resource group (all resources)
az group delete --name codejudge-rg --yes --no-wait
```

---

## Summary

✅ **Your project IS Azure-ready!** 

**Recommended path**:
1. Start with **Container Instances** ($40-50/month, 2 months free with credits)
2. Deploy monolith + PostgreSQL + Redis
3. Test with low traffic
4. Upgrade to App Service if needed

**Next Steps**:
1. Run `deploy-azure.ps1` script above
2. Monitor costs in Azure Portal
3. Set up budget alerts to avoid surprises

Need help with deployment? Let me know which option you'd like to pursue!
