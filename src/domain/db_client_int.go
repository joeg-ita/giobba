package domain

import (
	"context"
)

type DbClient[T any] interface {
	GetClient() T

	Close(ctx context.Context)
}
