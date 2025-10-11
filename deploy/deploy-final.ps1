# Final Simple Deployment Script
# Run this when network is stable

param(
    [string]$AppName = "codejudge-$(Get-Random -Minimum 10000 -Maximum 99999)"
)

$ResourceGroup = "codejudge-k8s-rg"
$Plan = "codejudge-plan"

Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host "  CodeJudge Azure Deployment" -ForegroundColor Cyan
Write-Host "========================================`n" -ForegroundColor Cyan

Write-Host "App Name: $AppName" -ForegroundColor Yellow
Write-Host "Resource Group: $ResourceGroup" -ForegroundColor Yellow
Write-Host "Location: Southeast Asia`n" -ForegroundColor Yellow

# Step 1: Create Web App
Write-Host "[1/3] Creating Web App..." -ForegroundColor Cyan
az webapp create `
  --resource-group $ResourceGroup `
  --plan $Plan `
  --name $AppName `
  --runtime "PYTHON:3.11"

if ($LASTEXITCODE -ne 0) {
    Write-Host "Failed. Check AZURE_DEPLOYMENT_MANUAL.md for manual steps." -ForegroundColor Red
    exit 1
}

# Step 2: Configure Settings
Write-Host "`n[2/3] Configuring settings..." -ForegroundColor Cyan
Add-Type -AssemblyName 'System.Web'
$JWTSecret = [System.Web.Security.Membership]::GeneratePassword(32, 8)

az webapp config appsettings set `
  --resource-group $ResourceGroup `
  --name $AppName `
  --settings `
    JWT_SECRET="$JWTSecret" `
    PORT=8080 `
    WEBSITES_PORT=8080 `
    GIN_MODE=release `
  --output none

# Step 3: Enable Logging
Write-Host "`n[3/3] Enabling logging..." -ForegroundColor Cyan
az webapp log config `
  --resource-group $ResourceGroup `
  --name $AppName `
  --application-logging filesystem `
  --level information `
  --output none

$AppURL = "https://$AppName.azurewebsites.net"

Write-Host "`n========================================" -ForegroundColor Green
Write-Host "  DEPLOYMENT SUCCESSFUL!" -ForegroundColor Green
Write-Host "========================================`n" -ForegroundColor Green

Write-Host "Your App URL:" -ForegroundColor Cyan
Write-Host "  $AppURL`n" -ForegroundColor White

Write-Host "Next Steps:" -ForegroundColor Yellow
Write-Host "  1. Deploy your code (see AZURE_DEPLOYMENT_MANUAL.md)" -ForegroundColor White
Write-Host "  2. View logs: az webapp log tail --resource-group $ResourceGroup --name $AppName" -ForegroundColor Gray
Write-Host "  3. Open app: az webapp browse --resource-group $ResourceGroup --name $AppName`n" -ForegroundColor Gray

# Save info
@"
Deployed: $(Get-Date)
App Name: $AppName
URL: $AppURL
Resource Group: $ResourceGroup
JWT Secret: $JWTSecret

View logs:
  az webapp log tail --resource-group $ResourceGroup --name $AppName

Deploy code:
  See AZURE_DEPLOYMENT_MANUAL.md for instructions

Delete app:
  az webapp delete --resource-group $ResourceGroup --name $AppName
"@ | Out-File -FilePath "azure-deployment-info.txt"

Write-Host "Info saved to: azure-deployment-info.txt" -ForegroundColor Cyan
Write-Host ""
