package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func init() { gin.SetMode(gin.TestMode) }

func newHandlerForTest() (*Handler, *fakeUserRepo) {
	repo := newFakeUserRepo()
	jwt := NewJWTIssuer("test-secret", "invest-guide", 1<<30)
	svc := &Service{repo: repo, jwt: jwt, bcryptCost: bcrypt.MinCost}
	return NewHandler(svc), repo
}

func doJSON(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func TestHandler_Register_Success(t *testing.T) {
	h, _ := newHandlerForTest()
	r := gin.New()
	r.POST("/auth/register", h.Register)

	body := `{"email":"a@b.com","password":"password123","displayName":"A"}`
	w := doJSON(r, "POST", "/auth/register", body)
	assert.Equal(t, http.StatusCreated, w.Code)

	var resp struct {
		Success bool         `json:"success"`
		Data    AuthResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.Equal(t, "a@b.com", resp.Data.User.Email)
	assert.NotEmpty(t, resp.Data.Token)
}

func TestHandler_Register_InvalidInput(t *testing.T) {
	h, _ := newHandlerForTest()
	r := gin.New()
	r.POST("/auth/register", h.Register)

	w := doJSON(r, "POST", "/auth/register", `{"email":"not-an-email"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Register_DuplicateEmail_Conflict(t *testing.T) {
	h, _ := newHandlerForTest()
	r := gin.New()
	r.POST("/auth/register", h.Register)

	body := `{"email":"a@b.com","password":"password123","displayName":"A"}`
	_ = doJSON(r, "POST", "/auth/register", body)
	w := doJSON(r, "POST", "/auth/register", body)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestHandler_Login_Success(t *testing.T) {
	h, _ := newHandlerForTest()
	r := gin.New()
	r.POST("/auth/register", h.Register)
	r.POST("/auth/login", h.Login)

	body := `{"email":"a@b.com","password":"password123","displayName":"A"}`
	_ = doJSON(r, "POST", "/auth/register", body)

	lb := `{"email":"a@b.com","password":"password123"}`
	w := doJSON(r, "POST", "/auth/login", lb)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Success bool         `json:"success"`
		Data    AuthResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Data.Token)
}

func TestHandler_Login_WrongPassword_Unauthorized(t *testing.T) {
	h, _ := newHandlerForTest()
	r := gin.New()
	r.POST("/auth/register", h.Register)
	r.POST("/auth/login", h.Login)

	_ = doJSON(r, "POST", "/auth/register", `{"email":"a@b.com","password":"password123","displayName":"A"}`)
	w := doJSON(r, "POST", "/auth/login", `{"email":"a@b.com","password":"wrong"}`)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
