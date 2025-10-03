# Azure Deployment Checklist

## Pre-Deployment Validation

- [ ] Run deployment readiness script: `bash scripts/check-deployment-readiness.sh`
- [ ] Verify all environment variables are configured
- [ ] Test services locally with production configuration
- [ ] Validate health endpoints are responding

## Azure Resource Creation Order

1. **Resource Group**
   ```bash
   az group create --name codejudge-rg --location eastus
   ```

2. **Container Registry**
   ```bash
   az acr create --resource-group codejudge-rg --name codejudgeregistry --sku Basic
   ```

3. **Database Services**
   - PostgreSQL Flexible Server (Burstable B1ms)
   - Azure Cache for Redis (Basic C0)

4. **Container Deployment**
   - App Service with container support, OR
   - Container Instances for cost-effective testing

## Deployment Steps

1. **Build and Push Images**
   ```bash
   # Login to ACR
   az acr login --name codejudgeregistry
   
   # Build and push all services
   docker-compose -f docker-compose.prod.yml build
   docker-compose -f docker-compose.prod.yml push
   ```

2. **Configure Environment Variables**
   - Use Azure Key Vault for sensitive data
   - Set connection strings for PostgreSQL and Redis
   - Configure file storage paths

3. **Deploy Services**
   - Use provided Kubernetes manifests, OR
   - Deploy individual containers to App Service

4. **Verify Deployment**
   - Check health endpoints: `/health` and `/ready`
   - Test core functionality
   - Monitor logs for errors

## Cost Optimization

- Use Basic tier services for development
- Set auto-shutdown schedules for non-production resources
- Monitor spending with Azure Cost Management
- Clean up unused resources regularly

## Troubleshooting

- **Container Start Issues**: Check environment variables and secrets
- **Database Connection**: Verify firewall rules and connection strings
- **Health Check Failures**: Check service dependencies and startup order
- **Performance Issues**: Monitor resource usage and scale appropriately

## Cost Optimization

- Use Basic tier services for development
- Set auto-shutdown schedules for non-production resources
- Monitor spending with Azure Cost Management
- Clean up unused resources regularly