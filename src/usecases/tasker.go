package usecases

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/joeg-ita/giobba/src/domain"
	"github.com/joeg-ita/giobba/src/services"
)

type Tasker struct {
	brokerClient  services.BrokerInt
	dbTasksClient services.DbTasksInt
	dbJobsClient  services.DbJobsInt
	restClient    services.RestInt
	LockDuration  time.Duration
}

func NewTaskUtils(brokerClient services.BrokerInt, dbTasksClient services.DbTasksInt, dbJobsClient services.DbJobsInt, restClient services.RestInt, lockDuration time.Duration) *Tasker {
	return &Tasker{
		brokerClient:  brokerClient,
		dbTasksClient: dbTasksClient,
		dbJobsClient:  dbJobsClient,
		restClient:    restClient,
		LockDuration:  lockDuration,
	}
}

func (t *Tasker) AddTask(task domain.Task) (string, error) {

	log.Printf("adding task %v", task)

	err := task.Validate()
	if err != nil {
		log.Printf("task validation error %v", err)
		return "", err
	}

	if task.Schedule != "" {
		job, err := domain.NewJob(task.Schedule, task.ID, task.Queue, task.ETA, task.ExpiresAt, true)
		if err != nil {
			return "", err
		}
		err = t.AddJob(job)
		if err != nil {
			return "", err
		}
		task.JobID = job.ID
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

func (t *Tasker) AddJob(job domain.Job) error {
	_, err := t.dbJobsClient.Save(context.Background(), job)
	return err
}

// Update KillTask to properly handle cancellation
func (t *Tasker) KillTask(ctx context.Context, worker *Worker, taskID string, queue string) error {
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

func (t *Tasker) RevokeTask(taskID string, queue string) error {

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
		t.brokerClient.UnSchedule(fmt.Sprintf("%v::%v::*", task.ID, task.Queue+QUEUE_SCHEDULE_POSTFIX), task.Queue+QUEUE_SCHEDULE_POSTFIX, true)
		t.brokerClient.UnLock(task.ID, task.Queue)

		if err != nil {
			return err
		}

		t.Notify(context.Background(), task)
	}
	log.Printf("task %s successfully revoked", taskID)
	return nil
}

func (t *Tasker) AutoTask(taskID string, queue string) error {

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

func (t *Tasker) TaskState(taskID string, queue string) (domain.TaskState, error) {

	task, err := t.Task(taskID, queue)
	if err != nil {
		return "", fmt.Errorf("failed to find task %s: %v", taskID, err)
	}
	log.Printf("task %v state %v ", task.ID, task.State)
	return task.State, nil
}

func (t *Tasker) Task(taskID string, queue string) (domain.Task, error) {

	task, err := t.brokerClient.GetTask(taskID, queue)
	if err != nil {
		return domain.Task{}, fmt.Errorf("failed to find task %s: %v", taskID, err)
	}
	log.Printf("retrieved Task %v", task.ID)
	return task, nil
}

func (t *Tasker) Callback(url string, payload map[string]interface{}) {
	err := t.restClient.Post(url, payload)
	if err != nil {
		log.Println(err)
	}
}

func (t *Tasker) Notify(ctx context.Context, task domain.Task) {

	serviceMessage := domain.ServiceMessage{
		Action: "ACTIVITY",
		Payload: map[string]interface{}{
			"workerId": task.WorkerID,
			"task":     task,
		},
	}

	err := t.brokerClient.Publish(ctx, ACTIVITIES_CHANNEL, serviceMessage)
	if err != nil {
		log.Printf("error publishing task to broker %v", err)
	}

	if t.dbTasksClient == nil {
		return
	} else {
		_, err := t.dbTasksClient.SaveTask(ctx, task)
		if err != nil {
			log.Printf("error sending task to database")
		}
	}

}
