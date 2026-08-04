package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 写一个临时 .env 并加载
func writeTempEnv(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestLoadDotEnvFile_Simple(t *testing.T) {
	path := writeTempEnv(t, "LLM_BASE_URL=https://example.com/v1\nLLM_MODEL=test-model\n")
	require.NoError(t, loadDotEnvFile(path))
	assert.Equal(t, "https://example.com/v1", os.Getenv("LLM_BASE_URL"))
	assert.Equal(t, "test-model", os.Getenv("LLM_MODEL"))
}

func TestLoadDotEnvFile_IgnoresCommentsAndBlank(t *testing.T) {
	path := writeTempEnv(t, "# 注释行\n\nPORT=9090\n")
	require.NoError(t, loadDotEnvFile(path))
	assert.Equal(t, "9090", os.Getenv("PORT"))
}

func TestLoadDotEnvFile_HandlesQuotes(t *testing.T) {
	path := writeTempEnv(t, `LLM_API_KEY="sk-abc 123"`+"\n")
	require.NoError(t, loadDotEnvFile(path))
	assert.Equal(t, "sk-abc 123", os.Getenv("LLM_API_KEY"))
}

func TestLoadDotEnvFile_DoesNotOverrideExistingEnv(t *testing.T) {
	t.Setenv("LLM_MODEL", "already-set")
	_ = os.Unsetenv("LLM_BASE_URL") // 确保未设置，.env 才会填充它
	path := writeTempEnv(t, "LLM_MODEL=from-file\nLLM_BASE_URL=https://x.com/v1\n")
	require.NoError(t, loadDotEnvFile(path))
	// 已存在的环境变量不被 .env 覆盖
	assert.Equal(t, "already-set", os.Getenv("LLM_MODEL"))
	// 未设置的被 .env 填充
	assert.Equal(t, "https://x.com/v1", os.Getenv("LLM_BASE_URL"))
}

func TestLoadDotEnvFile_MissingFile_NoError(t *testing.T) {
	assert.NoError(t, loadDotEnvFile(filepath.Join(t.TempDir(), "nope.env")))
}
