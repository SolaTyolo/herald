# E2E / Playground

Manual end-to-end UI for Herald. Not part of the API binary.

## Playground

Requires API (and Worker when `QUEUE_MODE=asynq`) already running.

```bash
# terminal 1–2
DEV_MODE=true make run-api
make run-worker

# terminal 3
make playground
# open http://localhost:8090
```

Paste the same value as process env `API_KEY` (see `.env` / `make` default `herald-dev-key`). The playground proxies `/v1/*` and `/health` to `HERALD_API` (default `http://localhost:8080`).

| Variable / flag | Default | Meaning |
|-----------------|---------|---------|
| `PLAYGROUND_ADDR` / `-addr` | `:8090` | Listen address |
| `HERALD_API` / `-api` | `http://localhost:8080` | Herald API base URL |

Tabs: **In-app**, **SMTP**, **Template**, **Workflow**.
