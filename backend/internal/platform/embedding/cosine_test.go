package embedding

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCosine_SameVector(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{1, 0, 0}
	assert.InDelta(t, 1.0, Cosine(a, b), 1e-6)
}

func TestCosine_Orthogonal(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{0, 1, 0}
	assert.InDelta(t, 0.0, Cosine(a, b), 1e-6)
}

func TestCosine_Opposite(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{-1, 0, 0}
	assert.InDelta(t, -1.0, Cosine(a, b), 1e-6)
}

func TestCosine_DimMismatch_Panics(t *testing.T) {
	defer func() { _ = recover() }()
	a := []float32{1, 0}
	b := []float32{1, 0, 0}
	_ = Cosine(a, b)
	t.Fatal("expected panic")
}

func TestTopK_Basic(t *testing.T) {
	vectors := [][]float32{
		{1, 0, 0},
		{0, 1, 0},
		{0.9, 0.1, 0},
		{0, 0, 1},
	}
	query := []float32{1, 0, 0}
	hits := TopK(query, vectors, 2)
	assert.Len(t, hits, 2)
	assert.Equal(t, 0, hits[0].Index)
	assert.InDelta(t, 1.0, hits[0].Score, 1e-6)
	assert.Equal(t, 2, hits[1].Index)
}

func TestTopK_EmptyQuery(t *testing.T) {
	hits := TopK(nil, [][]float32{{1, 0}}, 3)
	assert.Empty(t, hits)
}
