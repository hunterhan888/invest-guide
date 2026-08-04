package knowledge

import (
	"context"
)

type DocRepository interface {
	Create(ctx context.Context, doc *KnowledgeDoc) error
	Get(ctx context.Context, id string) (*KnowledgeDoc, error)
	Update(ctx context.Context, id string, params UpdateDocParams) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, page, pageSize int, country string) ([]*KnowledgeDoc, int64, error)
}

type ChunkRepository interface {
	BatchCreate(ctx context.Context, chunks []*KnowledgeChunk) error
	DeleteByDoc(ctx context.Context, docID string) error
	// SearchByVector 按 country 过滤，返回 TopK chunks；country 为空时不过滤
	SearchByVector(ctx context.Context, vec []float32, topK int, country string) ([]*KnowledgeChunk, error)
	// GetMany 按 ID 批量取
	GetMany(ctx context.Context, ids []string) ([]*KnowledgeChunk, error)
}

type UpdateDocParams struct {
	Status       *string
	ErrorMessage *string
	ChunkCount   *int
}
