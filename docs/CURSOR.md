# Cursor Agent 使用指南

## 1. 文档与配置

| 文件 | 作用 |
|------|------|
| [AGENTS.md](../AGENTS.md) | Agent 入口：架构约束、目录、命令 |
| [.cursor/rules/](../.cursor/rules/) | 持久化规则 |
| [.cursor/skills/](../.cursor/skills/) | 项目级 Skill |
| [docs/PROVIDERS.md](PROVIDERS.md) | YAML 路由 + Worker SDK |
| [docs/ARCHITECTURE.md](ARCHITECTURE.md) | 架构与环境变量 |

## 2. Rules

| 规则 | 范围 | 说明 |
|------|------|------|
| `herald-core.mdc` | 全局 | YAML 路由、分层、Store 端口 |
| `herald-go.mdc` | `**/*.go` | Go 约定 |
| `herald-providers.mdc` | routing / sdk / contract | Provider 与 SDK |
| `herald-store.mdc` | `internal/repository/**` | Store / GORM |

## 3. Skills

| Skill | 场景 |
|-------|------|
| `herald-store` | 修改 `repository.Store` / GORM |

## 4. 工作流

### 新远端 Provider

1. 读 `docs/PROVIDERS.md`
2. 用 `sdk/worker` 实现 `SendProvider`，部署 Worker（需 Redis 写结果）
3. `config/providers.yaml` 增加 `service` + `providers`（`type: remote`）
4. `POST /v1/integrations`，`providerId` = YAML 名

### 存储

1. Skill：`herald-store`
2. 改 `repository.Store` + `gormstore` / `filestore`
3. `repository.IsNotFound(err)`

### 构建

```bash
make build && make test
DEV_MODE=true make run-api
make run-worker
```

## 5. @ 引用

- `@docs/ARCHITECTURE.md`
- `@docs/PROVIDERS.md`
- `@pkg/contract/delivery.go`
- `@internal/repository/repository.go`

## 6. 排查

| 现象 | 检查 |
|------|------|
| Provider not found | `PROVIDERS_CONFIG` 是否声明；YAML 校验是否通过 |
| remote 无结果 | Worker Redis（`REDIS_ADDR` / `HERALD_RESULT_REDIS`）；`DELIVERY_RESULT_MODE` |
| remote 投递失败 | Worker `/delivery`、`/health`；endpoint；健康探测 |
| in_app 无 webhook | 共用 `REDIS_ADDR` / bridge |
| migration 失败 | 仅 API 迁移；`STORE_TYPE` / `DATABASE_URL` |
