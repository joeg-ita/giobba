package domain

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/joeg-ita/giobba/src/domain"
)

func TestJobValidation(t *testing.T) {
	tests := []struct {
		name        string
		job         domain.Job
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid job",
			job: domain.Job{
				ID:            uuid.New().String(),
				Schedule:      "*/5 * * * *",
				TaskID:        "task-123",
				TaskQueue:     "default",
				NextExecution: time.Now(),
				LastExecution: time.Time{},
				IsActive:      true,
			},
			expectError: false,
		},
		{
			name: "missing schedule",
			job: domain.Job{
				ID:            uuid.New().String(),
				Schedule:      "",
				TaskID:        "task-123",
				TaskQueue:     "default",
				NextExecution: time.Now(),
				LastExecution: time.Time{},
				IsActive:      true,
			},
			expectError: true,
			errorMsg:    "schedule is required",
		},
		{
			name: "invalid cron format - too few fields",
			job: domain.Job{
				ID:            uuid.New().String(),
				Schedule:      "* * * *",
				TaskID:        "task-123",
				TaskQueue:     "default",
				NextExecution: time.Now(),
				LastExecution: time.Time{},
				IsActive:      true,
			},
			expectError: true,
			errorMsg:    "missing field(s)",
		},
		{
			name: "invalid cron format - invalid minute",
			job: domain.Job{
				ID:            uuid.New().String(),
				Schedule:      "70 * * * *",
				TaskID:        "task-123",
				TaskQueue:     "default",
				NextExecution: time.Now(),
				LastExecution: time.Time{},
				IsActive:      true,
			},
			expectError: true,
			errorMsg:    "syntax error in minute field: '70'",
		},
		{
			name: "missing task queue",
			job: domain.Job{
				ID:            uuid.New().String(),
				Schedule:      "*/5 * * * *",
				TaskID:        "task-123",
				TaskQueue:     "",
				NextExecution: time.Now(),
				LastExecution: time.Time{},
				IsActive:      true,
			},
			expectError: true,
			errorMsg:    "task_queue is required",
		},
		{
			name: "inactive job without next execution",
			job: domain.Job{
				ID:            uuid.New().String(),
				Schedule:      "*/5 * * * *",
				TaskID:        "task-123",
				TaskQueue:     "default",
				NextExecution: time.Now(),
				LastExecution: time.Time{},
				IsActive:      false,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			job, err := domain.NewJob(tt.job.Schedule, tt.job.TaskID, tt.job.TaskQueue, tt.job.NextExecution, tt.job.LastExecution, tt.job.IsActive)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error message containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if job.Schedule != tt.job.Schedule {
					t.Errorf("expected schedule %q, got %q", tt.job.Schedule, job.Schedule)
				}
				if job.TaskID != tt.job.TaskID {
					t.Errorf("expected task ID %q, got %q", tt.job.TaskID, job.TaskID)
				}
				if job.TaskQueue != tt.job.TaskQueue {
					t.Errorf("expected task queue %q, got %q", tt.job.TaskQueue, job.TaskQueue)
				}
				if job.IsActive != tt.job.IsActive {
					t.Errorf("expected is_active %v, got %v", tt.job.IsActive, job.IsActive)
				}
			}
		})
	}
}
