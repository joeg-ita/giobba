package usecases

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/joeg-ita/giobba/src/domain"
	"github.com/joeg-ita/giobba/src/usecases"
)

func TestTaskKill(t *testing.T) {
	SetupTest()
	defer TeardownTest()

	fmt.Println("TestTaskKill...")

	now := time.Now()
	queue := "default"
	payload := map[string]interface{}{
		"user": "sub_a",
		"job":  "process_killable",
	}
	task_to_kill, _ := domain.NewTask("process", payload, queue, now, 9, domain.AUTO, "")
	task_to_killed_p9, _ := Scheduler.Tasker.AddTask(task_to_kill)

	tasks := []string{task_to_killed_p9}
	result := make(map[string]domain.Task)

	for {
		for _, tid := range tasks {
			task, _ := Scheduler.Tasker.Task(tid, queue)
			if task.State == domain.KILLED {
				result[tid] = task
			} else {
				t.Logf("Task not yet Killed")
			}

			if time.Now().Add(5*time.Second).After(task.CreatedAt) && task.State == domain.RUNNING {
				message := domain.ServiceMessage{
					Action: domain.KILL,
					Payload: map[string]interface{}{
						"taskId":     task.ID,
						"queue":      task.Queue,
						"shedulerId": task.SchedulerID,
						"workerId":   task.WorkerID,
					},
				}
				err := BrokerClient.Publish(context.Background(), usecases.SERVICES_CHANNEL, message)
				if err != nil {
					t.Errorf("KillTask error %v", err)
				}
			}
			time.Sleep(2 * time.Second)
		}
		if len(result) == len(tasks) {
			break
		}
	}
}
