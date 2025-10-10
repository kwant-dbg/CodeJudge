# Azure Quick Start - CodeJudge

## ✅ Yes, your project is Azure-ready!

## Option 1: One-Command Deployment (Easiest)

### Prerequisites
- Azure Student Account (with $100 credits)
- Azure CLI installed: `winget install Microsoft.AzureCLI`
- Docker Desktop running

### Deploy Now
```powershell
# Login to Azure
az login

# Deploy everything (takes ~20 minutes)
.\deploy-azure.ps1 -DeploymentType aci

# Or for App Service (better for production):
.\deploy-azure.ps1 -DeploymentType appservice
```

That's it! The script will:
1. ✅ Create all Azure resources
2. ✅ Build and push your Docker image
3. ✅ Set up PostgreSQL database
4. ✅ Set up Redis cache
5. ✅ Deploy your application
6. ✅ Give you the URL to access it

## Option 2: Manual Step-by-Step

See `AZURE_DEPLOYMENT.md` for detailed instructions.

## Cost Estimate

With Azure Student ($100 credits):

- **Container Instances**: ~$45/month → **2+ months free**
- **App Service**: ~$55/month → **1.8 months free**

All costs include:
- Web hosting
- PostgreSQL database
- Redis cache
- Container registry

## After Deployment

### View Your App
The script will output your app URL:
- Container Instances: `http://codejudge-app-XXXX.eastus.azurecontainer.io:8080`
- App Service: `https://codejudge-app-XXXX.azurewebsites.net`

### View Logs
```powershell
# Container Instances
az container logs --resource-group codejudge-rg --name codejudge-app-XXXX --follow

# App Service
az webapp log tail --resource-group codejudge-rg --name codejudge-app-XXXX
```

### Monitor Costs
1. Go to [Azure Portal](https://portal.azure.com)
2. Navigate to "Cost Management + Billing"
3. Set up budget alerts

### Update Your App
```powershell
# Rebuild and redeploy
docker-compose -f docker-compose.monolith.yml build monolith
.\deploy-azure.ps1 -DeploymentType aci
```

### Clean Up (Delete Everything)
```powershell
az group delete --name codejudge-rg --yes --no-wait
```

## Troubleshooting

### "Image not found" error
```powershell
# Rebuild and push
docker-compose -f docker-compose.monolith.yml build monolith
az acr login --name codejudgeacrXXXX
docker tag codejudge-monolith:latest codejudgeacrXXXX.azurecr.io/codejudge-monolith:latest
docker push codejudgeacrXXXX.azurecr.io/codejudge-monolith:latest
```

### Database connection issues
Make sure firewall rules allow Azure services:
```powershell
az postgres flexible-server firewall-rule create `
  --resource-group codejudge-rg `
  --name codejudge-db-XXXX `
  --rule-name AllowAzure `
  --start-ip-address 0.0.0.0 `
  --end-ip-address 0.0.0.0
```

### App not starting
Check logs for errors:
```powershell
az container logs --resource-group codejudge-rg --name codejudge-app-XXXX
```

## What's Included

Your Azure deployment includes:
- ✅ Monolith service with all features (auth, problems, submissions, plagiarism)
- ✅ PostgreSQL 14 database (managed)
- ✅ Redis cache (managed)
- ✅ Dark/Light theme toggle
- ✅ Automatic scaling (App Service)
- ✅ SSL/HTTPS (App Service)
- ✅ Health monitoring

## What's NOT Included (Yet)

- ❌ Judge service (C++ sandbox requires special permissions)
  - **Workaround**: Keep judge running locally with Redis connection to Azure

## Judge Service Setup (Hybrid Approach)

If you want to use the judge service:

1. **Keep judge running locally**
2. **Connect to Azure Redis**:

Update your local `docker-compose.monolith.yml`:
```yaml
judge:
  environment:
    - REDIS_URL=rediss://:YOUR_REDIS_KEY@codejudge-redis-XXXX.redis.cache.windows.net:6380
```

Get Redis key:
```powershell
az redis list-keys --resource-group codejudge-rg --name codejudge-redis-XXXX --query primaryKey -o tsv
```

This way:
- Web app runs on Azure (accessible from anywhere)
- Judge runs locally (secure sandbox execution)
- They communicate via Redis queue

## Next Steps

1. ✅ Run `.\deploy-azure.ps1`
2. ✅ Wait ~20 minutes for deployment
3. ✅ Visit your app URL
4. ✅ Create your first problem
5. ✅ Set up budget alerts in Azure Portal

## Need Help?

- Azure Portal: https://portal.azure.com
- Azure CLI Docs: https://docs.microsoft.com/cli/azure/
- Your deployment config is saved in `azure-deployment-config.txt`

---

**Ready to deploy?** Just run:
```powershell
.\deploy-azure.ps1
```
