package taskqueue

import "context"

// Task 是异步入库等场景提交的工作单元
type Task func(ctx context.Context) error

// Queue 抽象；生产可切 Redis，开发用内存 goroutine pool
type Queue interface {
	Enqueue(task Task) error
	Close(ctx context.Context) error
}
