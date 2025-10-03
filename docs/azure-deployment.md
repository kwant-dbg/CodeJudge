# Azure Deployment Guide

## Resource Recommendations

### Compute Resources
- **API Gateway**: 1 vCPU, 1GB RAM
- **Problems Service**: 1 vCPU, 1GB RAM  
- **Submissions Service**: 1 vCPU, 2GB RAM
- **Plagiarism Service**: 1 vCPU, 1GB RAM
- **Judge Service**: 2 vCPU, 2GB RAM (for code execution)

### Storage & Database
- **PostgreSQL Flexible Server**: Burstable tier (1-2 vCPU)
- **Redis Cache**: Basic tier (250MB - 1GB)
- **Container Registry**: Basic tier
- **Storage Account**: Standard LRS for file storage

## Deployment Strategies

### Option 1: App Service (Recommended for Simplicity)
- Deploy each service as individual App Service containers
- Built-in load balancing and auto-scaling
- Easy SSL/TLS certificate management
- Integrated with Application Insights

### Option 2: Container Instances (Cost-Effective)
- Lower overhead than App Service
- Pay-per-second billing
- Good for development and testing
- Manual networking configuration required

### Option 3: Azure Kubernetes Service (Advanced)
- Full orchestration capabilities
- Use provided Kubernetes manifests
- Best for complex deployments
- Higher management overhead

## Architecture Recommendations
- Use Azure Container Instances for cost efficiency
- Single PostgreSQL Flexible Server (Basic tier)
- Single Redis Cache (Basic tier) 
- Azure Container Registry (Basic tier)
- App Service Plan (Basic tier for production)

# Cost Optimization Tips:
# 1. Use "Free" tier where available
# 2. Scale down or stop services when not in use
# 3. Use shared resource groups
# 4. Monitor spending dashboard regularly
# 5. Clean up unused resources

# Environment Variables for Azure Deployment:
# DATABASE_URL=postgresql://user:pass@server.postgres.database.azure.com:5432/codejudge?sslmode=require
# REDIS_URL=rediss://cache.redis.cache.windows.net:6380
# AZURE_CONTAINER_REGISTRY=yourregistry.azurecr.io