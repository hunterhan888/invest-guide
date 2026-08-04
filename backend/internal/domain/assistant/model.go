package assistant

// ContextSource 是 RAG 检索命中的知识片段引用
type ContextSource struct {
	ChunkID string
	Title   string
	Snippet string
}

// AssembledContext 是装配 RAG 上下文的结果
type AssembledContext struct {
	SystemPrompt string
	Sources      []ContextSource
}
