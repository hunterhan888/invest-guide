package main

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/invest-guide/backend/internal/domain/assistant"
	"github.com/invest-guide/backend/internal/domain/knowledge"
	"github.com/invest-guide/backend/internal/mcp"
	"github.com/invest-guide/backend/internal/platform/cache"
	"github.com/invest-guide/backend/internal/platform/config"
	"github.com/invest-guide/backend/internal/platform/database"
	"github.com/invest-guide/backend/internal/platform/embedding"
	"github.com/invest-guide/backend/internal/platform/llm"
	"github.com/invest-guide/backend/internal/platform/logger"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config failed", "err", err)
		os.Exit(1)
	}
	// stdio 传输：协议走 stdout，日志必须走 stderr，否则会污染 MCP 消息流
	logger.Init(os.Stderr, cfg.LogLevel)
	slog.Info("starting invest-guide mcp server")

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		slog.Error("connect db failed", "err", err)
		os.Exit(1)
	}
	if err := database.AutoMigrate(db,
		&knowledge.KnowledgeDoc{}, &knowledge.KnowledgeChunk{},
	); err != nil {
		slog.Error("auto migrate failed", "err", err)
		os.Exit(1)
	}

	embedDim, _ := strconv.Atoi(cfg.EmbeddingDim)
	if embedDim <= 0 {
		embedDim = 1024
	}
	embedProvider := embedding.NewOpenAIProvider(cfg.EmbeddingBaseURL, cfg.EmbeddingAPIKey, cfg.EmbeddingModel, embedDim)
	knowledgeSvc := knowledge.NewService(
		knowledge.NewGORMDocRepository(db),
		knowledge.NewGORMChunkRepository(db),
		embedProvider,
		nil, // MCP 不依赖 taskqueue（无写入能力）
		knowledge.NewEmbeddingCache(cache.NewLRU(1000, time.Hour)),
	)
	llmProvider := llm.NewOpenAIProvider(cfg.LLMBaseURL, cfg.LLMAPIKey, cfg.LLMModel, cfg.LLMTimeout, cfg.LLMStreamTimeout)
	assistantSvc := assistant.NewService(llmProvider, knowledgeSearcherAdapter{svc: knowledgeSvc})

	svc := mcp.Service{
		Search: func(ctx context.Context, query, country string, topK int) ([]mcp.SearchHit, error) {
			resp, err := knowledgeSvc.Search(ctx, knowledge.SearchRequest{Query: query, Country: country, TopK: topK})
			if err != nil {
				return nil, err
			}
			hits := make([]mcp.SearchHit, len(resp.Chunks))
			for i, c := range resp.Chunks {
				hits[i] = mcp.SearchHit{ID: c.ID, Title: c.Title, Snippet: c.Snippet}
			}
			return hits, nil
		},
		Generate: func(ctx context.Context, question, country string) (string, []mcp.Source, error) {
			answer, sources, _, err := assistantSvc.Generate(ctx, "", question)
			if err != nil {
				return "", nil, err
			}
			srcs := make([]mcp.Source, len(sources))
			for i, s := range sources {
				srcs[i] = mcp.Source{Title: s.Title, Snippet: s.Snippet}
			}
			return answer, srcs, nil
		},
	}

	s := mcp.NewServer(svc)
	if err := server.ServeStdio(s); err != nil {
		slog.Error("mcp server error", "err", err)
		os.Exit(1)
	}
}

// knowledgeSearcherAdapter 把 knowledge.Service 适配为 assistant.KnowledgeSearcher
type knowledgeSearcherAdapter struct {
	svc *knowledge.Service
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
