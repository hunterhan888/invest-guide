package knowledge

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/invest-guide/backend/internal/platform/taskqueue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeQueue struct {
	enqueued int32
	fn       taskqueue.Task
}

func (f *fakeQueue) Enqueue(task taskqueue.Task) error {
	atomic.AddInt32(&f.enqueued, 1)
	f.fn = task
	return nil
}

func (f *fakeQueue) Close(ctx context.Context) error { return nil }

func (f *fakeQueue) runLast(ctx context.Context) error {
	if f.fn != nil {
		return f.fn(ctx)
	}
	return nil
}

func newServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&KnowledgeDoc{}, &KnowledgeChunk{}))
	return db
}

func newTestService(t *testing.T) (*Service, *gorm.DB, *fakeQueue) {
	t.Helper()
	db := newServiceTestDB(t)
	q := &fakeQueue{}
	svc := NewService(
		NewGORMDocRepository(db),
		NewGORMChunkRepository(db),
		&fakeEmbedding{dim: 4},
		q,
		nil,
	)
	return svc, db, q
}

func TestService_Create_EnqueuesAndReturnsPending(t *testing.T) {
	svc, _, q := newTestService(t)
	dto, err := svc.Create(context.Background(), CreateDocRequest{
		Title: "越南", Country: "越南", SourceType: SourceManual, Content: "河内是首都",
	})
	require.NoError(t, err)
	assert.Equal(t, StatusPending, dto.Status)
	assert.Equal(t, int32(1), atomic.LoadInt32(&q.enqueued))
}

func TestService_Get_NotFound(t *testing.T) {
	svc, _, _ := newTestService(t)
	_, err := svc.Get(context.Background(), "missing")
	assert.Error(t, err)
}

func TestService_Delete_RemovesChunksAndDoc(t *testing.T) {
	svc, db, q := newTestService(t)
	// 先走一遍 pipeline 让 doc ready
	docRepo := NewGORMDocRepository(db)
	doc := &KnowledgeDoc{ID: "d1", Title: "T", SourceType: SourceManual, Status: StatusPending}
	require.NoError(t, docRepo.Create(context.Background(), doc))
	chunkRepo := NewGORMChunkRepository(db)
	pipe := NewPipeline(docRepo, chunkRepo, &fakeEmbedding{dim: 4}, 100, 10)
	require.NoError(t, pipe.Run(context.Background(), "d1", "content here"))

	require.NoError(t, svc.Delete(context.Background(), "d1"))
	_, err := docRepo.Get(context.Background(), "d1")
	assert.Error(t, err)

	// 删除后 chunk 也应清空
	hits, err := chunkRepo.SearchByVector(context.Background(), make([]float32, 4), 5, "")
	require.NoError(t, err)
	assert.Empty(t, hits)
	_ = q
}

func TestService_Retry_FailedDoc(t *testing.T) {
	svc, db, q := newTestService(t)
	docRepo := NewGORMDocRepository(db)
	doc := &KnowledgeDoc{ID: "r1", Title: "T", SourceType: SourceManual, Status: StatusFailed,
		OriginalContent: "retry me"}
	require.NoError(t, docRepo.Create(context.Background(), doc))

	require.NoError(t, svc.Retry(context.Background(), "r1"))
	assert.Equal(t, int32(1), atomic.LoadInt32(&q.enqueued))
}

func TestService_Retry_NotFailed_Conflict(t *testing.T) {
	svc, db, _ := newTestService(t)
	docRepo := NewGORMDocRepository(db)
	doc := &KnowledgeDoc{ID: "r2", Title: "T", SourceType: SourceManual, Status: StatusReady}
	require.NoError(t, docRepo.Create(context.Background(), doc))

	err := svc.Retry(context.Background(), "r2")
	assert.Error(t, err)
}

func TestService_Search_ReturnsHits(t *testing.T) {
	svc, db, _ := newTestService(t)
	docRepo := NewGORMDocRepository(db)
	doc := &KnowledgeDoc{ID: "s1", Title: "越南指南", Country: "越南", SourceType: SourceManual, Status: StatusPending}
	require.NoError(t, docRepo.Create(context.Background(), doc))

	chunkRepo := NewGORMChunkRepository(db)
	pipe := NewPipeline(docRepo, chunkRepo, &fakeEmbedding{dim: 4}, 100, 10)
	require.NoError(t, pipe.Run(context.Background(), "s1", "河内是首都。胡志明市是经济中心。"))

	// service 用自己的 chunkRepo/embed
	svc.docs = docRepo
	svc.chunks = chunkRepo
	svc.embed = &fakeEmbedding{dim: 4}

	resp, err := svc.Search(context.Background(), SearchRequest{Query: "河内", Country: "越南"})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Chunks)
	assert.Equal(t, "越南指南", resp.Chunks[0].Title)
}
