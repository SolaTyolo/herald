# Cursor Agent 使用指南

本文说明如何在 Herald 项目中高效使用 **Cursor Agent**（Chat Agent、Background Agent、Rules、Skills）。

## 1. 文档与配置一览

| 文件 | 作用 |
|------|------|
| [AGENTS.md](../AGENTS.md) | Agent 入口：架构约束、目录、命令（Cursor 自动读取） |
| [.cursor/rules/](../.cursor/rules/) | 持久化规则（按 glob 或全局生效） |
| [.cursor/skills/](../.cursor/skills/) | 项目级 Skill（复杂工作流） |
| [docs/WASM_PROVIDERS.md](WASM_PROVIDERS.md) | **新增 WASM provider** |
| [docs/ARCHITECTURE.md](ARCHITECTURE.md) | 架构、已知限制、路线图 |

## 2. Cursor Rules（`.cursor/rules/`）

| 规则文件 | 范围 | 说明 |
|----------|------|------|
| `herald-core.mdc` | 全局 | WASM-only、分层、禁止事项 |
| `herald-go.mdc` | `**/*.go` | Go 编码约定 |
| `herald-wasm.mdc` | `plugins/wasm/**` | WASM provider |
| `herald-store.mdc` | `internal/repository/**` | Store / GORM |

## 3. Project Skills（`.cursor/skills/`）

| Skill | 触发场景 |
|-------|----------|
| `herald-store` | 修改 `repository.Store` / GORM |

## 4. 推荐 Agent 工作流

### 4.1 新 WASM Channel Provider

1. 阅读 `docs/WASM_PROVIDERS.md`
2. 在 `plugins/wasm/<name>/` 添加 `provider.json` + `provider.wasm`
3. 重启 API 与 Worker
4. `POST /v1/integrations`

### 4.2 存储 / 数据库

1. 使用 skill：`herald-store`
2. 只改 `repository.Store` 端口 + `gormstore`/`filestore`
3. 用 `repository.IsNotFound(err)` 判断不存在

### 4.3 构建

```bash
make build
make test
DEV_MODE=true make run-api
make run-worker
```

## 5. 常用 @ 引用

- `@docs/ARCHITECTURE.md` — 架构上下文
- `@docs/WASM_PROVIDERS.md` — WASM provider 开发
- `@internal/repository/repository.go` — Store 接口
- `@pkg/plugin/provider.go` — WASM JSON 契约类型

## 6. 故障排查

| 现象 | 检查 |
|------|------|
| Provider not found | `WASM_PLUGIN_DIR` 是否有 `provider.json` + `provider.wasm`；API 与 Worker 是否都已重启 |
| in_app 无 SSE | API 与 Worker 是否共用 `REDIS_ADDR` |
| migration 失败 | 仅 API 启动迁移；检查 `STORE_TYPE` / `DATABASE_URL` |

## 7. 参考

- [Herald ARCHITECTURE.md](ARCHITECTURE.md)
- [WASM_PROVIDERS.md](WASM_PROVIDERS.md)
