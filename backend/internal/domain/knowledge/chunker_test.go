package knowledge

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChunker_SplitsLongText(t *testing.T) {
	text := strings.Repeat("ab", 200) // 400 chars
	chunks := Chunk(text, 100, 10)
	assert.GreaterOrEqual(t, len(chunks), 4)
	for _, c := range chunks {
		assert.LessOrEqual(t, len(c), 100)
	}
}

func TestChunker_ShortText_OneChunk(t *testing.T) {
	text := "short text"
	chunks := Chunk(text, 100, 10)
	require.Len(t, chunks, 1)
	assert.Equal(t, text, chunks[0])
}

func TestChunker_OverlapBetweenChunks(t *testing.T) {
	text := "0123456789ABCDEFGHIJ" // 20 chars
	chunks := Chunk(text, 10, 4)   // chunkSize=10, overlap=4
	require.GreaterOrEqual(t, len(chunks), 2)
	assert.Equal(t, chunks[0][6:10], chunks[1][0:4])
}

func TestChunker_PreservesParagraphBoundaries(t *testing.T) {
	text := "First paragraph here.\n\nSecond paragraph here.\n\nThird one."
	chunks := Chunk(text, 1000, 0)
	require.Len(t, chunks, 1)
	assert.Contains(t, chunks[0], "First paragraph")
	assert.Contains(t, chunks[0], "Third one")
}

func TestChunker_EmptyText(t *testing.T) {
	chunks := Chunk("", 100, 10)
	assert.Empty(t, chunks)
}
