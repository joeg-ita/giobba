package test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/joeg-ita/giobba/src/entities"
	"github.com/joeg-ita/giobba/src/usecases"
)

var giobba *usecases.GiobbaStart

func TestMain(m *testing.M) {
	// Global setup before any tests run
	fmt.Println("Global test suite setup - runs once before all tests")
	setupTestSuite()

	// Run all the tests in the package
	exitCode := m.Run()

	// Global teardown after all tests have run
	fmt.Println("Global test suite teardown - runs once after all tests")
	teardownTestSuite()

	// Exit with the status from the test run
	os.Exit(exitCode)
}

// setupTestSuite performs one-time setup for the entire test suite
func setupTestSuite() {
	fmt.Println("Setting up test environment...")
	giobba = usecases.NewGiobbaStart()
	go giobba.Run()
}

// teardownTestSuite performs one-time cleanup after all tests have run
func teardownTestSuite() {
	fmt.Println("Cleaning up test environment...")
}

func TestMainTaskAndSubTasksAutoAndManual(t *testing.T) {

	fmt.Println("TestMainTaskAndSubTasksAutoAndManual...")

	scheduler := &giobba.Scheduler

	taskid, _ := scheduler.AddTask(entities.Task{
		ID:    uuid.NewString(),
		Name:  "process",
		Queue: "default",
		Payload: map[string]interface{}{
			"user": "a",
			"job":  "process_A",
		},
		StartMode: entities.AUTO,
	})

	taskidAuto, _ := scheduler.AddTask(entities.Task{
		ID:    uuid.NewString(),
		Name:  "process",
		Queue: "default",
		Payload: map[string]interface{}{
			"user": "sub_a",
			"job":  "process_subA",
		},
		StartMode: entities.AUTO,
		ParentID:  taskid,
	})

	taskidManual, _ := scheduler.AddTask(entities.Task{
		ID:    uuid.NewString(),
		Name:  "process",
		Queue: "default",
		Payload: map[string]interface{}{
			"user": "sub_b",
			"job":  "process_subB",
		},
		StartMode: entities.MANUAL,
		ParentID:  taskid,
	})

	for {
		state, _ := scheduler.TaskState(taskid, "default")
		if state == "COMPLETED" {
			break
		}
		time.Sleep(2 * time.Second)
	}
	for {
		state, _ := scheduler.TaskState(taskidAuto, "default")
		if state == "COMPLETED" {
			break
		}
		time.Sleep(2 * time.Second)
	}
	scheduler.AutoTask(taskidManual, "default")
	for {
		state, _ := scheduler.TaskState(taskidManual, "default")
		if state == "COMPLETED" {
			break
		}
		time.Sleep(2 * time.Second)
	}

}

func TestTasksWithDifferentPriorities(t *testing.T) {

	fmt.Println("TestTasksWithDifferentPriorities...")

	scheduler := &giobba.Scheduler

	now := time.Now().Add(5 * time.Second)
	queue := "default"
	taskid_p9, _ := scheduler.AddTask(entities.Task{
		ID:    uuid.NewString(),
		Name:  "process",
		Queue: queue,
		Payload: map[string]interface{}{
			"user": "tizio",
			"job":  "process",
		},
		ETA:       now,
		Priority:  9,
		StartMode: entities.AUTO,
	})

	taskid_p2, _ := scheduler.AddTask(entities.Task{
		ID:    uuid.NewString(),
		Name:  "process",
		Queue: queue,
		Payload: map[string]interface{}{
			"user": "tizio",
			"job":  "process",
		},
		ETA:       now,
		Priority:  2,
		StartMode: entities.AUTO,
	})

	taskid_p5, _ := scheduler.AddTask(entities.Task{
		ID:    uuid.NewString(),
		Name:  "process",
		Queue: queue,
		Payload: map[string]interface{}{
			"user": "tizio",
			"job":  "process",
		},
		ETA:       now,
		Priority:  5,
		StartMode: entities.AUTO,
	})

	tasks := []string{taskid_p2, taskid_p5, taskid_p9}
	result := make(map[string]entities.Task)

	for {
		for _, tid := range tasks {
			task, _ := scheduler.Task(tid, queue)
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
		t.Error("task_09 finished after task_05")
	}
	if result[taskid_p5].StartedAt.After(result[taskid_p2].StartedAt) {
		t.Error("task_05 finished after task_02")
	}

}
