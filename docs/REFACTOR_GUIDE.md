# Mono Branch Refactoring Guide

## 🎯 Goal
Clean up the codebase by removing microservice files and organizing for monolith architecture.

## 📋 Manual Refactoring Steps

### Step 1: Rename Service Directories
```powershell
# Rename services to simpler names
mv monolith-service monolith
mv judge-service judge
mv common-go common
```

### Step 2: Create Documentation Directory
```powershell
mkdir docs
mv MONOLITH_DEPLOYMENT.md docs/DEPLOYMENT.md
mv AZURE_DEPLOYMENT.md docs/AZURE.md
mv AZURE_QUICKSTART.md docs/AZURE_QUICKSTART.md
mv AZURE_READY.md docs/AZURE_READY.md
mv THEME_UPDATE.md docs/THEME_UPDATE.md
```

### Step 3: Create Deploy Directory
```powershell
mkdir deploy
mv deploy-monolith.ps1 deploy/local.ps1
mv deploy-azure.ps1 deploy/azure.ps1
mv scripts/seed_problems.* deploy/
```

### Step 4: Simplify Docker Compose
```powershell
# Backup old microservices compose
mv docker-compose.yml docker-compose.microservices.yml.bak

# Rename monolith compose to main
mv docker-compose.monolith.yml docker-compose.yml
```

### Step 5: Remove Microservice Files
```powershell
# Remove microservice directories
rm -r api-gateway
rm -r auth-service-go
rm -r problems-service-go
rm -r submissions-service-go
rm -r plagiarism-service-go

# Remove old docker compose files
rm docker-compose.prod.yml
rm docker-compose.images.yml
rm docker-compose.modern.yml

# Remove old documentation
rm AUTH_SETUP.md
rm JUDGE_SETUP.md
rm DEPLOYMENT_SUMMARY.md
rm PROJECT_SUMMARY.md

# Remove other unused files
rm -r kubernetes
rm -r maintenance-page
rm -r scripts
rm test.cpp
rm Dockerfile.go-service
```

### Step 6: Update go.work
Edit `go.work`:
```go
go 1.19

use (
    ./common
    ./monolith
)
```

### Step 7: Update docker-compose.yml
Replace all instances:
- `monolith-service/` → `monolith/`
- `judge-service/` → `judge/`
- `common-go/` → `common/`

### Step 8: Update Go Import Paths
In `monolith/main.go` and `monolith/handlers/*.go`:
- Replace `codejudge/common-go/` → `codejudge/common/`

### Step 9: Test
```powershell
# Rebuild with new structure
docker-compose up -d --build

# Verify it works
curl http://localhost:8080/health
```

## 📁 Final Structure

```
codejudge/
├── docs/                      # All documentation
│   ├── DEPLOYMENT.md
│   ├── AZURE.md
│   ├── AZURE_QUICKSTART.md
│   ├── AZURE_READY.md
│   └── THEME_UPDATE.md
│
├── monolith/                  # Main service
│   ├── handlers/
│   ├── static/
│   ├── Dockerfile.standalone
│   ├── go.mod
│   └── main.go
│
├── judge/                     # Judge service
│   ├── modern_main.cpp
│   ├── sandbox.cpp
│   ├── Dockerfile.modern
│   └── CMakeLists.txt
│
├── common/                    # Shared libraries
│   ├── auth/
│   ├── dbutil/
│   ├── httpx/
│   └── go.mod
│
├── deploy/                    # Deployment
│   ├── local.ps1
│   ├── azure.ps1
│   └── seed-db.*
│
├── .github/
├── .vscode/
├── docker-compose.yml         # Main compose file
├── .env.example
├── .gitignore
├── go.work
└── README.md
```

## ✅ What Gets Removed

### Directories (Not Needed):
- api-gateway/ (monolith handles routing)
- auth-service-go/ (auth built into monolith)
- problems-service-go/ (problems built into monolith)
- submissions-service-go/ (submissions built into monolith)  
- plagiarism-service-go/ (plagiarism built into monolith)
- kubernetes/ (microservices deployment)
- maintenance-page/ (not essential)
- scripts/ (old build scripts)

### Files (Not Needed):
- docker-compose.prod.yml (microservices)
- docker-compose.images.yml (microservices)
- docker-compose.modern.yml (not used)
- Dockerfile.go-service (generic template)
- AUTH_SETUP.md (microservices doc)
- JUDGE_SETUP.md (outdated)
- DEPLOYMENT_SUMMARY.md (old)
- PROJECT_SUMMARY.md (old)
- test.cpp (test file)

## 🎯 Benefits

✅ **Cleaner structure** - Obvious what's needed
✅ **Less confusion** - No microservice files mixing with monolith
✅ **Easier onboarding** - New devs understand structure quickly
✅ **Simpler deployment** - One docker-compose.yml
✅ **Better organization** - Docs and deploy scripts in dedicated folders
✅ **Reduced clutter** - ~50% fewer files

## ⚠️ Before You Start

1. **Commit current work**:
   ```powershell
   git add .
   git commit -m "checkpoint before refactoring"
   ```

2. **Or create a backup branch**:
   ```powershell
   git checkout -b mono-backup
   git checkout mono
   ```

## 🚀 Quick Refactor (Copy-Paste)

```powershell
# Stop services
docker-compose -f docker-compose.monolith.yml down

# Create new structure
mkdir docs, deploy

# Move docs
mv MONOLITH_DEPLOYMENT.md docs/DEPLOYMENT.md
mv AZURE_*.md docs/
mv THEME_UPDATE.md docs/

# Move deploy scripts
mv deploy-monolith.ps1 deploy/local.ps1
mv deploy-azure.ps1 deploy/azure.ps1
if (Test-Path scripts) { mv scripts/seed_problems.* deploy/ }

# Rename services
mv monolith-service monolith
mv judge-service judge
mv common-go common

# Update docker-compose
mv docker-compose.yml docker-compose.microservices.yml.bak
mv docker-compose.monolith.yml docker-compose.yml

# Remove old files
rm -r api-gateway, auth-service-go, problems-service-go, submissions-service-go, plagiarism-service-go -ErrorAction SilentlyContinue
rm docker-compose.prod.yml, docker-compose.images.yml, docker-compose.modern.yml, Dockerfile.go-service -ErrorAction SilentlyContinue
rm AUTH_SETUP.md, JUDGE_SETUP.md, DEPLOYMENT_SUMMARY.md, PROJECT_SUMMARY.md, test.cpp -ErrorAction SilentlyContinue
rm -r kubernetes, maintenance-page, scripts -ErrorAction SilentlyContinue

# Update go.work
@"
go 1.19

use (
    ./common
    ./monolith
)
"@ | Set-Content go.work

# Update paths in docker-compose.yml
(Get-Content docker-compose.yml) -replace 'monolith-service/', 'monolith/' -replace 'judge-service/', 'judge/' -replace 'common-go/', 'common/' | Set-Content docker-compose.yml

# Update Go imports
(Get-Content monolith/main.go) -replace 'common-go/', 'common/' | Set-Content monolith/main.go
Get-ChildItem monolith/handlers/*.go | ForEach-Object { (Get-Content $_) -replace 'common-go/', 'common/' | Set-Content $_ }

# Test
docker-compose up -d --build

Write-Host "✅ Refactoring complete! Check http://localhost:8080" -ForegroundColor Green
```

## 📝 Commit

```powershell
git add .
git commit -m "refactor: reorganize mono branch - cleaner structure"
git push origin mono
```

---

**Ready to refactor?** Copy the "Quick Refactor" commands above and run them!
