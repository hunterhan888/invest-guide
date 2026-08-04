# MCP Server 接入说明

Invest Guide 后端通过 **MCP（Model Context Protocol）** 暴露 RAG 检索与问答能力，
供其他 AI Agent（Claude、opencode、Cursor 等）以标准协议调用。

设计文档：[`docs/backend/mcp-design.md`](./mcp-design.md)

## 提供的 Tools

| Tool | 说明 | 输入 | 输出 |
|------|------|------|------|
| `knowledge_search` | 检索国别投资指南知识库 | `query`(必填)、`country`(可选)、`topK`(可选,1~20) | 命中的知识片段 `[{id,title,snippet}]` |
| `knowledge_ask` | 基于知识库回答投资问题（RAG + LLM） | `question`(必填)、`country`(可选) | `{answer, sources:[{title,snippet}]}` |

## 构建

```bash
make backend-mcp        # 编译到 backend/bin/mcp-server
```

## 运行前配置

MCP server 与 HTTP 服务共享 `.env`（项目根），需要：

| 配置 | 用途 |
|------|------|
| `DATABASE_URL` | 知识库（PostgreSQL，需已导入国别指南文档） |
| `EMBEDDING_*` | Embedding（检索向量化） |
| `LLM_*` | LLM（knowledge_ask 生成回答） |

## Agent 接入配置

MCP server 通过 **stdio** 与 Agent 通信（Agent 以子进程方式拉起）。

### Claude Desktop

编辑 `claude_desktop_config.json`：

```json
{
  "mcpServers": {
    "invest-guide": {
      "command": "/绝对路径/invest-guide/backend/bin/mcp-server",
      "args": []
    }
  }
}
```

> 注意：路径必须是绝对路径。`.env` 中的配置会自动加载，
> 无需在 config 里重复传环境变量。

### opencode

在 `opencode.json` 的 `mcp` 配置中添加：

```json
{
  "mcp": {
    "invest-guide": {
      "type": "stdio",
      "command": ["/绝对路径/invest-guide/backend/bin/mcp-server"],
      "enabled": true
    }
  }
}
```

### Cursor / 其他支持 MCP 的工具

在 MCP 配置中新增 stdio server，command 指向 `backend/bin/mcp-server` 即可。

## 验证接入

Agent 配置后，向 Agent 提问：

- "查询越南的企业所得税率" → 应调用 `knowledge_search`
- "越南的外资准入政策是什么？" → 应调用 `knowledge_ask` 并返回带来源的回答

也可用命令行直接验证：

```bash
# 启动 MCP server（stdio，阻塞等待客户端）
backend/bin/mcp-server

# 用 MCP 客户端工具（如 mcp-inspector / 自写 client）连接测试
```

## 常见问题

| 问题 | 排查 |
|------|------|
| Agent 报找不到 command | 检查路径是否绝对、二进制是否已 `make backend-mcp` 编译 |
| `knowledge_search` 返回空 | 知识库无数据或 embedding 未配好；先确认 HTTP API 的 `POST /knowledge-docs` 能入库 |
| `knowledge_ask` 报 `context deadline exceeded` | LLM 响应超时；检查 `.env` 的 `LLM_TIMEOUT`（同步问答建议 ≥60s）与 `LLM_API_KEY` |
