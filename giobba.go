package giobba

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/joeg-ita/giobba/src/external/config"
	"github.com/joeg-ita/giobba/src/services"
	"github.com/joeg-ita/giobba/src/usecases"
)

func Giobba() {
	fmt.Println("Giobba")
	cfg, _ := config.LoadConfig()
	fmt.Println("Configuration", cfg.Broker.Url)

	brokerClient := services.NewRedisBrokerByUrl(cfg.Broker.Url)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	scheduler := services.NewScheduler(ctx, brokerClient, cfg.Queues[0], cfg.WorkersNumber, cfg.LockDuration)

	giobba := usecases.NewGiobbaStart(&scheduler)
	giobba.Run()

	// graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c
	fmt.Println("Shutting down...")
	cancel()
	scheduler.Stop()
	fmt.Println("Shutdown complete")

}
