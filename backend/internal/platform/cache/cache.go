package cache

// Cache 缓存抽象；生产可切 Redis 实现，开发用 LRU
type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{})
	Delete(key string)
}
