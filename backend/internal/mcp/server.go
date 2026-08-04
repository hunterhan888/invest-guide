package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Service 是 MCP tools 依赖的最小业务能力集合
type Service struct {
	Search   SearchFunc
	Generate GenerateFunc
}

// SearchFunc 抽象 knowledge.Service.Search
type SearchFunc func(ctx context.Context, query, country string, topK int) ([]SearchHit, error)

// GenerateFunc 抽象 assistant.Service.Generate（RAG 问答，返回回答 + 引用来源）
type GenerateFunc func(ctx context.Context, question, country string) (answer string, sources []Source, err error)

// SearchHit 是检索命中的知识块
type SearchHit struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
}

// Source 是问答引用的知识来源
type Source struct {
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
}

// NewServer 构建 MCP server，注册 knowledge_search 与 knowledge_ask 两个 tool
func NewServer(svc Service) *server.MCPServer {
	s := server.NewMCPServer(
		"invest-guide",
		"0.1.0",
		server.WithToolCapabilities(true),
	)
	s.AddTool(newSearchTool(), searchHandler(svc.Search))
	s.AddTool(newAskTool(), askHandler(svc.Generate))
	return s
}

func newSearchTool() mcp.Tool {
	return mcp.NewTool("knowledge_search",
		mcp.WithDescription("检索国别投资指南知识库，返回与问题相关的知识片段。"+
			"用于查找投资相关的法律、税务、行业准入、园区、外汇等具体内容。"),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("检索问题，例如：越南的企业所得税率是多少？"),
		),
		mcp.WithString("country",
			mcp.Description("按国别过滤（可选），例如：越南"),
		),
		mcp.WithNumber("topK",
			mcp.Description("返回的知识片段数量（可选，1~20，默认 5）"),
		),
	)
}

func newAskTool() mcp.Tool {
	return mcp.NewTool("knowledge_ask",
		mcp.WithDescription("基于国别投资指南知识库回答投资相关问题。"+
			"内部执行 RAG 检索并让 LLM 生成完整回答，返回回答文本与引用来源。"),
		mcp.WithString("question",
			mcp.Required(),
			mcp.Description("投资相关问题，例如：越南的外资准入政策是什么？"),
		),
		mcp.WithString("country",
			mcp.Description("按国别过滤检索（可选），例如：越南"),
		),
	)
}
