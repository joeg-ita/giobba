package test

import (
	"fmt"
	"testing"
	"time"

	"github.com/joeg-ita/giobba/src/domain"
)

func TestTasksWithDifferenteDatetime(t *testing.T) {
	SetupTest()
	defer TeardownTest()

	fmt.Println("TestTasksWithDifferenteDatetime...")

	base_now := time.Now()

	now := base_now.Add(20 * time.Second)
	queue := "background"

	payload := map[string]interface{}{
		"user": "sub_a",
		"job":  "process_subA",
	}
	task_p20, _ := domain.NewTask("process", payload, queue, now, 9, domain.AUTO, "")
	taskid_after_20_sec, _ := Scheduler.Tasker.AddTask(task_p20)

	now = base_now.Add(5 * time.Second)
	task_p5, _ := domain.NewTask("process", payload, queue, now, 5, domain.AUTO, "")
	taskid_after_5_sec, _ := Scheduler.Tasker.AddTask(task_p5)

	tasks := []string{taskid_after_20_sec, taskid_after_5_sec}
	result := make(map[string]domain.Task)

	for {
		for _, tid := range tasks {
			task, _ := Scheduler.Tasker.Task(tid, queue)
			if task.State == "COMPLETED" {
				result[tid] = task
			}
			time.Sleep(2 * time.Second)
		}
		if len(result) == 2 {
			break
		}
	}

	if result[taskid_after_20_sec].StartedAt.Before(result[taskid_after_5_sec].StartedAt) {
		t.Error("task_09 finished after task_05")
	}
}
