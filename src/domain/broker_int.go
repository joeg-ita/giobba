package domain

import (
	"context"
	"time"
)

type BrokerInt interface {
	AddTask(ctx context.Context, task Task, queue string) (string, error)

	SaveTask(ctx context.Context, task Task, queue string) (string, error)

	GetTask(ctx context.Context, taskId string, queue string) (Task, error)

	DeleteTask(ctx context.Context, taskId string, queue string) error

	Schedule(ctx context.Context, task Task, queue string) error

	UnSchedule(ctx context.Context, taskId string, queue string, withWildcards bool) error

	GetScheduled(ctx context.Context, queue string) ([]string, error)

	Lock(ctx context.Context, taskId string, queue string, lockDuration time.Duration) bool

	RenewLock(ctx context.Context, taskId string, queue string, lockDuration time.Duration) bool

	UnLock(ctx context.Context, taskId string, queue string) error

	Subscribe(ctx context.Context, channels ...string) interface{}

	Publish(ctx context.Context, channel string, message ServiceMessage) error

	Close()
}
