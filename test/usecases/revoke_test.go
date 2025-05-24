package usecases

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/joeg-ita/giobba/src/domain"
	"github.com/joeg-ita/giobba/src/usecases"
)

func TestTaskRevoke(t *testing.T) {
	SetupTest()
	defer TeardownTest()

	fmt.Println("TestTaskRevoke...")

	now := time.Now().Add(2 * time.Minute)
	queue := "background"
	payload := map[string]interface{}{
		"user": "sub_a",
		"job":  "process_subA",
	}
	task_to_revoke, _ := domain.NewTask("process", payload, queue, now, 9, domain.AUTO, "")
	task_to_revoke_p9, _ := Scheduler.Tasker.AddTask(task_to_revoke)

	tasks := []string{task_to_revoke_p9}
	result := make(map[string]domain.Task)

	for {
		for _, tid := range tasks {
			task, _ := Scheduler.Tasker.Task(tid, queue)
			if task.State == domain.REVOKED {
				result[tid] = task
			} else {
				t.Logf("Task not yet Revoked")
			}
			if time.Now().After(now) {
				t.Errorf("Revoking error")
			}
			if time.Now().Add(30 * time.Second).After(task.CreatedAt) {
				message := domain.ServiceMessage{
					Action: domain.REVOKE,
					Payload: map[string]interface{}{
						"taskId":     task.ID,
						"queue":      task.Queue,
						"shedulerId": task.SchedulerID,
						"workerId":   task.WorkerID,
					},
				}
				err := BrokerClient.Publish(context.Background(), usecases.SERVICES_CHANNEL, message)
				if err != nil {
					t.Errorf("RevokeTask error %v", err)
				}
			}
			time.Sleep(2 * time.Second)
		}
		if len(result) == len(tasks) {
			break
		}
	}
}
