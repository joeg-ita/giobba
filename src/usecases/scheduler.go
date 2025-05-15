package usecases

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/joeg-ita/giobba/src/config"
	"github.com/joeg-ita/giobba/src/domain"
	"github.com/joeg-ita/giobba/src/services"
	"github.com/joeg-ita/giobba/src/utils"
	"github.com/redis/go-redis/v9"
)

const (
	QUEUE_SCHEDULE_POSTFIX = ":scheduled"
	SERVICES_CHANNEL       = "giobba-services"
	HEARTBEATS_CHANNEL     = "giobba-heartbeats"
	ACTIVITIES_CHANNEL     = "giobba-activities"
	HEARTBEAT_INTERVAL     = 5
)

type Worker struct {
	Id            string
	context       context.Context
	cancel        context.CancelFunc
	processing    bool
	currentTaskId string
	mutex         sync.RWMutex
}

type Scheduler struct {
	Id                string
	context           context.Context
	brokerClient      services.BrokerInt
	dbClient          services.DbTasksInt
	dbJobsClient      services.DbJobsInt
	restClient        services.RestInt
	Tasker            Tasker
	Queues            []string
	Hostname          string
	Handlers          map[string]services.TaskHandlerInt
	MaxWorkers        int
	Workers           map[string]*Worker
	LockDuration      time.Duration
	PollingTimeout    time.Duration
	Mutex             sync.Mutex
	FailedExecutions  int
	SuccessExecutions int
	shutdownChan      chan struct{}
	shutdownWg        sync.WaitGroup
	isShuttingDown    bool
	shutdownTimeout   time.Duration
	cleanupWg         sync.WaitGroup
}

func NewScheduler(
	ctx context.Context,
	brokerClient services.BrokerInt,
	dbTasksClient services.DbTasksInt,
	dbJobsClient services.DbJobsInt,
	cfg *config.Config) Scheduler {

	hostname, _ := os.Hostname()
	schedulerUuid := uuid.New().String()
	schedulerId := fmt.Sprintf("sched-%s-%s", hostname, schedulerUuid[:8])

	workers := make(map[string]*Worker, cfg.WorkersNumber)
	for i := range cfg.WorkersNumber {
		wCtx, wCanc := context.WithCancel(ctx)
		id := fmt.Sprintf("%s-w-%d", schedulerId, i)
		workers[id] = &Worker{
			Id:            id,
			context:       wCtx,
			cancel:        wCanc,
			processing:    false,
			currentTaskId: "",
		}
	}

	httpService := services.NewHttpRest()

	taskUtils := NewTaskUtils(brokerClient, dbTasksClient, dbJobsClient, httpService, time.Duration(cfg.LockDuration)*time.Second)

	return Scheduler{
		Id:                fmt.Sprintf("sched-%v-%s", hostname, schedulerUuid[:8]),
		context:           ctx,
		Tasker:            *taskUtils,
		brokerClient:      brokerClient,
		dbClient:          dbTasksClient,
		dbJobsClient:      dbJobsClient,
		restClient:        httpService,
		Hostname:          hostname,
		Handlers:          make(map[string]services.TaskHandlerInt),
		Queues:            cfg.Queues,
		Workers:           workers,
		MaxWorkers:        cfg.WorkersNumber,
		LockDuration:      time.Duration(cfg.LockDuration) * time.Second,
		PollingTimeout:    time.Duration(cfg.PollingTimeout) * time.Second,
		FailedExecutions:  0,
		SuccessExecutions: 0,
		shutdownChan:      make(chan struct{}),
		shutdownTimeout:   time.Duration(30) * time.Second,
	}
}

// RegisterHandler registra un handler per un tipo di task
func (s *Scheduler) RegisterHandler(taskName string, handler services.TaskHandlerInt) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	s.Handlers[taskName] = handler
}

func (s *Scheduler) Start() {
	log.Printf("Starting Scheduler [%v] with [%d] workers on task queues [%s]", s.Id, s.MaxWorkers, s.Queues)
	for i := range s.Workers {
		go s.startWorker(s.Workers[i])
	}
	go s.startCron(s.context)
	go s.startPubSub(s.context)
	go s.heartbeat(s.context)
}

func (s *Scheduler) startPubSub(ctx context.Context) {
	log.Printf("subscribing channel %s", SERVICES_CHANNEL)
	pubsub := (s.brokerClient.Subscribe(ctx, SERVICES_CHANNEL)).(*redis.PubSub)
	defer pubsub.Close()
	ch := pubsub.Channel()

	for msg := range ch {
		log.Printf("message on channel [%v] payload=[%v]", msg.Channel, msg.Payload)
		splitted := strings.Split(msg.Payload, ":")
		if len(splitted) == 3 {
			if strings.ToUpper(splitted[0]) == "KILL" {
				task, _ := s.Tasker.brokerClient.GetTask(splitted[2], splitted[1])
				s.Tasker.KillTask(ctx, s.Workers[task.WorkerID], splitted[2], splitted[1])
			} else if strings.ToUpper(splitted[0]) == "REVOKE" {
				s.Tasker.RevokeTask(splitted[2], splitted[1])
			} else if strings.ToUpper(splitted[0]) == "AUTO" {
				s.Tasker.AutoTask(splitted[2], splitted[1])
			}
		}
	}
}

func (s *Scheduler) heartbeat(context context.Context) {
	for {
		if time.Now().Unix()%HEARTBEAT_INTERVAL == 0 {
			s.brokerClient.Publish(context, HEARTBEATS_CHANNEL, map[string]interface{}{
				"id":           s.Id,
				"queues":       s.Queues,
				"workers":      s.MaxWorkers,
				"hostname":     s.Hostname,
				"successTasks": s.SuccessExecutions,
				"failedTasks":  s.FailedExecutions,
			})
			time.Sleep(1 * time.Second)
		}
	}
}

func (s *Scheduler) startWorker(worker *Worker) {
	log.Printf("worker [%s] started", worker.Id)

	for {
		select {
		case <-worker.context.Done():
			log.Printf("worker [%s] shutting down", worker.Id)
			return
		case <-s.shutdownChan:
			log.Printf("worker [%s] received shutdown signal", worker.Id)
			return
		default:
			// Use RLock for checking processing state
			worker.mutex.RLock()
			if worker.processing {
				worker.mutex.RUnlock()
				time.Sleep(s.PollingTimeout)
				continue
			}
			worker.mutex.RUnlock()

			// Lock only when we're about to change the state
			worker.mutex.Lock()
			worker.processing = true
			worker.mutex.Unlock()

			// Try to fetch and process a task
			if time.Now().Unix()%10 == 0 {
				log.Printf("worker %v (-_-) zzz", worker.Id)
			}

			// Add to wait group before processing task
			s.shutdownWg.Add(1)
			err := s.fetchAndProcessTask(worker)
			s.shutdownWg.Done()

			// Update worker state atomically
			worker.mutex.Lock()
			worker.processing = false
			worker.currentTaskId = ""
			worker.mutex.Unlock()

			if err != nil {
				// If error is due to no ready tasks, just wait
				if err == redis.Nil {
					time.Sleep(s.PollingTimeout)
				} else {
					log.Printf("worker %s encountered an error: %v", worker.Id, err)
					time.Sleep(s.PollingTimeout)
				}
			}
			time.Sleep(s.PollingTimeout)
		}
	}
}

func (s *Scheduler) startCron(ctx context.Context) {
	log.Printf("cron scheduler %v starting", s.Id)

	for {
		select {
		case <-ctx.Done():
			log.Printf("cron shutting down")
			return
		default:
			log.Printf("cron fetching tasks")
			jobs, err := s.Tasker.dbJobsClient.RetrieveMany(ctx, "", 0, 100, nil)
			log.Printf("retrieved %d scheduled tasks", len(jobs))
			if err != nil {
				log.Println(err.Error())
			}
			for i := range jobs {
				job := jobs[i]
				taskId := job.TaskID
				queue := job.TaskQueue
				if s.brokerClient.Lock(taskId, queue, s.LockDuration) {
					log.Printf("retrieving scheduled task %v", taskId)
					task, err := s.Tasker.Task(taskId, queue)
					if err != nil {
						log.Println(err.Error())
						job.IsActive = false
						s.Tasker.dbJobsClient.Save(ctx, job)
						continue
					}
					if task.State == domain.COMPLETED {
						nextExecution := utils.CalculateNextExecution(job.Schedule, time.Now())
						if job.LastExecution.After(nextExecution) {
							task.ETA = nextExecution
							task.State = domain.PENDING
							_, err := s.brokerClient.SaveTask(task, task.Queue)
							if err != nil {
								log.Printf("unable to update task %v", task.ID)
							}
							err = s.brokerClient.Schedule(task, task.Queue+QUEUE_SCHEDULE_POSTFIX)
							if err != nil {
								log.Printf("unable to schedule task %v on queue %v", task.ID, task.Queue)
							}
							job.NextExecution = nextExecution
							log.Printf("task %v scheduled with next execution %v", task.ID, task.ETA)
							s.Tasker.Notify(ctx, task)
						} else {
							job.IsActive = false
							log.Printf("task schedule expired for task %v", task.ID)
						}
						log.Printf("job update %v", job)
						s.Tasker.dbJobsClient.Save(ctx, job)
					}
					s.brokerClient.UnLock(taskId, queue)
				}
			}
		}
		time.Sleep(15 * time.Second)
	}
}

// Update fetchAndProcessTask to set currentTaskId before executing
func (s *Scheduler) fetchAndProcessTask(worker *Worker) error {

	var allScheduled []string
	queues := s.Queues

	for _, queue := range queues {
		scheduled, err := s.brokerClient.GetScheduled(queue + QUEUE_SCHEDULE_POSTFIX)
		if err != nil {
			return err
		}
		allScheduled = append(allScheduled, scheduled...)
	}

	if len(allScheduled) == 0 {
		return nil // Nessun task da eseguire
	}

	// Sort all items by score
	sort.Slice(allScheduled, func(i, j int) bool {
		item_i := strings.Split(allScheduled[i], "::")
		item_j := strings.Split(allScheduled[j], "::")
		score_i, _ := strconv.ParseFloat(item_i[2], 64)
		score_j, _ := strconv.ParseFloat(item_j[2], 64)
		return score_i < score_j
	})

	for _, schedTaskItem := range allScheduled {
		// Recuperiamo il task
		item := strings.Split(schedTaskItem, "::")
		taskID := string(item[0])
		taskScheduledQueue := string(item[1])
		taskQueue := strings.TrimSuffix(taskScheduledQueue, QUEUE_SCHEDULE_POSTFIX)
		task, err := s.brokerClient.GetTask(taskID, taskQueue)
		if err == redis.Nil {
			// Il task non esiste più, lo rimuoviamo dalla coda di scheduling
			s.brokerClient.UnSchedule(schedTaskItem, taskScheduledQueue)
			continue
		} else if err != nil {
			log.Printf("error fetching task %s from queue %s: %v", taskID, taskQueue, err)
			continue
		}

		// Verifichiamo se esiste un handler per questo tipo di task
		if _, ok := s.Handlers[task.Name]; !ok {
			log.Printf("no handler registered for task %s of type %s", taskID, task.Name)
			continue
		}

		// Verifichiamo che ETA sia stata raggiunta
		if task.ETA.After(time.Now()) {
			log.Printf("ETA %s not yet reached %s for task %s", task.ETA, time.Now(), taskID)
			continue
		}

		// Verifichiamo se il task è stato cancellato
		if task.State == domain.REVOKED || task.State == domain.KILLED {
			// Il task revoked, lo rimuoviamo dalla coda di scheduling
			s.brokerClient.UnSchedule(schedTaskItem, taskScheduledQueue)
			continue
		}

		if task.State == domain.RUNNING || task.State == domain.COMPLETED || task.StartMode == domain.MANUAL {
			// Il task è in esecuzione, completato o con attivazione manuale
			continue
		}

		// Verifichiamo se il task è in attesa di attivazione manuale
		if task.StartMode == domain.AUTO && task.ParentID != task.ID {
			// Se non è in modalità auto-attivazione e ha un parent, verifichiamo lo stato del parent
			parentTask, err := s.brokerClient.GetTask(task.ParentID, taskQueue)
			if err == redis.Nil {
				// Il parent non esiste, passiamo al prossimo task
				continue
			} else if err != nil {
				log.Printf("error fetching parent %s for task %s on queue %s: %v", task.ParentID, taskID, taskQueue, err)
				continue
			}

			// Verifichiamo se il parent è completo
			if parentTask.State != domain.COMPLETED {
				continue
			}
		}

		// Proviamo ad acquisire il lock per il task
		if s.brokerClient.Lock(taskID, taskQueue, s.LockDuration) {
			task.WorkerID = worker.Id
			task.State = domain.RUNNING
			task.StartedAt = time.Now()
			task.SchedulerID = s.Id
			s.brokerClient.SaveTask(task, taskQueue)
			s.brokerClient.UnSchedule(schedTaskItem, taskScheduledQueue)

			s.Tasker.Notify(context.Background(), task)

			// Update worker state atomically when setting currentTaskId
			worker.mutex.Lock()
			worker.currentTaskId = task.ID
			worker.mutex.Unlock()

			task, err := s.executeTask(worker, task)

			// Update worker state atomically when clearing currentTaskId
			worker.mutex.Lock()
			worker.currentTaskId = ""
			worker.mutex.Unlock()

			s.Tasker.Notify(context.Background(), task)

			if err != nil {
				return err
			}

		}
	}

	return nil
}

// Update the executeTask method to check for context cancellation
func (s *Scheduler) executeTask(worker *Worker, task domain.Task) (domain.Task, error) {
	log.Printf("worker %v is executing task %v", worker.Id, task.ID)

	// Create new context and update worker state atomically
	worker.mutex.Lock()
	ctx, cancel := context.WithCancel(s.context)
	worker.context = ctx
	worker.cancel = cancel
	worker.mutex.Unlock()

	// Execute the task with context awareness
	handler := s.Handlers[task.Name]

	go s.renewLock(worker.context, task.ID, task.Queue)

	// Create a channel for the task result
	done := make(chan services.HandlerResult, 1)

	// Start the task in a goroutine
	go func() {
		// Call the handler with the context
		result := handler.Run(worker.context, task)
		done <- result
	}()

	// Wait for either the task to complete or the context to be cancelled
	select {
	case result := <-done:
		// Task completed normally
		if result.Err == nil {
			log.Printf("task %s completed!", task.ID)
			task.State = domain.COMPLETED
			task.Result = result.Payload
			task.CompletedAt = time.Now()
			if task.Callback != "" {
				go s.Tasker.Callback(task.Callback, task.Result)
			}
			s.Mutex.Lock()
			s.SuccessExecutions++
			s.Mutex.Unlock()
		} else {
			log.Printf("task %s failed!", task.ID)

			// Increment retry counter
			task.Retries++

			// Check if we should retry the task
			if task.MaxRetries > 0 && task.Retries <= task.MaxRetries {
				log.Printf("retrying task %s (attempt %d/%d)", task.ID, task.Retries, task.MaxRetries)

				// Reset task state for retry
				task.State = domain.PENDING
				task.Error = result.Err.Error()
				task.UpdatedAt = time.Now()

				// Reschedule the task with a small delay
				task.ETA = time.Now().Add(time.Second * time.Duration(task.Retries))

				// Save and reschedule the task
				s.brokerClient.SaveTask(task, task.Queue)
				s.brokerClient.Schedule(task, task.Queue+QUEUE_SCHEDULE_POSTFIX)

				// Notify about retry
				s.Tasker.Notify(context.Background(), task)

				return task, nil
			}

			// Max retries reached or no retries configured
			task.State = domain.FAILED
			task.Error = result.Err.Error()
			s.Mutex.Lock()
			s.FailedExecutions++
			s.Mutex.Unlock()

			if task.CallbackErr != "" {
				errMsg := map[string]interface{}{
					"error":      task.Error,
					"retries":    task.Retries,
					"maxRetries": task.MaxRetries,
				}
				go s.Tasker.Callback(task.CallbackErr, errMsg)
			}
		}
		s.brokerClient.SaveTask(task, task.Queue)
		s.brokerClient.UnLock(task.ID, task.Queue)

		return task, result.Err
	case <-worker.context.Done():
		// Task was cancelled
		log.Printf("task %s cancelled", task.ID)
		errorMsg := fmt.Errorf("task cancelled")
		task.State = domain.KILLED
		task.CompletedAt = time.Now()
		task.Error = errorMsg.Error()
		s.brokerClient.SaveTask(task, task.Queue)
		s.brokerClient.UnLock(task.ID, task.Queue)
		if task.CallbackErr != "" {
			errMsg := map[string]interface{}{
				"error": task.Error,
			}
			go s.Tasker.Callback(task.CallbackErr, errMsg)
		}

		return task, errorMsg
	}
}

func (s *Scheduler) renewLock(ctx context.Context, taskId string, queue string) {
	ticker := time.NewTicker(s.LockDuration / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !s.brokerClient.RenewLock(ctx, taskId, queue, s.LockDuration) {
				return
			}
		case <-s.context.Done():
			return
		}
	}
}

// Stop gracefully stops the task scheduler
func (s *Scheduler) Stop() {
	log.Printf("Initiating graceful shutdown of scheduler %s", s.Id)

	// Set shutdown flag with proper mutex handling
	s.Mutex.Lock()
	if s.isShuttingDown {
		s.Mutex.Unlock()
		return
	}
	s.isShuttingDown = true
	s.Mutex.Unlock()

	// Create a context with timeout for shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()

	// Signal all workers to stop accepting new tasks
	close(s.shutdownChan)

	// Wait for all in-flight tasks to complete
	done := make(chan struct{})
	go func() {
		s.shutdownWg.Wait()
		close(done)
	}()

	// Wait for either all tasks to complete or timeout
	select {
	case <-done:
		log.Printf("All in-flight tasks completed successfully")
	case <-shutdownCtx.Done():
		log.Printf("Shutdown timed out after %v", s.shutdownTimeout)
	}

	// Cancel all worker contexts with proper mutex handling
	for _, worker := range s.Workers {
		worker.mutex.Lock()
		if worker.cancel != nil {
			worker.cancel()
		}
		worker.mutex.Unlock()
	}

	// Start cleanup operations
	s.cleanupWg.Add(3) // One for each cleanup operation

	// Cleanup Redis broker
	go func() {
		defer s.cleanupWg.Done()
		log.Printf("Cleaning up Redis broker connections...")
		s.brokerClient.Close()
		log.Printf("Redis broker cleanup completed")
	}()

	// Cleanup MongoDB tasks client
	go func() {
		defer s.cleanupWg.Done()
		log.Printf("Cleaning up MongoDB tasks client...")
		s.dbClient.Close(shutdownCtx)
		log.Printf("MongoDB tasks client cleanup completed")
	}()

	// Cleanup MongoDB jobs client
	go func() {
		defer s.cleanupWg.Done()
		log.Printf("Cleaning up MongoDB jobs client...")
		s.dbJobsClient.Close(shutdownCtx)
		log.Printf("MongoDB jobs client cleanup completed")
	}()

	// Wait for all cleanup operations to complete or timeout
	cleanupDone := make(chan struct{})
	go func() {
		s.cleanupWg.Wait()
		close(cleanupDone)
	}()

	select {
	case <-cleanupDone:
		log.Printf("All cleanup operations completed successfully")
	case <-shutdownCtx.Done():
		log.Printf("Cleanup operations timed out after %v", s.shutdownTimeout)
	}

	log.Printf("Scheduler %s shutdown complete", s.Id)
}
