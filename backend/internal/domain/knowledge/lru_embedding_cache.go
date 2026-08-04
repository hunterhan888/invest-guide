package knowledge

// lruEmbeddingCache 把 platform/cache.Cache 适配为 EmbeddingCache。
// key 为查询文本，value 为 []float32 向量。
type lruEmbeddingCache struct {
	store interface {
		Get(string) (interface{}, bool)
		Set(string, interface{})
	}
}

// NewEmbeddingCache 基于现有 LRU 缓存实现 EmbeddingCache。
// 容量与 TTL 由传入的 cache.Cache 决定（见 platform/cache.NewLRU）。
func NewEmbeddingCache(store interface {
	Get(string) (interface{}, bool)
	Set(string, interface{})
}) EmbeddingCache {
	return &lruEmbeddingCache{store: store}
}

func (l *lruEmbeddingCache) Get(text string) ([]float32, bool) {
	v, ok := l.store.Get("embed:" + text)
	if !ok {
		return nil, false
	}
	if vec, ok := v.([]float32); ok {
		return vec, true
	}
	return nil, false
}

func (l *lruEmbeddingCache) Set(text string, vec []float32) {
	l.store.Set("embed:"+text, vec)
}
