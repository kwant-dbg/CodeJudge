i hav# Automatic Failover to Maintenance Page

This guide sets up **automatic failover** from Azure to GitHub Pages when Azure is down/stopped.

## 🎯 Architecture

```
User Request
    ↓
Cloudflare DNS/Load Balancer
    ├─→ Primary: Azure Container Instance (health checked)
    │   └─ If healthy: Serve live application
    └─→ Fallback: GitHub Pages (always up)
        └─ If Azure down: Serve maintenance page
```

## 📋 Prerequisites

- ✅ Azure deployment running
- ✅ GitHub Pages enabled (gh-pages branch exists)
- ✅ Custom domain (optional but recommended)
- ✅ Cloudflare account (free tier is fine)

---

## 🚀 Setup Method 1: Cloudflare Load Balancer (Recommended)

### Step 1: Setup Cloudflare

1. **Sign up for Cloudflare**: https://dash.cloudflare.com/sign-up
2. **Add your domain** (or use a free one like `yourusername.cloudflare.com`)

### Step 2: Configure DNS Records

In Cloudflare DNS settings:

```
Type    Name    Content                                         Proxy   TTL
A       @       <your-azure-container-ip>                       ✅      Auto
CNAME   www     kwant-dbg.github.io                             ❌      Auto
```

### Step 3: Create Load Balancer

1. Go to **Traffic** → **Load Balancing**
2. Click **Create Load Balancer**

**Pool 1 (Azure - Primary)**:
```yaml
Name: azure-pool
Origin: <your-azure-fqdn>:8080
Health Check:
  - Type: HTTP
  - Port: 8080
  - Path: /health
  - Interval: 30s
  - Timeout: 10s
  - Retries: 2
```

**Pool 2 (GitHub Pages - Fallback)**:
```yaml
Name: maintenance-pool
Origin: kwant-dbg.github.io
Health Check:
  - Type: HTTPS
  - Path: /
  - Interval: 60s
```

**Load Balancer Settings**:
```yaml
Hostname: codejudge.yourdomain.com
Default Pool: azure-pool
Fallback Pool: maintenance-pool
TTL: 30 seconds
Steering Policy: Off - Failover
```

**Cost**: ~$5/month (Cloudflare Load Balancing)

---

## 🆓 Setup Method 2: Cloudflare Workers (FREE!)

Use Cloudflare Workers to check health and redirect automatically.

### Step 1: Create Worker Script

Go to **Workers** → **Create a Worker**

Paste this script:

```javascript
addEventListener('fetch', event => {
  event.respondWith(handleRequest(event.request))
})

async function handleRequest(request) {
  // Your Azure endpoint
  const AZURE_URL = 'http://your-azure-fqdn.southeastasia.azurecontainer.io:8080'
  const MAINTENANCE_URL = 'https://kwant-dbg.github.io/CodeJudge/'
  
  try {
    // Check if Azure is healthy
    const healthCheck = await fetch(`${AZURE_URL}/health`, {
      method: 'GET',
      timeout: 5000 // 5 second timeout
    })
    
    if (healthCheck.ok) {
      // Azure is up - proxy the request
      const url = new URL(request.url)
      const azureRequest = new Request(`${AZURE_URL}${url.pathname}${url.search}`, {
        method: request.method,
        headers: request.headers,
        body: request.body
      })
      
      return fetch(azureRequest)
    } else {
      // Azure is down - redirect to maintenance page
      return Response.redirect(MAINTENANCE_URL, 302)
    }
  } catch (error) {
    // Health check failed - show maintenance page
    return Response.redirect(MAINTENANCE_URL, 302)
  }
}
```

### Step 2: Configure Worker Route

1. Go to **Workers** → **your-worker** → **Triggers**
2. Add route: `*yourdomain.com/*`
3. Save and deploy

**Cost**: FREE! (100,000 requests/day on free tier)

---

## 🔧 Setup Method 3: Azure Front Door (Azure-Native)

If you want to stay within Azure ecosystem:

### Create Azure Front Door

```powershell
# Create Front Door profile
az afd profile create `
  --profile-name codejudge-fd `
  --resource-group codejudge-sea-rg `
  --sku Standard_AzureFrontDoor

# Add origin group with health probe
az afd origin-group create `
  --profile-name codejudge-fd `
  --origin-group-name azure-origins `
  --probe-path /health `
  --probe-protocol Http `
  --probe-interval-in-seconds 30 `
  --resource-group codejudge-sea-rg

# Add Azure Container as origin
az afd origin create `
  --profile-name codejudge-fd `
  --origin-group-name azure-origins `
  --origin-name azure-container `
  --host-name your-azure-fqdn.azurecontainer.io `
  --origin-host-header your-azure-fqdn.azurecontainer.io `
  --http-port 8080 `
  --priority 1 `
  --weight 1000 `
  --resource-group codejudge-sea-rg

# Add GitHub Pages as backup origin
az afd origin create `
  --profile-name codejudge-fd `
  --origin-group-name azure-origins `
  --origin-name github-pages `
  --host-name kwant-dbg.github.io `
  --origin-host-header kwant-dbg.github.io `
  --https-port 443 `
  --priority 2 `
  --weight 100 `
  --resource-group codejudge-sea-rg
```

**Cost**: ~$35-50/month (Azure Front Door Standard tier)

---

## ⚡ Quick Start: Cloudflare Worker (Recommended for You)

### 1. Get Your Azure FQDN

```powershell
az container show `
  --resource-group codejudge-sea-rg `
  --name codejudge-monolith `
  --query "ipAddress.fqdn" `
  -o tsv
```

### 2. Enable GitHub Pages

Already done! Your maintenance page is at:
- https://kwant-dbg.github.io/CodeJudge/

### 3. Create Cloudflare Worker

Use the script above, replacing:
- `AZURE_URL`: Your Azure FQDN from step 1
- `MAINTENANCE_URL`: Your GitHub Pages URL

### 4. Test It!

```powershell
# Stop Azure
az container stop --resource-group codejudge-sea-rg --name codejudge-monolith

# Visit your domain - should show maintenance page

# Start Azure
az container start --resource-group codejudge-sea-rg --name codejudge-monolith

# Visit your domain - should show live site (after ~30 seconds)
```

---

## 🎨 Customize Maintenance Page

The maintenance page is in the `gh-pages` branch. To update:

```powershell
# Switch to gh-pages branch
git checkout gh-pages

# Edit the maintenance page
notepad index.html

# Commit and push
git add .
git commit -m "Update maintenance page"
git push origin gh-pages

# Switch back to main branch
git checkout mono
```

---

## 📊 Monitoring

### Cloudflare Dashboard
- Go to **Analytics** → **Traffic**
- See origin health status
- View failover events

### Azure Container Logs
```powershell
az container logs `
  --resource-group codejudge-sea-rg `
  --name codejudge-monolith `
  --tail 50
```

### Health Check Endpoint
```powershell
# Check if Azure is healthy
Invoke-WebRequest -Uri "http://your-azure-fqdn:8080/health"
```

---

## 💰 Cost Comparison

| Method | Monthly Cost | Pros | Cons |
|--------|-------------|------|------|
| **Cloudflare Worker** | **FREE** | ✅ Zero cost<br>✅ Fast<br>✅ Global CDN | ⚠️ Requires custom domain |
| **Cloudflare Load Balancer** | ~$5 | ✅ Built-in health checks<br>✅ Easy setup | 💰 Small cost |
| **Azure Front Door** | ~$35-50 | ✅ Azure-native<br>✅ Advanced features | 💰 Expensive |

**Recommendation**: Use **Cloudflare Worker (FREE)** for your low-traffic scenario.

---

## 🔒 Security Notes

- Maintenance page is public (on GitHub Pages) - don't include sensitive info
- Health endpoint `/health` should be lightweight
- Consider adding rate limiting in Cloudflare
- Use HTTPS everywhere (Cloudflare provides free SSL)

---

## 📝 Next Steps

1. ✅ Choose a method (Worker recommended)
2. ✅ Sign up for Cloudflare (if using Worker)
3. ✅ Deploy the worker script
4. ✅ Test by stopping Azure
5. ✅ Customize maintenance page in gh-pages branch

Need help with any step? Let me know!
