package usecases

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/joeg-ita/giobba/src/domain"
	"github.com/joeg-ita/giobba/src/services"
)

type Tasks struct {
	brokerClient services.BrokerInt
	dbClient     services.DatabaseInt
	restClient   services.RestInt
	LockDuration time.Duration
}

func NewTaskUtils(brokerClient services.BrokerInt, dbClient services.DatabaseInt, restClient services.RestInt, lockDuration time.Duration) *Tasks {
	return &Tasks{
		brokerClient: brokerClient,
		dbClient:     dbClient,
		restClient:   restClient,
		LockDuration: lockDuration,
	}
}

// Update KillTask to properly handle cancellation
func (t *Tasks) KillTask(ctx context.Context, worker *Worker, taskID string, queue string) error {
	log.Printf("killing task %v", taskID)
	task, err := t.brokerClient.GetTask(taskID, queue)
	if err != nil {
		return fmt.Errorf("failed to find task %s: %v", taskID, err)
	}

	if task.State != domain.RUNNING {
		log.Printf("task %v not in %v state", taskID, domain.RUNNING)
		return nil
	}

	if worker == nil {
		return fmt.Errorf("worker for task %s not found", taskID)
	}

	// Cancel the worker context and create a new one
	worker.mutex.Lock()
	worker.cancel()

	// Create a new context for the worker
	newCtx, newCancel := context.WithCancel(ctx)
	worker.context = newCtx
	worker.cancel = newCancel
	worker.mutex.Unlock()

	t.Notify(context.Background(), task)

	log.Printf("Task %s successfully killed", taskID)
	return nil
}

func (t *Tasks) RevokeTask(taskID string, queue string) error {

	if t.brokerClient.Lock(taskID, queue, t.LockDuration) {
		log.Printf("revoking task %v", taskID)
		task, err := t.brokerClient.GetTask(taskID, queue)
		if err != nil {
			return fmt.Errorf("failed to find task %s: %v", taskID, err)
		}

		if task.State != domain.PENDING {
			log.Printf("task %v not in %v state", taskID, domain.PENDING)
			return nil
		}
		task.State = domain.REVOKED
		_, err = t.brokerClient.SaveTask(task, queue)
		t.brokerClient.UnLock(taskID, queue)

		if err != nil {
			return err
		}

		t.Notify(context.Background(), task)
	}
	log.Printf("task %s successfully revoked", taskID)
	return nil
}

func (t *Tasks) AutoTask(taskID string, queue string) error {

	if t.brokerClient.Lock(taskID, queue, t.LockDuration) {
		log.Printf("setting task %v to auto run", taskID)
		task, err := t.brokerClient.GetTask(taskID, queue)
		if err != nil {
			return fmt.Errorf("failed to find task %s: %v", taskID, err)
		}

		if task.State != domain.PENDING {
			log.Printf("task %v not in %v state", taskID, domain.PENDING)
			return nil
		}
		task.StartMode = domain.AUTO
		_, err = t.brokerClient.SaveTask(task, queue)
		t.brokerClient.UnLock(taskID, queue)

		if err != nil {
			return err
		}
		t.Notify(context.Background(), task)
	}
	log.Printf("task %s on queue %s successfully set to auto run", taskID, queue)
	return nil
}

func (t *Tasks) TaskState(taskID string, queue string) (domain.TaskState, error) {

	task, err := t.Task(taskID, queue)
	if err != nil {
		return "", fmt.Errorf("failed to find task %s: %v", taskID, err)
	}
	log.Printf("task %v state %v ", task.ID, task.State)
	return task.State, nil
}

func (t *Tasks) Task(taskID string, queue string) (domain.Task, error) {

	task, err := t.brokerClient.GetTask(taskID, queue)
	if err != nil {
		return domain.Task{}, fmt.Errorf("failed to find task %s: %v", taskID, err)
	}
	log.Printf("retrieved Task %v", task.ID)
	return task, nil
}

func (t *Tasks) AddTask(task domain.Task) (string, error) {

	log.Printf("adding task %v", task)

	err := task.Validate()
	if err != nil {
		log.Printf("task validation error %v", err)
		return "", err
	}

	taskId, err := t.brokerClient.AddTask(task, task.Queue)
	if err != nil {
		return "", err
	}
	err = t.brokerClient.Schedule(task, task.Queue+QUEUE_SCHEDULE_POSTFIX)
	if err != nil {
		return "", err
	}
	t.Notify(context.Background(), task)
	return taskId, nil
}

func (t *Tasks) Callback(url string, payload map[string]interface{}) {
	err := t.restClient.Post(url, payload)
	if err != nil {
		log.Println(err)
	}
}

func (t *Tasks) Notify(ctx context.Context, task domain.Task) {

	t.brokerClient.Publish(ctx, ACTIVITIES_CHANNEL, map[string]interface{}{
		"workerId": task.WorkerID,
		"task":     task,
	})

	if t.dbClient == nil {
		return
	} else {
		_, err := t.dbClient.SaveTask(ctx, task)
		if err != nil {
			log.Printf("error sending task to database")
		}
	}

}
