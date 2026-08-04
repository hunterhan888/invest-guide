package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

func askHandler(generate GenerateFunc) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		question, err := req.RequireString("question")
		if err != nil {
			return mcp.NewToolResultError("question 必填: " + err.Error()), nil
		}
		args := req.GetArguments()
		country, _ := args["country"].(string)

		if generate == nil {
			return mcp.NewToolResultError("knowledge_ask 未配置"), nil
		}
		answer, sources, err := generate(ctx, question, country)
		if err != nil {
			return mcp.NewToolResultError("回答生成失败: " + err.Error()), nil
		}

		return mcp.NewToolResultJSON(map[string]interface{}{
			"answer":  answer,
			"sources": sources,
		})
	}
}
