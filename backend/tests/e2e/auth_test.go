package e2e

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthFlow_RegisterThenLogin(t *testing.T) {
	srv := NewTestServer()
	defer srv.Close()

	regBody := `{"email":"a@b.com","password":"password123","displayName":"Alice"}`
	resp := postJSON(t, srv.URL+"/api/v1/auth/register", regBody, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var regResp struct {
		Success bool `json:"success"`
		Data    struct {
			Token string `json:"token"`
			User  struct {
				Email string `json:"email"`
			} `json:"user"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(mustReadAll(resp.Body), &regResp))
	assert.Equal(t, "a@b.com", regResp.Data.User.Email)
	assert.NotEmpty(t, regResp.Data.Token)

	resp2 := postJSON(t, srv.URL+"/api/v1/auth/register", regBody, "")
	assert.Equal(t, http.StatusConflict, resp2.StatusCode)

	loginBody := `{"email":"a@b.com","password":"password123"}`
	resp3 := postJSON(t, srv.URL+"/api/v1/auth/login", loginBody, "")
	require.Equal(t, http.StatusOK, resp3.StatusCode)
}

func TestAuth_RegisterInvalidBody(t *testing.T) {
	srv := NewTestServer()
	defer srv.Close()

	resp := postJSON(t, srv.URL+"/api/v1/auth/register", ``, "")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestAuth_LoginBadPassword(t *testing.T) {
	srv := NewTestServer()
	defer srv.Close()

	postJSON(t, srv.URL+"/api/v1/auth/register",
		`{"email":"a@b.com","password":"password123","displayName":"A"}`, "")

	resp := postJSON(t, srv.URL+"/api/v1/auth/login",
		`{"email":"a@b.com","password":"wrong"}`, "")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// TestAuth_LoginReturnedToken_Works: 验证 login 返回的 token 非空且结构正确
func TestAuth_LoginReturnedToken_Works(t *testing.T) {
	srv := NewTestServer()
	defer srv.Close()

	postJSON(t, srv.URL+"/api/v1/auth/register",
		`{"email":"a@b.com","password":"password123","displayName":"A"}`, "")

	resp := postJSON(t, srv.URL+"/api/v1/auth/login",
		`{"email":"a@b.com","password":"password123"}`, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var loginResp struct {
		Success bool `json:"success"`
		Data    struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(mustReadAll(resp.Body), &loginResp))
	assert.NotEmpty(t, loginResp.Data.Token)
}

func postJSON(t *testing.T, url, body, token string) *http.Response {
	t.Helper()
	return doRequest(t, "POST", url, body, token)
}

func doRequest(t *testing.T, method, url, body, token string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, url, bytes.NewBufferString(body))
	require.NoError(t, err)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

func mustReadAll(r io.ReadCloser) []byte {
	b, _ := io.ReadAll(r)
	return b
}
