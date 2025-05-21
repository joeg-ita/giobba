package test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/joeg-ita/giobba"
	"github.com/joeg-ita/giobba/src/config"
	"github.com/joeg-ita/giobba/src/domain"
	"github.com/joeg-ita/giobba/src/services"
	"github.com/joeg-ita/giobba/src/usecases"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var scheduler usecases.Scheduler
var mongodbClient services.DbClient[*mongo.Client]
var cfg *config.Config

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
	cfg, _ = config.ConfigFromYaml("giobba.yaml")
	brokerClient := services.NewRedisBrokerByUrl(cfg.Broker.Url)
	mongodbClient, _ = services.NewMongodbClient(cfg.Database)
	mongodbTasks, _ := services.NewMongodbTasks(mongodbClient, cfg.Database)
	mongodbJobs, _ := services.NewMongodbJobs(mongodbClient, cfg.Database)
	scheduler = usecases.NewScheduler(context.Background(), brokerClient, mongodbTasks, mongodbJobs, cfg)
	go giobba.Giobba()
}

// teardownTestSuite performs one-time cleanup after all tests have run
func teardownTestSuite() {
	fmt.Println("Cleaning up test environment...")
	mongodbClient.GetClient().Database(cfg.Database.DB).Drop(context.TODO())
	mongodbClient.Close(context.TODO())
}

func TestMainTaskAndSubTasksAutoAndManual(t *testing.T) {

	fmt.Println("TestMainTaskAndSubTasksAutoAndManual...")

	queue := "default"
	payload_01 := map[string]interface{}{
		"user": "sub_a",
		"job":  "process_subA",
	}
	task_01, _ := domain.NewTask("process", payload_01, queue, time.Now(), 5, domain.AUTO, "")
	taskid, _ := scheduler.Tasker.AddTask(task_01)

	payload_02 := map[string]interface{}{
		"user": "a",
		"job":  "process_A",
	}
	task_02, _ := domain.NewTask("process", payload_02, queue, time.Now(), 5, domain.AUTO, taskid)
	taskidAuto, _ := scheduler.Tasker.AddTask(task_02)

	payload_03 := map[string]interface{}{
		"user": "sub_b",
		"job":  "process_subB",
	}
	task_03, _ := domain.NewTask("process", payload_03, queue, time.Now(), 5, domain.MANUAL, taskid)
	taskidManual, _ := scheduler.Tasker.AddTask(task_03)

	for {
		state, _ := scheduler.Tasker.TaskState(taskid, "default")
		if state == "COMPLETED" {
			break
		}
		time.Sleep(2 * time.Second)
	}
	for {
		state, _ := scheduler.Tasker.TaskState(taskidAuto, "default")
		if state == "COMPLETED" {
			break
		}
		time.Sleep(2 * time.Second)
	}
	scheduler.Tasker.AutoTask(taskidManual, "default")
	for {
		state, _ := scheduler.Tasker.TaskState(taskidManual, "default")
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
	taskid_p9, _ := scheduler.Tasker.AddTask(task_p9)

	task_p2, _ := domain.NewTask("process", payload, queue, now, 2, domain.AUTO, "")
	taskid_p2, _ := scheduler.Tasker.AddTask(task_p2)

	task_p5, _ := domain.NewTask("process", payload, queue, now, 5, domain.AUTO, "")
	taskid_p5, _ := scheduler.Tasker.AddTask(task_p5)

	tasks := []string{taskid_p2, taskid_p5, taskid_p9}
	result := make(map[string]domain.Task)

	for {
		for _, tid := range tasks {
			task, _ := scheduler.Tasker.Task(tid, queue)
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
	taskid_after_20_sec, _ := scheduler.Tasker.AddTask(task_p20)

	now = base_now.Add(5 * time.Second)
	task_p5, _ := domain.NewTask("process", payload, queue, now, 5, domain.AUTO, "")
	taskid_after_5_sec, _ := scheduler.Tasker.AddTask(task_p5)

	tasks := []string{taskid_after_20_sec, taskid_after_5_sec}
	result := make(map[string]domain.Task)

	for {
		for _, tid := range tasks {
			task, _ := scheduler.Tasker.Task(tid, queue)
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

func TestJob(t *testing.T) {

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
	scheduledTaskId, _ := scheduler.Tasker.AddTask(scheduledTask)

	tasks := []string{scheduledTaskId}
	result := make(map[string]domain.Task)

	for {
		isDone := false
		for _, tid := range tasks {
			task, _ := scheduler.Tasker.Task(tid, queue)
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

func TestTaskRevoke(t *testing.T) {

	fmt.Println("TestTaskRevoke...")

	now := time.Now().Add(2 * time.Minute)
	queue := "background"
	payload := map[string]interface{}{
		"user": "sub_a",
		"job":  "process_subA",
	}
	task_to_revoke, _ := domain.NewTask("process", payload, queue, now, 9, domain.AUTO, "")
	task_to_revoke_p9, _ := scheduler.Tasker.AddTask(task_to_revoke)

	tasks := []string{task_to_revoke_p9}
	result := make(map[string]domain.Task)

	for {
		for _, tid := range tasks {
			task, _ := scheduler.Tasker.Task(tid, queue)
			if task.State == domain.REVOKED {
				result[tid] = task
			} else {
				t.Logf("Task not yet Revoked")
			}
			if time.Now().After(now) {
				t.Errorf("Revoking error")
			}
			if time.Now().Add(30 * time.Second).After(task.CreatedAt) {
				err := scheduler.Tasker.RevokeTask(task.ID, task.Queue)
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
