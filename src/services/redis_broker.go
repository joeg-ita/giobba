package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/joeg-ita/giobba/src/domain"

	"github.com/redis/go-redis/v9"
)

type RedisBroker struct {
	client *redis.Client
}

func NewRedisBrokerByOptions(options *redis.Options) *RedisBroker {
	client := redis.NewClient(options)
	return &RedisBroker{
		client: client,
	}
}

func NewRedisBroker(address string, password string, db int, protocol int) *RedisBroker {
	opt := &redis.Options{
		Addr:     address,  // "localhost:6379"
		Password: password, // No password set
		DB:       db,       // Use default DB
		Protocol: protocol, // Connection protocol ie: 2
	}
	return NewRedisBrokerByOptions(opt)
}

func NewRedisBrokerByUrl(url string) *RedisBroker {
	// url = "redis://<user>:<pass>@localhost:6379/<db>"
	opt, err := redis.ParseURL(url)
	if err != nil {
		panic(err)
	}
	return NewRedisBrokerByOptions(opt)
}

func (r *RedisBroker) AddTask(task domain.Task, queue string) (string, error) {

	now := time.Now()
	task.UpdatedAt = now

	return r.SaveTask(task, queue)
}

func (r *RedisBroker) SaveTask(task domain.Task, queue string) (string, error) {
	log.Printf("Saving task %v in queue %v", task.ID, queue)

	now := time.Now()
	task.UpdatedAt = now

	taskJSON, err := json.Marshal(task)
	if err != nil {
		return "", err
	}
	err = r.client.HSet(context.Background(), task.Queue, task.ID, taskJSON).Err()
	if err != nil {
		return "", fmt.Errorf("error while adding task to queue: %w", err)
	}
	return task.ID, nil
}

func (r *RedisBroker) DeleteTask(taskId string, queue string) error {

	err := r.client.HDel(context.Background(), queue, taskId).Err()
	if err != nil {
		return fmt.Errorf("error while deleting task to queue: %w", err)
	}
	return nil
}

func (r *RedisBroker) GetTask(taskId string, queue string) (domain.Task, error) {
	taskJSON, err := r.client.HGet(context.Background(), queue, taskId).Result()
	if err == redis.Nil {
		return domain.Task{}, fmt.Errorf("task not found: %w", err)
	}
	var task domain.Task
	if err := json.Unmarshal([]byte(taskJSON), &task); err != nil {
		return domain.Task{}, fmt.Errorf("deserialization error: %w", err)
	}
	return task, nil
}

func (r *RedisBroker) UnSchedule(taskId string, queue string) error {
	_, err := r.client.ZRem(context.Background(), queue, taskId).Result()
	if err != nil {
		return err
	}
	return nil
}

func (r *RedisBroker) Schedule(task domain.Task, queue string) error {
	score := float64(task.ETA.Unix())
	if score < float64(time.Now().Unix()) {
		score = float64(time.Now().Unix())
	}
	priority := float64(10-task.Priority) / (100000.0 / math.Pow(float64(10-task.Priority), 2))
	finalScore := score + priority

	r.client.ZAdd(context.Background(), queue, redis.Z{
		Score:  finalScore,
		Member: fmt.Sprintf("%s::%s::%f", task.ID, queue, finalScore),
	})

	return nil
}

// TODO relace with ZRange with BYSCORE option (REV to get inverse order)
func (r *RedisBroker) GetScheduled(queue string) ([]string, error) {

	now := time.Now()
	maxScore := float64(now.Add(10 * time.Millisecond).Unix())

	// Otteniamo i task pronti per l'esecuzione (con score <= maxScore)
	taskIDs, err := r.client.ZRangeArgs(context.Background(), redis.ZRangeArgs{
		Key:     queue,
		Start:   "-inf",
		Stop:    fmt.Sprintf("%f", maxScore),
		Offset:  0,
		Count:   100, // Limitiamo a 100 task per ciclo
		Rev:     false,
		ByScore: true,
	}).Result()

	if err != nil {
		return nil, err
	}

	return taskIDs, nil
}

func (r *RedisBroker) Lock(taskId string, queue string, lockDuration time.Duration) bool {
	lockKey := queue + "-LOCK-" + taskId

	// Utilizziamo SETNX per provare ad impostare il lock
	success, err := r.client.SetNX(context.Background(), lockKey, "1", lockDuration).Result()
	if err != nil {
		log.Printf("Errore nell'acquisizione del lock per il task %s: %v", taskId, err)
		return false
	}

	return success
}

func (r *RedisBroker) RenewLock(ctx context.Context, taskId string, queue string, lockDuration time.Duration) bool {
	lockKey := queue + "-LOCK-" + taskId
	log.Printf("Renewing lock for task %s", taskId)

	// Utilizziamo SETNX per provare ad impostare il lock
	success, err := r.client.SetXX(context.Background(), lockKey, "1", lockDuration).Result()
	if err != nil {
		log.Printf("Errore nel rinnovo del lock per il task %s: %v", taskId, err)
		return false
	}

	return success
}

func (r *RedisBroker) UnLock(taskId string, queue string) error {
	lockKey := queue + "-LOCK-" + taskId
	_, err := r.client.Del(context.Background(), lockKey).Result()
	if err != nil {
		return err
	}
	return nil
}

func (r *RedisBroker) Close() {
	r.client.Close()
}

func (r *RedisBroker) Subscribe(ctx context.Context, channels ...string) interface{} {
	return r.client.Subscribe(ctx, channels...)
}

func (r *RedisBroker) Publish(ctx context.Context, channel string, payload map[string]interface{}) error {
	message, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = r.client.Publish(ctx, channel, message).Result()
	return err
}
