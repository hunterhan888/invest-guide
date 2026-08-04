package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port               string
	DatabaseURL        string
	JWTSecret          string
	JWTExpiry          time.Duration
	LLMBaseURL         string
	LLMAPIKey          string
	LLMModel           string
	LLMTimeout         time.Duration
	LLMStreamTimeout   time.Duration
	EmbeddingBaseURL   string
	EmbeddingAPIKey    string
	EmbeddingModel     string
	EmbeddingDim       string
	CORSOrigins        string
	LogLevel           string
	LogFile            string
	RateLimitAPI       int
	RateLimitSensitive int
}

func Load() (*Config, error) {
	// 加载项目根 .env（本地开发）；不覆盖已存在的环境变量
	_ = loadDotEnvFromCwd()

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	// LLM / Embedding API Key 改为可选：缺失时允许启动，
	// 调用 LLM / Embedding 时返回 502（便于开发阶段先跑通其他链路）。
	llmKey := os.Getenv("LLM_API_KEY")

	jwtExpiry, err := time.ParseDuration(envOrDefault("JWT_EXPIRY", "24h"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_EXPIRY: %w", err)
	}
	llmTimeout, err := time.ParseDuration(envOrDefault("LLM_TIMEOUT", "30s"))
	if err != nil {
		return nil, fmt.Errorf("invalid LLM_TIMEOUT: %w", err)
	}
	llmStreamTimeout, err := time.ParseDuration(envOrDefault("LLM_STREAM_TIMEOUT", "120s"))
	if err != nil {
		return nil, fmt.Errorf("invalid LLM_STREAM_TIMEOUT: %w", err)
	}

	// LLM / Embedding 配置全部由用户自定义，无内置默认提供商
	llmBaseURL := envOrDefault("LLM_BASE_URL", "")
	llmModel := envOrDefault("LLM_MODEL", "")

	return &Config{
		Port:               envOrDefault("PORT", "8080"),
		DatabaseURL:        envOrDefault("DATABASE_URL", "sqlite://dev.db"),
		JWTSecret:          jwtSecret,
		JWTExpiry:          jwtExpiry,
		LLMBaseURL:         llmBaseURL,
		LLMAPIKey:          llmKey,
		LLMModel:           llmModel,
		LLMTimeout:         llmTimeout,
		LLMStreamTimeout:   llmStreamTimeout,
		EmbeddingBaseURL:   envOrDefault("EMBEDDING_BASE_URL", llmBaseURL),
		EmbeddingAPIKey:    envOrDefault("EMBEDDING_API_KEY", llmKey),
		EmbeddingModel:     envOrDefault("EMBEDDING_MODEL", ""),
		EmbeddingDim:       envOrDefault("EMBEDDING_DIM", "1024"),
		CORSOrigins:        envOrDefault("CORS_ORIGINS", "http://localhost:5173"),
		LogLevel:           envOrDefault("LOG_LEVEL", "info"),
		LogFile:            envOrDefault("LOG_FILE", ""),
		RateLimitAPI:       envIntOrDefault("RATE_LIMIT_API", 60),
		RateLimitSensitive: envIntOrDefault("RATE_LIMIT_SENSITIVE", 20),
	}, nil
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envIntOrDefault(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

// HTTPAddr 返回监听地址，便于 main.go 直接调用
func (c *Config) HTTPAddr() string { return ":" + c.Port }
