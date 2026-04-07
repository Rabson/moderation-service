# Separated Docker Compose Architecture

Your deployment is now modularized for flexibility and SSL support.

## 📁 Compose Files

### Core Files (always used together)

| File                              | Services                                | Purpose                                            |
| --------------------------------- | --------------------------------------- | -------------------------------------------------- |
| **docker-compose.shared.yml**     | postgres, redis, zookeeper, kafka       | Core infrastructure (volumes: postgres-data)       |
| **docker-compose.moderation.yml** | moderation-service, ollama, ollama-init | LLM classification pipeline (volumes: ollama-data) |
| **docker-compose.gateway.yml**    | gateway-service, api-service            | Public API gateway + internal API                  |

### Optional Files

| File                       | Services | Purpose                               |
| -------------------------- | -------- | ------------------------------------- |
| **docker-compose.ssl.yml** | nginx    | SSL/TLS termination (requires certs/) |

---

## 🚀 Quick Start

### Option A: HTTP (Development)

```bash
docker-compose -f docker-compose.shared.yml \
                -f docker-compose.moderation.yml \
                -f docker-compose.gateway.yml \
                up --build
```

Access: `http://localhost:8080`

### Option B: HTTPS (Production)

**Step 1:** Generate self-signed certificate (or use Let's Encrypt)

```bash
mkdir -p certs
openssl req -x509 -newkey rsa:4096 \
  -keyout certs/server.key \
  -out certs/server.crt \
  -days 365 -nodes \
  -subj "/CN=yourdomain.com"
```

**Step 2:** Start with SSL

```bash
docker-compose -f docker-compose.shared.yml \
                -f docker-compose.moderation.yml \
                -f docker-compose.gateway.yml \
                -f docker-compose.ssl.yml \
                up --build
```

Access: `https://localhost:443`

### Option C: Use Interactive Script

```bash
./setup.sh
```

---

## 🔐 SSL/HTTPS Without Code Changes

**Key Features:**

- ✅ nginx reverse proxy as SSL termination layer
- ✅ HTTP → HTTPS automatic redirect
- ✅ TLS 1.2 + 1.3 support
- ✅ Rate limiting (general + per-API-key)
- ✅ Security headers (HSTS, X-Frame-Options, etc.)
- ✅ **Zero code changes** — gateway service unchanged

**Architecture:**

```
Client (HTTPS)
    ↓
nginx (port 443, handles SSL)
    ↓
gateway-service (port 8080, HTTP)
    ↓
api-service (internal, port 8080)
    ↓
moderation-service (internal, port 8081)
    ↓
Ollama LLM (internal, port 11434)
```

---

## 🔑 API Examples

### Create API Key

```bash
curl -X POST https://localhost:443/admin/keys \
  --cacert certs/server.crt \
  -H 'X-Admin-Secret: change-me-in-production' \
  -H 'Content-Type: application/json' \
  -d '{"name":"test-key","requests_per_minute":60}'
```

### Moderate Text

```bash
curl -X POST https://localhost:443/moderate \
  --cacert certs/server.crt \
  -H 'X-API-Key: <key-from-above>' \
  -H 'Content-Type: application/json' \
  -d '{"text":"I will k1ll you"}'
```

### Health Check

```bash
curl https://localhost:443/healthz --cacert certs/server.crt
```

---

## 🧹 Clean Up

### Stop All Services

```bash
docker-compose -f docker-compose.shared.yml \
                -f docker-compose.moderation.yml \
                -f docker-compose.gateway.yml \
                down
```

### Remove Volumes (⚠️ data loss)

```bash
docker-compose -f docker-compose.shared.yml \
                -f docker-compose.moderation.yml \
                -f docker-compose.gateway.yml \
                down -v
```

### Logs

```bash
# All services
docker-compose -f docker-compose.shared.yml \
                -f docker-compose.moderation.yml \
                -f docker-compose.gateway.yml \
                logs -f

# Specific service
docker logs -f gateway-service
docker logs -f moderation-service
docker logs -f nginx-ssl
```

---

## 🛠 Custom SSL Certificates

### Using Let's Encrypt (Certbot)

```bash
# Install certbot if needed: sudo snap install certbot

# Generate certificate
sudo certbot certonly --standalone -d yourdomain.com

# Copy to project
mkdir -p certs
sudo cp /etc/letsencrypt/live/yourdomain.com/fullchain.pem certs/server.crt
sudo cp /etc/letsencrypt/live/yourdomain.com/privkey.pem certs/server.key
sudo chown $(whoami):$(whoami) certs/*

# Start with SSL
docker-compose -f docker-compose.shared.yml \
                -f docker-compose.moderation.yml \
                -f docker-compose.gateway.yml \
                -f docker-compose.ssl.yml \
                up --build
```

### Using AWS ACM or Other CA

1. Obtain fullchain.pem and private.key from your CA
2. Place in `./certs/` directory
3. Run with SSL compose flag

---

## 📊 Service Dependencies

```
nginx (OPTIONAL)
  └─ gateway-service (REQUIRED)
      └─ api-service (REQUIRED)
          └─ moderation-service (REQUIRED)
              ├─ ollama (REQUIRED)
              │   └─ ollama-init (bootstrap only)
              ├─ postgres (REQUIRED)
              └─ redis (REQUIRED)

postgres (SHARED) ← all services
redis (SHARED) ← all services
kafka (OPTIONAL, profiles: ["kafka"])
```

---

## 🎯 Composition Strategies

### Minimal (moderation only)

```bash
docker-compose -f docker-compose.shared.yml \
                -f docker-compose.moderation.yml \
                up --build
```

### Standard (with gateway)

```bash
docker-compose -f docker-compose.shared.yml \
                -f docker-compose.moderation.yml \
                -f docker-compose.gateway.yml \
                up --build
```

### Production (with SSL)

```bash
docker-compose -f docker-compose.shared.yml \
                -f docker-compose.moderation.yml \
                -f docker-compose.gateway.yml \
                -f docker-compose.ssl.yml \
                up --build -d
```

### With Kafka

```bash
docker-compose -f docker-compose.shared.yml \
                -f docker-compose.moderation.yml \
                -f docker-compose.gateway.yml \
                --profile kafka \
                up --build
```

---

## 🔗 Useful Links

- [nginx configuration](deploy/nginx/nginx.conf) - SSL/TLS & routing rules
- [docker-compose.shared.yml](docker-compose.shared.yml) - Shared infrastructure
- [docker-compose.moderation.yml](docker-compose.moderation.yml) - LLM stack
- [docker-compose.gateway.yml](docker-compose.gateway.yml) - API gateway
- [docker-compose.ssl.yml](docker-compose.ssl.yml) - SSL layer
- [COMPOSE_STRUCTURE.md](COMPOSE_STRUCTURE.md) - Detailed structure guide

---

## ⚠️ Important Notes

1. **Old docker-compose.yml:** Still exists but not used. You can delete it after confirming new setup works.
2. **Environment variables:** Still loaded from `.env` file
3. **Volumes:** Named volumes (postgres-data, ollama-data) persist across restarts
4. **Port assignments:** All internal services (8080, 8081, 11434) are still accessible for debugging
5. **SSL certificates:** Place in `certs/` directory before using docker-compose.ssl.yml
6. **nginx:** Only runs if you explicitly include `docker-compose.ssl.yml`

---

## ✅ Verification Checklist

- [ ] `docker compose config` passes for all file combinations
- [ ] `docker-compose up --build` starts all services successfully
- [ ] `curl http://localhost:8080/healthz` returns `{"status":"ok"}`
- [ ] API key creation works: `POST /admin/keys`
- [ ] Moderation endpoint works: `POST /moderate`
- [ ] Rate limiting enforced: 61+ requests/min returns 429
- [ ] SSL works (if enabled): `https://localhost:443` traffic encrypted
- [ ] Logs show no errors: `docker logs -f <service>`
