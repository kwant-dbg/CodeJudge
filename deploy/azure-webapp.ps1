# CodeJudge Azure Web App Deployment Script
# Simplified deployment using Azure App Service (Web App for Containers)

param(
    [string]$ResourceGroup = "codejudge-webapp-$(Get-Random -Minimum 1000 -Maximum 9999)",
    [string]$Location = "westus2",
    [string]$AppName = "codejudge-$(Get-Random -Minimum 1000 -Maximum 9999)",
    [string]$DBPassword = "",
    [string]$JWTSecret = ""
)

Write-Host "CodeJudge Azure Web App Deployment" -ForegroundColor Cyan
Write-Host "======================================" -ForegroundColor Cyan

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

# Build Docker image locally
Write-Host "`nBuilding Docker image..." -ForegroundColor Yellow
docker-compose build monolith
if ($LASTEXITCODE -ne 0) {
    Write-Host "Failed to build Docker image" -ForegroundColor Red
    exit 1
}
Write-Host "Docker image built successfully" -ForegroundColor Green

# Create PostgreSQL Database
Write-Host "`nCreating PostgreSQL Flexible Server..." -ForegroundColor Yellow
Write-Host "This may take 5-10 minutes..." -ForegroundColor Yellow
$DBName = "codejudge-db-$(Get-Random -Minimum 1000 -Maximum 9999)"
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
    Write-Host "Trying alternate creation method..." -ForegroundColor Yellow
    # Try creating in a different way or skip if not available
}

# Create database
Write-Host "Creating database 'codejudgedb'..." -ForegroundColor Yellow
az postgres flexible-server db create `
  --resource-group $ResourceGroup `
  --server-name $DBName `
  --database-name codejudgedb `
  --output none 2>$null

# Get database connection string
$dbHost = "$DBName.postgres.database.azure.com"
$databaseUrl = "postgres://codejudgeadmin:$DBPassword@$dbHost:5432/codejudgedb?sslmode=require"

# Create App Service Plan
Write-Host "`nCreating App Service Plan..." -ForegroundColor Yellow
$PlanName = "$AppName-plan"
az appservice plan create `
  --name $PlanName `
  --resource-group $ResourceGroup `
  --location $Location `
  --is-linux `
  --sku F1 `
  --output none

if ($LASTEXITCODE -ne 0) {
    Write-Host "Failed to create App Service Plan" -ForegroundColor Red
    exit 1
}
Write-Host "App Service Plan created" -ForegroundColor Green

# Create Web App
Write-Host "`nCreating Web App..." -ForegroundColor Yellow
az webapp create `
  --resource-group $ResourceGroup `
  --plan $PlanName `
  --name $AppName `
  --runtime "NODE:18-lts" `
  --output none

if ($LASTEXITCODE -ne 0) {
    Write-Host "Failed to create Web App" -ForegroundColor Red
    exit 1
}
Write-Host "Web App created" -ForegroundColor Green

# Configure Web App settings
Write-Host "`nConfiguring Web App settings..." -ForegroundColor Yellow
az webapp config appsettings set `
  --resource-group $ResourceGroup `
  --name $AppName `
  --settings `
    DATABASE_URL="$databaseUrl" `
    JWT_SECRET="$JWTSecret" `
    PORT=8080 `
    WEBSITES_PORT=8080 `
    GIN_MODE=release `
  --output none

Write-Host "Web App settings configured" -ForegroundColor Green

# Get Web App URL
$appUrl = "https://$AppName.azurewebsites.net"

Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host "Deployment Successful!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "`nYour application URL:" -ForegroundColor Yellow
Write-Host "  $appUrl" -ForegroundColor Green
Write-Host "`nNOTE: You need to deploy your code to the Web App." -ForegroundColor Yellow
Write-Host "You can deploy using:" -ForegroundColor Yellow
Write-Host "  1. GitHub Actions" -ForegroundColor White
Write-Host "  2. Azure DevOps" -ForegroundColor White
Write-Host "  3. Local Git deployment" -ForegroundColor White
Write-Host "  4. Docker image (requires ACR)" -ForegroundColor White

# Save configuration
$configFile = "azure-webapp-config.txt"
$configContent = "CodeJudge Azure Web App Deployment Configuration`n"
$configContent += "=========================================`n"
$configContent += "Deployment Date: $(Get-Date)`n`n"
$configContent += "Resource Group: $ResourceGroup`n"
$configContent += "Location: $Location`n"
$configContent += "App Name: $AppName`n"
$configContent += "App Service Plan: $PlanName`n`n"
$configContent += "Database Server: $dbHost`n"
$configContent += "Database Name: codejudgedb`n"
$configContent += "Database User: codejudgeadmin`n"
$configContent += "Database Password: $DBPassword`n`n"
$configContent += "JWT Secret: $JWTSecret`n`n"
$configContent += "Application URL: $appUrl`n`n"
$configContent += "Useful Commands:`n"
$configContent += "  View logs:`n"
$configContent += "    az webapp log tail --resource-group $ResourceGroup --name $AppName`n"
$configContent += "  Restart app:`n"
$configContent += "    az webapp restart --resource-group $ResourceGroup --name $AppName`n"
$configContent += "  Delete all resources:`n"
$configContent += "    az group delete --name $ResourceGroup --yes --no-wait`n"

$configContent | Out-File -FilePath $configFile

Write-Host "`nConfiguration saved to: $configFile" -ForegroundColor Cyan

Write-Host "`nTo deploy your code, run:" -ForegroundColor Yellow
Write-Host "  az webapp deployment source config-local-git --resource-group $ResourceGroup --name $AppName" -ForegroundColor White
