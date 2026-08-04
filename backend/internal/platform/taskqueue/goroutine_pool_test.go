package taskqueue

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGoroutinePool_ExecutesTask(t *testing.T) {
	q := NewGoroutinePool(2, 4)
	defer q.Close(context.Background())

	var count int32
	q.Enqueue(func(ctx context.Context) error {
		atomic.AddInt32(&count, 1)
		return nil
	})
	assert.Eventually(t, func() bool { return atomic.LoadInt32(&count) == 1 }, time.Second, 10*time.Millisecond)
}

func TestGoroutinePool_CancelStopsWorkers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	q := NewGoroutinePool(2, 4)
	cancel()
	assert.NoError(t, q.Close(ctx))
}

func TestGoroutinePool_ProcessesMultiple(t *testing.T) {
	q := NewGoroutinePool(4, 8)
	defer q.Close(context.Background())

	var count int32
	for i := 0; i < 10; i++ {
		q.Enqueue(func(ctx context.Context) error {
			atomic.AddInt32(&count, 1)
			return nil
		})
	}
	assert.Eventually(t, func() bool { return atomic.LoadInt32(&count) == 10 }, 2*time.Second, 20*time.Millisecond)
}
