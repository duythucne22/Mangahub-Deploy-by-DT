# Mangahub Deployment Guide: Railway + Neon

Complete guide for deploying your Mangahub app to production using **Neon (PostgreSQL)** + **Railway (Go server)** on the **free tier**.

---

## Prerequisites

- GitHub account
- Railway account (sign up at [railway.app](https://railway.app))
- Neon account (sign up at [neon.tech](https://neon.tech))
- Your Mangahub code pushed to GitHub

---

## Step 1: Set Up Neon Database (5 minutes)

### 1.1 Create Neon Project
1. Go to [neon.tech](https://neon.tech) → Sign in with GitHub
2. Click **"New Project"**
3. Choose:
   - **Name**: `mangahub-postgres` (or any name)
   - **Region**: Select closest to you (e.g., US East, EU West)
   - **PostgreSQL version**: 15 (default)
4. Click **"Create Project"**

### 1.2 Get Connection String
1. In your Neon dashboard, click **"Connection Details"**
2. Copy the **"Connection string"** - it looks like:
   ```
   postgres://[user]:[password]@[host]/[dbname]?sslmode=require
   ```
3. **Save this!** You'll need it for Railway

### 1.3 Run Database Schema
1. In Neon dashboard, click **"SQL Editor"**
2. Open your local `deployments/schema.sql` file
3. Copy the entire contents
4. Paste into Neon SQL Editor
5. Make sure you are using **Run** (not **EXPLAIN/Analyze**)
6. Click **"Run"** (top right)
7. Wait for ✅ "Query executed successfully"

### 1.4 (Optional) Add Seed Data
1. Still in SQL Editor, open your `seed.sql` file
2. Copy and paste the entire file
3. Click **"Run"**
4. You should now have test manga, users, genres, etc.

**✅ Neon setup complete!** Your database is now live and will never pause.

---

## Step 2: Deploy to Railway (10 minutes)

### 2.1 Create Railway Project
1. Go to [railway.app](https://railway.app) → Sign in with GitHub
2. Click **"New Project"**
3. Select **"Deploy from GitHub repo"**
4. Choose your **Mangahub repository**
5. Railway will auto-detect it's a Go project

### 2.2 Set Environment Variables
1. In Railway project, click **"Variables"** tab
2. Add these **required** variables:

| Variable | Value | Notes |
|----------|-------|-------|
| `CONFIG_FILE` | `./configs/production.yaml` | Path to production config |
| `DB_HOST` | `ep-xxx-xxx.neon.tech` | From Neon connection string |
| `DB_PORT` | `5432` | Default PostgreSQL port |
| `DB_USER` | `[your-neon-user]` | From Neon connection string |
| `DB_PASSWORD` | `[your-neon-password]` | From Neon connection string |
| `DB_NAME` | `neondb` | Or `postgres` (check Neon string) |
| `DB_SSLMODE` | `require` | **Important**: Always require SSL |
| `JWT_SECRET` | `[generate-random-string]` | Use: `openssl rand -base64 32` |
| `ENABLE_TCP` | `false` | Railway doesn't support raw TCP |
| `ENABLE_UDP` | `false` | Railway doesn't support UDP |
| `GIN_MODE` | `release` | Production mode |
| `PORT` | `8080` | Railway auto-sets this (optional) |

**Generate JWT Secret:**
```bash
# On your local machine:
openssl rand -base64 32
# Copy the output to JWT_SECRET
```

### 2.3 Configure Build
Railway auto-detects Go, but you can customize:

1. In Railway, click **"Settings"** → **"Build"**
2. **Build Command**: (leave default or set to)
   ```
   go build -o bin/server ./cmd/server
   ```
3. **Start Command**: (leave default or set to)
   ```
   ./bin/server
   ```

### 2.4 Deploy
1. Click **"Deploy"** (Railway auto-deploys on every GitHub push)
2. Wait 2-3 minutes for build + deploy
3. Check **"Deployments"** tab for progress
4. Look for ✅ "Deployment successful"

### 2.5 Get Your Live URL
1. In Railway project, click **"Settings"** → **"Domains"**
2. Click **"Generate Domain"**
3. Railway gives you: `https://[your-app].up.railway.app`
4. **Save this URL!** This is your live API

**✅ Railway deployment complete!** Your API is now live.

---

## Step 3: Test Your Deployment

### 3.1 Health Check
```bash
curl https://[your-app].up.railway.app/health
```
Expected response:
```json
{"status":"ok","timestamp":"2026-01-21T..."}
```

### 3.2 Test Auth Endpoints
```bash
# Register a user
curl -X POST https://[your-app].up.railway.app/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"TestPassword123!"}'

# Login
curl -X POST https://[your-app].up.railway.app/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"TestPassword123!"}'
```

### 3.3 Test Manga Endpoints
```bash
# List manga (should show seed data if you ran seed.sql)
curl https://[your-app].up.railway.app/api/v1/manga

# Search manga
curl "https://[your-app].up.railway.app/api/v1/manga/search?q=one&page=1&limit=20"
```

### 3.4 Test WebSocket (from browser console)
```javascript
const ws = new WebSocket('wss://[your-app].up.railway.app/ws/manga/manga-001?token=YOUR_JWT_TOKEN');
ws.onopen = () => console.log('Connected!');
ws.onmessage = (e) => console.log('Message:', e.data);
```

**✅ If all tests pass, your app is fully deployed!**

---

## Step 4: Update Your TUI Config (Local Dev)

Edit `configs/tui.yaml` to point to your production server:

```yaml
server:
  host: [your-app].up.railway.app
  http:
    port: 443  # HTTPS uses port 443
  grpc:
    port: 50051  # If you deploy gRPC separately
  ws:
    port: 443
    path: /ws/manga
```

Now run your TUI locally:
```bash
go run ./cmd/tui
```

It will connect to your live production server!

---

## Free Tier Limits & Monitoring

### Railway Free Tier
- **500 hours/month** execution time
- **512 MB** memory
- **100 GB** bandwidth
- **Monitor usage**: Railway Dashboard → "Usage" tab

**⚠️ Important**: 500 hours = ~20 days continuous runtime. After that:
- Option 1: App pauses until next month (free)
- Option 2: Upgrade to Hobby ($5/month) for unlimited hours

### Neon Free Tier
- **3 GB** storage
- **Always on** (no pause)
- **Unlimited** queries
- **Monitor usage**: Neon Dashboard → "Usage" tab

---

## Troubleshooting

### Problem: "Connection refused" or database errors

**Solution**: Check environment variables in Railway
```bash
# In Railway "Variables" tab, verify:
- DB_HOST is correct (from Neon)
- DB_PASSWORD is correct (no extra spaces)
- DB_SSLMODE is "require"
```

### Problem: "Build failed"

**Solution**: Check Railway build logs
1. Click "Deployments" → Latest deployment
2. Scroll to see error
3. Common issues:
   - Missing `go.mod` file
   - Import path errors (should be `mangahub/...`)
   - Missing dependencies: run `go mod tidy` locally, then push

### Problem: App deployed but returns 503

**Solution**: Check Railway logs
1. Click "Deployments" → View Logs
2. Look for panic or startup errors
3. Common issues:
   - Database connection failed (check ENV vars)
   - Port mismatch (Railway sets $PORT automatically)

### Problem: WebSocket connection fails

**Solution**: Use `wss://` (secure WebSocket) not `ws://`
```javascript
// ✅ Correct:
const ws = new WebSocket('wss://[your-app].up.railway.app/ws/manga/...');

// ❌ Wrong:
const ws = new WebSocket('ws://[your-app].up.railway.app/ws/manga/...');
```

---

## Next Steps

### Optional: Deploy gRPC as Separate Service

Railway allows multiple services per project:

1. In Railway project, click **"New"** → **"GitHub Repo"** (same repo)
2. Set **"Start Command"** to:
   ```
   ./bin/server-grpc
   ```
3. Add build script to only build gRPC server
4. Set environment variables (same as main service)
5. Railway gives you a second URL for gRPC

### Optional: Set Up Custom Domain

1. In Railway → "Settings" → "Domains"
2. Click **"Custom Domain"**
3. Enter your domain (e.g., `api.mangahub.com`)
4. Update your domain's DNS:
   - **Type**: CNAME
   - **Name**: `api` (or `@` for root)
   - **Value**: `[your-app].up.railway.app`
5. Wait 5-10 minutes for DNS propagation

### Optional: Enable GitHub Auto-Deploy

Railway auto-deploys on every push to `main` branch:

1. Push code to GitHub
2. Railway detects change
3. Builds + deploys automatically
4. View logs in Railway Dashboard

---

## Cost Summary (Free Tier)

| Service | Monthly Cost | Limitations |
|---------|--------------|-------------|
| **Neon (Database)** | $0 | 3 GB storage, always on |
| **Railway (Server)** | $0 | 500 hours (~20 days uptime) |
| **Total** | **$0** | Perfect for hobby projects |

**Upgrade path**:
- Railway Hobby: $5/month (unlimited hours, 8 GB RAM)
- Neon Pro: $19/month (more storage, branches, backups)

---

## Support & Resources

- **Railway Docs**: https://docs.railway.app
- **Neon Docs**: https://neon.tech/docs
- **Railway Discord**: https://discord.gg/railway
- **GitHub Issues**: Your Mangahub repo issues tab

---

**🎉 Congratulations!** Your Mangahub app is now live in production.

Share your API URL with friends to test it out!
