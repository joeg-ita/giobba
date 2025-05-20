package services

import (
	"context"
	"time"

	"github.com/joeg-ita/giobba/src/domain"
)

type BrokerInt interface {
	AddTask(task domain.Task, queue string) (string, error)

	SaveTask(task domain.Task, queue string) (string, error)

	GetTask(taskId string, queue string) (domain.Task, error)

	DeleteTask(taskId string, queue string) error

	Schedule(task domain.Task, queue string) error

	UnSchedule(taskId string, queue string) error

	GetScheduled(queue string) ([]string, error)

	Lock(taskId string, queue string, lockDuration time.Duration) bool

	RenewLock(ctx context.Context, taskId string, queue string, lockDuration time.Duration) bool

	UnLock(taskId string, queue string) error

	Subscribe(context context.Context, channels ...string) interface{}

	Publish(context context.Context, channel string, message domain.ServiceMessage) error

	Close()
}
