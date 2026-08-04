package e2e

import (
	"context"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	"github.com/invest-guide/backend/internal/domain/assistant"
	"github.com/invest-guide/backend/internal/domain/auth"
	"github.com/invest-guide/backend/internal/domain/conversation"
	"github.com/invest-guide/backend/internal/domain/knowledge"
	"github.com/invest-guide/backend/internal/domain/system"
	"github.com/invest-guide/backend/internal/platform/config"
	"github.com/invest-guide/backend/internal/platform/database"
	"github.com/invest-guide/backend/internal/platform/llm"
	"github.com/invest-guide/backend/internal/platform/router"
	"github.com/invest-guide/backend/internal/platform/taskqueue"
	"gorm.io/gorm"
)

type TestServer struct {
	*httptest.Server
	DB      *gorm.DB
	AuthSvc *auth.Service
}

// NewTestServer 构造完整装配的内存 server（auth + knowledge + conversation），用 fake LLM
func NewTestServer() *TestServer {
	return newTestServer(nil, 0)
}

// NewTestServerWithFakeLLM 用指定 delta 序列的 fake LLM，供 SSE 流测试
func NewTestServerWithFakeLLM(deltas []string, tokens int) *TestServer {
	return newTestServer(deltas, tokens)
}

func newTestServer(deltas []string, tokens int) *TestServer {
	db, _ := database.NewSQLite(":memory:")
	_ = db.AutoMigrate(&auth.User{}, &knowledge.KnowledgeDoc{}, &knowledge.KnowledgeChunk{},
		&conversation.Conversation{}, &conversation.Message{})

	jwt := auth.NewJWTIssuer("test-secret", "invest-guide", 1<<30)
	userRepo := auth.NewGORMUserRepository(db)
	authSvc := auth.NewService(userRepo, jwt)

	knowledgeSvc := knowledge.NewService(
		knowledge.NewGORMDocRepository(db),
		knowledge.NewGORMChunkRepository(db),
		&fakeE2EEmbedding{dim: 8},
		newE2EQueue(),
		nil,
	)
	knowledgeHandler := knowledge.NewHandler(knowledgeSvc)

	fakeLLM := llm.NewFakeProvider("", deltas, tokens)
	asstSvc := assistant.NewService(fakeLLM, router.NewKnowledgeSearcherAdapter(knowledgeSvc))
	conversationSvc := conversation.NewService(
		conversation.NewGORMConversationRepository(db),
		conversation.NewGORMMessageRepository(db),
		asstSvc,
	)
	conversationHandler := conversation.NewHandler(conversationSvc)

	deps := &router.Deps{
		Cfg:                 &config.Config{CORSOrigins: "*", RateLimitAPI: 0, RateLimitSensitive: 0},
		Version:             "0.0.1-test",
		SystemHandler:       system.NewHandler(system.NewService("0.0.1-test")),
		ModelsHandler:       system.NewModelsHandler("fake-llm", "fake-embed"),
		AuthHandler:         auth.NewHandler(authSvc),
		Authenticator:       &auth.AuthenticatorAdapter{Service: authSvc},
		LoginRateLimit:      nil, // E2E 测试禁用登录限流
		KnowledgeHandler:    knowledgeHandler,
		KnowledgeSearcher:   knowledgeSvc,
		LLMProvider:         fakeLLM,
		AssistantService:    asstSvc,
		ConversationHandler: conversationHandler,
		PrivateRoutes: func(g *gin.RouterGroup) {
			knowledge.Register(g, knowledgeHandler)
			conversation.Register(g, conversationHandler)
		},
	}
	r := router.New(deps)
	return &TestServer{
		Server:  httptest.NewServer(r),
		DB:      db,
		AuthSvc: authSvc,
	}
}

// fakeE2EEmbedding 返回固定维度向量，保证入库流水线可跑通
type fakeE2EEmbedding struct{ dim int }

func (f *fakeE2EEmbedding) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i := range texts {
		out[i] = make([]float32, f.dim)
	}
	return out, nil
}

func (f *fakeE2EEmbedding) Dim() int { return f.dim }

// e2eQueue 同步执行入库任务，让 E2E 无需等待异步完成
type e2eQueue struct{}

func newE2EQueue() *e2eQueue { return &e2eQueue{} }

func (q *e2eQueue) Enqueue(task taskqueue.Task) error {
	_ = task(context.Background())
	return nil
}

func (q *e2eQueue) Close(ctx context.Context) error { return nil }
