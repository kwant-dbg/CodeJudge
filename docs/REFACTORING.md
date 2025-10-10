# Mono Branch Refactoring Plan

## Current Issues
- Too many microservice-related files not needed for monolith
- Multiple docker-compose files causing confusion
- Documentation scattered
- Unnecessary service directories (api-gateway, auth-service-go, etc.)

## Proposed Structure

```
codejudge/
├── docs/                           # All documentation
│   ├── DEPLOYMENT.md              # Combined deployment guide
│   ├── AZURE.md                   # Azure deployment
│   ├── DEVELOPMENT.md             # Local development setup
│   └── ARCHITECTURE.md            # Monolith architecture
│
├── monolith/                      # Main application service
│   ├── handlers/                  # HTTP handlers
│   ├── static/                    # Frontend files
│   ├── Dockerfile                 # Production dockerfile
│   ├── go.mod
│   └── main.go
│
├── judge/                         # C++ judge service
│   ├── modern_main.cpp
│   ├── sandbox.cpp
│   ├── sandbox.h
│   ├── Dockerfile
│   └── CMakeLists.txt
│
├── common/                        # Shared Go libraries
│   ├── auth/
│   ├── dbutil/
│   ├── httpx/
│   └── go.mod
│
├── deploy/                        # Deployment scripts
│   ├── azure.ps1                  # Azure deployment
│   ├── local.ps1                  # Local deployment
│   └── seed-db.ps1                # Database seeding
│
├── docker-compose.yml             # Main compose file (renamed from monolith)
├── .env.example                   # Environment template
├── .gitignore
├── go.work                        # Go workspace
└── README.md                      # Updated readme

```

## Files to Remove (Not Needed for Monolith)

### Microservice Directories:
- [ ] api-gateway/
- [ ] auth-service-go/
- [ ] problems-service-go/
- [ ] submissions-service-go/
- [ ] plagiarism-service-go/

### Unnecessary Docker Files:
- [ ] docker-compose.prod.yml
- [ ] docker-compose.images.yml
- [ ] docker-compose.modern.yml
- [ ] Dockerfile.go-service

### Old Documentation:
- [ ] AUTH_SETUP.md (microservices)
- [ ] JUDGE_SETUP.md (outdated)
- [ ] DEPLOYMENT_SUMMARY.md (old)
- [ ] PROJECT_SUMMARY.md (old)

### Other:
- [ ] kubernetes/ (for microservices)
- [ ] maintenance-page/
- [ ] scripts/ (old build scripts)
- [ ] test.cpp
- [ ] .github/workflows/ (old workflows)

## Files to Reorganize

### Move to /docs:
- [ ] MONOLITH_DEPLOYMENT.md → docs/DEPLOYMENT.md
- [ ] AZURE_DEPLOYMENT.md → docs/AZURE.md
- [ ] AZURE_QUICKSTART.md → docs/AZURE_QUICKSTART.md
- [ ] AZURE_READY.md → docs/AZURE_READY.md
- [ ] THEME_UPDATE.md → docs/THEME_UPDATE.md

### Move to /deploy:
- [ ] deploy-monolith.ps1 → deploy/local.ps1
- [ ] deploy-azure.ps1 → deploy/azure.ps1
- [ ] scripts/seed_problems.* → deploy/seed-db.*

### Rename Services:
- [ ] monolith-service/ → monolith/
- [ ] judge-service/ → judge/
- [ ] common-go/ → common/

### Simplify Docker Compose:
- [ ] docker-compose.monolith.yml → docker-compose.yml
- [ ] Update all references

## Implementation Steps

1. Create new directory structure
2. Move files to new locations
3. Update import paths in Go code
4. Update docker-compose references
5. Update documentation
6. Remove old files
7. Test everything works
8. Update README with new structure

## Benefits

✅ Cleaner, more intuitive structure
✅ Easier to understand for new developers
✅ Focused on monolith architecture
✅ Better organized documentation
✅ Simpler deployment process
✅ Reduced confusion with unnecessary files

## Execution

Run the refactoring script:
```powershell
.\refactor-mono.ps1
```

Or manual steps documented below...
