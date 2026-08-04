package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() { gin.SetMode(gin.TestMode) }

func TestRequestID_InjectsHeader(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, c.GetString("request_id"))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	r.ServeHTTP(w, req)
	assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
	assert.Equal(t, w.Header().Get("X-Request-ID"), w.Body.String())
}

func TestCORS_AddsAllowOrigin(t *testing.T) {
	r := gin.New()
	r.Use(CORS("http://localhost:5173"))
	r.GET("/", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	r.ServeHTTP(w, req)
	assert.Equal(t, "http://localhost:5173", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestRecovery_CapturesPanic(t *testing.T) {
	r := gin.New()
	r.Use(Recovery())
	r.GET("/boom", func(c *gin.Context) { panic("x") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/boom", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRateLimit_AllowsUnderLimit(t *testing.T) {
	r := gin.New()
	r.Use(RateLimit(5, 3)) // 每秒 5 个，突发 3：允许 3 个连续请求
	r.GET("/", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestRequestLogger_LogsEachRequest(t *testing.T) {
	var buf bytes.Buffer
	orig := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	defer slog.SetDefault(orig)

	r := gin.New()
	r.Use(RequestID(), RequestLogger())
	r.GET("/health", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	r.POST("/fail", func(c *gin.Context) { c.String(http.StatusBadRequest, "bad") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)
	r.ServeHTTP(w, req)

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest("POST", "/fail", nil))

	out := buf.String()
	for _, want := range []string{`"msg":"request"`, `"method":"GET"`, `"path":"/health"`, `"status":200`, `"method":"POST"`, `"path":"/fail"`, `"status":400`} {
		assert.True(t, strings.Contains(out, want), "log should contain %s\n%s", want, out)
	}
	assert.True(t, strings.Contains(out, `"request_id":"`), "log should include request_id")
}
