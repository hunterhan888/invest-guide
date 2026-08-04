package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKnowledge_CreateThenGet(t *testing.T) {
	srv := NewTestServer()
	defer srv.Close()

	token := registerAndLogin(t, srv)

	body := `{"title":"越南投资指南","country":"越南","sourceType":"manual","content":"越南的工业园区集中在北部。河内是首都。胡志明市是经济中心。"}`
	resp := doAuth(t, "POST", srv.URL+"/api/v1/knowledge-docs", body, token)
	require.Equal(t, http.StatusAccepted, resp.StatusCode)

	var createResp struct {
		Success bool `json:"success"`
		Data    struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(mustReadAll(resp.Body), &createResp))
	require.NotEmpty(t, createResp.Data.ID)

	// 轮询状态直到 ready/failed（e2eQueue 同步执行，应立即 ready）
	var status string
	for i := 0; i < 20; i++ {
		time.Sleep(100 * time.Millisecond)
		gResp := doAuth(t, "GET", srv.URL+"/api/v1/knowledge-docs/"+createResp.Data.ID, "", token)
		var g struct {
			Success bool `json:"success"`
			Data    struct {
				Status     string `json:"status"`
				ChunkCount int    `json:"chunkCount"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(mustReadAll(gResp.Body), &g))
		status = g.Data.Status
		if status == "ready" || status == "failed" {
			break
		}
	}
	assert.Equal(t, "ready", status)

	lResp := doAuth(t, "GET", srv.URL+"/api/v1/knowledge-docs", "", token)
	require.Equal(t, http.StatusOK, lResp.StatusCode)
}

func TestKnowledge_Unauthenticated_Blocked(t *testing.T) {
	srv := NewTestServer()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/knowledge-docs")
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func registerAndLogin(t *testing.T, srv *TestServer) string {
	t.Helper()
	body := `{"email":"k@b.com","password":"password123","displayName":"K"}`
	resp := postJSON(t, srv.URL+"/api/v1/auth/register", body, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var r struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(mustReadAll(resp.Body), &r))
	return r.Data.Token
}

func doAuth(t *testing.T, method, url, body, token string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(method, url, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}
