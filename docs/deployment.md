# Deployment & Hosting Guide

This guide covers deploying **auto-store-api** to a production server (Hetzner Cloud recommended) using Docker Compose, exposing it over HTTPS, and connecting a separate frontend application.

## Architecture

```
Internet
   │
   ▼
Frontend (Vercel, Netlify, custom host)
   │  HTTPS → https://api.yourdomain.com/api/v1/...
   ▼
Caddy reverse proxy (:443)
   │
   ▼
Go API (:8089, localhost only in production)
   ├── PostgreSQL 15 (Docker, internal)
   ├── Redis 7 (Docker, internal)
   └── Worker (Docker, notifications + email)
```

| Component | Role |
|-----------|------|
| **API** (`cmd/api`) | HTTP server, GORM AutoMigrate on startup |
| **Worker** (`cmd/worker`) | Processes notification queue (email via SMTP) |
| **PostgreSQL** | Primary database — created automatically by Docker Compose |
| **Redis** | Rate limiting, sessions, notification queue |
| **Caddy** | HTTPS termination, reverse proxy (installed on host, not in Compose) |

> **Note:** Hetzner does not offer managed PostgreSQL. This project runs Postgres inside Docker on your VPS. That is sufficient for production when combined with backups and a firewall. See [Database](#database) below.

---

## Prerequisites

- A [Hetzner Cloud](https://console.hetzner.cloud/) account
- SSH key added to your Hetzner account
- A domain name (recommended for production HTTPS)
- Git repository access (public or deploy key for private repos)

**Recommended server spec to start:** CX22 or CPX21 (2 vCPU, 4 GB RAM), Ubuntu 24.04, location closest to your users.

---

## 1. Create and access the server

1. In Hetzner Cloud Console → **Add Server**
2. Choose Ubuntu 24.04, your preferred region, and attach your SSH key
3. Note the server's **public IP**
4. SSH in from your local machine:

```bash
ssh root@YOUR_SERVER_IP
```

### Initial hardening (recommended)

```bash
apt update && apt upgrade -y

# Create a non-root deploy user
adduser deploy
usermod -aG sudo deploy
mkdir -p /home/deploy/.ssh
cp ~/.ssh/authorized_keys /home/deploy/.ssh/
chown -R deploy:deploy /home/deploy/.ssh
chmod 700 /home/deploy/.ssh
chmod 600 /home/deploy/.ssh/authorized_keys

# Disable root/password SSH (edit /etc/ssh/sshd_config)
#   PermitRootLogin no
#   PasswordAuthentication no
# Then: systemctl restart sshd

# Firewall — only expose SSH and web ports
ufw allow OpenSSH
ufw allow 80/tcp
ufw allow 443/tcp
ufw enable
```

Do **not** expose ports `5432` (Postgres) or `6379` (Redis) to the public internet.

---

## 2. Install Docker

Log in as your deploy user:

```bash
ssh deploy@YOUR_SERVER_IP

curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
newgrp docker

sudo apt update
sudo apt install -y docker-compose-plugin git
docker compose version
```

---

## 3. Clone the repository

```bash
sudo mkdir -p /opt/auto-store-api
sudo chown $USER:$USER /opt/auto-store-api
cd /opt/auto-store-api

# Clone into the current directory (note the trailing dot)
git clone https://github.com/YOUR_USER/auto-store-api.git .
```

> **Common mistake:** Running `git clone <url>` without `.` creates a nested folder (`/opt/auto-store-api/auto-store-api/`). If that happened, run `cd auto-store-api` before the next steps.

---

## 4. Configure environment

```bash
cp .env.example .env
nano .env
```

### Required for production

```bash
PORT=8089
GIN_MODE=release

# Docker Compose service names (do not change DB_HOST / REDIS_HOST)
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=<strong-random-password>
DB_NAME=auto_store
DB_SSLMODE=disable

REDIS_HOST=redis
REDIS_PORT=6379

JWT_SECRET=<openssl rand -hex 32>
GUEST_JWT_SECRET=<openssl rand -hex 32>

# Your live frontend URL(s) — comma-separated, no trailing slash
CORS_ORIGINS=https://your-frontend.com
APP_FRONTEND_URL=https://your-frontend.com
```

Generate secrets:

```bash
openssl rand -hex 32
```

### Optional but recommended

```bash
# Email (required for worker notifications)
SMTP_HOST=smtp.yourprovider.com
SMTP_PORT=587
SMTP_USER=...
SMTP_PASSWORD=...
SMTP_FROM=noreply@yourdomain.com

# Paystack (see docs/payments.md)
PAYSTACK_SECRET_KEY=sk_live_...
PAYSTACK_PUBLIC_KEY=pk_live_...
PAYSTACK_WEBHOOK_SECRET=...
PAYSTACK_CALLBACK_URL=https://your-frontend.com/checkout/verify

# S3-compatible object storage (product images)
S3_BUCKET=...
S3_REGION=...
S3_ACCESS_KEY=...
S3_SECRET_KEY=...
S3_ENDPOINT=...
S3_PUBLIC_URL=...
```

See [.env.example](../.env.example) for the full list.

---

## 5. Database

You do **not** need to install PostgreSQL on Hetzner separately. Docker Compose creates it automatically when you run `docker compose up`:

| Setting | Value | Set by |
|---------|-------|--------|
| Database name | `auto_store` | `POSTGRES_DB` in Compose |
| User | `postgres` | `POSTGRES_USER` in Compose |
| Password | see below | `POSTGRES_PASSWORD` in Compose |
| Tables | auto-created | GORM AutoMigrate on API startup |

Data persists in the Docker volume `postgres_data` across restarts.

### Aligning database passwords

The stock `docker-compose.yml` hardcodes `postgres` as the password in three places (`api`, `worker`, and `postgres` services). That works out of the box for a first deploy.

For production, set a strong password and use the same value everywhere. Update `docker-compose.yml`:

```yaml
# postgres service
POSTGRES_PASSWORD: ${DB_PASSWORD:-postgres}

# api and worker services
DB_PASSWORD: ${DB_PASSWORD:-postgres}
```

Then set in `.env`:

```bash
DB_PASSWORD=your-strong-random-password
```

All three services must use the same password or the API will fail to connect.

---

## 6. Start the stack

```bash
cd /opt/auto-store-api
docker compose up -d --build
```

Verify all services are running:

```bash
docker compose ps
docker compose logs -f api
```

Test the health endpoint on the server:

```bash
curl http://localhost:8089/health
```

Swagger UI (after HTTPS is configured): `https://api.yourdomain.com/docs/index.html`

---

## 7. HTTPS with Caddy

For production, put Caddy in front of the API. It handles Let's Encrypt certificates automatically.

### DNS

Add an **A record** pointing your API subdomain to the server IP:

| Type | Name | Value |
|------|------|-------|
| A | `api` | `YOUR_SERVER_IP` |

### Bind API to localhost only

In `docker-compose.yml`, change the API port mapping so it is not publicly reachable:

```yaml
ports:
  - "127.0.0.1:8089:8089"
```

Restart:

```bash
docker compose up -d
```

### Install Caddy

```bash
sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https curl
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update && sudo apt install -y caddy
```

Create `/etc/caddy/Caddyfile`:

```caddyfile
api.yourdomain.com {
    reverse_proxy 127.0.0.1:8089
}
```

Reload Caddy:

```bash
sudo systemctl reload caddy
```

Verify:

```bash
curl https://api.yourdomain.com/health
```

---

## 8. Connect the frontend

The frontend cannot use `localhost:8089`. It must call your public API URL over HTTPS.

### Server-side (`.env` on Hetzner)

```bash
CORS_ORIGINS=https://your-frontend.com,https://www.your-frontend.com
APP_FRONTEND_URL=https://your-frontend.com
```

`CORS_ORIGINS` must exactly match the browser origin (scheme + host + port). `https://app.com` and `https://www.app.com` are different origins — include both if needed.

Restart after changes:

```bash
docker compose up -d
```

### Client-side (frontend environment)

Set your API base URL in the frontend build environment:

```bash
# Next.js
NEXT_PUBLIC_API_URL=https://api.yourdomain.com

# Vite
VITE_API_URL=https://api.yourdomain.com
```

Example requests:

```
GET  https://api.yourdomain.com/api/v1/products
POST https://api.yourdomain.com/api/v1/auth/login
```

### WebSocket (support chat)

Use `wss://` when the site is served over HTTPS:

```
wss://api.yourdomain.com/api/v1/ws/chat?token=YOUR_JWT
```

See [support-chat.md](./support-chat.md) for details.

### Paystack webhooks

Register in the Paystack dashboard:

```
POST https://api.yourdomain.com/webhooks/paystack
```

Set `PAYSTACK_CALLBACK_URL` to your **frontend** verify page, not the API. See [payments.md](./payments.md).

---

## 9. File storage (S3)

Product image uploads require S3-compatible storage. Options:

| Provider | Notes |
|----------|-------|
| **Hetzner Object Storage** | S3-compatible, same provider |
| **AWS S3 / Cloudflare R2** | If already in use |
| **MinIO on the same VPS** | Fine for dev; not recommended for production |

Example Hetzner Object Storage config:

```bash
S3_BUCKET=your-bucket
S3_REGION=fsn1
S3_ACCESS_KEY=...
S3_SECRET_KEY=...
S3_ENDPOINT=https://fsn1.your-objectstorage.com
S3_PUBLIC_URL=https://your-bucket.fsn1.your-objectstorage.com
```

See [frontend-product-image-upload.md](./frontend-product-image-upload.md).

---

## 10. Backups

Docker volumes are not backed up automatically. Schedule Postgres dumps:

```bash
sudo mkdir -p /opt/backups
crontab -e
```

Add:

```cron
0 3 * * * docker exec auto-store-api-postgres-1 pg_dump -U postgres auto_store | gzip > /opt/backups/auto_store_$(date +\%F).sql.gz
```

> Container name may differ. Run `docker compose ps` to confirm the Postgres container name.

Also enable **Hetzner server/volume snapshots** from the Cloud Console as a safety net.

Test a restore periodically:

```bash
gunzip -c /opt/backups/auto_store_YYYY-MM-DD.sql.gz | docker exec -i CONTAINER_NAME psql -U postgres auto_store
```

---

## 11. Deploying updates

### Manual deploy

```bash
cd /opt/auto-store-api
bash scripts/deploy.sh
```

Or step by step:

```bash
cd /opt/auto-store-api
git pull
docker compose up -d --build api worker
docker compose logs -f api worker
```

### Automatic deploy (GitHub Actions → Hetzner)

Pushing to `main` runs tests in GitHub Actions, then SSHs into the server and runs `scripts/deploy.sh`.

**One-time server setup**

1. Ensure the repo is cloned at `/opt/auto-store-api` (see [§3](#3-clone-the-repository)).
2. Make the deploy script executable:

```bash
chmod +x /opt/auto-store-api/scripts/deploy.sh
```

3. Confirm manual deploy works before enabling CI:

```bash
bash /opt/auto-store-api/scripts/deploy.sh
```

**One-time GitHub setup**

In the repository → **Settings** → **Secrets and variables** → **Actions**, add:

| Secret | Example | Purpose |
|--------|---------|---------|
| `HETZNER_HOST` | `123.45.67.89` or `api.yourdomain.com` | Server address |
| `HETZNER_USER` | `deploy` | SSH user (not `root`) |
| `HETZNER_SSH_KEY` | contents of private key | SSH auth for deploy |

Generate a deploy key (on your laptop):

```bash
ssh-keygen -t ed25519 -C "github-actions-deploy" -f ~/.ssh/auto-store-deploy -N ""
```

- Add the **public** key (`auto-store-deploy.pub`) to `/home/deploy/.ssh/authorized_keys` on the server.
- Add the **private** key (`auto-store-deploy`) as the `HETZNER_SSH_KEY` secret in GitHub.

The deploy user only needs permission to run Docker and write to `/opt/auto-store-api`. If `docker` requires sudo, add the user to the `docker` group (`sudo usermod -aG docker deploy`).

**What happens on each push to `main`**

1. `go test ./...` runs in GitHub Actions.
2. On success, Actions SSHs to the server and runs `scripts/deploy.sh`, which:
   - `git fetch` + `git reset --hard origin/main`
   - `docker compose up -d --build api worker`
   - `curl` health check on `http://127.0.0.1:8089/health`

Pull requests run tests only; they do not deploy.

To deploy a different branch, set `DEPLOY_BRANCH` on the server before running the script (default: `main`).

---

## 12. Production checklist

- [ ] Strong `JWT_SECRET`, `GUEST_JWT_SECRET`, and `DB_PASSWORD`
- [ ] `GIN_MODE=release`
- [ ] `CORS_ORIGINS` set to real frontend URL(s)
- [ ] Postgres and Redis **not** exposed publicly (remove `ports` from those services or bind to localhost)
- [ ] API bound to `127.0.0.1:8089` with Caddy handling HTTPS
- [ ] HTTPS working: `curl https://api.yourdomain.com/health`
- [ ] Worker container running (`docker compose ps`)
- [ ] SMTP configured and tested (register → verification email)
- [ ] Paystack webhook URL registered (if using payments)
- [ ] S3 uploads working (if using product images)
- [ ] Daily Postgres backups scheduled
- [ ] Frontend `API_URL` env var points to `https://api.yourdomain.com`
- [ ] GitHub Actions secrets set (`HETZNER_HOST`, `HETZNER_USER`, `HETZNER_SSH_KEY`) and auto-deploy tested

---

## Troubleshooting

### `cp: cannot stat '.env.example': No such file or directory`

You are in the wrong directory. After cloning, ensure you are inside the project root:

```bash
cd /opt/auto-store-api        # if you used `git clone ... .`
# or
cd /opt/auto-store-api/auto-store-api   # if clone created a nested folder
ls -la .env.example
```

### API container exits / database connection failed

```bash
docker compose logs api
docker compose logs postgres
```

Common causes:
- Password mismatch between `postgres`, `api`, and `worker` services
- Postgres not healthy yet — the API retries automatically when `DB_HOST=postgres`

### CORS errors in the browser

- Check `CORS_ORIGINS` matches the frontend origin exactly
- Restart API after changing `.env`: `docker compose up -d`
- Preflight requests must succeed — check the browser Network tab for the failing origin

### 502 Bad Gateway from Caddy

- API not running: `docker compose ps` and `docker compose logs api`
- Wrong upstream port — Caddy should proxy to `127.0.0.1:8089`

### Mixed content blocked

An HTTPS frontend cannot call an `http://` API. Ensure the frontend uses `https://api.yourdomain.com`.

### Paystack webhooks not firing

- Webhook URL must be HTTPS and publicly reachable
- Verify `PAYSTACK_WEBHOOK_SECRET` matches the Paystack dashboard
- Check API logs: `docker compose logs api | grep -i paystack`

---

## Alternative: external managed Postgres

As you scale, you can move Postgres off the VPS to a managed provider (Neon, Supabase, Ubicloud on Hetzner, etc.) and update `.env`:

```bash
DB_HOST=your-managed-host.example.com
DB_PORT=5432
DB_USER=...
DB_PASSWORD=...
DB_NAME=auto_store
DB_SSLMODE=require
```

Remove the `postgres` service from Compose and keep only `api`, `worker`, and `redis` on the VPS. No code changes are required.

---

## Related docs

- [payments.md](./payments.md) — Paystack setup and webhooks
- [notifications.md](./notifications.md) — Worker and email queue
- [support-chat.md](./support-chat.md) — WebSocket chat
- [frontend-product-image-upload.md](./frontend-product-image-upload.md) — S3 uploads
- [.env.example](../.env.example) — Full environment variable reference
