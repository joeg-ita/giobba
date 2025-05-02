package test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/joeg-ita/giobba"
	"github.com/joeg-ita/giobba/src/entities"
	"github.com/joeg-ita/giobba/src/services"
)

var scheduler services.Scheduler

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
	brokerClient := services.NewRedisBrokerByUrl(os.Getenv("GIOBBA_BROKER_URL"))
	scheduler = services.NewScheduler(context.Background(), brokerClient, []string{"default", "background"}, 1, 1)
	go giobba.Giobba()
}

// teardownTestSuite performs one-time cleanup after all tests have run
func teardownTestSuite() {
	fmt.Println("Cleaning up test environment...")
}

func TestMainTaskAndSubTasksAutoAndManual(t *testing.T) {

	fmt.Println("TestMainTaskAndSubTasksAutoAndManual...")

	queue := "default"
	payload_01 := map[string]interface{}{
		"user": "sub_a",
		"job":  "process_subA",
	}
	task_01 := entities.NewTask("process", payload_01, queue, time.Now(), 5, entities.AUTO, "")
	taskid, _ := scheduler.AddTask(task_01)

	payload_02 := map[string]interface{}{
		"user": "a",
		"job":  "process_A",
	}
	task_02 := entities.NewTask("process", payload_02, queue, time.Now(), 5, entities.AUTO, taskid)
	taskidAuto, _ := scheduler.AddTask(task_02)

	payload_03 := map[string]interface{}{
		"user": "sub_b",
		"job":  "process_subB",
	}
	task_03 := entities.NewTask("process", payload_03, queue, time.Now(), 5, entities.MANUAL, taskid)
	taskidManual, _ := scheduler.AddTask(task_03)

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

func TestTasksWithSameDatetimeDifferentPriorities(t *testing.T) {

	fmt.Println("TestTasksWithSameDatetimeDifferentPriorities...")

	now := time.Now().Add(5 * time.Second)
	queue := "background"
	payload := map[string]interface{}{
		"user": "sub_a",
		"job":  "process_subA",
	}
	task_p9 := entities.NewTask("process", payload, queue, now, 9, entities.AUTO, "")
	taskid_p9, _ := scheduler.AddTask(task_p9)

	task_p2 := entities.NewTask("process", payload, queue, now, 2, entities.AUTO, "")
	taskid_p2, _ := scheduler.AddTask(task_p2)

	task_p5 := entities.NewTask("process", payload, queue, now, 5, entities.AUTO, "")
	taskid_p5, _ := scheduler.AddTask(task_p5)

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

func TestTasksWithDifferenteDatetime(t *testing.T) {

	fmt.Println("TestTasksWithDifferenteDatetime...")

	base_now := time.Now()

	now := base_now.Add(20 * time.Second)
	queue := "background"

	payload := map[string]interface{}{
		"user": "sub_a",
		"job":  "process_subA",
	}
	task_p20 := entities.NewTask("process", payload, queue, now, 9, entities.AUTO, "")
	taskid_after_20_sec, _ := scheduler.AddTask(task_p20)

	now = base_now.Add(5 * time.Second)
	task_p5 := entities.NewTask("process", payload, queue, now, 5, entities.AUTO, "")
	taskid_after_5_sec, _ := scheduler.AddTask(task_p5)

	tasks := []string{taskid_after_20_sec, taskid_after_5_sec}
	result := make(map[string]entities.Task)

	for {
		for _, tid := range tasks {
			task, _ := scheduler.Task(tid, queue)
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
