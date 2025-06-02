package usecases

import (
	"context"
	"fmt"
	"sync"

	"github.com/joeg-ita/giobba"
	"github.com/joeg-ita/giobba/src/config"
	"github.com/joeg-ita/giobba/src/domain"
	"github.com/joeg-ita/giobba/src/services"
	"github.com/joeg-ita/giobba/src/usecases"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Scheduler is the shared scheduler instance for all tests
var Scheduler usecases.Scheduler
var BrokerClient domain.BrokerInt

var (
	mongodbClient domain.DbClient[*mongo.Client]
	cfg           *config.Config
	setupOnce     sync.Once
)

// SetupTest ensures the test environment is set up exactly once
func SetupTest() {
	setupOnce.Do(func() {
		fmt.Println("Setting up test environment...")
		cfg, _ = config.ConfigFromYaml("giobba.yaml")
		BrokerClient = services.NewRedisBrokerByUrl(cfg.Broker.Url)
		mongodbClient, _ = services.NewMongodbClient(cfg.Database)
		mongodbTasks, _ := services.NewMongodbTasks(mongodbClient, cfg.Database)
		mongodbJobs, _ := services.NewMongodbJobs(mongodbClient, cfg.Database)
		Scheduler = usecases.NewScheduler(context.Background(), BrokerClient, mongodbTasks, mongodbJobs, cfg)
		go giobba.Giobba()
	})
}

// TeardownTest cleans up the test environment
func TeardownTest() {
	fmt.Println("Cleaning up test environment...")
	mongodbClient.GetClient().Database(cfg.Database.DB).Drop(context.TODO())
	mongodbClient.Close(context.TODO())
}
