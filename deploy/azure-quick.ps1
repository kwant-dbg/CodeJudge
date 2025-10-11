# CodeJudge Simple Azure Deployment
# Using App Service without ACR (Container Registry)

param(
    [string]$ResourceGroup = "codejudge-app-$(Get-Random -Minimum 100 -Maximum 999)",
    [string]$AppName = "codejudge-$(Get-Random -Minimum 1000 -Maximum 9999)",
    [string]$DBPassword = "",
    [string]$JWTSecret = ""
)

Write-Host "`n==================================" -ForegroundColor Cyan
Write-Host "CodeJudge Azure Deployment" -ForegroundColor Cyan
Write-Host "==================================" -ForegroundColor Cyan

# Generate secrets
if ([string]::IsNullOrEmpty($DBPassword)) {
    Add-Type -AssemblyName 'System.Web'
    $DBPassword = "CJ" + [System.Web.Security.Membership]::GeneratePassword(14, 3)
    $DBPassword = $DBPassword -replace '[\\|;]', 'A'
    Write-Host "`nGenerated DB Password: $DBPassword" -ForegroundColor Yellow
}

if ([string]::IsNullOrEmpty($JWTSecret)) {
    $rng = [System.Security.Cryptography.RNGCryptoServiceProvider]::new()
    $bytes = New-Object byte[] 32
    $rng.GetBytes($bytes)
    $JWTSecret = [Convert]::ToBase64String($bytes)
    Write-Host "Generated JWT Secret: $JWTSecret" -ForegroundColor Yellow
}

# Check Azure login
Write-Host "`n[1/5] Checking Azure login..." -ForegroundColor Cyan
try {
    $account = az account show 2>$null | ConvertFrom-Json
    Write-Host "      Logged in as: $($account.user.name)" -ForegroundColor Green
} catch {
    Write-Host "      ERROR: Not logged in. Run 'az login' first." -ForegroundColor Red
    exit 1
}

# Try multiple regions until one works
$regions = @("westus2", "eastus2", "southcentralus", "westeurope", "uksouth")
$Location = $null

Write-Host "`n[2/5] Finding available region..." -ForegroundColor Cyan
foreach ($region in $regions) {
    Write-Host "      Trying $region..." -ForegroundColor Yellow
    az group create --name $ResourceGroup --location $region --output none 2>$null
    if ($LASTEXITCODE -eq 0) {
        $Location = $region
        Write-Host "      Using region: $Location" -ForegroundColor Green
        break
    }
}

if ($null -eq $Location) {
    Write-Host "      ERROR: No available regions found" -ForegroundColor Red
    exit 1
}

# Create App Service Plan (Free tier)
Write-Host "`n[3/5] Creating App Service Plan..." -ForegroundColor Cyan
$PlanName = "$AppName-plan"
az appservice plan create `
  --name $PlanName `
  --resource-group $ResourceGroup `
  --location $Location `
  --is-linux `
  --sku F1 `
  --output none

if ($LASTEXITCODE -ne 0) {
    Write-Host "      ERROR: Failed to create App Service Plan" -ForegroundColor Red
    exit 1
}
Write-Host "      App Service Plan created" -ForegroundColor Green

# Create Web App
Write-Host "`n[4/5] Creating Web App..." -ForegroundColor Cyan
az webapp create `
  --resource-group $ResourceGroup `
  --plan $PlanName `
  --name $AppName `
  --runtime "GO:1.19" `
  --output none

if ($LASTEXITCODE -ne 0) {
    Write-Host "      ERROR: Failed to create Web App" -ForegroundColor Red
    exit 1
}
Write-Host "      Web App created" -ForegroundColor Green

# Configure settings (SQLite fallback, no external DB needed for testing)
Write-Host "`n[5/5] Configuring Web App..." -ForegroundColor Cyan
az webapp config appsettings set `
  --resource-group $ResourceGroup `
  --name $AppName `
  --settings `
    JWT_SECRET="$JWTSecret" `
    PORT=8080 `
    WEBSITES_PORT=8080 `
    GIN_MODE=release `
  --output none

Write-Host "      Configuration complete" -ForegroundColor Green

# Get deployment credentials
Write-Host "`nSetting up deployment..." -ForegroundColor Cyan
$publishProfile = az webapp deployment list-publishing-profiles --resource-group $ResourceGroup --name $AppName --query "[0].{userName:userName, userPWD:userPWD}" -o json | ConvertFrom-Json

$appUrl = "https://$AppName.azurewebsites.net"

Write-Host "`n==================================" -ForegroundColor Green
Write-Host "   DEPLOYMENT SUCCESSFUL!" -ForegroundColor Green
Write-Host "==================================" -ForegroundColor Green

Write-Host "`nYour App URL:" -ForegroundColor Cyan
Write-Host "  $appUrl" -ForegroundColor White

Write-Host "`nResource Group:" -ForegroundColor Cyan
Write-Host "  $ResourceGroup" -ForegroundColor White

Write-Host "`nNext Steps:" -ForegroundColor Yellow
Write-Host "  1. Deploy your code using Git:" -ForegroundColor White
Write-Host "     cd d:\dev\codejudge\monolith" -ForegroundColor Gray
Write-Host "     git init" -ForegroundColor Gray
Write-Host "     git add ." -ForegroundColor Gray
Write-Host "     git commit -m 'Initial commit'" -ForegroundColor Gray
Write-Host "     az webapp deployment source config-local-git --resource-group $ResourceGroup --name $AppName" -ForegroundColor Gray
Write-Host ""
Write-Host "  2. View logs:" -ForegroundColor White
Write-Host "     az webapp log tail --resource-group $ResourceGroup --name $AppName" -ForegroundColor Gray
Write-Host ""
Write-Host "  3. Delete everything when done:" -ForegroundColor White
Write-Host "     az group delete --name $ResourceGroup --yes --no-wait" -ForegroundColor Gray

# Save config
$configFile = "azure-config.txt"
@"
CodeJudge Azure Deployment
==========================
Date: $(Get-Date)

Resource Group: $ResourceGroup
App Name: $AppName
Location: $Location
URL: $appUrl

JWT Secret: $JWTSecret

Commands:
  View logs: az webapp log tail --resource-group $ResourceGroup --name $AppName
  Restart: az webapp restart --resource-group $ResourceGroup --name $AppName
  Delete: az group delete --name $ResourceGroup --yes --no-wait
"@ | Out-File -FilePath $configFile

Write-Host "`nConfig saved to: $configFile" -ForegroundColor Cyan
Write-Host ""
