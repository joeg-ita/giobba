package services

import (
	"context"

	"github.com/joeg-ita/giobba/src/domain"
)

type DbJobsInt interface {
	Save(ctx context.Context, job domain.Job) (string, error)

	Retrieve(ctx context.Context, jobId string) (domain.Job, error)

	RetrieveMany(ctx context.Context, query string, skip int, limit int, sort map[string]int) ([]domain.Job, error)

	Delete(ctx context.Context, jobId string) (domain.Job, error)

	Close(ctx context.Context)
}
