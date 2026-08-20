package assistant

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestContextSource_JSONContract 锁定 ContextSource 序列化后的 JSON 字段名，
// 必须与前端 KnowledgeChunkRef（id/title/snippet）一致，防止再出现大小写漂移。
func TestContextSource_JSONContract(t *testing.T) {
	src := ContextSource{ChunkID: "c1", Title: "标题", Snippet: "片段"}
	b, err := json.Marshal(src)
	require.NoError(t, err)

	var m map[string]string
	require.NoError(t, json.Unmarshal(b, &m))

	assert.Equal(t, "c1", m["id"], `JSON 字段应为小写 "id"（来自 ChunkID）`)
	assert.Equal(t, "标题", m["title"], `JSON 字段应为小写 "title"`)
	assert.Equal(t, "片段", m["snippet"], `JSON 字段应为小写 "snippet"`)
	assert.NotContains(t, m, "ChunkID", "不应出现 Go 默认的大写字段名")
	assert.NotContains(t, m, "Title", "不应出现 Go 默认的大写字段名")
	assert.NotContains(t, m, "Snippet", "不应出现 Go 默认的大写字段名")
}
