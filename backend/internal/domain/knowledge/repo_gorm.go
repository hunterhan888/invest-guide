package knowledge

import (
	"context"
	"errors"

	"github.com/invest-guide/backend/internal/platform/response"
	"gorm.io/gorm"
)

type gormDocRepository struct {
	db *gorm.DB
}

func NewGORMDocRepository(db *gorm.DB) DocRepository {
	return &gormDocRepository{db: db}
}

func (r *gormDocRepository) Create(ctx context.Context, doc *KnowledgeDoc) error {
	return r.db.WithContext(ctx).Create(doc).Error
}

func (r *gormDocRepository) Get(ctx context.Context, id string) (*KnowledgeDoc, error) {
	var d KnowledgeDoc
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&d).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, response.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *gormDocRepository) Update(ctx context.Context, id string, params UpdateDocParams) error {
	updates := map[string]interface{}{}
	if params.Status != nil {
		updates["status"] = *params.Status
	}
	if params.ErrorMessage != nil {
		updates["error_message"] = *params.ErrorMessage
	}
	if params.ChunkCount != nil {
		updates["chunk_count"] = *params.ChunkCount
	}
	if len(updates) == 0 {
		return nil
	}
	res := r.db.WithContext(ctx).Model(&KnowledgeDoc{}).Where("id = ?", id).Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return response.ErrNotFound
	}
	return nil
}

func (r *gormDocRepository) Delete(ctx context.Context, id string) error {
	res := r.db.WithContext(ctx).Where("id = ?", id).Delete(&KnowledgeDoc{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return response.ErrNotFound
	}
	return nil
}

func (r *gormDocRepository) List(ctx context.Context, page, pageSize int, country string) ([]*KnowledgeDoc, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	q := r.db.WithContext(ctx).Model(&KnowledgeDoc{})
	if country != "" {
		q = q.Where("country = ?", country)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var docs []*KnowledgeDoc
	if err := q.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&docs).Error; err != nil {
		return nil, 0, err
	}
	return docs, total, nil
}

type gormChunkRepository struct {
	db *gorm.DB
}

// NewGORMChunkRepository 返回 ChunkRepository。SQLite（开发）用内存检索；
// PG（生产）走 pgvector 算子（本 plan 暂只实现 SQLite 路径）。
func NewGORMChunkRepository(db *gorm.DB) ChunkRepository {
	return &gormChunkRepository{db: db}
}

func (r *gormChunkRepository) BatchCreate(ctx context.Context, chunks []*KnowledgeChunk) error {
	if len(chunks) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(chunks, 100).Error
}

func (r *gormChunkRepository) DeleteByDoc(ctx context.Context, docID string) error {
	return r.db.WithContext(ctx).Where("doc_id = ?", docID).Delete(&KnowledgeChunk{}).Error
}

func (r *gormChunkRepository) SearchByVector(ctx context.Context, vec []float32, topK int, country string) ([]*KnowledgeChunk, error) {
	// SQLite 路径：取全部（按 country 过滤）→ 内存 cosine TopK
	q := r.db.WithContext(ctx).Model(&KnowledgeChunk{}).
		Select("knowledge_chunks.*").
		Joins("LEFT JOIN knowledge_docs ON knowledge_docs.id = knowledge_chunks.doc_id")
	if country != "" {
		q = q.Where("knowledge_docs.country = ?", country)
	}
	var chunks []*KnowledgeChunk
	if err := q.Find(&chunks).Error; err != nil {
		return nil, err
	}
	return inMemoryTopK(vec, chunks, topK), nil
}

func (r *gormChunkRepository) GetMany(ctx context.Context, ids []string) ([]*KnowledgeChunk, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var chunks []*KnowledgeChunk
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&chunks).Error; err != nil {
		return nil, err
	}
	return chunks, nil
}
