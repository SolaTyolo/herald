# Herald — Cursor Agent Guide

## Read first

| Doc | When |
|-----|------|
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | Architecture, env, deployment |
| [docs/PROVIDERS.md](docs/PROVIDERS.md) | YAML routing + remote worker contract |
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
4. **Role split** — `QUEUE_MODE=asynq`: API migrations + bridge subscriber; Worker runs Asynq. Same `PROVIDERS_CONFIG` and `REDIS_ADDR`. `local`: one process, no Redis.
5. **In-app** — Core built-in; Worker persists + bridge publish; REST poll; optional `webhookUrl` on API.
6. **Channel → provider** — Delivery resolves one YAML provider: `WorkflowStep.provider` when set, else the sole provider for that channel (error if multiple). No same-channel failover. Builtin SMTP credentials come from `SMTP_*` env, not the database.
7. **Remote workers** live in the user's process. Herald posts `pkg/contract.DeliveryRequest` to `POST /delivery`. Core must not import user provider implementations. Remote secrets stay in the worker.
8. **Minimal diffs** — Match surrounding code; no drive-by refactors.

## Layer map

```
cmd/                  → api, asynq worker
internal/             → domain, service, workflow, delivery, routing, bootstrap
config/providers.yaml → services + providers
pkg/contract/         → DeliveryRequest / DeliveryError / DeliveryResultEvent
```

## Commands

```bash
make test
make build
DEV_MODE=true make run-api
make run-worker
make playground   # e2e UI on :8090 — see e2e/README.md
make docker-up
```

## Environment

| Variable | Default | Notes |
|----------|---------|-------|
| `API_KEY` | — (required) | REST Bearer auth |
| `PROVIDERS_CONFIG` | `./config/providers.yaml` | YAML routing (hot-reload) |
| `SMTP_HOST` / `SMTP_FROM` / … | — | Builtin `smtp_email` credentials |
| `DELIVERY_RESULT_MODE` | `async` | Redis result queue / `sync` HTTP body |
| `QUEUE_MODE` | `asynq` | `local` or `asynq` |
| `REDIS_ADDR` | `localhost:6379` | required only for `asynq` |
| `REDIS_PASSWORD` | (empty) | optional Redis AUTH |
| `REDIS_DB` | `0` | optional Redis DB index |
| `STORE_TYPE` | `db` | `file` or `db` |
| `DEV_MODE` | `false` | local dev |

## Where to put changes

| Task | Location |
|------|----------|
| YAML routing | `internal/routing/` + `config/providers.yaml` |
| SMTP / remote / result queue | `internal/delivery/` |
| User Delivery Worker | user's own service; contract in `pkg/contract` |
| REST route | `internal/transport/http/handler/` + `internal/service/` |
| Dynamic templates | `internal/domain/template.go` + `internal/template/` + Store |
| Workflow | `internal/workflow/engine.go` |
| Store | `internal/repository/` |

## Cursor config

- Rules: `.cursor/rules/*.mdc`
- Skills: `.cursor/skills/herald-store/`
- CodeGraph: `.cursor/mcp.json` — use `codegraph_explore` for structure, call paths, and impact before grepping source
