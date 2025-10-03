# Script to build and push Docker images to GitHub Container Registry
# Usage: .\scripts\build-and-push-ghcr.ps1

$ErrorActionPreference = "Stop"

# Configuration
$REPO_OWNER = "kwant-dbg"
$REPO_NAME = "CodeJudge"
$REGISTRY = "ghcr.io"
$IMAGE_PREFIX = "$REGISTRY/$($REPO_OWNER.ToLower())/$($REPO_NAME.ToLower())"

# Get current git commit
$COMMIT_SHA = (git rev-parse --short HEAD).Trim()
$BRANCH = (git rev-parse --abbrev-ref HEAD).Trim()

Write-Host "🏗️ Building and pushing CodeJudge Docker images..." -ForegroundColor Green
Write-Host "Repository: $REPO_OWNER/$REPO_NAME"
Write-Host "Commit: $COMMIT_SHA"
Write-Host "Branch: $BRANCH"
Write-Host ""

# Services configuration
$SERVICES = @{
    "gateway" = @("api-gateway/Dockerfile", ".")
    "problems" = @("problems-service-go/Dockerfile", ".")
    "submissions" = @("submissions-service-go/Dockerfile", ".")
    "plagiarism" = @("plagiarism-service-go/Dockerfile", ".")
    "judge" = @("judge-service/Dockerfile", "judge-service")
}

# Build and push each service
foreach ($service in $SERVICES.Keys) {
    $dockerfile = $SERVICES[$service][0]
    $context = $SERVICES[$service][1]
    
    $IMAGE_NAME = "$IMAGE_PREFIX/$service"
    
    Write-Host "📦 Building $service..." -ForegroundColor Yellow
    Write-Host "  Dockerfile: $dockerfile"
    Write-Host "  Context: $context"
    
    # Build with multiple tags
    docker build `
        -t "$IMAGE_NAME`:$COMMIT_SHA" `
        -t "$IMAGE_NAME`:$BRANCH" `
        -t "$IMAGE_NAME`:latest" `
        -f $dockerfile `
        $context
    
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to build $service"
    }
    
    Write-Host "🚀 Pushing $service..." -ForegroundColor Cyan
    docker push "$IMAGE_NAME`:$COMMIT_SHA"
    docker push "$IMAGE_NAME`:$BRANCH" 
    docker push "$IMAGE_NAME`:latest"
    
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to push $service"
    }
    
    Write-Host "✅ $service pushed successfully" -ForegroundColor Green
    Write-Host ""
}

Write-Host "🎉 All images built and pushed successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "Images available at:" -ForegroundColor Cyan
foreach ($service in $SERVICES.Keys) {
    Write-Host "  $IMAGE_PREFIX/$service`:latest"
}