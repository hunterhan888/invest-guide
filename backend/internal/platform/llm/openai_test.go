package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAIProvider_Generate_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"hello"}}],"usage":{"total_tokens":42}}`))
	}))
	defer srv.Close()

	p := NewOpenAIProvider(srv.URL, "k", "test-model", 0, 0)
	resp, err := p.Generate(context.Background(), GenerateRequest{
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	})
	require.NoError(t, err)
	assert.Equal(t, "hello", resp.Content)
	assert.Equal(t, 42, resp.TokensUsed)
	assert.Equal(t, "test-model", p.Model())
}

func TestOpenAIProvider_Generate_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 阻塞远超 client timeout 的时间，确保客户端先超时
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := NewOpenAIProvider(srv.URL, "k", "m", 50*time.Millisecond, 0)
	_, err := p.Generate(context.Background(), GenerateRequest{})
	assert.Error(t, err)
}

func TestOpenAIProvider_Stream_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"He\"}}]}\n\n"))
		flusher.Flush()
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"llo\"}}]}\n\n"))
		flusher.Flush()
		_, _ = w.Write([]byte("data: {\"choices\":[{\"finish_reason\":\"stop\"}],\"usage\":{\"total_tokens\":5}}\n\n"))
		flusher.Flush()
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	}))
	defer srv.Close()

	p := NewOpenAIProvider(srv.URL, "k", "m", 0, 0)
	ch, err := p.Stream(context.Background(), GenerateRequest{})
	require.NoError(t, err)

	var deltas string
	var finalChunk StreamChunk
	for c := range ch {
		if c.Done {
			finalChunk = c
			break
		}
		deltas += c.Delta
	}
	assert.Equal(t, "Hello", deltas)
	assert.Equal(t, 5, finalChunk.TokensUsed)
	assert.Nil(t, finalChunk.Err)
}

func TestOpenAIProvider_Stream_ErrorEvent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := NewOpenAIProvider(srv.URL, "k", "m", 0, 0)
	_, err := p.Stream(context.Background(), GenerateRequest{})
	assert.Error(t, err) // HTTP 层失败直接返回 error
}
