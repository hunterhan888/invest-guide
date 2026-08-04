package assistant

import (
	"context"

	"github.com/invest-guide/backend/internal/platform/llm"
)

// KnowledgeSearcher 是 knowledge 检索接口（由 Deps 层提供适配器）
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
// 预算简化为字符数近似（4 chars ≈ 1 token，~6000 chars）
const maxContextChars = 6000

func (s *Service) AssembleContext(ctx context.Context, query, country string) (*AssembledContext, error) {
	var sources []ContextSource
	if s.searcher != nil {
		hits, err := s.searcher.Search(ctx, query, country, 5)
		if err == nil {
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
