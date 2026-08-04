package embedding

import (
	"math"
	"sort"
)

type Hit struct {
	Index int
	Score float64
}

func Cosine(a, b []float32) float64 {
	if len(a) != len(b) {
		panic("cosine: dim mismatch")
	}
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

func TopK(query []float32, vectors [][]float32, k int) []Hit {
	if query == nil || len(vectors) == 0 || k <= 0 {
		return nil
	}
	hits := make([]Hit, 0, len(vectors))
	for i, v := range vectors {
		hits = append(hits, Hit{Index: i, Score: Cosine(query, v)})
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })
	if k > len(hits) {
		k = len(hits)
	}
	return hits[:k]
}
