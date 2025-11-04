# Minikube Quick Reference Guide

## Essential Commands

### Cluster Management
```powershell
# Start Minikube
$env:MINIKUBE_HOME = "D:\minikube\.minikube"
D:\minikube\minikube.exe start

# Stop Minikube
D:\minikube\minikube.exe stop

# Delete cluster
D:\minikube\minikube.exe delete

# Get cluster status
D:\minikube\minikube.exe status

# Get Minikube IP
D:\minikube\minikube.exe ip
```

### Deployment Commands
```powershell
# Deploy all resources
D:\minikube\minikube.exe kubectl -- apply -f k8s/namespace.yaml
D:\minikube\minikube.exe kubectl -- apply -f k8s/configmap.yaml
D:\minikube\minikube.exe kubectl -- apply -f k8s/secrets.yaml
D:\minikube\minikube.exe kubectl -- apply -f k8s/postgres.yaml
D:\minikube\minikube.exe kubectl -- apply -f k8s/redis.yaml
D:\minikube\minikube.exe kubectl -- apply -f k8s/monolith.yaml
D:\minikube\minikube.exe kubectl -- apply -f k8s/judge.yaml

# Or deploy all at once
D:\minikube\minikube.exe kubectl -- apply -f k8s/
```

### Monitoring Commands
```powershell
# Check all pods
D:\minikube\minikube.exe kubectl -- get pods -n codejudge

# Watch pods in real-time
D:\minikube\minikube.exe kubectl -- get pods -n codejudge --watch

# Get detailed pod info
D:\minikube\minikube.exe kubectl -- describe pod <pod-name> -n codejudge

# View logs
D:\minikube\minikube.exe kubectl -- logs -n codejudge deployment/monolith --tail=50
D:\minikube\minikube.exe kubectl -- logs -n codejudge deployment/judge --tail=50
D:\minikube\minikube.exe kubectl -- logs -n codejudge deployment/postgres --tail=50

# Follow logs in real-time
D:\minikube\minikube.exe kubectl -- logs -n codejudge deployment/monolith -f
```

### Service Access
```powershell
# Get services
D:\minikube\minikube.exe kubectl -- get svc -n codejudge

# Port forward to localhost
D:\minikube\minikube.exe kubectl -- port-forward -n codejudge svc/monolith-service 8080:8080

# Get service URL (alternative method)
D:\minikube\minikube.exe service monolith-service -n codejudge --url
```

### Image Building
```powershell
# Configure Docker to use Minikube's daemon
D:\minikube\minikube.exe docker-env | Invoke-Expression

# Build images
docker build -t codejudge-monolith:latest -f monolith/Dockerfile.standalone .
docker build -t codejudge-judge:latest -f judge/Dockerfile.modern judge/

# List images in Minikube
docker images | Select-String codejudge
```

### Troubleshooting
```powershell
# Get pod events
D:\minikube\minikube.exe kubectl -- get events -n codejudge --sort-by='.lastTimestamp'

# Describe failing pod
D:\minikube\minikube.exe kubectl -- describe pod <pod-name> -n codejudge

# Execute command in pod
D:\minikube\minikube.exe kubectl -- exec -it <pod-name> -n codejudge -- /bin/sh

# Check resource usage
D:\minikube\minikube.exe kubectl -- top pods -n codejudge
D:\minikube\minikube.exe kubectl -- top nodes
```

### Scaling & Updates
```powershell
# Scale deployment
D:\minikube\minikube.exe kubectl -- scale deployment/monolith --replicas=3 -n codejudge

# Restart deployment
D:\minikube\minikube.exe kubectl -- rollout restart deployment/monolith -n codejudge

# Check rollout status
D:\minikube\minikube.exe kubectl -- rollout status deployment/monolith -n codejudge

# Rollback deployment
D:\minikube\minikube.exe kubectl -- rollout undo deployment/monolith -n codejudge
```

### Cleanup
```powershell
# Delete namespace (removes all resources)
D:\minikube\minikube.exe kubectl -- delete namespace codejudge

# Delete specific resource
D:\minikube\minikube.exe kubectl -- delete deployment monolith -n codejudge
D:\minikube\minikube.exe kubectl -- delete -f k8s/monolith.yaml
```

## Access URLs

- **Via NodePort**: `http://192.168.49.2:30080`
- **Via Port Forward**: `http://localhost:8080` (after running port-forward command)

## File Locations

- **Minikube Home**: `D:\minikube\.minikube`
- **Minikube Executable**: `D:\minikube\minikube.exe`
- **Kubernetes Manifests**: `D:\dev\codejudge\k8s\`
- **Documentation**: `D:\dev\codejudge\docs\MINIKUBE_DEPLOYMENT.md`

## Pod Status Meanings

- **Running**: Pod is running successfully
- **Pending**: Waiting for resources or scheduling
- **CrashLoopBackOff**: Pod keeps failing and restarting
- **Error**: Pod encountered an error
- **ContainerCreating**: Container is being created
- **ImagePullBackOff**: Cannot pull the container image

## Common Issues

### Issue: Pods CrashLoopBackOff
**Solution**: Wait for dependencies (PostgreSQL, Redis) to be ready first. Check logs.

### Issue: Cannot access via localhost
**Solution**: Run port-forward command or use Minikube IP with NodePort (30080).

### Issue: Images not found
**Solution**: Configure Docker daemon and rebuild images inside Minikube context.

### Issue: Out of resources
**Solution**: Stop and restart Minikube with more resources:
```powershell
D:\minikube\minikube.exe stop
D:\minikube\minikube.exe start --cpus=4 --memory=8192
```

## Daily Workflow

### Starting Your Day
```powershell
# 1. Set environment
$env:MINIKUBE_HOME = "D:\minikube\.minikube"

# 2. Start Minikube
D:\minikube\minikube.exe start

# 3. Check status
D:\minikube\minikube.exe kubectl -- get pods -n codejudge

# 4. Start port forwarding (in separate window)
D:\minikube\minikube.exe kubectl -- port-forward -n codejudge svc/monolith-service 8080:8080

# 5. Access application
# Open browser: http://localhost:8080
```

### Ending Your Day
```powershell
# Stop port forwarding (Ctrl+C in port-forward window)

# Stop Minikube
D:\minikube\minikube.exe stop
```

### Making Code Changes
```powershell
# 1. Configure Docker
D:\minikube\minikube.exe docker-env | Invoke-Expression

# 2. Rebuild image
docker build -t codejudge-monolith:latest -f monolith/Dockerfile.standalone .

# 3. Restart deployment
D:\minikube\minikube.exe kubectl -- rollout restart deployment/monolith -n codejudge

# 4. Check rollout
D:\minikube\minikube.exe kubectl -- rollout status deployment/monolith -n codejudge
```

## Interview Demo Script

### 1. Show Architecture
"Let me show you the microservices architecture deployed on Kubernetes..."
```powershell
D:\minikube\minikube.exe kubectl -- get pods -n codejudge
D:\minikube\minikube.exe kubectl -- get svc -n codejudge
```

### 2. Demonstrate Monitoring
"We have comprehensive logging and monitoring..."
```powershell
D:\minikube\minikube.exe kubectl -- logs -n codejudge deployment/monolith --tail=20
```

### 3. Show Scalability
"The application can scale horizontally..."
```powershell
D:\minikube\minikube.exe kubectl -- scale deployment/monolith --replicas=3 -n codejudge
D:\minikube\minikube.exe kubectl -- get pods -n codejudge
```

### 4. Demonstrate Self-Healing
"Kubernetes provides self-healing capabilities..."
```powershell
# Delete a pod
D:\minikube\minikube.exe kubectl -- delete pod <pod-name> -n codejudge

# Watch it automatically recreate
D:\minikube\minikube.exe kubectl -- get pods -n codejudge --watch
```

### 5. Show Access
"The application is accessible via multiple methods..."
```powershell
# Show NodePort
D:\minikube\minikube.exe kubectl -- get svc monolith-service -n codejudge

# Access via browser
# http://192.168.49.2:30080 or http://localhost:8080
```

## Key Points for Interviewer

✅ **Infrastructure as Code**: All configuration in version control  
✅ **Cloud-Native**: Uses Kubernetes, industry standard  
✅ **Security**: Sandboxed code execution with restricted capabilities  
✅ **Scalability**: Horizontal scaling with replicas  
✅ **Self-Healing**: Automatic pod restart on failures  
✅ **Observability**: Health checks, logs, metrics  
✅ **Storage**: Persistent volumes for data retention  
✅ **Resource Management**: Optimized for D: drive, not C:  
✅ **Production-Ready**: Same patterns as enterprise deployments  
✅ **Documentation**: Comprehensive guides and runbooks  

---

**Quick Access**: Save this file for easy reference during development and demos!
