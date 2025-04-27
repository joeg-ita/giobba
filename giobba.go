package giobba

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/joeg-ita/giobba/src/entities"
	"github.com/joeg-ita/giobba/src/services"
	"github.com/joeg-ita/giobba/src/usecases"
)

func Giobba() {
	fmt.Println("Giobba")

	giobba := usecases.NewGiobbaStart()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		giobba.Run()
		wg.Done()
	}()

	go func() {
		taskAddSubtask(&giobba.Scheduler)
		wg.Done()
	}()

	wg.Wait()

}

func taskAddSubtask(scheduler *services.Scheduler) {
	taskid, _ := scheduler.AddTask(entities.Task{
		ID:    uuid.NewString(),
		Name:  "process",
		Queue: "default",
		Payload: map[string]interface{}{
			"user": "a",
			"job":  "process_A",
		},
		StartMode: entities.AUTO,
	})

	scheduler.AddTask(entities.Task{
		ID:    uuid.NewString(),
		Name:  "process",
		Queue: "default",
		Payload: map[string]interface{}{
			"user": "sub_a",
			"job":  "process_subA",
		},
		StartMode: entities.AUTO,
		ParentID:  taskid,
	})

	scheduler.AddTask(entities.Task{
		ID:    uuid.NewString(),
		Name:  "process",
		Queue: "default",
		Payload: map[string]interface{}{
			"user": "sub_b",
			"job":  "process_subB",
		},
		StartMode: entities.MANUAL,
		ParentID:  taskid,
	})
}

func taskAdd(scheduler *services.Scheduler) {
	_, err := scheduler.AddTask(entities.Task{
		ID:    uuid.NewString(),
		Name:  "process",
		Queue: "default",
		Payload: map[string]interface{}{
			"user": "tizio",
			"job":  "process",
		},
		StartMode: entities.AUTO,
	})

	if err != nil {
		log.Panic("TASKKKK")
	}

	time.Sleep(time.Duration(20) * time.Second)

	_, err = scheduler.AddTask(entities.Task{
		ID:    uuid.NewString(),
		Name:  "process",
		Queue: "default",
		Payload: map[string]interface{}{
			"user": "tizio2",
			"job":  "process2",
		},
		StartMode: entities.AUTO,
	})
}
