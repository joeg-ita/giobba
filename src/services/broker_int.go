package services

import (
	"context"
	"time"

	"github.com/joeg-ita/giobba/src/entities"
)

type BrokerInt interface {
	AddTask(task entities.Task, queue string) (string, error)

	SaveTask(task entities.Task, queue string) (string, error)

	GetTask(taskId string, queue string) (entities.Task, error)

	DeleteTask(taskId string, queue string) error

	Schedule(task entities.Task, queue string) error

	UnSchedule(taskId string, queue string) error

	GetScheduled(queue string) ([]string, error)

	Lock(taskId string, queue string, lockDuration time.Duration) bool

	UnLock(taskId string, queue string) error

	Subscribe(context context.Context, channels ...string) interface{}

	Publish(context context.Context, channel string, payload string) error

	Close()
}
