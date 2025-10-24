# CodeJudge Azure Deployment Script
# Optimized for cost-effective deployment

param(
    [string]$ResourceGroup = "codejudge-rg",
    [string]$Location = "southeastasia",
    [string]$ACRName = "codejudgeacr$(Get-Random -Minimum 1000 -Maximum 9999)",
    [string]$AppName = "codejudge-app-$(Get-Random -Minimum 1000 -Maximum 9999)",
    [string]$DBName = "codejudge-db-$(Get-Random -Minimum 1000 -Maximum 9999)",
    [string]$DBPassword = "",
    [string]$JWTSecret = ""
)

Write-Host "🚀 CodeJudge Azure Deployment" -ForegroundColor Cyan
Write-Host "==============================" -ForegroundColor Cyan

# Generate secrets if not provided
if ([string]::IsNullOrEmpty($DBPassword)) {
    Add-Type -AssemblyName 'System.Web'
    $DBPassword = "CJ_" + [System.Web.Security.Membership]::GeneratePassword(16, 4)
    Write-Host "Generated DB Password: $DBPassword" -ForegroundColor Yellow
}

if ([string]::IsNullOrEmpty($JWTSecret)) {
    $JWTSecret = [Convert]::ToBase64String([System.Security.Cryptography.RandomNumberGenerator]::GetBytes(32))
    Write-Host "Generated JWT Secret: $JWTSecret" -ForegroundColor Yellow
}

# Check if logged in to Azure
Write-Host "`n📋 Checking Azure CLI..." -ForegroundColor Yellow
try {
    $account = az account show 2>$null | ConvertFrom-Json
    Write-Host "✅ Logged in as: $($account.user.name)" -ForegroundColor Green
    Write-Host "   Subscription: $($account.name)" -ForegroundColor Green
} catch {
    Write-Host "❌ Not logged in to Azure. Please run 'az login' first." -ForegroundColor Red
    exit 1
}

# Create Resource Group
Write-Host "`n📦 Creating Resource Group: $ResourceGroup in $Location..." -ForegroundColor Yellow
az group create --name $ResourceGroup --location $Location --output none
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Failed to create Resource Group" -ForegroundColor Red
    exit 1
}
Write-Host "✅ Resource Group created" -ForegroundColor Green

# Create Azure Container Registry
Write-Host "`n🐳 Creating Azure Container Registry: $ACRName..." -ForegroundColor Yellow
az acr create --resource-group $ResourceGroup --name $ACRName --sku Basic --output none
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Failed to create Container Registry" -ForegroundColor Red
    exit 1
}
Write-Host "✅ Container Registry created" -ForegroundColor Green

# Enable admin access for ACR
az acr update -n $ACRName --admin-enabled true --output none

# Login to ACR
Write-Host "`n🔐 Logging in to Container Registry..." -ForegroundColor Yellow
az acr login --name $ACRName
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Failed to login to ACR" -ForegroundColor Red
    exit 1
}

# Build and Push Docker Image
Write-Host "`n🏗️  Building Docker image..." -ForegroundColor Yellow
docker-compose build monolith
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Failed to build Docker image" -ForegroundColor Red
    exit 1
}

Write-Host "📤 Pushing image to ACR..." -ForegroundColor Yellow
$imageTag = "$ACRName.azurecr.io/codejudge-monolith:latest"
docker tag codejudge-monolith:latest $imageTag
docker push $imageTag
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Failed to push image" -ForegroundColor Red
    exit 1
}
Write-Host "✅ Image pushed successfully" -ForegroundColor Green

# Create PostgreSQL Database
Write-Host "`n🗄️  Creating PostgreSQL Flexible Server: $DBName..." -ForegroundColor Yellow
Write-Host "   This may take 5-10 minutes..." -ForegroundColor Yellow
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
  --public-access All `
  --yes `
  --output none

if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ PostgreSQL server created" -ForegroundColor Green
} else {
    Write-Host "❌ Failed to create PostgreSQL server" -ForegroundColor Red
    exit 1
}

# Create database
Write-Host "   Creating database 'codejudgedb'..." -ForegroundColor Yellow
az postgres flexible-server db create `
  --resource-group $ResourceGroup `
  --server-name $DBName `
  --database-name codejudgedb `
  --output none

# Get connection strings
Write-Host "`n🔗 Retrieving connection strings..." -ForegroundColor Yellow
$dbHost = "$DBName.postgres.database.azure.com"
$databaseUrl = "postgres://codejudgeadmin:$DBPassword@$dbHost:5432/codejudgedb?sslmode=require"

# Note: Using in-memory Redis to minimize costs
Write-Host "   Using container-based Redis for cost optimization..." -ForegroundColor Yellow
$redisUrl = "redis://localhost:6379"

# Get ACR credentials
$acrUser = az acr credential show --name $ACRName --query username -o tsv
$acrPassword = az acr credential show --name $ACRName --query "passwords[0].value" -o tsv

# Deploy to Azure Container Instances
Write-Host "`n🚀 Deploying to Azure Container Instances..." -ForegroundColor Yellow

# Deploy with custom YAML for multi-container setup (includes Redis sidecar)
$containerGroupYaml = @"
apiVersion: 2021-10-01
location: $Location
name: $AppName
properties:
  containers:
  - name: monolith
    properties:
      image: $imageTag
      resources:
        requests:
          cpu: 0.5
          memoryInGb: 0.5
      ports:
      - port: 8080
        protocol: TCP
      environmentVariables:
      - name: DATABASE_URL
        secureValue: $databaseUrl
      - name: REDIS_URL
        value: redis://localhost:6379
      - name: JWT_SECRET
        secureValue: $JWTSecret
      - name: PORT
        value: 8080
      - name: GIN_MODE
        value: release
  - name: redis
    properties:
      image: redis:7-alpine
      resources:
        requests:
          cpu: 0.25
          memoryInGb: 0.25
      ports:
      - port: 6379
        protocol: TCP
  osType: Linux
  ipAddress:
    type: Public
    ports:
    - protocol: TCP
      port: 8080
    dnsNameLabel: $AppName
  imageRegistryCredentials:
  - server: $ACRName.azurecr.io
    username: $acrUser
    password: $acrPassword
tags: {}
"@

$yamlFile = "container-group.yaml"
$containerGroupYaml | Out-File -FilePath $yamlFile -Encoding UTF8

az container create `
  --resource-group $ResourceGroup `
  --file $yamlFile

Remove-Item $yamlFile

if ($LASTEXITCODE -eq 0) {
    Write-Host "`n✅ Deployment successful!" -ForegroundColor Green
    $fqdn = az container show --resource-group $ResourceGroup --name $AppName --query ipAddress.fqdn -o tsv
    Write-Host "`n🌐 Your application is available at:" -ForegroundColor Cyan
    Write-Host "   http://$($fqdn):8080" -ForegroundColor Green
} else {
    Write-Host "❌ Deployment failed" -ForegroundColor Red
    exit 1
}

# Save configuration
$configFile = "azure-deployment-config.txt"
$configContent = @"
CodeJudge Azure Deployment Configuration
=========================================
Deployment Date: $(Get-Date)

Resource Group: $ResourceGroup
Location: $Location

Container Registry: $ACRName
Image: $imageTag

Database Server: $DBName.postgres.database.azure.com
Database Name: codejudgedb
Database User: codejudgeadmin
Database Password: $DBPassword

JWT Secret: $JWTSecret

Application URL: http://$($fqdn):8080

Cost Optimization Notes:
- Using container-based Redis (no separate Azure Redis Cache)
- PostgreSQL: Standard_B1ms (Burstable tier)
- Container Instance: 0.75 vCPU, 0.75GB RAM
- Estimated cost: ~$20-25/month

Useful Commands:
  View logs:
    az container logs --resource-group $ResourceGroup --name $AppName --follow
  Delete all resources:
    az group delete --name $ResourceGroup --yes --no-wait
"@

$configContent | Out-File -FilePath $configFile

Write-Host "`n📝 Configuration saved to: $configFile" -ForegroundColor Yellow

Write-Host "`n📊 Deployment Summary:" -ForegroundColor Cyan
Write-Host "   Resource Group: $ResourceGroup" -ForegroundColor White
Write-Host "   Location: $Location" -ForegroundColor White
Write-Host "   Container Registry: $ACRName" -ForegroundColor White
Write-Host "   Database: $DBName" -ForegroundColor White
Write-Host "   App Name: $AppName" -ForegroundColor White

Write-Host "`n✅ Deployment Complete! 🎉" -ForegroundColor Green
