package domain

import (
	"context"
)

type HandlerResult struct {
	Payload map[string]interface{}
	Err     error
}

type TaskHandlerInt interface {
	Run(ctx context.Context, task Task) HandlerResult
}
