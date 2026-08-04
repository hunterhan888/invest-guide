# Invest Guide 后端

Go + Gin 实现的国别投资指南 RAG 问答 API 服务。

## 技术栈

- Go 1.26 / Gin
- PostgreSQL 18 + pgvector（生产）/ SQLite（本地可选）
- JWT 鉴权
- SiliconFlow（OpenAI-compatible）LLM + Embedding
- `log/slog` 结构化日志

## 环境要求

| 依赖 | 版本 | 用途 |
|------|------|------|
| Go | 1.26+ | 编译运行 |
| Docker | 任意 | 本地启动 PostgreSQL（推荐） |
| SiliconFlow API Key | — | LLM 与 Embedding 调用 |

> 没有 Docker 也可以跑：把 `DATABASE_URL` 改成 `sqlite://dev.db`，但 SQLite 仅用于本地轻量验证。

## 快速启动

### 第 1 步：配置 `.env`

项目根目录的 `.env` 是唯一配置来源（后端启动时自动加载，不覆盖已 export 的环境变量）。

```bash
# 项目根目录
cp .env.example .env
```

编辑 `.env`，至少填 2 项必填/核心配置：

```bash
JWT_SECRET=换成随机长字符串（生产必须，本地可用默认值）
LLM_API_KEY=sk-你的SiliconFlow Key
```

其余保持默认即可（默认指向 SiliconFlow：Embedding = Qwen3-Embedding-0.6B；LLM 按 `.env.example` 为
deepseek-ai/DeepSeek-V4-Flash，代码内置默认 `Qwen/Qwen3.5-27B`，均可自行改 `LLM_MODEL`）。

### 第 2 步：启动 PostgreSQL（可选，本地默认配置需要）

```bash
docker compose up -d postgres
```

验证：

```bash
docker compose ps   # 应显示 invest-guide-postgres-1 (healthy)
```

### 第 3 步：启动后端

```bash
make backend-dev
```

看到如下日志即启动成功（端口来自 `.env` 的 `PORT`，默认 8080）：

```
{"level":"INFO","msg":"starting invest-guide backend","port":"8080"}
```

### 第 4 步：验证

```bash
curl http://localhost:8080/api/v1/system/health
# {"success":true,"data":{"status":"ok"}}
```

## 常用命令

| 命令 | 说明 |
|------|------|
| `make backend-dev` | 启动开发服务器（读 `.env`） |
| `make backend-test` | 运行全部测试 + 覆盖率 |
| `make backend-build` | 编译二进制到 `backend/bin/server` |
| `make backend-mcp` | 编译 MCP server 到 `backend/bin/mcp-server` |
| `make backend-vet` | 静态检查 |
| `make backend-fmt` | 检查格式化 |
| `docker compose up -d postgres` | 启动数据库 |
| `docker compose down` | 停止数据库 |

## MCP Server

后端内置 MCP server，供其他 AI Agent（Claude / opencode / Cursor）以标准协议调用
RAG 检索与问答：

```bash
make backend-mcp   # 编译到 backend/bin/mcp-server
```

- Tools：`knowledge_search`（检索知识库）、`knowledge_ask`（RAG 问答）
- 传输：stdio（Agent 以子进程方式拉起）
- 配置：复用 `.env`（`DATABASE_URL` / `EMBEDDING_*` / `LLM_*`）

详见 [`../docs/mcp.md`](../docs/mcp.md)。

## 配置项（.env）

| 变量 | 必填 | 默认 | 说明 |
|------|------|------|------|
| `JWT_SECRET` | ✅ | — | JWT 签名密钥（生产必须换） |
| `JWT_EXPIRY` | ❌ | `24h` | Token 有效期 |
| `PORT` | ❌ | `8080` | HTTP 监听端口 |
| `DATABASE_URL` | ❌ | `postgres://invest:invest@localhost:5432/investguide?sslmode=disable` | 数据库连接串 |
| `LLM_BASE_URL` | ❌ | `https://api.siliconflow.cn/v1` | LLM API 地址（OpenAI-compatible） |
| `LLM_API_KEY` | ❌ | — | LLM 密钥（缺失时启动但不支持 LLM 调用） |
| `LLM_MODEL` | ❌ | `Qwen/Qwen3.5-27B` | LLM 模型（`.env.example` 默认 `deepseek-ai/DeepSeek-V4-Flash`） |
| `LLM_TIMEOUT` | ❌ | `30s` | 非流式请求超时 |
| `LLM_STREAM_TIMEOUT` | ❌ | `120s` | 流式请求超时 |
| `EMBEDDING_BASE_URL` | ❌ | 同 `LLM_BASE_URL` | Embedding 地址 |
| `EMBEDDING_API_KEY` | ❌ | 同 `LLM_API_KEY` | Embedding 密钥 |
| `EMBEDDING_MODEL` | ❌ | `Qwen/Qwen3-Embedding-0.6B` | Embedding 模型 |
| `EMBEDDING_DIM` | ❌ | `1024` | 向量维度（须与模型支持一致） |
| `CORS_ORIGINS` | ❌ | `http://localhost:5173` | 允许的前端来源 |
| `LOG_LEVEL` | ❌ | `info` | 日志级别（debug/info/warn/error） |
| `LOG_FILE` | ❌ | — | 日志文件路径（相对 backend 工作目录，如 `logs/backend.log`）；留空输出到 stdout |
| `RATE_LIMIT_API` | ❌ | `60` | API 限流（次/分钟） |
| `RATE_LIMIT_SENSITIVE` | ❌ | `20` | 敏感操作限流（次/分钟） |

> API Key 均为可选：缺失时服务能启动，但调用 LLM / Embedding 会返回 `502`。

## API 文档

接口契约以 **OpenAPI 3.0** 为准：[`../docs/backend/api/openapi.yaml`](../docs/backend/api/openapi.yaml)

可用任意 OpenAPI 工具（Swagger UI / Redoc / Bruno / Postman）查看或导入。

## 目录结构

```
backend/
├── cmd/server/main.go          # 唯一入口（装配 + 优雅退出）
├── internal/
│   ├── platform/               # 基础层：config/logger/response/database/cache/taskqueue/middleware/router/llm/embedding
│   └── domain/                 # 领域层：auth/knowledge/conversation/assistant/system
├── migrations/                 # golang-migrate 迁移文件（生产用）
└── tests/e2e/                  # 端到端测试
```

## 本地开发说明

- **数据库**：开发默认连 PostgreSQL（需先 `docker compose up -d postgres`）；想用 SQLite 改 `DATABASE_URL=sqlite://dev.db`
- **表结构**：开发环境启动时用 GORM `AutoMigrate` 自动建表；生产用 `migrations/` 下的 SQL 迁移
- **pgvector 扩展**：生产建表前需 `CREATE EXTENSION IF NOT EXISTS vector`（见 `migrations/0001_init.up.sql`）

## 常见问题

**启动报 `connect db failed: connection refused`**
→ PostgreSQL 没启动，先 `docker compose up -d postgres`，或改 `.env` 用 SQLite。

**上传文档后 status 一直是 `processing` 或变 `failed`**
→ Embedding 调用失败。检查 `EMBEDDING_API_KEY` / `EMBEDDING_BASE_URL` 是否配好，看服务端日志确认具体错误。

**提问时 SSE 返回 502**
→ LLM 调用失败。检查 `LLM_API_KEY`，或 `LLM_MODEL` 是否 SiliconFlow 支持的模型。
