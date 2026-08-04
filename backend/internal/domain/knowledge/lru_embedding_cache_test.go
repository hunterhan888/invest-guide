package knowledge

import (
	"testing"
	"time"

	"github.com/invest-guide/backend/internal/platform/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmbeddingCache_SetGet(t *testing.T) {
	store := cache.NewLRU(10, time.Hour)
	c := NewEmbeddingCache(store)

	vec := []float32{0.1, 0.2, 0.3}
	c.Set("越南企业所得税", vec)

	got, ok := c.Get("越南企业所得税")
	require.True(t, ok)
	assert.Equal(t, vec, got)
}

func TestEmbeddingCache_Miss(t *testing.T) {
	store := cache.NewLRU(10, time.Hour)
	c := NewEmbeddingCache(store)

	_, ok := c.Get("不存在的查询")
	assert.False(t, ok)
}

func TestEmbeddingCache_Expires(t *testing.T) {
	store := cache.NewLRU(10, 10*time.Millisecond)
	c := NewEmbeddingCache(store)

	c.Set("query", []float32{1.0})
	time.Sleep(20 * time.Millisecond)
	_, ok := c.Get("query")
	assert.False(t, ok)
}

func TestEmbeddingCache_EvictsByCapacity(t *testing.T) {
	store := cache.NewLRU(2, time.Hour)
	c := NewEmbeddingCache(store)

	c.Set("a", []float32{1})
	c.Set("b", []float32{2})
	c.Set("c", []float32{3}) // 淘汰 a

	_, ok := c.Get("a")
	assert.False(t, ok)
	_, ok = c.Get("c")
	assert.True(t, ok)
}
