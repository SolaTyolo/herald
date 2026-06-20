# Herald — Cursor Agent Guide

## Read first

| Doc | When |
|-----|------|
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | Architecture, env, deployment |
| [docs/PROVIDERS.md](docs/PROVIDERS.md) | YAML routing + Worker SDK |
| [docs/CURSOR.md](docs/CURSOR.md) | Cursor rules, skills, workflows |
| [README.md](README.md) | Quick start |

## Project identity

- **Name:** Herald
- **Module:** `github.com/SolaTyolo/herald`
- **Provider model:** Traefik-style YAML — builtin `smtp_email` / `inapp` + user Delivery Workers

## Architecture rules

1. **Builtin vs remote** — only `smtp_email` and `inapp` are builtin (local). All other channels are `remote` HTTP workers. Do not add in-process plugin runtimes.
2. **Ports, not adapters** — Business code uses `repository.Store` and `routing.Table` / delivery, not GORM in domain layers.
3. **Use `repository.IsNotFound(err)`** — No driver-specific not-found checks.
4. **Role split** — API: migrations + bridge subscriber; Worker: Asynq + bridge publish. Same `PROVIDERS_CONFIG` and `REDIS_ADDR` on both.
5. **In-app** — Core built-in; Worker persists + bridge publish; REST poll; optional `webhookUrl` on API.
6. **Integration.providerId** must match a YAML `providerName`.
7. **Worker SDK** (`sdk/worker`) is for **user** Delivery Workers only; Herald core must not import it.
8. **Minimal diffs** — Match surrounding code; no drive-by refactors.

## Layer map

```
cmd/                  → api, asynq worker
internal/             → domain, service, workflow, delivery, routing, bootstrap
config/providers.yaml → services + providers
pkg/contract/         → DeliveryRequest / DeliveryError / DeliveryResultEvent
sdk/worker/           → user Delivery Worker SDK
```

## Commands

```bash
make test
make build
DEV_MODE=true make run-api
make run-worker
make docker-up
```

## Environment

| Variable | Default | Notes |
|----------|---------|-------|
| `PROVIDERS_CONFIG` | `./config/providers.yaml` | YAML routing (hot-reload) |
| `DELIVERY_RESULT_MODE` | `async` | Redis result queue / `sync` HTTP body |
| `REDIS_ADDR` | `localhost:6379` | Asynq + bridge + results |
| `STORE_TYPE` | `db` | `file` or `db` |
| `DEV_MODE` | `false` | local dev |

## Where to put changes

| Task | Location |
|------|----------|
| YAML routing | `internal/routing/` + `config/providers.yaml` |
| SMTP / remote / result queue | `internal/delivery/` |
| User Delivery Worker | user repo + `sdk/worker` |
| REST route | `internal/transport/http/handler/` + `internal/service/` |
| Workflow | `internal/workflow/engine.go` |
| Store | `internal/repository/` |

## Cursor config

- Rules: `.cursor/rules/*.mdc`
- Skills: `.cursor/skills/herald-store/`
