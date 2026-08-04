# Invest Guide

面向**国别投资指南**的 Web 端 AI 问答平台。用户通过自然语言提问，回答基于精选知识库通过检索增强生成（RAG）得到。

技术栈与架构见 [ARCHITECTURE.md](ARCHITECTURE.md)；贡献规范见 [AGENT.md](AGENT.md)。

## 一键启动（Docker，推荐）

```bash
cp .env.example .env           # 编辑填入 JWT_SECRET 和 LLM_API_KEY
docker compose up -d           # 启动全部服务（postgres + backend + frontend）
```

| 服务 | 地址 |
|------|------|
| 前端 | http://localhost:5173 |
| 后端 API | http://localhost:8080/api/v1/system/health |
| PostgreSQL | localhost:5432（`invest:invest@investguide`） |

```bash
docker compose logs -f          # 查看全部日志
docker compose logs -f backend  # 只看后端日志
```

> **热重载已启用**：修改 `backend/` 下 Go 源码 → Air 自动重编译后端；修改 `frontend/src/` → Vite HMR 即时更新前端。无需手动重启容器。

> **双模式**：Docker 和本地手动启动共用同一份 `.env`，互不干扰。如需切回本地开发模式，`docker compose down` 后照常使用 `make backend-dev` / `cd frontend && bun run dev`。

## 本地快速启动（不依赖 Docker）

### 1. 后端

```bash
cp .env.example .env      # 填 JWT_SECRET 与 LLM_API_KEY
docker compose up -d postgres   # 本地 PostgreSQL（或改用 SQLite）
make backend-dev          # 默认 http://localhost:8080
```

验证：

```bash
curl http://localhost:8080/api/v1/system/health
# {"success":true,"data":{"status":"ok"}}
```

完整说明（环境变量、MCP、常见问题）见 [backend/README.md](backend/README.md)。

### 2. 前端

```bash
cd frontend
cp .env.example .env
bun install
bun run dev               # http://localhost:5173
```

**开发模式默认对接真实后端**（`.env.development`：`VITE_USE_MOCK=false`，Vite 将 `/api/v1` 代理到 `VITE_API_PROXY_TARGET`，默认 `http://localhost:8080`）。

- `VITE_USE_MOCK=true`：使用内置 Mock（登录/会话/SSE 流式均可用），无需后端
- `VITE_USE_MOCK=false`：对接真实后端；调整 `frontend/.env.development` 中的 `VITE_API_PROXY_TARGET` 指向后端地址
- 测试运行在 mock 模式（`frontend/.env` 保持 `VITE_USE_MOCK=true`），不依赖后端

## 常用命令（Makefile）

| 命令 | 说明 |
|------|------|
| `make backend-dev` | 启动后端开发服务器 |
| `make backend-test` | 后端测试 + 覆盖率 |
| `make backend-mcp` | 编译 MCP server 到 `backend/bin/mcp-server` |
| `make frontend-dev` | 启动前端 dev server |
| `make test` | 后端测试 |
| `docker compose up -d postgres` | 启动本地 PostgreSQL |

## 质量门禁

```bash
cd backend && gofmt -l . && go vet ./... && go test ./...
cd frontend && bun run lint && bun run format && bunx tsc --noEmit && bun run test
```

## 相关文档

| 文档 | 用途 |
|------|------|
| [ARCHITECTURE.md](ARCHITECTURE.md) | 系统架构、模块职责、API 约定、安全与测试策略 |
| [AGENT.md](AGENT.md) | 贡献者与 Agent 规范 |
| [docs/backend/api/openapi.yaml](docs/backend/api/openapi.yaml) | API 契约（OpenAPI 3.0） |
| [docs/mcp.md](docs/mcp.md) | MCP server 接入说明 |

MVP 范围与任务拆分见 `docs/frontend/task.md`，实施计划见 `docs/frontend/plan.md`。
