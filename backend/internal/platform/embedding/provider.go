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
