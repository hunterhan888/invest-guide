package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() { gin.SetMode(gin.TestMode) }

// TestLoginRateLimit_TriggersAfterThreshold: 5 次失败/窗口，第 5 次失败后触发 429
func TestLoginRateLimit_TriggersAfterThreshold(t *testing.T) {
	r := gin.New()
	rl := NewLoginRateLimit(5)
	r.Use(rl.Handler())
	r.POST("/auth/login", func(c *gin.Context) {
		rl.RecordFailure(c)
		c.Status(http.StatusUnauthorized)
	})

	// 前 4 次失败放行
	for i := 0; i < 4; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("POST", "/auth/login", nil))
		assert.Equal(t, http.StatusUnauthorized, w.Code, "失败请求 %d 应放行", i+1)
	}
	// 第 5 次失败达到阈值 → 放行但已达上限；第 6 次应 429
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/auth/login", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code, "第 5 次失败仍放行（handler 层）")

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/auth/login", nil))
	assert.Equal(t, http.StatusTooManyRequests, w.Code, "第 6 次请求应被中间件拦截")
}

// TestLoginRateLimit_FailureOnly: 中间件标记为成功(failed=false)的请求不计数
func TestLoginRateLimit_FailureOnly(t *testing.T) {
	r := gin.New()
	rl := NewLoginRateLimit(2)
	r.Use(rl.Handler())
	// 模拟真实 handler：成功后调用 rl.MarkSuccess 清除计数；失败时调用 RecordFailure
	r.POST("/auth/login", func(c *gin.Context) {
		if c.GetHeader("X-Auth-OK") == "1" {
			c.Status(http.StatusOK)
			return
		}
		rl.RecordFailure(c)
		c.Status(http.StatusUnauthorized)
	})

	// 成功请求（handler 返回 2xx，中间件自动 MarkSuccess 清零）不累计失败
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/auth/login", nil)
		req.Header.Set("X-Auth-OK", "1")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "成功请求 %d 不应被限流", i+1)
	}

	// 失败 2 次后第 3 次被限流
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("POST", "/auth/login", nil))
		assert.Equal(t, http.StatusUnauthorized, w.Code, "失败请求 %d 应放行", i+1)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/auth/login", nil))
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}
