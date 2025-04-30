package usecases

import (
	"github.com/joeg-ita/giobba/src/handlers"
	"github.com/joeg-ita/giobba/src/services"
)

type GiobbaStart struct {
	Scheduler *services.Scheduler
}

func NewGiobbaStart(scheduler *services.Scheduler) *GiobbaStart {
	return &GiobbaStart{
		Scheduler: scheduler,
	}
}

func (s *GiobbaStart) Run() {

	for name, handler := range handlers.Handlers {
		if checkInterface(handler) {
			s.Scheduler.RegisterHandler(name, handler)
		}
	}

	s.Scheduler.Start()
}

func checkInterface(value interface{}) bool {
    _, ok := value.(services.TaskHandlerInt)
    return ok
}
