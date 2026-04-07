#!/bin/bash
# Quick setup script for separated Docker Compose

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "🔧 Moderation LLM - Docker Compose Setup"
echo "========================================"
echo ""

# Show menu
echo "Select setup option:"
echo "1) HTTP only (gateway on port 8080)"
echo "2) HTTPS (nginx SSL on port 443)"
echo "3) Moderation + Shared only (no gateway)"
echo "4) View available services"
echo ""

read -p "Enter choice (1-4): " choice

case $choice in
    1)
        echo ""
        echo "Starting HTTP setup..."
        echo "✓ Services will be available at:"
        echo "  - Gateway:     http://localhost:8080"
        echo "  - Moderation:  http://localhost:8081"
        echo "  - Ollama:      http://localhost:11434"
        echo "  - PostgreSQL:  localhost:5432"
        echo "  - Redis:       localhost:6379"
        echo ""
        docker-compose -f compose/docker-compose.shared.yml \
                -f compose/docker-compose.moderation.yml \
                -f compose/docker-compose.gateway.yml \
                        up --build
        ;;
    2)
        echo ""
        echo "Starting HTTPS setup..."
        
        # Check if certs exist
        if [ ! -f "certs/server.crt" ] || [ ! -f "certs/server.key" ]; then
            echo "⚠️  SSL certificates not found. Generating self-signed cert..."
            mkdir -p certs
            openssl req -x509 -newkey rsa:4096 \
              -keyout certs/server.key \
              -out certs/server.crt \
              -days 365 -nodes \
              -subj "/CN=localhost"
            echo "✓ Self-signed certificate created"
            echo "  ⚠️  For production: use real CA certificates"
        fi
        
        echo ""
        echo "✓ Services will be available at:"
        echo "  - HTTPS Gateway:  https://localhost:443"
        echo "  - HTTP redirect:  http://localhost:80"
        echo "  - Moderation:     http://localhost:8081 (internal)"
        echo "  - Ollama:         http://localhost:11434 (internal)"
        echo ""
        
        docker-compose -f compose/docker-compose.shared.yml \
                -f compose/docker-compose.moderation.yml \
                -f compose/docker-compose.gateway.yml \
                -f compose/docker-compose.ssl.yml \
                        up --build
        ;;
    3)
        echo ""
        echo "Starting Moderation + Shared (no gateway)..."
        docker-compose -f compose/docker-compose.shared.yml \
                -f compose/docker-compose.moderation.yml \
                        up --build
        ;;
    4)
        echo ""
        echo "📋 Available services:"
        echo ""
        echo "Shared Infrastructure:"
        docker-compose -f compose/docker-compose.shared.yml config --services | sed 's/^/  - /'
        echo ""
        echo "Moderation Stack:"
        docker-compose -f compose/docker-compose.moderation.yml config --services | sed 's/^/  - /'
        echo ""
        echo "Gateway & API:"
        docker-compose -f compose/docker-compose.gateway.yml config --services | sed 's/^/  - /'
        echo ""
        echo "Optional SSL (nginx):"
        docker-compose -f compose/docker-compose.ssl.yml config --services | sed 's/^/  - /'
        ;;
    *)
        echo "Invalid choice"
        exit 1
        ;;
esac
