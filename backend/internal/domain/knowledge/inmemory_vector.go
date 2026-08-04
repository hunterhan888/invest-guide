package knowledge

import (
	"github.com/invest-guide/backend/internal/platform/embedding"
)

// inMemoryTopK 用 embedding.TopK 对内存中 chunks 排序，返回 TopK 个
func inMemoryTopK(query []float32, chunks []*KnowledgeChunk, topK int) []*KnowledgeChunk {
	if len(chunks) == 0 || topK <= 0 {
		return nil
	}
	vectors := make([][]float32, len(chunks))
	for i, c := range chunks {
		vectors[i] = []float32(c.Embedding)
	}
	hits := embedding.TopK(query, vectors, topK)
	result := make([]*KnowledgeChunk, 0, len(hits))
	for _, h := range hits {
		result = append(result, chunks[h.Index])
	}
	return result
}
