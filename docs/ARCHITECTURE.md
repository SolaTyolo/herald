# Herald 架构分析与优化指南

> 文档版本：**3.2**  
> 最后更新：2026-06-20  
> Module：`github.com/SolaTyolo/herald`

## 1. 项目概述

**Herald** 是受 [Novu](https://github.com/novuhq/novu) 启发的通知基础设施，采用 [Paca](https://github.com/Paca-AI/paca) 风格的 **WASM 插件架构**：所有 channel provider（email / sms / push / chat）以 WASM 扩展交付；**核心不含任何 channel provider 实现**。`in_app` 由核心内置（写库 + 客户端 REST 拉取；可选 subscriber `webhookUrl` 主动推送）。

| 组件 | 技术 |
|------|------|
| API | Chi、REST、OTEL |
| Worker | Asynq + Redis |
| 持久化 | file：JSON 文件；db：GORM（PostgreSQL · MySQL · SQLite） |
| **Channel 插件** | wazero WASM + `plugins/wasm/*/provider.wasm` |
| In-app | 核心内置（写库 + REST 查询；可选 webhook 推送） |
| 可观测性 | slog + OpenTelemetry（traces / metrics） |

## 2. 目录与分层

```
cmd/api                    HTTP（bootstrap.RoleAPI）
cmd/worker                 异步任务（bootstrap.RoleWorker）
internal/
  domain/                  纯实体
  repository/              Store 端口 + Open 工厂
    filestore/             JSON 文档文件（STORE_TYPE=file）
    gormstore/             GORM/SQL（STORE_TYPE=db）
  service/                 用例层（Trigger、CRUD、Provider 安装）
  workflow/                步骤编排引擎（核心）
  delivery/                渠道投递 + WASM 调度 + in_app webhook
  queue/asynq/             任务入队（Asynq + Redis）
  realtime/
    bridge/                Worker ↔ API 事件总线（redis / kafka-http / rabbitmq-http）
  platform/plugin/wasm/    WASM 运行时
  telemetry/               slog + OTEL
  transport/http/          Handler + Auth + Metrics
  bootstrap/               组合根（按 Role 拆分）
pkg/plugin/                Provider WASM JSON 契约类型
plugins/wasm/              WASM 插件（每 provider 一目录）
.github/workflows/         CI
```

### 2.1 依赖方向

```mermaid
flowchart TB
  subgraph transport [Transport]
    H[handler.API]
    MW[AuthMiddleware + MetricsMiddleware]
  end
  subgraph app [Application]
    SVC[service.*]
    ENG[workflow.Engine]
    DEL[delivery.Service]
  end
  subgraph infra [Infrastructure]
    ST[(repository.Store)]
    Q[asynq]
    R[redis]
    BR[bridge]
    WH[in_app webhook]
    WASM[wasm.Runtime]
    OTEL[telemetry]
  end
  subgraph ext [WASM Extensions]
    WASMEXT["plugins/wasm/*/provider.wasm"]
  end
  H --> SVC
  MW --> ST
  SVC --> ST
  ENG --> ST
  ENG --> Q
  ENG --> DEL
  ENG --> R
  DEL --> ST
  DEL --> WASM
  DEL --> BR
  BR --> WH
  WASMEXT --> WASM
  OTEL -.-> H
```

**核心设计原则：**

- **端口与适配器**：业务层依赖 `repository.Store`；投递依赖 `wasm.Runtime`，不感知 GORM / 具体 WASM 实现语言
- **扩展与核心分离**：`cmd/*` 不 import 任何 provider 实现；新增 provider 仅增 WASM 目录 + Integration
- **双进程协作**：API 负责 HTTP/Migration；Worker 负责 job 执行；**两者须挂载同一 `WASM_PLUGIN_DIR`**
- **Worker ↔ API 事件总线**：`realtime/bridge`（redis / kafka-http / rabbitmq-http），与 Asynq 队列分离
- **In-app 客户端**：默认 `GET /v1/subscribers/{id}/messages` 拉取；若配置了 `webhookUrl` 则额外 POST 推送到下游

## 3. 持久化

### 3.1 两种 Store 模式

| `STORE_TYPE` | 行为 | 典型场景 |
|--------------|------|----------|
| `file` | `{STORE_FILE_PATH}/` 下 JSON（`bootstrap.json`、`envs/{envId}/…`） | 本地开发、无数据库 |
| `db` | GORM + `DB_DRIVER` | 生产 |

### 3.2 DB 驱动（`STORE_TYPE=db`）

| `DB_DRIVER` | 默认 `DATABASE_URL` |
|-------------|---------------------|
| `postgres` | `postgres://herald:herald@localhost:5432/herald?sslmode=disable` |
| `mysql` | `herald:herald@tcp(localhost:3306)/herald?charset=utf8mb4&parseTime=True&Loc=Local` |
| `sqlite` | `./data/herald.db` |

Migration：GORM `AutoMigrate`，**仅 API 启动时**执行。

### 3.3 Store 端口

`repository.Store` 定义全部持久化操作；`repository.IsNotFound(err)` 统一判断记录不存在。

## 4. 插件架构

### 4.1 WASM 扩展（唯一路径）

新增 email/sms/push/chat provider **无需修改核心代码**。详见 **[WASM_PROVIDERS.md](WASM_PROVIDERS.md)**。

| 步骤 | 操作 |
|------|------|
| 1 | 在 `WASM_PLUGIN_DIR/<name>/` 放置 `provider.json` + `provider.wasm` |
| 2 | 重启 **API 与 Worker**（`LoadFromDir`） |
| 3 | `POST /v1/integrations`，`providerId` = manifest `id` |

运行时：`delivery` 要求 `wasm.HasProvider(id)` → `wasm.Send`；Send JSON 含 `context`；插件可 import `herald.*` host 函数。

沙箱：16MB 内存、10s 调用超时；`manifest.permissions` 在加载时校验，`http_fetch` 按白名单放行。

#### Host imports（Paca 式扩展点）

| Import | 说明 | 权限 |
|--------|------|------|
| `herald.log` | 写 slog | 默认 |
| `herald.get_context` | 读 CallContext JSON | 默认 |
| `herald.http_fetch` | 出站 HTTP | `network:https://host` |

SDK：`plugins/sdk/herald/`（TinyGo `//go:wasmimport`）

#### WASM 运行时现状

| 能力 | 状态 |
|------|------|
| `malloc` + `Send` 导出 | 已实现 |
| Send JSON `context` 字段 | **已实现** |
| `ValidateConfig` 导出 | 已接入 Integration 创建 |
| Host `log` / `get_context` / `http_fetch` | **已实现** |
| manifest `permissions` | **加载时校验 + http_fetch 强制** |
| 每次 Send 实例化 module | 有性能开销（待实例池） |
| 热更新 | 需重启 |

### 4.2 In-app（核心内置）

```
in_app 步骤 → Worker 写 messages 表 → bridge.Publisher
         → (redis | kafka-http | rabbitmq-http)
         → API bridge.Subscriber →（可选）POST subscriber.webhookUrl
```

- **Worker ↔ API**：`realtime/bridge`，与 Asynq 队列无关；环境变量 `WORKER_API_PUBSUB`
- **主路径**：Message 落库，客户端 `GET /v1/subscribers/{id}/messages` 拉取（PATCH 标记已读）
- **可选推送**：subscriber 配置了 `webhookUrl` 时，API 额外 POST 到下游
- **redis**（默认）、**kafka-http**、**rabbitmq-http**（Rabbit 订阅经 sidecar）

创建 subscriber 示例（webhook 可选）：

```json
{ "subscriberId": "user-1", "webhookUrl": "https://your-app.example/hooks/in-app" }
```

### 4.3 投递路径

```
delivery.SendChannel
  ├─ in_app  → CreateMessage + bridge.Publisher（API 可选 webhook 推送）
  └─ channel → wasm.HasProvider(id) → wasm.Send（未加载则报错）
```

## 5. 核心数据流

```mermaid
sequenceDiagram
  participant Client
  participant API
  participant Store
  participant Redis
  participant Worker
  participant WASM as WASM Provider

  Client->>API: POST /v1/events/trigger
  API->>Store: Notification + Jobs
  API->>Redis: enqueue job[0]
  Redis->>Worker: ProcessJob
  Worker->>Store: GetNotification + payload
  Worker->>WASM: Send（wazero 沙箱）
  Worker->>Store: Message + execution_logs
  Note over Worker,Redis: in_app 时 Worker publish → API subscribe → SSE
  Worker->>Redis: enqueue job[n+1]
```

## 6. 可观测性

| 变量 | 说明 | 默认 |
|------|------|------|
| `LOG_LEVEL` | debug / info / warn / error | `info` |
| `LOG_FORMAT` | text / json | `text` |
| `OTEL_ENABLED` | 启用 OTLP 导出 | `false` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP HTTP 端点 | `http://localhost:4318` |
| `OTEL_SERVICE_NAME` | bootstrap 按 Role 设为 `herald-api` / `herald-worker` | `herald` |

## 7. 架构评审

### 7.1 优势

- **域模型清晰**：Workflow / Subscriber / Topic / Integration 与 Novu 对齐
- **WASM 扩展零核心改动**：provider 独立编译、独立发布
- **存储可替换**：file / postgres / mysql / sqlite
- **In-app 跨进程 SSE**：Redis Pub/Sub 桥接 Worker 与 API

### 7.2 已完成优化

| 版本 | 内容 |
|------|------|
| 2.0 | P0/P1（payload、delay、bcrypt、WASM 沙箱、Role 拆分等） |
| 3.0 | GORM 多存储、OTEL |
| 3.1 | WASM-first 文档对齐；`HasProvider` 路由修复 |
| **3.4** | Host imports（log / get_context / http_fetch）+ Send context + permissions |

### 7.3 已知限制与待办

#### P1 — 扩展性 / 性能 / 运维

| ID | 问题 | 建议 |
|----|------|------|
| P1-2 | Digest 高并发竞态 | 事务 + `SELECT FOR UPDATE` |
| P1-3 | Topic fan-out 同步阻塞 trigger HTTP | Asynq fan-out 任务 |
| P1-5 | WASM manifest `permissions` 未强制 | ~~安装/加载时校验~~ **已完成**；后续可加更细粒度 |
| P1-6 | WASM 每次 Send 实例化 module | 实例池或长驻 instance |
| P1-8 | Worker 无 OTEL 埋点 | 包装 Asynq handler |

#### P2 — 代码卫生 / DX

| ID | 问题 | 建议 |
|----|------|------|
| P2-1 | `gormstore` 按域拆分 | 已完成（`workflows.go` 等） |
| P2-4 | Asynq 重试可能重复发送 | 任务幂等键 / provider 侧 dedup |
| P2-5 | WASM 热更新需重启 | reload API 或 SIGHUP |
| P2-6 | 测试覆盖薄 | 补 `delivery`、`gormstore`、`wasm/runtime`、E2E |
| P2-7 | API Key prefix 桶内 O(n) bcrypt | 增加 key id 直查 |
| P2-8 | 云 WASM provider 缺真实签名示例 | 文档 + 参考插件 |

### 7.4 建议实施顺序

1. **P1-5** WASM permissions 强制  
2. **P1-3** Topic 异步 fan-out  
3. **P1-6** WASM 实例池  
4. **P2-6** 测试覆盖

## 8. 安全模型

| 资产 | 机制 |
|------|------|
| API Key | bcrypt + `key_prefix` 索引 |
| Integration 凭据 | AES-GCM；`ENCRYPTION_KEY` 32 字节 |
| WASM | 16MB 内存、10s 超时、路径校验 |
| Message API | 更新绑定 `(env_id, subscriber_pk)` |

## 9. 部署约束

1. **Migration**：仅 API 执行（GORM AutoMigrate）
2. **WASM 插件**：API **与 Worker** 必须访问**相同** `WASM_PLUGIN_DIR`
3. **Redis**：API 与 Worker 共用（Asynq + in-app SSE + throttle）
4. **DEV_MODE**：本地可 `DEV_MODE=true`；生产必须显式 `ENCRYPTION_KEY`
5. **Worker + API**：必须同时运行

## 10. 启动命令

```bash
make docker-up
# 放置 WASM 插件到 plugins/wasm/<name>/
DEV_MODE=true make build
DEV_MODE=true make run-api    # terminal 1
make run-worker               # terminal 2
```

## 11. Provider 来源

| 类型 | 来源 | 新增方式 |
|------|------|----------|
| email / sms / push / chat | `plugins/wasm/*/provider.wasm` | 新增 WASM 目录，不改核心 |
| in_app | 核心内置 | 无需 provider 插件 |

## 12. 环境变量速查

| 变量 | 默认 | 说明 |
|------|------|------|
| `STORE_TYPE` | `db` | `file` 或 `db` |
| `DB_DRIVER` | `postgres` | `postgres` / `mysql` / `sqlite` |
| `DATABASE_URL` | 按 driver | 数据库 DSN |
| `STORE_FILE_PATH` | `./data` | file 模式目录 |
| `REDIS_ADDR` | `localhost:6379` | Asynq + in-app SSE + throttle |
| `ENCRYPTION_KEY` | — | 32 字节 |
| `DEV_MODE` | `false` | 本地开发 |
| **`WASM_PLUGIN_DIR`** | `./plugins/wasm` | **WASM 插件根目录（必配）** |
| `HTTP_ADDR` | `:8080` | API 监听 |

## 13. 测试与 CI

```bash
make test
```

## 14. 变更日志

| 版本 | 日期 | 说明 |
|------|------|------|
| 3.3 | 2026-06-20 | 代码清理：dead code、installed_providers、精简 pkg/plugin |
| 3.2 | 2026-06-20 | WASM-only；Redis in-app SSE；ValidateConfig；health 探测 |
| 3.1 | 2026-06-20 | WASM-first 文档对齐 |
| 3.0 | 2026-06-20 | Herald 更名、GORM、OTEL |

## 15. 参考

- [WASM_PROVIDERS.md](WASM_PROVIDERS.md) — WASM 扩展开发
- [AGENTS.md](../AGENTS.md) — Cursor Agent 约束
- [CURSOR.md](CURSOR.md) — Cursor 工作流
