# build-go-service.ps1 - PowerShell script to build Go services using the generic Dockerfile

param(
    [Parameter(Position=0)]
    [string]$Command,
    
    [Parameter(Position=1)]
    [string]$ServiceName,
    
    [Parameter(Position=2)]
    [string]$Registry = "ghcr.io/kwant-dbg"
)

# Service configurations
$Services = @{
    "problems-service-go" = "8000"
    "submissions-service-go" = "8001"
    "plagiarism-service-go" = "8002"
    "api-gateway" = "8080"
}

function Build-Service {
    param(
        [string]$ServiceName
    )
    
    $ServicePort = $Services[$ServiceName]
    if (-not $ServicePort) {
        Write-Error "Unknown service '$ServiceName'"
        Write-Host "Available services: $($Services.Keys -join ', ')"
        exit 1
    }
    
    Write-Host "Building $ServiceName (port $ServicePort)..." -ForegroundColor Green
    
    # Build using the generic Dockerfile
    $buildArgs = @(
        "build"
        "--build-arg", "SERVICE_NAME=$ServiceName"
        "--build-arg", "SERVICE_PORT=$ServicePort"
        "-f", "Dockerfile.go-service"
        "-t", "codejudge/$ServiceName`:latest"
        "."
    )
    
    & docker @buildArgs
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Successfully built codejudge/$ServiceName`:latest" -ForegroundColor Green
    } else {
        Write-Error "Failed to build $ServiceName"
        exit 1
    }
}

function Build-AllServices {
    Write-Host "Building all Go services..." -ForegroundColor Cyan
    foreach ($service in $Services.Keys) {
        Build-Service -ServiceName $service
    }
    Write-Host "All services built successfully!" -ForegroundColor Green
}

function Push-Service {
    param(
        [string]$ServiceName,
        [string]$RegistryUrl
    )
    
    if (-not $Services.ContainsKey($ServiceName)) {
        Write-Error "Unknown service '$ServiceName'"
        exit 1
    }
    
    $tag = "$RegistryUrl/codejudge-$ServiceName`:latest"
    
    Write-Host "Tagging and pushing $ServiceName to $RegistryUrl..." -ForegroundColor Green
    
    & docker tag "codejudge/$ServiceName`:latest" $tag
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to tag $ServiceName"
        exit 1
    }
    
    & docker push $tag
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Successfully pushed $tag" -ForegroundColor Green
    } else {
        Write-Error "Failed to push $ServiceName"
        exit 1
    }
}

function Show-Usage {
    Write-Host @"
Usage: .\build-go-service.ps1 [COMMAND] [SERVICE_NAME] [OPTIONS]

Commands:
    build [SERVICE_NAME]    Build a specific service (or 'all' for all services)
    push [SERVICE_NAME]     Push a service to registry
    help                    Show this help message

Available services: $($Services.Keys -join ', ')

Examples:
    .\build-go-service.ps1 build problems-service-go      # Build problems service
    .\build-go-service.ps1 build all                      # Build all services
    .\build-go-service.ps1 push submissions-service-go    # Push submissions service to default registry
    .\build-go-service.ps1 push api-gateway ghcr.io/user  # Push to custom registry

"@
}

# Main script logic
switch ($Command.ToLower()) {
    "build" {
        if ($ServiceName -eq "all") {
            Build-AllServices
        } elseif ($ServiceName) {
            Build-Service -ServiceName $ServiceName
        } else {
            Write-Error "Please specify a service name or 'all'"
            Show-Usage
            exit 1
        }
    }
    "push" {
        if ($ServiceName) {
            Push-Service -ServiceName $ServiceName -RegistryUrl $Registry
        } else {
            Write-Error "Please specify a service name"
            Show-Usage
            exit 1
        }
    }
    { $_ -in @("help", "--help", "-h") } {
        Show-Usage
    }
    default {
        if ($Command) {
            Write-Error "Unknown command '$Command'"
        } else {
            Write-Error "Please specify a command"
        }
        Show-Usage
        exit 1
    }
}