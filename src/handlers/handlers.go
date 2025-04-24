package handlers

import (
	"context"
	"fmt"
	"giobba/src/entities"
	"giobba/src/services"
	"log"
	"time"
)

var Handlers = map[string]services.TaskHandlerInt{
	"process": &Process{},
}

type Process struct{}

func (t *Process) Run(ctx context.Context, task entities.Task) error {
	log.Printf("Processing task: %s", task.Name)

	for i := 0; i < 50; i++ {
		// Check for cancellation during the inner loop
		select {
		case <-ctx.Done():
			log.Println("Inner loop cancelled!")
			return fmt.Errorf("chiusa")
		default:
			// Continue with work
		}

		if i%4 == 0 {
			log.Printf("processing %v", i)
		}
		time.Sleep(time.Duration(1000) * time.Millisecond)
	}
	return nil
}
