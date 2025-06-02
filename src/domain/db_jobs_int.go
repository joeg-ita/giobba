package domain

import (
	"context"
)

type DbJobsInt interface {
	Save(ctx context.Context, job Job) (string, error)

	Retrieve(ctx context.Context, jobId string) (Job, error)

	RetrieveMany(ctx context.Context, query string, skip int, limit int, sort map[string]int) ([]Job, error)

	Delete(ctx context.Context, jobId string) (Job, error)

	Close(ctx context.Context)
}
