package services

import (
	"context"
	"fmt"

	"github.com/joeg-ita/giobba/src/config"
	"github.com/joeg-ita/giobba/src/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongodbJobs struct {
	client     DbClient[*mongo.Client]
	cfg        config.Database
	collection *mongo.Collection
}

func NewMongodbJobs(dbClient DbClient[*mongo.Client], cfg config.Database) (*MongodbJobs, error) {

	collection := dbClient.GetClient().Database(cfg.DB).Collection(cfg.JobsCollection)

	return &MongodbJobs{
		client:     dbClient,
		cfg:        cfg,
		collection: collection,
	}, nil
}

func (m *MongodbJobs) Save(ctx context.Context, job domain.Job) (string, error) {

	// Create filter to match by ID
	filter := bson.D{{Key: "_id", Value: job.ID}}

	// Create update with $set operator using the job struct directly
	update := bson.D{{Key: "$set", Value: job}}

	// Perform update operation
	result, err := m.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return "", err
	}

	// If no document was updated, insert a new one
	if result.MatchedCount == 0 {
		// Create a document with _id explicitly set
		doc := bson.D{
			{Key: "_id", Value: job.ID},
		}
		// Convert job to BSON document
		jobDoc, err := bson.Marshal(job)
		if err != nil {
			return "", err
		}
		var jobMap bson.M
		if err := bson.Unmarshal(jobDoc, &jobMap); err != nil {
			return "", err
		}
		// Add all other job fields
		for k, v := range jobMap {
			if k != "_id" { // Skip _id as we already set it
				doc = append(doc, bson.E{Key: k, Value: v})
			}
		}
		_, err = m.collection.InsertOne(ctx, doc)
		if err != nil {
			return "", err
		}
	}

	return job.ID, nil
}

func (m *MongodbJobs) Delete(ctx context.Context, jobId string) (domain.Job, error) {
	if jobId == "" {
		return domain.Job{}, fmt.Errorf("job ID cannot be empty")
	}

	// Create filter to match by ID
	filter := bson.D{{Key: "_id", Value: jobId}}

	// Retrieves the first matching document
	var job domain.Job
	err := m.collection.FindOne(ctx, filter).Decode(&job)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return domain.Job{}, fmt.Errorf("task not found: %s", jobId)
		}
		return domain.Job{}, fmt.Errorf("failed to get task: %w", err)
	}
	return job, nil
}

func (m *MongodbJobs) Retrieve(ctx context.Context, jobId string) (domain.Job, error) {
	if jobId == "" {
		return domain.Job{}, fmt.Errorf("job ID cannot be empty")
	}

	// Create filter to match by ID
	filter := bson.D{{Key: "_id", Value: jobId}}

	// Retrieves the first matching document
	var job domain.Job
	err := m.collection.FindOne(ctx, filter).Decode(&job)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return domain.Job{}, fmt.Errorf("job not found: %s", jobId)
		}
		return domain.Job{}, fmt.Errorf("failed to get job: %w", err)
	}
	return job, nil
}

func (m *MongodbJobs) RetrieveMany(ctx context.Context, query string, skip int, limit int, sort map[string]int) ([]domain.Job, error) {

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
		return nil, fmt.Errorf("failed to find jobs: %w", err)
	}
	defer cursor.Close(ctx)

	// Decode results into job slice
	var jobs []domain.Job
	if err = cursor.All(ctx, &jobs); err != nil {
		return nil, fmt.Errorf("failed to decode tasks: %w", err)
	}

	return jobs, nil
}

func (m *MongodbJobs) Close(ctx context.Context) {
	if m.client != nil {
		m.client.Close(ctx)
	}
}
