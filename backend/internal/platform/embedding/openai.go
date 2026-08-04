package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type OpenAIProvider struct {
	baseURL     string
	apiKey      string
	model       string
	dim         int
	client      *http.Client
	retryDelays []time.Duration
}

func NewOpenAIProvider(baseURL, apiKey, model string, dim int) *OpenAIProvider {
	return &OpenAIProvider{
		baseURL:     normalizeBaseURL(baseURL),
		apiKey:      apiKey,
		model:       model,
		dim:         dim,
		client:      &http.Client{Timeout: 30 * time.Second},
		retryDelays: []time.Duration{0, time.Second, 2 * time.Second, 4 * time.Second},
	}
}

// normalizeBaseURL 规范化 base URL，去掉尾斜杠与多余的 /v1，避免重复拼接
// 例：https://api.siliconflow.cn/v1/ → https://api.siliconflow.cn
func normalizeBaseURL(u string) string {
	u = strings.TrimRight(u, "/")
	if strings.HasSuffix(u, "/v1") {
		u = strings.TrimSuffix(u, "/v1")
	}
	return u
}

type embedRequest struct {
	Input      []string `json:"input"`
	Model      string   `json:"model"`
	Dimensions *int     `json:"dimensions,omitempty"` // 部分 provider（如 SiliconFlow Qwen3）支持强制输出维度
}

type embedResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (p *OpenAIProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	body, _ := json.Marshal(embedRequest{Input: texts, Model: p.model, Dimensions: &p.dim})
	url := p.baseURL + "/v1/embeddings"

	var resp embedResponse
	if err := p.doWithRetry(ctx, url, body, &resp); err != nil {
		return nil, err
	}

	if len(resp.Data) != len(texts) {
		return nil, fmt.Errorf("%w: expected %d vectors, got %d", ErrProviderUnavailable, len(texts), len(resp.Data))
	}

	out := make([][]float32, len(resp.Data))
	for i, d := range resp.Data {
		if len(d.Embedding) != p.dim {
			return nil, fmt.Errorf("%w: vector %d has dim %d, expected %d", ErrInvalidDim, i, len(d.Embedding), p.dim)
		}
		out[i] = d.Embedding
	}
	return out, nil
}

func (p *OpenAIProvider) Dim() int { return p.dim }

// doWithRetry 重试策略：429 / 5xx → 指数退避，最多 3 次
func (p *OpenAIProvider) doWithRetry(ctx context.Context, url string, body []byte, out *embedResponse) error {
	delays := p.retryDelays
	var lastErr error

	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delays[attempt]):
			}
		}

		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+p.apiKey)

		resp, err := p.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("%w: status %d", ErrProviderUnavailable, resp.StatusCode)
			resp.Body.Close()
			continue
		}
		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return fmt.Errorf("%w: status %d: %s", ErrProviderUnavailable, resp.StatusCode, bodyBytes)
		}

		decodeErr := json.NewDecoder(resp.Body).Decode(out)
		resp.Body.Close()
		if decodeErr != nil {
			return fmt.Errorf("%w: decode: %v", ErrProviderUnavailable, decodeErr)
		}
		return nil
	}
	return lastErr
}
