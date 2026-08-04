package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

func searchHandler(search SearchFunc) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query, err := req.RequireString("query")
		if err != nil {
			return mcp.NewToolResultError("query 必填: " + err.Error()), nil
		}
		args := req.GetArguments()
		country, _ := args["country"].(string)
		topK := req.GetInt("topK", 5)
		if topK < 1 || topK > 20 {
			topK = 5
		}

		if search == nil {
			return mcp.NewToolResultError("knowledge_search 未配置"), nil
		}
		hits, err := search(ctx, query, country, topK)
		if err != nil {
			return mcp.NewToolResultError("检索失败: " + err.Error()), nil
		}
		if len(hits) == 0 {
			return mcp.NewToolResultText("未检索到相关知识片段。"), nil
		}
		return mcp.NewToolResultJSON(map[string]interface{}{
			"hits": hits,
		})
	}
}
