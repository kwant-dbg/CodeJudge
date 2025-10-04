
# CodeJudge: High-Performance Online Judging System

CodeJudge is a cloud-native backend for powering competitive programming and automated code evaluation platforms. It is built on a distributed microservices architecture using Go and C++ for high throughput, scalability, and security.

<img width="1024" height="1024" alt="29, 2025 - 12_10PM" src="https://github.com/user-attachments/assets/9ab15fcd-070d-46b2-84ae-07ee72f3b07a" />

## Key Features

- **Secure C++ Sandbox:** Leverages `fork`, `exec`, and `setrlimit` for low-level process isolation and resource management, preventing malicious code execution.
- **Horizontally Scalable:** Go and C++ services are decoupled via a Redis message bus, allowing for independent scaling of workers and other components.
- **Algorithmic Plagiarism Detection:** Implements a shingling and winnowing pipeline to detect structural code similarity, moving beyond simple token matching.
- **Dockerized & Cloud-Native:** Fully containerized with `docker-compose.yml` for local development, plus Kubernetes manifests for cluster deployment.

## Microservices Overview

| Service                  | Language | Path                   | Role                                 |
|--------------------------|----------|------------------------|--------------------------------------|
| **API Gateway**          | Go       | `api-gateway/`         | HTTP API entrypoint, routing, static UI |
| **Problems Service**     | Go       | `problems-service-go/` | Problem CRUD, metadata, DB access    |
| **Submissions Service**  | Go       | `submissions-service-go/` | Handles code submissions, job queue |
| **Plagiarism Service**   | Go       | `plagiarism-service-go/` | Plagiarism detection pipeline      |
| **Judge Service**        | C++      | `judge-service/`       | Secure code execution, sandboxing    |
| **Database**             | Postgres | Docker/K8s             | Primary datastore                    |
| **Message Queue**        | Redis    | Docker/K8s             | Job queue & message bus              |

## Technology Stack

| Component         | Technology                | Role                                 |
|-------------------|--------------------------|--------------------------------------|
| Backend Services  | Go (net/http, go-redis)  | API, orchestration, business logic   |
| Judge Engine      | C++17                    | Sandboxed code execution             |
| Database          | PostgreSQL               | Persistent storage                   |
| Message Queue     | Redis                    | Job queue, pub/sub                   |
| Containerization  | Docker, Docker Compose   | Local deployment                     |
| Infrastructure    | Kubernetes               | Cluster deployment & provisioning    |

## Docker Images

### GitHub Container Registry
Pre-built Docker images are available at GitHub Container Registry:

```bash
# Pull latest images
docker pull ghcr.io/kwant-dbg/codejudge/codejudge-api-gateway:latest
docker pull ghcr.io/kwant-dbg/codejudge/codejudge-problems-service-go:latest
docker pull ghcr.io/kwant-dbg/codejudge/codejudge-submissions-service-go:latest
docker pull ghcr.io/kwant-dbg/codejudge/codejudge-plagiarism-service-go:latest
docker pull ghcr.io/kwant-dbg/codejudge/codejudge-judge-service:latest
```

### Building Images Locally
```bash
# Build all services
docker-compose build

# Build specific service
docker-compose build gateway
```

### Using Pre-built Images
```bash
# Run using published images (faster)
docker-compose -f docker-compose.images.yml up -d

# Run with local builds
docker-compose up --build -d
```

### Publishing Images
```bash
# Manual publish to GitHub Container Registry
./scripts/build-and-push-ghcr.sh

# Windows PowerShell
.\scripts\build-and-push-ghcr.ps1
```

Images are automatically built and published via GitHub Actions on every push to main/master.

## Quick Start

**Prerequisites:** Docker and Docker Compose

1. **Clone & Enter Project:**
   ```bash
   git clone https://github.com/kwant-dbg/CodeJudge.git
   cd CodeJudge
   ```

2. **Launch All Services:**
   ```bash
   # Development
   docker-compose up --build -d
   
   # Production-ready with health checks
   docker-compose -f docker-compose.prod.yml up --build -d
   ```
   The API Gateway is exposed at [http://localhost:8080](http://localhost:8080).

3. **Check Deployment Readiness (Optional):**
   ```bash
   # Run pre-deployment checks
   bash scripts/check-deployment-readiness.sh
   ```

## Maintenance page (GitHub Pages)

If you need to temporarily take the site offline to save resources, this repository includes a small static maintenance page and a lightweight deploy helper.

- Files: `maintenance-page/index.html`, `maintenance-page/styles.css`, `maintenance-page/CNAME`.
- How to update: edit files under `maintenance-page/` and push to `main`/`master`. A small workflow (`.github/workflows/deploy-maintenance-on-change.yml`) will run only when `maintenance-page/**` changes and will publish the content to the `gh-pages` branch.
- Manual publish: alternatively, commit directly to the `gh-pages` branch (it contains only the published page) if you prefer manual control.
- DNS: the repository contains a `CNAME` set to `codejudge.live`. Configure DNS at Name.com (four GitHub Pages A records for the apex and a CNAME for `www`) to point the domain to GitHub Pages.

Note: The main CI will not run for simple edits under `maintenance-page/` because the deploy helper is narrowly scoped to those path changes.

## Advanced Deployment

### Azure Deployment Ready

This project is optimized for efficient cloud deployment with:

   Resource-efficient Docker builds using Go workspaces
   Standardized health endpoints for container orchestration
   Production docker-compose with health checks and resource limits
   Shared utilities to minimize deployment complexity

For production Kubernetes deployment, you'll need to create manifests for all services and configure:

   ConfigMaps for environment variables
   Secrets for database and Redis credentials
   Services for internal communication
   Ingress for external access Apply them to any compatible cluster:

Independent Service Testing

   All Go services can be run independently with their own go.mod
   The C++ judge service can be built and tested in isolation
   Shared utilities in common-go/ provide consistent behavior

**Azure Prerequisites:**

   Azure Container Registry
   PostgreSQL Flexible Server 
   Redis Cache
   App Service Plan or Container Instances

See `docs/azure-deployment.md` for resource planning and `.env.example` for configuration.

### Kubernetes

## Kubernetes Deployment

Sample Kubernetes manifests for some services are in `kubernetes/deploy/`:
- `plagiarism-service.yaml` - Plagiarism detection worker
- `submissions-service.yaml` - Submission handling service

For production Kubernetes deployment, you'll need to create manifests for all services and configure:
- ConfigMaps for environment variables
- Secrets for database and Redis credentials
- Services for internal communication
- Ingress for external access
Apply them to any compatible cluster:

```bash
kubectl apply -f kubernetes/deploy/
```

## Local Development & Testing

### Go Workspace Development
This project uses Go workspaces for efficient multi-module development:

```bash
# Sync workspace
go work sync

# Run tests across all modules
cd common-go && go test ./...

# Build specific service
cd problems-service-go && go build
```

### Independent Service Testing
- All Go services can be run independently with their own `go.mod`
- The C++ judge service can be built and tested in isolation
- Shared utilities in `common-go/` provide consistent behavior

### Integration Testing
```bash
# Full stack with development settings
docker-compose up --build

# Production-like with health checks
docker-compose -f docker-compose.prod.yml up --build

# Check deployment readiness
bash scripts/check-deployment-readiness.sh
```

### Maven Tasks (if present)
```powershell
mvn -B verify
mvn -B test
```

## API Endpoints

| Method | Path | Description |
| :--- | :--- | :--- |
| `GET` | `/api/problems/` | Retrieve all problems. |
| `POST`| `/api/problems/` | Create a new problem. |
| `POST`| `/api/submissions`| Enqueue a new code submission for judging. |
| `GET` | `/api/plagiarism/reports` | Get plagiarism analysis reports. |


**Example Submission:** `POST /api/submissions`
```json
{
  "problem_id": 1,
  "language": "cpp",
  "source_code": "#include <iostream>\nint main() { int a, b; std::cin >> a >> b; std::cout << a + b << std::endl; return 0; }"
}
```
---
© 2025 Harshit Sharma
