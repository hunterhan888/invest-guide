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
