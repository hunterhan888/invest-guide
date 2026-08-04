package llm

import (
	"bufio"
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
	baseURL      string
	apiKey       string
	model        string
	client       *http.Client
	streamClient *http.Client
}

func NewOpenAIProvider(baseURL, apiKey, model string, timeout, streamTimeout time.Duration) *OpenAIProvider {
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	if streamTimeout == 0 {
		streamTimeout = 120 * time.Second
	}
	return &OpenAIProvider{
		baseURL:      normalizeBaseURL(baseURL),
		apiKey:       apiKey,
		model:        model,
		client:       &http.Client{Timeout: timeout},
		streamClient: &http.Client{Timeout: streamTimeout},
	}
}

// normalizeBaseURL 规范化 base URL，去掉尾斜杠与多余的 /v1，避免重复拼接
func normalizeBaseURL(u string) string {
	u = strings.TrimRight(u, "/")
	if strings.HasSuffix(u, "/v1") {
		u = strings.TrimSuffix(u, "/v1")
	}
	return u
}

func (p *OpenAIProvider) Model() string { return p.model }

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float32       `json:"temperature,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message      chatMessage `json:"message"`
		Delta        chatMessage `json:"delta"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (p *OpenAIProvider) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	body, _ := json.Marshal(p.toChatRequest(req))
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrUnavailable, resp.StatusCode)
	}
	var cr chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return nil, fmt.Errorf("%w: decode: %v", ErrUnavailable, err)
	}
	if len(cr.Choices) == 0 {
		return nil, fmt.Errorf("%w: no choices", ErrUnavailable)
	}
	return &GenerateResponse{
		Content:    cr.Choices[0].Message.Content,
		TokensUsed: cr.Usage.TotalTokens,
	}, nil
}

func (p *OpenAIProvider) Stream(ctx context.Context, req GenerateRequest) (<-chan StreamChunk, error) {
	body, _ := json.Marshal(p.toChatRequest(req, true))
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := p.streamClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("%w: status %d", ErrUnavailable, resp.StatusCode)
	}

	ch := make(chan StreamChunk, 16)
	go p.pumpSSE(ctx, resp, ch)
	return ch, nil
}

func (p *OpenAIProvider) pumpSSE(ctx context.Context, resp *http.Response, ch chan<- StreamChunk) {
	defer close(ch)
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	for {
		select {
		case <-ctx.Done():
			ch <- StreamChunk{Done: true, Err: ctx.Err()}
			return
		default:
		}
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				ch <- StreamChunk{Done: true}
				return
			}
			ch <- StreamChunk{Done: true, Err: fmt.Errorf("%w: read: %v", ErrUnavailable, err)}
			return
		}
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		if !bytes.HasPrefix(line, []byte("data: ")) {
			continue
		}
		data := bytes.TrimPrefix(line, []byte("data: "))
		if bytes.Equal(data, []byte("[DONE]")) {
			ch <- StreamChunk{Done: true}
			return
		}
		var cr chatResponse
		if err := json.Unmarshal(data, &cr); err != nil {
			continue
		}
		if cr.Error != nil {
			ch <- StreamChunk{Done: true, Err: fmt.Errorf("%w: %s", ErrUnavailable, cr.Error.Message)}
			return
		}
		if len(cr.Choices) > 0 {
			c := cr.Choices[0]
			if c.Delta.Content != "" {
				ch <- StreamChunk{Delta: c.Delta.Content}
			}
			if c.FinishReason == "stop" {
				ch <- StreamChunk{Done: true, TokensUsed: cr.Usage.TotalTokens}
				return
			}
		}
	}
}

func (p *OpenAIProvider) toChatRequest(req GenerateRequest, stream ...bool) chatRequest {
	msgs := make([]chatMessage, len(req.Messages))
	for i, m := range req.Messages {
		msgs[i] = chatMessage{Role: m.Role, Content: m.Content}
	}
	cr := chatRequest{
		Model:       nonEmpty(req.Model, p.model),
		Messages:    msgs,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}
	if len(stream) > 0 && stream[0] {
		cr.Stream = true
	}
	return cr
}

func nonEmpty(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}
