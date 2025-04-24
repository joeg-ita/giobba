package entities

import (
	"time"
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
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Payload     map[string]interface{} `json:"payload"`
	Queue       string                 `json:"queue"`
	State       TaskState              `json:"state"`
	ETA         time.Time              `json:"eta"`
	CreatedAt   time.Time              `json:"created_at,omitempty"`
	UpdatedAt   time.Time              `json:"updated_at,omitempty"`
	StartedAt   time.Time              `json:"started_at,omitempty"`
	FinishedAt  time.Time              `json:"finished_at,omitempty"`
	ExpireAt    time.Time              `json:"expire_at,omitempty"`
	Timeout     time.Duration          `json:"timeout"`
	Error       string                 `json:"error"`
	Result      map[string]interface{} `json:"result,omitempty"`
	Retries     int                    `json:"retries"`
	MaxRetries  int                    `json:"max_retries,omitempty"`
	Priority    int                    `json:"priority"`
	Tags        []string               `json:"tags,omitempty"`
	SchedulerID string                 `json:"scheduler_id,omitempty"`
	WorkerID    string                 `json:"worker_id,omitempty"`
	ParentID    string                 `json:"parent_id"`
	ChildrenID  []string               `json:"children_id,omitempty"`
	StartMode   StartMode              `json:"startMode"`
	Callback    string                 `json:"callback"`
	CallbackErr string                 `json:"callback_err"`
}
