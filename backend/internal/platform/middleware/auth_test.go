package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type fakeAuthSvc struct {
	token string
	uid   string
	err   error
}

func (f *fakeAuthSvc) Authenticate(ctx context.Context, token string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	if token != f.token {
		return "", f.err
	}
	return f.uid, nil
}

func init() { gin.SetMode(gin.TestMode) }

func TestAuth_MissingHeader_Returns401(t *testing.T) {
	r := gin.New()
	svc := &fakeAuthSvc{}
	r.Use(Auth(svc))
	r.GET("/x", func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/x", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuth_BadScheme_Returns401(t *testing.T) {
	r := gin.New()
	r.Use(Auth(&fakeAuthSvc{}))
	r.GET("/x", func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Token abc")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuth_ValidToken_SetsUserID(t *testing.T) {
	svc := &fakeAuthSvc{token: "good", uid: "user-1"}
	r := gin.New()
	r.Use(Auth(svc))

	var gotUserID string
	r.GET("/x", func(c *gin.Context) {
		gotUserID = CurrentUserID(c)
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Bearer good")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "user-1", gotUserID)
}

func TestAuth_InvalidToken_Returns401(t *testing.T) {
	svc := &fakeAuthSvc{err: assert.AnError}
	r := gin.New()
	r.Use(Auth(svc))
	r.GET("/x", func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Bearer bad")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
