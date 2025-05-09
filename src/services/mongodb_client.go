package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/joeg-ita/giobba/src/config"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongodbClient struct {
	Client   *mongo.Client
	Database *mongo.Database
}

func NewMongodbClient(cfg config.Database) (*MongodbClient, error) {
	Client, err := mongo.Connect(options.Client().ApplyURI(cfg.Url))
	if err != nil {
		log.Println("error starting mongodb client", err)
		return nil, err
	}

	Database := Client.Database(cfg.DB)

	colls := []string{cfg.TasksCollection, cfg.JobsCollection}

	for i := range colls {
		createWildcardTextIndexIfNotExists(Database.Collection(colls[i]))
	}

	return &MongodbClient{
		Client:   Client,
		Database: Database,
	}, nil
}

func (m *MongodbClient) GetClient() *mongo.Client {
	return m.Client
}

func (m *MongodbClient) Close(ctx context.Context) {
	if m.Client != nil {
		if err := m.Client.Disconnect(ctx); err != nil {
			// Log error but don't return it since this is a cleanup operation
			fmt.Printf("Error disconnecting MongoDB Client: %v\n", err)
		}
	}
}

func createWildcardTextIndexIfNotExists(collection *mongo.Collection) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Define the index name

	indexName := fmt.Sprintf("%v_text_index", collection.Name())

	// Check if the index already exists
	cursor, err := collection.Indexes().List(ctx)
	if err != nil {
		return fmt.Errorf("error listing indexes: %w", err)
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		return fmt.Errorf("error processing indexes: %w", err)
	}

	// Check if our index already exists
	indexExists := false
	for _, index := range results {
		if name, ok := index["name"].(string); ok && name == indexName {
			indexExists = true
			log.Printf("Index '%s' already exists\n", indexName)
			break
		}
	}

	// Create the index if it doesn't exist
	if !indexExists {
		// Create a text index on all fields using "$**"
		indexModel := mongo.IndexModel{
			Keys:    bson.D{{"$**", "text"}},
			Options: options.Index().SetName(indexName),
		}

		createdIndex, err := collection.Indexes().CreateOne(ctx, indexModel)
		if err != nil {
			return fmt.Errorf("error creating wildcard text index: %w", err)
		}
		log.Printf("Created new wildcard text index: %s\n", createdIndex)
	}

	return nil
}

func createTextIndexIfNotExists(
	collection *mongo.Collection,
	indexFields bson.D,
	indexName string,
	weights bson.D,
	language string,
) error {
	// err = createTextIndexIfNotExists(
	// 	collection,
	// 	bson.D{
	// 		{"title", "text"},
	// 		{"description", "text"},
	// 		{"tags", "text"},
	// 	},
	// 	"compound_text_index",
	// 	bson.D{
	// 		{"title", 10},
	// 		{"description", 5},
	// 		{"tags", 1},
	// 	},
	// 	"english",
	// )
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check if the index already exists
	cursor, err := collection.Indexes().List(ctx)
	if err != nil {
		return fmt.Errorf("error listing indexes: %w", err)
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		return fmt.Errorf("error processing indexes: %w", err)
	}

	// Check if our index already exists
	indexExists := false
	for _, index := range results {
		if name, ok := index["name"].(string); ok && name == indexName {
			indexExists = true
			log.Printf("Index '%s' already exists\n", indexName)
			break
		}
	}

	// Create the index if it doesn't exist
	if !indexExists {
		indexOptions := options.Index().SetName(indexName)

		// Add weights if provided
		if weights != nil {
			indexOptions.SetWeights(weights)
		}

		// Set language if provided
		if language != "" {
			indexOptions.SetDefaultLanguage(language)
		}

		indexModel := mongo.IndexModel{
			Keys:    indexFields,
			Options: indexOptions,
		}

		createdIndex, err := collection.Indexes().CreateOne(ctx, indexModel)
		if err != nil {
			return fmt.Errorf("error creating index: %w", err)
		}
		log.Printf("Created new index: %s\n", createdIndex)
	}

	return nil
}
