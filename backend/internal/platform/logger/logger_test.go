package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenFile_CreatesParentDirAndAppends(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "app.log")

	f, err := OpenFile(path)
	require.NoError(t, err)
	_, _ = f.Write([]byte("line1\n"))
	require.NoError(t, f.Close())

	// 再次打开应追加而非覆盖
	f2, err := OpenFile(path)
	require.NoError(t, err)
	_, _ = f2.Write([]byte("line2\n"))
	require.NoError(t, f2.Close())

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "line1\nline2\n", string(content))
}

func TestInitJSON(t *testing.T) {
	var buf bytes.Buffer
	Init(&buf, "debug")
	FromContext(context.Background()).Info("hello", "key", "val")

	var entry map[string]interface{}
	assert.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	assert.Equal(t, "hello", entry["msg"])
	assert.Equal(t, "val", entry["key"])
	assert.Equal(t, "INFO", entry["level"])
}

func TestFromContext_WithRequestID(t *testing.T) {
	var buf bytes.Buffer
	Init(&buf, "info")
	ctx := WithRequestID(context.Background(), "req-123")
	FromContext(ctx).Info("scoped")
	var entry map[string]interface{}
	assert.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	assert.Equal(t, "req-123", entry["request_id"])
}
