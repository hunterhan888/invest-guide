package assistant

// ContextSource 是 RAG 检索命中的知识片段引用
// JSON 字段名与前端 KnowledgeChunkRef 对齐（id/title/snippet）
type ContextSource struct {
	ChunkID string `json:"id"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
}

// AssembledContext 是装配 RAG 上下文的结果
type AssembledContext struct {
	SystemPrompt string
	Sources      []ContextSource
}
