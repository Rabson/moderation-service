.PHONY: help start start-http start-https start-moderation stop down logs logs-gateway logs-moderation logs-moderation-detail logs-ollama logs-nginx clean cert cert-check api-key test validate build compose-config ps status

# Color output
BLUE := \033[0;34m
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m # No Color

# Compose file flags
SHARED := -f docker-compose.shared.yml
MODERATION := -f docker-compose.moderation.yml
GATEWAY := -f docker-compose.gateway.yml
SSL := -f docker-compose.ssl.yml
SWAGGER := -f docker-compose.swagger.yml
BIN_DIR := bin

# Default target
.DEFAULT_GOAL := help

help: ## Show this help message
	@echo "$(BLUE)Moderation LLM - Make Commands$(NC)"
	@echo "================================"
	@echo ""
	@echo "$(YELLOW)🚀 Startup Commands:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '(start|up|build)' | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  make %-25s %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)🛑 Stop/Cleanup:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '(stop|down|clean|remove)' | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  make %-25s %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)📊 Logs & Status:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '(logs|ps|status)' | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  make %-25s %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)🔐 SSL/Security:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '(cert|ssl|key)' | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  make %-25s %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)🔑 API Management:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '(api|key)' | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  make %-25s %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)🧪 Development:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '(build|validate|test|config)' | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  make %-25s %s\n", $$1, $$2}'
	@echo ""

# ==================== STARTUP ====================

start: start-http ## Start all services (HTTP - default)

start-http: validate ## Start with HTTP (port 8080)
	@echo "$(GREEN)Starting all services (HTTP)...$(NC)"
	@echo "Gateway will be available at: http://localhost:8080"
	docker-compose $(SHARED) $(MODERATION) $(GATEWAY) up --build

start-https: cert validate ## Start with HTTPS (port 443, requires SSL cert)
	@echo "$(GREEN)Starting all services (HTTPS)...$(NC)"
	@echo "Gateway will be available at: https://localhost:443"
	docker-compose $(SHARED) $(MODERATION) $(GATEWAY) $(SSL) up --build

start-moderation: validate ## Start moderation stack only (no gateway)
	@echo "$(GREEN)Starting moderation stack (without gateway)...$(NC)"
	@echo "Moderation service available at: http://localhost:8081"
	docker-compose $(SHARED) $(MODERATION) up --build

start-daemon: start-http -d ## Start all services in background
	@echo "$(GREEN)Services started in background$(NC)"
	@sleep 2 && $(MAKE) ps

# ==================== STOP/DOWN ====================

stop: ## Stop all running services
	@echo "$(YELLOW)Stopping all services...$(NC)"
	docker-compose $(SHARED) $(MODERATION) $(GATEWAY) stop

down: ## Stop and remove all containers
	@echo "$(YELLOW)Stopping and removing containers...$(NC)"
	docker-compose $(SHARED) $(MODERATION) $(GATEWAY) down

down-ssl: ## Stop services including SSL proxy
	@echo "$(YELLOW)Stopping all services (including SSL)...$(NC)"
	docker-compose $(SHARED) $(MODERATION) $(GATEWAY) $(SSL) down

clean: down ## Clean: stop containers and remove volumes
	@echo "$(RED)Cleaning up: removing volumes...$(NC)"
	docker-compose $(SHARED) $(MODERATION) $(GATEWAY) down -v
	@echo "$(GREEN)✓ Cleanup complete$(NC)"

clean-all: clean ## Complete cleanup (same as 'clean')

remove-images: ## Remove all project images
	@echo "$(RED)Removing images...$(NC)"
	docker rmi moderation-llm-gateway-service moderation-llm-api-service moderation-llm-moderation-service -f 2>/dev/null || true
	@echo "$(GREEN)✓ Images removed$(NC)"

# ==================== LOGS & STATUS ====================

logs: ## View logs from all services
	@echo "$(BLUE)Showing logs from all services...$(NC)"
	docker-compose $(SHARED) $(MODERATION) $(GATEWAY) logs -f

logs-gateway: ## View gateway-service logs
	@echo "$(BLUE)Gateway logs:$(NC)"
	docker logs -f gateway-service

logs-moderation: ## View moderation-service logs (last 50 lines)
	@echo "$(BLUE)Moderation logs:$(NC)"
	docker logs -f moderation-service --tail 50

logs-moderation-detail: ## View moderation-service logs with full detail
	@echo "$(BLUE)Moderation logs (detailed):$(NC)"
	docker logs -f moderation-service --tail 100

logs-api: ## View api-service logs
	@echo "$(BLUE)API service logs:$(NC)"
	docker logs -f api-service

logs-ollama: ## View ollama logs
	@echo "$(BLUE)Ollama logs:$(NC)"
	docker logs -f ollama --tail 50

logs-redis: ## View redis logs
	@echo "$(BLUE)Redis logs:$(NC)"
	docker logs -f redis

logs-postgres: ## View postgres logs
	@echo "$(BLUE)Postgres logs:$(NC)"
	docker logs -f postgres

logs-nginx: ## View nginx SSL proxy logs
	@echo "$(BLUE)Nginx logs:$(NC)"
	docker logs -f nginx-ssl

logs-swagger: ## View swagger-ui logs
	@echo "$(BLUE)Swagger UI logs:$(NC)"
	docker logs -f swagger-ui

ps: ## Show running containers
	@echo "$(BLUE)Running containers:$(NC)"
	@docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" | grep -E 'NAMES|gateway|api|moderation|redis|postgres|ollama|nginx' || docker ps

status: ps ## Show service status (alias for ps)

# ==================== SSL/SECURITY ====================

cert: ## Generate self-signed SSL certificate
	@if [ -f "certs/server.crt" ] && [ -f "certs/server.key" ]; then \
		echo "$(YELLOW)SSL certificates already exist, skipping generation$(NC)"; \
	else \
		echo "$(GREEN)Generating self-signed SSL certificate...$(NC)"; \
		mkdir -p certs; \
		openssl req -x509 -newkey rsa:4096 \
			-keyout certs/server.key \
			-out certs/server.crt \
			-days 365 -nodes \
			-subj "/CN=localhost"; \
		echo "$(GREEN)✓ Certificate created: certs/server.crt$(NC)"; \
		ls -lh certs/; \
	fi

cert-check: ## Verify SSL certificate is valid
	@echo "$(BLUE)Checking SSL certificate...$(NC)"
	@if [ -f "certs/server.crt" ]; then \
		echo "$(GREEN)✓ Certificate exists$(NC)"; \
		openssl x509 -in certs/server.crt -text -noout | grep -E 'Subject:|Not Before|Not After'; \
		echo ""; \
		echo "Validity period:"; \
		openssl x509 -in certs/server.crt -noout -dates; \
	else \
		echo "$(RED)✗ Certificate not found at certs/server.crt$(NC)"; \
		echo "Run: make cert"; \
	fi

cert-regenerate: ## Regenerate SSL certificate (force)
	@echo "$(RED)Regenerating SSL certificate...$(NC)"
	@rm -f certs/server.crt certs/server.key
	@$(MAKE) cert

# ==================== API MANAGEMENT ====================

api-key: ## Create a new API key (interactive)
	@echo "$(BLUE)Creating API key...$(NC)"
	@read -p "Enter key name (e.g., 'dev-key'): " name; \
	read -p "Enter requests per minute limit (default: 100): " rpm; \
	rpm=$${rpm:-100}; \
	curl -X POST http://localhost:8080/admin/keys \
		-H 'X-Admin-Secret: change-me-in-production' \
		-H 'Content-Type: application/json' \
		-d "{\"name\":\"$$name\",\"requests_per_minute\":$$rpm}" | jq .

api-key-list: ## List all API keys
	@echo "$(BLUE)Listing API keys...$(NC)"
	@curl -s -X GET http://localhost:8080/admin/keys \
		-H 'X-Admin-Secret: change-me-in-production' | jq .

api-key-delete: ## Delete API key (requires key ID)
	@read -p "Enter API key ID to delete: " id; \
	curl -X DELETE http://localhost:8080/admin/keys/$$id \
		-H 'X-Admin-Secret: change-me-in-production' | jq .

api-health: ## Check API health
	@echo "$(BLUE)Checking API health...$(NC)"
	@curl -s http://localhost:8080/healthz | jq .

api-moderate: ## Test moderation endpoint (interactive)
	@read -p "Enter text to moderate: " text; \
	read -p "Enter API key: " key; \
	curl -X POST http://localhost:8080/moderate \
		-H "X-API-Key: $$key" \
		-H 'Content-Type: application/json' \
		-d "{\"text\":\"$$text\"}" | jq .

# ==================== DEVELOPMENT ====================

build: ## Build Go services only (no Docker)
	@echo "$(BLUE)Building Go services...$(NC)"
	@mkdir -p $(BIN_DIR)
	@cd gateway-service && go build -o ../$(BIN_DIR)/gateway-service ./cmd/gateway && echo "$(GREEN)✓ gateway-service -> $(BIN_DIR)/gateway-service$(NC)"
	@cd api-service && go build -o ../$(BIN_DIR)/api-service ./cmd/api && echo "$(GREEN)✓ api-service -> $(BIN_DIR)/api-service$(NC)"
	@cd moderation-service && go build -o ../$(BIN_DIR)/moderation-service ./cmd/moderation && echo "$(GREEN)✓ moderation-service -> $(BIN_DIR)/moderation-service$(NC)"

validate: ## Validate all docker-compose files
	@echo "$(BLUE)Validating docker-compose files...$(NC)"
	@docker-compose $(SHARED) $(MODERATION) $(GATEWAY) config > /dev/null && echo "$(GREEN)✓ HTTP compose valid$(NC)" || echo "$(RED)✗ HTTP compose invalid$(NC)"
	@docker-compose $(SHARED) $(MODERATION) $(GATEWAY) $(SSL) config > /dev/null && echo "$(GREEN)✓ HTTPS compose valid$(NC)" || echo "$(RED)✗ HTTPS compose invalid$(NC)"

compose-config: ## Show merged compose configuration (HTTP)
	@echo "$(BLUE)HTTP Compose Configuration:$(NC)"
	@docker-compose $(SHARED) $(MODERATION) $(GATEWAY) config

compose-config-https: ## Show merged compose configuration (HTTPS)
	@echo "$(BLUE)HTTPS Compose Configuration:$(NC)"
	@docker-compose $(SHARED) $(MODERATION) $(GATEWAY) $(SSL) config

docs-up: ## Start Swagger UI on http://localhost:8088
	@echo "$(GREEN)Starting Swagger UI...$(NC)"
	docker-compose $(SWAGGER) up -d
	@echo "Swagger UI: http://localhost:8088"

docs-down: ## Stop Swagger UI
	@echo "$(YELLOW)Stopping Swagger UI...$(NC)"
	docker-compose $(SWAGGER) down

docs-openapi: ## Print OpenAPI spec file path
	@echo "OpenAPI spec: deploy/swagger/openapi.yaml"

test: ## Run health checks on all services
	@echo "$(BLUE)Running health checks...$(NC)"
	@echo ""
	@echo "$(YELLOW)Gateway:$(NC)"
	@curl -s http://localhost:8080/healthz | jq . || echo "$(RED)✗ Gateway not responding$(NC)"
	@echo ""
	@echo "$(YELLOW)Moderation:$(NC)"
	@curl -s http://localhost:8081/healthz | jq . || echo "$(RED)✗ Moderation not responding$(NC)"
	@echo ""
	@echo "$(YELLOW)PostgreSQL:$(NC)"
	@docker exec postgres pg_isready -U moderation && echo "$(GREEN)✓ Connected$(NC)" || echo "$(RED)✗ Not responding$(NC)"
	@echo ""
	@echo "$(YELLOW)Redis:$(NC)"
	@docker exec redis redis-cli ping || echo "$(RED)✗ Not responding$(NC)"
	@echo ""
	@echo "$(YELLOW)Ollama:$(NC)"
	@curl -s http://localhost:11434/api/tags | jq . || echo "$(RED)✗ Not responding$(NC)"

test-moderation: ## Test moderation endpoint with sample data
	@echo "$(BLUE)Testing moderation endpoint...$(NC)"
	@read -p "Enter API key (or press Enter for default test): " key; \
	key=$${key:-test-key}; \
	echo "Sending: 'I will k1ll you'"; \
	curl -X POST http://localhost:8080/moderate \
		-H "X-API-Key: $$key" \
		-H 'Content-Type: application/json' \
		-d '{"text":"I will k1ll you"}' | jq .

test-batch: ## Test batch moderation endpoint
	@echo "$(BLUE)Testing batch moderation endpoint...$(NC)"
	@read -p "Enter API key: " key; \
	curl -X POST http://localhost:8080/moderate/batch \
		-H "X-API-Key: $$key" \
		-H 'Content-Type: application/json' \
		-d '{"texts":["hello","I will k1ll you","nice weather"]}' | jq .

# ==================== UTILITY ====================

version: ## Show versions of key components
	@echo "$(BLUE)Component Versions:$(NC)"
	@echo "Docker: $$(docker --version)"
	@echo "Docker Compose: $$(docker-compose --version)"
	@echo "Go: $$(go version 2>/dev/null || echo 'not installed')"
	@echo "OpenSSL: $$(openssl version)"

info: ## Show project information
	@echo "$(BLUE)Moderation LLM Project$(NC)"
	@echo "======================"
	@echo ""
	@echo "$(YELLOW)Services:$(NC)"
	@echo "  - gateway-service (port 8080)"
	@echo "  - api-service (port 8080, internal)"
	@echo "  - moderation-service (port 8081)"
	@echo "  - ollama (port 11434)"
	@echo "  - postgresql (port 5432)"
	@echo "  - redis (port 6379)"
	@echo ""
	@echo "$(YELLOW)Files:$(NC)"
	@echo "  - .env: Environment variables"
	@echo "  - certs/: SSL certificates"
	@echo "  - docker-compose.*.yml: Modular compose files"
	@echo "  - deploy/nginx/nginx.conf: SSL proxy config"
	@echo ""

setup: ## Run interactive setup (alias for ./setup.sh)
	@./setup.sh

# ==================== QUICK ALIASES ====================

up: start ## Alias: start services (HTTP)
up-https: start-https ## Alias: start services (HTTPS)
up-moderation: start-moderation ## Alias: start moderation only
restart: stop start ## Restart all services
rebuild: clean build start ## Clean build and start

dev: ## Development mode: build + start HTTP
	@$(MAKE) build
	@$(MAKE) start-http

prod: cert ## Production mode: generate cert + validate
	@$(MAKE) validate
	@echo "$(GREEN)✓ Ready for production deployment$(NC)"
	@echo "Run: make start-https"

.PHONY: all
all: help
