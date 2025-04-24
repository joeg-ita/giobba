package usecases

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/joeg-ita/giobba/src/handlers"
	"github.com/joeg-ita/giobba/src/services"
)

type GiobbaStart struct {
	Scheduler services.Scheduler
}

func NewGiobbaStart() *GiobbaStart {
	return &GiobbaStart{}
}

func (s *GiobbaStart) Run() {
	queueClient := services.NewRedisBrokerByUrl(os.Getenv("REDIS_URL"))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Scheduler = services.NewScheduler(ctx, queueClient, "default", 3, 180)

	for name, handler := range handlers.Handlers {
		s.Scheduler.RegisterHandler(name, handler)
	}

	s.Scheduler.Start()

	// Handle graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c
	fmt.Println("Shutting down...")
	cancel()
	s.Scheduler.Stop()
	fmt.Println("Shutdown complete")
}
