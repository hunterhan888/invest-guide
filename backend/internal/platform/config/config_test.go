package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// isolateFromRealDotEnv 把测试工作目录切到临时目录，避免 loadDotEnvFromCwd
// 向上查找到项目根的真实 .env（含真实密钥/模型名），污染默认值断言。
func isolateFromRealDotEnv(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
}

func TestLoad_Defaults(t *testing.T) {
	isolateFromRealDotEnv(t)
	t.Setenv("JWT_SECRET", "test-secret-32chars-minimum-length!")
	t.Setenv("LLM_API_KEY", "")

	cfg, err := Load()
	assert.NoError(t, err)
	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "sqlite://dev.db", cfg.DatabaseURL)
	assert.Equal(t, 24*time.Hour, cfg.JWTExpiry)
	// LLM/Embedding 无内置默认提供商，留空由用户配置
	assert.Empty(t, cfg.LLMBaseURL)
	assert.Empty(t, cfg.LLMModel)
	assert.Empty(t, cfg.EmbeddingModel)
	assert.Equal(t, "1024", cfg.EmbeddingDim)
	assert.Equal(t, "", cfg.LogFile)
	assert.Equal(t, 60, cfg.RateLimitAPI)
}

func TestLoad_MissingJWTSecret_FailsFast(t *testing.T) {
	isolateFromRealDotEnv(t)
	t.Setenv("JWT_SECRET", "")
	_, err := Load()
	assert.Error(t, err)
}

func TestLoad_NoAPIKeys_Allowed(t *testing.T) {
	isolateFromRealDotEnv(t)
	t.Setenv("JWT_SECRET", "test-secret-32chars-minimum-length!")
	t.Setenv("LLM_API_KEY", "")
	cfg, err := Load()
	assert.NoError(t, err)
	assert.Empty(t, cfg.LLMAPIKey)
	assert.Empty(t, cfg.EmbeddingAPIKey)
}

func TestLoad_EmbeddingFallsBackToLLM(t *testing.T) {
	isolateFromRealDotEnv(t)
	t.Setenv("JWT_SECRET", "test-secret-32chars-minimum-length!")
	t.Setenv("LLM_API_KEY", "llm-key")
	t.Setenv("LLM_BASE_URL", "https://example.com/v1")
	cfg, err := Load()
	assert.NoError(t, err)
	assert.Equal(t, "llm-key", cfg.EmbeddingAPIKey)
	assert.Equal(t, "https://example.com/v1", cfg.EmbeddingBaseURL)
}

func TestLoad_OverridesFromEnv(t *testing.T) {
	isolateFromRealDotEnv(t)
	t.Setenv("PORT", "9090")
	t.Setenv("JWT_SECRET", "test-secret-32chars-minimum-length!")
	t.Setenv("LLM_API_KEY", "test-key")
	t.Setenv("LLM_MODEL", "deepseek-chat")

	cfg, err := Load()
	assert.NoError(t, err)
	assert.Equal(t, "9090", cfg.Port)
	assert.Equal(t, "deepseek-chat", cfg.LLMModel)
}
