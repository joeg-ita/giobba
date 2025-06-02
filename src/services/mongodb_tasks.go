package services

import (
	"context"
	"fmt"
	"time"

	"github.com/joeg-ita/giobba/src/config"
	"github.com/joeg-ita/giobba/src/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongodbTasks struct {
	client     domain.DbClient[*mongo.Client]
	cfg        config.Database
	collection *mongo.Collection
}

func NewMongodbTasks(dbClient domain.DbClient[*mongo.Client], cfg config.Database) (*MongodbTasks, error) {
	collection := dbClient.GetClient().Database(cfg.DB).Collection(cfg.TasksCollection)

	return &MongodbTasks{
		client:     dbClient,
		cfg:        cfg,
		collection: collection,
	}, nil
}

func (m *MongodbTasks) SaveTask(ctx context.Context, task domain.Task) (string, error) {
	// Create filter to match by ID
	filter := bson.D{{Key: "_id", Value: task.ID}}

	// Create update with $set operator using the task struct directly
	update := bson.D{{Key: "$set", Value: task}}

	// Perform update operation
	result, err := m.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return "", err
	}

	// If no document was updated, insert a new one
	if result.MatchedCount == 0 {
		// Convert task to BSON document
		taskDoc, err := bson.Marshal(task)
		if err != nil {
			return "", err
		}
		var taskMap bson.M
		if err := bson.Unmarshal(taskDoc, &taskMap); err != nil {
			return "", err
		}

		// Ensure _id is set correctly
		taskMap["_id"] = task.ID
		delete(taskMap, "id") // Remove the "id" field if it exists

		_, err = m.collection.InsertOne(ctx, taskMap)
		if err != nil {
			return "", err
		}
	}

	return task.ID, nil
}

func (m *MongodbTasks) GetTask(ctx context.Context, taskId string) (domain.Task, error) {
	if taskId == "" {
		return domain.Task{}, fmt.Errorf("task ID cannot be empty")
	}

	// Create filter to match by ID
	filter := bson.D{{Key: "_id", Value: taskId}}

	// Retrieves the first matching document
	var task domain.Task
	err := m.collection.FindOne(ctx, filter).Decode(&task)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return domain.Task{}, fmt.Errorf("task not found: %s", taskId)
		}
		return domain.Task{}, fmt.Errorf("failed to get task: %w", err)
	}
	return task, nil
}

func (m *MongodbTasks) GetTasks(ctx context.Context, query string, skip int, limit int, sort map[string]int) ([]domain.Task, error) {

	// Create filter based on query string
	var filter bson.M
	if query != "" {
		filter = bson.M{"$text": bson.M{"$search": query}}
	} else {
		filter = bson.M{}
	}

	// Create find options for pagination and sorting
	findOptions := options.Find()
	findOptions.SetSkip(int64(skip))
	findOptions.SetLimit(int64(limit))

	// Add sorting if specified
	if len(sort) > 0 {
		sortDoc := bson.M{}
		for field, direction := range sort {
			sortDoc[field] = direction
		}
		findOptions.SetSort(sortDoc)
	}

	// Execute find operation
	cursor, err := m.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to find tasks: %w", err)
	}
	defer cursor.Close(ctx)

	// Decode results into tasks slice
	var tasks []domain.Task
	if err = cursor.All(ctx, &tasks); err != nil {
		return nil, fmt.Errorf("failed to decode tasks: %w", err)
	}

	return tasks, nil
}

func (m *MongodbTasks) GetStuckTasks(ctx context.Context, lockDuration time.Duration) ([]domain.Task, error) {
	// Calculate the cutoff time for stuck tasks
	cutoffTime := time.Now().Add(-lockDuration)

	// Create filter to find RUNNING tasks that started before the cutoff time
	filter := bson.M{
		"state": domain.RUNNING,
		"started_at": bson.M{
			"$lt": cutoffTime,
		},
	}

	// Execute find operation
	cursor, err := m.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find stuck tasks: %w", err)
	}
	defer cursor.Close(ctx)

	// Decode results into tasks slice
	var tasks []domain.Task
	if err = cursor.All(ctx, &tasks); err != nil {
		return nil, fmt.Errorf("failed to decode stuck tasks: %w", err)
	}

	return tasks, nil
}

func (m *MongodbTasks) Close(ctx context.Context) {
	if m.client != nil {
		m.client.Close(ctx)
	}
}
