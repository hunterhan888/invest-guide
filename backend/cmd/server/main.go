package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/invest-guide/backend/internal/domain/assistant"
	"github.com/invest-guide/backend/internal/domain/auth"
	"github.com/invest-guide/backend/internal/domain/conversation"
	"github.com/invest-guide/backend/internal/domain/knowledge"
	"github.com/invest-guide/backend/internal/domain/system"
	"github.com/invest-guide/backend/internal/platform/cache"
	"github.com/invest-guide/backend/internal/platform/config"
	"github.com/invest-guide/backend/internal/platform/database"
	"github.com/invest-guide/backend/internal/platform/embedding"
	"github.com/invest-guide/backend/internal/platform/llm"
	"github.com/invest-guide/backend/internal/platform/logger"
	"github.com/invest-guide/backend/internal/platform/middleware"
	"github.com/invest-guide/backend/internal/platform/router"
	"github.com/invest-guide/backend/internal/platform/taskqueue"
)

const version = "0.0.1-dev"

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config failed", "err", err)
		os.Exit(1)
	}

	logOut := io.Writer(os.Stdout)
	if cfg.LogFile != "" {
		f, err := logger.OpenFile(cfg.LogFile)
		if err != nil {
			slog.Error("open log file failed", "err", err, "file", cfg.LogFile)
			os.Exit(1)
		}
		defer f.Close()
		logOut = f
	}
	logger.Init(logOut, cfg.LogLevel)
	slog.Info("starting invest-guide backend", "version", version, "port", cfg.Port)

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		slog.Error("connect db failed", "err", err)
		os.Exit(1)
	}
	// 开发（SQLite）环境用 AutoMigrate 建表；生产（PostgreSQL）走 golang-migrate
	if err := database.AutoMigrate(db,
		&auth.User{}, &knowledge.KnowledgeDoc{}, &knowledge.KnowledgeChunk{},
		&conversation.Conversation{}, &conversation.Message{},
	); err != nil {
		slog.Error("auto migrate failed", "err", err)
		os.Exit(1)
	}

	cacheInst := cache.NewLRU(1000, time.Hour)
	taskQ := taskqueue.NewGoroutinePool(4, 16)

	jwtIssuer := auth.NewJWTIssuer(cfg.JWTSecret, "invest-guide", cfg.JWTExpiry)
	userRepo := auth.NewGORMUserRepository(db)
	authSvc := auth.NewService(userRepo, jwtIssuer)

	embedDim, _ := strconv.Atoi(cfg.EmbeddingDim)
	if embedDim <= 0 {
		embedDim = 1024
	}
	embedProvider := embedding.NewOpenAIProvider(cfg.EmbeddingBaseURL, cfg.EmbeddingAPIKey, cfg.EmbeddingModel, embedDim)
	knowledgeSvc := knowledge.NewService(
		knowledge.NewGORMDocRepository(db),
		knowledge.NewGORMChunkRepository(db),
		embedProvider,
		taskQ,
		knowledge.NewEmbeddingCache(cacheInst),
	)
	knowledgeHandler := knowledge.NewHandler(knowledgeSvc)

	llmProvider := llm.NewOpenAIProvider(cfg.LLMBaseURL, cfg.LLMAPIKey, cfg.LLMModel, cfg.LLMTimeout, cfg.LLMStreamTimeout)

	searcherAdapter := router.NewKnowledgeSearcherAdapter(knowledgeSvc)
	assistantSvc := assistant.NewService(llmProvider, searcherAdapter)
	conversationSvc := conversation.NewService(
		conversation.NewGORMConversationRepository(db),
		conversation.NewGORMMessageRepository(db),
		assistantSvc,
	)
	conversationHandler := conversation.NewHandler(conversationSvc)

	deps := &router.Deps{
		Cfg:                 cfg,
		DB:                  db,
		Cache:               cacheInst,
		TaskQueue:           taskQ,
		Version:             version,
		SystemHandler:       system.NewHandler(system.NewService(version)),
		ModelsHandler:       system.NewModelsHandler(cfg.LLMModel, cfg.EmbeddingModel),
		AuthHandler:         auth.NewHandler(authSvc),
		Authenticator:       &auth.AuthenticatorAdapter{Service: authSvc},
		LoginRateLimit:      middleware.NewLoginRateLimit(5),
		KnowledgeHandler:    knowledgeHandler,
		KnowledgeSearcher:   knowledgeSvc,
		LLMProvider:         llmProvider,
		AssistantService:    assistantSvc,
		ConversationHandler: conversationHandler,
		PrivateRoutes: func(g *gin.RouterGroup) {
			knowledge.Register(g, knowledgeHandler)
			conversation.Register(g, conversationHandler)
		},
	}

	r := router.New(deps)
	srv := &http.Server{
		Addr:              cfg.HTTPAddr(),
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "err", err)
	}
	if err := taskQ.Close(shutdownCtx); err != nil {
		slog.Error("taskqueue close error", "err", err)
	}
	slog.Info("stopped")
}
