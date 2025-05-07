package handlers

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/joeg-ita/giobba/src/domain"
	"github.com/joeg-ita/giobba/src/services"
)

// BaseHandler provides a base implementation that users can embed in their custom handlers
type BaseHandler struct {
	Name string
}

// Run is the base implementation that users can override
func (h *BaseHandler) Run(ctx context.Context, task domain.Task) error {
	return fmt.Errorf("Run method must be implemented by custom handler")
}

// Helper function to create a new task
func NewTask(name string, payload map[string]interface{}, queue string, eta time.Time, priority int, mode domain.StartMode, parentId string) (domain.Task, error) {
	return domain.NewTask(name, payload, queue, eta, priority, mode, parentId)
}

// Helper function to create a task with default values
func NewDefaultTask(name string, payload map[string]interface{}, queue string) (domain.Task, error) {
	return domain.NewTask(
		name,
		payload,
		queue,
		time.Now(),
		5, // default priority
		domain.AUTO,
		"",
	)
}

// Helper function to create a child task
func NewChildTask(name string, payload map[string]interface{}, queue string, parentId string) (domain.Task, error) {
	return domain.NewTask(
		name,
		payload,
		queue,
		time.Now(),
		5, // default priority
		domain.AUTO,
		parentId,
	)
}

// Helper function to create a manual task
func NewManualTask(name string, payload map[string]interface{}, queue string) (domain.Task, error) {
	return domain.NewTask(
		name,
		payload,
		queue,
		time.Now(),
		5, // default priority
		domain.MANUAL,
		"",
	)
}

// Helper function to create a high priority task
func NewHighPriorityTask(name string, payload map[string]interface{}, queue string) (domain.Task, error) {
	return domain.NewTask(
		name,
		payload,
		queue,
		time.Now(),
		10, // high priority
		domain.AUTO,
		"",
	)
}

// Helper function to create a scheduled task
func NewScheduledTask(name string, payload map[string]interface{}, queue string, eta time.Time) (domain.Task, error) {
	return domain.NewScheduledTask(
		name,
		payload,
		queue,
		eta,
		"5 * * * *",
		true,
		5,
	)
}

// Example of a custom handler implementation
type Process struct {
	BaseHandler
}

func (t *Process) Run(ctx context.Context, task domain.Task) error {
	log.Printf("Processing task: %s", task.Name)

	for i := 0; i < 5; i++ {
		// Check for cancellation during the inner loop
		select {
		case <-ctx.Done():
			log.Println("Inner loop cancelled!")
			return fmt.Errorf("chiusa")
		default:
			// Continue with work
		}

		if i%4 == 0 {
			log.Printf("processing %v", i)
		}
		time.Sleep(time.Duration(1000) * time.Millisecond)
	}
	return nil
}

// Register all handlers
var Handlers = map[string]services.TaskHandlerInt{
	"process": &Process{},
}
