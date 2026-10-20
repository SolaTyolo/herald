# Herald

受 [Novu](https://github.com/novuhq/novu) 启发的 Go 通知基础设施。渠道投递采用 **Traefik 风格 YAML 路由**：内置 **SMTP** 与 **应用内消息 (inapp)**；其余通道由用户自建 Delivery Worker。

**English:** [README.md](README.md)

| 文档 | 说明 |
|------|------|
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | 架构、数据流、环境变量 |
| [docs/PROVIDERS.md](docs/PROVIDERS.md) | YAML 路由 + 用户 Worker 契约 |
| [docs/CURSOR.md](docs/CURSOR.md) | Cursor Agent 工作流 |
| [AGENTS.md](AGENTS.md) | Agent 约束 |

## 特性

- **工作流引擎**：email / sms / push / chat / inapp + delay / digest / throttle
- **动态模板**：多版本、一份生效；步骤用 `templateId` 引用，payload 做变量
- 订阅者、Topic、偏好
- **内置** `smtp_email`（环境变量 `SMTP_*`）/ `inapp`；**远端**走用户自建 HTTP Worker（`POST /delivery`）
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
                    ├─ inapp / smtp_email → 本地执行
                    └─ remote → POST 用户 Worker /delivery
                         └─ 结果经 Redis 回传（默认 async）
```

- **API**：HTTP、迁移；Topic 触发返回 `async: true`
- **Worker**：工作流步骤 + Topic fan-out
- `QUEUE_MODE=asynq`（默认）时共用 `PROVIDERS_CONFIG` 与 `REDIS_ADDR`；`local` 单进程、不用 Redis
- 本地联调 UI：`make playground` → `http://localhost:8090`（独立于 API，见 [e2e/README.md](e2e/README.md)）

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

在 `.env` 中设置必填的 `API_KEY`，请求头：

```
Authorization: Bearer <API_KEY>
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

内置 SMTP 凭据（API 与 Worker 都要配）：

```bash
export SMTP_HOST=smtp.example.com
export SMTP_PORT=587
export SMTP_FROM=noreply@example.com
export SMTP_USERNAME=user
export SMTP_PASSWORD=pass
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
  delivery/            smtp / inapp / remote + 结果队列
  routing/             YAML 路由 + 热加载 + 健康探测
  repository/          Store 端口
pkg/contract/          投递契约（用户 Worker 按此实现）
```

## 环境变量

| 变量 | 默认 | 说明 |
|------|------|------|
| `API_KEY` | —（必填） | REST Bearer 鉴权 |
| `DEV_MODE` | `false` | 本地开发 |
| `HTTP_ADDR` | `:8080` | API 监听 |
| `SMTP_HOST` / `SMTP_FROM` | — | 内置邮件凭据 |
| `PROVIDERS_CONFIG` | `./config/providers.yaml` | YAML 路由 |
| `DELIVERY_RESULT_MODE` | `async` | `async` / `sync` |
| `QUEUE_MODE` | `asynq` | `local` 或 `asynq` |
| `REDIS_ADDR` | `localhost:6379` | 仅 `asynq` 需要 |
| `REDIS_PASSWORD` | （空） | 可选 Redis AUTH |
| `REDIS_DB` | `0` | 可选 Redis 逻辑库 |
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
