# Cursor Agent 使用指南

## 1. 文档与配置

| 文件 | 作用 |
|------|------|
| [AGENTS.md](../AGENTS.md) | Agent 入口：架构约束、目录、命令 |
| [.cursor/rules/](../.cursor/rules/) | 持久化规则 |
| [.cursor/skills/](../.cursor/skills/) | 项目级 Skill |
| [docs/PROVIDERS.md](PROVIDERS.md) | YAML 路由 + 用户 Worker 契约 |
| [docs/ARCHITECTURE.md](ARCHITECTURE.md) | 架构与环境变量 |

## 2. Rules

| 规则 | 范围 | 说明 |
|------|------|------|
| `herald-core.mdc` | 全局 | YAML 路由、分层、Store 端口 |
| `herald-go.mdc` | `**/*.go` | Go 约定 |
| `herald-providers.mdc` | routing / contract | Provider 路由与投递契约 |
| `herald-store.mdc` | `internal/repository/**` | Store / GORM |

## 3. Skills

| Skill | 场景 |
|-------|------|
| `herald-store` | 修改 `repository.Store` / GORM |

## 4. 工作流

### 新远端 Provider

1. 读 `docs/PROVIDERS.md`
2. 按 `pkg/contract` 自建 HTTP Worker（`POST /delivery`、`GET /health`）；async 模式写 Redis 结果队列
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

## 6. CodeGraph

本地代码图，供 Agent 通过 MCP 工具 `codegraph_explore` 查符号、调用链和影响范围。配置在 `.cursor/mcp.json`；索引在 `.codegraph/`（数据库不入库，改文件后自动同步）。

本机需已安装 CLI（`codegraph` 在 `PATH`）。新克隆后执行一次：

```bash
codegraph init
```

改完 MCP 配置后重启 Cursor。

## 7. 排查

| 现象 | 检查 |
|------|------|
| Provider not found | `PROVIDERS_CONFIG` 是否声明；YAML 校验是否通过 |
| remote 无结果 | `QUEUE_MODE=asynq` 时的 Redis（`REDIS_ADDR`）；`DELIVERY_RESULT_MODE` |
| remote 投递失败 | Worker `/delivery`、`/health`；endpoint；健康探测 |
| in_app 无 webhook | `asynq` 共用 `REDIS_ADDR`；`local` 为进程内 bridge |
| migration 失败 | 仅 API 迁移；`STORE_TYPE` / `DATABASE_URL` |
