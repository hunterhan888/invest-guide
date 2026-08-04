package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimit 按 IP 限流：每秒 rate 次请求，突发 burst
func RateLimit(ratePerSec, burst int) gin.HandlerFunc {
	var (
		mu       sync.Mutex
		buckets  = make(map[string]*rate.Limiter)
		lastSeen = make(map[string]time.Time)
	)
	go cleanupBuckets(buckets, lastSeen, &mu)

	return func(c *gin.Context) {
		key := c.ClientIP()
		mu.Lock()
		l, ok := buckets[key]
		if !ok {
			l = rate.NewLimiter(rate.Limit(ratePerSec), burst)
			buckets[key] = l
		}
		lastSeen[key] = time.Now()
		mu.Unlock()

		if !l.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error":   "rate limited",
				"code":    "RATE_LIMITED",
			})
			return
		}
		c.Next()
	}
}

func cleanupBuckets(buckets map[string]*rate.Limiter, lastSeen map[string]time.Time, mu *sync.Mutex) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		mu.Lock()
		for k, t := range lastSeen {
			if time.Since(t) > 3*time.Minute {
				delete(buckets, k)
				delete(lastSeen, k)
			}
		}
		mu.Unlock()
	}
}
