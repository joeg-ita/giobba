package services

import (
	"context"
	"testing"
	"time"

	"github.com/joeg-ita/giobba/src/entities"
	"github.com/joeg-ita/giobba/src/external/config"
	"github.com/joeg-ita/giobba/src/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB() (*services.MongodbDatabase, func()) {
	// Use a test database configuration
	cfg := config.Database{
		Url:        "mongodb://localhost:27017",
		DB:         "test_db",
		Collection: "test_tasks",
	}

	db, _ := services.NewMongodbDatabase(cfg)
	ctx := context.Background()

	// Cleanup function to drop the test collection and close the connection
	cleanup := func() {
		collection := db.Client.Database(cfg.DB).Collection(cfg.Collection)
		collection.Drop(ctx)
		db.Close(ctx)
	}

	return db, cleanup
}

func TestSaveTask(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()

	ctx := context.Background()

	t.Run("should save new task", func(t *testing.T) {
		task := entities.Task{
			ID:        "test-task-1",
			Name:      "Test Task",
			Queue:     "test-queue",
			State:     entities.PENDING,
			StartMode: entities.AUTO,
			ETA:       time.Now().Add(time.Hour),
			Priority:  5,
			Payload: map[string]interface{}{
				"test": "data",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		id, err := db.SaveTask(ctx, task)
		require.NoError(t, err)
		assert.Equal(t, task.ID, id)
		// Verify task was saved
		savedTask, err := db.GetTask(ctx, task.ID)
		require.NoError(t, err)
		assert.Equal(t, task.ID, savedTask.ID)
		assert.Equal(t, task.Name, savedTask.Name)
		assert.Equal(t, task.Queue, savedTask.Queue)
		assert.Equal(t, task.State, savedTask.State)
	})

	t.Run("should update existing task", func(t *testing.T) {
		task := entities.Task{
			ID:        "test-task-2",
			Name:      "Original Task",
			Queue:     "test-queue",
			State:     entities.PENDING,
			StartMode: entities.AUTO,
			ETA:       time.Now().Add(time.Hour),
			Priority:  5,
			Payload: map[string]interface{}{
				"test": "data",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Save initial task
		_, err := db.SaveTask(ctx, task)
		require.NoError(t, err)

		// Update task
		task.Name = "Updated Task"
		task.State = entities.RUNNING
		task.UpdatedAt = time.Now()

		id, err := db.SaveTask(ctx, task)
		require.NoError(t, err)
		assert.Equal(t, task.ID, id)

		// Verify task was updated
		savedTask, err := db.GetTask(ctx, task.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Task", savedTask.Name)
		assert.Equal(t, entities.RUNNING, savedTask.State)
	})
}

func TestGetTask(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()

	ctx := context.Background()

	t.Run("should return error for empty task ID", func(t *testing.T) {
		_, err := db.GetTask(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "task ID cannot be empty")
	})

	t.Run("should return error for non-existent task", func(t *testing.T) {
		_, err := db.GetTask(ctx, "non-existent-id")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "task not found")
	})

	t.Run("should return task when it exists", func(t *testing.T) {
		task := entities.Task{
			ID:        "test-task-3",
			Name:      "Test Task",
			Queue:     "test-queue",
			State:     entities.PENDING,
			StartMode: entities.AUTO,
			ETA:       time.Now().Add(time.Hour),
			Priority:  5,
			Payload: map[string]interface{}{
				"test": "data",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		_, err := db.SaveTask(ctx, task)
		require.NoError(t, err)

		savedTask, err := db.GetTask(ctx, task.ID)
		require.NoError(t, err)
		assert.Equal(t, task.ID, savedTask.ID)
		assert.Equal(t, task.Name, savedTask.Name)
		assert.Equal(t, task.Queue, savedTask.Queue)
		assert.Equal(t, task.State, savedTask.State)
	})
}

func TestGetTasks(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()

	ctx := context.Background()

	// Create test tasks
	tasks := []entities.Task{
		{
			ID:        "task-1",
			Name:      "First Task",
			Queue:     "test-queue",
			State:     entities.PENDING,
			StartMode: entities.AUTO,
			ETA:       time.Now().Add(time.Hour),
			Priority:  5,
			Payload: map[string]interface{}{
				"test": "data1",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "task-2",
			Name:      "Second Task",
			Queue:     "test-queue",
			State:     entities.PENDING,
			StartMode: entities.AUTO,
			ETA:       time.Now().Add(time.Hour),
			Priority:  5,
			Payload: map[string]interface{}{
				"test": "data2",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "task-3",
			Name:      "Third Task",
			Queue:     "test-queue",
			State:     entities.PENDING,
			StartMode: entities.AUTO,
			ETA:       time.Now().Add(time.Hour),
			Priority:  5,
			Payload: map[string]interface{}{
				"test": "data3",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	// Save all tasks
	for _, task := range tasks {
		_, err := db.SaveTask(ctx, task)
		require.NoError(t, err)
	}

	t.Run("should return all tasks with default parameters", func(t *testing.T) {
		result, err := db.GetTasks(ctx, "", 0, 10, nil)
		require.NoError(t, err)
		assert.Len(t, result, 3)
	})

	t.Run("should respect limit parameter", func(t *testing.T) {
		result, err := db.GetTasks(ctx, "", 0, 2, nil)
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("should respect skip parameter", func(t *testing.T) {
		result, err := db.GetTasks(ctx, "", 1, 10, nil)
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("should sort tasks by name", func(t *testing.T) {
		sort := map[string]int{"name": 1}
		result, err := db.GetTasks(ctx, "", 0, 10, sort)
		require.NoError(t, err)
		assert.Len(t, result, 3)
		assert.Equal(t, "First Task", result[0].Name)
		assert.Equal(t, "Second Task", result[1].Name)
		assert.Equal(t, "Third Task", result[2].Name)
	})
}
