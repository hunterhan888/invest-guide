package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLRU_SetGet(t *testing.T) {
	c := NewLRU(2, time.Hour)
	c.Set("a", "1")
	v, ok := c.Get("a")
	assert.True(t, ok)
	assert.Equal(t, "1", v)
}

func TestLRU_EvictsOnCapacity(t *testing.T) {
	c := NewLRU(2, time.Hour)
	c.Set("a", "1")
	c.Set("b", "2")
	c.Set("c", "3") // 应淘汰 a
	_, ok := c.Get("a")
	assert.False(t, ok)
	_, ok = c.Get("b")
	assert.True(t, ok)
}

func TestLRU_ExpiresAfterTTL(t *testing.T) {
	c := NewLRU(2, 10*time.Millisecond)
	c.Set("a", "1")
	time.Sleep(20 * time.Millisecond)
	_, ok := c.Get("a")
	assert.False(t, ok)
}

func TestLRU_Delete(t *testing.T) {
	c := NewLRU(2, time.Hour)
	c.Set("a", "1")
	c.Delete("a")
	_, ok := c.Get("a")
	assert.False(t, ok)
}
