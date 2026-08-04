# 设计：后端 MCP Server

- 日期：2026-08-03
- 状态：已批准（用户授权实现）
- 范围：`backend/` 新增 MCP server，供外部 agent 通过 MCP 协议调用 RAG 检索与问答

## 背景

后端已提供 HTTP API（国别投资指南 RAG 问答）。为了让其他 agent（Claude、opencode、
其他 LLM 应用）能以**标准 MCP 协议**直接调用后端能力，新增一个 MCP server。

## 决策摘要

| 项 | 选择 | 理由 |
|----|------|------|
| 部署形态 | Go 内嵌（复用现有 service 层） | 无重复实现，复用 knowledge/assistant 已有逻辑 |
| 进程形态 | 独立 `cmd/mcp-server` 二进制 | agent 以子进程方式配置 MCP server，标准做法 |
| 传输 | stdio | MCP 标准，agent 本地拉起最通用 |
| SDK | `mark3labs/mcp-go` | Go 生态主流 MCP SDK（v0.57 活跃维护） |
| 鉴权 | 无（内部信任） | MCP server 视为服务账号，检索/问答无需用户上下文 |
| Tools | `knowledge_search` + `knowledge_ask` | 覆盖 agent 最核心的"查投资知识"需求 |
| 溯源 | ask 返回 `sources` 引用 | 让 agent 能回到知识块来源 |

## 架构

```
Agent (Claude/opencode/其他) ──stdio(MCP JSON-RPC)──> cmd/mcp-server
                                                         │
                                           复用 internal/domain 的 service 层
                                                         │
                                          DB + LLM/Embedding Provider
```

- 独立二进制 `cmd/mcp-server`，同一 module，复用 service 层
- stdio 传输（agent 子进程拉起）
- 无鉴权（内部信任，MCP server 相当于服务账号）

## MCP Tools

### knowledge_search — RAG 检索

- **输入**
  - `query` (string, 必填) — 检索问题
  - `country` (string, 可选) — 按国别过滤
  - `topK` (int, 可选, 默认 5) — 返回条数（1~20）
- **输出**：知识块列表 `[{ id, title, snippet }]`
- **实现**：`knowledge.Service.Search`

### knowledge_ask — LLM 问答

- **输入**
  - `question` (string, 必填) — 用户问题
  - `country` (string, 可选) — RAG 检索按国别过滤
- **输出**：`{ answer: string, sources: [{ title, snippet }] }`
- **实现**：复用 `assistant.Service.Generate`（RAG 上下文装配 + LLM 同步生成）

## 文件结构

```
backend/
├── cmd/mcp-server/
│   └── main.go              # MCP 入口：config.Load + database.Connect + 装配 service + stdio 启动
└── internal/mcp/
    ├── server.go            # MCP server 构建 + tools 注册
    ├── tools_search.go      # knowledge_search tool
    ├── tools_ask.go         # knowledge_ask tool
    └── server_test.go       # tools 单元测试（fake service）
```

领域层（knowledge/assistant）保持纯净，不依赖 MCP 包。

## 配置

复用现有 `.env` / 环境变量（`config.Load`）：
- `DATABASE_URL`、`LLM_*`、`EMBEDDING_*`、`JWT_SECRET`

## 测试与验证

1. **单测**：`internal/mcp/` 用 fake service 测两个 tool 的输入解析 + 输出格式
2. **E2E**：启动 `mcp-server`，用 MCP client 发 `tools/list` + `tools/call` 验证真实检索/问答
3. **文档**：`docs/mcp.md` 写接入说明（agent 配置示例）

## 接入示例（agent 配置）

Claude Desktop `claude_desktop_config.json`：

```json
{
  "mcpServers": {
    "invest-guide": {
      "command": "/path/to/backend/bin/mcp-server",
      "args": []
    }
  }
}
```

## 不在范围内

- HTTP/SSE 传输（streamable HTTP）— 后续如需远程 agent 再加
- MCP 鉴权 — 当前内部信任，后续可加固定 API Key
- conversation 完整 CRUD 暴露 — 当前只暴露检索 + 问答
