# CodeJudge Azure Deployment for Southeast Asia
# Simplified deployment script

Write-Host "`n==================================" -ForegroundColor Cyan
Write-Host "CodeJudge Azure Deployment" -ForegroundColor Cyan
Write-Host "Using Southeast Asia Region" -ForegroundColor Cyan
Write-Host "==================================" -ForegroundColor Cyan

# Use Southeast Asia since it's proven to work
$ResourceGroup = "codejudge-k8s-rg"
$Location = "southeastasia"
$AppName = "codejudge-app-$(Get-Random -Minimum 1000 -Maximum 9999)"
$PlanName = "codejudge-plan"

# Generate secrets
Add-Type -AssemblyName 'System.Web'
$DBPassword = "CJ" + [System.Web.Security.Membership]::GeneratePassword(14, 3)
$DBPassword = $DBPassword -replace '[\\|;]', 'A'

$rng = [System.Security.Cryptography.RNGCryptoServiceProvider]::new()
$bytes = New-Object byte[] 32
$rng.GetBytes($bytes)
$JWTSecret = [Convert]::ToBase64String($bytes)

Write-Host "`nGenerated Secrets:" -ForegroundColor Yellow
Write-Host "  DB Password: $DBPassword" -ForegroundColor Gray
Write-Host "  JWT Secret: $JWTSecret" -ForegroundColor Gray

# Check Azure login
Write-Host "`n[1/4] Checking Azure login..." -ForegroundColor Cyan
try {
    $account = az account show 2>$null | ConvertFrom-Json
    Write-Host "      Logged in as: $($account.user.name)" -ForegroundColor Green
} catch {
    Write-Host "      ERROR: Not logged in" -ForegroundColor Red
    exit 1
}

# Resource group already exists, so we skip creating it
Write-Host "`n[2/4] Using existing resource group: $ResourceGroup" -ForegroundColor Cyan

# Create App Service Plan (Basic B1 - better than Free for student credits)
Write-Host "`n[3/4] Creating App Service Plan..." -ForegroundColor Cyan
az appservice plan create `
  --name $PlanName `
  --resource-group $ResourceGroup `
  --location $Location `
  --is-linux `
  --sku B1 `
  --output none 2>$null

if ($LASTEXITCODE -ne 0) {
    Write-Host "      Plan might already exist, checking..." -ForegroundColor Yellow
    $existingPlan = az appservice plan show --name $PlanName --resource-group $ResourceGroup 2>$null
    if ($null -ne $existingPlan) {
        Write-Host "      Using existing App Service Plan" -ForegroundColor Green
    } else {
        Write-Host "      ERROR: Failed to create App Service Plan" -ForegroundColor Red
        exit 1
    }
} else {
    Write-Host "      App Service Plan created" -ForegroundColor Green
}

# Create Web App (using Docker container)
Write-Host "`n[4/4] Creating Web App with Docker..." -ForegroundColor Cyan
az webapp create `
  --resource-group $ResourceGroup `
  --plan $PlanName `
  --name $AppName `
  --deployment-container-image-name "nginx:alpine" `
  --output none

if ($LASTEXITCODE -ne 0) {
    Write-Host "      ERROR: Failed to create Web App" -ForegroundColor Red
    Write-Host "      The name might be taken. Try running the script again." -ForegroundColor Yellow
    exit 1
}
Write-Host "      Web App created successfully!" -ForegroundColor Green

# Configure App Settings
Write-Host "`nConfiguring app settings..." -ForegroundColor Cyan
az webapp config appsettings set `
  --resource-group $ResourceGroup `
  --name $AppName `
  --settings `
    JWT_SECRET="$JWTSecret" `
    PORT=8080 `
    WEBSITES_PORT=8080 `
    GIN_MODE=release `
  --output none

$appUrl = "https://$AppName.azurewebsites.net"

Write-Host "`n=================================="-ForegroundColor Green
Write-Host "  DEPLOYMENT SUCCESSFUL!" -ForegroundColor Green
Write-Host "==================================" -ForegroundColor Green

Write-Host "`nYour App Details:" -ForegroundColor Cyan
Write-Host "  URL: $appUrl" -ForegroundColor White
Write-Host "  Resource Group: $ResourceGroup" -ForegroundColor White
Write-Host "  Location: $Location" -ForegroundColor White

Write-Host "`nNext: Deploy your code" -ForegroundColor Yellow
Write-Host "Option 1 - Using Local Git:" -ForegroundColor White
Write-Host "  1. Get the Git URL:" -ForegroundColor Gray
Write-Host "     `$gitUrl = az webapp deployment source config-local-git --resource-group $ResourceGroup --name $AppName --query url -o tsv" -ForegroundColor Gray
Write-Host "  2. Add remote and push:" -ForegroundColor Gray
Write-Host "     cd monolith" -ForegroundColor Gray
Write-Host "     git init" -ForegroundColor Gray
Write-Host "     git add ." -ForegroundColor Gray
Write-Host "     git commit -m 'Deploy to Azure'" -ForegroundColor Gray
Write-Host "     git remote add azure `$gitUrl" -ForegroundColor Gray
Write-Host "     git push azure master" -ForegroundColor Gray

Write-Host "`nOption 2 - Using ZIP Deploy (Simpler):" -ForegroundColor White
Write-Host "  az webapp deployment source config-zip --resource-group $ResourceGroup --name $AppName --src monolith.zip" -ForegroundColor Gray

Write-Host "`nUseful Commands:" -ForegroundColor Yellow
Write-Host "  Browse: az webapp browse --resource-group $ResourceGroup --name $AppName" -ForegroundColor Gray
Write-Host "  Logs: az webapp log tail --resource-group $ResourceGroup --name $AppName" -ForegroundColor Gray
Write-Host "  Restart: az webapp restart --resource-group $ResourceGroup --name $AppName" -ForegroundColor Gray

# Save configuration
$configFile = "azure-deployment-info.txt"
@"
CodeJudge Azure Deployment
==========================
Deployed: $(Get-Date)

Application URL: $appUrl
Resource Group: $ResourceGroup
App Name: $AppName
Location: $Location

JWT Secret: $JWTSecret

To deploy code:
  az webapp deployment source config-local-git --resource-group $ResourceGroup --name $AppName

View logs:
  az webapp log tail --resource-group $ResourceGroup --name $AppName

Delete resources:
  az webapp delete --resource-group $ResourceGroup --name $AppName
  az appservice plan delete --resource-group $ResourceGroup --name $PlanName
"@ | Out-File -FilePath $configFile

Write-Host "`nConfiguration saved to: $configFile" -ForegroundColor Cyan
Write-Host ""
