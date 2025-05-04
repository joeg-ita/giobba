package test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/joeg-ita/giobba"
	"github.com/joeg-ita/giobba/src/domain"
	"github.com/joeg-ita/giobba/src/services"
	"github.com/joeg-ita/giobba/src/usecases"
)

var scheduler usecases.Scheduler

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
	scheduler = usecases.NewScheduler(context.Background(), brokerClient, []string{"default", "background"}, 1, 1)
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
	task_01, _ := domain.NewTask("process", payload_01, queue, time.Now(), 5, domain.AUTO, "")
	taskid, _ := scheduler.TaskUtils.AddTask(task_01)

	payload_02 := map[string]interface{}{
		"user": "a",
		"job":  "process_A",
	}
	task_02, _ := domain.NewTask("process", payload_02, queue, time.Now(), 5, domain.AUTO, taskid)
	taskidAuto, _ := scheduler.TaskUtils.AddTask(task_02)

	payload_03 := map[string]interface{}{
		"user": "sub_b",
		"job":  "process_subB",
	}
	task_03, _ := domain.NewTask("process", payload_03, queue, time.Now(), 5, domain.MANUAL, taskid)
	taskidManual, _ := scheduler.TaskUtils.AddTask(task_03)

	for {
		state, _ := scheduler.TaskUtils.TaskState(taskid, "default")
		if state == "COMPLETED" {
			break
		}
		time.Sleep(2 * time.Second)
	}
	for {
		state, _ := scheduler.TaskUtils.TaskState(taskidAuto, "default")
		if state == "COMPLETED" {
			break
		}
		time.Sleep(2 * time.Second)
	}
	scheduler.TaskUtils.AutoTask(taskidManual, "default")
	for {
		state, _ := scheduler.TaskUtils.TaskState(taskidManual, "default")
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
	task_p9, _ := domain.NewTask("process", payload, queue, now, 9, domain.AUTO, "")
	taskid_p9, _ := scheduler.TaskUtils.AddTask(task_p9)

	task_p2, _ := domain.NewTask("process", payload, queue, now, 2, domain.AUTO, "")
	taskid_p2, _ := scheduler.TaskUtils.AddTask(task_p2)

	task_p5, _ := domain.NewTask("process", payload, queue, now, 5, domain.AUTO, "")
	taskid_p5, _ := scheduler.TaskUtils.AddTask(task_p5)

	tasks := []string{taskid_p2, taskid_p5, taskid_p9}
	result := make(map[string]domain.Task)

	for {
		for _, tid := range tasks {
			task, _ := scheduler.TaskUtils.Task(tid, queue)
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
	task_p20, _ := domain.NewTask("process", payload, queue, now, 9, domain.AUTO, "")
	taskid_after_20_sec, _ := scheduler.TaskUtils.AddTask(task_p20)

	now = base_now.Add(5 * time.Second)
	task_p5, _ := domain.NewTask("process", payload, queue, now, 5, domain.AUTO, "")
	taskid_after_5_sec, _ := scheduler.TaskUtils.AddTask(task_p5)

	tasks := []string{taskid_after_20_sec, taskid_after_5_sec}
	result := make(map[string]domain.Task)

	for {
		for _, tid := range tasks {
			task, _ := scheduler.TaskUtils.Task(tid, queue)
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
