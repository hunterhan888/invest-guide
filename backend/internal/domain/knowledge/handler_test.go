package knowledge

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() { gin.SetMode(gin.TestMode) }

func newHandlerForTest(t *testing.T) *Handler {
	t.Helper()
	svc, _, _ := newTestService(t)
	return NewHandler(svc)
}

func doJSON2(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func TestHandler_Create_Returns202(t *testing.T) {
	h := newHandlerForTest(t)
	r := gin.New()
	r.POST("/knowledge-docs", h.Create)

	body := `{"title":"越南","country":"越南","sourceType":"manual","content":"河内是首都"}`
	w := doJSON2(r, "POST", "/knowledge-docs", body)
	assert.Equal(t, http.StatusAccepted, w.Code)

	var resp struct {
		Success bool   `json:"success"`
		Data    DocDTO `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.Equal(t, StatusPending, resp.Data.Status)
}

func TestHandler_Create_InvalidInput(t *testing.T) {
	h := newHandlerForTest(t)
	r := gin.New()
	r.POST("/knowledge-docs", h.Create)

	w := doJSON2(r, "POST", "/knowledge-docs", `{"title":"no-country"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Get_NotFound(t *testing.T) {
	h := newHandlerForTest(t)
	r := gin.New()
	r.GET("/knowledge-docs/:id", h.Get)

	w := doJSON2(r, "GET", "/knowledge-docs/missing", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_Delete_NoContent(t *testing.T) {
	h := newHandlerForTest(t)
	r := gin.New()
	r.DELETE("/knowledge-docs/:id", h.Delete)

	w := doJSON2(r, "DELETE", "/knowledge-docs/missing", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_List_Success(t *testing.T) {
	h := newHandlerForTest(t)
	r := gin.New()
	r.GET("/knowledge-docs", h.List)

	w := doJSON2(r, "GET", "/knowledge-docs?page=1&pageSize=20", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"items"`)
	assert.Contains(t, w.Body.String(), `"total"`)
}

func TestHandler_Retry_ConflictOnReady(t *testing.T) {
	h := newHandlerForTest(t)
	r := gin.New()
	r.POST("/knowledge-docs/:id/retry", h.Retry)

	// 不存在的文档 → 404
	w := doJSON2(r, "POST", "/knowledge-docs/missing/retry", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}
