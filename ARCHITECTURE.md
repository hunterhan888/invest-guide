# 架构

Invest Guide 是一个面向**国别投资指南**的 Web 端 AI 问答平台。用户通过自然语言提问，
了解各国投资相关内容，回答基于精选知识库通过检索增强生成（RAG）得到。

产品以 **Web 应用优先**发布；后续渠道（微信等）计划通过同一 HTTP API 接入。

---

## 技术栈

| 组件 | 技术 |
|------|------|
| 后端框架 | Go 1.26 / Gin |
| 数据库 | PostgreSQL 18 + pgvector |
| 鉴权 | JWT |
| LLM 集成 | `LLMProvider` 接口 + 多 Provider 适配器；默认 SiliconFlow（OpenAI-compatible）；流式经 SSE |
| Embedding | SiliconFlow `Qwen/Qwen3-Embedding-0.6B`（OpenAI-compatible） |
| 向量检索 | pgvector |
| 异步任务 | 内存 goroutine pool + channel（生产可切 Redis） |
| 缓存 | 内存 LRU（生产可切 Redis） |
| 日志 | `log/slog`（结构化 JSON） |
| 前端 | React 19 + TypeScript（Vite），CSS Modules + 自研 primitives；依赖 react-markdown / zustand / swr / react-router / i18next |

---

## 项目根结构

```
invest-guide/
├── AGENT.md
├── ARCHITECTURE.md
├── README.md
├── .gitignore
├── Makefile                    # 常用命令入口
├── docker-compose.yml          # 本地开发（PostgreSQL + pgvector）
├── .env.example                # 环境变量模板
├── scripts/                    # 数据预处理与批量导入脚本（Python，非后端代码）
│   ├── parse_pdfs.py           # PDF → markdown（markitdown）
│   └── import_to_backend.py    # markdown → 批量调 HTTP API 灌库
├── docs/                       # 项目文档（与代码并行演进）
│   ├── backend/                # 后端相关文档
│   │   ├── api/                # API 契约（openapi.yaml，OpenAPI 3.0）+ README
│   │   ├── mcp-design.md       # MCP server 设计
│   │   └── task/               # 后端实施计划（Plan 1-4）
│   └── mcp.md                  # MCP server 接入说明（Agent 配置）
├── backend/
│   ├── go.mod
│   ├── cmd/
│   │   ├── server/             # HTTP API 服务入口（唯一 HTTP 入口）
│   │   │   └── main.go
│   │   └── mcp-server/         # MCP server 入口（stdio，供 Agent 调用）
│   │       └── main.go
│   ├── internal/
│   │   ├── platform/           # 基础层（config / database / response / middleware / router / logger / cache / taskqueue / llm / embedding）
│   │   ├── domain/             # 领域层（auth / conversation / knowledge / assistant / system / channel）
│   │   └── mcp/                # MCP tools 注册与实现（复用 domain service）
│   ├── migrations/             # golang-migrate 迁移文件（*.up.sql / *.down.sql）
│   └── tests/
│       └── e2e/                # 端到端测试（覆盖完整 HTTP 链路）
└── frontend/
    ├── package.json
    ├── tsconfig.json
    ├── vite.config.ts
    ├── index.html
    ├── public/                 # 静态资源（favicon 等，原样拷贝至 dist）
    └── src/
```

---

## 高层架构

```
┌──────────────────────────────────────────────────────────────┐
│                    Web Frontend (React + TS)                   │
│  pages · components · stores · hooks · i18n · api · SWR        │
├──────────────────────────────────────────────────────────────┤
│               HTTP /api/v1  (JSON + SSE)                       │
├──────────────────────────────────────────────────────────────┤
│                      Backend (Go + Gin)                        │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │  Middleware: CORS → RequestID → Logger → Recovery       │  │
│  │              → RateLimit → Auth → Handler                │  │
│  ├──────────┬──────────┬──────────┬──────────┬─────────────┤  │
│  │   auth   │conversa- │knowledge │assistant │   system     │  │
│  │          │  tion    │ (RAG)    │(LLM/SSE) │              │  │
│  ├──────────┴──────────┴──────────┴──────────┴─────────────┤  │
│  │  repository 层 (GORM)          │  Embedding Provider     │  │
│  │                                │  LLM Provider           │  │
│  │                                │  Task Queue (async)     │  │
│  ├────────────────────────────────┴─────────────────────────┤  │
│  │  PostgreSQL + pgvector          │  In-Memory Cache (LRU)  │  │
│  └──────────────────────────────────┴────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
```

依赖严格向下流动：handler → service → repository → database。
领域模块之间松耦合；跨模块调用通过接口进行。

---

## 后端模块职责

所有代码位于 `backend/internal/` 下，分两层：`platform/`（基础层）与 `domain/`（领域层）。

### 基础层（`internal/platform/`）

| 模块 | 职责 |
|------|------|
| `config` | 配置加载（环境变量、默认值、密钥）；`Config` 结构体 |
| `database` | 数据库连接、迁移执行器、事务辅助 |
| `response` | 统一的成功/错误响应封装类型 + 错误码映射 |
| `middleware` | CORS、日志/请求 ID、recovery、限流、鉴权 |
| `router` | 路由注册、中间件装配 |
| `logger` | `slog` 初始化、请求级日志上下文 |
| `cache` | LRU 缓存抽象 + 内存实现；生产可切 Redis |
| `taskqueue` | 轻量异步任务队列（goroutine pool + channel） |
| `llm` | `LLMProvider` 接口 + OpenAI-compatible 实现（Generate/Stream，含 SSE 消费） |
| `embedding` | `EmbeddingProvider` 接口 + OpenAI-compatible 实现（含指数退避重试）+ cosine 相似度 |

### 领域层（`internal/domain/`）

| 模块 | 职责 |
|------|------|
| `auth` | 用户注册/登录、JWT 签发与校验、当前用户提取 |
| `conversation` | 问答会话、消息持久化、SSE 流式编排 |
| `knowledge` | 国别指南文档管理、入库流水线（解析 → 分块 → 向量化）、向量检索 |
| `assistant` | `LLMProvider` 调用方、prompt 构建、RAG 上下文组装、Agent 编排（接口定义在 `platform/llm`、`platform/embedding`） |
| `system` | 健康检查、版本、模型列表 |
| `channel` | （预留）后续渠道适配器（微信 等）— 同一 HTTP API |

### 依赖规则

- ✅ 上层可依赖下层（service → repository → database）
- ✅ 领域模块可依赖 `config`、`database`、`response`、`logger`、`cache`、`taskqueue`
- ✅ 跨模块调用通过所属模块定义的接口进行
- ❌ 下层不得依赖上层
- ❌ 禁止循环依赖
- ❌ service 不得导入 `gin`

---

## 领域模块结构（后端）

每个领域模块遵循相同的内部组织：

```
backend/internal/domain/conversation/
├── route.go        # 模块路由注册
├── handler.go      # HTTP handler（薄层：绑定 → 调 service → 响应）
├── service.go      # 业务逻辑（业务规则的唯一归属）
├── repository.go   # 数据访问接口
├── repo_gorm.go    # 接口的 GORM 实现
└── model.go        # 领域模型 + GORM 实体
```

### 文件职责

- **handler.go** — 绑定/校验请求、调用 service、将错误映射为 HTTP 响应。
  不含业务逻辑。
- **service.go** — 全部业务规则与编排；只依赖接口（repository、LLM provider）；
  不导入 gin。
- **repository.go** — 仅接口（如 `ConversationRepository`）。
- **repo_gorm.go** — 具体的 GORM 实现。
- **model.go** — 领域模型与 JSON 契约结构体。

### Handler 形态（Go + Gin）

```go
// POST /api/v1/conversations
func (h *ConversationHandler) Create(c *gin.Context) {
    var req CreateConversationRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Fail(c, ErrInvalidInput)
        return
    }
    session, err := h.service.Create(c.Request.Context(), c.GetString("userID"), req)
    if err != nil {
        response.Fail(c, err)
        return
    }
    response.Ok(c, http.StatusCreated, session)
}
```

### 何时新建模块 vs. 扩展现有模块

**新建模块的条件：**
- 代表一个独立领域（有独立的数据模型与生命周期）
- 需要独立的路由前缀（如 `/api/v1/knowledge/...`）
- 与既有领域无强耦合

**扩展现有模块的条件：**
- 该功能是某个既有领域的子功能
- 共享相同的数据模型
- 路由是某个既有前缀的子路径

---

## 数据模型

### 核心实体关系

```
User 1───* Conversation 1───* Message
                                │
                          (引用检索到的知识块)
                                │
KnowledgeDoc 1───* KnowledgeChunk *───1 Embedding
```

### 实体定义

**User**

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string (UUID) | 主键 |
| `email` | string | 唯一，登录凭证 |
| `password_hash` | string | bcrypt |
| `display_name` | string | 昵称 |
| `created_at` | timestamp | |
| `updated_at` | timestamp | |

**Conversation**

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string (UUID) | 主键 |
| `user_id` | string (FK) | 所属用户 |
| `title` | string | 会话标题（首条消息摘要或用户命名） |
| `country` | string | 当前对话聚焦的国家（可选，辅助检索） |
| `created_at` | timestamp | |
| `updated_at` | timestamp | 最后活跃时间 |

**Message**

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string (UUID) | 主键 |
| `conversation_id` | string (FK) | 所属会话 |
| `role` | string | `user` / `assistant` |
| `content` | text | 消息正文 |
| `sources` | JSON | 引用的 `KnowledgeChunk` ID 列表 |
| `tokens_used` | int | LLM token 消耗（assistant 消息） |
| `created_at` | timestamp | |

**KnowledgeDoc**

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string (UUID) | 主键 |
| `title` | string | 文档标题（如"越南投资指南 2024"） |
| `country` | string | 国别标签 |
| `source_type` | string | `manual` / `upload` / `url` |
| `source_url` | string (nullable) | 当 `source_type=url` 时的来源 URL；其他类型为 null |
| `status` | string | `pending` → `processing` → `ready` / `failed` |
| `error_message` | string (nullable) | 当 `status=failed` 时的失败原因；其他状态为 null |
| `chunk_count` | int | 分块数量 |
| `created_at` | timestamp | |
| `updated_at` | timestamp | |

**KnowledgeChunk**

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string (UUID) | 主键 |
| `doc_id` | string (FK) | 所属文档 |
| `seq` | int | 块序号 |
| `content` | text | 块文本 |
| `embedding` | vector(1024) | pgvector 向量列（维度须与 `EMBEDDING_DIM` 一致） |
| `metadata` | JSON | 额外元数据（章节标题、页码等） |

> **注意**：SQLite 不支持 pgvector。开发环境中向量检索使用内存计算（brute-force cosine），
> 生产环境使用 PostgreSQL + pgvector。repository 接口屏蔽此差异。

---

## RAG 知识管线

> **PDF 预处理**：原始国别指南 PDF 由外部脚本 `scripts/parse_pdfs.py`（基于
> [markitdown](https://github.com/microsoft/markitdown)）预先转换为 markdown，存放在
> `data/parsed-text/`。后端入库流水线**不直接处理 PDF**，只接受纯文本 / markdown /
> HTML 字符串作为 `source_type=upload` 的 `content` 字段。批量灌库使用
> `scripts/import_to_backend.py` 调用 HTTP API。

### 入库流水线（异步）

```
KnowledgeDoc (status=pending)
    │
    ▼  taskqueue.Enqueue
┌──────────────┐
│  1. Parse    │  解析输入文本（markdown/HTML → 纯文本）；PDF 已由外部预处理
└──────┬───────┘
       ▼
┌──────────────┐
│  2. Chunk    │  按 token 数分块（512 tokens，10% overlap）
│              │  保留段落边界；记录章节元数据
└──────┬───────┘
       ▼
┌──────────────┐
│  3. Embed    │  调用 Embedding API 批量向量化
│              │  支持重试（指数退避，最多 3 次）
└──────┬───────┘
       ▼
┌──────────────┐
│  4. Store    │  写入 KnowledgeChunk + embedding
│              │  更新 KnowledgeDoc.status = ready
└──────────────┘
```

- 入库任务异步执行，API 立即返回 `202 Accepted` + 文档 ID
- 客户端通过 `GET /api/v1/knowledge-docs/{id}` 轮询状态
- 失败时 `status=failed`，记录错误信息，支持手动重试

### 检索流程

```
用户提问
    │
    ▼
┌──────────────┐
│ 1. Embed     │  将用户问题向量化
└──────┬───────┘
       ▼
┌──────────────┐
│ 2. Retrieve  │  向量相似度检索 Top-K（K=5）
│              │  可选：按 country 字段预过滤
└──────┬───────┘
       ▼
┌──────────────┐
│ 3. Rerank    │  （可选）按元数据、时间新鲜度加权重排
└──────┬───────┘
       ▼
┌──────────────┐
│ 4. Assemble  │  将检索块组装为 prompt 上下文
│              │  截断至 token 预算（context window 的 60%）
└──────┬───────┘
       ▼
   LLM 生成
```

---

## LLM Provider 抽象

> 接口定义位置：`LLMProvider` → `internal/platform/llm/`；`EmbeddingProvider` →
> `internal/platform/embedding/`（与 embedding 向量检索的 cosine 工具同包）。

### 接口定义

```go
type LLMProvider interface {
    // 同步生成
    Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error)
    // 流式生成（通过 channel 推送 delta）
    Stream(ctx context.Context, req GenerateRequest) (<-chan StreamChunk, error)
}

type EmbeddingProvider interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
}
```

### 多 Provider 支持

| Provider | 用途 | 配置 |
|----------|------|------|
| SiliconFlow（OpenAI-compatible） | 主 LLM（Qwen3 系列，默认） | `LLM_BASE_URL`、`LLM_API_KEY`、`LLM_MODEL` |
| OpenAI-compatible | 主 LLM（可切换到 OpenAI / DeepSeek / 通义等） | 同上，改 `LLM_BASE_URL` 与 `LLM_MODEL` |
| OpenAI-compatible | Embedding（Qwen3-Embedding-0.6B） | `EMBEDDING_BASE_URL`、`EMBEDDING_API_KEY`、`EMBEDDING_MODEL` |

所有 provider 遵循 OpenAI API 格式，通过 `base_url` 切换后端。本项目默认使用
SiliconFlow（`https://api.siliconflow.cn/v1`），其模型 ID 为带前缀形式（如
`Qwen/Qwen3-Embedding-0.6B`）。

**通用性**：实现不绑定任何特定 provider。`LLM_BASE_URL` / `EMBEDDING_BASE_URL` 与
API Key 由用户自行配置，可切换到任意 OpenAI-compatible 服务（SiliconFlow、OpenAI、
DeepSeek、通义等）。Embedding 请求会携带 `dimensions` 参数（取值 `EMBEDDING_DIM`），
强制模型按配置维度输出（SiliconFlow Qwen3 系列支持 64~1024）；不支持该参数的
provider 会忽略它，服务端仍以返回向量的实际维度为准校验。

### 容错策略

| 场景 | 策略 |
|------|------|
| 超时 | 请求超时 30s（流式 120s）；超时返回 `504 GATEWAY_TIMEOUT` |
| 限流（上游 429） | 指数退避重试，最多 3 次（1s → 2s → 4s） |
| 网络错误 | 同上重试 |
| 模型不可用 | 返回 `502 BAD_GATEWAY` + 错误码 `LLM_UNAVAILABLE` |
| Token 超限 | 检索前预估 token 数；超限时自动缩减上下文块数量 |

### 流式 SSE 并发模型

```
handler → service.Stream()
              │
              ├── goroutine: 调 LLMProvider.Stream()，读 channel
              │
              └── 主 goroutine: 遍历 channel → 写 SSE 响应
                    │
                    └── 客户端断开 → ctx.Done() → 取消 LLM 请求
```

- 每个流式请求启动 1 个 goroutine 消费 LLM 输出
- `context.Context` 贯穿全链路：客户端断连 → ctx 取消 → LLM 请求中止
- goroutine 泄漏防护：defer close channel + select ctx.Done()

---

## API 约定

### 基础路径与 REST 命名

```
/api/v1/{resources}                     # 集合操作（GET 列表，POST 创建）
/api/v1/{resources}/{id}                # 单项操作（GET, PATCH, DELETE）
/api/v1/{resources}/{id}/{subresources} # 嵌套资源
/api/v1/{resources}/{id}/{action}       # 动作操作（仅当 CRUD 无法表达时）
```

规则：
- 始终使用 `/api/v1/` 前缀
- 资源名与路径段用 kebab-case（`conversations`、`knowledge-docs`）
- 动作路由用动词（`stream`、`retry`、`stop`）
- JSON 字段名为 camelCase（Go 结构体 tag `json:"camelCase"`）

### 路由总表

| 方法 | 路径 | 鉴权 | 说明 |
|------|------|------|------|
| POST | `/api/v1/auth/register` | ❌ | 注册 |
| POST | `/api/v1/auth/login` | ❌ | 登录 |
| GET | `/api/v1/conversations` | ✅ | 会话列表（分页） |
| POST | `/api/v1/conversations` | ✅ | 新建会话 |
| GET | `/api/v1/conversations/{id}` | ✅ | 会话详情 |
| DELETE | `/api/v1/conversations/{id}` | ✅ | 删除会话 |
| GET | `/api/v1/conversations/{id}/messages` | ✅ | 消息历史 |
| POST | `/api/v1/conversations/{id}/messages` | ✅ | 发送提问（创建用户消息，返回 `201` + `messageId`，触发异步回答生成） |
| GET | `/api/v1/conversations/{id}/messages/{messageId}/stream` | ✅ | SSE 流式回答（订阅该用户消息对应的助手回答） |
| GET | `/api/v1/knowledge-docs` | ✅ | 文档列表 |
| POST | `/api/v1/knowledge-docs` | ✅ | 上传文档（异步入库，返回 `202` + 文档 ID） |
| GET | `/api/v1/knowledge-docs/{id}` | ✅ | 文档状态 |
| DELETE | `/api/v1/knowledge-docs/{id}` | ✅ | 删除文档 |
| POST | `/api/v1/knowledge-docs/{id}/retry` | ✅ | 重新触发失败文档的入库流水线 |
| GET | `/api/v1/system/health` | ❌ | 健康检查 |
| GET | `/api/v1/system/version` | ❌ | 版本信息 |
| GET | `/api/v1/system/models` | ❌ | 可用 LLM / Embedding 模型列表 |

### 统一响应

**成功**（`data` 与 `message` 为 null 时省略）：

```json
{ "success": true, "data": { }, "message": "optional" }
```

**错误：**

```json
{ "success": false, "error": "面向人类的错误信息", "code": "ERROR_CODE" }
```

### HTTP 状态码映射

| 错误 | 状态码 | Code | 用途 |
|------|--------|------|------|
| `ErrInvalidInput` | 400 | INVALID_INPUT | 请求格式错误 / 校验失败 |
| `ErrUnauthorized` | 401 | UNAUTHORIZED | 缺失或过期的 token |
| `ErrForbidden` | 403 | FORBIDDEN | 无权限 |
| `ErrNotFound` | 404 | NOT_FOUND | 资源不存在 |
| `ErrConflict` | 409 | CONFLICT | 状态冲突 |
| `ErrRateLimited` | 429 | RATE_LIMITED | 触发限流 |
| `ErrInternal` | 500 | INTERNAL_ERROR | 未预期的内部错误 |
| 上游失败 | 502 | BAD_GATEWAY | LLM / Embedding / 上游服务失败 |
| 上游超时 | 504 | GATEWAY_TIMEOUT | LLM 响应超时 |

### 分页

基于偏移量，请求 `?page=1&pageSize=20`（页码从 1 开始）：

```json
{ "items": [ ], "total": 100, "hasMore": true }
```

### 流式（SSE）

流式回答使用 Server-Sent Events，端点
`GET /api/v1/conversations/{id}/messages/{messageId}/stream`（订阅该用户消息对应的
助手回答流）：

**正常流程**（事件顺序）：

```
event: heartbeat
data: {}

event: sources
data: {"chunks": [{"id": "...", "title": "...", "snippet": "..."}]}

event: message
data: {"delta": "文本片段"}

event: done
data: {"messageId": "...", "tokensUsed": 350}
```

**错误流程**（终止性失败，`error` 之后不再有 `done`）：

```
event: error
data: {"code": "LLM_TIMEOUT", "message": "..."}
```

| 事件 | 用途 |
|------|------|
| `heartbeat` | 每 15 秒发送，保持连接活跃，防止代理超时断连 |
| `sources` | 检索引用的知识块（在首次 `message` 之前一次性发送） |
| `message` | 增量文本片段（可重复多次） |
| `done` | 流正常结束（最终助手消息 ID + token 消耗） |
| `error` | 终止性失败（出现后连接关闭，不再发送 `done`） |

**断线重连**：客户端可通过 `Last-Event-ID` header 重连，服务端从上次断点续传
（仅限当前会话的最近一条消息流，已完成的流不可重放）。

选用 SSE 而非 WebSocket，是为了让任何 HTTP 客户端（包括后续的微信适配器）
都能消费。

---

## 数据层

### Repository 模式

所有数据库访问都经接口：

```go
type ConversationRepository interface {
    Get(ctx context.Context, id string) (*Conversation, error)
    Create(ctx context.Context, conv *Conversation) error
    Update(ctx context.Context, id string, params UpdateParams) error
    Delete(ctx context.Context, id string) error
    ListByUser(ctx context.Context, userID string, page, pageSize int) ([]*Conversation, int64, error)
}

type KnowledgeChunkRepository interface {
    SearchByVector(ctx context.Context, vec []float32, topK int, country string) ([]*KnowledgeChunk, error)
    BatchCreate(ctx context.Context, chunks []*KnowledgeChunk) error
}
```

规则：
- service 只依赖接口，绝不依赖 GORM 类型
- 具体实现：`gormConversationRepository`、`gormKnowledgeChunkRepository`
- 向量检索在开发环境用内存 brute-force cosine，生产用 pgvector `<=>` 算子
- 集成测试使用内存 SQLite 数据库

### 迁移

- 文件位于 `backend/migrations/`，格式 `NNNN_description.up.sql` /
  `NNNN_description.down.sql`（golang-migrate）
- 启动时自动执行迁移（或通过 `make migrate`）
- 结构变更必须经迁移文件 — 禁止手动改库
- pgvector 扩展在第一个需要向量列的迁移中 `CREATE EXTENSION IF NOT EXISTS vector`

### 错误传播

```
repository (ErrNotFound, ErrConflict, ...)
      ↓ errors.Is
service   (用 %w 包装)
      ↓
handler   (response.Fail → 状态码 + ErrorResponse)
```

映射：`ErrNotFound` → 404 · `ErrConflict` → 409 · `ErrInvalidInput` → 400 ·
其余 → 500（内部细节绝不泄露）。

---

## 缓存策略

| 缓存层 | 内容 | TTL | 失效策略 |
|--------|------|-----|----------|
| 内存 LRU | Embedding 结果（按文本 hash） | 1h | 容量上限 1000 条 |
| 内存 LRU | 热门问题检索结果 | 10min | 文档更新时清除 |
| HTTP 响应头 | 静态资源 | — | 由前端 CDN/浏览器管理 |
| SWR（前端） | API 列表数据 | — | stale-while-revalidate |

- 缓存接口定义在 `internal/platform/cache/`，开发用内存 LRU，生产可切 Redis
- **Embedding 结果缓存已启用**（`cache.NewLRU(1000, 1h)`，经 `knowledge.NewEmbeddingCache` 接入 HTTP 与 MCP 两个入口）；"热门问题检索结果"缓存为规划项，尚未实现
- 知识文档状态变更时，主动失效相关缓存
- LLM 生成结果**不缓存**（每次回答可能有上下文差异）

---

## 依赖注入

唯一装配点在 `backend/cmd/server/main.go`：

```go
func main() {
    cfg := config.Load()            // 1. 加载配置
    logger.Init(os.Stdout, cfg.LogLevel)
    db := database.Connect(cfg.DatabaseURL)     // 2. 连接数据库
    database.AutoMigrate(db, &auth.User{}, &knowledge.KnowledgeDoc{},
        &knowledge.KnowledgeChunk{}, &conversation.Conversation{}, &conversation.Message{})
    cacheInst := cache.NewLRU(1000, time.Hour)  // 3. 缓存
    taskQ := taskqueue.NewGoroutinePool(4, 16)  // 4. 任务队列
    // 5. 各领域 service / handler 显式装配
    deps := &router.Deps{...}                   // 6. 构建全部依赖
    r := router.New(deps)                       // 7. 注册路由 + 中间件
    srv := &http.Server{Addr: cfg.HTTPAddr(), Handler: r}
    go srv.ListenAndServe()                     // 8. 启动服务
    // 9. 信号监听 + 优雅关闭（srv.Shutdown、taskQ.Close）
}
```

- 装配全部在 `main.go` 中显式完成：DB → repository → service → handler
- handler 通过各模块的 `Handler` 结构体接收依赖（构造器注入）
- service 不得自行构建其依赖
- `taskqueue` 在启动时拉起 worker goroutine，退出时经 `taskQ.Close(ctx)` 优雅关闭

---

## 配置管理

### 环境变量

| 变量 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `PORT` | ❌ | `8080` | HTTP 监听端口 |
| `DATABASE_URL` | ❌ | `sqlite://dev.db` | 数据库连接串 |
| `JWT_SECRET` | ✅ | — | JWT 签名密钥（生产必填） |
| `JWT_EXPIRY` | ❌ | `24h` | Token 有效期 |
| `LLM_BASE_URL` | ❌ | `https://api.siliconflow.cn/v1` | LLM API 地址（OpenAI-compatible） |
| `LLM_API_KEY` | ❌ | — | LLM API 密钥（缺失时允许启动，调用 LLM 返回 502） |
| `LLM_MODEL` | ❌ | `Qwen/Qwen3.5-27B` | 默认模型 |
| `LLM_TIMEOUT` | ❌ | `30s` | 非流式请求超时 |
| `LLM_STREAM_TIMEOUT` | ❌ | `120s` | 流式请求超时 |
| `EMBEDDING_BASE_URL` | ❌ | 同 `LLM_BASE_URL` | Embedding API 地址（OpenAI-compatible） |
| `EMBEDDING_API_KEY` | ❌ | 同 `LLM_API_KEY` | Embedding API 密钥（缺失时调用返回 502） |
| `EMBEDDING_MODEL` | ❌ | `Qwen/Qwen3-Embedding-0.6B` | Embedding 模型 |
| `EMBEDDING_DIM` | ❌ | `1024` | 向量维度（须与所选模型输出维度一致） |
| `CORS_ORIGINS` | ❌ | `http://localhost:5173` | 允许的前端来源（逗号分隔） |
| `LOG_LEVEL` | ❌ | `info` | 日志级别（debug/info/warn/error） |
| `LOG_FILE` | ❌ | — | 日志文件路径（相对 backend 工作目录）；留空输出到 stdout |
| `RATE_LIMIT_API` | ❌ | `60` | API 限流（次/分钟） |
| `RATE_LIMIT_SENSITIVE` | ❌ | `20` | 敏感操作限流（次/分钟） |

- `.env` 文件用于本地开发（项目根），已被 gitignore；`config.Load()` 启动时自动从
  当前目录向上查找并加载 `.env`（**不覆盖**已存在的环境变量，shell export 优先）
- 生产环境通过环境变量注入，绝不将密钥写入代码或配置文件
- `config.Load()` 启动时校验必填项（`JWT_SECRET`），缺失即 fail-fast；
  LLM / Embedding API Key 可选，缺失时允许启动但对应调用返回 502

---

## 安全模型

### 中间件栈（由外到内）

```
CORS
  → RequestID
    → RequestLogger（结构化日志：request_id/method/path/status/耗时，slog JSON）
      → Recovery（不向客户端暴露堆栈）
        → 限流（按组）
          → 鉴权（按路由组启用，非全局）
            → Handler
```

### JWT

- 算法：HS256；有效期：24h
- 载荷：`user_id`、`exp`、`iat`、`iss`（`investguide`）
- 密钥来自 `JWT_SECRET` 环境变量（绝不硬编码）
- 提取方式：`Authorization: Bearer <token>`
- 公开路由（login、register、health、version）跳过鉴权中间件

### 限流

| 级别 | 限制 | 时间窗 | 范围 |
|------|------|--------|------|
| 鉴权 | 5 次失败 | 15 分钟 | 登录/注册（按 IP） |
| API | 60 次请求 | 1 分钟 | 通用端点（按用户/IP） |
| 敏感 | 20 次请求 | 1 分钟 | 流式、知识上传（按用户） |

### 其他

- CORS 仅允许配置的来源（`CORS_ORIGINS`）
- 所有请求结构体通过 gin binding tag 做输入校验
- 密钥来自环境变量；`.env` 已被 gitignore
- 错误响应绝不暴露堆栈或内部细节
- 密码使用 bcrypt（cost=12）存储

---

## 可观测性

### 日志

- 使用 `log/slog`，输出 JSON 格式
- 每条请求日志携带 `request_id`（由 RequestID 中间件注入）
- 日志级别：`debug`（开发）/ `info`（生产）/ `warn` / `error`
- 敏感信息（API key、密码、token）绝不写入日志

```go
logger.Info("conversation created",
    slog.String("request_id", reqID),
    slog.String("user_id", userID),
    slog.String("conversation_id", convID),
)
```

### Metrics（预留）

- 端点 `GET /api/v1/system/metrics`（Prometheus 格式，需鉴权）
- 关键指标：
  - `http_requests_total{method, path, status}` — 请求计数
  - `http_request_duration_seconds{method, path}` — 请求延迟
  - `llm_tokens_used_total{model}` — LLM token 消耗
  - `knowledge_chunks_indexed_total` — 知识入库进度

### 请求追踪

- `RequestID` 中间件为每个请求生成 UUID，写入响应头 `X-Request-ID`
- 日志、错误响应均携带 `request_id`，便于跨层关联

---

## 测试策略

| 层 | 位置 | 数据库 | 用途 |
|----|------|--------|------|
| 单元测试 | `*_test.go` 内联 | 无（mock） | 函数级逻辑 |
| 集成测试 | `backend/internal/domain/<module>/` | 内存 SQLite | Handler + service + repository |
| E2E | `backend/tests/e2e/` | 内存 SQLite | 完整 HTTP 链路 |

- 所有需要数据库的测试使用真实 GORM repo + 内存 SQLite 数据库
- LLM 调用一律 mock（`fakeLLMProvider`，输出确定）
- Embedding 调用一律 mock（`fakeEmbeddingProvider`，返回固定向量）
- 优先表驱动测试；每个断言都验证真实行为（禁止模糊断言）
- 测试失败处理：修实现而非改测试；仅当预期行为确实变更时才更新测试

---

## 前端架构

```
frontend/src/
├── api/          # 类型化 HTTP 客户端 — 每个后端领域一个模块，附带 JWT；含 API 契约类型（types.ts）
├── primitives/   # 自研基础组件（Button, Input, Textarea, Modal, Dropdown, Tooltip, Toast, DisclosureRow, Pill, Icon）+ 内联 SVG 图标（icons.tsx）
├── layout/       # 应用布局（AppFrame, Sidebar, DetailsPanel, UserMenu, AppLayout）
├── pages/        # 路由页面（LoginPage, RegisterPage, HomePage, ConversationPage）
├── features/     # 页面级功能组件（auth 表单、conversation TurnStatus、home HomeComposer）
├── components/   # 跨功能共享组件（conversation MarkdownRenderer, MessageBubble, ...）
├── hooks/        # 自定义 hooks（useConversation, useSSEStream, ...）
├── stores/       # Zustand store（auth, conversation, ui, theme）
├── theme/        # 主题（ThemeProvider + zustand themeStore 持久化）
├── i18n/         # 语言配置 + 资源文件（zh-CN, en-US）
├── styles/       # 设计 token（--dsw-*）+ 全局样式（tokens.css, base.css, scrollbar.css）
├── router.tsx    # 路由定义 + 鉴权守卫
└── main.tsx
```

### 设计 token 与主题

- 颜色、间距、动效统一使用 `frontend/src/styles/tokens.css` 定义的 `--dsw-*` 设计 token
- 浅色定义在 `body`，暗色定义在 `body[data-ds-dark-theme]` — 双主题在 CSS 侧只有这一个切换点
- `theme/ThemeProvider` 维护 `body[data-ds-dark-theme]` 属性，主题偏好由 zustand
  `themeStore` 持久化到 `localStorage`（默认跟随系统 `prefers-color-scheme`）
- 禁止硬编码色值；全局样式只在 `styles/` 下

### 数据获取

- **SWR** 用于列表和详情数据的获取与缓存
- API 客户端（`api/`）是唯一接触 HTTP 的地方；页面绝不直接调用 `fetch`
- 乐观更新：发送消息后立即在 UI 显示，SSE 回填实际内容

### SSE 流式消费

- 流程：先 `POST /conversations/{id}/messages` 拿到 `messageId`，再以
  `fetch` + `ReadableStream` 打开
  `GET /conversations/{id}/messages/{messageId}/stream` 订阅助手回答
  （EventSource 无法发送 header，而 JWT 需要 header）
- `useSSEStream` hook 封装：自动重连（指数退避）、心跳超时检测、错误处理
- 断线重连时携带 `Last-Event-ID`（仅限该条消息流尚未完成时续传）

### 错误处理

- 全局 `ErrorBoundary` 捕获 React 渲染错误
- API 错误经 SWR 的 `onError` 统一处理：401 → 跳转登录、其他 → Toast 提示
- SSE 错误事件在会话窗口内 inline 显示，提供"重试"按钮

### 状态管理

| Store | 职责 |
|-------|------|
| `authStore` | JWT token、用户信息、登录/登出 |
| `conversationStore` | 当前会话列表 + 活跃会话 ID |
| `uiStore` | 侧边栏折叠、主题、语言等 UI 状态 |

---

## 新增功能

**第 1 步** — 创建 `backend/internal/domain/<module>/`，含 `route.go`、`handler.go`、
`service.go`、`repository.go`、`repo_gorm.go`、`model.go`。

**第 2 步** — 在 `backend/internal/platform/router/router.go` 注册路由；如需鉴权则附加
鉴权中间件。

**第 3 步** — 新增 GORM model + 迁移文件 `backend/migrations/NNNN_....sql`。

**第 4 步** — 实现 repository 接口；将 service 接入 `server.NewDeps`。

**第 5 步** — 补充集成测试（内存 SQLite）与 E2E 测试覆盖 HTTP 流程。

**第 6 步** — 若 API 契约变更，同步更新 `frontend/src/api/` 下的契约类型与客户端。

**第 7 步** — 若模块边界或约定变更，更新 **ARCHITECTURE.md**。

### 检查清单

- [ ] 模块遵循标准结构（route/handler/service/repository/model）
- [ ] 依赖方向正确；无循环
- [ ] 已定义 repository 接口；GORM 实现独立
- [ ] 路由使用 `/api/v1/` 前缀且为 kebab-case
- [ ] 使用统一的成功/错误响应封装
- [ ] 任何结构变更都附迁移文件
- [ ] 含测试；LLM / Embedding 调用已 mock
- [ ] 契约变更时同步前端 types + API 客户端
- [ ] ARCHITECTURE.md / AGENT.md 已同步
