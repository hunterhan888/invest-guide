package assistant

import (
	"context"
	"errors"
	"testing"

	"github.com/invest-guide/backend/internal/platform/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSearcher struct {
	hits []ContextSource
	err  error
}

func (f *fakeSearcher) Search(ctx context.Context, q, c string, k int) ([]ContextSource, error) {
	return f.hits, f.err
}

func TestService_AssembleContext_IncludesSources(t *testing.T) {
	svc := NewService(nil, &fakeSearcher{
		hits: []ContextSource{{ChunkID: "c1", Title: "T1", Snippet: "S1"}},
	})
	ctx, err := svc.AssembleContext(context.Background(), "q", "")
	require.NoError(t, err)
	assert.Len(t, ctx.Sources, 1)
	assert.Contains(t, ctx.SystemPrompt, "T1")
}

func TestService_AssembleContext_SearchErrorStillReturns(t *testing.T) {
	svc := NewService(nil, &fakeSearcher{err: errors.New("boom")})
	ctx, err := svc.AssembleContext(context.Background(), "q", "")
	require.NoError(t, err)
	assert.Empty(t, ctx.Sources)
}

func TestService_Generate_FullFlow(t *testing.T) {
	p := llm.NewFakeProvider("response", nil, 7)
	svc := NewService(p, &fakeSearcher{
		hits: []ContextSource{{Title: "T", Snippet: "S"}},
	})
	resp, sources, tokens, err := svc.Generate(context.Background(), "", "查询")
	require.NoError(t, err)
	assert.Equal(t, "response", resp)
	assert.Len(t, sources, 1)
	assert.Equal(t, 7, tokens)
}

func TestService_Stream_FullFlow(t *testing.T) {
	p := llm.NewFakeProvider("", []string{"He", "llo"}, 3)
	svc := NewService(p, &fakeSearcher{
		hits: []ContextSource{{Title: "T", Snippet: "S"}},
	})
	ch, sources, err := svc.Stream(context.Background(), "查询", "")
	require.NoError(t, err)
	assert.Len(t, sources, 1)

	var out string
	var final llm.StreamChunk
	for c := range ch {
		if c.Done {
			final = c
		} else {
			out += c.Delta
		}
	}
	assert.Equal(t, "Hello", out)
	assert.Equal(t, 3, final.TokensUsed)
}
