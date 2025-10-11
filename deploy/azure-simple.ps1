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
    [string]$JWTSecret = ""
)

Write-Host "CodeJudge Azure Deployment" -ForegroundColor Cyan
Write-Host "==============================" -ForegroundColor Cyan

# Generate secrets if not provided
if ([string]::IsNullOrEmpty($DBPassword)) {
    Add-Type -AssemblyName 'System.Web'
    $DBPassword = "CJ_" + [System.Web.Security.Membership]::GeneratePassword(16, 4)
    Write-Host "Generated DB Password: $DBPassword" -ForegroundColor Yellow
}

if ([string]::IsNullOrEmpty($JWTSecret)) {
    $rng = [System.Security.Cryptography.RNGCryptoServiceProvider]::new()
    $bytes = New-Object byte[] 32
    $rng.GetBytes($bytes)
    $JWTSecret = [Convert]::ToBase64String($bytes)
    Write-Host "Generated JWT Secret: $JWTSecret" -ForegroundColor Yellow
}

# Check if logged in to Azure
Write-Host "`nChecking Azure CLI..." -ForegroundColor Yellow
try {
    $account = az account show 2>$null | ConvertFrom-Json
    Write-Host "Logged in as: $($account.user.name)" -ForegroundColor Green
    Write-Host "Subscription: $($account.name)" -ForegroundColor Green
} catch {
    Write-Host "Not logged in to Azure. Please run 'az login' first." -ForegroundColor Red
    exit 1
}

# Create Resource Group
Write-Host "`nCreating Resource Group: $ResourceGroup in $Location..." -ForegroundColor Yellow
az group create --name $ResourceGroup --location $Location --output none
if ($LASTEXITCODE -ne 0) {
    Write-Host "Failed to create Resource Group" -ForegroundColor Red
    exit 1
}
Write-Host "Resource Group created" -ForegroundColor Green

# Create Azure Container Registry
Write-Host "`nCreating Azure Container Registry: $ACRName..." -ForegroundColor Yellow
az acr create --resource-group $ResourceGroup --name $ACRName --sku Basic --output none
if ($LASTEXITCODE -ne 0) {
    Write-Host "Failed to create Container Registry" -ForegroundColor Red
    exit 1
}
Write-Host "Container Registry created" -ForegroundColor Green

# Enable admin access for ACR
az acr update -n $ACRName --admin-enabled true --output none

# Login to ACR
Write-Host "`nLogging in to Container Registry..." -ForegroundColor Yellow
az acr login --name $ACRName
if ($LASTEXITCODE -ne 0) {
    Write-Host "Failed to login to ACR" -ForegroundColor Red
    exit 1
}

# Build and Push Docker Image
Write-Host "`nBuilding Docker image..." -ForegroundColor Yellow
docker-compose build monolith
if ($LASTEXITCODE -ne 0) {
    Write-Host "Failed to build Docker image" -ForegroundColor Red
    exit 1
}

Write-Host "Pushing image to ACR..." -ForegroundColor Yellow
$imageTag = "$ACRName.azurecr.io/codejudge-monolith:latest"
docker tag codejudge-monolith:latest $imageTag
docker push $imageTag
if ($LASTEXITCODE -ne 0) {
    Write-Host "Failed to push image" -ForegroundColor Red
    exit 1
}
Write-Host "Image pushed successfully" -ForegroundColor Green

# Create PostgreSQL Database
Write-Host "`nCreating PostgreSQL Flexible Server: $DBName..." -ForegroundColor Yellow
Write-Host "This may take 5-10 minutes..." -ForegroundColor Yellow
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
    Write-Host "PostgreSQL server created" -ForegroundColor Green
} else {
    Write-Host "Failed to create PostgreSQL server" -ForegroundColor Red
    exit 1
}

# Create database
Write-Host "Creating database 'codejudgedb'..." -ForegroundColor Yellow
az postgres flexible-server db create `
  --resource-group $ResourceGroup `
  --server-name $DBName `
  --database-name codejudgedb `
  --output none

# Create Redis Cache
Write-Host "`nCreating Redis Cache: $RedisName..." -ForegroundColor Yellow
Write-Host "This may take 10-15 minutes..." -ForegroundColor Yellow
az redis create `
  --resource-group $ResourceGroup `
  --name $RedisName `
  --location $Location `
  --sku Basic `
  --vm-size c0 `
  --output none

if ($LASTEXITCODE -eq 0) {
    Write-Host "Redis cache created" -ForegroundColor Green
} else {
    Write-Host "Failed to create Redis cache" -ForegroundColor Red
    exit 1
}

# Get connection strings
Write-Host "`nRetrieving connection strings..." -ForegroundColor Yellow
$dbHost = "$DBName.postgres.database.azure.com"
$databaseUrl = "postgres://codejudgeadmin:$DBPassword@$dbHost:5432/codejudgedb?sslmode=require"

$redisKey = az redis list-keys --resource-group $ResourceGroup --name $RedisName --query primaryKey -o tsv
$redisUrl = "rediss://:$redisKey@$RedisName.redis.cache.windows.net:6380"

# Get ACR credentials
$acrUser = az acr credential show --name $ACRName --query username -o tsv
$acrPassword = az acr credential show --name $ACRName --query "passwords[0].value" -o tsv

# Deploy to Azure Container Instances
Write-Host "`nDeploying to Azure Container Instances..." -ForegroundColor Yellow

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
    Write-Host "`nDeployment successful!" -ForegroundColor Green
    $fqdn = az container show --resource-group $ResourceGroup --name $AppName --query ipAddress.fqdn -o tsv
    Write-Host "`nYour application is available at:" -ForegroundColor Cyan
    Write-Host "   http://$fqdn:8080" -ForegroundColor Green
} else {
    Write-Host "Deployment failed" -ForegroundColor Red
    exit 1
}

# Save configuration
$configFile = "azure-deployment-config.txt"
$configContent = "CodeJudge Azure Deployment Configuration`n"
$configContent += "=========================================`n"
$configContent += "Deployment Date: $(Get-Date)`n`n"
$configContent += "Resource Group: $ResourceGroup`n"
$configContent += "Location: $Location`n`n"
$configContent += "Container Registry: $ACRName`n"
$configContent += "Image: $imageTag`n`n"
$configContent += "Database Server: $DBName.postgres.database.azure.com`n"
$configContent += "Database Name: codejudgedb`n"
$configContent += "Database User: codejudgeadmin`n"
$configContent += "Database Password: $DBPassword`n`n"
$configContent += "Redis: $RedisName.redis.cache.windows.net`n`n"
$configContent += "JWT Secret: $JWTSecret`n`n"
$configContent += "Application URL: http://$fqdn:8080`n`n"
$configContent += "Useful Commands:`n"
$configContent += "  View logs:`n"
$configContent += "    az container logs --resource-group $ResourceGroup --name $AppName --follow`n"
$configContent += "  Delete all resources:`n"
$configContent += "    az group delete --name $ResourceGroup --yes --no-wait`n"

$configContent | Out-File -FilePath $configFile

Write-Host "`nConfiguration saved to: $configFile" -ForegroundColor Yellow

Write-Host "`nDeployment Summary:" -ForegroundColor Cyan
Write-Host "   Resource Group: $ResourceGroup" -ForegroundColor White
Write-Host "   Container Registry: $ACRName" -ForegroundColor White
Write-Host "   Database: $DBName" -ForegroundColor White
Write-Host "   Redis: $RedisName" -ForegroundColor White
Write-Host "   App Name: $AppName" -ForegroundColor White

Write-Host "`nDeployment Complete!" -ForegroundColor Green
