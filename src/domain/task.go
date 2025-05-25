package domain

import (
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/joeg-ita/giobba/src/utils"
)

type TaskState string

const (
	PENDING   TaskState = "PENDING"
	RUNNING   TaskState = "RUNNING"
	COMPLETED TaskState = "COMPLETED"
	FAILED    TaskState = "FAILED"
	REVOKED   TaskState = "REVOKED"
	KILLED    TaskState = "KILLED"
)

type StartMode string

const (
	AUTO   StartMode = "AUTO"
	MANUAL StartMode = "MANUAL"
)

type Task struct {
	ID               string                 `json:"id" bson:"_id"`
	Name             string                 `json:"name" bson:"name"`
	Payload          map[string]interface{} `json:"payload" bson:"payload"`
	Queue            string                 `json:"queue" bson:"queue"`
	State            TaskState              `json:"state" bson:"state"`
	ETA              time.Time              `json:"eta" bson:"eta"`
	Priority         int                    `json:"priority" bson:"priority"`
	ParentID         string                 `json:"parent_id" bson:"parent_id"`
	StartMode        StartMode              `json:"start_mode" bson:"start_mode"`
	Schedule         string                 `json:"schedule,omitempty" bson:"start_modeschedule,omitempty"`
	IsScheduleActive bool                   `json:"is_schedule_active,omitempty" bson:"is_schedule_active,omitempty"`
	JobID            string                 `json:"job_id,omitempty" bson:"job_id,omitempty"`
	Error            string                 `json:"error,omitempty" bson:"error,omitempty"`
	CreatedAt        time.Time              `json:"created_at,omitempty" bson:"created_at,omitempty"`
	UpdatedAt        time.Time              `json:"updated_at,omitempty" bson:"updated_at,omitempty"`
	StartedAt        time.Time              `json:"started_at,omitempty" bson:"started_at,omitempty"`
	CompletedAt      time.Time              `json:"completed_at,omitempty" bson:"completed_at,omitempty"`
	ExpiresAt        time.Time              `json:"expires_at,omitempty" bson:"expires_at,omitempty"`
	Result           map[string]interface{} `json:"result,omitempty" bson:"result,omitempty"`
	Retries          int                    `json:"retries,omitempty" bson:"retries,omitempty"`
	MaxRetries       int                    `json:"max_retries,omitempty" bson:"max_retries,omitempty"`
	SchedulerID      string                 `json:"scheduler_id,omitempty" bson:"scheduler_id,omitempty"`
	WorkerID         string                 `json:"worker_id,omitempty" bson:"worker_id,omitempty"`
	Callback         string                 `json:"callback,omitempty" bson:"callback,omitempty"`
	CallbackErr      string                 `json:"callback_err,omitempty" bson:"callback_err,omitempty"`
}

func NewTask(name string, payload map[string]interface{}, queue string, eta time.Time, priority int, mode StartMode, parentId string) (Task, error) {
	id := uuid.New().String()
	if parentId == "" {
		parentId = id
	}
	timestamp := time.Now()

	task := Task{
		ID:        id,
		Name:      name,
		Payload:   payload,
		Queue:     queue,
		ETA:       eta,
		CreatedAt: timestamp,
		Priority:  priority,
		StartMode: mode,
		State:     PENDING,
		ParentID:  parentId,
	}

	err := task.Validate()
	if err != nil {
		return Task{}, err
	}

	return task, nil
}

func NewScheduledTask(name string, payload map[string]interface{}, queue string, eta time.Time, schedule string, isScheduleActive bool, expiresAt time.Time, priority int) (Task, error) {
	id := uuid.New().String()
	timestamp := time.Now()

	task := Task{
		ID:               id,
		Name:             name,
		Payload:          payload,
		Queue:            queue,
		ETA:              eta,
		CreatedAt:        timestamp,
		ExpiresAt:        expiresAt,
		Schedule:         schedule,
		IsScheduleActive: isScheduleActive,
		Priority:         priority,
		StartMode:        AUTO,
		State:            PENDING,
		ParentID:         id,
	}

	err := task.Validate()
	if err != nil {
		return Task{}, err
	}

	fmt.Println(task)

	return task, nil
}

func (t *Task) Validate() error {
	if t.ID == "" {
		return fmt.Errorf("task ID is required")
	}

	if !isValidUUID(t.ID) {
		return fmt.Errorf("invalid task ID format: must be a valid UUID")
	}

	if t.Name == "" {
		return fmt.Errorf("task name is required")
	}

	if t.Queue == "" {
		return fmt.Errorf("task queue is required")
	}

	if t.State == "" {
		return fmt.Errorf("task state is required")
	}

	// Validate State is one of the defined TaskStates
	validState := false
	for _, state := range []TaskState{PENDING, RUNNING, COMPLETED, FAILED, REVOKED, KILLED} {
		if t.State == state {
			validState = true
			break
		}
	}
	if !validState {
		return fmt.Errorf("invalid task state: %s", t.State)
	}

	// Validate StartMode
	if t.StartMode != AUTO && t.StartMode != MANUAL {
		return fmt.Errorf("invalid start mode: %s", t.StartMode)
	}

	// Validate Retries and MaxRetries
	if t.Retries < 0 {
		return fmt.Errorf("retries cannot be negative")
	}
	if t.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative")
	}
	if t.MaxRetries > 0 && t.Retries > t.MaxRetries {
		return fmt.Errorf("retries cannot be greater than max retries")
	}

	if t.Priority < 0 || t.Priority > 10 {
		return fmt.Errorf("priority must be a value in range [0,10]. higher value higher priority")
	}

	if t.ETA.IsZero() {
		return fmt.Errorf("eta datetime required")
	}

	if t.Schedule != "" {
		return utils.ParseCronSchedule(t.Schedule)
	}

	// Validate Callback URL if present
	if !isValidURL(t.Callback) {
		return fmt.Errorf("invalid callback URL format")
	}

	if !isValidURL(t.CallbackErr) {
		return fmt.Errorf("invalid callback error URL format")
	}

	if !isValidUUID(t.ParentID) {
		return fmt.Errorf("invalid parent ID format: must be a valid UUID")
	}

	return nil
}

// Add proper state transition validation
func (t *Task) ValidateStateTransition(newState TaskState) error {
	validTransitions := map[TaskState][]TaskState{
		PENDING:   {RUNNING, REVOKED},
		RUNNING:   {COMPLETED, FAILED, KILLED},
		COMPLETED: {PENDING}, // Only for scheduled tasks
		FAILED:    {PENDING}, // Only for retries
	}

	if transitions, ok := validTransitions[t.State]; ok {
		for _, validState := range transitions {
			if newState == validState {
				return nil
			}
		}
	}
	return fmt.Errorf("invalid state transition from %s to %s", t.State, newState)
}

func isValidURL(str string) bool {
	if str == "" {
		return true // Empty URL is considered valid (optional URL)
	}

	u, err := url.Parse(str)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func isValidUUID(str string) bool {
	if str == "" {
		return true
	}
	// Add UUID validation
	if _, err := uuid.Parse(str); err != nil {
		return false
	}
	return true
}
