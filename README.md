# Herald

Novu-inspired notification infrastructure in Go. Channel providers (email, sms, push, chat) are **WASM extensions**; the core is framework-only. In-app is built into the core (persist messages + REST query; optional subscriber webhook push).

**中文文档:** [README.zh-CN.md](README.zh-CN.md)

**Architecture:** [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) · **WASM plugins:** [docs/WASM_PROVIDERS.md](docs/WASM_PROVIDERS.md)  
**Cursor Agent:** [AGENTS.md](AGENTS.md) · [docs/CURSOR.md](docs/CURSOR.md)

## Features

- Workflow engine: **email, sms, push, chat, in-app** + delay / digest / throttle
- Subscribers, topics, preferences, integrations
- **WASM channel providers** (wazero sandbox)
- Pluggable storage: **file** (JSON) or **db** (PostgreSQL / MySQL / SQLite via GORM)
- OpenTelemetry traces & metrics

## Quick start

```bash
cp .env.example .env   # see README.zh-CN.md for details
make docker-up
# Add WASM plugins under plugins/wasm/<name>/ (see docs/WASM_PROVIDERS.md)
DEV_MODE=true make build

# terminal 1
DEV_MODE=true make run-api

# terminal 2
make run-worker
```

Both API and Worker must see the same `WASM_PLUGIN_DIR`.

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
- `plugins/wasm/` — WASM provider plugins
- `pkg/plugin` — shared types & WASM JSON contract

## Environment

| Variable | Default | Description |
|----------|---------|-------------|
| `WASM_PLUGIN_DIR` | `./plugins/wasm` | WASM plugins root |
| `STORE_TYPE` | `db` | `file` or `db` |
| `DB_DRIVER` | `postgres` | `postgres`, `mysql`, `sqlite` |
| `REDIS_ADDR` | `localhost:6379` | Asynq + throttle |
| `ENCRYPTION_KEY` | — | 32-byte key (required in production) |
| `DEV_MODE` | `false` | Local dev defaults |
| `HTTP_ADDR` | `:8080` | API listen address |

Full list: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md#12-环境变量速查).

## Tests

```bash
make test
```

## License

Apache License 2.0 — see [LICENSE](LICENSE).
