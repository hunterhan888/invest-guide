package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// LoginRateLimit 实现"按 IP 的失败尝试限流"：滑动时间窗口内仅失败请求计数，
// 成功后重置该 IP 的计数。语义符合"5 次失败/15 分钟"。
type LoginRateLimit struct {
	window   time.Duration
	maxFails int

	mu      sync.Mutex
	byIP    map[string]*loginWindow
	nowFunc func() time.Time // 测试可注入
}

type loginWindow struct {
	failures []time.Time
}

func NewLoginRateLimit(maxFails int) *LoginRateLimit {
	return &LoginRateLimit{
		window:   15 * time.Minute,
		maxFails: maxFails,
		byIP:     make(map[string]*loginWindow),
		nowFunc:  time.Now,
	}
}

// MarkSuccess 由 handler 在登录/注册成功后调用，重置该 IP 的失败计数
func (l *LoginRateLimit) MarkSuccess(c *gin.Context) {
	ip := c.ClientIP()
	l.mu.Lock()
	delete(l.byIP, ip)
	l.mu.Unlock()
}

// Handler 返回中间件：请求到达时若该 IP 失败次数已达上限则直接 429。
// 具体是否计入失败由 handler 通过 MarkSuccess 控制（成功清零，失败自然累积）。
func (l *LoginRateLimit) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if l.isBlocked(ip) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error":   "too many failed attempts, try again later",
				"code":    "RATE_LIMITED",
			})
			return
		}
		c.Next()
		// 若 handler 未 abort 且响应是成功（2xx），视为成功并清零
		if !c.IsAborted() && c.Writer.Status() >= 200 && c.Writer.Status() < 300 {
			l.MarkSuccess(c)
		}
	}
}

func (l *LoginRateLimit) isBlocked(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.nowFunc()
	w, ok := l.byIP[ip]
	if !ok {
		return false
	}
	// 清理过期失败记录
	cutoff := now.Add(-l.window)
	kept := w.failures[:0]
	for _, t := range w.failures {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	w.failures = kept
	if len(kept) == 0 {
		delete(l.byIP, ip)
		return false
	}
	return len(kept) >= l.maxFails
}

// RecordFailure 由 handler 在登录/注册失败后调用，记录一次失败
func (l *LoginRateLimit) RecordFailure(c *gin.Context) {
	ip := c.ClientIP()
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.nowFunc()
	w, ok := l.byIP[ip]
	if !ok {
		w = &loginWindow{}
		l.byIP[ip] = w
	}
	cutoff := now.Add(-l.window)
	kept := w.failures[:0]
	for _, t := range w.failures {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	w.failures = append(kept, now)
}
