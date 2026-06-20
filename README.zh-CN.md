# Herald

受 [Novu](https://github.com/novuhq/novu) 启发的 Go 通知基础设施。渠道投递采用 **Traefik 风格 YAML 路由**：内置 **SMTP** 与 **应用内消息 (inapp)**；其余通道由用户自建 Delivery Worker。

**English:** [README.md](README.md)

| 文档 | 说明 |
|------|------|
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | 架构、数据流、环境变量 |
| [docs/PROVIDERS.md](docs/PROVIDERS.md) | YAML 路由 + Worker SDK |
| [docs/CURSOR.md](docs/CURSOR.md) | Cursor Agent 工作流 |
| [AGENTS.md](AGENTS.md) | Agent 约束 |

## 特性

- **工作流引擎**：email / sms / push / chat / in_app + delay / digest / throttle
- 订阅者、Topic、偏好、Integration
- **内置** `smtp_email` / `inapp`；**远端**走用户 HTTP Worker（`sdk/worker`）
- 默认 Redis **异步结果队列**；Topic 异步 fan-out
- 存储：**file** 或 **db**（PostgreSQL / MySQL / SQLite，GORM）
- OpenTelemetry 链路 / 指标

## 架构概览

```
Client → API（REST）
           ↓ 入队
         Redis / Asynq
           ↓
         Worker → workflow → delivery
                    ├─ in_app / smtp_email → 本地执行
                    └─ remote → POST 用户 Worker /delivery
                         └─ 结果经 Redis 回传（默认 async）
```

- **API**：HTTP、迁移；Topic 触发返回 `async: true`
- **Worker**：工作流步骤 + Topic fan-out
- 共用 `PROVIDERS_CONFIG` 与 `REDIS_ADDR`

## 快速开始

```bash
cp .env.example .env
set -a && source .env && set +a
make docker-up
make build
make run-api      # 终端 1
make run-worker   # 终端 2
```

编辑 [`config/providers.yaml`](config/providers.yaml) 配置路由。新增短信/推送等：自建 Worker + YAML `type: remote`。见 [PROVIDERS.md](docs/PROVIDERS.md)。

## 认证

```
Authorization: Bearer <your-key>
```

## 示例

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

创建 SMTP Integration（`providerId` 须与 YAML 一致）：

```bash
curl -X POST http://localhost:8080/v1/integrations \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "channel": "email",
    "providerId": "smtp_email",
    "name": "SMTP",
    "credentials": {
      "host": "smtp.example.com",
      "port": "587",
      "username": "user",
      "password": "pass",
      "from": "noreply@example.com"
    },
    "primary": true,
    "active": true
  }'
```

```bash
curl -H "Authorization: Bearer $API_KEY" http://localhost:8080/v1/providers
```

## 目录结构

```
cmd/api                HTTP
cmd/worker             Asynq 任务消费
config/providers.yaml  Traefik 风格路由
internal/
  workflow/            工作流引擎
  delivery/            smtp / in_app / remote + 结果队列
  routing/             YAML 路由 + 热加载 + 健康探测
  repository/          Store 端口
pkg/contract/          投递契约
sdk/worker/            用户 Delivery Worker SDK
```

## 环境变量

| 变量 | 默认 | 说明 |
|------|------|------|
| `DEV_MODE` | `false` | 本地开发 |
| `HTTP_ADDR` | `:8080` | API 监听 |
| `ENCRYPTION_KEY` | — | **32 字节** |
| `PROVIDERS_CONFIG` | `./config/providers.yaml` | YAML 路由 |
| `DELIVERY_RESULT_MODE` | `async` | `async` / `sync` |
| `REDIS_ADDR` | `localhost:6379` | Asynq + bridge + 结果队列 |
| `STORE_TYPE` | `db` | `file` 或 `db` |

完整列表见 [.env.example](.env.example) 与 [ARCHITECTURE §9](docs/ARCHITECTURE.md#9-环境变量)。

## 常用命令

| 命令 | 说明 |
|------|------|
| `make docker-up` | 启动 Postgres + Redis |
| `make build` | 编译 |
| `make run-api` / `make run-worker` | 启动进程 |
| `make test` | 测试 |

## License

Apache License 2.0 — 见 [LICENSE](LICENSE)。
