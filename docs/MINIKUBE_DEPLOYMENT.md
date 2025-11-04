# CodeJudge Minikube Deployment Guide

## Overview

This document describes the complete Kubernetes deployment of CodeJudge using Minikube, designed for local development and demonstration purposes. The entire setup is configured to run on disk D to preserve SSD space on disk C.

## Architecture

### System Components

```
┌─────────────────────────────────────────────────────────────┐
│                     Minikube Cluster                        │
│  ┌───────────────────────────────────────────────────────┐  │
│  │            Namespace: codejudge                       │  │
│  │                                                       │  │
│  │  ┌──────────────┐      ┌──────────────┐            │  │
│  │  │  Monolith    │◄────►│  PostgreSQL  │            │  │
│  │  │  (Web App)   │      │  (Database)  │            │  │
│  │  │  Port: 8080  │      │  Port: 5432  │            │  │
│  │  └──────┬───────┘      └──────────────┘            │  │
│  │         │                                           │  │
│  │         │              ┌──────────────┐            │  │
│  │         └─────────────►│    Redis     │            │  │
│  │         │              │   (Cache)    │            │  │
│  │         │              │  Port: 6379  │            │  │
│  │         │              └──────────────┘            │  │
│  │         │                                           │  │
│  │         │              ┌──────────────┐            │  │
│  │         └─────────────►│    Judge     │            │  │
│  │                        │ (Executor)   │            │  │
│  │                        └──────────────┘            │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
              │
              ▼
    http://localhost:8080
    http://192.168.49.2:30080
```

### Technology Stack

- **Container Orchestration**: Kubernetes via Minikube
- **Container Runtime**: Docker Desktop
- **Application Components**:
  - **Monolith**: Go-based web application (Gin framework)
  - **Judge Service**: C++ code execution engine with sandboxing
  - **PostgreSQL 14**: Primary database
  - **Redis 7**: Caching and message queue

## Installation Process

### 1. Prerequisites Setup

**Disk Configuration:**
- All Minikube data stored on D: drive
- Location: `D:\minikube\.minikube`
- Minikube executable: `D:\minikube\minikube.exe`

**Requirements:**
- Docker Desktop (installed and running)
- Windows 10/11 with PowerShell
- At least 4GB RAM available for Minikube
- 10GB+ free disk space on D: drive

### 2. Minikube Installation

```powershell
# Create Minikube directory on D: drive
New-Item -ItemType Directory -Force -Path "D:\minikube"

# Download Minikube
Invoke-WebRequest -OutFile "D:\minikube\minikube.exe" `
  -Uri "https://github.com/kubernetes/minikube/releases/latest/download/minikube-windows-amd64.exe"

# Set environment variables
$env:MINIKUBE_HOME = "D:\minikube\.minikube"
$env:Path += ";D:\minikube"
```

### 3. Minikube Cluster Creation

```powershell
# Start Minikube with Docker driver
D:\minikube\minikube.exe start --driver=docker

# Verify cluster is running
D:\minikube\minikube.exe status
```

**Cluster Specifications:**
- CPUs: 2
- Memory: 4GB
- Driver: Docker
- Kubernetes Version: 1.34.0

## Kubernetes Deployment

### Directory Structure

```
k8s/
├── namespace.yaml           # Namespace definition
├── configmap.yaml          # Application configuration
├── secrets.yaml            # Sensitive data (credentials)
├── postgres.yaml           # PostgreSQL deployment + PVC
├── redis.yaml             # Redis deployment + PVC
├── monolith.yaml          # Main application deployment
└── judge.yaml             # Judge service deployment
```

### Resource Definitions

#### 1. Namespace (`namespace.yaml`)
- Creates isolated namespace: `codejudge`
- Separates resources from other applications

#### 2. ConfigMap (`configmap.yaml`)
- Non-sensitive configuration values
- Environment variables for all services
- Settings: JWT secret, timeouts, limits, etc.

#### 3. Secrets (`secrets.yaml`)
- Database credentials
- Connection strings
- Stored as Kubernetes secrets (base64 encoded)

#### 4. PostgreSQL (`postgres.yaml`)
```yaml
Components:
- PersistentVolumeClaim (5Gi)
- Deployment (1 replica)
- Service (ClusterIP on port 5432)
- Health checks (liveness & readiness probes)
```

**Features:**
- Persistent storage for database data
- Health monitoring via `pg_isready`
- Automatic restart on failure

#### 5. Redis (`redis.yaml`)
```yaml
Components:
- PersistentVolumeClaim (1Gi)
- Deployment (1 replica)
- Service (ClusterIP on port 6379)
- Health checks via redis-cli ping
```

#### 6. Monolith Application (`monolith.yaml`)
```yaml
Components:
- Deployment (1 replica)
- Service (NodePort 30080)
- Configuration via ConfigMap & Secrets
- HTTP health checks on /health endpoint
```

**Key Features:**
- Exposes web interface on port 8080
- NodePort service for external access
- Automatic database migration on startup
- Graceful shutdown handling

#### 7. Judge Service (`judge.yaml`)
```yaml
Components:
- PersistentVolumeClaim (2Gi for submissions)
- Deployment (1 replica)
- Security context with capabilities
```

**Security Features:**
- Limited capabilities (SYS_ADMIN, SYS_CHROOT, SETUID, SETGID)
- Seccomp profile enabled
- Privilege escalation disabled
- Sandboxed code execution environment

### Deployment Sequence

The components are deployed in this specific order to handle dependencies:

```powershell
# 1. Create namespace
D:\minikube\minikube.exe kubectl -- apply -f k8s/namespace.yaml

# 2. Create configuration
D:\minikube\minikube.exe kubectl -- apply -f k8s/configmap.yaml
D:\minikube\minikube.exe kubectl -- apply -f k8s/secrets.yaml

# 3. Deploy data services (must be ready first)
D:\minikube\minikube.exe kubectl -- apply -f k8s/postgres.yaml
D:\minikube\minikube.exe kubectl -- apply -f k8s/redis.yaml

# Wait for data services to be ready (~30-60 seconds)

# 4. Deploy application services
D:\minikube\minikube.exe kubectl -- apply -f k8s/monolith.yaml
D:\minikube\minikube.exe kubectl -- apply -f k8s/judge.yaml
```

## Docker Image Building

### Building Images in Minikube

To make images available to Minikube without pushing to a registry:

```powershell
# Configure Docker to use Minikube's Docker daemon
D:\minikube\minikube.exe docker-env | Invoke-Expression

# Build monolith image
docker build -t codejudge-monolith:latest -f monolith/Dockerfile.standalone .

# Build judge image
docker build -t codejudge-judge:latest -f judge/Dockerfile.modern judge/
```

**Benefits:**
- No external registry required
- Faster iteration during development
- Images stored within Minikube cluster
- Set `imagePullPolicy: Never` in deployment manifests

## Accessing the Application

### Method 1: NodePort (Direct Access)

```
URL: http://192.168.49.2:30080
```

- Uses Minikube's IP address
- Port 30080 is exposed on the Kubernetes node
- Works immediately after deployment
- Accessible from host machine

### Method 2: Port Forwarding (Localhost)

```powershell
# Start port forwarding
D:\minikube\minikube.exe kubectl -- port-forward -n codejudge svc/monolith-service 8080:8080

# Access via localhost
URL: http://localhost:8080
```

**Advantages:**
- Clean localhost URL
- Standard port (8080)
- Better for demos and testing

## Monitoring and Management

### Checking Pod Status

```powershell
# View all pods in codejudge namespace
D:\minikube\minikube.exe kubectl -- get pods -n codejudge

# Expected output:
NAME                        READY   STATUS    RESTARTS   AGE
judge-7ddc646ff5-76j82      1/1     Running   4          10m
monolith-5b8cbcd8d4-ngw5d   1/1     Running   3          10m
postgres-6754698b97-5g9x4   1/1     Running   0          10m
redis-65758c764c-kb2fg      1/1     Running   0          10m
```

### Viewing Logs

```powershell
# Monolith logs
D:\minikube\minikube.exe kubectl -- logs -n codejudge deployment/monolith --tail=50

# Judge logs
D:\minikube\minikube.exe kubectl -- logs -n codejudge deployment/judge --tail=50

# PostgreSQL logs
D:\minikube\minikube.exe kubectl -- logs -n codejudge deployment/postgres --tail=50

# Follow logs in real-time
D:\minikube\minikube.exe kubectl -- logs -n codejudge deployment/monolith -f
```

### Service Information

```powershell
# List all services
D:\minikube\minikube.exe kubectl -- get svc -n codejudge

# Describe specific service
D:\minikube\minikube.exe kubectl -- describe svc monolith-service -n codejudge
```

### Resource Usage

```powershell
# Check pod resource consumption
D:\minikube\minikube.exe kubectl -- top pods -n codejudge

# Check node resources
D:\minikube\minikube.exe kubectl -- top nodes
```

## Persistent Storage

### Volume Claims

Three persistent volumes are provisioned:

1. **PostgreSQL Data** (5Gi)
   - Path: `/var/lib/postgresql/data`
   - Contains all database tables and indexes
   - Survives pod restarts and deletions

2. **Redis Data** (1Gi)
   - Path: `/data`
   - Stores Redis persistence files (RDB/AOF)
   - Maintains cache across restarts

3. **Judge Submissions** (2Gi)
   - Path: `/app/submissions`
   - Stores user code submissions
   - Shared storage for code execution

### Listing Volumes

```powershell
# View persistent volume claims
D:\minikube\minikube.exe kubectl -- get pvc -n codejudge

# View persistent volumes
D:\minikube\minikube.exe kubectl -- get pv
```

## Troubleshooting

### Common Issues and Solutions

#### Issue 1: Pods in CrashLoopBackOff

**Symptoms:**
```
NAME                        READY   STATUS             RESTARTS
monolith-5b8cbcd8d4-ngw5d   0/1     CrashLoopBackOff   3
```

**Diagnosis:**
```powershell
D:\minikube\minikube.exe kubectl -- logs -n codejudge deployment/monolith
```

**Common Causes:**
- Database not ready (PostgreSQL still starting)
- Missing environment variables
- Connection string errors
- Image pull failures

**Solution:**
- Wait 1-2 minutes for dependencies to be ready
- Kubernetes will automatically retry
- Verify secrets and configmap are applied

#### Issue 2: Cannot Access Application

**Checks:**
1. Verify pods are running:
   ```powershell
   D:\minikube\minikube.exe kubectl -- get pods -n codejudge
   ```

2. Check service endpoints:
   ```powershell
   D:\minikube\minikube.exe kubectl -- get svc -n codejudge
   ```

3. Get Minikube IP:
   ```powershell
   D:\minikube\minikube.exe ip
   ```

4. Test connectivity:
   ```powershell
   curl http://192.168.49.2:30080/health
   ```

#### Issue 3: Docker Images Not Found

**Error Message:**
```
Failed to pull image "codejudge-monolith:latest": rpc error: code = Unknown desc = Error response from daemon: pull access denied
```

**Solution:**
```powershell
# Ensure using Minikube's Docker daemon
D:\minikube\minikube.exe docker-env | Invoke-Expression

# Rebuild images
docker build -t codejudge-monolith:latest -f monolith/Dockerfile.standalone .
docker build -t codejudge-judge:latest -f judge/Dockerfile.modern judge/

# Verify images exist
docker images | Select-String codejudge
```

#### Issue 4: Insufficient Resources

**Symptoms:**
- Pods stuck in Pending state
- Node pressure warnings

**Solution:**
```powershell
# Stop Minikube
D:\minikube\minikube.exe stop

# Start with more resources
D:\minikube\minikube.exe start --cpus=4 --memory=8192
```

### Health Check Endpoints

- **Monolith**: `http://localhost:8080/health`
- **PostgreSQL**: Inside cluster - `pg_isready`
- **Redis**: Inside cluster - `redis-cli ping`

## Maintenance Operations

### Stopping the Cluster

```powershell
# Graceful shutdown
D:\minikube\minikube.exe stop

# This preserves all data and configuration
```

### Starting the Cluster

```powershell
# Set environment variable
$env:MINIKUBE_HOME = "D:\minikube\.minikube"

# Start cluster
D:\minikube\minikube.exe start

# Verify all pods are running
D:\minikube\minikube.exe kubectl -- get pods -n codejudge
```

### Updating Deployments

```powershell
# After modifying a YAML file
D:\minikube\minikube.exe kubectl -- apply -f k8s/monolith.yaml

# Restart deployment
D:\minikube\minikube.exe kubectl -- rollout restart deployment/monolith -n codejudge

# Check rollout status
D:\minikube\minikube.exe kubectl -- rollout status deployment/monolith -n codejudge
```

### Scaling Services

```powershell
# Scale monolith to 3 replicas
D:\minikube\minikube.exe kubectl -- scale deployment/monolith --replicas=3 -n codejudge

# Verify scaling
D:\minikube\minikube.exe kubectl -- get pods -n codejudge
```

### Complete Cleanup

```powershell
# Delete all resources in namespace
D:\minikube\minikube.exe kubectl -- delete namespace codejudge

# Delete Minikube cluster
D:\minikube\minikube.exe delete

# Remove Minikube completely (optional)
Remove-Item -Recurse -Force D:\minikube
```

## Performance Considerations

### Resource Allocation

**Current Configuration:**
- Monolith: No limits (shares node resources)
- PostgreSQL: No limits (shares node resources)
- Redis: No limits (shares node resources)
- Judge: Limited capabilities for security

**Recommended Production Limits:**
```yaml
resources:
  requests:
    memory: "256Mi"
    cpu: "250m"
  limits:
    memory: "512Mi"
    cpu: "500m"
```

### Optimization Tips

1. **Database Connection Pooling**: Already implemented in the application
2. **Redis Caching**: Used for session management and leaderboards
3. **Persistent Volumes**: Use SSD-backed storage for better performance
4. **Horizontal Scaling**: Can scale monolith replicas for higher load

## Security Considerations

### Network Policies

Currently using default Kubernetes networking:
- All pods can communicate within the namespace
- Services are exposed via ClusterIP (internal only)
- Only monolith has NodePort (external access)

### Secret Management

- Secrets stored as Kubernetes Secret objects
- Base64 encoded (not encrypted at rest by default)
- For production, consider using sealed-secrets or external secret managers

### Judge Service Security

The judge service runs with restricted capabilities:
```yaml
securityContext:
  capabilities:
    add: [SYS_ADMIN, SYS_CHROOT, SETUID, SETGID]
    drop: [ALL]
  allowPrivilegeEscalation: false
```

**Sandboxing Features:**
- User code runs in isolated namespaces
- Chroot jails for filesystem isolation
- Resource limits (CPU, memory, time)
- Network isolation

## Advantages of This Setup

### For Development

1. **Local Testing**: Full production-like environment on local machine
2. **Fast Iteration**: Build and deploy without external dependencies
3. **Resource Isolation**: Each component in separate containers
4. **Easy Debugging**: Direct access to logs and metrics

### For Interviews/Demos

1. **Professional Setup**: Shows understanding of cloud-native architecture
2. **Scalability Demo**: Can demonstrate horizontal scaling
3. **Microservices Architecture**: Clear separation of concerns
4. **DevOps Skills**: Kubernetes, Docker, container orchestration
5. **Infrastructure as Code**: All configuration in version control

### Technical Benefits

1. **Declarative Configuration**: All infrastructure defined in YAML
2. **Self-Healing**: Kubernetes automatically restarts failed pods
3. **Service Discovery**: Built-in DNS for service-to-service communication
4. **Load Balancing**: Automatic traffic distribution across replicas
5. **Rolling Updates**: Zero-downtime deployments

## Git Branch Strategy

### Branch: `minikube-deploy`

Created from `mono` branch with Kubernetes additions:

```
mono (main branch)
 │
 └── minikube-deploy (Kubernetes deployment)
     └── k8s/ (all Kubernetes manifests)
```

### Committing Changes

```powershell
# Stage Kubernetes files
git add k8s/

# Commit with descriptive message
git commit -m "Add Kubernetes manifests for Minikube deployment

- Created namespace, configmap, and secrets
- Added PostgreSQL and Redis with persistent storage
- Deployed monolith application with NodePort service
- Configured judge service with security constraints
- All components tested and running successfully"

# Push to remote
git push -u origin minikube-deploy
```

## Comparison: Docker Compose vs Kubernetes

| Feature | Docker Compose | Kubernetes (Minikube) |
|---------|---------------|----------------------|
| **Orchestration** | Simple, single-host | Advanced, production-ready |
| **Scaling** | Manual | Automatic with replicas |
| **Self-Healing** | No | Yes (automatic restarts) |
| **Service Discovery** | DNS names | Kubernetes DNS + Services |
| **Load Balancing** | Basic | Built-in with Services |
| **Health Checks** | Basic | Liveness + Readiness probes |
| **Storage** | Docker volumes | PersistentVolumeClaims |
| **Networking** | Bridge network | CNI with network policies |
| **Configuration** | Environment vars | ConfigMaps + Secrets |
| **Production Ready** | No | Yes (same as cloud) |

## Next Steps

### For Production Deployment

1. **Cloud Provider**: Deploy to AKS (Azure), EKS (AWS), or GKE (Google Cloud)
2. **Ingress Controller**: Add nginx-ingress or traefik for HTTP routing
3. **TLS Certificates**: Configure cert-manager for HTTPS
4. **Monitoring**: Add Prometheus + Grafana
5. **Logging**: Implement ELK stack or Loki
6. **CI/CD**: GitHub Actions for automated deployments
7. **Database**: Use managed database service (Azure Database for PostgreSQL)
8. **Redis**: Use managed Redis (Azure Cache for Redis)
9. **Secrets**: External secret manager (Azure Key Vault, AWS Secrets Manager)
10. **Backup**: Implement Velero for cluster backups

### Additional Features to Demonstrate

1. **Helm Charts**: Package application as Helm chart
2. **GitOps**: Implement ArgoCD or Flux
3. **Service Mesh**: Add Istio or Linkerd
4. **Auto-scaling**: Configure HPA (Horizontal Pod Autoscaler)
5. **Network Policies**: Implement pod-to-pod communication rules

## Interview Talking Points

When explaining this to an interviewer, emphasize:

1. **Problem Solving**: Configured everything on D: drive to preserve SSD space
2. **Architecture Design**: Microservices with clear separation of concerns
3. **Cloud-Native**: Using industry-standard tools (Kubernetes, Docker)
4. **Security**: Implemented sandboxing for code execution with restricted capabilities
5. **Scalability**: Can easily scale components independently
6. **Observability**: Health checks, logging, and monitoring capabilities
7. **DevOps**: Infrastructure as Code, reproducible deployments
8. **Production-Ready**: Same patterns used in enterprise environments
9. **Documentation**: Comprehensive documentation for knowledge transfer
10. **Best Practices**: Follows Kubernetes conventions and security standards

## Conclusion

This Minikube deployment demonstrates a production-ready architecture for the CodeJudge application. It showcases modern DevOps practices, containerization, orchestration, and cloud-native application design. The setup is portable, scalable, and follows industry best practices, making it an excellent foundation for both development and production deployments.

---

**Author**: CodeJudge Team  
**Date**: November 4, 2025  
**Version**: 1.0  
**Branch**: `minikube-deploy`
