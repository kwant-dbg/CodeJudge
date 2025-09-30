# Add Terraform and Azure CLI to PATH for this session
$env:PATH += ";C:\terraform"
$env:PATH += ";C:\Program Files\Microsoft SDKs\Azure\CLI2\wbin"

# Set Service Principal Credentials
$env:ARM_CLIENT_ID="YOUR_CLIENT_ID"
$env:ARM_CLIENT_SECRET="YOUR_CLIENT_SECRET"
$env:ARM_TENANT_ID="YOUR_TENANT_ID"
$env:ARM_SUBSCRIPTION_ID="YOUR_SUBSCRIPTION_ID"

# Disable Azure CLI authentication to force Service Principal usage
$env:ARM_USE_CLI="false"

Write-Host "Environment configured. Running terraform $args"

# Change to the terraform directory and run the command
Set-Location d:\dev\codejudge\terraform
terraform $args
