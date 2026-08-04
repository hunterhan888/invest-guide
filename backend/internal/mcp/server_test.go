package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSearchReq(args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "knowledge_search",
			Arguments: args,
		},
	}
}

func newAskReq(args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "knowledge_ask",
			Arguments: args,
		},
	}
}

// resultText 提取 CallToolResult 的文本内容
func resultText(t *testing.T, r *mcp.CallToolResult) string {
	t.Helper()
	require.Len(t, r.Content, 1)
	tc, ok := r.Content[0].(mcp.TextContent)
	require.True(t, ok, "content 应为 TextContent, got %T", r.Content[0])
	return tc.Text
}

func TestSearchHandler_Success(t *testing.T) {
	search := func(ctx context.Context, query, country string, topK int) ([]SearchHit, error) {
		assert.Equal(t, "越南企业所得税", query)
		assert.Equal(t, "越南", country)
		assert.Equal(t, 5, topK)
		return []SearchHit{
			{ID: "c1", Title: "越南指南", Snippet: "企业所得税率 20%"},
		}, nil
	}
	h := searchHandler(search)
	result, err := h(context.Background(), newSearchReq(map[string]any{
		"query": "越南企业所得税", "country": "越南",
	}))
	require.NoError(t, err)
	assert.False(t, result.IsError)

	var body struct {
		Hits []SearchHit `json:"hits"`
	}
	require.NoError(t, json.Unmarshal([]byte(resultText(t, result)), &body))
	require.Len(t, body.Hits, 1)
	assert.Equal(t, "c1", body.Hits[0].ID)
}

func TestSearchHandler_MissingQuery(t *testing.T) {
	h := searchHandler(func(ctx context.Context, q, c string, k int) ([]SearchHit, error) {
		t.Fatal("不应调用 search")
		return nil, nil
	})
	result, err := h(context.Background(), newSearchReq(map[string]any{}))
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, resultText(t, result), "query")
}

func TestSearchHandler_EmptyResult(t *testing.T) {
	h := searchHandler(func(ctx context.Context, q, c string, k int) ([]SearchHit, error) {
		return nil, nil
	})
	result, err := h(context.Background(), newSearchReq(map[string]any{"query": "不存在的东西"}))
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, resultText(t, result), "未检索到")
}

func TestSearchHandler_Error(t *testing.T) {
	h := searchHandler(func(ctx context.Context, q, c string, k int) ([]SearchHit, error) {
		return nil, errors.New("db down")
	})
	result, err := h(context.Background(), newSearchReq(map[string]any{"query": "q"}))
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, resultText(t, result), "db down")
}

func TestSearchHandler_NilService(t *testing.T) {
	h := searchHandler(nil)
	result, err := h(context.Background(), newSearchReq(map[string]any{"query": "q"}))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestAskHandler_Success(t *testing.T) {
	generate := func(ctx context.Context, question, country string) (string, []Source, error) {
		assert.Equal(t, "越南外资准入", question)
		return "越南允许外资独资。", []Source{{Title: "越南指南", Snippet: "外资准入政策"}}, nil
	}
	h := askHandler(generate)
	result, err := h(context.Background(), newAskReq(map[string]any{"question": "越南外资准入"}))
	require.NoError(t, err)
	assert.False(t, result.IsError)

	var body struct {
		Answer  string   `json:"answer"`
		Sources []Source `json:"sources"`
	}
	require.NoError(t, json.Unmarshal([]byte(resultText(t, result)), &body))
	assert.Equal(t, "越南允许外资独资。", body.Answer)
	require.Len(t, body.Sources, 1)
	assert.Equal(t, "越南指南", body.Sources[0].Title)
}

func TestAskHandler_MissingQuestion(t *testing.T) {
	h := askHandler(nil)
	result, err := h(context.Background(), newAskReq(map[string]any{}))
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, resultText(t, result), "question")
}

func TestServer_ToolRegistered(t *testing.T) {
	s := NewServer(Service{})
	tools := s.ListTools()
	require.Contains(t, tools, "knowledge_search")
	require.Contains(t, tools, "knowledge_ask")
	require.Len(t, tools, 2)
}
