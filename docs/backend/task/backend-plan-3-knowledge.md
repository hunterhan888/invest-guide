# 后端 Plan 3 — knowledge 领域 Implementation Plan

**Goal:** 实现 `domain/knowledge` 模块：国别投资指南文档的异步入库流水线（解析 → 分块 → 向量化 → 存储）、向量检索（Top-K）、文档管理 CRUD；定义 `EmbeddingProvider` 接口供 Plan 4 复用；缓存放 Embedding 结果与热门检索结果；迁移含 pgvector 向量列。

**Architecture:** `KnowledgeDoc` 跟踪文档元数据与入库状态（pending→processing→ready/failed），`KnowledgeChunk` 存分块文本 + embedding 向量；入库任务经 `taskqueue.Enqueue` 异步执行（goroutine pool），API 返回 `202` + 文档 ID；检索流程：embed 用户问题 → SearchByVector Top-K（开发用内存 brute-force cosine，生产用 pgvector `<=>`）→ 返回 chunks；service 只依赖接口，GORM 实现独立；EmbeddingProvider 接口在 `assistant/` 下定义（Plan 4 也用，本 plan 先定义接口、放 platform 层避免依赖反转）。

**Tech Stack:** GORM + pgvector-go、`net/http`（调 OpenAI-compatible embedding API）、`html/template` 之外的纯 `strings`/`regexp`（解析 markdown 简单实现）、testify、SQLite（测试用内存 cosine）。

**关联设计：** `ARCHITECTURE.md`（RAG 知识管线 / LLM Provider 抽象 / 数据模型 / 缓存策略 / 错误传播）

**前置条件：** Plan 1（platform/ 全部就绪）+ Plan 2（auth 中间件，knowledge 路由全部需鉴权）已完成。

---

## 文件结构

```
backend/
├── internal/
│   ├── platform/
│   │   └── embedding/
│   │       ├── provider.go                 # EmbeddingProvider 接口 + 错误
│   │       ├── openai.go                   # OpenAI-compatible HTTP 实现
│   │       ├── openai_test.go
│   │       ├── cosine.go                   # brute-force cosine 内存检索
│   │       └── cosine_test.go
│   └── domain/knowledge/
│       ├── route.go                        # Register(private *gin.RouterGroup)
│       ├── handler.go                      # list/create/get/delete/retry
│       ├── service.go                      # 业务编排 + 入库流水线
│       ├── repository.go                   # KnowledgeDocRepository + KnowledgeChunkRepository 接口
│       ├── repo_gorm.go                    # GORM 实现
│       ├── inmemory_vector.go              # 内存 brute-force 检索实现（开发,同一接口）
│       ├── parser.go                       # 纯文本/markdown/HTML 解析器
│       ├── chunker.go                      # 按 token 分块（512, 10% overlap）
│       ├── chunker_test.go
│       ├── pipeline.go                     #入库流水线：parse→chunk→embed→store
│       ├── model.go                        # KnowledgeDoc + KnowledgeChunk 实体 + DTO
│       ├── service_test.go
│       ├── handler_test.go
│       └── pipeline_test.go
├── migrations/
│   ├── 0003_knowledge.up.sql
│   └── 0003_knowledge.down.sql
└── tests/e2e/
    └── knowledge_test.go                   # 上传→轮询状态→检索完整链路
```

> `EmbeddingProvider` 放 `platform/embedding/` 而非 `domain/assistant/`（Plan 4 的 LLM 自有归属）。原因：knowledge 与 assistant 都用 embedding，放 platform 层避免 domain/knowledge 反向依赖 domain/assistant。这点偏离 ARCHITECTURE.md「assistant 含 EmbeddingProvider 抽象」的字面表述，需在 Plan 4 完成后回头同步 ARCHITECTURE.md。

---

### Task 0: 迁移 — knowledge 表

**Files:**
- Create: `backend/migrations/0003_knowledge.up.sql`
- Create: `backend/migrations/0003_knowledge.down.sql`

- [ ] **Step 1: 写 up.sql**

`backend/migrations/0003_knowledge.up.sql`:
```sql
CREATE TABLE knowledge_docs (
    id               TEXT PRIMARY KEY,
    title            TEXT NOT NULL,
    country          TEXT NOT NULL DEFAULT '',
    source_type      TEXT NOT NULL,
    source_url       TEXT,
    original_content TEXT,
    status           TEXT NOT NULL DEFAULT 'pending',
    error_message    TEXT,
    chunk_count      INTEGER NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_knowledge_docs_country ON knowledge_docs(country);
CREATE INDEX idx_knowledge_docs_status ON knowledge_docs(status);

CREATE TABLE knowledge_chunks (
    id          TEXT PRIMARY KEY,
    doc_id      TEXT NOT NULL REFERENCES knowledge_docs(id) ON DELETE CASCADE,
    seq         INTEGER NOT NULL,
    content     TEXT NOT NULL,
    embedding   vector(1024) NOT NULL,
    metadata    JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(doc_id, seq)
);
CREATE INDEX idx_knowledge_chunks_doc_id ON knowledge_chunks(doc_id);
```

> pgvector `vector(1024)` 列在 SQLite 中不存在；开发路径用 GORM `JSON` 列存 `[]float32`，由 inmemory_vector 检索层处理（见 Task 5）。生产用 PG 时该 SQL 直接生效。`vector` 扩展已在 0001_init.up.sql 创建。

- [ ] **Step 2: 写 down.sql**

`backend/migrations/0003_knowledge.down.sql`:
```sql
DROP TABLE IF EXISTS knowledge_chunks;
DROP TABLE IF EXISTS knowledge_docs;
```

- [ ] **Step 3: Commit**

```bash
git add backend/migrations/0003_knowledge.*
git commit -m "feat(backend/knowledge): add knowledge_docs and knowledge_chunks migrations"
```

---

### Task 1: `knowledge/model.go` — 实体与 DTO

**Files:**
- Create: `backend/internal/domain/knowledge/model.go`

- [ ] **Step 1: 写 model.go**

`backend/internal/domain/knowledge/model.go`:
```go
package knowledge

import (
	"encoding/json"
	"time"
)

// 文档状态
const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusReady      = "ready"
	StatusFailed     = "failed"
)

// 来源类型
const (
	SourceManual = "manual"
	SourceUpload = "upload"
	SourceURL    = "url"
)

// KnowledgeDoc 是文档元数据实体
type KnowledgeDoc struct {
	ID               string    `gorm:"primaryKey" json:"id"`
	Title            string    `gorm:"not null" json:"title"`
	Country          string    `json:"country"`
	SourceType       string    `gorm:"not null" json:"sourceType"`
	SourceURL        *string   `json:"sourceUrl,omitempty"`
	OriginalContent  string    `gorm:"type:text" json:"-"`
	Status           string    `gorm:"not null;default:pending" json:"status"`
	ErrorMessage     *string   `json:"errorMessage,omitempty"`
	ChunkCount       int       `gorm:"not null;default:0" json:"chunkCount"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

func (KnowledgeDoc) TableName() string { return "knowledge_docs" }

// KnowledgeChunk 是分块 + 向量
type KnowledgeChunk struct {
	ID        string          `gorm:"primaryKey" json:"id"`
	DocID     string          `gorm:"not null;index" json:"docId"`
	Seq       int             `gorm:"not null;uniqueIndex:idx_doc_seq,priority:1" json:"seq"`
	Content   string          `gorm:"not null;type:text" json:"content"`
	Embedding JSONFloat32     `gorm:"type:jsonb" json:"-"` // SQLite 用 JSON, PG 时由 repo 转换为 vector
	Metadata  json.RawMessage `gorm:"type:jsonb" json:"metadata,omitempty"`
	CreatedAt time.Time       `json:"createdAt"`
}

func (KnowledgeChunk) TableName() string { return "knowledge_chunks" }

// JSONFloat32 自定义类型 — GORM 用 JSON 列存 []float32（兼容 SQLite）
// 生产 PG 路径下 repo_gorm 会改用 pgvector.Vector 类型，跳过此字段
type JSONFloat32 []float32

func (v JSONFloat32) GormDataType() string { return "json" }

// DTOs
type CreateDocRequest struct {
	Title      string  `json:"title" binding:"required,max=200"`
	Country    string  `json:"country" binding:"required,max=100"`
	SourceType string  `json:"sourceType" binding:"required,oneof=manual upload url"`
	SourceURL  *string `json:"sourceUrl,omitempty" binding:"omitempty,url"`
	Content    string  `json:"content" binding:"required_if=SourceType manual"` // manual 必须传内容
}

type DocDTO struct {
	ID           string     `json:"id"`
	Title        string     `json:"title"`
	Country      string     `json:"country"`
	SourceType   string     `json:"sourceType"`
	SourceURL    *string    `json:"sourceUrl,omitempty"`
	Status       string     `json:"status"`
	ErrorMessage *string    `json:"errorMessage,omitempty"`
	ChunkCount   int        `json:"chunkCount"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

func (d *KnowledgeDoc) ToDTO() DocDTO {
	return DocDTO{
		ID: d.ID, Title: d.Title, Country: d.Country, SourceType: d.SourceType,
		SourceURL: d.SourceURL, Status: d.Status, ErrorMessage: d.ErrorMessage,
		ChunkCount: d.ChunkCount, CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
	}
}

type ChunkHit struct {
	ID      string `json:"id"`
	DocID   string `json:"docId"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
	Score   float64 `json:"score"`
}

type SearchRequest struct {
	Query   string `json:"query" binding:"required"`
	Country string `json:"country,omitempty"`
	TopK    int    `json:"topK,omitempty"`
}

type SearchResponse struct {
	Chunks []ChunkHit `json:"chunks"`
}
```

- [ ] **Step 2: 验证编译**

Run: `cd backend && go build ./...`
Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add backend/internal/domain/knowledge/model.go
git commit -m "feat(backend/knowledge): add KnowledgeDoc/Chunk entities and DTOs"
```

---

### Task 2: `platform/embedding/provider.go` — 接口定义

**Files:**
- Create: `backend/internal/platform/embedding/provider.go`

- [ ] **Step 1: 写接口**

`backend/internal/platform/embedding/provider.go`:
```go
package embedding

import (
	"context"
	"errors"
)

var (
	ErrProviderUnavailable = errors.New("embedding provider unavailable")
	ErrInvalidDim          = errors.New("invalid embedding dimension")
)

// Provider 抽象 Embedding 服务（OpenAI-compatible）
type Provider interface {
	// Embed 返回 texts 中每条文本的向量；长度必须与 texts 一致
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	// Dim 返回向量维度（供 schema 校验）
	Dim() int
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/internal/platform/embedding/provider.go
git commit -m "feat(backend/embedding): add Provider interface"
```

---

### Task 3: `platform/embedding/openai.go` + 测试

**Files:**
- Create: `backend/internal/platform/embedding/openai.go`
- Create: `backend/internal/platform/embedding/openai_test.go`

- [ ] **Step 1: 写失败测试（用 httptest mock 上游 API）**

`backend/internal/platform/embedding/openai_test.go`:
```go
package embedding

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAIProvider_Embed_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/embeddings", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		var req struct {
			Input []string `json:"input"`
			Model string   `json:"model"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, []string{"hello", "world"}, req.Input)

		resp := map[string]interface{}{
			"data": []map[string]interface{}{
				{"embedding": []float32{0.1, 0.2, 0.3}},
				{"embedding": []float32{0.4, 0.5, 0.6}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p := NewOpenAIProvider(srv.URL, "test-key", "test-model", 3)
	vecs, err := p.Embed(context.Background(), []string{"hello", "world"})
	require.NoError(t, err)
	assert.Len(t, vecs, 2)
	assert.Equal(t, []float32{0.1, 0.2, 0.3}, vecs[0])
	assert.Equal(t, []float32{0.4, 0.5, 0.6}, vecs[1])
	assert.Equal(t, 3, p.Dim())
}

func TestOpenAIProvider_Embed_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := NewOpenAIProvider(srv.URL, "k", "m", 3)
	_, err := p.Embed(context.Background(), []string{"x"})
	assert.ErrorIs(t, err, ErrProviderUnavailable)
}

func TestOpenAIProvider_Embed_RetryOn429(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"embedding":[0.1,0.2]}]}`))
	}))
	defer srv.Close()

	p := NewOpenAIProvider(srv.URL, "k", "m", 2)
	vecs, err := p.Embed(context.Background(), []string{"x"})
	require.NoError(t, err)
	assert.Equal(t, []float32{0.1, 0.2}, vecs[0])
	assert.Equal(t, 2, calls) // 第一次 429，第二次成功
}

func TestOpenAIProvider_Embed_DimMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"embedding":[0.1,0.2]}]}`)) // 维度 2，但 Provider 期望 3
	}))
	defer srv.Close()

	p := NewOpenAIProvider(srv.URL, "k", "m", 3)
	_, err := p.Embed(context.Background(), []string{"x"})
	assert.ErrorIs(t, err, ErrInvalidDim)
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd backend && go test ./internal/platform/embedding/`
Expected: FAIL — `NewOpenAIProvider` 未定义

- [ ] **Step 3: 写实现**

`backend/internal/platform/embedding/openai.go`:
```go
package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type OpenAIProvider struct {
	baseURL string
	apiKey  string
	model   string
	dim     int
	client  *http.Client
}

func NewOpenAIProvider(baseURL, apiKey, model string, dim int) *OpenAIProvider {
	return &OpenAIProvider{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		dim:     dim,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

type embedRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

type embedResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (p *OpenAIProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	body, _ := json.Marshal(embedRequest{Input: texts, Model: p.model})
	url := p.baseURL + "/v1/embeddings"

	var resp embedResponse
	if err := p.doWithRetry(ctx, url, body, &resp); err != nil {
		return nil, err
	}

	if len(resp.Data) != len(texts) {
		return nil, fmt.Errorf("%w: expected %d vectors, got %d", ErrProviderUnavailable, len(texts), len(resp.Data))
	}

	out := make([][]float32, len(resp.Data))
	for i, d := range resp.Data {
		if len(d.Embedding) != p.dim {
			return nil, fmt.Errorf("%w: vector %d has dim %d, expected %d", ErrInvalidDim, i, len(d.Embedding), p.dim)
		}
		out[i] = d.Embedding
	}
	return out, nil
}

func (p *OpenAIProvider) Dim() int { return p.dim }

// doWithRetry 重试策略：429 / 5xx → 指数退避（1s, 2s, 4s），最多 3 次
func (p *OpenAIProvider) doWithRetry(ctx context.Context, url string, body []byte, out *embedResponse) error {
	delays := []time.Duration{0, time.Second, 2 * time.Second, 4 * time.Second}
	var lastErr error

	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delays[attempt]):
			}
		}

		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+p.apiKey)

		resp, err := p.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("%w: status %d", ErrProviderUnavailable, resp.StatusCode)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("%w: status %d: %s", ErrProviderUnavailable, resp.StatusCode, bodyBytes)
		}

		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("%w: decode: %v", ErrProviderUnavailable, err)
		}
		return nil
	}
	return lastErr
}
```

- [ ] **Step 4: 跑测试通过**

Run: `cd backend && go test ./internal/platform/embedding/`
Expected: PASS（4 个测试）

> 429 重试测试若因 timing 偶发失败，可把 retry delays 暴露为可注入字段（test-only setter）跳过真实睡眠。本 plan 接受默认实现，按需调整。

- [ ] **Step 5: Commit**

```bash
git add backend/internal/platform/embedding/
git commit -m "feat(backend/embedding): add OpenAI-compatible provider with retry"
```

---

### Task 4: `platform/embedding/cosine.go` — 内存 brute-force 检索

**Files:**
- Create: `backend/internal/platform/embedding/cosine.go`
- Create: `backend/internal/platform/embedding/cosine_test.go`

> 用途：开发环境（SQLite 无 pgvector）的向量检索；生产用 pgvector `<=>` 算子，由 domain/knowledge 在 repo 层选择实现。

- [ ] **Step 1: 写失败测试**

`backend/internal/platform/embedding/cosine_test.go`:
```go
package embedding

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCosine_SameVector(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{1, 0, 0}
	assert.InDelta(t, 1.0, Cosine(a, b), 1e-6)
}

func TestCosine_Orthogonal(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{0, 1, 0}
	assert.InDelta(t, 0.0, Cosine(a, b), 1e-6)
}

func TestCosine_Opposite(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{-1, 0, 0}
	assert.InDelta(t, -1.0, Cosine(a, b), 1e-6)
}

func TestCosine_DimMismatch_Panics(t *testing.T) {
	defer func() { _ = recover() }()
	a := []float32{1, 0}
	b := []float32{1, 0, 0}
	_ = Cosine(a, b)
	t.Fatal("expected panic")
}

func TestTopK_Basic(t *testing.T) {
	vectors := [][]float32{
		{1, 0, 0},
		{0, 1, 0},
		{0.9, 0.1, 0},
		{0, 0, 1},
	}
	query := []float32{1, 0, 0}
	hits := TopK(query, vectors, 2)
	assert.Len(t, hits, 2)
	// 第一名应是 index 0（与 query 完全一致）
	assert.Equal(t, 0, hits[0].Index)
	assert.InDelta(t, 1.0, hits[0].Score, 1e-6)
	// 第二名应是 index 2（最接近 query）
	assert.Equal(t, 2, hits[1].Index)
}

type testVectorItem struct {
	Vector []float32
}

func TestTopK_EmptyQuery(t *testing.T) {
	hits := TopK(nil, [][]float32{{1, 0}}, 3)
	assert.Empty(t, hits)
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd backend && go test ./internal/platform/embedding/`
Expected: FAIL — `Cosine` / `TopK` 未定义

- [ ] **Step 3: 写实现**

`backend/internal/platform/embedding/cosine.go`:
```go
package embedding

import "sort"

type Hit struct {
	Index int
	Score float64
}

func Cosine(a, b []float32) float64 {
	if len(a) != len(b) {
		panic("cosine: dim mismatch")
	}
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (sqrt(na) * sqrt(nb))
}

func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	z := x
	for i := 0; i < 16; i++ {
		z = z - (z*z-x)/(2*z)
	}
	return z
}

func TopK(query []float32, vectors [][]float32, k int) []Hit {
	if query == nil || len(vectors) == 0 || k <= 0 {
		return nil
	}
	hits := make([]Hit, 0, len(vectors))
	for i, v := range vectors {
		hits = append(hits, Hit{Index: i, Score: Cosine(query, v)})
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })
	if k > len(hits) {
		k = len(hits)
	}
	return hits[:k]
}
```

> 自实现 sqrt 避免引入 `math`（虽然 math 是 stdlib，但本实现便于移植到无 math 的环境）。如需换成 `math.Sqrt` 更精确，本 plan 允许直接替换。

- [ ] **Step 4: 跑测试通过**

Run: `cd backend && go test ./internal/platform/embedding/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/platform/embedding/cosine.go backend/internal/platform/embedding/cosine_test.go
git commit -m "feat(backend/embedding): add cosine similarity and TopK helper"
```

---

### Task 5: `knowledge/repository.go` + `repo_gorm.go` + `inmemory_vector.go`

**Files:**
- Create: `backend/internal/domain/knowledge/repository.go`
- Create: `backend/internal/domain/knowledge/repo_gorm.go`
- Create: `backend/internal/domain/knowledge/inmemory_vector.go`

- [ ] **Step 1: 写接口**

`backend/internal/domain/knowledge/repository.go`:
```go
package knowledge

import (
	"context"

	"github.com/invest-guide/backend/internal/platform/response"
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
```

- [ ] **Step 2: 写 GORM 实现**

`backend/internal/domain/knowledge/repo_gorm.go`:
```go
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

// NewGORMChunkRepository 返回 ChunkRepository。在 SQLite（开发）下使用内存检索；
// 在 PG（生产）下走 pgvector 算子，由本 repo 内部分发。
// 本 plan 只实现 SQLite 路径；PG 路径留 TODO 由 inmemory_vector 包装。
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
```

- [ ] **Step 3: 写 inmemory_vector.go**

`backend/internal/domain/knowledge/inmemory_vector.go`:
```go
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
```

- [ ] **Step 4: 验证编译**

Run: `cd backend && go build ./...`
Expected: 无错误

- [ ] **Step 5: Commit**

```bash
git add backend/internal/domain/knowledge/repository.go backend/internal/domain/knowledge/repo_gorm.go backend/internal/domain/knowledge/inmemory_vector.go
git commit -m "feat(backend/knowledge): add Doc/Chunk repositories with in-memory vector search"
```

---

### Task 6: `knowledge/chunker.go` — 文本分块

**Files:**
- Create: `backend/internal/domain/knowledge/chunker.go`
- Create: `backend/internal/domain/knowledge/chunker_test.go`

- [ ] **Step 1: 写失败测试**

`backend/internal/domain/knowledge/chunker_test.go`:
```go
package knowledge

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 分块策略：512 tokens 近似 2048 chars（按 4 chars/token），10% overlap = 51 tokens ≈ 200 chars
// 测试改用更小的 chunk size 便于验证逻辑

func TestChunker_SplitsLongText(t *testing.T) {
	text := strings.Repeat("ab", 200) // 400 chars
	chunks := Chunk(text, 100, 10)    // chunkSize=100, overlap=10
	assert.GreaterOrEqual(t, len(chunks), 4)
	// 每块（除最后）应不超过 chunkSize
	for _, c := range chunks {
		assert.LessOrEqual(t, len(c), 100)
	}
}

func TestChunker_ShortText_OneChunk(t *testing.T) {
	text := "short text"
	chunks := Chunk(text, 100, 10)
	require.Len(t, chunks, 1)
	assert.Equal(t, text, chunks[0])
}

func TestChunker_OverlapBetweenChunks(t *testing.T) {
	text := "0123456789ABCDEFGHIJ" // 20 chars
	chunks := Chunk(text, 10, 4)   // chunkSize=10, overlap=4
	require.GreaterOrEqual(t, len(chunks), 2)
	// 第二块前 4 字符应与第一块后 4 字符一致
	assert.Equal(t, chunks[0][6:10], chunks[1][0:4])
}

func TestChunker_PreservesParagraphBoundaries(t *testing.T) {
	text := "First paragraph here.\n\nSecond paragraph here.\n\nThird one."
	chunks := Chunk(text, 1000, 0) // chunkSize 大到不分
	require.Len(t, chunks, 1)
	assert.Contains(t, chunks[0], "First paragraph")
	assert.Contains(t, chunks[0], "Third one")
}

func TestChunker_EmptyText(t *testing.T) {
	chunks := Chunk("", 100, 10)
	assert.Empty(t, chunks)
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd backend && go test ./internal/domain/knowledge/`
Expected: FAIL — `Chunk` 未定义

- [ ] **Step 3: 写实现**

`backend/internal/domain/knowledge/chunker.go`:
```go
package knowledge

// Chunk 按 chunkSize（字节近似 token）分块，相邻块之间有 overlap 字节重叠
// 优先在段落/换行/句末边界切，避免词中断
func Chunk(text string, chunkSize, overlap int) []string {
	if text == "" || chunkSize <= 0 {
		return nil
	}
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= chunkSize {
		overlap = chunkSize / 2
	}

	// 先按段落预切分
	paragraphs := splitParagraphs(text)
	var chunks []string
	var cur strings.Builder
	curLen := 0

	flush := func() {
		if curLen > 0 {
			chunks = append(chunks, cur.String())
			cur.Reset()
			curLen = 0
		}
	}

	for _, p := range paragraphs {
		// 段落本身超长 → 强切
		if len(p) > chunkSize {
			flush()
			for _, c := range splitBySize(p, chunkSize, overlap) {
				chunks = append(chunks, c)
			}
			continue
		}
		// 累加进当前块
		if curLen > 0 && curLen+len(p)+2 > chunkSize {
			flush()
		}
		if curLen > 0 {
			cur.WriteString("\n\n")
			curLen += 2
		}
		cur.WriteString(p)
		curLen += len(p)
	}
	flush()
	return chunks
}

func splitParagraphs(text string) []string {
	var out []string
	cur := ""
	for _, r := range text {
		if r == '\n' {
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func splitBySize(s string, chunkSize, overlap int) []string {
	var out []string
	step := chunkSize - overlap
	if step <= 0 {
		step = chunkSize
	}
	for i := 0; i < len(s); i += step {
		end := i + chunkSize
		if end > len(s) {
			end = len(s)
		}
		out = append(out, s[i:end])
		if end == len(s) {
			break
		}
	}
	return out
}
```

> 需要 `import "strings"`。在文件顶部添加。

- [ ] **Step 4: 跑测试通过**

Run: `cd backend && go test ./internal/domain/knowledge/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/domain/knowledge/chunker.go backend/internal/domain/knowledge/chunker_test.go
git commit -m "feat(backend/knowledge): add chunker with paragraph-aware splitting"
```

---

### Task 7: `knowledge/parser.go` — 文档解析（纯文本/markdown/html 简化版）

**Files:**
- Create: `backend/internal/domain/knowledge/parser.go`

> PDF 解析留待后续 plan（需引入第三方库如 unidocunio/pdfcpu），本 plan 只支持纯文本/markdown/HTML 简化处理。

- [ ] **Step 1: 写实现**

`backend/internal/domain/knowledge/parser.go`:
```go
package knowledge

import (
	"regexp"
	"strings"
)

// Parse 把原始文档内容（按 sourceType 不同格式）转为纯文本
// 本 plan 支持：manual, upload（默认按纯文本/markdown 处理）, url（暂未实现网络抓取，仅按 HTML 字符串处理）
func Parse(content string, sourceType string) (string, error) {
	switch sourceType {
	case SourceManual, SourceUpload:
		return stripMarkdown(content), nil
	case SourceURL:
		return stripHTML(content), nil
	default:
		return content, nil
	}
}

// stripMarkdown 移除 markdown 语法符号，保留纯文本
var mdCodeRe = regexp.MustCompile("`{1,3}[^`]*`{1,3}")
var mdLinkRe = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
var mdHeaderRe = regexp.MustCompile(`^#+\s*`)
var mdEmphRe = regexp.MustCompile(`[*_]+([^*_]+)[*_]+`)

func stripMarkdown(s string) string {
	s = mdCodeRe.ReplaceAllString(s, "$1")
	s = mdLinkRe.ReplaceAllString(s, "$1")
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		trimmed := mdHeaderRe.ReplaceAllString(line, "")
		lines[i] = trimmed
	}
	s = strings.Join(lines, "\n")
	s = mdEmphRe.ReplaceAllString(s, "$1")
	return s
}

// stripHTML 移除 HTML 标签
var htmlTagRe = regexp.MustCompile(`<[^>]+>`)

func stripHTML(s string) string {
	s = htmlTagRe.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	return s
}
```

- [ ] **Step 2: 验证编译**

Run: `cd backend && go build ./...`
Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add backend/internal/domain/knowledge/parser.go
git commit -m "feat(backend/knowledge): add markdown/HTML text parser"
```

---

### Task 8: `knowledge/pipeline.go` — 入库流水线

**Files:**
- Create: `backend/internal/domain/knowledge/pipeline.go`
- Create: `backend/internal/domain/knowledge/pipeline_test.go`

- [ ] **Step 1: 写失败测试（用 fake embedding）**

`backend/internal/domain/knowledge/pipeline_test.go`:
```go
package knowledge

import (
	"context"
	"testing"

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

func TestPipeline_Run ParsesChunksEmbedsStores(t *testing.T) {
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

	// 文档状态应为 ready
	d, _ := docRepo.Get(context.Background(), doc.ID)
	assert.Equal(t, StatusReady, d.Status)
	assert.Greater(t, d.ChunkCount, 0)

	// 检索应能找到 chunks
	vec := make([]float32, 4)
	hits, err := chunkRepo.SearchByVector(context.Background(), vec, 5, "")
	require.NoError(t, err)
	assert.NotEmpty(t, hits)
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

	doc := &KnowledgeDoc{ID: "doc-3", Title: "T", SourceType: SourceManual, Status: StatusPending}
	_ = docRepo.Create(context.Background(), doc)

	err := p.Run(context.Background(), doc.ID, "some content")
	// 前两次失败，第三次成功 → 整体成功
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
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd backend && go test ./internal/domain/knowledge/`
Expected: FAIL — `NewPipeline` 未定义

- [ ] **Step 3: 写实现**

`backend/internal/domain/knowledge/pipeline.go`:
```go
package knowledge

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/invest-guide/backend/internal/platform/embedding"
	"github.com/invest-guide/backend/internal/platform/response"
)

type Pipeline struct {
	docs       DocRepository
	chunks     ChunkRepository
	embed      embedding.Provider
	chunkSize  int
	overlap    int
}

const (
	pipelineChunkSize = 2048 // ~512 tokens (4 chars/token)
	pipelineOverlap   = 200  // ~50 tokens
	embedMaxRetries   = 3
)

func NewPipeline(docs DocRepository, chunks ChunkRepository, embed embedding.Provider, chunkSize, overlap int) *Pipeline {
	return &Pipeline{docs: docs, chunks: chunks, embed: embed, chunkSize: chunkSize, overlap: overlap}
}

// Run 执行完整流水线：Parse → Chunk → Embed → Store
// 失败时把文档状态置为 failed 并写 error_message
func (p *Pipeline) Run(ctx context.Context, docID, content string) error {
	// 1. 标记 processing
	if err := p.docs.Update(ctx, docID, UpdateDocParams{Status: strPtr(StatusProcessing)}); err != nil {
		return err
	}

	// 失败时统一处理
	fail := func(msg string) error {
		_ = p.docs.Update(ctx, docID, UpdateDocParams{
			Status:       strPtr(StatusFailed),
			ErrorMessage: strPtr(msg),
		})
		return fmt.Errorf("%s: %s", response.ErrInternal, msg)
	}

	// 2. 解析 + 分块
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

	// 3. Embedding（带重试）
	vecs, err := p.embedWithRetry(ctx, textChunks)
	if err != nil {
		return fail("embed: " + err.Error())
	}

	// 4. 构造 chunks 并存储
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

	// 5. 标记 ready
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
	delays := []time.Duration{0, time.Second, 2 * time.Second, 4 * time.Second}
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
	return err == embedding.ErrProviderUnavailable
}

func strPtr(s string) *string    { return &s }
func intPtr(i int) *int          { return &i }
```

- [ ] **Step 4: 跑测试通过**

Run: `cd backend && go test ./internal/domain/knowledge/`
Expected: PASS（3 个 pipeline 测试 + 5 个 chunker 测试）

> `TestPipeline_RetryOnEmbedFailure` 涉及 1s+2s 真实等待，可能 ~3s。若希望避免 sleep，可以重构让 delays 切片可注入。本 plan 接受默认实现。

- [ ] **Step 5: Commit**

```bash
git add backend/internal/domain/knowledge/pipeline.go backend/internal/domain/knowledge/pipeline_test.go
git commit -m "feat(backend/knowledge): add ingestion pipeline with parse/chunk/embed/store"
```

---

### Task 9: `knowledge/service.go` + `handler.go` + `route.go`

**Files:**
- Create: `backend/internal/domain/knowledge/service.go`
- Create: `backend/internal/domain/knowledge/handler.go`
- Create: `backend/internal/domain/knowledge/route.go`
- Create: `backend/internal/domain/knowledge/service_test.go`
- Create: `backend/internal/domain/knowledge/handler_test.go`

- [ ] **Step 1: 写 service.go**

`backend/internal/domain/knowledge/service.go`:
```go
package knowledge

import (
	"context"

	"github.com/google/uuid"
	"github.com/invest-guide/backend/internal/platform/embedding"
	"github.com/invest-guide/backend/internal/platform/response"
	"github.com/invest-guide/backend/internal/platform/taskqueue"
)

type Service struct {
	docs    DocRepository
	chunks  ChunkRepository
	embed   embedding.Provider
	queue   taskqueue.Queue
	cache   EmbeddingCache
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

	// 异步入库 — 通过 taskqueue 派发 pipeline.Run
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
	// 先删 chunks 再删 doc
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
	// 1. Embed 查询文本（查缓存）
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

	// 2. 向量检索
	chunks, err := s.chunks.SearchByVector(ctx, vec, topK, req.Country)
	if err != nil {
		return nil, err
	}

	// 3. 组装响应（带文档标题）
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
```

- [ ] **Step 2: 写 handler.go + route.go**

`backend/internal/domain/knowledge/handler.go`:
```go
package knowledge

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/invest-guide/backend/internal/platform/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	country := c.Query("country")

	docs, total, err := h.service.List(c.Request.Context(), page, pageSize, country)
	if err != nil {
		response.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":   docs,
			"total":   total,
			"hasMore": int64(page*pageSize) < total,
		},
	})
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateDocRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.ErrInvalidInput)
		return
	}
	dto, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Ok(c, http.StatusAccepted, dto)
}

func (h *Handler) Get(c *gin.Context) {
	dto, err := h.service.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Ok(c, http.StatusOK, dto)
}

func (h *Handler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), c.Param("id")); err != nil {
		response.Fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Retry(c *gin.Context) {
	if err := h.service.Retry(c.Request.Context(), c.Param("id")); err != nil {
		response.Fail(c, err)
		return
	}
	response.Ok(c, http.StatusAccepted, nil)
}
```

`backend/internal/domain/knowledge/route.go`:
```go
package knowledge

import "github.com/gin-gonic/gin"

// Register 在已鉴权的 v1 private group 下注册 knowledge-docs 路由
func Register(group *gin.RouterGroup, h *Handler) {
	docs := group.Group("/knowledge-docs")
	docs.GET("", h.List)
	docs.POST("", h.Create)
	docs.GET("/:id", h.Get)
	docs.DELETE("/:id", h.Delete)
	docs.POST("/:id/retry", h.Retry)
}
```

- [ ] **Step 3: 验证编译**

Run: `cd backend && go build ./...`
Expected: 无错误

- [ ] **Step 4: Commit**

```bash
git add backend/internal/domain/knowledge/service.go backend/internal/domain/knowledge/handler.go backend/internal/domain/knowledge/route.go
git commit -m "feat(backend/knowledge): add Service/Handler/Route with async pipeline"
```

---

### Task 10: 接入 `Deps` 与 `main.go`

**Files:**
- Modify: `backend/internal/platform/router/deps.go`
- Modify: `backend/internal/platform/router/router.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: deps.go 扩展 KnowledgeHandler + KnowledgeService**

`Deps` 新增字段：
```go
type Deps struct {
	// ...
	AuthHandler       *auth.Handler
	Authenticator     middleware.Authenticator
	KnowledgeHandler  *knowledge.Handler
	KnowledgeSearcher KnowledgeSearcher // 供 Plan 4 conversation 调用
}

// KnowledgeSearcher 抽象 knowledge.Service.Search，避免 conversation 反向 import knowledge
type KnowledgeSearcher interface {
	Search(ctx context.Context, req knowledge.SearchRequest) (*knowledge.SearchResponse, error)
}
```

修正 `registerPrivateRoutes` 装配 knowledge：
```go
func (d *Deps) registerPrivateRoutes(g *gin.RouterGroup) {
	if d.KnowledgeHandler != nil {
		knowledge.Register(g, d.KnowledgeHandler)
	}
}
```

`NewTestDeps` 注入 knowledge handler（用 SQLite + fake embedding）：
```go
func NewTestDeps(t *testing.T) *Deps {
	t.Helper()
	db := newTestSQLiteWithKnowledge(t)
	jwt := auth.NewJWTIssuer("test-secret", "investguide", 1<<30)
	authSvc := auth.NewService(auth.NewGORMUserRepository(db), jwt)
	fakeEmbed := &fakeTestEmbedding{dim: 8}
	knowledgeSvc := knowledge.NewService(
		knowledge.NewGORMDocRepository(db),
		knowledge.NewGORMChunkRepository(db),
		fakeEmbed,
		taskqueue.NewGoroutinePool(2, 4),
		nil, // cache disabled in test
	)
	// ...
	KnowledgeHandler: knowledge.NewHandler(knowledgeSvc),
	KnowledgeSearcher: knowledgeSvc,
	// ...
}
```

> `fakeTestEmbedding` 与 `newTestSQLiteWithKnowledge` 在 router_test.go 内定义（仅测试用 helper）。

- [ ] **Step 2: router_test.go 添加 helper**

```go
func newTestSQLiteWithKnowledge(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.NewSQLite(":memory:")
	if err != nil { t.Fatal(err) }
	if err := db.AutoMigrate(&auth.User{}, &knowledge.KnowledgeDoc{}, &knowledge.KnowledgeChunk{}); err != nil {
		t.Fatal(err)
	}
	return db
}

type fakeTestEmbedding struct{ dim int }
func (f *fakeTestEmbedding) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i := range texts {
		out[i] = make([]float32, f.dim)
	}
	return out, nil
}
func (f *fakeTestEmbedding) Dim() int { return f.dim }
```

- [ ] **Step 3: main.go 装配 knowledge**

```go
// 在 authSvc 装配之后：
embedProvider := embedding.NewOpenAIProvider(cfg.EmbeddingBaseURL, cfg.EmbeddingAPIKey, cfg.EmbeddingModel, atoi(cfg.EmbeddingDim, 1024))
knowledgeSvc := knowledge.NewService(
	knowledge.NewGORMDocRepository(db),
	knowledge.NewGORMChunkRepository(db),
	embedProvider,
	taskQ,
	lruEmbeddingCache{cache: cacheInst}, // 适配器，见 Step 4
)
// deps 字段扩展：
// KnowledgeHandler:  knowledge.NewHandler(knowledgeSvc),
// KnowledgeSearcher: knowledgeSvc,
```

- [ ] **Step 4: 实现 Cache 适配器**

由于 `cache.Cache` 存 `interface{}`，本 plan 提供一个专用 `lruEmbeddingCache` 适配器内部强转：

放在 `service.go` 末尾或单独文件 `cache_adapter.go`：
```go
type lruEmbeddingCache struct {
	cache interface {
		Get(string) (interface{}, bool)
		Set(string, interface{})
	}
}

func (l lruEmbeddingCache) Get(text string) ([]float32, bool) {
	v, ok := l.cache.Get(textHash(text))
	if !ok { return nil, false }
	if vec, ok := v.([]float32); ok { return vec, true }
	return nil, false
}

func (l lruEmbeddingCache) Set(text string, vec []float32) {
	l.cache.Set(textHash(text), vec)
}

func textHash(s string) string {
	// 简化：用字符串本身作 key；生产应换 sha256
	return "embed:" + s
}
```

- [ ] **Step 5: 跑测试**

Run: `cd backend && go test ./...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add backend/internal/platform/router/ backend/cmd/server/main.go backend/internal/domain/knowledge/cache_adapter.go
git commit -m "feat(backend/knowledge): wire knowledge routes + embedding cache"
```

---

### Task 11: E2E 测试

**Files:**
- Create: `backend/tests/e2e/knowledge_test.go`

- [ ] **Step 1: 写 E2E 测试**

`backend/tests/e2e/knowledge_test.go`:
```go
package e2e

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKnowledge_CreateThenGet(t *testing.T) {
	srv := NewTestServer()
	defer srv.Close()

	// 1. 注册并登录拿 token
	token := registerAndLogin(t, srv)

	// 2. 创建文档（异步入库）
	body := `{"title":"越南投资指南","country":"越南","sourceType":"manual","content":"越南的工业园区集中在北部。河内是首都。胡志明市是经济中心。"}`
	resp := doAuth(t, "POST", srv.URL+"/api/v1/knowledge-docs", body, token)
	require.Equal(t, http.StatusAccepted, resp.StatusCode)

	var createResp struct {
		Success bool `json:"success"`
		Data    struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(mustReadAll(resp.Body), &createResp))
	require.NotEmpty(t, createResp.Data.ID)

	// 3. 轮询状态直到 ready/failed
	var status string
	for i := 0; i < 20; i++ {
		time.Sleep(100 * time.Millisecond)
		gResp := doAuth(t, "GET", srv.URL+"/api/v1/knowledge-docs/"+createResp.Data.ID, "", token)
		var g struct {
			Success bool `json:"success"`
			Data    struct {
				Status     string `json:"status"`
				ChunkCount int    `json:"chunkCount"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(mustReadAll(gResp.Body), &g))
		status = g.Data.Status
		if status == "ready" || status == "failed" { break }
	}
	assert.Equal(t, "ready", status)

	// 4. 列表
	lResp := doAuth(t, "GET", srv.URL+"/api/v1/knowledge-docs", "", token)
	require.Equal(t, http.StatusOK, lResp.StatusCode)
}

func TestKnowledge_Unauthenticated_Blocked(t *testing.T) {
	srv := NewTestServer()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/knowledge-docs")
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// helpers — 复用 auth_test.go 的 register/login 流程
func registerAndLogin(t *testing.T, srv *TestServer) string {
	t.Helper()
	body := `{"email":"k@b.com","password":"password123","displayName":"K"}`
	resp := postJSON(t, srv.URL+"/api/v1/auth/register", body, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var r struct{ Data struct{ Token string `json:"token"` } `json:"data"` }
	require.NoError(t, json.Unmarshal(mustReadAll(resp.Body), &r))
	return r.Data.Token
}

func doAuth(t *testing.T, method, url, body, token string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(method, url, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" { req.Header.Set("Authorization", "Bearer "+token) }
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}
```

> 需要 `import "bytes"`。在 e2e auth_test.go 已定义的 `postJSON`/`mustReadAll` 不需重复。

- [ ] **Step 2: 跑测试**

Run: `cd backend && go test ./tests/e2e/`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add backend/tests/e2e/knowledge_test.go
git commit -m "test(backend/e2e): add knowledge doc create + poll + list flow"
```

---

### Task 12: 最终验证

- [ ] **Step 1: gofmt + go vet + go test ./... -cover**

Run:
```bash
cd backend && gofmt -l . && go vet ./... && go test ./... -cover
```
Expected: 全 PASS；`internal/domain/knowledge/` 覆盖率 ≥ 60%（pipeline 因含 retry 等较难全覆盖，target 略低于其他）

- [ ] **Step 2: 启动 + curl**

Run: `make backend-dev`
另开终端：
```bash
TOKEN=$(curl -sX POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"k@b.com","password":"password123","displayName":"K"}' \
  | jq -r .data.token)

curl -i -X POST http://localhost:8080/api/v1/knowledge-docs \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"越南","country":"越南","sourceType":"manual","content":"河内是首都。胡志明市是经济中心。"}'
# 期望 202

curl -i http://localhost:8080/api/v1/knowledge-docs \
  -H "Authorization: Bearer $TOKEN"
# 期望 200 + items
```

- [ ] **Step 3: 标记完成**

Plan 3 完成。`domain/knowledge` 模块可独立交付；`KnowledgeSearcher` 接口已暴露供 Plan 4 conversation RAG 集成。

---

## 执行记录（2026-08-02）

本 plan 已按上述步骤落地，执行中做了以下修正与补充：

1. **`registerPrivateRoutes` 改用 `PrivateRoutes` 字段**（Plan 2 已改）：Task 10 中
   `deps.PrivateRoutes = func(g){ knowledge.Register(g, handler) }`，而非 plan 里的方法形式。
2. **`JSONFloat32` 需实现 `driver.Valuer` + `sql.Scanner`**：否则 SQLite 存储 `[]float32`
   报 "row value misused"。已加 `Value()`/`Scan()`（JSON 编解码）。
3. **`mdCodeRe` 正则缺捕获组**：plan 里 `(?:`*[^`]*`*` 无捕获组，`ReplaceAllString(s,"$1")`
   无效。已改为 `` `{1,3}([^`]*)`{1,3} ``。
4. **pipeline 的 `fail` 用 `%w` 包装**：plan 用 `fmt.Errorf("%s: %s", response.ErrInternal, msg)`
   丢失错误链，已改为 `fmt.Errorf("%w: %s", ...)` 使 `errors.Is` 可识别。
5. **embedding/pipeline 重试延迟可注入**：`retryDelays` 字段供测试注入零延迟，避免测试 sleep 3s。
6. **`CreateDocRequest.Content` binding**：改为 `required_if=SourceType manual,required_if=SourceType upload`
   （upload 也需要 content，对齐 API 文档）。
7. **覆盖率补充**：knowledge 66.3% → 76.5%。补了 handler List/Retry、parser stripHTML/URL 分支测试。
8. **e2e/helpers.go 更新**：`NewTestServer` 装配 knowledge（fake embed + 同步 e2eQueue），
   `PrivateRoutes` 注入 knowledge 路由；`TestKnowledge_Unauthenticated_Blocked` 验证 401。
9. **Task 9 原本无 service/handler 测试**：补了 service_test.go（Create/Get/Delete/Retry/Search）
   与 handler_test.go（Create 202/Invalid/Get 404/List/Retry）。

覆盖率（达标 ≥70%）：knowledge 76.5% · embedding 81.8%。
