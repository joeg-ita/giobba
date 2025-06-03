package domain

import (
	"context"
	"time"
)

type TaskRepositoryInt interface {
	Create(ctx context.Context, task Task) (string, error)

	Update(ctx context.Context, task Task) (string, error)

	GetTask(ctx context.Context, taskId string) (Task, error)

	GetTasks(ctx context.Context, query string, skip int, limit int, sort map[string]int) ([]Task, error)

	GetStuckTasks(ctx context.Context, lockDuration time.Duration) ([]Task, error)

	Close(ctx context.Context)
}
