# Herald

Novu-inspired notification infrastructure in Go. Channel delivery uses **Traefik-style YAML routing**: builtin **SMTP** and **in-app**, plus **user Delivery Workers** for other channels.

**中文文档:** [README.zh-CN.md](README.zh-CN.md)

**Architecture:** [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) · **Providers:** [docs/PROVIDERS.md](docs/PROVIDERS.md)  
**Cursor Agent:** [AGENTS.md](AGENTS.md) · [docs/CURSOR.md](docs/CURSOR.md)

## Features

- Workflow engine: **email, sms, push, chat, inapp** + delay / digest / throttle
- **Dynamic templates**: versions with one active version; workflow steps reference `templateId` and render trigger payload
- Subscribers, topics, preferences
- Builtin `smtp_email` (env `SMTP_*`) + `inapp`; remote providers via user HTTP workers (`POST /delivery`)
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

`QUEUE_MODE=asynq` (default): API and Worker share `PROVIDERS_CONFIG` and `REDIS_ADDR`. `QUEUE_MODE=local` runs jobs in the API process and does not use Redis.

Manual e2e UI (separate from the API):

```bash
make playground   # http://localhost:8090 — see e2e/README.md
```

Set `API_KEY` in `.env` (required). Callers send:

```
Authorization: Bearer <API_KEY>
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
- `pkg/contract` — delivery contract for user HTTP workers

## Environment

| Variable | Default | Description |
|----------|---------|-------------|
| `API_KEY` | — (required) | REST Bearer auth |
| `PROVIDERS_CONFIG` | `./config/providers.yaml` | YAML routing (hot-reload) |
| `DELIVERY_RESULT_MODE` | `async` | `async` Redis queue / `sync` HTTP body |
| `STORE_TYPE` | `db` | `file` or `db` |
| `DB_DRIVER` | `postgres` | `postgres`, `mysql`, `sqlite` |
| `QUEUE_MODE` | `asynq` | `local` (no Redis) or `asynq` |
| `REDIS_ADDR` | `localhost:6379` | Only when `QUEUE_MODE=asynq` |
| `REDIS_PASSWORD` | (empty) | Optional Redis AUTH |
| `REDIS_DB` | `0` | Optional Redis DB index |
| `SMTP_HOST` / `SMTP_FROM` | — | Builtin email credentials |
| `DEV_MODE` | `false` | Local dev defaults |
| `HTTP_ADDR` | `:8080` | API listen address |

Full list: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md#9-环境变量) · [`.env.example`](.env.example).

## Tests

```bash
make test
```

## License

Apache License 2.0 — see [LICENSE](LICENSE).
