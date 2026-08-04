package cache

import (
	"container/list"
	"sync"
	"time"
)

type lruEntry struct {
	key       string
	value     interface{}
	expiresAt time.Time
}

type LRU struct {
	capacity int
	ttl      time.Duration
	mu       sync.Mutex
	order    *list.List
	items    map[string]*list.Element
}

func NewLRU(capacity int, ttl time.Duration) *LRU {
	return &LRU{
		capacity: capacity,
		ttl:      ttl,
		order:    list.New(),
		items:    make(map[string]*list.Element),
	}
}

func (l *LRU) Get(key string) (interface{}, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	el, ok := l.items[key]
	if !ok {
		return nil, false
	}
	entry := el.Value.(*lruEntry)
	if time.Now().After(entry.expiresAt) {
		l.removeElement(el)
		return nil, false
	}
	l.order.MoveToFront(el)
	return entry.value, true
}

func (l *LRU) Set(key string, value interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if el, ok := l.items[key]; ok {
		entry := el.Value.(*lruEntry)
		entry.value = value
		entry.expiresAt = time.Now().Add(l.ttl)
		l.order.MoveToFront(el)
		return
	}
	entry := &lruEntry{key: key, value: value, expiresAt: time.Now().Add(l.ttl)}
	el := l.order.PushFront(entry)
	l.items[key] = el
	if l.order.Len() > l.capacity {
		l.evict()
	}
}

func (l *LRU) Delete(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if el, ok := l.items[key]; ok {
		l.removeElement(el)
	}
}

func (l *LRU) removeElement(el *list.Element) {
	entry := el.Value.(*lruEntry)
	l.order.Remove(el)
	delete(l.items, entry.key)
}

func (l *LRU) evict() {
	el := l.order.Back()
	if el != nil {
		l.removeElement(el)
	}
}
