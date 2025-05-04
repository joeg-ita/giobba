package services

import (
	"context"

	"github.com/joeg-ita/giobba/src/domain"
)

type TaskHandlerInt interface {
	Run(ctx context.Context, task domain.Task) error
}
