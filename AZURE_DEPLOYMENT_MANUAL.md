# CodeJudge - Manual Azure Deployment Guide
===============================================

## Prerequisites
- Azure CLI installed and logged in (az login)
- Docker installed and running
- Your codejudge project in d:\dev\codejudge

## Step-by-Step Deployment

### Step 1: Create Web App
```powershell
# Set variables
$appName = "codejudge-$(Get-Random -Minimum 10000 -Maximum 99999)"
$resourceGroup = "codejudge-k8s-rg"
$plan = "codejudge-plan"

# Create the web app (try different runtimes if one fails)
az webapp create `
  --resource-group $resourceGroup `
  --plan $plan `
  --name $appName `
  --runtime "PYTHON:3.11"

# Save the app name
echo $appName | Out-File -FilePath "deployed-app-name.txt"
Write-Host "App created: $appName"
Write-Host "URL: https://$appName.azurewebsites.net"
```

### Step 2: Configure App Settings
```powershell
# Generate secrets
Add-Type -AssemblyName 'System.Web'
$JWTSecret = [System.Web.Security.Membership]::GeneratePassword(32, 8)

# Set environment variables
az webapp config appsettings set `
  --resource-group $resourceGroup `
  --name $appName `
  --settings `
    JWT_SECRET="$JWTSecret" `
    PORT=8080 `
    WEBSITES_PORT=8080 `
    GIN_MODE=release
```

### Step 3: Deploy Using ZIP
```powershell
# Option A: Build and deploy your Go app as a ZIP

# 1. Build your Go application
cd d:\dev\codejudge\monolith
go build -o codejudge-monolith .

# 2. Create a startup script
@"
#!/bin/sh
./codejudge-monolith
"@ | Out-File -FilePath startup.sh -Encoding ASCII

# 3. Create ZIP
Compress-Archive -Path .\codejudge-monolith, .\startup.sh, .\static\* -DestinationPath ..\deploy.zip -Force

# 4. Deploy ZIP
cd ..
az webapp deploy `
  --resource-group $resourceGroup `
  --name $appName `
  --src-path deploy.zip `
  --type zip
```

### Step 4 (Alternative): Deploy Using Docker Hub
```powershell
# If you have a Docker Hub account:

# 1. Tag and push your image
docker tag codejudge-monolith:latest YOUR_DOCKERHUB_USERNAME/codejudge:latest
docker push YOUR_DOCKERHUB_USERNAME/codejudge:latest

# 2. Configure web app to use your Docker image
az webapp config container set `
  --name $appName `
  --resource-group $resourceGroup `
  --docker-custom-image-name YOUR_DOCKERHUB_USERNAME/codejudge:latest
```

### Step 5: Enable Logging
```powershell
# Enable application logging
az webapp log config `
  --resource-group $resourceGroup `
  --name $appName `
  --application-logging true `
  --level information

# Stream logs
az webapp log tail `
  --resource-group $resourceGroup `
  --name $appName
```

### Step 6: Test Your Deployment
```powershell
# Open in browser
az webapp browse --resource-group $resourceGroup --name $appName

# Or visit directly:
# https://YOUR_APP_NAME.azurewebsites.net
```

## Troubleshooting

### If the app doesn't start:
```powershell
# Check logs
az webapp log tail --resource-group $resourceGroup --name $appName

# Restart the app
az webapp restart --resource-group $resourceGroup --name $appName

# Check app status
az webapp show --resource-group $resourceGroup --name $appName --query "{State:state, URL:defaultHostName}"
```

### If you need to change runtime:
```powershell
# For Python apps, you can create a requirements.txt and deploy
# For Node.js:
az webapp config set --resource-group $resourceGroup --name $appName --linux-fx-version "NODE|18-lts"

# For Docker:
az webapp config set --resource-group $resourceGroup --name $appName --linux-fx-version "DOCKER|nginx:alpine"
```

## Quick Commands Reference

### View all your apps:
```powershell
az webapp list --resource-group $resourceGroup --output table
```

### Delete an app:
```powershell
az webapp delete --resource-group $resourceGroup --name $appName
```

### Delete everything:
```powershell
# This will delete the entire resource group
az group delete --name $resourceGroup --yes --no-wait
```

## Cost Management
- B1 App Service Plan: ~$13/month
- With $100 Azure Student credits, you get 7+ months free
- Stop/start app service plan to save costs:
```powershell
# Stop (saves money)
az appservice plan update --name $plan --resource-group $resourceGroup --sku FREE

# Or delete when not needed
az appservice plan delete --name $plan --resource-group $resourceGroup
```

## Notes
- The App Service Plan "codejudge-plan" is already created in Southeast Asia
- You can create multiple web apps on the same plan at no extra cost
- Free tier (F1) is available but has limitations
- B1 tier gives you better performance and is still cost-effective
