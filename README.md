# moderation-llm

Production-ready moderation and language processing stack with:

- gateway-service (API key auth, rate limiting, admin key management)
- api-service (internal proxy layer)
- moderation-service (moderation, text transcription normalization, translation, audio transcription)
- configurable LLM provider for text tasks (Ollama, OpenAI, Google GenAI)
- PostgreSQL and Redis with host persistence
- optional Kafka profile
- Swagger docs and separate Admin UI

## Architecture

Client -> Gateway -> API Service -> Moderation Service -> LLM Provider (Ollama/OpenAI/Google GenAI)

Support services:

- PostgreSQL (audit logs + api_keys)
- Redis (cache, key validation cache, rate-limit counters)
- Kafka + Zookeeper (optional)

## Project Layout

```text
moderation-llm/
├── compose/
│   ├── docker-compose.shared.yml
│   ├── docker-compose.moderation.yml
│   ├── docker-compose.gateway.yml
│   ├── docker-compose.ssl.yml
│   ├── docker-compose.swagger.yml
│   ├── docker-compose.admin-ui.yml
│   └── docker-compose.yml
├── admin-ui/
│   └── index.html
├── gateway-service/
├── api-service/
├── moderation-service/
├── deploy/
│   ├── nginx/
│   ├── postgres/init/
│   └── swagger/
├── data/
│   ├── postgres/
│   ├── redis/
│   ├── zookeeper/
│   └── kafka/
├── Makefile
├── .env
└── .env.example
```

## Main Endpoints

Gateway base URL: `http://localhost:8080`

- `GET /healthz` (public)
- `POST /moderate` (requires `X-API-Key`)
- `POST /moderate/batch` (requires `X-API-Key`)
- `POST /transcribe` (requires `X-API-Key`)
- `POST /transcribe/audio` (requires `X-API-Key`)
- `POST /translate` (requires `X-API-Key`)
- `POST /admin/keys` (requires `X-Admin-Secret`)
- `GET /admin/keys` (requires `X-Admin-Secret`)
- `DELETE /admin/keys/{id}` (requires `X-Admin-Secret`)

## Quick Start

1. Copy env file:

```bash
cp .env.example .env
```

2. Start core stack in detached mode:

```bash
make up-core-daemon
```

3. Check health:

```bash
make test
```

## Common Make Commands

- `make up-core-daemon` start shared + moderation + gateway stack with build
- `make start` start stack in foreground
- `make stop` stop running containers
- `make down` stop and remove containers
- `make validate` validate compose configurations
- `make logs` stream logs from core services
- `make docs-up` start Swagger UI
- `make docs-down` stop Swagger UI
- `make admin-ui-up` start separate Admin UI
- `make admin-ui-down` stop separate Admin UI

## Swagger

- OpenAPI template: `deploy/swagger/openapi.template.yaml`
- Generated spec in container flow: `deploy/swagger/openapi.yaml`
- Swagger UI: `http://localhost:${SWAGGER_UI_PORT}` (default 8088)

Swagger server target is editable from UI via server variables (`scheme`, `host`, `port`) and defaults from `.env`:

- `SWAGGER_SERVER_SCHEME`
- `SWAGGER_SERVER_HOST`
- `SWAGGER_SERVER_PORT`

## Admin UI (Separate)

Run:

```bash
make admin-ui-up
```

Open:

```text
http://localhost:${ADMIN_UI_PORT}
```

Use it to:

- test gateway connectivity
- list keys
- create keys
- deactivate keys

## Resource and Machine Notes

- Resource limits are defined per service in compose and are env-configurable.
- For low-resource machines, `.env` includes a conservative profile suitable for about 4GB RAM / 2 CPU.

Important knobs:

- `LLM_MODERATE_PROVIDER`, `LLM_MODERATE_MODEL`, `LLM_MODERATE_BASE_URL`, `LLM_MODERATE_API_KEY`
- `LLM_TRANSCRIBE_PROVIDER`, `LLM_TRANSCRIBE_MODEL`, `LLM_TRANSCRIBE_BASE_URL`, `LLM_TRANSCRIBE_API_KEY`
- `LLM_TRANSLATE_PROVIDER`, `LLM_TRANSLATE_MODEL`, `LLM_TRANSLATE_BASE_URL`, `LLM_TRANSLATE_API_KEY`
- `LLM_TRANSCRIBE_AUDIO_PROVIDER`, `LLM_TRANSCRIBE_AUDIO_MODEL`, `LLM_TRANSCRIBE_AUDIO_BASE_URL`, `LLM_TRANSCRIBE_AUDIO_API_KEY`
- `OPENAI_API_KEY`, `GOOGLE_API_KEY`
- `OLLAMA_CPUS`, `OLLAMA_MEM_LIMIT`, `OLLAMA_MEM_RESERVATION`
- `MODERATION_CPUS`, `MODERATION_MEM_LIMIT`
- `GATEWAY_CPUS`, `API_CPUS`
- `KAFKA_*` (only when Kafka profile is enabled)

## Risk Scoring Behavior

`risk_score` uses the higher of:

- weighted score (`hate*0.4 + violence*0.3 + sexual*0.2 + spam*0.1`)
- dominant unsafe label (`max(hate, violence, sexual, spam)`)

Action thresholds:

- `< 0.3` => `allow`
- `0.3 - 0.7` => `review`
- `> 0.7` => `block`

## Data Persistence on Host

Host-backed data directories:

- `data/postgres`
- `data/redis`
- `data/zookeeper`
- `data/kafka`

These are ignored by git.

## Optional Stacks

Start SSL layer (nginx):

```bash
docker-compose -f compose/docker-compose.shared.yml \
  -f compose/docker-compose.moderation.yml \
  -f compose/docker-compose.gateway.yml \
  -f compose/docker-compose.ssl.yml up --build -d
```

Start Kafka profile:

```bash
docker-compose -f compose/docker-compose.shared.yml --profile kafka up -d
```

## LLM Provider Configuration

Configure providers/models per endpoint type:

```env
# moderate
LLM_MODERATE_PROVIDER=ollama
LLM_MODERATE_MODEL=gemma:2b
LLM_MODERATE_BASE_URL=http://ollama:11434
LLM_MODERATE_API_KEY=

# transcribe (text normalization)
LLM_TRANSCRIBE_PROVIDER=ollama
LLM_TRANSCRIBE_MODEL=gemma:2b
LLM_TRANSCRIBE_BASE_URL=http://ollama:11434
LLM_TRANSCRIBE_API_KEY=

# translate
LLM_TRANSLATE_PROVIDER=ollama
LLM_TRANSLATE_MODEL=gemma:2b
LLM_TRANSLATE_BASE_URL=http://ollama:11434
LLM_TRANSLATE_API_KEY=

# transcribe audio
LLM_TRANSCRIBE_AUDIO_PROVIDER=openai
LLM_TRANSCRIBE_AUDIO_MODEL=gpt-4o-mini-transcribe
LLM_TRANSCRIBE_AUDIO_BASE_URL=https://api.openai.com/v1
LLM_TRANSCRIBE_AUDIO_API_KEY=
```

You can also keep provider-specific keys in:

- `OPENAI_API_KEY`
- `GOOGLE_API_KEY`

If a task-specific `*_API_KEY` is empty, moderation-service falls back to provider-specific keys (`OPENAI_API_KEY` or `GOOGLE_API_KEY`).

These let you use different models/providers per endpoint type (`/moderate`, `/transcribe`, `/transcribe/audio`).

## Troubleshooting

- If browser calls fail from Swagger/Admin UI, ensure `CORS_ALLOWED_ORIGINS` includes the UI origin(s).
- If SSL stack fails on ports 80/443, another process is using those ports.
- If audio transcription returns provider errors, set `LLM_TRANSCRIBE_AUDIO_PROVIDER`, `LLM_TRANSCRIBE_AUDIO_API_KEY`, and `OPENAI_API_KEY` as needed.
