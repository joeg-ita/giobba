package test

import (
	"fmt"
	"testing"
	"time"

	"github.com/joeg-ita/giobba/src/domain"
)

func TestJob(t *testing.T) {
	SetupTest()
	defer TeardownTest()

	fmt.Println("TestJob...")

	now := time.Now()
	expiresAt := now.Add(90 * time.Second)
	queue := "background"

	payload := map[string]interface{}{
		"user": "sub_a",
		"job":  "process_subA",
	}
	scheduledTask, err := domain.NewScheduledTask("process", payload, queue, now, "*/1 * * * *", true, expiresAt, 9)
	if err != nil {
		t.Log(err.Error())
	}
	t.Log(scheduledTask)
	scheduledTaskId, _ := Scheduler.Tasker.AddTask(scheduledTask)

	tasks := []string{scheduledTaskId}
	result := make(map[string]domain.Task)

	for {
		isDone := false
		for _, tid := range tasks {
			task, _ := Scheduler.Tasker.Task(tid, queue)
			if task.State == "COMPLETED" && task.ExpiresAt.Before(time.Now()) {
				result[tid] = task
				isDone = true
				break
			} else if task.ExpiresAt.Before(time.Now()) {
				isDone = true
				break
			}
		}
		if isDone {
			break
		}
		time.Sleep(10 * time.Second)
	}
}
