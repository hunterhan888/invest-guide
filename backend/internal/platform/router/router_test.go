package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHealthEndpoint_Returns200(t *testing.T) {
	deps := NewTestDeps(t)
	r := New(deps)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/system/health", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.True(t, body["success"].(bool))
	data, ok := body["data"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "ok", data["status"])
}

func TestVersionEndpoint_ReturnsVersion(t *testing.T) {
	deps := NewTestDeps(t)
	r := New(deps)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/system/version", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	data, ok := body["data"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "0.0.1-test", data["version"])
}

func TestUnknownRoute_Returns404(t *testing.T) {
	deps := NewTestDeps(t)
	r := New(deps)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/nope", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestAuthRegister_Public: auth 路由公开可达（无需 token）
func TestAuthRegister_Public(t *testing.T) {
	deps := NewTestDeps(t)
	r := New(deps)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/register",
		bytes.NewBufferString(`{"email":"pub@b.com","password":"password123","displayName":"P"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
}

// TestPrivateGroup_RequiresToken: 鉴权路由组下的路由，未带 token 返回 401
func TestPrivateGroup_RequiresToken(t *testing.T) {
	deps := NewTestDeps(t)
	// 注入一个测试用私有路由
	deps.PrivateRoutes = func(g *gin.RouterGroup) {
		g.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })
	}
	r := New(deps)

	// 未带 token → 401
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/ping", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
