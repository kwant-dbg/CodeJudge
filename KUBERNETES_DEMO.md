# 🚀 CodeJudge Kubernetes Deployment Demo

## Executive Summary

This project demonstrates a **production-ready microservices architecture** deployed on **Kubernetes** using **Minikube**. The deployment showcases modern DevOps practices, container orchestration, and cloud-native application design.

---

## 📊 Architecture Overview

```
┌─────────────────────────────────────────────────────┐
│           Kubernetes Cluster (Minikube)             │
│                                                     │
│  ┌──────────────┐    ┌──────────────┐            │
│  │   Monolith   │───►│  PostgreSQL  │            │
│  │  (Go/Gin)    │    │   Database   │            │
│  └──────┬───────┘    └──────────────┘            │
│         │                                          │
│         ├──────────►┌──────────────┐            │
│         │           │     Redis    │            │
│         │           │   (Cache)    │            │
│         │           └──────────────┘            │
│         │                                          │
│         └──────────►┌──────────────┐            │
│                     │  Judge       │            │
│                     │  (C++ Exec)  │            │
│                     └──────────────┘            │
└─────────────────────────────────────────────────────┘
              ▼
    http://localhost:8080
```

---

## 🎯 Key Features Demonstrated

### 1. **Cloud-Native Architecture**
- ✅ Containerized microservices
- ✅ Service discovery via Kubernetes DNS
- ✅ Declarative infrastructure (Infrastructure as Code)
- ✅ Horizontal scalability

### 2. **DevOps Best Practices**
- ✅ Container orchestration with Kubernetes
- ✅ Health monitoring (liveness & readiness probes)
- ✅ Persistent storage management
- ✅ Configuration management (ConfigMaps & Secrets)

### 3. **Security**
- ✅ Sandboxed code execution environment
- ✅ Restricted Linux capabilities
- ✅ Namespace isolation
- ✅ Secret management

### 4. **Observability**
- ✅ Centralized logging
- ✅ Health check endpoints
- ✅ Resource monitoring
- ✅ Event tracking

---

## 🛠️ Technology Stack

| Component | Technology | Purpose |
|-----------|-----------|---------|
| **Orchestration** | Kubernetes (Minikube) | Container management |
| **Container Runtime** | Docker | Application containerization |
| **Backend** | Go 1.23 + Gin | Web application framework |
| **Code Execution** | C++ | Sandboxed code judge |
| **Database** | PostgreSQL 14 | Data persistence |
| **Cache/Queue** | Redis 7 | Caching & message queue |
| **IaC** | YAML Manifests | Infrastructure definition |

---

## 📁 Project Structure

```
codejudge/
├── k8s/                          # Kubernetes manifests
│   ├── namespace.yaml           # Namespace isolation
│   ├── configmap.yaml           # Application config
│   ├── secrets.yaml             # Sensitive data
│   ├── postgres.yaml            # Database + PVC
│   ├── redis.yaml              # Cache + PVC
│   ├── monolith.yaml           # Main app + NodePort
│   └── judge.yaml              # Code executor + PVC
├── docs/
│   ├── MINIKUBE_DEPLOYMENT.md      # Detailed deployment guide
│   └── MINIKUBE_QUICK_REFERENCE.md # Command cheat sheet
├── monolith/                    # Go application
├── judge/                       # C++ code executor
└── common/                      # Shared libraries
```

---

## 🚦 Quick Start

### Prerequisites
- ✅ Windows 10/11
- ✅ Docker Desktop installed and running
- ✅ 4GB+ RAM available
- ✅ 10GB+ free disk space on D: drive

### Setup (5 minutes)

```powershell
# 1. Set environment
$env:MINIKUBE_HOME = "D:\minikube\.minikube"

# 2. Start Minikube
D:\minikube\minikube.exe start

# 3. Deploy application
D:\minikube\minikube.exe kubectl -- apply -f k8s/

# 4. Wait for pods to be ready (~2 minutes)
D:\minikube\minikube.exe kubectl -- get pods -n codejudge --watch

# 5. Access application
# Method 1: NodePort - http://192.168.49.2:30080
# Method 2: Port Forward - http://localhost:8080
D:\minikube\minikube.exe kubectl -- port-forward -n codejudge svc/monolith-service 8080:8080
```

---

## 🎬 Live Demo Script (5 minutes)

### 1. Show Architecture (30 seconds)
```powershell
# Display all running services
D:\minikube\minikube.exe kubectl -- get all -n codejudge
```
**Talking Point**: "Here you can see our microservices architecture with 4 main components: web application, database, cache, and code execution engine."

### 2. Demonstrate Self-Healing (60 seconds)
```powershell
# Delete a pod
D:\minikube\minikube.exe kubectl -- delete pod <monolith-pod-name> -n codejudge

# Watch automatic recreation
D:\minikube\minikube.exe kubectl -- get pods -n codejudge --watch
```
**Talking Point**: "Kubernetes provides automatic self-healing. When a pod fails, it's immediately recreated."

### 3. Show Scalability (60 seconds)
```powershell
# Scale to 3 replicas
D:\minikube\minikube.exe kubectl -- scale deployment/monolith --replicas=3 -n codejudge

# Verify scaling
D:\minikube\minikube.exe kubectl -- get pods -n codejudge
```
**Talking Point**: "The application scales horizontally. We can easily add more instances to handle increased load."

### 4. Demonstrate Monitoring (60 seconds)
```powershell
# View real-time logs
D:\minikube\minikube.exe kubectl -- logs -n codejudge deployment/monolith --tail=20 -f
```
**Talking Point**: "We have centralized logging for all services, making debugging and monitoring straightforward."

### 5. Access Application (60 seconds)
```
Open browser: http://localhost:8080
```
**Talking Point**: "The application is accessible via multiple methods - NodePort for production-like access, or port forwarding for local development."

---

## 💡 Technical Highlights

### Infrastructure as Code
- All infrastructure defined in version-controlled YAML files
- Reproducible deployments across environments
- Easy to review and audit changes

### Persistent Storage
- 3 PersistentVolumeClaims for data durability
- Survives pod restarts and deletions
- Automatic volume provisioning

### Configuration Management
- **ConfigMaps**: Non-sensitive configuration
- **Secrets**: Sensitive data (credentials, keys)
- Environment-based configuration injection

### Health Monitoring
```yaml
livenessProbe:  # Is the container running?
  httpGet:
    path: /health
    port: 8080
    
readinessProbe:  # Is the container ready for traffic?
  httpGet:
    path: /health
    port: 8080
```

### Security Features
- Sandboxed code execution (chroot, namespaces)
- Restricted Linux capabilities
- Network isolation between services
- Secret encryption at rest

---

## 📈 Scalability & Performance

### Current Configuration
- **Monolith**: 1 replica (can scale to N)
- **Database**: 1 replica (primary-replica setup possible)
- **Redis**: 1 replica (clustering available)
- **Judge**: 1 replica (can scale to N)

### Scaling Strategy
```powershell
# Horizontal scaling
kubectl scale deployment/monolith --replicas=5 -n codejudge

# Automatic scaling (HPA)
kubectl autoscale deployment/monolith --min=2 --max=10 --cpu-percent=80 -n codejudge
```

### Resource Optimization
- Optimized disk usage (D: drive, not C:)
- Efficient container images (multi-stage builds)
- Connection pooling for database
- Redis caching for performance

---

## 🔍 Troubleshooting

### Check Pod Status
```powershell
D:\minikube\minikube.exe kubectl -- get pods -n codejudge
```

### View Logs
```powershell
D:\minikube\minikube.exe kubectl -- logs -n codejudge deployment/monolith --tail=50
```

### Describe Pod (detailed info)
```powershell
D:\minikube\minikube.exe kubectl -- describe pod <pod-name> -n codejudge
```

### Common Issues
- **CrashLoopBackOff**: Wait for dependencies (DB, Redis) to start
- **ImagePullBackOff**: Rebuild images in Minikube's Docker daemon
- **Pending**: Check resource availability

---

## 🎓 Learning Outcomes

This project demonstrates proficiency in:

1. **Container Orchestration**: Kubernetes deployment and management
2. **Microservices Architecture**: Service decomposition and communication
3. **DevOps Practices**: CI/CD readiness, IaC, monitoring
4. **Cloud-Native Design**: Scalability, resilience, observability
5. **Security**: Sandboxing, secrets management, least privilege
6. **Documentation**: Comprehensive guides and runbooks

---

## 🚀 Production Readiness

### What's Included
✅ Health checks and monitoring  
✅ Persistent storage  
✅ Service discovery  
✅ Configuration management  
✅ Security policies  
✅ Comprehensive documentation  

### Next Steps for Production
- [ ] Deploy to cloud (AKS, EKS, GKE)
- [ ] Add Ingress controller (nginx, traefik)
- [ ] Implement TLS/SSL certificates
- [ ] Add monitoring (Prometheus, Grafana)
- [ ] Implement CI/CD pipeline
- [ ] Use managed databases
- [ ] Add backup solution (Velero)

---

## 📚 Documentation

- **Full Deployment Guide**: [`docs/MINIKUBE_DEPLOYMENT.md`](docs/MINIKUBE_DEPLOYMENT.md)
- **Quick Reference**: [`docs/MINIKUBE_QUICK_REFERENCE.md`](docs/MINIKUBE_QUICK_REFERENCE.md)
- **Architecture**: [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md)
- **Deployment**: [`docs/DEPLOYMENT.md`](docs/DEPLOYMENT.md)

---

## 🤝 Interview Questions Ready

**Q: Why Kubernetes instead of Docker Compose?**  
A: Kubernetes provides production-ready orchestration with self-healing, horizontal scaling, service discovery, and is the industry standard for container orchestration.

**Q: How do you handle data persistence?**  
A: Using PersistentVolumeClaims that survive pod restarts. PostgreSQL, Redis, and Judge submissions all use persistent storage.

**Q: How is security implemented?**  
A: Multi-layered approach: sandboxed code execution, restricted Linux capabilities, namespace isolation, and secret management.

**Q: Can this scale?**  
A: Yes, both horizontally (more replicas) and vertically (more resources per pod). Can implement auto-scaling with HPA.

**Q: What about monitoring?**  
A: Health checks on all services, centralized logging, and ready for Prometheus/Grafana integration.

---

## 📞 Contact

**Branch**: `minikube-deploy`  
**Documentation**: `docs/MINIKUBE_DEPLOYMENT.md`  
**Quick Start**: `docs/MINIKUBE_QUICK_REFERENCE.md`

---

**Built with ❤️ using Kubernetes, Docker, Go, and C++**
