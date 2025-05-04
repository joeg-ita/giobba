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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	scheduler := usecases.NewScheduler(ctx, brokerClient, cfg.Queues, cfg.WorkersNumber, cfg.LockDuration)

	dbClient, err := services.NewMongodbDatabase(cfg.Database)
	if err != nil {
		log.Printf("unable to load database client")
	} else {
		scheduler.AddDbClient(dbClient)
	}

	for name, handler := range handlers.Handlers {
		if utils.CheckInterface(handler) {
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
