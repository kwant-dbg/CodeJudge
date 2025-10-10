# CodeJudge Monolith Deployment Script for Windows
# Usage: .\deploy-monolith.ps1 [start|stop|restart|status|logs|clean|health]

param(
    [Parameter(Position=0)]
    [ValidateSet('start', 'stop', 'restart', 'status', 'logs', 'clean', 'health')]
    [string]$Action = 'start'
)

$ComposeFile = "docker-compose.monolith.yml"

function Start-Monolith {
    Write-Host "Starting CodeJudge Monolith Service..." -ForegroundColor Green
    docker-compose -f $ComposeFile up -d --build
    Write-Host "`nMonolith service started successfully!" -ForegroundColor Green
    Write-Host "Access the service at: http://localhost:8080" -ForegroundColor Cyan
}

function Stop-Monolith {
    Write-Host "Stopping CodeJudge Monolith Service..." -ForegroundColor Yellow
    docker-compose -f $ComposeFile down
    Write-Host "Monolith service stopped successfully!" -ForegroundColor Green
}

function Restart-Monolith {
    Write-Host "Restarting CodeJudge Monolith Service..." -ForegroundColor Yellow
    Stop-Monolith
    Start-Sleep -Seconds 2
    Start-Monolith
}

function Get-Status {
    Write-Host "Checking service status..." -ForegroundColor Cyan
    docker-compose -f $ComposeFile ps
}

function Get-Logs {
    Write-Host "Fetching logs (Ctrl+C to exit)..." -ForegroundColor Cyan
    docker-compose -f $ComposeFile logs -f
}

function Clean-Monolith {
    Write-Host "Cleaning up (removing volumes and containers)..." -ForegroundColor Red
    $confirmation = Read-Host "This will delete all data. Are you sure? (yes/no)"
    if ($confirmation -eq 'yes') {
        docker-compose -f $ComposeFile down -v
        Write-Host "Cleanup completed!" -ForegroundColor Green
    } else {
        Write-Host "Cleanup cancelled." -ForegroundColor Yellow
    }
}

function Test-Health {
    Write-Host "Testing health endpoint..." -ForegroundColor Cyan
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:8080/health" -UseBasicParsing
        if ($response.StatusCode -eq 200) {
            Write-Host "Health check passed!" -ForegroundColor Green
            Write-Host "Status Code: $($response.StatusCode)" -ForegroundColor Cyan
        } else {
            Write-Host "Health check failed!" -ForegroundColor Red
            Write-Host "Status Code: $($response.StatusCode)" -ForegroundColor Red
        }
    } catch {
        Write-Host "Health check failed - Service may not be running" -ForegroundColor Red
        Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
    }
}

# Main script logic
switch ($Action) {
    'start' { Start-Monolith }
    'stop' { Stop-Monolith }
    'restart' { Restart-Monolith }
    'status' { Get-Status }
    'logs' { Get-Logs }
    'clean' { Clean-Monolith }
    'health' { Test-Health }
}
