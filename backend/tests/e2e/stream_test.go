package e2e

import (
	"bufio"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSEStream_FullFlow(t *testing.T) {
	srv := NewTestServerWithFakeLLM([]string{"Hello ", "world"}, 7)
	defer srv.Close()

	token := registerAndLogin(t, srv)

	resp := doAuth(t, "POST", srv.URL+"/api/v1/conversations", `{"country":"越南"}`, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var cr struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(mustReadAll(resp.Body), &cr))
	convID := cr.Data.ID

	resp = doAuth(t, "POST", srv.URL+"/api/v1/conversations/"+convID+"/messages",
		`{"content":"你好"}`, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var pm struct {
		Data struct {
			MessageID string `json:"messageId"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(mustReadAll(resp.Body), &pm))
	msgID := pm.Data.MessageID

	req, _ := http.NewRequest("GET", srv.URL+"/api/v1/conversations/"+convID+"/messages/"+msgID+"/stream", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "text/event-stream")
	client := &http.Client{Timeout: 10 * time.Second}
	streamResp, err := client.Do(req)
	require.NoError(t, err)
	defer streamResp.Body.Close()
	require.Equal(t, http.StatusOK, streamResp.StatusCode)

	scanner := bufio.NewScanner(streamResp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var messageDeltas []string
	var gotDone bool
	var gotError bool
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			eventType := strings.TrimPrefix(line, "event: ")
			if !scanner.Scan() {
				break
			}
			dataLine := strings.TrimPrefix(scanner.Text(), "data: ")

			switch eventType {
			case "message":
				var d struct {
					Delta string `json:"delta"`
				}
				_ = json.Unmarshal([]byte(dataLine), &d)
				messageDeltas = append(messageDeltas, d.Delta)
			case "done":
				gotDone = true
			case "error":
				gotError = true
			}
		}
		if gotDone || gotError {
			break
		}
	}

	assert.False(t, gotError, "should not receive error event")
	assert.True(t, gotDone, "should receive done event")
	assert.Equal(t, []string{"Hello ", "world"}, messageDeltas)
}

func TestSSEStream_InvalidMessageID_NotFound(t *testing.T) {
	srv := NewTestServerWithFakeLLM(nil, 0)
	defer srv.Close()

	token := registerAndLogin(t, srv)

	resp := doAuth(t, "POST", srv.URL+"/api/v1/conversations", `{}`, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var cr struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(mustReadAll(resp.Body), &cr))

	resp = doAuth(t, "GET",
		srv.URL+"/api/v1/conversations/"+cr.Data.ID+"/messages/nonexistent/stream", "", token)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
