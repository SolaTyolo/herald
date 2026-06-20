---
name: herald-store
description: Implements or extends Herald persistence via repository.Store and GORM. Use when adding store methods, migrations, file/db mode, DB_DRIVER changes, or repository.IsNotFound handling.
---

# Herald Store Development

## Add a Store method

1. Add signature to `internal/repository/repository.go` (`Store` interface)
2. Implement in **both** backends when behavior should match:
   - `internal/repository/gormstore/` — pick the domain file (`workflows.go`, `subscribers.go`, …)
   - `internal/repository/filestore/` — same domain file name, JSON files under `envs/{envId}/`
3. Use `repository.IsNotFound(err)` in callers (`workflow`, `service`, `handler`)

## GORM model (db mode only)

- Add struct to `gormstore/models.go` with `TableName()` and `toDomain()`
- Register in `allModels()` for AutoMigrate
- JSON columns: `[]byte` with `gorm:"type:json"`
- UUIDs: `string` size 36

## Config

Env vars in `internal/config/config.go`:

- `STORE_TYPE`: `file` | `db`
- `DB_DRIVER`: `postgres` | `mysql` | `sqlite` (when `STORE_TYPE=db`)
- `DATABASE_URL`, `STORE_FILE_PATH`

| Mode | Backend | Layout |
|------|---------|--------|
| `file` | `filestore` | `{STORE_FILE_PATH}/bootstrap.json`, `envs/{envId}/*.json` |
| `db` | `gormstore` | GORM AutoMigrate |

## Test

```bash
STORE_TYPE=file DEV_MODE=true go test ./... -count=1
STORE_TYPE=db DB_DRIVER=postgres DATABASE_URL='...' go test ./... -count=1
```

## Do not

- Do not use raw SQL in business packages; extend `repository.Store` instead.
- Run migrations from Worker bootstrap
- Break existing unique indexes without migration plan
- Put SQLite under `STORE_TYPE=file` — use `STORE_TYPE=db` + `DB_DRIVER=sqlite` for embedded SQL
