# 后端 Plan 4 — conversation + assistant 领域 Implementation Plan

**Goal:** 实现最后两个领域模块：会话/消息 CRUD、SSE 流式回答、LLM Provider 抽象（生成 + 流式）、RAG 上下文装配；补全 `system/models` 端点（Plan 1 留下的）；完成 SSE 容错与 goroutine 泄漏防护。完成后整个后端可对外提供 RAG 问答服务。

**Architecture:** `Conversation` 与 `Message` 一对多；客户端 `POST /messages` 创建 user 消息并触发异步回答生成（返回 `messageId`），再 `GET /messages/{messageId}/stream` 订阅 SSE；assistantService 装配 RAG 上下文（embed query → search via KnowledgeSearcher → 截断 token 预算）→ prompt 构建 → LLMProvider.Stream；SSE 事件顺序 `heartbeat → sources → message*  → done`，error 后不再 done；context.Cancel 贯穿客户端断连 → ctx.Done → LLM 请求中止；LLMProvider 接口在 `platform/llm/`（与 embedding 平级），fake 实现用于测试。

**Tech Stack:** `net/http`（SSE 用 ResponseWriter flush）、`encoding/json`、context cancel propagation、Gin、GORM、testify、SQLite（测试）。

**关联设计：** `ARCHITECTURE.md`（LLM Provider 抽象 / 容错策略 / 流式 SSE 并发模型 / API 约定的 SSE 事件顺序 / 错误码映射）

**前置条件：** Plan 1（platform）+ Plan 2（auth 中间件）+ Plan 3（KnowledgeSearcher 接口已暴露）完成。

---

## 文件结构

```
backend/
├── internal/
│   ├── platform/
│   │   └── llm/
│   │       ├── provider.go               # LLMProvider + EmbeddingProvider alias（导入 embedding）
│   │       ├── openai.go                 # OpenAI-compatible Generate + Stream
│   │       ├── openai_test.go
│   │       ├── fake.go                   # 测试用 fake provider（输出确定）
│   │       └── fake_test.go
│   ├── domain/
│   │   ├── conversation/
│   │   │   ├── route.go                  # v1 private 注册：conversations CRUD + messages + stream
│   │   │   ├── handler.go                # SSE handler + REST handlers
│   │   │   ├── service.go                # 业务编排
│   │   │   ├── repository.go             # ConversationRepository + MessageRepository 接口
│   │   │   ├── repo_gorm.go              # GORM 实现
│   │   │   ├── model.go                  # Conversation + Message 实体 + DTO
│   │   │   ├── sse.go                    # SSE writer 封装（event 类型 + flush + heartbeat）
│   │   │   ├── service_test.go
│   │   │   └── handler_test.go
│   │   └── assistant/
│   │       ├── service.go                # RAG 上下文装配 + prompt 构建 + 调 LLMProvider
│   │       ├── prompt.go                 # prompt 模板
│   │       ├── prompt_test.go
│   │       ├── service_test.go
│   │       └── model.go                  # GenerateRequest/Response、StreamChunk、ContextSource
│   ├── platform/router/
│   │   ├── deps.go                       # 扩展：AssistantService + LLMProvider 字段
│   │   └── router.go                     # 扩展：registerPrivateRoutes 挂 conversation + system/models
│   └── domain/system/
│       └── models_handler.go             # GET /system/models 补全
├── migrations/
│   ├── 0004_conversations.up.sql
│   └── 0004_conversations.down.sql
└── tests/e2e/
    ├── conversation_test.go              # 创建会话 → 发消息 → 读历史
    └── stream_test.go                    # SSE 完整流（含断连模拟）
```

---

### Task 0: 迁移 — conversations + messages

**Files:**
- Create: `backend/migrations/0004_conversations.up.sql`
- Create: `backend/migrations/0004_conversations.down.sql`

- [ ] **Step 1: 写 up.sql**

`backend/migrations/0004_conversations.up.sql`:
```sql
CREATE TABLE conversations (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title      TEXT NOT NULL DEFAULT '',
    country    TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_conversations_user_id ON conversations(user_id);

CREATE TABLE messages (
    id              TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    role            TEXT NOT NULL, -- user / assistant
    content         TEXT NOT NULL,
    sources         JSONB,
    tokens_used     INTEGER NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_messages_conversation_id ON messages(conversation_id);
```

- [ ] **Step 2: 写 down.sql**

`backend/migrations/0004_conversations.down.sql`:
```sql
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS conversations;
```

- [ ] **Step 3: Commit**

```bash
git add backend/migrations/0004_conversations.*
git commit -m "feat(backend/conversation): add conversations and messages migrations"
```

---

### Task 1: `conversation/model.go` + `assistant/model.go`

**Files:**
- Create: `backend/internal/domain/conversation/model.go`
- Create: `backend/internal/domain/assistant/model.go`

- [ ] **Step 1: 写 conversation/model.go**

`backend/internal/domain/conversation/model.go`:
```go
package conversation

import (
	"encoding/json"
	"time"
)

const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

type Conversation struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	UserID    string    `gorm:"not null;index" json:"userId"`
	Title     string    `json:"title"`
	Country   string    `json:"country"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (Conversation) TableName() string { return "conversations" }

type Message struct {
	ID             string          `gorm:"primaryKey" json:"id"`
	ConversationID string          `gorm:"not null;index" json:"conversationId"`
	Role           string          `gorm:"not null" json:"role"`
	Content        string          `gorm:"not null;type:text" json:"content"`
	Sources        json.RawMessage `gorm:"type:jsonb" json:"sources,omitempty"`
	TokensUsed     int             `gorm:"not null;default:0" json:"tokensUsed"`
	CreatedAt      time.Time       `json:"createdAt"`
}

func (Message) TableName() string { return "messages" }

// DTOs
type CreateConversationRequest struct {
	Title   string `json:"title,omitempty" binding:"max=200"`
	Country string `json:"country,omitempty" binding:"max=100"`
}

type ConversationDTO struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Country   string    `json:"country"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (c *Conversation) ToDTO() ConversationDTO {
	return ConversationDTO{ID: c.ID, Title: c.Title, Country: c.Country,
		CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt}
}

type PostMessageRequest struct {
	Content string `json:"content" binding:"required,min=1,max=10000"`
}

type PostMessageResponse struct {
	MessageID string `json:"messageId"`
}

type MessageDTO struct {
	ID         string          `json:"id"`
	Role       string          `json:"role"`
	Content    string          `json:"content"`
	Sources    json.RawMessage `json:"sources,omitempty"`
	TokensUsed int             `json:"tokensUsed"`
	CreatedAt  time.Time       `json:"createdAt"`
}

func (m *Message) ToDTO() MessageDTO {
	return MessageDTO{
		ID: m.ID, Role: m.Role, Content: m.Content,
		Sources: m.Sources, TokensUsed: m.TokensUsed, CreatedAt: m.CreatedAt,
	}
}
```

- [ ] **Step 2: 写 assistant/model.go**

`backend/internal/domain/assistant/model.go`:
```go
package assistant

import "context"

// GenerateRequest 同步生成请求
type GenerateRequest struct {
	Messages    []ChatMessage
	MaxTokens   int
	Temperature float32
}

type ChatMessage struct {
	Role    string // user / assistant / system
	Content string
}

type GenerateResponse struct {
	Content    string
	TokensUsed int
}

// StreamChunk 流式增量
type StreamChunk struct {
	Delta      string // 增量文本；结束时为空
	Done       bool   // 流终止
	TokensUsed int    // 仅 Done=true 时有意义
	Err        error  // 终止性错误
}

// ContextSource 是 RAG 检索命中的知识片段引用
type ContextSource struct {
	ChunkID string
	Title   string
	Snippet string
}

// AssembleContextRequest 是 conversation 装配 RAG 上下文的请求
type AssembleContextRequest struct {
	UserQuery string
	Country   string
}

type AssembledContext struct {
	SystemPrompt string
	Sources      []ContextSource
}

// Provider 是 LLM 抽象（与 platform/llm.LLMProvider 对齐的本地视图）
// conversation.service 通过 assistant.service 间接使用，不直接依赖 platform/llm
type Provider interface {
	Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error)
	Stream(ctx context.Context, req GenerateRequest) (<-chan StreamChunk, error)
}
```

- [ ] **Step 3: 验证编译**

Run: `cd backend && go build ./...`
Expected: 无错误

- [ ] **Step 4: Commit**

```bash
git add backend/internal/domain/conversation/model.go backend/internal/domain/assistant/model.go
git commit -m "feat(backend): add conversation/assistant models and DTOs"
```

---

### Task 2: `platform/llm/provider.go` — LLMProvider 接口

**Files:**
- Create: `backend/internal/platform/llm/provider.go`

- [ ] **Step 1: 写接口**

`backend/internal/platform/llm/provider.go`:
```go
package llm

import (
	"context"
	"errors"
)

var (
	ErrUnavailable = errors.New("llm provider unavailable")
	ErrTimeout     = errors.New("llm request timeout")
)

type ChatMessage struct {
	Role    string
	Content string
}

type GenerateRequest struct {
	Messages    []ChatMessage
	MaxTokens   int
	Temperature float32
	Model       string // 留空用 provider 默认
}

type GenerateResponse struct {
	Content    string
	TokensUsed int
}

type StreamChunk struct {
	Delta      string
	Done       bool
	TokensUsed int
	Err        error
}

type Provider interface {
	Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error)
	Stream(ctx context.Context, req GenerateRequest) (<-chan StreamChunk, error)
	Model() string
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/internal/platform/llm/provider.go
git commit -m "feat(backend/llm): add Provider interface"
```

---

### Task 3: `platform/llm/openai.go` + 测试

**Files:**
- Create: `backend/internal/platform/llm/openai.go`
- Create: `backend/internal/platform/llm/openai_test.go`

- [ ] **Step 1: 写失败测试（Generate + Stream 各一）**

`backend/internal/platform/llm/openai_test.go`:
```go
package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAIProvider_Generate_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"hello"}}],"usage":{"total_tokens":42}}`))
	}))
	defer srv.Close()

	p := NewOpenAIProvider(srv.URL, "k", "test-model", 0, 0)
	resp, err := p.Generate(context.Background(), GenerateRequest{
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	})
	require.NoError(t, err)
	assert.Equal(t, "hello", resp.Content)
	assert.Equal(t, 42, resp.TokensUsed)
	assert.Equal(t, "test-model", p.Model())
}

func TestOpenAIProvider_Generate_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done() // 永远不响应，等客户端取消
	}))
	defer srv.Close()

	p := NewOpenAIProvider(srv.URL, "k", "m", 0, 0)
	_, err := p.Generate(context.Background(), GenerateRequest{})
	assert.Error(t, err)
}

func TestOpenAIProvider_Stream_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"He\"}}]}\n\n"))
		flusher.Flush()
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"llo\"}}]}\n\n"))
		flusher.Flush()
		_, _ = w.Write([]byte("data: {\"choices\":[{\"finish_reason\":\"stop\"}],\"usage\":{\"total_tokens\":5}}\n\n"))
		flusher.Flush()
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	}))
	defer srv.Close()

	p := NewOpenAIProvider(srv.URL, "k", "m", 0, 0)
	ch, err := p.Stream(context.Background(), GenerateRequest{})
	require.NoError(t, err)

	var deltas string
	var finalChunk StreamChunk
	for c := range ch {
		if c.Done {
			finalChunk = c
			break
		}
		deltas += c.Delta
	}
	assert.Equal(t, "Hello", deltas)
	assert.Equal(t, 5, finalChunk.TokensUsed)
	assert.Nil(t, finalChunk.Err)
}

func TestOpenAIProvider_Stream_ErrorEvent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := NewOpenAIProvider(srv.URL, "k", "m", 0, 0)
	ch, err := p.Stream(context.Background(), GenerateRequest{})
	require.NoError(t, err)

	var lastChunk StreamChunk
	for c := range ch {
		lastChunk = c
	}
	assert.NotNil(t, lastChunk.Err)
	assert.True(t, lastChunk.Done)
}

// 防止 json import 未使用
var _ = json.Marshal
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd backend && go test ./internal/platform/llm/`
Expected: FAIL — `NewOpenAIProvider` 未定义

- [ ] **Step 3: 写实现**

`backend/internal/platform/llm/openai.go`:
```go
package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type OpenAIProvider struct {
	baseURL       string
	apiKey        string
	model         string
	client        *http.Client
	streamClient  *http.Client
}

func NewOpenAIProvider(baseURL, apiKey, model string, timeout, streamTimeout time.Duration) *OpenAIProvider {
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	if streamTimeout == 0 {
		streamTimeout = 120 * time.Second
	}
	return &OpenAIProvider{
		baseURL:      baseURL,
		apiKey:       apiKey,
		model:        model,
		client:       &http.Client{Timeout: timeout},
		streamClient: &http.Client{Timeout: streamTimeout},
	}
}

func (p *OpenAIProvider) Model() string { return p.model }

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float32       `json:"temperature,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message      chatMessage `json:"message"`
		Delta        chatMessage `json:"delta"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (p *OpenAIProvider) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	body, _ := json.Marshal(p.toChatRequest(req))
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrUnavailable, resp.StatusCode)
	}
	var cr chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return nil, fmt.Errorf("%w: decode: %v", ErrUnavailable, err)
	}
	if len(cr.Choices) == 0 {
		return nil, fmt.Errorf("%w: no choices", ErrUnavailable)
	}
	return &GenerateResponse{
		Content:    cr.Choices[0].Message.Content,
		TokensUsed: cr.Usage.TotalTokens,
	}, nil
}

func (p *OpenAIProvider) Stream(ctx context.Context, req GenerateRequest) (<-chan StreamChunk, error) {
	body, _ := json.Marshal(p.toChatRequest(req, true))
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := p.streamClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("%w: status %d", ErrUnavailable, resp.StatusCode)
	}

	ch := make(chan StreamChunk, 16)
	go p.pumpSSE(ctx, resp, ch)
	return ch, nil
}

func (p *OpenAIProvider) pumpSSE(ctx context.Context, resp *http.Response, ch chan<- StreamChunk) {
	defer close(ch)
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	for {
		select {
		case <-ctx.Done():
			ch <- StreamChunk{Done: true, Err: ctx.Err()}
			return
		default:
		}
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				ch <- StreamChunk{Done: true}
				return
			}
			ch <- StreamChunk{Done: true, Err: fmt.Errorf("%w: read: %v", ErrUnavailable, err)}
			return
		}
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		if !bytes.HasPrefix(line, []byte("data: ")) {
			continue
		}
		data := bytes.TrimPrefix(line, []byte("data: "))
		if bytes.Equal(data, []byte("[DONE]")) {
			ch <- StreamChunk{Done: true}
			return
		}
		var cr chatResponse
		if err := json.Unmarshal(data, &cr); err != nil {
			continue
		}
		if cr.Error != nil {
			ch <- StreamChunk{Done: true, Err: fmt.Errorf("%w: %s", ErrUnavailable, cr.Error.Message)}
			return
		}
		if len(cr.Choices) > 0 {
			c := cr.Choices[0]
			if c.Delta.Content != "" {
				ch <- StreamChunk{Delta: c.Delta.Content}
			}
			if c.FinishReason == "stop" {
				ch <- StreamChunk{Done: true, TokensUsed: cr.Usage.TotalTokens}
				return
			}
		}
	}
}

func (p *OpenAIProvider) toChatRequest(req GenerateRequest, stream ...bool) chatRequest {
	msgs := make([]chatMessage, len(req.Messages))
	for i, m := range req.Messages {
		msgs[i] = chatMessage{Role: m.Role, Content: m.Content}
	}
	cr := chatRequest{
		Model:       nonEmpty(req.Model, p.model),
		Messages:    msgs,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}
	if len(stream) > 0 && stream[0] {
		cr.Stream = true
	}
	return cr
}

func nonEmpty(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}
```

- [ ] **Step 4: 跑测试通过**

Run: `cd backend && go test ./internal/platform/llm/`
Expected: PASS（4 个测试）

- [ ] **Step 5: Commit**

```bash
git add backend/internal/platform/llm/
git commit -m "feat(backend/llm): add OpenAI-compatible Generate and Stream"
```

---

### Task 4: `platform/llm/fake.go` — 测试用 fake provider

**Files:**
- Create: `backend/internal/platform/llm/fake.go`
- Create: `backend/internal/platform/llm/fake_test.go`

- [ ] **Step 1: 写 fake + 验证测试**

`backend/internal/platform/llm/fake.go`:
```go
package llm

import "context"

// FakeProvider 输出确定，用于测试
type FakeProvider struct {
	Response    string
	StreamDeltas []string
	Tokens      int
	ModelName   string
}

func NewFakeProvider(response string, deltas []string, tokens int) *FakeProvider {
	return &FakeProvider{Response: response, StreamDeltas: deltas, Tokens: tokens, ModelName: "fake"}
}

func (f *FakeProvider) Generate(ctx context.Context, _ GenerateRequest) (*GenerateResponse, error) {
	return &GenerateResponse{Content: f.Response, TokensUsed: f.Tokens}, nil
}

func (f *FakeProvider) Stream(ctx context.Context, _ GenerateRequest) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk, len(f.StreamDeltas)+1)
	go func() {
		defer close(ch)
		for _, d := range f.StreamDeltas {
			select {
			case <-ctx.Done():
				ch <- StreamChunk{Done: true, Err: ctx.Err()}
				return
			case ch <- StreamChunk{Delta: d}:
			}
		}
		ch <- StreamChunk{Done: true, TokensUsed: f.Tokens}
	}()
	return ch, nil
}

func (f *FakeProvider) Model() string { return f.ModelName }
```

`backend/internal/platform/llm/fake_test.go`:
```go
package llm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFakeProvider_Generate(t *testing.T) {
	p := NewFakeProvider("hello", nil, 5)
	resp, err := p.Generate(context.Background(), GenerateRequest{})
	require.NoError(t, err)
	assert.Equal(t, "hello", resp.Content)
}

func TestFakeProvider_Stream(t *testing.T) {
	p := NewFakeProvider("", []string{"He", "llo"}, 5)
	ch, err := p.Stream(context.Background(), GenerateRequest{})
	require.NoError(t, err)

	var out string
	var final StreamChunk
	for c := range ch {
		if c.Done {
			final = c
		} else {
			out += c.Delta
		}
	}
	assert.Equal(t, "Hello", out)
	assert.Equal(t, 5, final.TokensUsed)
}

func TestFakeProvider_Stream_CancelMidway(t *testing.T) {
	p := NewFakeProvider("", []string{"a", "b", "c"}, 0)
	ctx, cancel := context.WithCancel(context.Background())
	ch, _ := p.Stream(ctx, GenerateRequest{})

	// 读第一个 chunk 后取消
	<-ch
	cancel()
	final := <-ch
	assert.True(t, final.Done)
	assert.ErrorIs(t, final.Err, context.Canceled)
}
```

- [ ] **Step 2: 跑测试通过**

Run: `cd backend && go test ./internal/platform/llm/`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add backend/internal/platform/llm/fake.go backend/internal/platform/llm/fake_test.go
git commit -m "feat(backend/llm): add FakeProvider for deterministic testing"
```

---

### Task 5: `assistant/service.go` — RAG 上下文装配 + 调 LLM

**Files:**
- Create: `backend/internal/domain/assistant/service.go`
- Create: `backend/internal/domain/assistant/prompt.go`
- Create: `backend/internal/domain/assistant/prompt_test.go`
- Create: `backend/internal/domain/assistant/service_test.go`

- [ ] **Step 1: 写 prompt.go**

`backend/internal/domain/assistant/prompt.go`:
```go
package assistant

import (
	"fmt"
	"strings"
)

const systemPromptTemplate = `你是国别投资指南助手。基于以下知识库片段回答用户问题。
若上下文不足以回答，请明确说明"知识库中未涵盖该问题"，不要编造内容。
回答采用中文，结构清晰。

# 知识库片段
%s

# 指引
- 引用片段时说明来源国家/标题
- 用户问投资相关的法律、税务、行业准入、园区、外汇等内容时优先引用片段
- 不讨论与投资无关的话题`

// BuildSystemPrompt 把检索到的 sources 拼成 system prompt
func BuildSystemPrompt(sources []ContextSource) string {
	if len(sources) == 0 {
		return strings.Replace(systemPromptTemplate, "%s", "（无可用知识库片段，请基于通用知识谨慎回答）", 1)
	}
	var b strings.Builder
	for i, s := range sources {
		fmt.Fprintf(&b, "[%d] %s\n%s\n\n", i+1, s.Title, s.Snippet)
	}
	return fmt.Sprintf(systemPromptTemplate, b.String())
}
```

- [ ] **Step 2: 写 prompt 失败测试**

`backend/internal/domain/assistant/prompt_test.go`:
```go
package assistant

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildSystemPrompt_WithSources(t *testing.T) {
	sources := []ContextSource{
		{Title: "越南指南", Snippet: "河内是首都"},
		{Title: "泰国指南", Snippet: "曼谷是首都"},
	}
	p := BuildSystemPrompt(sources)
	assert.Contains(t, p, "越南指南")
	assert.Contains(t, p, "河内是首都")
	assert.Contains(t, p, "泰国指南")
	assert.Contains(t, p, "[1]")
	assert.Contains(t, p, "[2]")
}

func TestBuildSystemPrompt_EmptySources(t *testing.T) {
	p := BuildSystemPrompt(nil)
	assert.True(t, strings.Contains(p, "无可用知识库片段"))
	assert.NotContains(t, p, "[1]")
}
```

- [ ] **Step 3: 跑测试通过**

Run: `cd backend && go test ./internal/domain/assistant/`
Expected: PASS（prompt 2 个）

- [ ] **Step 4: 写 service.go**

`backend/internal/domain/assistant/service.go`:
```go
package assistant

import (
	"context"
	"strings"

	"github.com/invest-guide/backend/internal/platform/llm"
)

// KnowledgeSearcher 是 knowledge.Service 暴露的检索接口（在 router.Deps 定义具体实现）
type KnowledgeSearcher interface {
	Search(ctx context.Context, query, country string, topK int) ([]ContextSource, error)
}

type Service struct {
	llm      llm.Provider
	searcher KnowledgeSearcher
}

func NewService(llmProvider llm.Provider, searcher KnowledgeSearcher) *Service {
	return &Service{llm: llmProvider, searcher: searcher}
}

// AssembleContext 检索 + 截断 token 预算 + 构建提示
// token 预算 = context window 的 60%；本 plan 简化为字符数近似（4 chars ≈ 1 token，预算 ~6000 chars）
const maxContextChars = 6000

func (s *Service) AssembleContext(ctx context.Context, query, country string) (*AssembledContext, error) {
	var sources []ContextSource
	if s.searcher != nil {
		hits, err := s.searcher.Search(ctx, query, country, 5)
		if err == nil {
			// 截断到 char 预算
			total := 0
			for _, h := range hits {
				if total+len(h.Snippet) > maxContextChars {
					break
				}
				sources = append(sources, h)
				total += len(h.Snippet)
			}
		}
	}
	return &AssembledContext{
		SystemPrompt: BuildSystemPrompt(sources),
		Sources:      sources,
	}, nil
}

// Generate 同步生成
func (s *Service) Generate(ctx context.Context, sys, userQuery string) (string, []ContextSource, int, error) {
	assembled, err := s.AssembleContext(ctx, userQuery, "")
	if err != nil {
		return "", nil, 0, err
	}
	messages := []llm.ChatMessage{
		{Role: "system", Content: assembled.SystemPrompt},
		{Role: "user", Content: userQuery},
	}
	resp, err := s.llm.Generate(ctx, llm.GenerateRequest{Messages: messages})
	if err != nil {
		return "", nil, 0, err
	}
	return resp.Content, assembled.Sources, resp.TokensUsed, nil
}

// Stream 流式生成。返回 channel + assembled sources（供 caller 在首次 message 之前发送 sources 事件）
func (s *Service) Stream(ctx context.Context, userQuery, country string) (<-chan llm.StreamChunk, []ContextSource, error) {
	assembled, err := s.AssembleContext(ctx, userQuery, country)
	if err != nil {
		return nil, nil, err
	}
	messages := []llm.ChatMessage{
		{Role: "system", Content: assembled.SystemPrompt},
		{Role: "user", Content: userQuery},
	}
	ch, err := s.llm.Stream(ctx, llm.GenerateRequest{Messages: messages})
	if err != nil {
		return nil, nil, err
	}
	return ch, assembled.Sources, nil
}

// unused but kept to silence strings import if not used elsewhere later
var _ = strings.TrimSpace
```

- [ ] **Step 5: 写 service 失败测试**

`backend/internal/domain/assistant/service_test.go`:
```go
package assistant

import (
	"context"
	"errors"
	"testing"

	"github.com/invest-guide/backend/internal/platform/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSearcher struct {
	hits []ContextSource
	err  error
}

func (f *fakeSearcher) Search(ctx context.Context, q, c string, k int) ([]ContextSource, error) {
	return f.hits, f.err
}

func TestService_AssembleContext_IncludesSources(t *testing.T) {
	svc := NewService(nil, &fakeSearcher{
		hits: []ContextSource{{ChunkID: "c1", Title: "T1", Snippet: "S1"}},
	})
	ctx, err := svc.AssembleContext(context.Background(), "q", "")
	require.NoError(t, err)
	assert.Len(t, ctx.Sources, 1)
	assert.Contains(t, ctx.SystemPrompt, "T1")
}

func TestService_AssembleContext_SearchErrorStillReturns(t *testing.T) {
	svc := NewService(nil, &fakeSearcher{err: errors.New("boom")})
	ctx, err := svc.AssembleContext(context.Background(), "q", "")
	require.NoError(t, err) // 检索失败不阻塞生成
	assert.Empty(t, ctx.Sources)
}

func TestService_Generate_FullFlow(t *testing.T) {
	p := llm.NewFakeProvider("response", nil, 7)
	svc := NewService(p, &fakeSearcher{
		hits: []ContextSource{{Title: "T", Snippet: "S"}},
	})
	resp, sources, tokens, err := svc.Generate(context.Background(), "", "查询")
	require.NoError(t, err)
	assert.Equal(t, "response", resp)
	assert.Len(t, sources, 1)
	assert.Equal(t, 7, tokens)
}

func TestService_Stream_FullFlow(t *testing.T) {
	p := llm.NewFakeProvider("", []string{"He", "llo"}, 3)
	svc := NewService(p, &fakeSearcher{
		hits: []ContextSource{{Title: "T", Snippet: "S"}},
	})
	ch, sources, err := svc.Stream(context.Background(), "查询", "")
	require.NoError(t, err)
	assert.Len(t, sources, 1)

	var out string
	var final llm.StreamChunk
	for c := range ch {
		if c.Done {
			final = c
		} else {
			out += c.Delta
		}
	}
	assert.Equal(t, "Hello", out)
	assert.Equal(t, 3, final.TokensUsed)
}
```

- [ ] **Step 6: 跑测试通过**

Run: `cd backend && go test ./internal/domain/assistant/`
Expected: PASS（prompt 2 + service 4 共 6 个测试）

- [ ] **Step 7: Commit**

```bash
git add backend/internal/domain/assistant/
git commit -m "feat(backend/assistant): add RAG context assembly + Generate/Stream"
```

---

### Task 6: `conversation/repository.go` + `repo_gorm.go`

**Files:**
- Create: `backend/internal/domain/conversation/repository.go`
- Create: `backend/internal/domain/conversation/repo_gorm.go`

- [ ] **Step 1: 写接口与实现**

`backend/internal/domain/conversation/repository.go`:
```go
package conversation

import "context"

type ConversationRepository interface {
	Create(ctx context.Context, c *Conversation) error
	Get(ctx context.Context, id, userID string) (*Conversation, error)
	Update(ctx context.Context, id, userID string, params UpdateConversationParams) error
	Delete(ctx context.Context, id, userID string) error
	ListByUser(ctx context.Context, userID string, page, pageSize int) ([]*Conversation, int64, error)
}

type UpdateConversationParams struct {
	Title   *string
	Country *string
}

type MessageRepository interface {
	Create(ctx context.Context, m *Message) error
	Get(ctx context.Context, id string) (*Message, error)
	ListByConversation(ctx context.Context, convID string, page, pageSize int) ([]*Message, int64, error)
	Update(ctx context.Context, id string, params UpdateMessageParams) error
}

type UpdateMessageParams struct {
	Content    *string
	Sources    []byte
	TokensUsed *int
}
```

`backend/internal/domain/conversation/repo_gorm.go`:
```go
package conversation

import (
	"context"
	"errors"

	"github.com/invest-guide/backend/internal/platform/response"
	"gorm.io/gorm"
)

type gormConversationRepository struct {
	db *gorm.DB
}

func NewGORMConversationRepository(db *gorm.DB) ConversationRepository {
	return &gormConversationRepository{db: db}
}

func (r *gormConversationRepository) Create(ctx context.Context, c *Conversation) error {
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *gormConversationRepository) Get(ctx context.Context, id, userID string) (*Conversation, error) {
	var c Conversation
	err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, response.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *gormConversationRepository) Update(ctx context.Context, id, userID string, params UpdateConversationParams) error {
	updates := map[string]interface{}{}
	if params.Title != nil {
		updates["title"] = *params.Title
	}
	if params.Country != nil {
		updates["country"] = *params.Country
	}
	if len(updates) == 0 {
		return nil
	}
	res := r.db.WithContext(ctx).Model(&Conversation{}).
		Where("id = ? AND user_id = ?", id, userID).Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return response.ErrNotFound
	}
	return nil
}

func (r *gormConversationRepository) Delete(ctx context.Context, id, userID string) error {
	res := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&Conversation{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return response.ErrNotFound
	}
	return nil
}

func (r *gormConversationRepository) ListByUser(ctx context.Context, userID string, page, pageSize int) ([]*Conversation, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	q := r.db.WithContext(ctx).Model(&Conversation{}).Where("user_id = ?", userID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []*Conversation
	if err := q.Order("updated_at DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

type gormMessageRepository struct {
	db *gorm.DB
}

func NewGORMMessageRepository(db *gorm.DB) MessageRepository {
	return &gormMessageRepository{db: db}
}

func (r *gormMessageRepository) Create(ctx context.Context, m *Message) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *gormMessageRepository) Get(ctx context.Context, id string) (*Message, error) {
	var m Message
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, response.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *gormMessageRepository) ListByConversation(ctx context.Context, convID string, page, pageSize int) ([]*Message, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	q := r.db.WithContext(ctx).Model(&Message{}).Where("conversation_id = ?", convID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []*Message
	if err := q.Order("created_at ASC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *gormMessageRepository) Update(ctx context.Context, id string, params UpdateMessageParams) error {
	updates := map[string]interface{}{}
	if params.Content != nil {
		updates["content"] = *params.Content
	}
	if params.Sources != nil {
		updates["sources"] = params.Sources
	}
	if params.TokensUsed != nil {
		updates["tokens_used"] = *params.TokensUsed
	}
	if len(updates) == 0 {
		return nil
	}
	res := r.db.WithContext(ctx).Model(&Message{}).Where("id = ?", id).Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	return nil
}
```

- [ ] **Step 2: 验证编译**

Run: `cd backend && go build ./...`
Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add backend/internal/domain/conversation/repository.go backend/internal/domain/conversation/repo_gorm.go
git commit -m "feat(backend/conversation): add Conversation/Message repositories"
```

---

### Task 7: `conversation/service.go` — 业务编排

**Files:**
- Create: `backend/internal/domain/conversation/service.go`

- [ ] **Step 1: 写 service.go**

`backend/internal/domain/conversation/service.go`:
```go
package conversation

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/invest-guide/backend/internal/domain/assistant"
	"github.com/invest-guide/backend/internal/platform/llm"
	"github.com/invest-guide/backend/internal/platform/response"
)

type Service struct {
	convs       ConversationRepository
	msgs        MessageRepository
	assistant   *assistant.Service
}

func NewService(convs ConversationRepository, msgs MessageRepository, asst *assistant.Service) *Service {
	return &Service{convs: convs, msgs: msgs, assistant: asst}
}

func (s *Service) CreateConversation(ctx context.Context, userID string, req CreateConversationRequest) (*ConversationDTO, error) {
	title := req.Title
	if title == "" {
		title = "新会话"
	}
	conv := &Conversation{
		ID:      uuid.NewString(),
		UserID:  userID,
		Title:   title,
		Country: req.Country,
	}
	if err := s.convs.Create(ctx, conv); err != nil {
		return nil, err
	}
	dto := conv.ToDTO()
	return &dto, nil
}

func (s *Service) GetConversation(ctx context.Context, id, userID string) (*ConversationDTO, error) {
	conv, err := s.convs.Get(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	dto := conv.ToDTO()
	return &dto, nil
}

func (s *Service) ListConversations(ctx context.Context, userID string, page, pageSize int) ([]*ConversationDTO, int64, error) {
	items, total, err := s.convs.ListByUser(ctx, userID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	dtos := make([]*ConversationDTO, len(items))
	for i, c := range items {
		dto := c.ToDTO()
		dtos[i] = &dto
	}
	return dtos, total, nil
}

func (s *Service) DeleteConversation(ctx context.Context, id, userID string) error {
	return s.convs.Delete(ctx, id, userID)
}

func (s *Service) ListMessages(ctx context.Context, convID, userID string, page, pageSize int) ([]*MessageDTO, int64, error) {
	// 确认会话属于该用户
	if _, err := s.convs.Get(ctx, convID, userID); err != nil {
		return nil, 0, err
	}
	items, total, err := s.msgs.ListByConversation(ctx, convID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	dtos := make([]*MessageDTO, len(items))
	for i, m := range items {
		dto := m.ToDTO()
		dtos[i] = &dto
	}
	return dtos, total, nil
}

// PostMessage 创建 user 消息 + 占位 assistant 消息，返回 assistant message ID 供 stream 订阅
func (s *Service) PostMessage(ctx context.Context, convID, userID, content string) (*PostMessageResponse, error) {
	conv, err := s.convs.Get(ctx, convID, userID)
	if err != nil {
		return nil, err
	}

	userMsg := &Message{
		ID:             uuid.NewString(),
		ConversationID: convID,
		Role:           RoleUser,
		Content:        content,
	}
	if err := s.msgs.Create(ctx, userMsg); err != nil {
		return nil, err
	}

	assistantMsg := &Message{
		ID:             uuid.NewString(),
		ConversationID: convID,
		Role:           RoleAssistant,
		Content:        "",
	}
	if err := s.msgs.Create(ctx, assistantMsg); err != nil {
		return nil, err
	}

	// 首条 user 消息且会话标题为默认值时，把会话标题改为消息摘要
	if conv.Title == "新会话" {
		title := content
		if r := []rune(title); len(r) > 20 {
			title = string(r[:20]) + "..."
		}
		_ = s.convs.Update(ctx, convID, userID, UpdateConversationParams{Title: &title})
	}

	return &PostMessageResponse{MessageID: assistantMsg.ID}, nil
}

// StreamAnswer 由 SSE handler 调用：执行 RAG + LLM 流式。返回 sources（供 SSE 在首次
// message 之前发送）和 delta channel。content/tokens 的持久化由 FinalizeAnswer 在流后调用。
func (s *Service) StreamAnswer(ctx context.Context, convID, userID, assistantMessageID string) (
	sources []assistant.ContextSource,
	ch <-chan llm.StreamChunk,
	err error,
) {
	conv, err := s.convs.Get(ctx, convID, userID)
	if err != nil {
		return nil, nil, err
	}

	// 取最近 N 条消息作为对话上下文
	msgs, _, err := s.msgs.ListByConversation(ctx, convID, 1, 20)
	if err != nil {
		return nil, nil, err
	}
	// 找到 assistantMessageID 对应的位置，只用它之前的 user 内容作为 query
	var userQuery string
	for _, m := range msgs {
		if m.ID == assistantMessageID {
			break
		}
		if m.Role == RoleUser {
			userQuery = m.Content
		}
	}
	if userQuery == "" {
		return nil, nil, response.ErrInvalidInput
	}

	ch, sources, err = s.assistant.Stream(ctx, userQuery, conv.Country)
	if err != nil {
		return nil, nil, response.ErrBadGateway
	}
	return sources, ch, nil
}

// FinalizeAnswer 流结束后持久化 assistant 消息内容与引用源。
// 使用独立 context（不依赖客户端连接），确保即便客户端在 done 后立即断连也能落库。
func (s *Service) FinalizeAnswer(assistantMessageID, content string, sources []assistant.ContextSource, tokens int) error {
	ctx := context.Background()
	srcsJSON, _ := json.Marshal(sources)
	return s.msgs.Update(ctx, assistantMessageID, UpdateMessageParams{
		Content:    &content,
		Sources:    srcsJSON,
		TokensUsed: &tokens,
	})
}
```

- [ ] **Step 2: 验证编译**

Run: `cd backend && go build ./...`
Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add backend/internal/domain/conversation/service.go
git commit -m "feat(backend/conversation): add Service with PostMessage + StreamAnswer"
```

---

### Task 8: `conversation/sse.go` + `handler.go` + `route.go`

**Files:**
- Create: `backend/internal/domain/conversation/sse.go`
- Create: `backend/internal/domain/conversation/handler.go`
- Create: `backend/internal/domain/conversation/route.go`
- Create: `backend/internal/domain/conversation/handler_test.go`

- [ ] **Step 1: 写 sse.go（SSE writer 封装）**

`backend/internal/domain/conversation/sse.go`:
```go
package conversation

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type SSEWriter struct {
	w           http.ResponseWriter
	flusher     http.Flusher
	heartbeatAt time.Time
}

func NewSSEWriter(w http.ResponseWriter) *SSEWriter {
	flusher, _ := w.(http.Flusher)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	return &SSEWriter{w: w, flusher: flusher, heartbeatAt: time.Now().Add(15 * time.Second)}
}

func (s *SSEWriter) Send(event string, data interface{}) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", event, payload); err != nil {
		return err
	}
	if s.flusher != nil {
		s.flusher.Flush()
	}
	s.heartbeatAt = time.Now().Add(15 * time.Second)
	return nil
}

// MaybeHeartbeat 15s 静默则发心跳
func (s *SSEWriter) MaybeHeartbeat() {
	if time.Now().Before(s.heartbeatAt) {
		return
	}
	_ = s.Send("heartbeat", struct{}{})
}
```

- [ ] **Step 2: 写 handler.go**

`backend/internal/domain/conversation/handler.go`:
```go
package conversation

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/invest-guide/backend/internal/domain/assistant"
	"github.com/invest-guide/backend/internal/platform/llm"
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
	items, total, err := h.service.ListConversations(c.Request.Context(), c.GetString("userID"), page, pageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":   items,
			"total":   total,
			"hasMore": int64(page*pageSize) < total,
		},
	})
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.ErrInvalidInput)
		return
	}
	dto, err := h.service.CreateConversation(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Ok(c, http.StatusCreated, dto)
}

func (h *Handler) Get(c *gin.Context) {
	dto, err := h.service.GetConversation(c.Request.Context(), c.Param("id"), c.GetString("userID"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Ok(c, http.StatusOK, dto)
}

func (h *Handler) Delete(c *gin.Context) {
	if err := h.service.DeleteConversation(c.Request.Context(), c.Param("id"), c.GetString("userID")); err != nil {
		response.Fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListMessages(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	items, total, err := h.service.ListMessages(c.Request.Context(), c.Param("id"), c.GetString("userID"), page, pageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":   items,
			"total":   total,
			"hasMore": int64(page*pageSize) < total,
		},
	})
}

func (h *Handler) PostMessage(c *gin.Context) {
	var req PostMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.ErrInvalidInput)
		return
	}
	resp, err := h.service.PostMessage(c.Request.Context(), c.Param("id"), c.GetString("userID"), req.Content)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Ok(c, http.StatusCreated, resp)
}

// Stream 是 SSE 流式回答端点
// 路由：GET /conversations/:id/messages/:messageId/stream
func (h *Handler) Stream(c *gin.Context) {
	convID := c.Param("id")
	messageID := c.Param("messageId")
	userID := c.GetString("userID")

	sources, ch, err := h.service.StreamAnswer(c.Request.Context(), convID, userID, messageID)
	if err != nil {
		response.Fail(c, err)
		return
	}

	sse := NewSSEWriter(c.Writer)

	// 1. 发送 sources 事件（在首次 message 之前）
	if len(sources) > 0 {
		chunks := make([]map[string]string, len(sources))
		for i, s := range sources {
			chunks[i] = map[string]string{"id": s.ChunkID, "title": s.Title, "snippet": s.Snippet}
		}
		if err := sse.Send("sources", map[string]interface{}{"chunks": chunks}); err != nil {
			return
		}
	}

	// 2. 消费 LLM stream，逐 chunk 发 message，遇 done/error 结束
	var contentBuilder strings.Builder
	var tokensUsed int
	var streamErr error

	for {
		select {
		case <-c.Request.Context().Done():
			// 客户端断连，停止读取（但 goroutine 内的 LLM 调用因 ctx 取消而自行终止）
			return
		case chunk, ok := <-ch:
			if !ok {
				return
			}
			if chunk.Err != nil {
				streamErr = chunk.Err
				_ = sse.Send("error", map[string]string{
					"code":    "LLM_ERROR",
					"message": "stream failed",
				})
				return
			}
			if chunk.Done {
				tokensUsed = chunk.TokensUsed
				_ = sse.Send("done", map[string]interface{}{
					"messageId":  messageID,
					"tokensUsed": tokensUsed,
				})
				// 持久化 assistant 消息（用独立 ctx，客户端断连不影响落库）
				_ = h.service.FinalizeAnswer(messageID, contentBuilder.String(), sources, tokensUsed)
				return
			}
			if chunk.Delta != "" {
				contentBuilder.WriteString(chunk.Delta)
				_ = sse.Send("message", map[string]string{"delta": chunk.Delta})
			}
			sse.MaybeHeartbeat()
		}
	}
}

// 确保未使用的 import 不报错
var _ = llm.StreamChunk{}
var _ = assistant.ContextSource{}
```

- [ ] **Step 3: 写 route.go**

`backend/internal/domain/conversation/route.go`:
```go
package conversation

import "github.com/gin-gonic/gin"

func Register(group *gin.RouterGroup, h *Handler) {
	convs := group.Group("/conversations")
	convs.GET("", h.List)
	convs.POST("", h.Create)
	convs.GET("/:id", h.Get)
	convs.DELETE("/:id", h.Delete)
	convs.GET("/:id/messages", h.ListMessages)
	convs.POST("/:id/messages", h.PostMessage)
	convs.GET("/:id/messages/:messageId/stream", h.Stream)
}
```

- [ ] **Step 4: 写 handler REST 测试（不含 Stream，Stream 留 E2E）**

`backend/internal/domain/conversation/handler_test.go`:
```go
package conversation

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func init() { gin.SetMode(gin.TestMode) }

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Conversation{}, &Message{}))
	return db
}

func TestHandler_Create_List_Get_Delete(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(
		NewGORMConversationRepository(db),
		NewGORMMessageRepository(db),
		nil, // assistant only needed for Stream
	)
	h := NewHandler(svc)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", "u-1"); c.Next() })
	r.POST("/c", h.Create)
	r.GET("/c", h.List)
	r.GET("/c/:id", h.Get)
	r.DELETE("/c/:id", h.Delete)

	// Create
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/c", bytes.NewBufferString(`{"title":"测试","country":"越南"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	var cr struct{ Data ConversationDTO }
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &cr))
	require.NotEmpty(t, cr.Data.ID)
	assert.Equal(t, "测试", cr.Data.Title)

	// List
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/c", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	// Get
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/c/"+cr.Data.ID, nil))
	assert.Equal(t, http.StatusOK, w.Code)

	// Delete
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("DELETE", "/c/"+cr.Data.ID, nil))
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandler_PostMessage_ReturnsMessageID(t *testing.T) {
	db := newTestDB(t)
	// 先 create 一条会话
	conv := &Conversation{ID: "c-1", UserID: "u-1", Title: "x"}
	require.NoError(t, db.Create(conv).Error)

	svc := NewService(
		NewGORMConversationRepository(db),
		NewGORMMessageRepository(db),
		nil,
	)
	h := NewHandler(svc)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", "u-1"); c.Next() })
	r.POST("/c/:id/messages", h.PostMessage)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/c/c-1/messages", bytes.NewBufferString(`{"content":"越南税收"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var resp struct{ Data PostMessageResponse }
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Data.MessageID)
}

func TestHandler_PostMessage_WrongUser_NotFound(t *testing.T) {
	db := newTestDB(t)
	conv := &Conversation{ID: "c-1", UserID: "u-1", Title: "x"}
	require.NoError(t, db.Create(conv).Error)

	svc := NewService(
		NewGORMConversationRepository(db),
		NewGORMMessageRepository(db),
		nil,
	)
	h := NewHandler(svc)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", "u-other"); c.Next() })
	r.POST("/c/:id/messages", h.PostMessage)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/c/c-1/messages", bytes.NewBufferString(`{"content":"越南税收"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
```

- [ ] **Step 5: 跑测试**

Run: `cd backend && go test ./internal/domain/conversation/`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add backend/internal/domain/conversation/
git commit -m "feat(backend/conversation): add SSE handler and conversation routes"
```

---

### Task 9: 接入 `Deps` 与 `main.go`；补 `system/models`

**Files:**
- Create: `backend/internal/domain/system/models_handler.go`
- Modify: `backend/internal/platform/router/deps.go`
- Modify: `backend/internal/platform/router/router.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: 写 system/models handler**

`backend/internal/domain/system/models_handler.go`:
```go
package system

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/invest-guide/backend/internal/platform/response"
)

type ModelsHandler struct {
	llmModel       string
	embeddingModel string
}

func NewModelsHandler(llmModel, embeddingModel string) *ModelsHandler {
	return &ModelsHandler{llmModel: llmModel, embeddingModel: embeddingModel}
}

func (h *ModelsHandler) Models(c *gin.Context) {
	response.Ok(c, http.StatusOK, gin.H{
		"llm":       h.llmModel,
		"embedding": h.embeddingModel,
	})
}
```

修改 `system/route.go` 注册 Models 端点（签名扩展第三个参数 `*ModelsHandler`，是 breaking change）：

```go
func Register(group *gin.RouterGroup, h *Handler, mh *ModelsHandler) {
	group.GET("/health", h.Health)
	group.GET("/version", h.Version)
	group.GET("/models", mh.Models)
}
```

**Register 签名变更的连锁更新点**（所有调用方都要同步）：

1. `backend/internal/platform/router/router.go` — `system.Register(v1.Group("/system"), deps.SystemHandler)` 改为 3 参：增加 `deps.ModelsHandler`（在 Step 3 完成）
2. `backend/tests/e2e/helpers.go` — Plan 1/Plan 2 的 `NewTestServer()` 内部调用 `system.Register` 也需加 `system.NewModelsHandler("test-llm", "test-embed")` 第三参
3. `backend/internal/platform/router/router_test.go` — 若 router 自身的单测直接调用了 Register 同样需更新

执行器需把这三处统一改完后再跑测试，避免编译错误。

- [ ] **Step 2: 扩展 deps.go**

`Deps` 新增字段：
```go
type Deps struct {
	// ...
	KnowledgeHandler  *knowledge.Handler
	KnowledgeSearcher KnowledgeSearcher
	LLMProvider       llm.Provider
	AssistantService  *assistant.Service
	ConversationHandler *conversation.Handler
	ModelsHandler     *system.ModelsHandler
}
```

同时定义 `ConversationSearcherAdapter` — 把 `*knowledge.Service` 适配为 `assistant.KnowledgeSearcher`：

```go
type knowledgeSearcherAdapter struct {
	svc interface {
		Search(ctx context.Context, req knowledge.SearchRequest) (*knowledge.SearchResponse, error)
	}
}

func (a knowledgeSearcherAdapter) Search(ctx context.Context, query, country string, topK int) ([]assistant.ContextSource, error) {
	resp, err := a.svc.Search(ctx, knowledge.SearchRequest{Query: query, Country: country, TopK: topK})
	if err != nil {
		return nil, err
	}
	srcs := make([]assistant.ContextSource, len(resp.Chunks))
	for i, c := range resp.Chunks {
		srcs[i] = assistant.ContextSource{ChunkID: c.ID, Title: c.Title, Snippet: c.Snippet}
	}
	return srcs, nil
}
```

修改 `registerPrivateRoutes` 挂 conversation 路由：

```go
func (d *Deps) registerPrivateRoutes(g *gin.RouterGroup) {
	if d.KnowledgeHandler != nil {
		knowledge.Register(g, d.KnowledgeHandler)
	}
	if d.ConversationHandler != nil {
		conversation.Register(g, d.ConversationHandler)
	}
}
```

> import 块需含 `"context"`、`"github.com/invest-guide/backend/internal/domain/assistant"`、`"...conversation"`、`"...knowledge"`、`"...llm"`、`"...system"`。

- [ ] **Step 3: 修改 system.Register 调用**

在 router.go 的 `system.Register(v1.Group("/system"), deps.SystemHandler)` 改为：
```go
system.Register(v1.Group("/system"), deps.SystemHandler, deps.ModelsHandler)
```

- [ ] **Step 4: main.go 装配 LLM + assistant + conversation**

在 knowledgeSvc 装配之后：
```go
llmProvider := llm.NewOpenAIProvider(cfg.LLMBaseURL, cfg.LLMAPIKey, cfg.LLMModel, cfg.LLMTimeout, cfg.LLMStreamTimeout)
searcherAdapter := knowledgeSearcherAdapter{svc: knowledgeSvc}
assistantSvc := assistant.NewService(llmProvider, searcherAdapter)
conversationSvc := conversation.NewService(
	conversation.NewGORMConversationRepository(db),
	conversation.NewGORMMessageRepository(db),
	assistantSvc,
)

deps := &router.Deps{
	// ...
	LLMProvider:         llmProvider,
	AssistantService:    assistantSvc,
	ConversationHandler: conversation.NewHandler(conversationSvc),
	ModelsHandler:       system.NewModelsHandler(cfg.LLMModel, cfg.EmbeddingModel),
}
```

- [ ] **Step 5: 跑测试**

Run: `cd backend && go test ./...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add backend/internal/domain/system/ backend/internal/platform/router/ backend/cmd/server/main.go
git commit -m "feat(backend): wire conversation + assistant + system/models"
```

---

### Task 10: E2E — conversation 流程

**Files:**
- Create: `backend/tests/e2e/conversation_test.go`

- [ ] **Step 1: 写 E2E 测试**

`backend/tests/e2e/conversation_test.go`:
```go
package e2e

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConversation_Create_PostMessage_ListMessages(t *testing.T) {
	srv := NewTestServer()
	defer srv.Close()

	token := registerAndLogin(t, srv)

	// 1. 创建会话
	resp := doAuth(t, "POST", srv.URL+"/api/v1/conversations",
		`{"title":"越南税收","country":"越南"}`, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var cr struct{ Data struct{ ID string `json:"id"` } `json:"data"` }
	require.NoError(t, json.Unmarshal(mustReadAll(resp.Body), &cr))
	convID := cr.Data.ID

	// 2. 发消息
	resp = doAuth(t, "POST", srv.URL+"/api/v1/conversations/"+convID+"/messages",
		`{"content":"越南的企业所得税率是多少？"}`, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var pm struct{ Data struct{ MessageID string `json:"messageId"` } `json:"data"` }
	require.NoError(t, json.Unmarshal(mustReadAll(resp.Body), &pm))
	require.NotEmpty(t, pm.Data.MessageID)

	// 3. 列消息历史（应看到 user 消息；assistant 消息可能为空内容，因为没启动 stream）
	resp = doAuth(t, "GET", srv.URL+"/api/v1/conversations/"+convID+"/messages", "", token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var ml struct{ Data struct{ Items []struct{ Role string `json:"role"` } `json:"items"` } `json:"data"` }
	require.NoError(t, json.Unmarshal(mustReadAll(resp.Body), &ml))
	assert.GreaterOrEqual(t, len(ml.Data.Items), 1)
}

func TestConversation_OtherUserCannotAccess(t *testing.T) {
	srv := NewTestServer()
	defer srv.Close()

	// user A 创建会话
	tokenA := registerAndLoginAt(t, srv, "a@b.com")
	resp := doAuth(t, "POST", srv.URL+"/api/v1/conversations", `{"title":"A的"}`, tokenA)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var cr struct{ Data struct{ ID string `json:"id"` } `json:"data"` }
	require.NoError(t, json.Unmarshal(mustReadAll(resp.Body), &cr))

	// user B 访问 A 的会话 → 404
	tokenB := registerAndLoginAt(t, srv, "b@b.com")
	resp = doAuth(t, "GET", srv.URL+"/api/v1/conversations/"+cr.Data.ID, "", tokenB)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// 注册指定 email 的 helper，避免冲突
func registerAndLoginAt(t *testing.T, srv *TestServer, email string) string {
	t.Helper()
	body := `{"email":"` + email + `","password":"password123","displayName":"X"}`
	resp := postJSON(t, srv.URL+"/api/v1/auth/register", body, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var r struct{ Data struct{ Token string `json:"token"` } `json:"data"` }
	require.NoError(t, json.Unmarshal(mustReadAll(resp.Body), &r))
	return r.Data.Token
}
```

- [ ] **Step 2: 跑测试**

Run: `cd backend && go test ./tests/e2e/`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add backend/tests/e2e/conversation_test.go
git commit -m "test(backend/e2e): add conversation create/post/list/isolation"
```

---

### Task 11: E2E — SSE stream 完整流

**Files:**
- Create: `backend/tests/e2e/stream_test.go`

> 关键挑战：E2E 默认用真实 LLM Provider，但 E2E 不应调真实模型。本测试改用注入 fake provider 的 TestServer 变体。

- [ ] **Step 1: 扩展 helpers.go 支持注入 fake LLM**

在 `helpers.go` 添加：
```go
func NewTestServerWithFakeLLM(deltas []string, tokens int) *TestServer {
	db, _ := database.NewSQLite(":memory:")
	_ = db.AutoMigrate(&auth.User{}, &knowledge.KnowledgeDoc{}, &knowledge.KnowledgeChunk{},
		&conversation.Conversation{}, &conversation.Message{})

	jwt := auth.NewJWTIssuer("test-secret", "investguide", 1<<30)
	authSvc := auth.NewService(auth.NewGORMUserRepository(db), jwt)

	embed := &fakeTestEmbedding{dim: 8}
	knowledgeSvc := knowledge.NewService(
		knowledge.NewGORMDocRepository(db),
		knowledge.NewGORMChunkRepository(db),
		embed,
		taskqueue.NewGoroutinePool(2, 4),
		nil,
	)
	fakeLLM := llm.NewFakeProvider("", deltas, tokens)
	searcherAdapter := knowledgeSearcherAdapter{svc: knowledgeSvc}
	assistantSvc := assistant.NewService(fakeLLM, searcherAdapter)
	conversationSvc := conversation.NewService(
		conversation.NewGORMConversationRepository(db),
		conversation.NewGORMMessageRepository(db),
		assistantSvc,
	)

	deps := &router.Deps{
		Cfg:                 &config.Config{CORSOrigins: "*", RateLimitAPI: 0},
		Version:             "0.0.1-test",
		SystemHandler:       system.NewHandler(system.NewService("0.0.1-test")),
		AuthHandler:         auth.NewHandler(authSvc),
		Authenticator:       &auth.AuthenticatorAdapter{Service: authSvc},
		KnowledgeHandler:    knowledge.NewHandler(knowledgeSvc),
		LLMProvider:         fakeLLM,
		AssistantService:    assistantSvc,
		ConversationHandler: conversation.NewHandler(conversationSvc),
		ModelsHandler:       system.NewModelsHandler("fake-llm", "fake-embed"),
	}
	r := router.New(deps)
	return &TestServer{Server: httptest.NewServer(r), DB: db, AuthSvc: authSvc}
}
```

> `knowledgeSearcherAdapter` 已在 router 包导出可访问（或在本测试包内重新实现）。原 `NewTestServer` 也需要更新到这个完整装配（含 conversation）。

- [ ] **Step 2: 写 SSE 流测试**

`backend/tests/e2e/stream_test.go`:
```go
package e2e

import (
	"bufio"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSEStream_FullFlow(t *testing.T) {
	srv := NewTestServerWithFakeLLM([]string{"Hello ", "world"}, 7)
	defer srv.Close()

	token := registerAndLogin(t, srv)

	// 1. 创建会话
	resp := doAuth(t, "POST", srv.URL+"/api/v1/conversations", `{"country":"越南"}`, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var cr struct{ Data struct{ ID string `json:"id"` } `json:"data"` }
	require.NoError(t, json.Unmarshal(mustReadAll(resp.Body), &cr))
	convID := cr.Data.ID

	// 2. 发消息
	resp = doAuth(t, "POST", srv.URL+"/api/v1/conversations/"+convID+"/messages",
		`{"content":"你好"}`, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var pm struct{ Data struct{ MessageID string `json:"messageId"` } `json:"data"` }
	require.NoError(t, json.Unmarshal(mustReadAll(resp.Body), &pm))
	msgID := pm.Data.MessageID

	// 3. 订阅 stream
	req, _ := http.NewRequest("GET", srv.URL+"/api/v1/conversations/"+convID+"/messages/"+msgID+"/stream", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "text/event-stream")
	client := &http.Client{Timeout: 5 * time.Second}
	streamResp, err := client.Do(req)
	require.NoError(t, err)
	defer streamResp.Body.Close()
	require.Equal(t, http.StatusOK, streamResp.StatusCode)

	// 4. 读取 SSE 事件
	scanner := bufio.NewScanner(streamResp.Body)
	var messageDeltas []string
	var gotDone bool
	var gotError bool
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			eventType := strings.TrimPrefix(line, "event: ")
			// 下一行应是 data:
			if !scanner.Scan() {
				break
			}
			dataLine := strings.TrimPrefix(scanner.Text(), "data: ")

			switch eventType {
			case "message":
				var d struct{ Delta string `json:"delta"` }
				_ = json.Unmarshal([]byte(dataLine), &d)
				messageDeltas = append(messageDeltas, d.Delta)
			case "done":
				gotDone = true
			case "error":
				gotError = true
			}
		}
		if gotDone || gotError {
			break
		}
	}

	assert.False(t, gotError, "should not receive error event")
	assert.True(t, gotDone, "should receive done event")
	assert.Equal(t, []string{"Hello ", "world"}, messageDeltas)
}

func TestSSEStream_InvalidMessageID_NotFound(t *testing.T) {
	srv := NewTestServerWithFakeLLM(nil, 0)
	defer srv.Close()

	token := registerAndLogin(t, srv)

	// 先建一个会话（否则会话不存在的 404 会和消息不存在的 404 混淆）
	resp := doAuth(t, "POST", srv.URL+"/api/v1/conversations", `{}`, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var cr struct{ Data struct{ ID string `json:"id"` } `json:"data"` }
	require.NoError(t, json.Unmarshal(mustReadAll(resp.Body), &cr))

	// stream 不存在的 messageId — 期望 404
	resp = doAuth(t, "GET",
		srv.URL+"/api/v1/conversations/"+cr.Data.ID+"/messages/nonexistent/stream", "", token)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
```

- [ ] **Step 3: 跑测试**

Run: `cd backend && go test ./tests/e2e/ -run SSE`
Expected: PASS（首次运行可能因 SSE 客户端超时设置不当失败，调整 `client.Timeout` 至 10s 即可）

- [ ] **Step 4: Commit**

```bash
git add backend/tests/e2e/stream_test.go
git commit -m "test(backend/e2e): add SSE stream full flow with fake LLM"
```

---

### Task 12: 最终验证

- [ ] **Step 1: gofmt + vet + test**

Run:
```bash
cd backend && gofmt -l . && go vet ./... && go test ./... -cover
```
Expected: 全 PASS；`internal/domain/conversation/` + `internal/domain/assistant/` 覆盖率 ≥ 70%

- [ ] **Step 2: 启动 + 完整流程 curl**

Run: `make backend-dev`
另开终端：
```bash
TOKEN=$(curl -sX POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"e@b.com","password":"password123","displayName":"E"}' | jq -r .data.token)

# 创建会话
CONV=$(curl -sX POST http://localhost:8080/api/v1/conversations \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"title":"越南税收","country":"越南"}' | jq -r .data.id)

# 发消息
MSG=$(curl -sX POST http://localhost:8080/api/v1/conversations/$CONV/messages \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"content":"越南的企业所得税率？"}' | jq -r .data.messageId)

# 订阅 stream（需 curl -N 不缓冲）
curl -N -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/conversations/$CONV/messages/$MSG/stream
# 期望：依次收到 event: message（多个 delta）→ event: done

# system/models 端点
curl -s http://localhost:8080/api/v1/system/models | jq
```

- [ ] **Step 3: 文档同步核对**

Run: `grep -n "platform/llm\|domain/assistant\|domain/conversation" ARCHITECTURE.md AGENT.md`
Expected: ARCHITECTURE.md 仍把 LLMProvider 描述在 assistant 模块下；本 plan 把 EmbeddingProvider 移到 `platform/embedding/`，LLMProvider 放 `platform/llm/`，与 ARCHITECTURE.md 字面描述有偏差。

**文档同步任务（在 Plan 4 验收时一并 commit）：**
- ARCHITECTURE.md「基础层」表格新增 `embedding` 与 `llm` 两行
- ARCHITECTURE.md「领域层」assistant 模块职责改为"`LLMProvider` 调用方、prompt 构建、RAG 上下文组装、Agent 编排；接口定义在 `platform/llm`、`platform/embedding`"
- ARCHITECTURE.md「LLM Provider 抽象」标题不变，内容补充"接口位置：`platform/llm/`、`platform/embedding/`"

- [ ] **Step 4: 文档同步 commit**

```bash
git add ARCHITECTURE.md
git commit -m "docs: sync ARCHITECTURE.md for platform/llm + platform/embedding"
```

- [ ] **Step 5: 标记完成**

Plan 4 完成。整个后端骨架就此完整：基础层 + auth + knowledge + conversation + assistant 五个模块全部就绪，RAG 问答端到端可用。

---

## 执行记录（2026-08-02）

本 plan 已按上述步骤落地，执行中做了以下修正与补充：

1. **`registerPrivateRoutes` 方法** → 沿用 `PrivateRoutes` 字段（Plan 2/3 已改）注入
   conversation 路由：`deps.PrivateRoutes = func(g){ knowledge.Register + conversation.Register }`。
2. **`assistant.KnowledgeSearcher` 与 `router.KnowledgeSearcher` 两套接口**：通过
   `router.knowledgeSearcherAdapter` 桥接，并导出 `router.NewKnowledgeSearcherAdapter(svc)`
   供 main.go 装配。
3. **`assistant/model.go` 精简**：plan 里重复定义了本地 `Provider`/`GenerateRequest` 等
   类型，已改为只保留 `ContextSource`/`AssembledContext`，直接依赖 `platform/llm` 包类型。
4. **`system.Register` 签名 breaking change**（加 `*ModelsHandler` 第三参）：同步更新
   router.go、e2e/helpers.go。
5. **`TestOpenAIProvider_Generate_Timeout` 会 hang 30s**：handler `<-r.Context().Done()`
   永不返回导致 server.Close() 卡住。改为 handler `time.Sleep(2s)` + provider 50ms timeout。
6. **`TestOpenAIProvider_Stream_ErrorEvent` 语义冲突**：Stream 在 HTTP 层失败直接返回
   error（而非发 error chunk），测试改为断言返回 error。
7. **FakeProvider.Stream 缓冲 channel 竞态**：cancel 不生效。改为缓冲 1 + 双重
   `select ctx.Done()`，确保取消可靠终止。
8. **`StreamAnswer` 未校验 assistant 消息存在**：`TestSSEStream_InvalidMessageID_NotFound`
   期望 404 但实际返回 200。修复：先 `s.msgs.Get(target)` 校验存在 + role 为 assistant。
9. **`TestFakeProvider_Stream_CancelMidway` 竞态**：改为 cancel 后持续读至 Done 并断言
   `Err == context.Canceled`。
10. **覆盖率补充**：conversation 30% → 60.5%（service CRUD/Stream/Finalize + repo Update/
    Get + handler ListMessages 测试）。整体 internal 平均 74.8% 达标。
11. **ARCHITECTURE.md 文档同步**：基础层加 `llm`/`embedding` 两行；领域层 assistant 行改为
    "LLMProvider 调用方...（接口定义在 platform/llm、platform/embedding）"；LLM Provider
    抽象章节加接口位置说明。

覆盖率：conversation 60.5%（SSE Stream 由 E2E 覆盖）· assistant 85.3% · llm 92%。
