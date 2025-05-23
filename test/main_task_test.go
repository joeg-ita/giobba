package test

import (
	"fmt"
	"testing"
	"time"

	"github.com/joeg-ita/giobba/src/domain"
)

func TestMainTaskAndSubTasksAutoAndManual(t *testing.T) {
	SetupTest()
	defer TeardownTest()

	fmt.Println("TestMainTaskAndSubTasksAutoAndManual...")

	queue := "default"
	payload_01 := map[string]interface{}{
		"user": "sub_a",
		"job":  "process_subA",
	}
	task_01, _ := domain.NewTask("process", payload_01, queue, time.Now(), 5, domain.AUTO, "")
	taskid, _ := Scheduler.Tasker.AddTask(task_01)

	payload_02 := map[string]interface{}{
		"user": "a",
		"job":  "process_A",
	}
	task_02, _ := domain.NewTask("process", payload_02, queue, time.Now(), 5, domain.AUTO, taskid)
	taskidAuto, _ := Scheduler.Tasker.AddTask(task_02)

	payload_03 := map[string]interface{}{
		"user": "sub_b",
		"job":  "process_subB",
	}
	task_03, _ := domain.NewTask("process", payload_03, queue, time.Now(), 5, domain.MANUAL, taskid)
	taskidManual, _ := Scheduler.Tasker.AddTask(task_03)

	for {
		state, _ := Scheduler.Tasker.TaskState(taskid, "default")
		if state == "COMPLETED" {
			break
		}
		time.Sleep(2 * time.Second)
	}
	for {
		state, _ := Scheduler.Tasker.TaskState(taskidAuto, "default")
		if state == "COMPLETED" {
			break
		}
		time.Sleep(2 * time.Second)
	}
	Scheduler.Tasker.AutoTask(taskidManual, "default")
	for {
		state, _ := Scheduler.Tasker.TaskState(taskidManual, "default")
		if state == "COMPLETED" {
			break
		}
		time.Sleep(2 * time.Second)
	}
}
