package services

import (
	"context"

	"github.com/joeg-ita/giobba/src/domain"
)

type HandlerResult struct {
	Payload map[string]interface{}
	Err     error
}

type TaskHandlerInt interface {
	Run(ctx context.Context, task domain.Task) HandlerResult
}
