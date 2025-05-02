package services

import (
	"context"

	"github.com/joeg-ita/giobba/src/entities"
)

type DatabaseInt interface {
	SaveTask(ctx context.Context, task entities.Task) (string, error)

	GetTask(ctx context.Context, taskId string) (entities.Task, error)

	GetTasks(ctx context.Context, query string, skip int, limit int, sort map[string]int) ([]entities.Task, error)

	Close(ctx context.Context)
}
