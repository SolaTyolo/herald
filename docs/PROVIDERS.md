# Herald Providers

> Module：`github.com/SolaTyolo/herald`

## 模型

| 类型 | providerName | 执行位置 |
|------|--------------|----------|
| **builtin** | `smtp_email`、`inapp` | Asynq Worker 进程本地 |
| **remote** | 用户自定义（如 `tencent_sms`） | 用户 Delivery Worker（HTTP） |

配置对齐 **Traefik v3 File Provider**：`services` 定义后端，`providers` 定义路由。

- 路径：`PROVIDERS_CONFIG`（默认 `./config/providers.yaml`）
- 热加载：`fsnotify`；非法配置拒绝，保留上一份有效表

## YAML

见 [`config/providers.yaml`](../config/providers.yaml)。

```yaml
services:
  user-sms-worker:
    endpoint: "http://user-sms-worker:8080/delivery"
    healthCheck:
      path: "/health"
      interval: "5s"
      timeout: "2s"

providers:
  smtp_email:
    type: builtin
    channel: email
    timeout: 10s
  inapp:
    type: builtin
    channel: inapp
    timeout: 5s
  tencent_sms:
    type: remote
    service: user-sms-worker
    channel: sms
    timeout: 10s
```

### 校验

- `providerName` 全局唯一
- `builtin` 禁止 `service`；名称仅 `smtp_email` / `inapp`
- `remote` 必须引用已存在的 `service`
- `service.endpoint` 为合法 `http` / `https` URL
- Workflow 按 **channel** 编排；投递时 `ProvidersForChannel` → 按名称排序尝试

## 凭据

| provider | 凭据位置 |
|----------|----------|
| `smtp_email` | 环境变量 `SMTP_HOST` / `SMTP_PORT` / `SMTP_FROM` / `SMTP_USERNAME` / `SMTP_PASSWORD` |
| remote | 用户 Worker 进程自行保管；Herald 传空 `providerConfig.raw` |
| `inapp` | 无 |

## 内置 SMTP

```bash
export SMTP_HOST=smtp.example.com
export SMTP_PORT=587   # 587=STARTTLS；465=隐式 TLS（SMTPS）
export SMTP_FROM=noreply@example.com
export SMTP_USERNAME=user
export SMTP_PASSWORD=pass
```

腾讯云等企业邮：`465` 走隐式 TLS；`SMTP_USERNAME` 一般为发信账号（常为完整邮箱），不是显示名。

API 与 Worker 需配置相同值。

## 内置 InApp

写 `messages` + bridge；可选 subscriber `webhookUrl`；客户端 `GET /v1/subscribers/{id}/messages`。

## 用户 Delivery Worker

用户自己实现并部署 HTTP 服务。Herald 只认 `pkg/contract` 里的 JSON 契约，不提供 SDK。

端点：`POST /delivery`（YAML `service.endpoint`）、`GET /health`（健康探测）。

### POST /delivery

```json
{
  "taskId": "job-uuid",
  "providerName": "tencent_sms",
  "traceId": "transaction-id",
  "async": true,
  "providerConfig": {
    "timeout_sec": 10,
    "raw": { "secretId": "xxx", "appId": "123" }
  },
  "message": {
    "channel": "sms",
    "to": "+86138xxxx",
    "content": "hello",
    "metadata": {}
  },
  "context": {
    "transactionId": "transaction-id",
    "workflowId": "wf-uuid",
    "notificationId": "notif-uuid"
  }
}
```

## 结果回传（`DELIVERY_RESULT_MODE`）

| 模式 | 行为 |
|------|------|
| **`async`（默认）** | 请求带 `async` / `X-Herald-Async: 1`；Worker `202 Accepted`，完成后写 Redis `DeliveryResultEvent`；Herald `BRPOP` 按 `taskId` 取结果 |
| **`sync`** | 等待 HTTP 响应体中的 `DeliveryError` |

`DELIVERY_RESULT_MODE=async` 只在 `QUEUE_MODE=asynq` 下可用。用户 Worker 完成后把 `DeliveryResultEvent` JSON `LPUSH` 到同一 Redis（`REDIS_ADDR`）。

| Redis key | 用途 |
|-----------|------|
| `herald:delivery:result:{taskId}` | 单任务等待（TTL 15m） |
| `herald:delivery:results` | 审计列表（约最近 1 万条） |

## Topic 触发

API 校验后入队 fan-out，立即返回：

```json
{ "transactionId": "...", "async": true }
```

用 `transactionId` 查 notifications / jobs 跟踪进度。

## 投递链路与保障

```
Trigger → Asynq → workflow.Engine
  → inapp → 本地写库
  → 其他 → ProvidersForChannel → routing
       → smtp_email → 本地 SMTP（SMTP_*）
       → remote → POST /delivery（+ 结果队列或 HTTP 体）
```

- `retryable=true` → Asynq 重试；否则 `SkipRetry`
- 同 `jobId` 已有 message → 幂等跳过
- service `healthCheck` 失败 → remote 可重试错误

## GET /v1/providers

返回 YAML 路由表：`id` / `channel` / `type`；remote 含 `service` / `endpoint`。
