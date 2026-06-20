# Herald

受 [Novu](https://github.com/novuhq/novu) 启发的 Go 通知基础设施。渠道 Provider（email / sms / push / chat）以 **WASM 插件**交付；**核心不含 Provider 实现**。`in_app` 由核心内置（写库 + REST 查询；可选 `webhookUrl` 推送下游）。

**English:** [README.md](README.md)

| 文档 | 说明 |
|------|------|
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | 架构、数据流、环境变量 |
| [docs/WASM_PROVIDERS.md](docs/WASM_PROVIDERS.md) | 新增 WASM Provider |
| [docs/CURSOR.md](docs/CURSOR.md) | Cursor Agent 工作流 |
| [AGENTS.md](AGENTS.md) | Agent 约束 |

## 特性

- **工作流引擎**：email / sms / push / chat / in_app + delay / digest / throttle
- 订阅者、Topic、偏好、Integration
- **WASM 渠道插件**（wazero 沙箱 + `herald.*` host imports）
- 可插拔存储：**file**（JSON 文件）或 **db**（PostgreSQL / MySQL / SQLite，GORM）
- OpenTelemetry 链路 / 指标

## 架构概览

```
Client → API（REST）
           ↓ 入队
         Redis / Asynq
           ↓
         Worker → workflow → delivery
                    ├─ in_app  → 写库 + bridge → API（可选 webhook 推送）
                    └─ 其他渠道 → WASM Send
```

- **API**：HTTP、数据库迁移（GORM AutoMigrate）
- **Worker**：异步执行工作流步骤
- 两者必须共用同一 `WASM_PLUGIN_DIR` 和 `REDIS_ADDR`

## 快速开始

### 1. 环境配置

```bash
cp .env.example .env
# 按需编辑 .env
set -a && source .env && set +a
```

### 2. 基础设施（Postgres + Redis）

```bash
make docker-up
```

### 3. WASM 插件

在 `plugins/wasm/<name>/` 放置 `provider.json` + `provider.wasm`，详见 [WASM_PROVIDERS.md](docs/WASM_PROVIDERS.md)。

### 4. 构建与启动

```bash
make build

# 终端 1 — API
make run-api

# 终端 2 — Worker
make run-worker
```

首次启动 API 会在日志中打印默认 **ApiKey**（`DEV_MODE=true` 时会明文输出一次）。

### 无 Docker（file 模式）

```bash
STORE_TYPE=file DEV_MODE=true make run-api
# 另开终端
make run-worker
```

## 认证

所有业务接口（除 `/health`）需要：

```
Authorization: Bearer <your-key>
```

## 示例：触发通知

```bash
export API_KEY=hr_xxxxxxxx   # 启动时日志中的 key

curl -X POST http://localhost:8080/v1/events/trigger \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "welcome",
    "to": {"subscriberId": "user-1"},
    "payload": {"name": "Alice"}
  }'
```

创建 Integration（需已加载对应 WASM provider）：

```bash
curl -X POST http://localhost:8080/v1/integrations \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "channel": "email",
    "providerId": "sendgrid",
    "name": "SendGrid",
    "credentials": {"apiKey": "SG.xxx", "from": "noreply@example.com"},
    "primary": true,
    "active": true
  }'
```

查看已加载 Provider：

```bash
curl -H "Authorization: Bearer $API_KEY" http://localhost:8080/v1/providers
```

## 目录结构

```
cmd/api              HTTP + SSE
cmd/worker           Asynq 任务消费
internal/
  workflow/          工作流引擎
  delivery/          渠道投递（in_app 内置 + WASM）
  platform/plugin/wasm/  WASM 运行时与 host imports
  repository/        Store 端口（GORM）
plugins/wasm/        WASM 插件目录
plugins/sdk/herald/  TinyGo host import SDK
pkg/plugin/          WASM JSON 契约类型
```

## 环境变量

完整说明见 [.env.example](.env.example) 与 [ARCHITECTURE §12](docs/ARCHITECTURE.md#12-环境变量速查)。

| 变量 | 默认 | 说明 |
|------|------|------|
| `DEV_MODE` | `false` | 本地开发；允许默认 `ENCRYPTION_KEY` |
| `HTTP_ADDR` | `:8080` | API 监听地址 |
| `ENCRYPTION_KEY` | — | **32 字节**；生产必改 |
| `WASM_PLUGIN_DIR` | `./plugins/wasm` | WASM 插件根目录 |
| `REDIS_ADDR` | `localhost:6379` | Asynq 队列 + 限流 + bridge（`WORKER_API_PUBSUB=redis`） |
| `WORKER_API_PUBSUB` | `redis` | Worker↔API：`redis` / `local` / `rabbitmq-http` / `kafka-http` |
| `STORE_TYPE` | `db` | `file` 或 `db` |
| `DB_DRIVER` | `postgres` | `postgres` / `mysql` / `sqlite` |
| `DATABASE_URL` | 见 .env.example | 数据库 DSN |
| `STORE_FILE_PATH` | `./data` | file 模式数据目录 |
| `MAX_RECIPIENTS` | `100` | Topic 触发单次上限 |

## 常用命令

| 命令 | 说明 |
|------|------|
| `make docker-up` | 启动 Postgres + Redis |
| `make docker-down` | 停止容器 |
| `make build` | 编译 api / worker |
| `make run-api` | 运行 API |
| `make run-worker` | 运行 Worker |
| `make test` | 运行测试 |

## 测试

```bash
make test
```

## 许可证

[Apache License 2.0](LICENSE)
