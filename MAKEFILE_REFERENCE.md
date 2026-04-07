# Makefile Commands Reference

Complete list of all available `make` commands for managing the moderation LLM platform.

---

## 🚀 Quick Start

```bash
# Start in development (HTTP)
make start

# Start in production (HTTPS)
make start-https

# Or use the interactive setup
make setup
```

---

## 📋 All Available Commands

### 🎯 Startup Commands

| Command                 | Description                                |
| ----------------------- | ------------------------------------------ |
| `make start`            | Start all services (HTTP) - **DEFAULT**    |
| `make start-http`       | Explicitly start with HTTP (port 8080)     |
| `make start-https`      | Start with HTTPS (port 443, requires cert) |
| `make start-moderation` | Start moderation stack only (no gateway)   |
| `make start-daemon`     | Start services in background mode          |
| `make up`               | Alias for `start`                          |
| `make up-https`         | Alias for `start-https`                    |
| `make up-moderation`    | Alias for `start-moderation`               |
| `make dev`              | Development mode: build Go + start HTTP    |
| `make prod`             | Production mode: generate cert + validate  |

### 🛑 Stop & Cleanup

| Command              | Description                               |
| -------------------- | ----------------------------------------- |
| `make stop`          | Stop all running services                 |
| `make down`          | Stop and remove containers                |
| `make down-ssl`      | Stop services including SSL proxy         |
| `make clean`         | Clean: stop containers and remove volumes |
| `make clean-all`     | Alias for `clean`                         |
| `make remove-images` | Remove all project Docker images          |
| `make restart`       | Restart all services                      |
| `make rebuild`       | Clean build and start again               |

### 📊 Logs & Monitoring

| Command                       | Description                                |
| ----------------------------- | ------------------------------------------ |
| `make logs`                   | View logs from all services (follow)       |
| `make logs-gateway`           | Gateway-service logs                       |
| `make logs-moderation`        | Moderation-service logs (last 50 lines)    |
| `make logs-moderation-detail` | Moderation-service full detail (100 lines) |
| `make logs-api`               | API-service logs                           |
| `make logs-ollama`            | Ollama LLM logs                            |
| `make logs-redis`             | Redis logs                                 |
| `make logs-postgres`          | PostgreSQL logs                            |
| `make logs-nginx`             | Nginx SSL proxy logs                       |
| `make ps`                     | Show running containers                    |
| `make status`                 | Alias for `ps`                             |

### 🔐 SSL/Security

| Command                | Description                          |
| ---------------------- | ------------------------------------ |
| `make cert`            | Generate self-signed SSL certificate |
| `make cert-check`      | Verify SSL certificate validity      |
| `make cert-regenerate` | Force regenerate SSL certificate     |

### 🔑 API Management

| Command               | Description                            |
| --------------------- | -------------------------------------- |
| `make api-key`        | Create new API key (interactive)       |
| `make api-key-list`   | List all API keys                      |
| `make api-key-delete` | Delete API key (interactive)           |
| `make api-health`     | Check API health endpoint              |
| `make api-moderate`   | Test moderation endpoint (interactive) |

### 🧪 Development & Testing

| Command                     | Description                          |
| --------------------------- | ------------------------------------ |
| `make build`                | Build Go services (no Docker)        |
| `make validate`             | Validate all docker-compose files    |
| `make compose-config`       | Show merged HTTP compose config      |
| `make compose-config-https` | Show merged HTTPS compose config     |
| `make test`                 | Run health checks on all services    |
| `make test-moderation`      | Test moderation endpoint with sample |
| `make test-batch`           | Test batch moderation endpoint       |

### ℹ️ Information

| Command        | Description                          |
| -------------- | ------------------------------------ |
| `make help`    | Show this help message (recommended) |
| `make version` | Show Docker/Compose/Go versions      |
| `make info`    | Show project information             |
| `make setup`   | Run interactive setup script         |

---

## 💡 Common Workflows

### Development Workflow

```bash
# Build and start in development
make dev

# View logs
make logs-moderation

# Test API
make api-health

# Create API key
make api-key

# Test moderation
make test-moderation

# Stop when done
make stop
```

### Production Deployment

```bash
# Generate SSL certificate
make cert

# Validate configuration
make validate

# Start with HTTPS
make start-https

# Monitor logs
make logs-nginx
make logs-gateway

# Manage API keys
make api-key-list
make api-key
```

### Testing & Debugging

```bash
# Run all health checks
make test

# View specific service logs
make logs-gateway
make logs-moderation
make logs-ollama

# Check compose configuration
make compose-config

# Test moderation endpoint
make test-moderation
make test-batch
```

### Cleanup & Restart

```bash
# Stop services gracefully
make stop

# Full restart
make restart

# Clean everything
make clean

# Rebuild from scratch
make rebuild
```

---

## 🎯 Usage Examples

### Create API Key

```bash
make api-key

# Interactive prompt:
# Enter key name: my-test-key
# Enter requests per minute: 100

# Returns:
# {
#   "api_key": "abcd1234...",
#   "name": "my-test-key",
#   "requests_per_minute": 100
# }
```

### Test Moderation

```bash
make test-moderation

# Interactive prompt:
# Enter text to moderate: I will hurt you
# Enter API key: <from above>

# Returns:
# {
#   "action": "block",
#   "score": 0.92,
#   ...
# }
```

### View Logs

```bash
# All services
make logs

# Specific service
make logs-gateway
make logs-moderation
make logs-ollama
```

### Monitor Status

```bash
make ps
# NAMES              STATUS                 PORTS
# gateway-service    Up 2 minutes           8080→8080
# api-service        Up 2 minutes
# moderation-service Up 2 minutes           8081→8081
# ollama             Up 3 minutes           11434→11434
# postgres           Up 3 minutes           5432→5432
# redis              Up 3 minutes           6379→6379
```

---

## 🔧 Advanced Usage

### View Merged Compose

```bash
# HTTP configuration
make compose-config | head -50

# HTTPS configuration
make compose-config-https | head -50
```

### Validate Environment

```bash
# Check all files
make validate

# Expected output:
# ✓ HTTP compose valid
# ✓ HTTPS compose valid
```

### Check Versions

```bash
make version

# Shows:
# Docker: Docker version 24.x.x
# Docker Compose: Docker Compose version 2.x.x
# Go: go version 1.22
# OpenSSL: OpenSSL 3.x.x
```

### Performance Test

```bash
# Run batch moderation test
make test-batch

# Test 3 items in one request
```

---

## 📝 Notes

1. **Colors in Output**: Commands use colored output for readability
   - 🔵 Blue = Informational
   - 🟢 Green = Success
   - 🟡 Yellow = Warning/Notice
   - 🔴 Red = Error

2. **Interactive Commands**: Some commands prompt for user input
   - `make api-key` - Enter key name and RPM
   - `make api-key-delete` - Enter key ID
   - `make api-moderate` - Enter text and API key

3. **Environment**: All commands respect variables from `.env` file

4. **Compose Files**: Makefile handles combining multiple compose files automatically
   - HTTP: `shared` + `moderation` + `gateway`
   - HTTPS: HTTP + `ssl`

5. **Help System**: Run `make help` for formatted command list

---

## ⚠️ Important Safety Notes

- `make clean` and `make remove-images` destructive operations
- Always backup data before running cleanup commands
- Use `make stop` to gracefully stop services before cleanup
- SSL certificates are self-signed by default (not for production)

---

## 🚨 Troubleshooting

### "make: command not found"

```bash
# Install make (macOS)
brew install make

# Install make (Linux)
sudo apt-get install build-essential
```

### Makefile colors not working

```bash
# Use NO_COLOR environment variable
NO_COLOR=1 make logs
```

### One of the compose files invalid

```bash
# Validate individually
make validate

# Check specific file
docker-compose -f docker-compose.shared.yml config
```

### API key endpoint not accessible

```bash
# Verify services running
make ps

# Check gateway logs
make logs-gateway

# Test basic health
make api-health
```

---

**For more information:** See [SSL_AND_COMPOSE_GUIDE.md](SSL_AND_COMPOSE_GUIDE.md) or [QUICK_REFERENCE.md](QUICK_REFERENCE.md)
