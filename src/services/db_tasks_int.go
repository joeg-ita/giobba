package services

import (
	"context"

	"github.com/joeg-ita/giobba/src/domain"
)

type DbTasksInt interface {
	SaveTask(ctx context.Context, task domain.Task) (string, error)

	GetTask(ctx context.Context, taskId string) (domain.Task, error)

	GetTasks(ctx context.Context, query string, skip int, limit int, sort map[string]int) ([]domain.Task, error)

	Close(ctx context.Context)
}
