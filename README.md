# moderation-llm

Production-ready content moderation stack using local Gemma inference (Ollama), Go services, Redis cache, PostgreSQL audit logs, and optional Kafka event publishing.

## Project Structure

```text
moderation-llm/
├── api-service/
│   ├── cmd/api/main.go
│   ├── internal/config/config.go
│   ├── internal/gateway/ratelimit.go
│   ├── internal/gateway/server.go
│   ├── Dockerfile
│   └── go.mod
├── moderation-service/
│   ├── cmd/moderation/main.go
│   ├── internal/config/config.go
│   ├── internal/http/handlers.go
│   ├── internal/http/server.go
│   ├── internal/kafka/producer.go
│   ├── internal/llm/client.go
│   ├── internal/moderation/engine.go
│   ├── internal/moderation/preprocess.go
│   ├── internal/moderation/rules.go
│   ├── internal/moderation/types.go
│   ├── internal/storage/cache.go
│   ├── internal/storage/postgres.go
│   ├── Dockerfile
│   └── go.mod
├── deploy/postgres/init/001_init.sql
├── .env.example
├── .gitignore
├── docker-compose.yml
└── README.md
```

## Features

- API Gateway (Go): `/moderate`, `/moderate/batch`, health, rate limiting.
- Moderation Service (Go):
  - Preprocessing: lowercase, symbol removal, leetspeak normalization.
  - Rule-based scoring (regex + keywords).
  - LLM classification via Ollama Gemma.
  - Weighted risk scoring and action decision (`allow`, `review`, `block`).
  - Redis result cache.
  - PostgreSQL audit log persistence.
  - Graceful fallback to rule-based scores if LLM fails/timeouts.
- Optional Kafka event emission via env toggle.

## API Contract

`POST /moderate`

Request:

```json
{
  "text": "..."
}
```

Response:

```json
{
  "labels": {
    "hate": 0.0,
    "violence": 0.0,
    "sexual": 0.0,
    "spam": 0.0,
    "safe": 1.0
  },
  "risk_score": 0.0,
  "action": "allow"
}
```

Batch endpoint (bonus): `POST /moderate/batch`

Request:

```json
{
  "texts": ["text one", "text two"]
}
```

Response:

```json
{
  "results": [
    {
      "labels": {
        "hate": 0.0,
        "violence": 0.0,
        "sexual": 0.0,
        "spam": 0.0,
        "safe": 1.0
      },
      "risk_score": 0.0,
      "action": "allow"
    }
  ]
}
```

## Risk Scoring

```text
risk_score =
  hate * 0.4 +
  violence * 0.3 +
  sexual * 0.2 +
  spam * 0.1
```

Thresholds:

- `< 0.3` -> `allow`
- `0.3 - 0.7` -> `review`
- `> 0.7` -> `block`

## Setup

1. Copy environment file:

```bash
cp .env.example .env
```

2. Start stack:

```bash
docker compose up --build
```

3. Start stack with Kafka profile (optional):

```bash
docker compose --profile kafka up --build
```

## Example Requests

Moderate single text through gateway:

```bash
curl -sS -X POST http://localhost:8080/moderate \
  -H 'Content-Type: application/json' \
  -d '{"text":"h@te speech and click here for free money"}' | jq
```

Moderate batch:

```bash
curl -sS -X POST http://localhost:8080/moderate/batch \
  -H 'Content-Type: application/json' \
  -d '{"texts":["hello world","I will k1ll you"]}' | jq
```

Health checks:

```bash
curl -sS http://localhost:8080/healthz
curl -sS http://localhost:8081/healthz
```

## Environment Variables

Use `.env.example` as source of truth. Important values:

- `OLLAMA_MODEL` (default `gemma:2b`)
- `MODERATION_SERVICE_URL`
- `LLM_TIMEOUT`
- `CACHE_TTL`
- `POSTGRES_DSN`
- `KAFKA_ENABLED`
- `KAFKA_BROKERS`
- `KAFKA_TOPIC`

## Swagger Docs

OpenAPI spec is available at:

- `deploy/swagger/openapi.yaml`

Run Swagger UI locally:

```bash
docker-compose -f docker-compose.swagger.yml up -d
```

Then open:

- `http://localhost:8088`

Make shortcuts:

```bash
make docs-up
make docs-down
```

## Run everything

```

docker-compose -f docker-compose.shared.yml -f docker-compose.moderation.yml -f docker-compose.gateway.yml up --build -d
```
