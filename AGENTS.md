# Herald — Cursor Agent Guide

This file orients Cursor Agent when working in `github.com/SolaTyolo/herald`.

## Read first

| Doc | When |
|-----|------|
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | Architecture, known limits, roadmap |
| [docs/WASM_PROVIDERS.md](docs/WASM_PROVIDERS.md) | **New channel provider** |
| [docs/CURSOR.md](docs/CURSOR.md) | Cursor rules, skills, workflows |
| [README.md](README.md) | Quick start |

## Project identity

- **Name:** Herald
- **Module:** `github.com/SolaTyolo/herald`
- **Provider model:** **WASM-only** — `plugins/wasm/*/provider.wasm`

## Non-negotiable architecture rules

1. **Core stays provider-free** — `cmd/api` / `cmd/worker` must not import provider implementations.
2. **New channel provider** — add WASM under `plugins/wasm/<name>/` only.
3. **Ports, not adapters** — Business code uses `repository.Store` and `wasm.Runtime` (via delivery), not GORM in domain layers.
4. **Use `repository.IsNotFound(err)`** — No driver-specific not-found checks.
5. **Role split** — API: migrations + bridge subscriber (webhook delivery); Worker: Asynq + bridge publish. Both need same `WASM_PLUGIN_DIR` and `REDIS_ADDR`.
6. **In-app** — Core built-in; Worker persists messages + bridge publish; clients poll REST; optional `webhookUrl` push on API.
7. **Minimal diffs** — Match surrounding code; no drive-by refactors.

## Layer map

```
cmd/                 → entrypoints
internal/            → domain, service, workflow, delivery, bootstrap
plugins/wasm/        → WASM providers
pkg/plugin/          → shared WASM JSON contract types
```

## Common commands

```bash
make test
make build
DEV_MODE=true make run-api
make run-worker
make docker-up
```

## Environment essentials

| Variable | Default | Notes |
|----------|---------|-------|
| `WASM_PLUGIN_DIR` | `./plugins/wasm` | **Required** for channel providers |
| `REDIS_ADDR` | `localhost:6379` | Asynq + bridge when `WORKER_API_PUBSUB=redis` |
| `STORE_TYPE` | `db` | `file` or `db` |
| `DEV_MODE` | `false` | local dev |

## Where to put changes

| Task | Location |
|------|----------|
| New WASM provider | `plugins/wasm/<name>/` — see WASM_PROVIDERS.md |
| New REST route | `internal/transport/http/handler/` + `internal/service/` |
| Workflow logic | `internal/workflow/engine.go` |
| WASM runtime | `internal/platform/plugin/wasm/runtime.go` |
| Store | `internal/repository/gormstore/` + `repository.Store` |

## Known gaps

See [docs/ARCHITECTURE.md §7.3](docs/ARCHITECTURE.md#73-已知限制与待办): WASM permissions, topic async fan-out, WASM instance pool, worker OTEL.

## Cursor project config

- Rules: `.cursor/rules/*.mdc`
- Skills: `.cursor/skills/herald-store/`
