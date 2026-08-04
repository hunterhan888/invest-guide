package llm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFakeProvider_Generate(t *testing.T) {
	p := NewFakeProvider("hello", nil, 5)
	resp, err := p.Generate(context.Background(), GenerateRequest{})
	require.NoError(t, err)
	assert.Equal(t, "hello", resp.Content)
}

func TestFakeProvider_Stream(t *testing.T) {
	p := NewFakeProvider("", []string{"He", "llo"}, 5)
	ch, err := p.Stream(context.Background(), GenerateRequest{})
	require.NoError(t, err)

	var out string
	var final StreamChunk
	for c := range ch {
		if c.Done {
			final = c
		} else {
			out += c.Delta
		}
	}
	assert.Equal(t, "Hello", out)
	assert.Equal(t, 5, final.TokensUsed)
}

func TestFakeProvider_Stream_CancelMidway(t *testing.T) {
	p := NewFakeProvider("", []string{"a", "b", "c"}, 0)
	ctx, cancel := context.WithCancel(context.Background())
	ch, _ := p.Stream(ctx, GenerateRequest{})

	// 读第一个 chunk 后取消
	<-ch
	cancel()

	// 持续读到 Done：取消后应尽快以 error 终止
	for c := range ch {
		if c.Done {
			assert.ErrorIs(t, c.Err, context.Canceled)
			return
		}
	}
	t.Fatal("stream did not terminate after cancel")
}
