package router

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/invest-guide/backend/internal/domain/assistant"
	"github.com/invest-guide/backend/internal/domain/auth"
	"github.com/invest-guide/backend/internal/domain/conversation"
	"github.com/invest-guide/backend/internal/domain/knowledge"
	"github.com/invest-guide/backend/internal/domain/system"
	"github.com/invest-guide/backend/internal/platform/cache"
	"github.com/invest-guide/backend/internal/platform/config"
	"github.com/invest-guide/backend/internal/platform/database"
	"github.com/invest-guide/backend/internal/platform/llm"
	"github.com/invest-guide/backend/internal/platform/middleware"
	"github.com/invest-guide/backend/internal/platform/taskqueue"
	"gorm.io/gorm"
)

// Deps 是装配中心构造的依赖容器，所有 handler 从此处获取依赖
type Deps struct {
	Cfg       *config.Config
	DB        *gorm.DB
	Cache     cache.Cache
	TaskQueue taskqueue.Queue
	Version   string

	SystemHandler  *system.Handler
	AuthHandler    *auth.Handler
	Authenticator  middleware.Authenticator
	LoginRateLimit *middleware.LoginRateLimit

	// Knowledge 领域（Plan 3 注入）
	KnowledgeHandler  *knowledge.Handler
	KnowledgeSearcher KnowledgeSearcher

	// Conversation + Assistant 领域（Plan 4 注入）
	LLMProvider         llm.Provider
	AssistantService    *assistant.Service
	ConversationHandler *conversation.Handler
	ModelsHandler       *system.ModelsHandler

	// PrivateRoutes 可选：注册鉴权路由组下的领域路由
	PrivateRoutes func(g *gin.RouterGroup)
}

// KnowledgeSearcher 抽象 knowledge.Service.Search，避免其他 domain 反向 import knowledge
type KnowledgeSearcher interface {
	Search(ctx context.Context, req knowledge.SearchRequest) (*knowledge.SearchResponse, error)
}

// knowledgeSearcherAdapter 把 *knowledge.Service 适配为 assistant.KnowledgeSearcher
type knowledgeSearcherAdapter struct {
	svc KnowledgeSearcher
}

// NewKnowledgeSearcherAdapter 返回 assistant.KnowledgeSearcher 适配器（供 main.go 装配）
func NewKnowledgeSearcherAdapter(svc KnowledgeSearcher) assistant.KnowledgeSearcher {
	return knowledgeSearcherAdapter{svc: svc}
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

// NewTestDeps 仅用于路由测试，构造内存 SQLite 的最小依赖集（含 auth + knowledge + conversation）
func NewTestDeps(t *testing.T) *Deps {
	t.Helper()
	db, err := database.NewSQLite(":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&auth.User{}, &knowledge.KnowledgeDoc{}, &knowledge.KnowledgeChunk{},
		&conversation.Conversation{}, &conversation.Message{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	jwtIssuer := auth.NewJWTIssuer("test-secret", "invest-guide", 1<<30)
	authSvc := auth.NewService(auth.NewGORMUserRepository(db), jwtIssuer)

	knowledgeSvc := knowledge.NewService(
		knowledge.NewGORMDocRepository(db),
		knowledge.NewGORMChunkRepository(db),
		&fakeTestEmbedding{dim: 8},
		taskqueue.NewGoroutinePool(2, 4),
		nil,
	)

	// conversation 装配：用 fake LLM + searcher adapter
	fakeLLM := llm.NewFakeProvider("", []string{"Hi"}, 3)
	asstSvc := assistant.NewService(fakeLLM, knowledgeSearcherAdapter{svc: knowledgeSvc})
	conversationSvc := conversation.NewService(
		conversation.NewGORMConversationRepository(db),
		conversation.NewGORMMessageRepository(db),
		asstSvc,
	)

	return &Deps{
		Cfg:                 &config.Config{CORSOrigins: "*", RateLimitAPI: 0, RateLimitSensitive: 0},
		DB:                  db,
		Version:             "0.0.1-test",
		SystemHandler:       system.NewHandler(system.NewService("0.0.1-test")),
		ModelsHandler:       system.NewModelsHandler("fake-llm", "fake-embed"),
		AuthHandler:         auth.NewHandler(authSvc),
		Authenticator:       &auth.AuthenticatorAdapter{Service: authSvc},
		LoginRateLimit:      middleware.NewLoginRateLimit(5),
		KnowledgeHandler:    knowledge.NewHandler(knowledgeSvc),
		KnowledgeSearcher:   knowledgeSvc,
		LLMProvider:         fakeLLM,
		AssistantService:    asstSvc,
		ConversationHandler: conversation.NewHandler(conversationSvc),
	}
}

// fakeTestEmbedding 测试用固定向量
type fakeTestEmbedding struct{ dim int }

func (f *fakeTestEmbedding) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i := range texts {
		out[i] = make([]float32, f.dim)
	}
	return out, nil
}

func (f *fakeTestEmbedding) Dim() int { return f.dim }

// RegisterPrivateRoutes 调用注入的 PrivateRoutes（若设置）
func (d *Deps) RegisterPrivateRoutes(g *gin.RouterGroup) {
	if d.PrivateRoutes != nil {
		d.PrivateRoutes(g)
	}
}
