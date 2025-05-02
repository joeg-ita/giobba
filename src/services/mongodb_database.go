package services

import (
	"context"
	"fmt"

	"github.com/joeg-ita/giobba/src/entities"
	"github.com/joeg-ita/giobba/src/external/config"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongodbDatabase struct {
	client     *mongo.Client
	database   string
	collection string
}

func NewMongodbDatabase(cfg config.Database) MongodbDatabase {
	client, err := mongo.Connect(options.Client().ApplyURI(cfg.Url))
	if err != nil {
		panic(err)
	}

	return MongodbDatabase{
		client:     client,
		database:   cfg.DB,
		collection: cfg.Collection,
	}
}

func (m *MongodbDatabase) SaveTask(ctx context.Context, task entities.Task) (string, error) {
	collection := m.client.Database(m.database).Collection(m.collection)

	// Convert task to BSON document
	doc, err := bson.Marshal(task)
	if err != nil {
		return "", err
	}

	// Create filter to match by ID
	filter := bson.D{{Key: "_id", Value: task.ID}}

	// Create update with $set operator
	update := bson.D{{Key: "$set", Value: doc}}

	// Perform update operation
	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return "", err
	}

	// If no document was updated, insert a new one
	if result.MatchedCount == 0 {
		_, err = collection.InsertOne(ctx, doc)
		if err != nil {
			return "", err
		}
	}

	return task.ID, nil
}

func (m *MongodbDatabase) GetTask(ctx context.Context, taskId string) (entities.Task, error) {
	if taskId == "" {
		return entities.Task{}, fmt.Errorf("task ID cannot be empty")
	}

	collection := m.client.Database(m.database).Collection(m.collection)

	// Create filter to match by ID
	filter := bson.D{{Key: "_id", Value: taskId}}

	// Retrieves the first matching document
	var task entities.Task
	err := collection.FindOne(ctx, filter).Decode(&task)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return entities.Task{}, fmt.Errorf("task not found: %s", taskId)
		}
		return entities.Task{}, fmt.Errorf("failed to get task: %w", err)
	}
	return task, nil
}

func (m *MongodbDatabase) GetTasks(ctx context.Context, query string, skip int, limit int, sort map[string]int) ([]entities.Task, error) {
	collection := m.client.Database(m.database).Collection(m.collection)

	// Create filter based on query string
	var filter bson.D
	if query != "" {
		filter = bson.D{{Key: "$text", Value: bson.D{{Key: "$search", Value: query}}}}
	}

	// Create find options for pagination and sorting
	findOptions := options.Find()
	findOptions.SetSkip(int64(skip))
	findOptions.SetLimit(int64(limit))

	// Add sorting if specified
	if len(sort) > 0 {
		sortDoc := bson.D{}
		for field, direction := range sort {
			sortDoc = append(sortDoc, bson.E{Key: field, Value: direction})
		}
		findOptions.SetSort(sortDoc)
	}

	// Execute find operation
	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to find tasks: %w", err)
	}
	defer cursor.Close(ctx)

	// Decode results into tasks slice
	var tasks []entities.Task
	if err = cursor.All(ctx, &tasks); err != nil {
		return nil, fmt.Errorf("failed to decode tasks: %w", err)
	}

	return tasks, nil
}

func (m *MongodbDatabase) Close(ctx context.Context) {
	if m.client != nil {
		if err := m.client.Disconnect(ctx); err != nil {
			// Log error but don't return it since this is a cleanup operation
			fmt.Printf("Error disconnecting MongoDB client: %v\n", err)
		}
	}
}
