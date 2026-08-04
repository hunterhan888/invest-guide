package taskqueue

import (
	"context"
	"sync"
)

type GoroutinePool struct {
	wg      sync.WaitGroup
	tasks   chan Task
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool
	mu      sync.Mutex
}

func NewGoroutinePool(workers, buffer int) *GoroutinePool {
	ctx, cancel := context.WithCancel(context.Background())
	p := &GoroutinePool{
		tasks:  make(chan Task, buffer),
		ctx:    ctx,
		cancel: cancel,
	}
	for i := 0; i < workers; i++ {
		p.wg.Add(1)
		go p.worker()
	}
	return p
}

func (p *GoroutinePool) worker() {
	defer p.wg.Done()
	for {
		select {
		case <-p.ctx.Done():
			return
		case task, ok := <-p.tasks:
			if !ok {
				return
			}
			_ = task(p.ctx)
		}
	}
}

func (p *GoroutinePool) Enqueue(task Task) error {
	p.mu.Lock()
	if p.stopped {
		p.mu.Unlock()
		return p.ctx.Err()
	}
	p.mu.Unlock()
	select {
	case p.tasks <- task:
		return nil
	case <-p.ctx.Done():
		return p.ctx.Err()
	}
}

func (p *GoroutinePool) Close(ctx context.Context) error {
	p.mu.Lock()
	if p.stopped {
		p.mu.Unlock()
		return nil
	}
	p.stopped = true
	p.mu.Unlock()

	p.cancel()
	close(p.tasks)
	p.wg.Wait()
	return nil
}
