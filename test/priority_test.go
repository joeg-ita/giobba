package test

import (
	"fmt"
	"testing"
	"time"

	"github.com/joeg-ita/giobba/src/domain"
)

func TestTasksWithSameDatetimeDifferentPriorities(t *testing.T) {
	SetupTest()
	defer TeardownTest()

	fmt.Println("TestTasksWithSameDatetimeDifferentPriorities...")

	now := time.Now().Add(5 * time.Second)
	queue := "background"
	payload := map[string]interface{}{
		"user": "sub_a",
		"job":  "process_subA",
	}
	task_p9, _ := domain.NewTask("process", payload, queue, now, 9, domain.AUTO, "")
	taskid_p9, _ := Scheduler.Tasker.AddTask(task_p9)

	task_p2, _ := domain.NewTask("process", payload, queue, now, 2, domain.AUTO, "")
	taskid_p2, _ := Scheduler.Tasker.AddTask(task_p2)

	task_p5, _ := domain.NewTask("process", payload, queue, now, 5, domain.AUTO, "")
	taskid_p5, _ := Scheduler.Tasker.AddTask(task_p5)

	tasks := []string{taskid_p2, taskid_p5, taskid_p9}
	result := make(map[string]domain.Task)

	for {
		for _, tid := range tasks {
			task, _ := Scheduler.Tasker.Task(tid, queue)
			if task.State == "COMPLETED" {
				result[tid] = task
			}
			time.Sleep(2 * time.Second)
		}
		if len(result) == 3 {
			break
		}
	}

	if result[taskid_p9].StartedAt.After(result[taskid_p5].StartedAt) {
		t.Log("taskid_p9 eta", result[taskid_p9].ETA, "startedAt", result[taskid_p9].StartedAt, "priority", result[taskid_p9].Priority)
		t.Log("taskid_p5 eta", result[taskid_p5].ETA, "startedAt", result[taskid_p5].StartedAt, "priority", result[taskid_p5].Priority)
		t.Error("task_09 started after task_05")
	}
	if result[taskid_p5].StartedAt.After(result[taskid_p2].StartedAt) {
		t.Log("taskid_p5 eta", result[taskid_p5].ETA, "startedAt", result[taskid_p5].StartedAt, "priority", result[taskid_p5].Priority)
		t.Log("taskid_p2 eta", result[taskid_p2].ETA, "startedAt", result[taskid_p2].StartedAt, "priority", result[taskid_p2].Priority)
		t.Error("task_05 started after task_02")
	}
	if result[taskid_p9].StartedAt.After(result[taskid_p2].StartedAt) {
		t.Log("taskid_p9 eta", result[taskid_p9].ETA, "startedAt", result[taskid_p9].StartedAt, "priority", result[taskid_p9].Priority)
		t.Log("taskid_p2 eta", result[taskid_p2].ETA, "startedAt", result[taskid_p2].StartedAt, "priority", result[taskid_p2].Priority)
		t.Error("task_09 started after task_p2")
	}
}
