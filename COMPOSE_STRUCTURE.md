#!/bin/bash

# Separated Docker Compose Setup

## Usage

### Option 1: Standard HTTP (All services)

```bash
docker-compose -f docker-compose.shared.yml \
                -f docker-compose.moderation.yml \
                -f docker-compose.gateway.yml \
                up --build
```

### Option 2: With SSL/HTTPS (nginx reverse proxy)

First, generate self-signed certificate:

```bash
mkdir -p certs
openssl req -x509 -newkey rsa:4096 \
  -keyout certs/server.key \
  -out certs/server.crt \
  -days 365 -nodes \
  -subj "/CN=localhost"
```

Then run with SSL:

```bash
docker-compose -f docker-compose.shared.yml \
                -f docker-compose.moderation.yml \
                -f docker-compose.gateway.yml \
                -f docker-compose.ssl.yml \
                up --build
```

### Option 3: Moderation + Shared only (no gateway)

```bash
docker-compose -f docker-compose.shared.yml \
                -f docker-compose.moderation.yml \
                up --build
```

### Option 4: Production CA certificate (Let's Encrypt)

```bash
# Using certbot to generate real certs
certbot certonly --standalone -d yourdomain.com
# Copy to certs/
cp /etc/letsencrypt/live/yourdomain.com/fullchain.pem certs/server.crt
cp /etc/letsencrypt/live/yourdomain.com/privkey.pem certs/server.key
```

## File Structure

| File                          | Purpose                                          |
| ----------------------------- | ------------------------------------------------ |
| docker-compose.shared.yml     | PostgreSQL, Redis, Kafka (shared infrastructure) |
| docker-compose.moderation.yml | Moderation service, Ollama LLM, model init       |
| docker-compose.gateway.yml    | API Gateway, internal API service                |
| docker-compose.ssl.yml        | OPTIONAL: nginx SSL termination proxy            |

## SSL Without Code Changes

The nginx reverse proxy (docker-compose.ssl.yml) handles:

- HTTPS termination
- HTTP → HTTPS redirect (port 80 → 443)
- SSL/TLS v1.2 + v1.3
- Security headers (HSTS, X-Frame-Options, etc.)
- Rate limiting by API key
- Long timeouts for LLM processing

**Gateway service code unchanged** — nginx handles all SSL/TLS encryption.

## Network Ports

| Service            | Port  | Notes                       |
| ------------------ | ----- | --------------------------- |
| nginx (HTTP)       | 80    | Only redirects to HTTPS     |
| nginx (HTTPS)      | 443   | Public entry point with SSL |
| gateway-service    | 8080  | Internal, behind nginx      |
| api-service        | 8080  | Internal, no public port    |
| moderation-service | 8081  | Internal, no public port    |
| ollama             | 11434 | Internal (optional public)  |
| postgres           | 5432  | Internal with health checks |
| redis              | 6379  | Internal with health checks |

## API Examples

### With HTTP (default):

```bash
curl -X POST http://localhost:8080/moderate \
  -H 'X-API-Key: <key>' \
  -H 'Content-Type: application/json' \
  -d '{"text":"I will k1ll you"}'
```

### With HTTPS (nginx SSL):

```bash
curl -X POST https://localhost:443/moderate \
  --cacert certs/server.crt \
  -H 'X-API-Key: <key>' \
  -H 'Content-Type: application/json' \
  -d '{"text":"I will k1ll you"}'
```

## Stopping Services

```bash
# Stop all (same -f flags as up)
docker-compose -f docker-compose.shared.yml \
                -f docker-compose.moderation.yml \
                -f docker-compose.gateway.yml \
                down

# Remove volumes too (careful!)
docker-compose -f docker-compose.shared.yml \
                -f docker-compose.moderation.yml \
                -f docker-compose.gateway.yml \
                down -v
```

## Monitoring

```bash
# View all services
docker ps

# Logs for specific service
docker logs -f gateway-service
docker logs -f moderation-service

# With SSL proxy
docker logs -f nginx-ssl
```
