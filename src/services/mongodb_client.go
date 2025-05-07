package services

import (
	"context"
	"fmt"
	"log"

	"github.com/joeg-ita/giobba/src/config"
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
