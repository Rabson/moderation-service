# Quick Reference: New Compose Structure

## 📋 Before vs After

### Before (Monolithic)

```bash
docker-compose up --build
# Single file: docker-compose.yml
# All services tightly coupled
```

### After (Modular)

```bash
# HTTP (development)
docker-compose -f docker-compose.shared.yml \
                -f docker-compose.moderation.yml \
                -f docker-compose.gateway.yml \
                up --build

# HTTPS (production - with SSL)
docker-compose -f docker-compose.shared.yml \
                -f docker-compose.moderation.yml \
                -f docker-compose.gateway.yml \
                -f docker-compose.ssl.yml \
                up --build
```

---

## 🎯 Common Commands

| Goal                   | Command                                                                                                                                           |
| ---------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Start (HTTP)**       | `docker-compose -f docker-compose.shared.yml -f docker-compose.moderation.yml -f docker-compose.gateway.yml up --build`                           |
| **Start (HTTPS)**      | `docker-compose -f docker-compose.shared.yml -f docker-compose.moderation.yml -f docker-compose.gateway.yml -f docker-compose.ssl.yml up --build` |
| **Start (daemon)**     | Add `-d` flag                                                                                                                                     |
| **Stop all**           | `docker-compose -f docker-compose.shared.yml -f docker-compose.moderation.yml -f docker-compose.gateway.yml down`                                 |
| **View logs**          | `docker-compose -f docker-compose.shared.yml -f docker-compose.moderation.yml -f docker-compose.gateway.yml logs -f`                              |
| **Specific logs**      | `docker logs -f <service-name>`                                                                                                                   |
| **Interactive script** | `./setup.sh`                                                                                                                                      |

---

## 📦 File Sizes (Modular Approach)

- **docker-compose.shared.yml** (1.3 KB) — infrastructure only
- **docker-compose.moderation.yml** (2.1 KB) — LLM stack only
- **docker-compose.gateway.yml** (1.6 KB) — API gateway only
- **docker-compose.ssl.yml** (986 B) — SSL layer (optional)

**Total:** 5.6 KB (vs. 6+ KB for monolithic)

---

## 🔐 SSL Setup (30 seconds)

```bash
# Step 1: Generate cert
mkdir -p certs && openssl req -x509 -newkey rsa:4096 \
  -keyout certs/server.key -out certs/server.crt \
  -days 365 -nodes -subj "/CN=localhost"

# Step 2: Start with SSL
docker-compose -f docker-compose.shared.yml \
                -f docker-compose.moderation.yml \
                -f docker-compose.gateway.yml \
                -f docker-compose.ssl.yml \
                up --build
```

✅ HTTPS on port 443 (no code changes!)

---

## 🌍 Deployment Patterns

### Development (HTTP)

```bash
docker-compose -f docker-compose.shared.yml \
                -f docker-compose.moderation.yml \
                -f docker-compose.gateway.yml \
                up --build
```

- Simple, no SSL overhead
- All ports exposed for debugging

### Staging (HTTPS)

```bash
# Generate self-signed cert
openssl req -x509 -newkey rsa:4096 \
  -keyout certs/server.key -out certs/server.crt \
  -days 365 -nodes -subj "/CN=staging.example.com"

docker-compose -f docker-compose.shared.yml \
                -f docker-compose.moderation.yml \
                -f docker-compose.gateway.yml \
                -f docker-compose.ssl.yml \
                up --build -d
```

- Self-signed SSL
- Background mode

### Production (HTTPS + Real Cert)

```bash
# Use real certificate from CA/Let's Encrypt
cp /path/to/cert.pem certs/server.crt
cp /path/to/privkey.pem certs/server.key

docker-compose -f docker-compose.shared.yml \
                -f docker-compose.moderation.yml \
                -f docker-compose.gateway.yml \
                -f docker-compose.ssl.yml \
                up --build -d
```

- Real SSL certificate
- Rate limiting enabled
- Security headers active

---

## 🔗 What Changed

### New Files

- ✅ `docker-compose.shared.yml` — PostgreSQL, Redis, Kafka
- ✅ `docker-compose.moderation.yml` — Moderation + Ollama
- ✅ `docker-compose.gateway.yml` — Gateway + API
- ✅ `docker-compose.ssl.yml` — nginx SSL termination
- ✅ `deploy/nginx/nginx.conf` — nginx configuration
- ✅ `setup.sh` — Interactive setup script
- ✅ `SSL_AND_COMPOSE_GUIDE.md` — Complete guide
- ✅ `COMPOSE_STRUCTURE.md` — Architecture overview

### Code Changes

- **ZERO** code changes to Go services
- Environment variables still work via `.env`
- Volume mounts unchanged
- Health checks unchanged

### Backward Compatibility

- Old `docker-compose.yml` still exists (but not needed)
- Can delete after confirming new setup works

---

## 💡 Key Benefits

| Feature                      | Benefit                          |
| ---------------------------- | -------------------------------- |
| **Modular compose files**    | Flexible deployment combinations |
| **SSL without code changes** | nginx handles all TLS            |
| **Rate limiting**            | DDoS protection built-in         |
| **Security headers**         | HSTS, X-Frame-Options, etc.      |
| **HTTP redirect**            | Automatic upgrade to HTTPS       |
| **Interactive setup**        | Easy onboarding via `./setup.sh` |

---

## 🚨 Troubleshooting

### Services won't start?

```bash
docker-compose -f docker-compose.shared.yml \
                -f docker-compose.moderation.yml \
                -f docker-compose.gateway.yml \
                config  # Validates all files
```

### nginx SSL issues?

```bash
# Check cert exists
ls -la certs/server.crt certs/server.key

# View nginx logs
docker logs nginx-ssl

# Verify cert is valid
openssl x509 -in certs/server.crt -text -noout
```

### API key not working?

```bash
# Check gateway logs
docker logs gateway-service

# Verify admin endpoint accessible
curl -X GET http://localhost:8080/admin/keys \
  -H 'X-Admin-Secret: change-me-in-production'
```

---

## 📚 Documentation

- [SSL_AND_COMPOSE_GUIDE.md](SSL_AND_COMPOSE_GUIDE.md) — Complete guide (recommended read)
- [COMPOSE_STRUCTURE.md](COMPOSE_STRUCTURE.md) — Architecture & usage
- [deploy/nginx/nginx.conf](deploy/nginx/nginx.conf) — nginx SSL config
- `.env` — Environment variables

---

## ⚡ TL;DR

**Start development (HTTP):**

```bash
docker-compose -f docker-compose.shared.yml -f docker-compose.moderation.yml -f docker-compose.gateway.yml up --build
```

**Start production (HTTPS):**

```bash
mkdir -p certs && openssl req -x509 -newkey rsa:4096 \
  -keyout certs/server.key -out certs/server.crt -days 365 -nodes -subj "/CN=localhost"

docker-compose -f docker-compose.shared.yml -f docker-compose.moderation.yml \
                -f docker-compose.gateway.yml -f docker-compose.ssl.yml up --build
```

**Or just run:**

```bash
./setup.sh
```

✅ Done!
