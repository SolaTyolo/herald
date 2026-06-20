# Herald

Novu-inspired notification infrastructure in Go. Channel delivery uses **Traefik-style YAML routing**: builtin **SMTP** and **in-app**, plus **user Delivery Workers** for other channels.

**中文文档:** [README.zh-CN.md](README.zh-CN.md)

**Architecture:** [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) · **Providers:** [docs/PROVIDERS.md](docs/PROVIDERS.md)  
**Cursor Agent:** [AGENTS.md](AGENTS.md) · [docs/CURSOR.md](docs/CURSOR.md)

## Features

- Workflow engine: **email, sms, push, chat, in-app** + delay / digest / throttle
- Subscribers, topics, preferences, integrations
- Builtin `smtp_email` + `inapp`; remote providers via user HTTP workers (`sdk/worker`)
- Async delivery results over Redis (default); Topic fan-out via Asynq
- Storage: **file** (JSON) or **db** (PostgreSQL / MySQL / SQLite via GORM)
- OpenTelemetry traces & metrics

## Quick start

```bash
cp .env.example .env
make docker-up
# Edit config/providers.yaml (builtin smtp_email + inapp by default)
DEV_MODE=true make build

# terminal 1
DEV_MODE=true make run-api

# terminal 2
make run-worker
```

API and Worker must share the same `PROVIDERS_CONFIG` and `REDIS_ADDR`.

```
Authorization: Bearer <key>
```

## Example trigger

```bash
curl -X POST http://localhost:8080/v1/events/trigger \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "welcome",
    "to": {"subscriberId": "user-1"},
    "payload": {"name": "Alice"}
  }'
```

## Layout

- `cmd/api` — REST API
- `cmd/worker` — Asynq job processor
- `config/providers.yaml` — Traefik-style routing
- `pkg/contract` — delivery contract
- `sdk/worker` — user Delivery Worker SDK

## Environment

| Variable | Default | Description |
|----------|---------|-------------|
| `PROVIDERS_CONFIG` | `./config/providers.yaml` | YAML routing (hot-reload) |
| `DELIVERY_RESULT_MODE` | `async` | `async` Redis queue / `sync` HTTP body |
| `STORE_TYPE` | `db` | `file` or `db` |
| `DB_DRIVER` | `postgres` | `postgres`, `mysql`, `sqlite` |
| `REDIS_ADDR` | `localhost:6379` | Asynq + bridge + result queue |
| `ENCRYPTION_KEY` | — | 32-byte key (required in production) |
| `DEV_MODE` | `false` | Local dev defaults |
| `HTTP_ADDR` | `:8080` | API listen address |

Full list: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md#9-环境变量) · [`.env.example`](.env.example).

## Tests

```bash
make test
```

## License

Apache License 2.0 — see [LICENSE](LICENSE).
