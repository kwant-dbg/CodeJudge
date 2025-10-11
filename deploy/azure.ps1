# CodeJudge Azure Deployment Script
# Simplified deployment for Azure Student accounts

param(
    [string]$ResourceGroup = "codejudge-rg",
    [string]$Location = "eastus",
    [string]$ACRName = "codejudgeacr$(Get-Random -Minimum 1000 -Maximum 9999)",
    [string]$AppName = "codejudge-app-$(Get-Random -Minimum 1000 -Maximum 9999)",
    [string]$DBName = "codejudge-db-$(Get-Random -Minimum 1000 -Maximum 9999)",
    [string]$RedisName = "codejudge-redis-$(Get-Random -Minimum 1000 -Maximum 9999)",
    [string]$DBPassword = "",
    [string]$JWTSecret = "",
    [ValidateSet("aci", "appservice", "aks")]
    [string]$DeploymentType = "aci"
)

# Colors
function Write-ColorOutput($ForegroundColor) {
    $fc = $host.UI.RawUI.ForegroundColor
    $host.UI.RawUI.ForegroundColor = $ForegroundColor
    if ($args) {
        Write-Output $args
    }
    $host.UI.RawUI.ForegroundColor = $fc
}

Write-ColorOutput Cyan "🚀 CodeJudge Azure Deployment"
Write-ColorOutput Cyan "=============================="

# Generate secrets if not provided
if ([string]::IsNullOrEmpty($DBPassword)) {
    $DBPassword = "CJ_" + [System.Web.Security.Membership]::GeneratePassword(16, 4)
    Write-ColorOutput Yellow "Generated DB Password: $DBPassword"
}

if ([string]::IsNullOrEmpty($JWTSecret)) {
    $JWTSecret = [Convert]::ToBase64String([System.Security.Cryptography.RandomNumberGenerator]::GetBytes(32))
    Write-ColorOutput Yellow "Generated JWT Secret: $JWTSecret"
}

# Check if logged in to Azure
Write-ColorOutput Yellow "`n📋 Checking Azure CLI..."
$account = az account show 2>$null | ConvertFrom-Json
if (-not $account) {
    Write-ColorOutput Red "❌ Not logged in to Azure. Running 'az login'..."
    az login
    if ($LASTEXITCODE -ne 0) {
        Write-ColorOutput Red "❌ Azure login failed. Exiting."
        exit 1
    }
} else {
    Write-ColorOutput Green "✅ Logged in as: $($account.user.name)"
    Write-ColorOutput Green "   Subscription: $($account.name)"
}

# Create Resource Group
Write-ColorOutput Yellow "`n📦 Creating Resource Group: $ResourceGroup in $Location..."
az group create --name $ResourceGroup --location $Location --output none
if ($LASTEXITCODE -eq 0) {
    Write-ColorOutput Green "✅ Resource Group created"
} else {
    Write-ColorOutput Red "❌ Failed to create Resource Group"
    exit 1
}

# Create Azure Container Registry
Write-ColorOutput Yellow "`n🐳 Creating Azure Container Registry: $ACRName..."
az acr create --resource-group $ResourceGroup --name $ACRName --sku Basic --output none
if ($LASTEXITCODE -eq 0) {
    Write-ColorOutput Green "✅ Container Registry created"
} else {
    Write-ColorOutput Red "❌ Failed to create Container Registry"
    exit 1
}

# Enable admin access for ACR
az acr update -n $ACRName --admin-enabled true --output none

# Login to ACR
Write-ColorOutput Yellow "`n🔐 Logging in to Container Registry..."
az acr login --name $ACRName
if ($LASTEXITCODE -ne 0) {
    Write-ColorOutput Red "❌ Failed to login to ACR"
    exit 1
}

# Build and Push Docker Image
Write-ColorOutput Yellow "`n🏗️  Building Docker image..."
docker-compose build monolith
if ($LASTEXITCODE -ne 0) {
    Write-ColorOutput Red "❌ Failed to build Docker image"
    exit 1
}

Write-ColorOutput Yellow "📤 Pushing image to ACR..."
$imageTag = "$ACRName.azurecr.io/codejudge-monolith:latest"
docker tag codejudge-monolith:latest $imageTag
docker push $imageTag
if ($LASTEXITCODE -ne 0) {
    Write-ColorOutput Red "❌ Failed to push image"
    exit 1
}
Write-ColorOutput Green "✅ Image pushed successfully"

# Create PostgreSQL Database
Write-ColorOutput Yellow "`n🗄️  Creating PostgreSQL Flexible Server: $DBName..."
Write-ColorOutput Yellow "   This may take 5-10 minutes..."
az postgres flexible-server create `
  --resource-group $ResourceGroup `
  --name $DBName `
  --location $Location `
  --admin-user codejudgeadmin `
  --admin-password $DBPassword `
  --sku-name Standard_B1ms `
  --tier Burstable `
  --version 14 `
  --storage-size 32 `
  --public-access 0.0.0.0-255.255.255.255 `
  --yes `
  --output none

if ($LASTEXITCODE -eq 0) {
    Write-ColorOutput Green "✅ PostgreSQL server created"
} else {
    Write-ColorOutput Red "❌ Failed to create PostgreSQL server"
    exit 1
}

# Create database
Write-ColorOutput Yellow "   Creating database 'codejudgedb'..."
az postgres flexible-server db create `
  --resource-group $ResourceGroup `
  --server-name $DBName `
  --database-name codejudgedb `
  --output none

# Create Redis Cache
Write-ColorOutput Yellow "`n🔴 Creating Redis Cache: $RedisName..."
Write-ColorOutput Yellow "   This may take 10-15 minutes..."
az redis create `
  --resource-group $ResourceGroup `
  --name $RedisName `
  --location $Location `
  --sku Basic `
  --vm-size c0 `
  --output none

if ($LASTEXITCODE -eq 0) {
    Write-ColorOutput Green "✅ Redis cache created"
} else {
    Write-ColorOutput Red "❌ Failed to create Redis cache"
    exit 1
}

# Get connection strings
Write-ColorOutput Yellow "`n🔗 Retrieving connection strings..."
$dbHost = "$DBName.postgres.database.azure.com"
$databaseUrl = "postgres://codejudgeadmin:$DBPassword@$dbHost:5432/codejudgedb?sslmode=require"

$redisKey = az redis list-keys --resource-group $ResourceGroup --name $RedisName --query primaryKey -o tsv
$redisUrl = "rediss://:$redisKey@$RedisName.redis.cache.windows.net:6380"

# Get ACR credentials
$acrUser = az acr credential show --name $ACRName --query username -o tsv
$acrPassword = az acr credential show --name $ACRName --query "passwords[0].value" -o tsv

# Deploy based on type
if ($DeploymentType -eq "aci") {
    Write-ColorOutput Yellow "`n🚀 Deploying to Azure Container Instances..."
    
    az container create `
      --resource-group $ResourceGroup `
      --name $AppName `
      --image $imageTag `
      --registry-login-server "$ACRName.azurecr.io" `
      --registry-username $acrUser `
      --registry-password $acrPassword `
      --dns-name-label $AppName `
      --ports 8080 `
      --environment-variables `
        DATABASE_URL="$databaseUrl" `
        REDIS_URL="$redisUrl" `
        JWT_SECRET="$JWTSecret" `
        PORT=8080 `
        GIN_MODE=release `
      --cpu 1 --memory 1.5 `
      --output none

    if ($LASTEXITCODE -eq 0) {
        Write-ColorOutput Green "`n✅ Deployment successful!"
        $fqdn = az container show --resource-group $ResourceGroup --name $AppName --query ipAddress.fqdn -o tsv
        Write-ColorOutput Cyan "`n🌐 Your application is available at:"
        Write-ColorOutput Green "   http://$fqdn:8080"
    } else {
        Write-ColorOutput Red "❌ Deployment failed"
        exit 1
    }
    
} elseif ($DeploymentType -eq "appservice") {
    Write-ColorOutput Yellow "`n🚀 Deploying to Azure App Service..."
    
    # Create App Service Plan
    Write-ColorOutput Yellow "   Creating App Service Plan..."
    az appservice plan create `
      --name "$AppName-plan" `
      --resource-group $ResourceGroup `
      --location $Location `
      --is-linux `
      --sku B1 `
      --output none

    # Create Web App
    Write-ColorOutput Yellow "   Creating Web App..."
    az webapp create `
      --resource-group $ResourceGroup `
      --plan "$AppName-plan" `
      --name $AppName `
      --deployment-container-image-name $imageTag `
      --output none

    # Configure settings
    Write-ColorOutput Yellow "   Configuring app settings..."
    az webapp config appsettings set `
      --resource-group $ResourceGroup `
      --name $AppName `
      --settings `
        DATABASE_URL="$databaseUrl" `
        REDIS_URL="$redisUrl" `
        JWT_SECRET="$JWTSecret" `
        PORT=8080 `
        WEBSITES_PORT=8080 `
        GIN_MODE=release `
      --output none

    # Configure container
    az webapp config container set `
      --name $AppName `
      --resource-group $ResourceGroup `
      --docker-custom-image-name $imageTag `
      --docker-registry-server-url "https://$ACRName.azurecr.io" `
      --docker-registry-server-user $acrUser `
      --docker-registry-server-password $acrPassword `
      --output none

    if ($LASTEXITCODE -eq 0) {
        Write-ColorOutput Green "`n✅ Deployment successful!"
        Write-ColorOutput Cyan "`n🌐 Your application is available at:"
        Write-ColorOutput Green "   https://$AppName.azurewebsites.net"
    } else {
        Write-ColorOutput Red "❌ Deployment failed"
        exit 1
    }
}

# Save configuration
$configFile = "azure-deployment-config.txt"
@"
CodeJudge Azure Deployment Configuration
=========================================
Deployment Date: $(Get-Date)
Deployment Type: $DeploymentType

Resource Group: $ResourceGroup
Location: $Location

Container Registry: $ACRName
Image: $imageTag

Database Server: $DBName.postgres.database.azure.com
Database Name: codejudgedb
Database User: codejudgeadmin
Database Password: $DBPassword

Redis: $RedisName.redis.cache.windows.net

JWT Secret: $JWTSecret

Application URL:
"@ | Out-File -FilePath $configFile

if ($DeploymentType -eq "aci") {
    $fqdn = az container show --resource-group $ResourceGroup --name $AppName --query ipAddress.fqdn -o tsv
    "  http://$fqdn:8080" | Out-File -FilePath $configFile -Append
} else {
    "  https://$AppName.azurewebsites.net" | Out-File -FilePath $configFile -Append
}

Write-ColorOutput Yellow "`n📝 Configuration saved to: $configFile"

Write-ColorOutput Cyan "`n📊 Deployment Summary:"
Write-ColorOutput White "   Resource Group: $ResourceGroup"
Write-ColorOutput White "   Container Registry: $ACRName"
Write-ColorOutput White "   Database: $DBName"
Write-ColorOutput White "   Redis: $RedisName"
Write-ColorOutput White "   App Name: $AppName"

Write-ColorOutput Cyan "`n🔧 Useful Commands:"
Write-ColorOutput White "   View logs:"
if ($DeploymentType -eq "aci") {
    Write-ColorOutput White "     az container logs --resource-group $ResourceGroup --name $AppName --follow"
} else {
    Write-ColorOutput White "     az webapp log tail --resource-group $ResourceGroup --name $AppName"
}
Write-ColorOutput White "   Delete all resources:"
Write-ColorOutput White "     az group delete --name $ResourceGroup --yes --no-wait"

Write-ColorOutput Green "`n✅ Deployment Complete! 🎉"
