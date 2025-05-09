package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/joeg-ita/giobba/src/utils"
)

type Job struct {
	ID            string    `json:"id"`
	LastExecution time.Time `json:"last_execution,omitempty"`
	NextExecution time.Time `json:"next_execution"`
	Schedule      string    `json:"schedule"`
	TaskID        string    `json:"task_id"`
	TaskQueue     string    `json:"task_queue"`
	CreatedAt     time.Time `json:"created_at,omitempty"`
	UpdatedAt     time.Time `json:"updated_at,omitempty"`
	IsActive      bool      `json:"is_active,omitempty"`
}

func NewJob(schedule string, taskId string, taskQueue string, from time.Time, lastExecution time.Time, isActive bool) (Job, error) {
	id := uuid.New().String()

	var nextExecution time.Time
	if isActive {
		nextExecution = utils.CalculateNextExecution(schedule, from)
	}

	job := Job{
		ID:            id,
		LastExecution: lastExecution,
		NextExecution: nextExecution,
		Schedule:      schedule,
		TaskID:        taskId,
		TaskQueue:     taskQueue,
		CreatedAt:     time.Now(),
		IsActive:      isActive,
	}

	err := job.Validate()
	if err != nil {
		return Job{}, err
	}

	return job, nil
}

func (j *Job) Validate() error {
	return nil
}
