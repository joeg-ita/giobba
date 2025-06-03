package domain

import (
	"context"
)

type JobRepositoryInt interface {
	Create(ctx context.Context, job Job) (string, error)

	Update(ctx context.Context, job Job) (string, error)

	Get(ctx context.Context, jobId string) (Job, error)

	List(ctx context.Context, query string, skip int, limit int, sort map[string]int) ([]Job, error)

	Delete(ctx context.Context, jobId string) (Job, error)

	Close(ctx context.Context)
}
