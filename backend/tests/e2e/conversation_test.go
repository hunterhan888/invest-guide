package e2e

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConversation_Create_PostMessage_ListMessages(t *testing.T) {
	srv := NewTestServer()
	defer srv.Close()

	token := registerAndLogin(t, srv)

	resp := doAuth(t, "POST", srv.URL+"/api/v1/conversations",
		`{"title":"越南税收","country":"越南"}`, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var cr struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(mustReadAll(resp.Body), &cr))
	convID := cr.Data.ID

	resp = doAuth(t, "POST", srv.URL+"/api/v1/conversations/"+convID+"/messages",
		`{"content":"越南的企业所得税率是多少？"}`, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var pm struct {
		Data struct {
			MessageID string `json:"messageId"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(mustReadAll(resp.Body), &pm))
	require.NotEmpty(t, pm.Data.MessageID)

	resp = doAuth(t, "GET", srv.URL+"/api/v1/conversations/"+convID+"/messages", "", token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var ml struct {
		Data struct {
			Items []struct {
				Role string `json:"role"`
			} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(mustReadAll(resp.Body), &ml))
	assert.GreaterOrEqual(t, len(ml.Data.Items), 1)
}

func TestConversation_OtherUserCannotAccess(t *testing.T) {
	srv := NewTestServer()
	defer srv.Close()

	tokenA := registerAndLoginAt(t, srv, "a@b.com")
	resp := doAuth(t, "POST", srv.URL+"/api/v1/conversations", `{"title":"A的"}`, tokenA)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var cr struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(mustReadAll(resp.Body), &cr))

	tokenB := registerAndLoginAt(t, srv, "b@b.com")
	resp = doAuth(t, "GET", srv.URL+"/api/v1/conversations/"+cr.Data.ID, "", tokenB)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func registerAndLoginAt(t *testing.T, srv *TestServer, email string) string {
	t.Helper()
	body := `{"email":"` + email + `","password":"password123","displayName":"X"}`
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
