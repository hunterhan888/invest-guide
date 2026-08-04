package knowledge

import (
	"context"

	"github.com/google/uuid"
	"github.com/invest-guide/backend/internal/platform/embedding"
	"github.com/invest-guide/backend/internal/platform/response"
	"github.com/invest-guide/backend/internal/platform/taskqueue"
)

type Service struct {
	docs   DocRepository
	chunks ChunkRepository
	embed  embedding.Provider
	queue  taskqueue.Queue
	cache  EmbeddingCache
}

// EmbeddingCache 缓存 embedding 结果（按文本 hash）
type EmbeddingCache interface {
	Get(text string) ([]float32, bool)
	Set(text string, vec []float32)
}

func NewService(docs DocRepository, chunks ChunkRepository, embed embedding.Provider, queue taskqueue.Queue, cache EmbeddingCache) *Service {
	return &Service{docs: docs, chunks: chunks, embed: embed, queue: queue, cache: cache}
}

// Create 创建 pending 文档并异步入库
func (s *Service) Create(ctx context.Context, req CreateDocRequest) (*DocDTO, error) {
	doc := &KnowledgeDoc{
		ID:              uuid.NewString(),
		Title:           req.Title,
		Country:         req.Country,
		SourceType:      req.SourceType,
		SourceURL:       req.SourceURL,
		OriginalContent: req.Content, // 保存原文供 Retry 使用
		Status:          StatusPending,
	}
	if err := s.docs.Create(ctx, doc); err != nil {
		return nil, err
	}

	content := req.Content
	if err := s.queue.Enqueue(func(ctx context.Context) error {
		p := NewPipeline(s.docs, s.chunks, s.embed, pipelineChunkSize, pipelineOverlap)
		return p.Run(ctx, doc.ID, content)
	}); err != nil {
		msg := "enqueue failed: " + err.Error()
		_ = s.docs.Update(ctx, doc.ID, UpdateDocParams{Status: strPtr(StatusFailed), ErrorMessage: strPtr(msg)})
		return nil, response.ErrInternal
	}

	dto := doc.ToDTO()
	return &dto, nil
}

func (s *Service) Get(ctx context.Context, id string) (*DocDTO, error) {
	doc, err := s.docs.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	dto := doc.ToDTO()
	return &dto, nil
}

func (s *Service) List(ctx context.Context, page, pageSize int, country string) ([]*DocDTO, int64, error) {
	docs, total, err := s.docs.List(ctx, page, pageSize, country)
	if err != nil {
		return nil, 0, err
	}
	dtos := make([]*DocDTO, len(docs))
	for i, d := range docs {
		dto := d.ToDTO()
		dtos[i] = &dto
	}
	return dtos, total, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.chunks.DeleteByDoc(ctx, id); err != nil {
		return err
	}
	return s.docs.Delete(ctx, id)
}

// Retry 重新触发出错的文档入库
func (s *Service) Retry(ctx context.Context, id string) error {
	doc, err := s.docs.Get(ctx, id)
	if err != nil {
		return err
	}
	if doc.Status != StatusFailed {
		return response.ErrConflict
	}
	if doc.OriginalContent == "" {
		return response.ErrConflict // 无原文，需重新上传
	}
	if err := s.docs.Update(ctx, id, UpdateDocParams{Status: strPtr(StatusPending), ErrorMessage: nil}); err != nil {
		return err
	}
	content := doc.OriginalContent
	return s.queue.Enqueue(func(ctx context.Context) error {
		p := NewPipeline(s.docs, s.chunks, s.embed, pipelineChunkSize, pipelineOverlap)
		return p.Run(ctx, doc.ID, content)
	})
}

// Search 检索接口（供 Plan 4 conversation 集成 RAG 时调用）
func (s *Service) Search(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	topK := req.TopK
	if topK <= 0 || topK > 20 {
		topK = 5
	}
	var vec []float32
	if s.cache != nil {
		if v, ok := s.cache.Get(req.Query); ok {
			vec = v
		}
	}
	if vec == nil {
		vecs, err := s.embed.Embed(ctx, []string{req.Query})
		if err != nil {
			return nil, response.ErrBadGateway
		}
		vec = vecs[0]
		if s.cache != nil {
			s.cache.Set(req.Query, vec)
		}
	}

	chunks, err := s.chunks.SearchByVector(ctx, vec, topK, req.Country)
	if err != nil {
		return nil, err
	}

	hits := make([]ChunkHit, 0, len(chunks))
	for _, c := range chunks {
		doc, _ := s.docs.Get(ctx, c.DocID)
		title := ""
		if doc != nil {
			title = doc.Title
		}
		snippet := c.Content
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}
		hits = append(hits, ChunkHit{
			ID: c.ID, DocID: c.DocID, Title: title, Snippet: snippet, Score: 0,
		})
	}
	return &SearchResponse{Chunks: hits}, nil
}
