package llm

import "context"

// FakeProvider 输出确定，用于测试（跨包测试也引用，故放生产文件）
type FakeProvider struct {
	Response     string
	StreamDeltas []string
	Tokens       int
	ModelName    string
}

func NewFakeProvider(response string, deltas []string, tokens int) *FakeProvider {
	return &FakeProvider{Response: response, StreamDeltas: deltas, Tokens: tokens, ModelName: "fake"}
}

func (f *FakeProvider) Generate(ctx context.Context, _ GenerateRequest) (*GenerateResponse, error) {
	return &GenerateResponse{Content: f.Response, TokensUsed: f.Tokens}, nil
}

func (f *FakeProvider) Stream(ctx context.Context, _ GenerateRequest) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk, 1)
	go func() {
		defer close(ch)
		for _, d := range f.StreamDeltas {
			select {
			case <-ctx.Done():
				ch <- StreamChunk{Done: true, Err: ctx.Err()}
				return
			default:
			}
			// 阻塞发送，但仍可被 ctx 取消打断
			select {
			case <-ctx.Done():
				ch <- StreamChunk{Done: true, Err: ctx.Err()}
				return
			case ch <- StreamChunk{Delta: d}:
			}
		}
		select {
		case <-ctx.Done():
			ch <- StreamChunk{Done: true, Err: ctx.Err()}
		case ch <- StreamChunk{Done: true, TokensUsed: f.Tokens}:
		}
	}()
	return ch, nil
}

func (f *FakeProvider) Model() string { return f.ModelName }
