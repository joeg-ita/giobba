package giobba

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joeg-ita/giobba/src/config"
	"github.com/joeg-ita/giobba/src/handlers"
	"github.com/joeg-ita/giobba/src/services"
	"github.com/joeg-ita/giobba/src/usecases"
	"github.com/joeg-ita/giobba/src/utils"
)

func Giobba() {
	fmt.Println("Giobba")
	cfg, _ := config.LoadConfig()
	fmt.Println("Configuration", cfg.Broker.Url)

	brokerClient := services.NewRedisBrokerByUrl(cfg.Broker.Url)

	mongodbClient, err := services.NewMongodbClient(cfg.Database)
	if err != nil {
		log.Printf("unable to load database client")

	}
	mongodbJobs, err := services.NewMongodbJobs(mongodbClient, cfg.Database)
	mongodbTasks, err := services.NewMongodbTasks(mongodbClient, cfg.Database)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	scheduler := usecases.NewScheduler(ctx, brokerClient, mongodbTasks, mongodbJobs, cfg.Queues, cfg.WorkersNumber, cfg.LockDuration)

	for name, handler := range handlers.Handlers {
		if utils.CheckInterface[services.TaskHandlerInt](handler) {
			scheduler.RegisterHandler(name, handler)
		}
	}

	scheduler.Start()

	// graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c
	fmt.Println("Shutting down...")
	cancel()
	scheduler.Stop()
	fmt.Println("Shutdown complete")

}
