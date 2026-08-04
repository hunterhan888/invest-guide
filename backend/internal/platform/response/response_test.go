package response

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() { gin.SetMode(gin.TestMode) }

func TestOk_SuccessNoMessage(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	Ok(c, http.StatusOK, gin.H{"id": "1"})

	var body SuccessResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.True(t, body.Success)
	data, ok := body.Data.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "1", data["id"])
	// Message 为空时省略，不出现在 JSON body 中
	assert.Empty(t, body.Message)
	assert.NotContains(t, w.Body.String(), "message")
}

func TestFail_InvalidInput(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	Fail(c, ErrInvalidInput)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var body ErrorResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.False(t, body.Success)
	assert.Equal(t, "INVALID_INPUT", body.Code)
}

func TestFail_NotFound(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	Fail(c, ErrNotFound)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestFail_WrappedErrorStillMatches(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	wrapped := fmt.Errorf("ctx: %w", ErrNotFound)
	Fail(c, wrapped)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestFail_CodeMappings(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{ErrUnauthorized, http.StatusUnauthorized, "UNAUTHORIZED"},
		{ErrForbidden, http.StatusForbidden, "FORBIDDEN"},
		{ErrConflict, http.StatusConflict, "CONFLICT"},
		{ErrRateLimited, http.StatusTooManyRequests, "RATE_LIMITED"},
		{ErrBadGateway, http.StatusBadGateway, "BAD_GATEWAY"},
		{ErrGatewayTimeout, http.StatusGatewayTimeout, "GATEWAY_TIMEOUT"},
		{ErrInternal, http.StatusInternalServerError, "INTERNAL_ERROR"},
	}
	for _, tc := range cases {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		Fail(c, tc.err)
		assert.Equal(t, tc.status, w.Code, "status for %v", tc.err)
		var body ErrorResponse
		_ = json.Unmarshal(w.Body.Bytes(), &body)
		assert.Equal(t, tc.code, body.Code)
	}
}

func TestFail_InternalDoesNotLeakDetail(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	Fail(c, fmt.Errorf("secret internal: %w", ErrInternal))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var body ErrorResponse
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	assert.NotContains(t, body.Error, "secret")
	assert.Equal(t, "internal error", body.Error)
}
