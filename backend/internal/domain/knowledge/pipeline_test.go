package knowledge

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/invest-guide/backend/internal/platform/embedding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeEmbedding struct {
	dim int
}

func (f *fakeEmbedding) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i := range texts {
		v := make([]float32, f.dim)
		for j := range v {
			v[j] = float32(len(texts[i]) % 10)
		}
		out[i] = v
	}
	return out, nil
}

func (f *fakeEmbedding) Dim() int { return f.dim }

func newPipelineTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&KnowledgeDoc{}, &KnowledgeChunk{}))
	return db
}

func TestPipeline_RunParsesChunksEmbedsStores(t *testing.T) {
	db := newPipelineTestDB(t)
	docRepo := NewGORMDocRepository(db)
	chunkRepo := NewGORMChunkRepository(db)
	embed := &fakeEmbedding{dim: 4}
	p := NewPipeline(docRepo, chunkRepo, embed, 100, 10)

	doc := &KnowledgeDoc{ID: "doc-1", Title: "T", SourceType: SourceManual, Status: StatusPending}
	require.NoError(t, docRepo.Create(context.Background(), doc))

	content := "First paragraph.\n\nSecond paragraph.\n\nThird one."
	err := p.Run(context.Background(), doc.ID, content)
	require.NoError(t, err)

	d, _ := docRepo.Get(context.Background(), doc.ID)
	assert.Equal(t, StatusReady, d.Status)
	assert.Greater(t, d.ChunkCount, 0)

	vec := make([]float32, 4)
	hits, err := chunkRepo.SearchByVector(context.Background(), vec, 5, "")
	require.NoError(t, err)
	assert.NotEmpty(t, hits)
}

// TestPipeline_MultipleDocs_NoChunkCollision 复现唯一索引 idx_doc_seq 遗漏 doc_id 的缺陷：
// 两个文档各自从 seq=0 开始分块，若索引只在 seq 上全局唯一，第二个文档会插入冲突。
func TestPipeline_MultipleDocs_NoChunkCollision(t *testing.T) {
	db := newPipelineTestDB(t)
	docRepo := NewGORMDocRepository(db)
	chunkRepo := NewGORMChunkRepository(db)
	embed := &fakeEmbedding{dim: 4}
	p := NewPipeline(docRepo, chunkRepo, embed, 100, 10)

	for _, id := range []string{"doc-a", "doc-b", "doc-c"} {
		doc := &KnowledgeDoc{ID: id, Title: "T", SourceType: SourceManual, Status: StatusPending}
		require.NoError(t, docRepo.Create(context.Background(), doc))
	}

	for _, id := range []string{"doc-a", "doc-b", "doc-c"} {
		content := "第一段内容。" + strings.Repeat("这是用于产生多个分块的正文内容。", 20) + "\n\n第二段内容。" + strings.Repeat("继续补充正文以保证足够长。", 20) + "\n\n第三段内容。"
		err := p.Run(context.Background(), id, content)
		require.NoError(t, err, "pipeline failed for %s", id)

		d, _ := docRepo.Get(context.Background(), id)
		assert.Equal(t, StatusReady, d.Status, "doc %s should be ready", id)
		assert.Greater(t, d.ChunkCount, 0, "doc %s should have chunks", id)
	}

	perDoc := len(Chunk("第一段内容。"+strings.Repeat("这是用于产生多个分块的正文内容。", 20)+"\n\n第二段内容。"+strings.Repeat("继续补充正文以保证足够长。", 20)+"\n\n第三段内容。", 100, 10))
	var total int64
	require.NoError(t, db.Model(&KnowledgeChunk{}).Count(&total).Error)
	assert.EqualValues(t, 3*perDoc, total, "all docs' chunks should coexist")
}

func TestPipeline_Run_EmptyContent_Fails(t *testing.T) {
	db := newPipelineTestDB(t)
	docRepo := NewGORMDocRepository(db)
	chunkRepo := NewGORMChunkRepository(db)
	embed := &fakeEmbedding{dim: 4}
	p := NewPipeline(docRepo, chunkRepo, embed, 100, 10)

	doc := &KnowledgeDoc{ID: "doc-2", Title: "T", SourceType: SourceManual, Status: StatusPending}
	_ = docRepo.Create(context.Background(), doc)

	err := p.Run(context.Background(), doc.ID, "")
	require.Error(t, err)

	d, _ := docRepo.Get(context.Background(), doc.ID)
	assert.Equal(t, StatusFailed, d.Status)
	assert.NotNil(t, d.ErrorMessage)
}

func TestPipeline_RetryOnEmbedFailure(t *testing.T) {
	db := newPipelineTestDB(t)
	docRepo := NewGORMDocRepository(db)
	chunkRepo := NewGORMChunkRepository(db)
	embed := &failingEmbedding{dim: 4, failFirst: 2}
	p := NewPipeline(docRepo, chunkRepo, embed, 100, 10)
	p.retryDelays = []time.Duration{0, 0, 0, 0}

	doc := &KnowledgeDoc{ID: "doc-3", Title: "T", SourceType: SourceManual, Status: StatusPending}
	_ = docRepo.Create(context.Background(), doc)

	err := p.Run(context.Background(), doc.ID, "some content")
	require.NoError(t, err)
	assert.Equal(t, 3, embed.calls)
}

type failingEmbedding struct {
	dim       int
	failFirst int
	calls     int
}

func (f *failingEmbedding) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	f.calls++
	if f.calls <= f.failFirst {
		return nil, embedding.ErrProviderUnavailable
	}
	out := make([][]float32, len(texts))
	for i := range texts {
		out[i] = make([]float32, f.dim)
	}
	return out, nil
}

func (f *failingEmbedding) Dim() int { return f.dim }

// TestPipeline_RetryOnEmbedFailure_Wrapped 复现 shouldRetryEmbed 用 == 而非 errors.Is
// 匹配包装错误导致的缺陷：provider 以 %w 包装 ErrProviderUnavailable（如 429）时，
// 重试逻辑应仍能识别并重试。
func TestPipeline_RetryOnEmbedFailure_Wrapped(t *testing.T) {
	db := newPipelineTestDB(t)
	docRepo := NewGORMDocRepository(db)
	chunkRepo := NewGORMChunkRepository(db)
	embed := &wrappedFailingEmbedding{dim: 4, failFirst: 2}
	p := NewPipeline(docRepo, chunkRepo, embed, 100, 10)
	p.retryDelays = []time.Duration{0, 0, 0, 0}

	doc := &KnowledgeDoc{ID: "doc-4", Title: "T", SourceType: SourceManual, Status: StatusPending}
	_ = docRepo.Create(context.Background(), doc)

	err := p.Run(context.Background(), doc.ID, "some content")
	require.NoError(t, err, "wrapped ErrProviderUnavailable should be retried")
	assert.Equal(t, 3, embed.calls)

	d, _ := docRepo.Get(context.Background(), doc.ID)
	assert.Equal(t, StatusReady, d.Status)
}

type wrappedFailingEmbedding struct {
	dim       int
	failFirst int
	calls     int
}

func (f *wrappedFailingEmbedding) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	f.calls++
	if f.calls <= f.failFirst {
		return nil, fmt.Errorf("%w: status 429", embedding.ErrProviderUnavailable)
	}
	out := make([][]float32, len(texts))
	for i := range texts {
		out[i] = make([]float32, f.dim)
	}
	return out, nil
}

func (f *wrappedFailingEmbedding) Dim() int { return f.dim }
