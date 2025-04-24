package services

import (
	"context"
	"fmt"
	"giobba/src/entities"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type TaskHandlerInt interface {
	Run(ctx context.Context, task entities.Task) error
}

// type TaskHandler func(ctx context.Context, task entities.Task) error

const (
	QUEUE_SCHEDULE_POSTFIX = ":scheduled"
	PUBSUB_CHANNEL         = "giobba"
)

type Worker struct {
	Id            string
	context       context.Context
	cancel        context.CancelFunc
	processing    bool
	currentTaskId string
	mutex         sync.Mutex
}

type Scheduler struct {
	Id             string
	context        context.Context
	queueClient    BrokerInt
	restClient     RestInt
	Queue          string
	Hostname       string
	Handlers       map[string]TaskHandlerInt
	MaxWorkers     int
	Workers        []*Worker
	LockDuration   time.Duration
	PollingTimeout time.Duration
	Mutex          sync.Mutex
}

func NewScheduler(ctx context.Context, queueClient BrokerInt, queue string, maxWorkers int, lockDuration int) Scheduler {

	schedulerId := uuid.New().String()

	workers := make([]*Worker, maxWorkers)
	for i := 0; i < maxWorkers; i++ {
		wCtx, wCanc := context.WithCancel(ctx)
		workers[i] = &Worker{
			Id:            fmt.Sprintf("worker-%s-%d", schedulerId[:8], i),
			context:       wCtx,
			cancel:        wCanc,
			processing:    false,
			currentTaskId: "",
		}
	}

	hostname, _ := os.Hostname()

	return Scheduler{
		Id:             fmt.Sprintf("sched-%v-%s", hostname, schedulerId[:8]),
		context:        ctx,
		queueClient:    queueClient,
		restClient:     NewHttpRest(),
		Hostname:       hostname,
		Handlers:       make(map[string]TaskHandlerInt),
		Queue:          queue,
		Workers:        workers,
		MaxWorkers:     maxWorkers,
		LockDuration:   time.Duration(lockDuration) * time.Second,
		PollingTimeout: time.Duration(1) * time.Second,
	}
}

// RegisterHandler registra un handler per un tipo di task
func (s *Scheduler) RegisterHandler(taskName string, handler TaskHandlerInt) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	s.Handlers[taskName] = handler
}

func (s *Scheduler) Start() {
	log.Printf("Starting Scheduler [%v] with [%d] workers on task queue [%s]", s.Id, s.MaxWorkers, s.Queue)
	for i := 0; i < len(s.Workers); i++ {
		go s.startWorker(s.Workers[i])
	}
	go s.startPubSub()
}

func (s *Scheduler) startPubSub() {
	log.Printf("Subscribing channel %s", PUBSUB_CHANNEL)
	pubsub := (s.queueClient.Subscribe(s.context, PUBSUB_CHANNEL)).(*redis.PubSub)
	defer pubsub.Close()
	ch := pubsub.Channel()

	for msg := range ch {
		log.Printf("Message on channel [%v] payload=[%v]", msg.Channel, msg.Payload)
		splitted := strings.Split(msg.Payload, ":")
		if len(splitted) == 3 {
			if strings.ToUpper(splitted[0]) == "KILL" {
				s.killTask(splitted[2], splitted[1])
			} else if strings.ToUpper(splitted[0]) == "REVOKE" {
				s.revokeTask(splitted[2], splitted[1])
			} else if strings.ToUpper(splitted[0]) == "AUTO" {
				s.autoTask(splitted[2], splitted[1])
			}
		}

	}
}

func (s *Scheduler) startWorker(worker *Worker) {
	log.Printf("Worker %s started", worker.Id)

	for {
		select {
		case <-worker.context.Done():
			log.Printf("Worker %s shutting down", worker.Id)
			return
		default:
			// Check if worker is already processing
			worker.mutex.Lock()
			if worker.processing {
				worker.mutex.Unlock()
				time.Sleep(s.PollingTimeout)
				continue
			}
			worker.processing = true
			worker.mutex.Unlock()

			// Try to fetch and process a task
			if time.Now().Unix()%10 == 0 {
				log.Printf("worker %v idle :-/", worker.Id)
			}
			err := s.fetchAndProcessTask(worker)

			worker.mutex.Lock()
			worker.processing = false
			worker.currentTaskId = ""
			worker.mutex.Unlock()

			if err != nil {
				// If error is due to no ready tasks, just wait
				if err == redis.Nil {
					time.Sleep(s.PollingTimeout)
				} else {
					log.Printf("Worker %s encountered an error: %v", worker.Id, err)
					time.Sleep(s.PollingTimeout)
				}
			}
			time.Sleep(s.PollingTimeout)
		}
	}
}

// Update fetchAndProcessTask to set currentTaskId before executing
func (s *Scheduler) fetchAndProcessTask(worker *Worker) error {

	taskIDs, err := s.queueClient.GetScheduled(s.Queue + QUEUE_SCHEDULE_POSTFIX)

	if err != nil {
		return err
	}

	if len(taskIDs) == 0 {
		return nil // Nessun task da eseguire
	}

	for _, taskID := range taskIDs {
		// Recuperiamo il task
		task, err := s.queueClient.GetTask(taskID, s.Queue)
		if err == redis.Nil {
			// Il task non esiste più, lo rimuoviamo dalla coda di scheduling
			s.queueClient.UnSchedule(task.ID, s.Queue+QUEUE_SCHEDULE_POSTFIX)
			continue
		} else if err != nil {
			log.Printf("Errore nel recupero del task %s: %v", taskID, err)
			continue
		}

		// Verifichiamo se esiste un handler per questo tipo di task
		if _, ok := s.Handlers[task.Name]; !ok {
			log.Printf("Nessun handler registrato per il task %s di tipo %s", taskID, task.Name)
			continue
		}

		// Verifichiamo se il task è stato cancellato
		if task.State == entities.REVOKED {
			// Il task revoked, lo rimuoviamo dalla coda di scheduling
			s.queueClient.UnSchedule(task.ID, s.Queue+QUEUE_SCHEDULE_POSTFIX)
			continue
		}

		if task.State == entities.RUNNING || task.State == entities.COMPLETED || task.StartMode == entities.MANUAL {
			// Il task è in esecuzione, completato o con attivazione manuale
			continue
		}

		// Verifichiamo se il task è in attesa di attivazione manuale
		if task.StartMode == entities.AUTO && task.ParentID != task.ID {
			// Se non è in modalità auto-attivazione e ha un parent, verifichiamo lo stato del parent
			parentTask, err := s.queueClient.GetTask(task.ParentID, s.Queue)
			if err == redis.Nil {
				// Il parent non esiste, passiamo al prossimo task
				continue
			} else if err != nil {
				log.Printf("Errore nel recupero del parent %s per il task %s: %v", task.ParentID, taskID, err)
				continue
			}

			// Verifichiamo se il parent è completo
			if parentTask.State != entities.COMPLETED {
				continue
			}
		}

		// Proviamo ad acquisire il lock per il task
		if s.queueClient.Lock(taskID, s.Queue, s.LockDuration) {
			task.WorkerID = worker.Id
			task.State = entities.RUNNING
			task.StartedAt = time.Now()
			task.SchedulerID = s.Id
			s.queueClient.SaveTask(task, s.Queue)
			s.queueClient.UnSchedule(task.ID, s.Queue+QUEUE_SCHEDULE_POSTFIX)

			// Set the currentTaskId before executing the task
			worker.mutex.Lock()
			worker.currentTaskId = task.ID
			worker.mutex.Unlock()

			err := s.executeTask(worker, task)

			// Clear the currentTaskId after execution
			worker.mutex.Lock()
			worker.currentTaskId = ""
			worker.mutex.Unlock()

			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Update the executeTask method to check for context cancellation
func (s *Scheduler) executeTask(worker *Worker, task entities.Task) error {
	log.Printf("worker %v is executing task %v", worker.Id, task.ID)

	worker.mutex.Lock()
	ctx, cancel := context.WithCancel(s.context)
	worker.context = ctx
	worker.cancel = cancel
	worker.mutex.Unlock()

	// Execute the task with context awareness
	handler := s.Handlers[task.Name]

	// Create a channel for the task result
	done := make(chan error, 1)

	// Start the task in a goroutine
	go func() {
		// Call the handler with the context
		result := handler.Run(worker.context, task)
		done <- result
	}()

	// Wait for either the task to complete or the context to be cancelled
	select {
	case err := <-done:
		// Task completed normally
		if err == nil {
			log.Printf("task %s completed!", task.ID)
			task.State = entities.COMPLETED
			task.FinishedAt = time.Now()
			if task.Callback != "" {
				go s.callback(task.Callback, task.Result)
			}
		} else {
			log.Printf("task %s failed!", task.ID)
			task.State = entities.FAILED
			task.Error = err.Error()
			if task.CallbackErr != "" {
				errMsg := map[string]interface{}{
					"error": task.Error,
				}
				go s.callback(task.CallbackErr, errMsg)
			}

		}
		s.queueClient.SaveTask(task, s.Queue)
		s.queueClient.UnLock(task.ID, s.Queue)

		return err
	case <-worker.context.Done():
		// Task was cancelled
		log.Printf("task %s cancelled", task.ID)
		errorMsg := fmt.Errorf("task cancelled")
		task.State = entities.KILLED
		task.FinishedAt = time.Now()
		task.Error = errorMsg.Error()
		s.queueClient.SaveTask(task, s.Queue)
		s.queueClient.UnLock(task.ID, s.Queue)
		if task.CallbackErr != "" {
			errMsg := map[string]interface{}{
				"error": task.Error,
			}
			go s.callback(task.CallbackErr, errMsg)
		}

		return errorMsg
	}

}

// Update killTask to properly handle cancellation
func (s *Scheduler) killTask(taskID string, queue string) error {
	log.Printf("killing task %v", taskID)
	task, err := s.queueClient.GetTask(taskID, queue)
	if err != nil {
		return fmt.Errorf("failed to find task %s: %v", taskID, err)
	}

	if task.State != entities.RUNNING {
		log.Printf("task %v not in %v state", taskID, entities.RUNNING)
		return nil
	}

	// Find the worker processing this task
	var targetWorker *Worker
	for _, worker := range s.Workers {
		worker.mutex.Lock()
		if worker.Id == task.WorkerID {
			targetWorker = worker
			worker.mutex.Unlock()
			break
		}
		worker.mutex.Unlock()
	}

	if targetWorker == nil {
		return fmt.Errorf("worker for task %s not found", taskID)
	}

	// Cancel the worker context and create a new one
	targetWorker.mutex.Lock()
	targetWorker.cancel()

	// Create a new context for the worker
	newCtx, newCancel := context.WithCancel(s.context)
	targetWorker.context = newCtx
	targetWorker.cancel = newCancel
	targetWorker.mutex.Unlock()

	log.Printf("Task %s successfully killed", taskID)
	return nil
}

func (s *Scheduler) revokeTask(taskID string, queue string) error {

	if s.queueClient.Lock(taskID, s.Queue, s.LockDuration) {
		log.Printf("revoking task %v", taskID)
		task, err := s.queueClient.GetTask(taskID, queue)
		if err != nil {
			return fmt.Errorf("failed to find task %s: %v", taskID, err)
		}

		if task.State != entities.PENDING {
			log.Printf("task %v not in %v state", taskID, entities.PENDING)
			return nil
		}
		task.State = entities.REVOKED
		_, err = s.queueClient.SaveTask(task, queue)
		s.queueClient.UnLock(taskID, queue)

		if err != nil {
			return err
		}
	}
	log.Printf("Task %s successfully revoked", taskID)
	return nil
}

func (s *Scheduler) autoTask(taskID string, queue string) error {

	if s.queueClient.Lock(taskID, s.Queue, s.LockDuration) {
		log.Printf("setting task %v to auto run", taskID)
		task, err := s.queueClient.GetTask(taskID, queue)
		if err != nil {
			return fmt.Errorf("failed to find task %s: %v", taskID, err)
		}

		if task.State != entities.PENDING {
			log.Printf("task %v not in %v state", taskID, entities.PENDING)
			return nil
		}
		task.StartMode = entities.AUTO
		_, err = s.queueClient.SaveTask(task, queue)
		s.queueClient.UnLock(taskID, queue)

		if err != nil {
			return err
		}
	}
	log.Printf("Task %s successfully set to auto run", taskID)
	return nil
}

func (s *Scheduler) AddTask(task entities.Task) (string, error) {

	log.Printf("Adding task %v", task)

	taskId, err := s.queueClient.AddTask(task, task.Queue)
	if err != nil {
		return "", err
	}
	err = s.queueClient.Schedule(task, s.Queue+QUEUE_SCHEDULE_POSTFIX)
	if err != nil {
		return "", err
	}
	return taskId, nil
}

func (s *Scheduler) callback(url string, payload map[string]interface{}) {
	err := s.restClient.Post(url, payload)
	if err != nil {
		log.Println(err)
	}
}

// Stop gracefully stops the task scheduler
func (s *Scheduler) Stop() {

	// Close Redis connection
	s.queueClient.Close()
}
