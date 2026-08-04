package knowledge

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/invest-guide/backend/internal/platform/embedding"
	"github.com/invest-guide/backend/internal/platform/response"
)

type Pipeline struct {
	docs        DocRepository
	chunks      ChunkRepository
	embed       embedding.Provider
	chunkSize   int
	overlap     int
	retryDelays []time.Duration
}

const (
	pipelineChunkSize = 2048 // ~512 tokens (4 chars/token)
	pipelineOverlap   = 200  // ~50 tokens
	embedMaxRetries   = 3
)

func NewPipeline(docs DocRepository, chunks ChunkRepository, embed embedding.Provider, chunkSize, overlap int) *Pipeline {
	return &Pipeline{
		docs:        docs,
		chunks:      chunks,
		embed:       embed,
		chunkSize:   chunkSize,
		overlap:     overlap,
		retryDelays: []time.Duration{0, time.Second, 2 * time.Second, 4 * time.Second},
	}
}

// Run 执行完整流水线：Parse → Chunk → Embed → Store
// 失败时把文档状态置为 failed 并写 error_message
func (p *Pipeline) Run(ctx context.Context, docID, content string) error {
	if err := p.docs.Update(ctx, docID, UpdateDocParams{Status: strPtr(StatusProcessing)}); err != nil {
		return err
	}

	fail := func(msg string) error {
		_ = p.docs.Update(ctx, docID, UpdateDocParams{
			Status:       strPtr(StatusFailed),
			ErrorMessage: strPtr(msg),
		})
		return fmt.Errorf("%w: %s", response.ErrInternal, msg)
	}

	doc, err := p.docs.Get(ctx, docID)
	if err != nil {
		return fail("fetch doc: " + err.Error())
	}
	parsed, err := Parse(content, doc.SourceType)
	if err != nil {
		return fail("parse: " + err.Error())
	}
	textChunks := Chunk(parsed, p.chunkSize, p.overlap)
	if len(textChunks) == 0 {
		return fail("no content after parse")
	}

	vecs, err := p.embedWithRetry(ctx, textChunks)
	if err != nil {
		return fail("embed: " + err.Error())
	}

	if err := p.chunks.DeleteByDoc(ctx, docID); err != nil {
		return fail("clear existing chunks: " + err.Error())
	}
	entities := make([]*KnowledgeChunk, len(textChunks))
	now := time.Now()
	for i, txt := range textChunks {
		entities[i] = &KnowledgeChunk{
			ID:        uuid.NewString(),
			DocID:     docID,
			Seq:       i,
			Content:   txt,
			Embedding: JSONFloat32(vecs[i]),
			CreatedAt: now,
		}
	}
	if err := p.chunks.BatchCreate(ctx, entities); err != nil {
		return fail("store chunks: " + err.Error())
	}

	if err := p.docs.Update(ctx, docID, UpdateDocParams{
		Status:     strPtr(StatusReady),
		ChunkCount: intPtr(len(entities)),
	}); err != nil {
		return err
	}
	return nil
}

// embedWithRetry 实现指数退避：1s → 2s → 4s
func (p *Pipeline) embedWithRetry(ctx context.Context, texts []string) ([][]float32, error) {
	delays := p.retryDelays
	var lastErr error
	for attempt := 0; attempt < embedMaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delays[attempt]):
			}
		}
		vecs, err := p.embed.Embed(ctx, texts)
		if err == nil {
			return vecs, nil
		}
		lastErr = err
		if !shouldRetryEmbed(err) {
			return nil, err
		}
	}
	return nil, lastErr
}

func shouldRetryEmbed(err error) bool {
	return errors.Is(err, embedding.ErrProviderUnavailable)
}

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }
