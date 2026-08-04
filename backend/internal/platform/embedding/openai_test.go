package embedding

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAIProvider_Embed_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/embeddings", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		var req struct {
			Input []string `json:"input"`
			Model string   `json:"model"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, []string{"hello", "world"}, req.Input)

		resp := map[string]interface{}{
			"data": []map[string]interface{}{
				{"embedding": []float32{0.1, 0.2, 0.3}},
				{"embedding": []float32{0.4, 0.5, 0.6}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p := NewOpenAIProvider(srv.URL, "test-key", "test-model", 3)
	vecs, err := p.Embed(context.Background(), []string{"hello", "world"})
	require.NoError(t, err)
	assert.Len(t, vecs, 2)
	assert.Equal(t, []float32{0.1, 0.2, 0.3}, vecs[0])
	assert.Equal(t, []float32{0.4, 0.5, 0.6}, vecs[1])
	assert.Equal(t, 3, p.Dim())
}

func TestOpenAIProvider_Embed_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := NewOpenAIProvider(srv.URL, "k", "m", 3)
	p.retryDelays = []time.Duration{0, 0, 0, 0}
	_, err := p.Embed(context.Background(), []string{"x"})
	assert.ErrorIs(t, err, ErrProviderUnavailable)
}

func TestOpenAIProvider_Embed_RetryOn429(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"embedding":[0.1,0.2]}]}`))
	}))
	defer srv.Close()

	p := NewOpenAIProvider(srv.URL, "k", "m", 2)
	p.retryDelays = []time.Duration{0, 0, 0, 0} // 测试注入零延迟
	vecs, err := p.Embed(context.Background(), []string{"x"})
	require.NoError(t, err)
	assert.Equal(t, []float32{0.1, 0.2}, vecs[0])
	assert.Equal(t, 2, calls)
}

func TestOpenAIProvider_Embed_DimMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"embedding":[0.1,0.2]}]}`))
	}))
	defer srv.Close()

	p := NewOpenAIProvider(srv.URL, "k", "m", 3)
	_, err := p.Embed(context.Background(), []string{"x"})
	assert.ErrorIs(t, err, ErrInvalidDim)
}

func TestNormalizeBaseURL(t *testing.T) {
	cases := []struct{ in, want string }{
		{"https://api.siliconflow.cn/v1", "https://api.siliconflow.cn"},
		{"https://api.siliconflow.cn/v1/", "https://api.siliconflow.cn"},
		{"https://api.openai.com/v1", "https://api.openai.com"},
		{"https://api.siliconflow.cn", "https://api.siliconflow.cn"},
		{"https://api.siliconflow.cn/", "https://api.siliconflow.cn"},
	}
	for _, tc := range cases {
		got := normalizeBaseURL(tc.in)
		assert.Equal(t, tc.want, got, "normalizeBaseURL(%q)", tc.in)
	}
}

// TestEmbedURL_SiliconFlow：验证最终请求 URL 拼接正确（不会出现 /v1/v1）
func TestEmbedURL_SiliconFlow(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"embedding":[0.1,0.2]}]}`))
	}))
	defer srv.Close()

	// 传入带 /v1 的 base URL
	p := NewOpenAIProvider(srv.URL+"/v1", "k", "m", 2)
	_, err := p.Embed(context.Background(), []string{"x"})
	require.NoError(t, err)
	assert.Equal(t, "/v1/embeddings", gotPath)
}

// TestEmbed_SendsDimensions：验证 dimensions 参数被发送（SiliconFlow Qwen3 需要）
func TestEmbed_SendsDimensions(t *testing.T) {
	var gotReq embedRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotReq)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"embedding":[0.1,0.2,0.3]}]}`))
	}))
	defer srv.Close()

	p := NewOpenAIProvider(srv.URL, "k", "m", 3)
	_, err := p.Embed(context.Background(), []string{"x"})
	require.NoError(t, err)
	require.NotNil(t, gotReq.Dimensions, "dimensions 应被发送")
	assert.Equal(t, 3, *gotReq.Dimensions)
}
